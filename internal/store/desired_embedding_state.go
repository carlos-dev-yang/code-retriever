package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"cidx/internal/config"
	"cidx/internal/vector"
)

const desiredStateBatch = 900

// DesiredEmbeddingStates classifies arbitrary future segment keys against the
// requested serving profile. It deliberately does not require meta.active to
// match: index dry-runs must preview profile reconciliation without writes.
func (store *ProductionStore) DesiredEmbeddingStates(ctx context.Context, resolved config.ResolvedConfig, hashes []string) (map[string]EmbeddingState, error) {
	if err := resolved.ValidateIntegrity(); err != nil {
		return nil, err
	}
	unique := map[string]bool{}
	for _, hash := range hashes {
		if !validSHA256(hash) {
			return nil, fmt.Errorf("invalid canonical input hash")
		}
		unique[hash] = true
	}
	out := map[string]EmbeddingState{}
	if len(unique) == 0 {
		return out, nil
	}
	keys := make([]string, 0, len(unique))
	for key := range unique {
		keys = append(keys, key)
	}
	tx, err := store.Read.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	result, err := desiredEmbeddingStates(ctx, tx, resolved, keys)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

type desiredStateReader interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func desiredEmbeddingStates(ctx context.Context, reader desiredStateReader, resolved config.ResolvedConfig, keys []string) (map[string]EmbeddingState, error) {
	out := map[string]EmbeddingState{}
	for start := 0; start < len(keys); start += desiredStateBatch {
		end := start + desiredStateBatch
		if end > len(keys) {
			end = len(keys)
		}
		marks := make([]string, end-start)
		args := make([]any, 0, end-start+2)
		for i, key := range keys[start:end] {
			marks[i] = "(?)"
			args = append(args, key)
		}
		query := `WITH keys(hash) AS (VALUES ` + strings.Join(marks, ",") + `) SELECT k.hash,v.dimensions,v.codec_id,v.codec_version,v.blob,v.scale,v.norm,v.source_profile,v.vector_space_profile,v.raw_vector_sha256,v.materialization_fingerprint,v.materialized_at,COALESCE((SELECT classification FROM embedding_failures f WHERE f.source_profile=? AND f.canonical_input_sha256=k.hash ORDER BY f.id DESC LIMIT 1),'') FROM keys k LEFT JOIN vector_cache v ON v.canonical_input_sha256=k.hash AND v.serving_profile=?` // source must precede profile placeholder
		args = append(args, string(resolved.Profiles.Fingerprints.Source), string(resolved.Profiles.Fingerprints.VectorStorage))
		rows, err := reader.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var hash, failure string
			var dimensions, version sql.NullInt64
			var codec, source, space, raw, materialized, at sql.NullString
			var blob []byte
			var scale, norm sql.NullFloat64
			if err := rows.Scan(&hash, &dimensions, &codec, &version, &blob, &scale, &norm, &source, &space, &raw, &materialized, &at, &failure); err != nil {
				rows.Close()
				return nil, err
			}
			valid := false
			if dimensions.Valid && codec.Valid && version.Valid && source.Valid && space.Valid && raw.Valid && materialized.Valid && at.Valid {
				stored := vector.StoredVector{Dimensions: int(dimensions.Int64), CodecID: codec.String, CodecVersion: uint16(version.Int64), Blob: blob}
				if scale.Valid {
					stored.Scale = float32(scale.Float64)
				}
				if norm.Valid {
					stored.Norm = float32(norm.Float64)
				}
				_, timeErr := time.Parse(time.RFC3339Nano, at.String)
				valid = source.String == string(resolved.Profiles.Fingerprints.Source) && space.String == string(resolved.Profiles.Fingerprints.VectorSpace) && materialized.String == string(resolved.Profiles.Fingerprints.VectorStorage) && validSHA256(raw.String) && timeErr == nil && ValidateServingVector(resolved, stored) == nil
			}
			out[hash] = DeriveEmbeddingState(valid, failure == "terminal")
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	}
	return out, nil
}
