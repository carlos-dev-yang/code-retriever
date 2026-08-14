package store

import (
	"context"
	"fmt"

	"cidx/internal/config"
	"cidx/internal/profile"
)

type EmbeddingState string

const (
	EmbeddingReady   EmbeddingState = "ready"
	EmbeddingPending EmbeddingState = "pending"
	EmbeddingFailed  EmbeddingState = "failed"
)

func DeriveEmbeddingState(validActiveVector bool, applicableFailure bool) EmbeddingState {
	if validActiveVector {
		return EmbeddingReady
	}
	if applicableFailure {
		return EmbeddingFailed
	}
	return EmbeddingPending
}

func (store *ProductionStore) AppliedProfiles(ctx context.Context) (config.AppliedProfiles, error) {
	var applied config.AppliedProfiles
	var index, canonical, source, space, storage, serving string
	err := store.Read.db.QueryRowContext(ctx, `SELECT schema_version,active_generation,manifest_sha256,index_profile,canonical_text_profile,source_profile,vector_space_profile,vector_storage_profile,active_serving_profile FROM meta WHERE id=1`).Scan(&applied.SchemaVersion, &applied.ActiveGeneration, &applied.ManifestSHA256, &index, &canonical, &source, &space, &storage, &serving)
	if err != nil {
		return config.AppliedProfiles{}, err
	}
	if applied.SchemaVersion <= 0 || applied.ActiveGeneration < 0 || serving == "" {
		return config.AppliedProfiles{}, fmt.Errorf("invalid applied production metadata")
	}
	applied.ActiveServingProfile = configFingerprint(serving)
	applied.Fingerprints.Index, applied.Fingerprints.CanonicalText, applied.Fingerprints.Source, applied.Fingerprints.VectorSpace, applied.Fingerprints.VectorStorage = configFingerprint(index), configFingerprint(canonical), configFingerprint(source), configFingerprint(space), configFingerprint(storage)
	return applied, nil
}
func configFingerprint(value string) profile.Fingerprint { return profile.Fingerprint(value) }
