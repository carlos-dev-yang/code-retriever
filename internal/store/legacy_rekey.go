package store

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"cidx/internal/config"
	"cidx/internal/vector"
)

// ServingVectorRekey is a fully copied, immutable local vector reuse record.
// The legacy evidence fields are retained in the publish payload so its source
// identity remains auditable; the inserted row uses the desired R4 profiles.
type ServingVectorRekey struct {
	CanonicalInputSHA256                              string
	Stored                                            vector.StoredVector
	RawVectorSHA256, MaterializedAt                   string
	LegacySourceProfile, LegacyVectorSpaceProfile     string
	LegacyMaterializationFingerprint, LegacyServingID string
}

// PlanLegacyServingVectorRekeys proves, without touching a provider or lab
// database, whether inactive pre-R4 vectors may be copied to desired serving
// keys. Any failed proof deliberately produces no reuse rather than an error.
func (store *ProductionStore) PlanLegacyServingVectorRekeys(ctx context.Context, snapshot IndexSnapshot, desired config.ResolvedConfig, futureCanonicalInputs []string) ([]ServingVectorRekey, error) {
	metadata := config.LegacyServingProfileMetadata{
		CanonicalTextFingerprint: snapshot.Applied.Fingerprints.CanonicalText,
		CanonicalTextJSON:        snapshot.StoredProfiles.CanonicalTextJSON,
		SourceFingerprint:        snapshot.Applied.Fingerprints.Source,
		SourceJSON:               snapshot.StoredProfiles.SourceJSON,
		VectorSpaceFingerprint:   snapshot.Applied.Fingerprints.VectorSpace,
		VectorSpaceJSON:          snapshot.StoredProfiles.VectorSpaceJSON,
		VectorStorageFingerprint: snapshot.Applied.Fingerprints.VectorStorage,
		VectorStorageJSON:        snapshot.StoredProfiles.VectorStorageJSON,
		ActiveServingProfile:     snapshot.Applied.ActiveServingProfile,
	}
	if !config.LegacyServingProfileEquivalent(desired, metadata) {
		return nil, nil
	}
	keys := uniqueValidHashes(futureCanonicalInputs)
	if len(keys) == 0 {
		return nil, nil
	}
	tx, err := store.Read.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var generation int64
	var source, space, storage, active string
	if err := tx.QueryRowContext(ctx, `SELECT active_generation,source_profile,vector_space_profile,vector_storage_profile,active_serving_profile FROM meta WHERE id=1`).Scan(&generation, &source, &space, &storage, &active); err != nil {
		return nil, err
	}
	if generation != snapshot.Applied.ActiveGeneration || source != string(snapshot.Applied.Fingerprints.Source) || space != string(snapshot.Applied.Fingerprints.VectorSpace) || storage != string(snapshot.Applied.Fingerprints.VectorStorage) || active != string(snapshot.Applied.ActiveServingProfile) {
		return nil, nil
	}
	var plan []ServingVectorRekey
	for _, key := range keys {
		row, ok, err := readLegacyVector(ctx, tx, active, key)
		if err != nil {
			return nil, err
		}
		if !ok || row.LegacySourceProfile != source || row.LegacyVectorSpaceProfile != space || row.LegacyMaterializationFingerprint != storage || !validSHA256(row.RawVectorSHA256) {
			continue
		}
		if _, err := time.Parse(time.RFC3339Nano, row.MaterializedAt); err != nil || ValidateServingVector(desired, row.Stored) != nil {
			continue
		}
		row.LegacyServingID = active
		plan = append(plan, row)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return plan, nil
}

func uniqueValidHashes(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		if validSHA256(value) {
			set[value] = struct{}{}
		}
	}
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func readLegacyVector(ctx context.Context, tx *sql.Tx, servingProfile, inputHash string) (ServingVectorRekey, bool, error) {
	var row ServingVectorRekey
	var dimensions, version int
	var codec string
	var scale, norm sql.NullFloat64
	err := tx.QueryRowContext(ctx, `SELECT dimensions,codec_id,codec_version,blob,scale,norm,source_profile,vector_space_profile,raw_vector_sha256,materialization_fingerprint,materialized_at FROM vector_cache WHERE serving_profile=? AND canonical_input_sha256=?`, servingProfile, inputHash).Scan(&dimensions, &codec, &version, &row.Stored.Blob, &scale, &norm, &row.LegacySourceProfile, &row.LegacyVectorSpaceProfile, &row.RawVectorSHA256, &row.LegacyMaterializationFingerprint, &row.MaterializedAt)
	if err == sql.ErrNoRows {
		return ServingVectorRekey{}, false, nil
	}
	if err != nil {
		return ServingVectorRekey{}, false, fmt.Errorf("read legacy serving vector: %w", err)
	}
	row.CanonicalInputSHA256 = inputHash
	row.Stored.Dimensions, row.Stored.CodecID, row.Stored.CodecVersion = dimensions, codec, uint16(version)
	if scale.Valid {
		row.Stored.Scale = float32(scale.Float64)
	}
	if norm.Valid {
		row.Stored.Norm = float32(norm.Float64)
	}
	return row, true, nil
}
