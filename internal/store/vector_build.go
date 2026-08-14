package store

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"cidx/internal/config"
	"cidx/internal/vector"
)

// MaterializedVector is a fully validated, f32-free production candidate.
// It is prepared outside the production write transaction.
type MaterializedVector struct {
	CanonicalInputSHA256       string
	RawVectorSHA256            string
	MaterializationFingerprint string
	Stored                     vector.StoredVector
}

// VectorPublishExpectation pins a completed build to the index state from
// which its raw coverage was calculated.
type VectorPublishExpectation struct {
	Generation     int64
	ManifestSHA256 string
	ServingProfile string
}

// PublishMaterializedVectors replaces exactly the active serving profile's
// complete vector set in one transaction. A failed build or stale expectation
// leaves the previous set untouched.
func (store *ProductionStore) PublishMaterializedVectors(ctx context.Context, resolved config.ResolvedConfig, expected VectorPublishExpectation, rows []MaterializedVector) error {
	if err := resolved.ValidateIntegrity(); err != nil {
		return err
	}
	if expected.Generation < 0 || !validSHA256(expected.ManifestSHA256) || !validSHA256(expected.ServingProfile) {
		return fmt.Errorf("invalid vector publish expectation")
	}
	if expected.ServingProfile != string(resolved.Profiles.Fingerprints.VectorStorage) {
		return fmt.Errorf("PROFILE_RECONCILIATION_REQUIRED")
	}
	byHash := make(map[string]MaterializedVector, len(rows))
	for _, row := range rows {
		if !validSHA256(row.CanonicalInputSHA256) || !validSHA256(row.RawVectorSHA256) || row.MaterializationFingerprint != expected.ServingProfile {
			return fmt.Errorf("invalid materialized vector provenance")
		}
		if _, duplicate := byHash[row.CanonicalInputSHA256]; duplicate {
			return fmt.Errorf("duplicate materialized vector input")
		}
		if err := ValidateServingVector(resolved, row.Stored); err != nil {
			return err
		}
		byHash[row.CanonicalInputSHA256] = row
	}

	tx, err := store.Write.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var generation int64
	var manifest, active, source, space, storage string
	if err := tx.QueryRowContext(ctx, `SELECT active_generation,manifest_sha256,active_serving_profile,source_profile,vector_space_profile,vector_storage_profile FROM meta WHERE id=1`).Scan(&generation, &manifest, &active, &source, &space, &storage); err != nil {
		return err
	}
	if generation != expected.Generation || manifest != expected.ManifestSHA256 || active != expected.ServingProfile || storage != expected.ServingProfile || source != string(resolved.Profiles.Fingerprints.Source) || space != string(resolved.Profiles.Fingerprints.VectorSpace) {
		return fmt.Errorf("VECTOR_BUILD_STATE_CHANGED")
	}
	activeHashes := map[string]struct{}{}
	segments, err := tx.QueryContext(ctx, `SELECT DISTINCT canonical_input_sha256 FROM embedding_segments WHERE serving_profile=? ORDER BY canonical_input_sha256`, active)
	if err != nil {
		return err
	}
	for segments.Next() {
		var hash string
		if err := segments.Scan(&hash); err != nil {
			segments.Close()
			return err
		}
		activeHashes[hash] = struct{}{}
	}
	if err := segments.Err(); err != nil {
		segments.Close()
		return err
	}
	if err := segments.Close(); err != nil {
		return err
	}
	if len(activeHashes) != len(byHash) {
		return fmt.Errorf("RAW_COVERAGE_INCOMPLETE")
	}
	for hash := range activeHashes {
		if _, ok := byHash[hash]; !ok {
			return fmt.Errorf("RAW_COVERAGE_INCOMPLETE")
		}
	}

	// Delete only this profile inside the same transaction. Readers pinned
	// before commit retain the old set; new readers see all replacement rows.
	if _, err := tx.ExecContext(ctx, `DELETE FROM vector_cache WHERE serving_profile=?`, active); err != nil {
		return err
	}
	hashes := make([]string, 0, len(byHash))
	for hash := range byHash {
		hashes = append(hashes, hash)
	}
	sort.Strings(hashes)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, hash := range hashes {
		row := byHash[hash]
		if _, err := tx.ExecContext(ctx, `INSERT INTO vector_cache(serving_profile,canonical_input_sha256,dimensions,codec_id,codec_version,blob,scale,norm,materialization_fingerprint,source_profile,vector_space_profile,raw_vector_sha256,materialized_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, active, hash, row.Stored.Dimensions, row.Stored.CodecID, row.Stored.CodecVersion, row.Stored.Blob, nullableFloat(row.Stored.Scale), nullableFloat(row.Stored.Norm), row.MaterializationFingerprint, source, space, row.RawVectorSHA256, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

// GarbageCollectInactiveVectors is deliberately separate from publication.
// It cannot delete the currently active profile.
func (store *ProductionStore) GarbageCollectInactiveVectors(ctx context.Context) (int64, error) {
	tx, err := store.Write.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var active string
	if err := tx.QueryRowContext(ctx, `SELECT active_serving_profile FROM meta WHERE id=1`).Scan(&active); err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM vector_cache WHERE serving_profile<>?`, active)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
