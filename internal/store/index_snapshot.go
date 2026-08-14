package store

import (
	"context"
	"database/sql"
	"fmt"

	"cidx/internal/config"
)

// IndexSnapshot is the small, immutable planning view copied under one read
// transaction. It deliberately contains no database handle or live source.
type IndexSnapshot struct {
	Applied config.AppliedProfiles
	Files   map[string]IndexedFile
}
type IndexedSegment struct {
	ID                                                           int64
	Path, Kind, QualifiedSymbol, Signature, CanonicalInputSHA256 string
	SourceBody                                                   []byte
	Number, DisplayStart, DisplayEnd                             int
	Projections                                                  []PreparedIndexRange
}

type IndexedFile struct {
	Path   string
	SHA256 string
}

func (store *ProductionStore) IndexSnapshot(ctx context.Context) (IndexSnapshot, error) {
	tx, err := store.Read.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return IndexSnapshot{}, err
	}
	defer tx.Rollback()
	var result IndexSnapshot
	var index, canonical, source, space, storage, serving string
	if err := tx.QueryRowContext(ctx, `SELECT schema_version,active_generation,manifest_sha256,index_profile,canonical_text_profile,source_profile,vector_space_profile,vector_storage_profile,active_serving_profile FROM meta WHERE id=1`).Scan(&result.Applied.SchemaVersion, &result.Applied.ActiveGeneration, &result.Applied.ManifestSHA256, &index, &canonical, &source, &space, &storage, &serving); err != nil {
		return result, err
	}
	if result.Applied.SchemaVersion != ProductionSchemaVersion || serving == "" {
		return result, fmt.Errorf("invalid production index metadata")
	}
	result.Applied.ActiveServingProfile = configFingerprint(serving)
	result.Applied.Fingerprints.Index, result.Applied.Fingerprints.CanonicalText, result.Applied.Fingerprints.Source, result.Applied.Fingerprints.VectorSpace, result.Applied.Fingerprints.VectorStorage = configFingerprint(index), configFingerprint(canonical), configFingerprint(source), configFingerprint(space), configFingerprint(storage)
	rows, err := tx.QueryContext(ctx, `SELECT path,indexed_sha256 FROM files ORDER BY path`)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	result.Files = map[string]IndexedFile{}
	for rows.Next() {
		var f IndexedFile
		if err := rows.Scan(&f.Path, &f.SHA256); err != nil {
			return result, err
		}
		result.Files[f.Path] = f
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	if err := tx.Commit(); err != nil {
		return result, err
	}
	return result, nil
}
func (store *ProductionStore) ReconciliationSegments(ctx context.Context, expected int64) ([]IndexedSegment, error) {
	tx, err := store.Read.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var active int64
	if err := tx.QueryRowContext(ctx, `SELECT active_generation FROM meta WHERE id=1`).Scan(&active); err != nil {
		return nil, err
	}
	if active != expected {
		return nil, fmt.Errorf("BASE_GENERATION_CHANGED")
	}
	rows, err := tx.QueryContext(ctx, `SELECT s.id,f.path,c.kind,c.qualified_symbol,c.signature,s.canonical_input_sha256,c.source_body,s.segment_number,s.display_start_byte,s.display_end_byte FROM embedding_segments s JOIN chunks c ON c.id=s.chunk_id JOIN files f ON f.id=c.file_id ORDER BY s.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []IndexedSegment
	for rows.Next() {
		var s IndexedSegment
		if err := rows.Scan(&s.ID, &s.Path, &s.Kind, &s.QualifiedSymbol, &s.Signature, &s.CanonicalInputSHA256, &s.SourceBody, &s.Number, &s.DisplayStart, &s.DisplayEnd); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	projections, err := tx.QueryContext(ctx, `SELECT segment_id,start_byte,end_byte FROM segment_projections ORDER BY segment_id,ordinal`)
	if err != nil {
		return nil, err
	}
	byID := map[int64]int{}
	for i := range out {
		byID[out[i].ID] = i
	}
	for projections.Next() {
		var id int64
		var r PreparedIndexRange
		if err := projections.Scan(&id, &r.StartByte, &r.EndByte); err != nil {
			projections.Close()
			return nil, err
		}
		if i, ok := byID[id]; ok {
			out[i].Projections = append(out[i].Projections, r)
		}
	}
	if err := projections.Err(); err != nil {
		projections.Close()
		return nil, err
	}
	if err := projections.Close(); err != nil {
		return nil, err
	}
	return out, tx.Commit()
}
