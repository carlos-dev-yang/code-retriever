package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"cidx/internal/config"
	"cidx/internal/vector"
)

// StatusSnapshot is copied from one short read transaction before the
// application inspects live files. It contains no source bodies.
type StatusSnapshot struct {
	Applied                                                                       config.AppliedProfiles
	FilesByPath                                                                   map[string]IndexedFile
	Files, Chunks, Segments, CoverageReady, CoverageTotal, Ready, Pending, Failed int
	IndexAttemptedAt, IndexSucceededAt, EmbedAttemptedAt, EmbedSucceededAt        string
}

func (store *ProductionStore) StatusSnapshot(ctx context.Context, resolved config.ResolvedConfig) (StatusSnapshot, error) {
	if err := resolved.ValidateIntegrity(); err != nil {
		return StatusSnapshot{}, err
	}
	tx, err := store.Read.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return StatusSnapshot{}, err
	}
	defer tx.Rollback()
	var value StatusSnapshot
	var index, canonical, source, space, storage, serving string
	if err := tx.QueryRowContext(ctx, `SELECT schema_version,active_generation,manifest_sha256,index_profile,canonical_text_profile,source_profile,vector_space_profile,vector_storage_profile,active_serving_profile,index_attempted_at,index_succeeded_at,embed_attempted_at,embed_succeeded_at FROM meta WHERE id=1`).Scan(&value.Applied.SchemaVersion, &value.Applied.ActiveGeneration, &value.Applied.ManifestSHA256, &index, &canonical, &source, &space, &storage, &serving, &value.IndexAttemptedAt, &value.IndexSucceededAt, &value.EmbedAttemptedAt, &value.EmbedSucceededAt); err != nil {
		return value, err
	}
	if value.Applied.SchemaVersion != ProductionSchemaVersion || serving == "" {
		return value, fmt.Errorf("invalid production index metadata")
	}
	value.Applied.ActiveServingProfile = configFingerprint(serving)
	value.Applied.Fingerprints.Index, value.Applied.Fingerprints.CanonicalText, value.Applied.Fingerprints.Source, value.Applied.Fingerprints.VectorSpace, value.Applied.Fingerprints.VectorStorage = configFingerprint(index), configFingerprint(canonical), configFingerprint(source), configFingerprint(space), configFingerprint(storage)
	value.FilesByPath = map[string]IndexedFile{}
	rows, err := tx.QueryContext(ctx, `SELECT path,indexed_sha256 FROM files ORDER BY path`)
	if err != nil {
		return value, err
	}
	distinct := map[string]EmbeddingState{}
	for rows.Next() {
		var file IndexedFile
		if err := rows.Scan(&file.Path, &file.SHA256); err != nil {
			rows.Close()
			return value, err
		}
		value.FilesByPath[file.Path] = file
		value.Files++
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return value, err
	}
	if err := rows.Close(); err != nil {
		return value, err
	}
	if err := tx.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM chunks),(SELECT count(*) FROM embedding_segments)`).Scan(&value.Chunks, &value.Segments); err != nil {
		return value, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT s.canonical_input_sha256,v.dimensions,v.codec_id,v.codec_version,v.blob,v.scale,v.norm,v.source_profile,v.vector_space_profile,v.raw_vector_sha256,v.materialization_fingerprint,v.materialized_at,COALESCE((SELECT classification FROM embedding_failures f WHERE f.source_profile=m.source_profile AND f.canonical_input_sha256=s.canonical_input_sha256 ORDER BY f.id DESC LIMIT 1),'') FROM embedding_segments s JOIN meta m ON m.id=1 LEFT JOIN vector_cache v ON v.serving_profile=m.active_serving_profile AND v.canonical_input_sha256=s.canonical_input_sha256 WHERE s.serving_profile=m.active_serving_profile ORDER BY s.id`)
	if err != nil {
		return value, err
	}
	for rows.Next() {
		var hash, failure string
		var dimensions, version sql.NullInt64
		var codec, sourceRow, spaceRow, rawSHA, materialization, at sql.NullString
		var blob []byte
		var scale, norm sql.NullFloat64
		if err := rows.Scan(&hash, &dimensions, &codec, &version, &blob, &scale, &norm, &sourceRow, &spaceRow, &rawSHA, &materialization, &at, &failure); err != nil {
			rows.Close()
			return value, err
		}
		valid := false
		if dimensions.Valid && codec.Valid && version.Valid && sourceRow.Valid && spaceRow.Valid && rawSHA.Valid && materialization.Valid && at.Valid {
			stored := vector.StoredVector{Dimensions: int(dimensions.Int64), CodecID: codec.String, CodecVersion: uint16(version.Int64), Blob: blob}
			if scale.Valid {
				stored.Scale = float32(scale.Float64)
			}
			if norm.Valid {
				stored.Norm = float32(norm.Float64)
			}
			_, timeErr := time.Parse(time.RFC3339Nano, at.String)
			valid = sourceRow.String == string(resolved.Profiles.Fingerprints.Source) && spaceRow.String == string(resolved.Profiles.Fingerprints.VectorSpace) && materialization.String == string(resolved.Profiles.Fingerprints.VectorStorage) && validSHA256(rawSHA.String) && timeErr == nil && ValidateServingVector(resolved, stored) == nil
		}
		state := DeriveEmbeddingState(valid, failure == "terminal")
		value.CoverageTotal++
		if state == EmbeddingReady {
			value.CoverageReady++
		}
		prior := distinct[hash]
		if state == EmbeddingReady || prior == "" || (state == EmbeddingFailed && prior == EmbeddingPending) {
			distinct[hash] = state
		}
	}
	for _, state := range distinct {
		switch state {
		case EmbeddingReady:
			value.Ready++
		case EmbeddingFailed:
			value.Failed++
		default:
			value.Pending++
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return value, err
	}
	if err := rows.Close(); err != nil {
		return value, err
	}
	return value, tx.Commit()
}
