package store

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"cidx/internal/config"
	"cidx/internal/vector"
)

// LexicalSearchSnapshot is intentionally light: it neither reads active
// segments nor vector rows, so malformed vector state cannot alter FTS.
type LexicalSearchSnapshot struct {
	Applied          config.AppliedProfiles
	FTSCandidates    []HybridFTSCandidate
	SymbolCandidates []HybridSymbolCandidate
	PathCandidates   []HybridPathCandidate
	Chunks           map[int64]HybridChunk
}

// HybridPreflightSnapshot validates active referenced vector rows before a
// paid request, without loading chunks, bodies, projections, or FTS data.
type HybridPreflightSnapshot struct {
	Applied           config.AppliedProfiles
	ProfileMatches    bool
	ValidVectorKeys   int
	InvalidVectorRows bool
}

func (store *ProductionStore) HybridPreflightSnapshot(ctx context.Context, resolved config.ResolvedConfig) (HybridPreflightSnapshot, error) {
	tx, err := store.Read.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return HybridPreflightSnapshot{}, err
	}
	defer tx.Rollback()
	applied, err := readAppliedProfiles(ctx, tx)
	if err != nil {
		return HybridPreflightSnapshot{}, err
	}
	out := HybridPreflightSnapshot{Applied: applied, ProfileMatches: profilesMatch(resolved, applied)}
	if out.ProfileMatches {
		rows, err := tx.QueryContext(ctx, `
			WITH active_keys AS (
				SELECT DISTINCT s.canonical_input_sha256 AS canonical_input_sha256
				FROM embedding_segments s JOIN meta m ON m.id=1
				WHERE s.serving_profile=m.active_serving_profile
			)
			SELECT k.canonical_input_sha256,v.dimensions,v.codec_id,v.codec_version,v.blob,v.scale,v.norm,v.source_profile,v.vector_space_profile,v.raw_vector_sha256,v.materialization_fingerprint,v.materialized_at
			FROM active_keys k JOIN meta m ON m.id=1
			LEFT JOIN vector_cache v ON v.serving_profile=m.active_serving_profile AND v.canonical_input_sha256=k.canonical_input_sha256
			ORDER BY k.canonical_input_sha256`)
		if err != nil {
			return HybridPreflightSnapshot{}, err
		}
		for rows.Next() {
			_, _, present, valid, err := scanStoredVector(rows, resolved)
			if err != nil {
				rows.Close()
				return HybridPreflightSnapshot{}, err
			}
			if valid {
				out.ValidVectorKeys++
			} else if present {
				out.InvalidVectorRows = true
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return HybridPreflightSnapshot{}, err
		}
		if err := rows.Close(); err != nil {
			return HybridPreflightSnapshot{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return HybridPreflightSnapshot{}, err
	}
	return out, nil
}

type HybridSnapshotRequest struct {
	FTS           FTSSearchRequest
	SymbolAnchors []LexicalAnchor
	PathAnchors   []LexicalAnchor
}

// HybridSearchSnapshot is one immutable pinned view. Chunks and vector keys
// are copied once; segments only point to those two maps.
type HybridSearchSnapshot struct {
	Applied             config.AppliedProfiles
	ProfileMatches      bool
	FTSCandidates       []HybridFTSCandidate
	SymbolCandidates    []HybridSymbolCandidate
	PathCandidates      []HybridPathCandidate
	Chunks              map[int64]HybridChunk
	Segments            []HybridSegment
	Vectors             map[string]vector.StoredVector
	CoverageNumerator   int
	CoverageDenominator int
	InvalidVectorRows   bool
}

// HybridFTSCandidate is the canonical lexical-parent record consumed by the
// existing lexical-vs-dense RRF stage. Before local fusion it represents the
// descriptive lane; after fusion the rank fields record every contributing
// local lane while FTSCandidate retains descriptive BM25 diagnostics.
type HybridFTSCandidate struct {
	FTSCandidate
	LexicalScore                           float64
	SymbolRank, PathRank, DescriptiveRank  int
	SymbolMatchTier, PathMatchTier         int
	SymbolAnchorMatched, PathAnchorMatched string
}

type HybridSymbolCandidate struct {
	ChunkID       int64
	MatchTier     int
	MatchedAnchor string
}

type HybridPathCandidate struct {
	ChunkID       int64
	MatchTier     int
	MatchedAnchor string
}

type HybridChunk struct {
	ID                                                       int64
	Path, Language, Kind, Symbol, QualifiedSymbol, Signature string
	StartByte, EndByte, StartLine, EndLine                   int
	SourceBody                                               []byte
	IndexedSHA256                                            string
}

type HybridSegment struct {
	ID, ChunkID              int64
	CanonicalInputSHA256     string
	DisplayStart, DisplayEnd int
}

func (store *ProductionStore) LexicalSearchSnapshot(ctx context.Context, request HybridSnapshotRequest) (LexicalSearchSnapshot, error) {
	if err := validateSnapshotRequest(request); err != nil {
		return LexicalSearchSnapshot{}, err
	}
	tx, err := store.Read.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return LexicalSearchSnapshot{}, err
	}
	defer tx.Rollback()
	applied, err := readAppliedProfiles(ctx, tx)
	if err != nil {
		return LexicalSearchSnapshot{}, err
	}
	candidates, chunks, err := loadFTSCandidates(ctx, tx, request.FTS, nil)
	if err != nil {
		return LexicalSearchSnapshot{}, err
	}
	symbols, err := loadSymbolCandidates(ctx, tx, request.SymbolAnchors, request.FTS.CandidateK, chunks)
	if err != nil {
		return LexicalSearchSnapshot{}, err
	}
	paths, err := loadPathCandidates(ctx, tx, request.PathAnchors, request.FTS.CandidateK, chunks)
	if err != nil {
		return LexicalSearchSnapshot{}, err
	}
	if err := tx.Commit(); err != nil {
		return LexicalSearchSnapshot{}, err
	}
	return LexicalSearchSnapshot{Applied: applied, FTSCandidates: candidates, SymbolCandidates: symbols, PathCandidates: paths, Chunks: chunks}, nil
}

func (store *ProductionStore) HybridSearchSnapshot(ctx context.Context, resolved config.ResolvedConfig, request HybridSnapshotRequest) (HybridSearchSnapshot, error) {
	if err := validateSnapshotRequest(request); err != nil {
		return HybridSearchSnapshot{}, err
	}
	return store.vectorSnapshot(ctx, resolved, &request)
}

// VectorSearchSnapshot loads the authoritative active vector/segment/parent
// state without constructing or running an FTS request. It is the snapshot
// entry point for isolated dense-codec evaluation.
func (store *ProductionStore) VectorSearchSnapshot(ctx context.Context, resolved config.ResolvedConfig) (HybridSearchSnapshot, error) {
	return store.vectorSnapshot(ctx, resolved, nil)
}

func (store *ProductionStore) vectorSnapshot(ctx context.Context, resolved config.ResolvedConfig, lexicalRequest *HybridSnapshotRequest) (HybridSearchSnapshot, error) {
	tx, err := store.Read.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return HybridSearchSnapshot{}, err
	}
	defer tx.Rollback()
	applied, err := readAppliedProfiles(ctx, tx)
	if err != nil {
		return HybridSearchSnapshot{}, err
	}
	out := HybridSearchSnapshot{Applied: applied, ProfileMatches: profilesMatch(resolved, applied), Chunks: map[int64]HybridChunk{}, Vectors: map[string]vector.StoredVector{}}
	if lexicalRequest != nil {
		fts, chunks, err := loadFTSCandidates(ctx, tx, lexicalRequest.FTS, out.Chunks)
		if err != nil {
			return HybridSearchSnapshot{}, err
		}
		out.FTSCandidates = fts
		for id, chunk := range chunks {
			out.Chunks[id] = chunk
		}
		out.SymbolCandidates, err = loadSymbolCandidates(ctx, tx, lexicalRequest.SymbolAnchors, lexicalRequest.FTS.CandidateK, out.Chunks)
		if err != nil {
			return HybridSearchSnapshot{}, err
		}
		out.PathCandidates, err = loadPathCandidates(ctx, tx, lexicalRequest.PathAnchors, lexicalRequest.FTS.CandidateK, out.Chunks)
		if err != nil {
			return HybridSearchSnapshot{}, err
		}
	}
	rows, err := tx.QueryContext(ctx, `SELECT s.id,s.chunk_id,s.canonical_input_sha256,s.display_start_byte,s.display_end_byte FROM embedding_segments s JOIN meta m ON m.id=1 WHERE s.serving_profile=m.active_serving_profile ORDER BY s.id`)
	if err != nil {
		return HybridSearchSnapshot{}, err
	}
	for rows.Next() {
		var segment HybridSegment
		if err := rows.Scan(&segment.ID, &segment.ChunkID, &segment.CanonicalInputSHA256, &segment.DisplayStart, &segment.DisplayEnd); err != nil {
			rows.Close()
			return HybridSearchSnapshot{}, err
		}
		out.Segments = append(out.Segments, segment)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return HybridSearchSnapshot{}, err
	}
	if err := rows.Close(); err != nil {
		return HybridSearchSnapshot{}, err
	}
	rows, err = tx.QueryContext(ctx, `
		SELECT DISTINCT c.id,f.path,f.language,c.kind,c.symbol,c.qualified_symbol,c.signature,c.start_byte,c.end_byte,c.start_line,c.end_line,c.source_body,f.indexed_sha256
		FROM embedding_segments s JOIN meta m ON m.id=1 JOIN chunks c ON c.id=s.chunk_id JOIN files f ON f.id=c.file_id
		WHERE s.serving_profile=m.active_serving_profile ORDER BY c.id`)
	if err != nil {
		return HybridSearchSnapshot{}, err
	}
	for rows.Next() {
		chunk, err := scanChunk(rows)
		if err != nil {
			rows.Close()
			return HybridSearchSnapshot{}, err
		}
		if err := validateChunk(chunk); err != nil {
			out.InvalidVectorRows = true
		}
		if _, exists := out.Chunks[chunk.ID]; !exists {
			chunk.SourceBody = append([]byte(nil), chunk.SourceBody...)
			out.Chunks[chunk.ID] = chunk
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return HybridSearchSnapshot{}, err
	}
	if err := rows.Close(); err != nil {
		return HybridSearchSnapshot{}, err
	}
	validByKey, presentByKey := map[string]bool{}, map[string]bool{}
	rows, err = tx.QueryContext(ctx, `
		WITH active_keys AS (
			SELECT DISTINCT s.canonical_input_sha256 AS canonical_input_sha256
			FROM embedding_segments s JOIN meta m ON m.id=1
			WHERE s.serving_profile=m.active_serving_profile
		)
		SELECT k.canonical_input_sha256,v.dimensions,v.codec_id,v.codec_version,v.blob,v.scale,v.norm,v.source_profile,v.vector_space_profile,v.raw_vector_sha256,v.materialization_fingerprint,v.materialized_at
		FROM active_keys k JOIN meta m ON m.id=1
		LEFT JOIN vector_cache v ON v.serving_profile=m.active_serving_profile AND v.canonical_input_sha256=k.canonical_input_sha256
		ORDER BY k.canonical_input_sha256`)
	if err != nil {
		return HybridSearchSnapshot{}, err
	}
	for rows.Next() {
		key, stored, present, valid, err := scanStoredVector(rows, resolved)
		if err != nil {
			rows.Close()
			return HybridSearchSnapshot{}, err
		}
		presentByKey[key], validByKey[key] = present, valid
		if present && valid {
			out.Vectors[key] = stored
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return HybridSearchSnapshot{}, err
	}
	if err := rows.Close(); err != nil {
		return HybridSearchSnapshot{}, err
	}
	for _, segment := range out.Segments {
		out.CoverageDenominator++
		chunk, ok := out.Chunks[segment.ChunkID]
		if !ok || validateChunk(chunk) != nil || segment.DisplayStart < 0 || segment.DisplayEnd <= segment.DisplayStart || segment.DisplayEnd > len(chunk.SourceBody) {
			out.InvalidVectorRows = true
			continue
		}
		if validByKey[segment.CanonicalInputSHA256] {
			out.CoverageNumerator++
		} else if presentByKey[segment.CanonicalInputSHA256] {
			out.InvalidVectorRows = true
		}
	}
	if err := tx.Commit(); err != nil {
		return HybridSearchSnapshot{}, err
	}
	return out, nil
}

func validateSnapshotRequest(request HybridSnapshotRequest) error {
	if request.FTS.MatchExpression == "" || request.FTS.CandidateK <= 0 || !finitePositive(request.FTS.SymbolWeight) || !finitePositive(request.FTS.BodyWeight) {
		return fmt.Errorf("invalid hybrid snapshot request")
	}
	for _, token := range request.FTS.SelectedTokens {
		if !safeLexicalToken(token) {
			return fmt.Errorf("invalid descriptive token")
		}
	}
	for _, anchors := range [][]LexicalAnchor{request.SymbolAnchors, request.PathAnchors} {
		for _, anchor := range anchors {
			if anchor.Raw == "" || anchor.Normalized == "" {
				return fmt.Errorf("invalid lexical anchor")
			}
		}
	}
	return nil
}

func loadFTSCandidates(ctx context.Context, tx *sql.Tx, request FTSSearchRequest, existing map[int64]HybridChunk) ([]HybridFTSCandidate, map[int64]HybridChunk, error) {
	rows, err := tx.QueryContext(ctx, `WITH fts_matches AS (SELECT rowid AS chunk_id,-bm25(chunk_fts,?,?) AS bm25_score FROM chunk_fts WHERE chunk_fts MATCH ?), candidates AS (SELECT m.chunk_id,m.bm25_score,c.id IS NULL OR f.id IS NULL AS orphaned,COALESCE(f.path,'') AS path,COALESCE(f.language,'') AS language,COALESCE(c.kind,'') AS kind,COALESCE(c.symbol,'') AS symbol,COALESCE(c.qualified_symbol,'') AS qualified_symbol,COALESCE(c.signature,'') AS signature,COALESCE(c.start_byte,0) AS start_byte,COALESCE(c.end_byte,0) AS end_byte,COALESCE(c.start_line,0) AS start_line,COALESCE(c.end_line,0) AS end_line,c.source_body AS source_body,COALESCE(f.indexed_sha256,'') AS indexed_sha256,CASE WHEN c.id IS NOT NULL AND EXISTS(SELECT 1 FROM symbols s WHERE s.chunk_id=c.id AND s.original_name=c.qualified_symbol AND s.normalized_name=?) THEN 1 ELSE 0 END AS exact_qualified_symbol FROM fts_matches m LEFT JOIN chunks c ON c.id=m.chunk_id LEFT JOIN files f ON f.id=c.file_id) SELECT chunk_id,bm25_score,orphaned,path,language,kind,symbol,qualified_symbol,signature,start_byte,end_byte,start_line,end_line,source_body,indexed_sha256,exact_qualified_symbol FROM candidates ORDER BY bm25_score DESC,exact_qualified_symbol DESC,path ASC,qualified_symbol ASC,chunk_id ASC LIMIT ?`, request.SymbolWeight, request.BodyWeight, request.MatchExpression, request.ExactNormalizedSymbol, request.CandidateK)
	if err != nil {
		return nil, nil, fmt.Errorf("FTS query failed: %w", err)
	}
	chunks, result := map[int64]HybridChunk{}, []HybridFTSCandidate{}
	for rows.Next() {
		var item HybridFTSCandidate
		var orphaned, exact int
		var body []byte
		var sha string
		if err := rows.Scan(&item.ChunkID, &item.BM25Score, &orphaned, &item.Path, &item.Language, &item.Kind, &item.Symbol, &item.QualifiedSymbol, &item.Signature, &item.StartByte, &item.EndByte, &item.StartLine, &item.EndLine, &body, &sha, &exact); err != nil {
			rows.Close()
			return nil, nil, err
		}
		if orphaned != 0 {
			rows.Close()
			return nil, nil, fmt.Errorf("%w: FTS row %d has no authoritative chunk", ErrIndexCorrupt, item.ChunkID)
		}
		item.IndexedSHA256 = sha
		item.ExactQualifiedSymbol = exact != 0
		chunk := HybridChunk{ID: item.ChunkID, Path: item.Path, Language: item.Language, Kind: item.Kind, Symbol: item.Symbol, QualifiedSymbol: item.QualifiedSymbol, Signature: item.Signature, StartByte: item.StartByte, EndByte: item.EndByte, StartLine: item.StartLine, EndLine: item.EndLine, SourceBody: append([]byte(nil), body...), IndexedSHA256: sha}
		if err := validateChunk(chunk); err != nil {
			rows.Close()
			return nil, nil, err
		}
		if known, ok := existing[item.ChunkID]; ok {
			chunk = known
		}
		chunks[item.ChunkID] = chunk
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, nil, err
	}
	if err := loadMatchedTermCounts(ctx, tx, request.SelectedTokens, result); err != nil {
		return nil, nil, err
	}
	return result, chunks, nil
}

func loadMatchedTermCounts(ctx context.Context, tx *sql.Tx, tokens []string, candidates []HybridFTSCandidate) error {
	if len(tokens) == 0 || len(candidates) == 0 {
		return nil
	}
	byID := make(map[int64]int, len(candidates))
	placeholders := make([]string, len(candidates))
	for index := range candidates {
		byID[candidates[index].ChunkID] = index
		placeholders[index] = "?"
		candidates[index].SelectedTerms = len(tokens)
	}
	query := `SELECT rowid FROM chunk_fts WHERE chunk_fts MATCH ? AND rowid IN (` + strings.Join(placeholders, ",") + `)`
	for _, token := range tokens {
		arguments := make([]any, 0, len(candidates)+1)
		arguments = append(arguments, `"`+token+`"`)
		for _, candidate := range candidates {
			arguments = append(arguments, candidate.ChunkID)
		}
		rows, err := tx.QueryContext(ctx, query, arguments...)
		if err != nil {
			return err
		}
		for rows.Next() {
			var chunkID int64
			if err := rows.Scan(&chunkID); err != nil {
				rows.Close()
				return err
			}
			candidateIndex, exists := byID[chunkID]
			if !exists {
				rows.Close()
				return fmt.Errorf("unexpected FTS coverage row")
			}
			candidates[candidateIndex].MatchedTerms++
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
	}
	return nil
}

func safeLexicalToken(token string) bool {
	if token == "" {
		return false
	}
	for _, r := range token {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

type orderedSymbolCandidate struct {
	HybridSymbolCandidate
	anchorOrder int
}

func loadSymbolCandidates(ctx context.Context, tx *sql.Tx, anchors []LexicalAnchor, candidateK int, chunks map[int64]HybridChunk) ([]HybridSymbolCandidate, error) {
	best := make(map[int64]orderedSymbolCandidate)
	for anchorIndex, anchor := range anchors {
		rows, err := tx.QueryContext(ctx, `
			SELECT CASE
			         WHEN s.original_name=? AND s.original_name=c.qualified_symbol THEN 1
			         WHEN s.original_name=? AND s.original_name=c.symbol THEN 2
			         WHEN s.normalized_name=? AND s.original_name=c.qualified_symbol THEN 3
			         WHEN s.normalized_name=? AND s.original_name=c.symbol THEN 4
			         ELSE 5
			       END AS match_tier,
			       c.id,f.path,f.language,c.kind,c.symbol,c.qualified_symbol,c.signature,
			       c.start_byte,c.end_byte,c.start_line,c.end_line,c.source_body,f.indexed_sha256
			FROM symbols s JOIN chunks c ON c.id=s.chunk_id JOIN files f ON f.id=c.file_id
			WHERE s.original_name=? OR s.normalized_name=?
			ORDER BY match_tier,f.path,c.qualified_symbol,c.start_byte,c.id
			LIMIT ?`, anchor.Raw, anchor.Raw, anchor.Normalized, anchor.Normalized, anchor.Raw, anchor.Normalized, candidateK)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var tier int
			var chunk HybridChunk
			if err := rows.Scan(&tier, &chunk.ID, &chunk.Path, &chunk.Language, &chunk.Kind, &chunk.Symbol, &chunk.QualifiedSymbol, &chunk.Signature, &chunk.StartByte, &chunk.EndByte, &chunk.StartLine, &chunk.EndLine, &chunk.SourceBody, &chunk.IndexedSHA256); err != nil {
				rows.Close()
				return nil, err
			}
			if err := validateChunk(chunk); err != nil {
				rows.Close()
				return nil, err
			}
			chunk.SourceBody = append([]byte(nil), chunk.SourceBody...)
			chunks[chunk.ID] = chunk
			candidate := orderedSymbolCandidate{HybridSymbolCandidate: HybridSymbolCandidate{ChunkID: chunk.ID, MatchTier: tier, MatchedAnchor: anchor.Normalized}, anchorOrder: anchorIndex}
			known, exists := best[chunk.ID]
			if !exists || symbolCandidateBefore(candidate, known, chunks) {
				best[chunk.ID] = candidate
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	}
	ordered := make([]orderedSymbolCandidate, 0, len(best))
	for _, candidate := range best {
		ordered = append(ordered, candidate)
	}
	sort.Slice(ordered, func(i, j int) bool { return symbolCandidateBefore(ordered[i], ordered[j], chunks) })
	if len(ordered) > candidateK {
		ordered = ordered[:candidateK]
	}
	result := make([]HybridSymbolCandidate, len(ordered))
	for index := range ordered {
		result[index] = ordered[index].HybridSymbolCandidate
	}
	return result, nil
}

func symbolCandidateBefore(left, right orderedSymbolCandidate, chunks map[int64]HybridChunk) bool {
	if left.MatchTier != right.MatchTier {
		return left.MatchTier < right.MatchTier
	}
	if left.anchorOrder != right.anchorOrder {
		return left.anchorOrder < right.anchorOrder
	}
	return chunkBefore(chunks[left.ChunkID], chunks[right.ChunkID])
}

type orderedPathCandidate struct {
	HybridPathCandidate
	anchorOrder int
}

func loadPathCandidates(ctx context.Context, tx *sql.Tx, anchors []LexicalAnchor, candidateK int, chunks map[int64]HybridChunk) ([]HybridPathCandidate, error) {
	best := make(map[int64]orderedPathCandidate)
	for anchorIndex, anchor := range anchors {
		suffixPattern := "%/" + escapeLike(anchor.Raw)
		normalizedPattern := "%" + strings.Join(strings.Fields(anchor.Normalized), "%") + "%"
		rows, err := tx.QueryContext(ctx, `
			SELECT CASE WHEN f.path=? THEN 1 WHEN f.path LIKE ? ESCAPE '!' THEN 2 ELSE 3 END AS match_tier,
			       c.id,f.path,f.language,c.kind,c.symbol,c.qualified_symbol,c.signature,
			       c.start_byte,c.end_byte,c.start_line,c.end_line,c.source_body,f.indexed_sha256
			FROM files f JOIN chunks c ON c.file_id=f.id
			WHERE f.path=? OR f.path LIKE ? ESCAPE '!' OR LOWER(f.path) LIKE ? ESCAPE '!'
			ORDER BY match_tier,f.path,c.start_byte,c.qualified_symbol,c.id
			LIMIT ?`, anchor.Raw, suffixPattern, anchor.Raw, suffixPattern, normalizedPattern, candidateK)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var tier int
			var chunk HybridChunk
			if err := rows.Scan(&tier, &chunk.ID, &chunk.Path, &chunk.Language, &chunk.Kind, &chunk.Symbol, &chunk.QualifiedSymbol, &chunk.Signature, &chunk.StartByte, &chunk.EndByte, &chunk.StartLine, &chunk.EndLine, &chunk.SourceBody, &chunk.IndexedSHA256); err != nil {
				rows.Close()
				return nil, err
			}
			if err := validateChunk(chunk); err != nil {
				rows.Close()
				return nil, err
			}
			chunk.SourceBody = append([]byte(nil), chunk.SourceBody...)
			chunks[chunk.ID] = chunk
			candidate := orderedPathCandidate{HybridPathCandidate: HybridPathCandidate{ChunkID: chunk.ID, MatchTier: tier, MatchedAnchor: anchor.Normalized}, anchorOrder: anchorIndex}
			known, exists := best[chunk.ID]
			if !exists || pathCandidateBefore(candidate, known, chunks) {
				best[chunk.ID] = candidate
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	}
	ordered := make([]orderedPathCandidate, 0, len(best))
	for _, candidate := range best {
		ordered = append(ordered, candidate)
	}
	sort.Slice(ordered, func(i, j int) bool { return pathCandidateBefore(ordered[i], ordered[j], chunks) })
	if len(ordered) > candidateK {
		ordered = ordered[:candidateK]
	}
	result := make([]HybridPathCandidate, len(ordered))
	for index := range ordered {
		result[index] = ordered[index].HybridPathCandidate
	}
	return result, nil
}

func pathCandidateBefore(left, right orderedPathCandidate, chunks map[int64]HybridChunk) bool {
	if left.MatchTier != right.MatchTier {
		return left.MatchTier < right.MatchTier
	}
	if left.anchorOrder != right.anchorOrder {
		return left.anchorOrder < right.anchorOrder
	}
	return chunkBefore(chunks[left.ChunkID], chunks[right.ChunkID])
}

func chunkBefore(left, right HybridChunk) bool {
	if left.Path != right.Path {
		return left.Path < right.Path
	}
	if left.QualifiedSymbol != right.QualifiedSymbol {
		return left.QualifiedSymbol < right.QualifiedSymbol
	}
	if left.StartByte != right.StartByte {
		return left.StartByte < right.StartByte
	}
	return left.ID < right.ID
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, "!", "!!")
	value = strings.ReplaceAll(value, "%", "!%")
	return strings.ReplaceAll(value, "_", "!_")
}

func scanChunk(rows *sql.Rows) (HybridChunk, error) {
	var out HybridChunk
	err := rows.Scan(&out.ID, &out.Path, &out.Language, &out.Kind, &out.Symbol, &out.QualifiedSymbol, &out.Signature, &out.StartByte, &out.EndByte, &out.StartLine, &out.EndLine, &out.SourceBody, &out.IndexedSHA256)
	return out, err
}
func validateChunk(chunk HybridChunk) error {
	if chunk.ID <= 0 || chunk.StartByte < 0 || chunk.EndByte <= chunk.StartByte || chunk.EndByte-chunk.StartByte != len(chunk.SourceBody) || chunk.StartLine <= 0 || chunk.EndLine < chunk.StartLine {
		return fmt.Errorf("invalid indexed chunk")
	}
	return nil
}

func scanStoredVector(rows *sql.Rows, resolved config.ResolvedConfig) (string, vector.StoredVector, bool, bool, error) {
	var key string
	var scan snapshotVectorScan
	if err := rows.Scan(&key, &scan.dimensions, &scan.codec, &scan.version, &scan.blob, &scan.scale, &scan.norm, &scan.source, &scan.space, &scan.rawSHA, &scan.materialization, &scan.at); err != nil {
		return "", vector.StoredVector{}, false, false, err
	}
	stored, present, valid := scan.stored(resolved)
	return key, stored, present, valid, nil
}

type snapshotVectorScan struct {
	dimensions, version                               sql.NullInt64
	codec, source, space, rawSHA, materialization, at sql.NullString
	blob                                              []byte
	scale, norm                                       sql.NullFloat64
}

func (scan *snapshotVectorScan) stored(resolved config.ResolvedConfig) (vector.StoredVector, bool, bool) {
	if !scan.dimensions.Valid || !scan.codec.Valid || !scan.version.Valid {
		return vector.StoredVector{}, false, false
	}
	stored := vector.StoredVector{Dimensions: int(scan.dimensions.Int64), CodecID: scan.codec.String, CodecVersion: uint16(scan.version.Int64), Blob: append([]byte(nil), scan.blob...)}
	if scan.scale.Valid {
		stored.Scale = float32(scan.scale.Float64)
	}
	if scan.norm.Valid {
		stored.Norm = float32(scan.norm.Float64)
	}
	if !scan.source.Valid || !scan.space.Valid || !scan.rawSHA.Valid || !scan.materialization.Valid || !scan.at.Valid {
		return stored, true, false
	}
	if _, err := time.Parse(time.RFC3339Nano, scan.at.String); err != nil || scan.source.String != string(resolved.Profiles.Fingerprints.Source) || scan.space.String != string(resolved.Profiles.Fingerprints.VectorSpace) || scan.materialization.String != string(resolved.Profiles.Fingerprints.VectorStorage) || !validSHA256(scan.rawSHA.String) {
		return stored, true, false
	}
	return stored, true, ValidateServingVector(resolved, stored) == nil
}

func readAppliedProfiles(ctx context.Context, reader interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}) (config.AppliedProfiles, error) {
	var out config.AppliedProfiles
	var index, canonical, source, space, storage, serving string
	if err := reader.QueryRowContext(ctx, `SELECT schema_version,active_generation,manifest_sha256,index_profile,canonical_text_profile,source_profile,vector_space_profile,vector_storage_profile,active_serving_profile FROM meta WHERE id=1`).Scan(&out.SchemaVersion, &out.ActiveGeneration, &out.ManifestSHA256, &index, &canonical, &source, &space, &storage, &serving); err != nil {
		return out, err
	}
	if out.SchemaVersion != ProductionSchemaVersion || out.ActiveGeneration < 0 || serving == "" {
		return out, fmt.Errorf("invalid production search metadata")
	}
	out.ActiveServingProfile = configFingerprint(serving)
	out.Fingerprints.Index, out.Fingerprints.CanonicalText, out.Fingerprints.Source, out.Fingerprints.VectorSpace, out.Fingerprints.VectorStorage = configFingerprint(index), configFingerprint(canonical), configFingerprint(source), configFingerprint(space), configFingerprint(storage)
	return out, nil
}
func profilesMatch(resolved config.ResolvedConfig, applied config.AppliedProfiles) bool {
	return resolved.ValidateIntegrity() == nil && applied.Fingerprints.Source == resolved.Profiles.Fingerprints.Source && applied.Fingerprints.VectorSpace == resolved.Profiles.Fingerprints.VectorSpace && applied.Fingerprints.VectorStorage == resolved.Profiles.Fingerprints.VectorStorage && applied.ActiveServingProfile == resolved.Profiles.Fingerprints.VectorStorage
}
