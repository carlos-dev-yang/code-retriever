package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
)

var ErrIndexCorrupt = errors.New("index corrupt")

// FTSSearchRequest contains only values prepared by the lexical service. Its
// MatchExpression is never accepted from an external request directly.
type FTSSearchRequest struct {
	MatchExpression       string
	CandidateK            int
	SymbolWeight          float64
	BodyWeight            float64
	ExactNormalizedSymbol string
}

type FTSSnapshot struct {
	Generation     int64
	ManifestSHA256 string
	Candidates     []FTSCandidate
}

// TruthSnapshot is the narrow, read-only indexed-parent inventory used by
// development evaluation preflight. Its rows are authoritative chunks, not
// FTS candidates or generated row identifiers.
type TruthSnapshot struct {
	Generation     int64
	ManifestSHA256 string
	Chunks         []TruthChunk
}
type TruthChunk struct {
	Path, IndexedSHA256, Kind, QualifiedSymbol string
	StartByte, EndByte                         int
}

func (store *ProductionStore) TruthSnapshot(ctx context.Context) (TruthSnapshot, error) {
	tx, err := store.Read.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return TruthSnapshot{}, err
	}
	defer tx.Rollback()
	var result TruthSnapshot
	if err := tx.QueryRowContext(ctx, `SELECT active_generation,manifest_sha256 FROM meta WHERE id=1`).Scan(&result.Generation, &result.ManifestSHA256); err != nil {
		return TruthSnapshot{}, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT f.path,f.indexed_sha256,c.kind,c.qualified_symbol,c.start_byte,c.end_byte FROM chunks c JOIN files f ON f.id=c.file_id ORDER BY f.path,c.start_byte,c.end_byte,c.id`)
	if err != nil {
		return TruthSnapshot{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var chunk TruthChunk
		if err := rows.Scan(&chunk.Path, &chunk.IndexedSHA256, &chunk.Kind, &chunk.QualifiedSymbol, &chunk.StartByte, &chunk.EndByte); err != nil {
			return TruthSnapshot{}, err
		}
		result.Chunks = append(result.Chunks, chunk)
	}
	if err := rows.Err(); err != nil {
		return TruthSnapshot{}, err
	}
	if err := tx.Commit(); err != nil {
		return TruthSnapshot{}, err
	}
	return result, nil
}

// FTSCandidate is copied from authoritative tables while the FTS read
// transaction remains open. BM25Score has a normalized higher-is-better
// direction; its absolute value is deliberately not an external contract.
type FTSCandidate struct {
	ChunkID                                                      int64
	Path, IndexedSHA256, Language, Kind, Symbol, QualifiedSymbol string
	Signature                                                    string
	StartByte, EndByte, StartLine, EndLine                       int
	BM25Score                                                    float64
	ExactQualifiedSymbol                                         bool
}

// SearchFTS pins meta, FTS ranking, and authoritative chunk data to one read
// transaction. It performs no filesystem, provider, or lab operation.
func (store *ProductionStore) SearchFTS(ctx context.Context, request FTSSearchRequest) (FTSSnapshot, error) {
	if request.MatchExpression == "" || request.CandidateK <= 0 || !finitePositive(request.SymbolWeight) || !finitePositive(request.BodyWeight) {
		return FTSSnapshot{}, fmt.Errorf("invalid FTS search request")
	}
	tx, err := store.Read.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return FTSSnapshot{}, err
	}
	defer tx.Rollback()

	var result FTSSnapshot
	if err := tx.QueryRowContext(ctx, `SELECT active_generation,manifest_sha256 FROM meta WHERE id=1`).Scan(&result.Generation, &result.ManifestSHA256); err != nil {
		return FTSSnapshot{}, err
	}
	rows, err := tx.QueryContext(ctx, `
		WITH fts_matches AS (
			SELECT rowid AS chunk_id,-bm25(chunk_fts,?,?) AS bm25_score
			FROM chunk_fts
			WHERE chunk_fts MATCH ?
		), candidates AS (
			SELECT m.chunk_id,m.bm25_score,
			       c.id IS NULL OR f.id IS NULL AS orphaned,
			       COALESCE(f.path,'') AS path,COALESCE(f.indexed_sha256,'') AS indexed_sha256,COALESCE(f.language,'') AS language,COALESCE(c.kind,'') AS kind,
			       COALESCE(c.symbol,'') AS symbol,COALESCE(c.qualified_symbol,'') AS qualified_symbol,COALESCE(c.signature,'') AS signature,
			       COALESCE(c.start_byte,0) AS start_byte,COALESCE(c.end_byte,0) AS end_byte,COALESCE(c.start_line,0) AS start_line,COALESCE(c.end_line,0) AS end_line,
			       CASE WHEN c.id IS NOT NULL AND EXISTS(
					SELECT 1 FROM symbols s
					WHERE s.chunk_id=c.id AND s.original_name=c.qualified_symbol AND s.normalized_name=?
				) THEN 1 ELSE 0 END AS exact_qualified_symbol
			FROM fts_matches m
			LEFT JOIN chunks c ON c.id=m.chunk_id
			LEFT JOIN files f ON f.id=c.file_id
		)
		SELECT chunk_id,bm25_score,orphaned,path,indexed_sha256,language,kind,symbol,qualified_symbol,signature,
		       start_byte,end_byte,start_line,end_line,exact_qualified_symbol
		FROM candidates
		ORDER BY bm25_score DESC,exact_qualified_symbol DESC,path ASC,qualified_symbol ASC,chunk_id ASC
		LIMIT ?`, request.SymbolWeight, request.BodyWeight, request.MatchExpression, request.ExactNormalizedSymbol, request.CandidateK)
	if err != nil {
		return FTSSnapshot{}, fmt.Errorf("FTS query failed: %w", err)
	}
	for rows.Next() {
		var candidate FTSCandidate
		var orphaned, exact int
		if err := rows.Scan(&candidate.ChunkID, &candidate.BM25Score, &orphaned, &candidate.Path, &candidate.IndexedSHA256, &candidate.Language, &candidate.Kind, &candidate.Symbol, &candidate.QualifiedSymbol, &candidate.Signature, &candidate.StartByte, &candidate.EndByte, &candidate.StartLine, &candidate.EndLine, &exact); err != nil {
			rows.Close()
			return FTSSnapshot{}, err
		}
		if orphaned != 0 {
			rows.Close()
			return FTSSnapshot{}, fmt.Errorf("%w: FTS row %d has no authoritative chunk", ErrIndexCorrupt, candidate.ChunkID)
		}
		candidate.ExactQualifiedSymbol = exact != 0
		result.Candidates = append(result.Candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return FTSSnapshot{}, err
	}
	if err := rows.Close(); err != nil {
		return FTSSnapshot{}, err
	}
	if err := tx.Commit(); err != nil {
		return FTSSnapshot{}, err
	}
	return result, nil
}

func finitePositive(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}
