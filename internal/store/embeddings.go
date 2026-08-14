package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"cidx/internal/config"
	"cidx/internal/vector"
)

type ActiveSegmentState struct {
	SegmentID int64
	State     EmbeddingState
}

// ActiveSegmentStates derives readiness from the committed active profile.
// It never trusts a mutable readiness bit or an inactive/stale vector row.
func (store *ProductionStore) ActiveSegmentStates(ctx context.Context, resolved config.ResolvedConfig) ([]ActiveSegmentState, error) {
	tx, err := store.Read.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := requireResolvedActiveProfile(ctx, tx, resolved); err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT s.id, v.dimensions, v.codec_id, v.codec_version, v.blob, v.scale, v.norm, v.source_profile, v.vector_space_profile, v.raw_vector_sha256, v.materialization_fingerprint, v.materialized_at,
		       COALESCE((SELECT classification FROM embedding_failures f WHERE f.source_profile=m.source_profile AND f.canonical_input_sha256=s.canonical_input_sha256 ORDER BY f.id DESC LIMIT 1),'')
		FROM embedding_segments s
		JOIN meta m ON m.id=1
		LEFT JOIN vector_cache v ON v.serving_profile=m.active_serving_profile AND v.canonical_input_sha256=s.canonical_input_sha256
		WHERE s.serving_profile=m.active_serving_profile
		ORDER BY s.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var states []ActiveSegmentState
	for rows.Next() {
		var id int64
		var dimensions, version sql.NullInt64
		var codec sql.NullString
		var blob []byte
		var sourceProfile, spaceProfile, rawSHA, materialization, materializedAt sql.NullString
		var scale, norm sql.NullFloat64
		var failureClass string
		if err := rows.Scan(&id, &dimensions, &codec, &version, &blob, &scale, &norm, &sourceProfile, &spaceProfile, &rawSHA, &materialization, &materializedAt, &failureClass); err != nil {
			return nil, err
		}
		valid := false
		if dimensions.Valid && codec.Valid && version.Valid {
			stored := vector.StoredVector{Dimensions: int(dimensions.Int64), CodecID: codec.String, CodecVersion: uint16(version.Int64), Blob: blob}
			if scale.Valid {
				stored.Scale = float32(scale.Float64)
			}
			if norm.Valid {
				stored.Norm = float32(norm.Float64)
			}
			_, timestampErr := time.Parse(time.RFC3339Nano, materializedAt.String)
			valid = sourceProfile.Valid && spaceProfile.Valid && rawSHA.Valid && materialization.Valid && materializedAt.Valid && sourceProfile.String == string(resolved.Profiles.Fingerprints.Source) && spaceProfile.String == string(resolved.Profiles.Fingerprints.VectorSpace) && materialization.String == string(resolved.Profiles.Fingerprints.VectorStorage) && validSHA256(rawSHA.String) && timestampErr == nil && ValidateServingVector(resolved, stored) == nil
		}
		states = append(states, ActiveSegmentState{SegmentID: id, State: DeriveEmbeddingState(valid, failureClass == "terminal")})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return states, nil
}

func (store *ProductionStore) ActiveCoverage(ctx context.Context, resolved config.ResolvedConfig) (ready, total int, err error) {
	states, err := store.ActiveSegmentStates(ctx, resolved)
	if err != nil {
		return 0, 0, err
	}
	for _, state := range states {
		total++
		if state.State == EmbeddingReady {
			ready++
		}
	}
	return ready, total, nil
}

type activeProfileReader interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func requireResolvedActiveProfile(ctx context.Context, reader activeProfileReader, resolved config.ResolvedConfig) error {
	if err := resolved.ValidateIntegrity(); err != nil {
		return err
	}
	var active string
	if err := reader.QueryRowContext(ctx, `SELECT active_serving_profile FROM meta WHERE id=1`).Scan(&active); err != nil {
		return err
	}
	if active != string(resolved.Profiles.Fingerprints.VectorStorage) {
		return fmt.Errorf("resolved serving profile does not match active production profile")
	}
	return nil
}

// UpsertServingVector validates the concrete active representation and makes a
// successful materialization clear the corresponding prior failure atomically.
func (store *ProductionStore) UpsertServingVector(ctx context.Context, resolved config.ResolvedConfig, inputHash, materializationFingerprint, rawVectorSHA256 string, stored vector.StoredVector) error {
	if !validSHA256(inputHash) || materializationFingerprint != string(resolved.Profiles.Fingerprints.VectorStorage) || !validSHA256(rawVectorSHA256) {
		return fmt.Errorf("vector input hash and materialization fingerprint are required")
	}
	if err := ValidateServingVector(resolved, stored); err != nil {
		return err
	}
	tx, err := store.Write.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := requireResolvedActiveProfile(ctx, tx, resolved); err != nil {
		return err
	}
	profile := string(resolved.Profiles.Fingerprints.VectorStorage)
	if _, err := tx.ExecContext(ctx, `INSERT INTO vector_cache(serving_profile,canonical_input_sha256,dimensions,codec_id,codec_version,blob,scale,norm,materialization_fingerprint,source_profile,vector_space_profile,raw_vector_sha256,materialized_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(serving_profile,canonical_input_sha256) DO UPDATE SET dimensions=excluded.dimensions,codec_id=excluded.codec_id,codec_version=excluded.codec_version,blob=excluded.blob,scale=excluded.scale,norm=excluded.norm,materialization_fingerprint=excluded.materialization_fingerprint,source_profile=excluded.source_profile,vector_space_profile=excluded.vector_space_profile,raw_vector_sha256=excluded.raw_vector_sha256,materialized_at=excluded.materialized_at`, profile, inputHash, stored.Dimensions, stored.CodecID, stored.CodecVersion, stored.Blob, nullableFloat(stored.Scale), nullableFloat(stored.Norm), materializationFingerprint, string(resolved.Profiles.Fingerprints.Source), string(resolved.Profiles.Fingerprints.VectorSpace), rawVectorSHA256, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM embedding_failures WHERE source_profile=? AND canonical_input_sha256=?`, string(resolved.Profiles.Fingerprints.Source), inputHash); err != nil {
		return err
	}
	return tx.Commit()
}

func nullableFloat(value float32) any {
	if value == 0 {
		return nil
	}
	return value
}

func (store *ProductionStore) RecordEmbeddingFailure(ctx context.Context, resolved config.ResolvedConfig, inputHash, errorClass, message string) error {
	if inputHash == "" || errorClass == "" || message == "" {
		return fmt.Errorf("embedding failure fields are required")
	}
	tx, err := store.Write.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := requireResolvedActiveProfile(ctx, tx, resolved); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO embedding_failures(source_profile,canonical_input_sha256,classification,attempts,error_class,last_error,last_attempted_at) VALUES(?,?,?,?,?,?,?)`, string(resolved.Profiles.Fingerprints.Source), inputHash, "terminal", 1, errorClass, message, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return tx.Commit()
}

// ValidateServingVector checks integrity metadata against the one resolved
// active representation. Database values never select dimensions or codecs.
func ValidateServingVector(resolved config.ResolvedConfig, stored vector.StoredVector) error {
	if err := resolved.ValidateIntegrity(); err != nil {
		return err
	}
	if stored.Dimensions != resolved.Embedding.TargetDimensions {
		return fmt.Errorf("stored vector dimensions do not match active profile")
	}
	expected, err := expectedCodecID(resolved.Embedding.StorageCodec)
	if err != nil {
		return err
	}
	if stored.CodecID != expected {
		return fmt.Errorf("stored vector codec does not match active profile")
	}
	return stored.Validate()
}
func expectedCodecID(codec string) (string, error) {
	switch codec {
	case config.StorageCodecBinary:
		return vector.BinaryCodecID, nil
	case config.StorageCodecInt8:
		return vector.Int8CodecID, nil
	default:
		return "", fmt.Errorf("unsupported active storage codec")
	}
}
