package app

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"cidx/internal/config"
	"cidx/internal/embedlock"
	"cidx/internal/lab"
	"cidx/internal/store"
	"cidx/internal/vector"
)

// DevMaterialize is the unstable application use case that Phase 13 will
// expose as a development command. It opens both databases only here; serving
// and search packages retain no lab dependency.
type DevMaterialize struct {
	Production *store.ProductionStore
	Lab        *lab.Store
	Resolved   config.ResolvedConfig
}

type MaterializationPlan struct {
	Generation                                      int64
	RequiredRaw, RawHits, MissingRaw, ExpectedBytes int
	ManifestSHA256                                  string
	ServingProfile                                  string
	required                                        []string
}

type MaterializationResult struct {
	BuildID   string
	Staged    int
	Published bool
}

func (m DevMaterialize) Plan(ctx context.Context) (MaterializationPlan, error) {
	if m.Production == nil || m.Lab == nil {
		return MaterializationPlan{}, fmt.Errorf("production and lab stores are required")
	}
	if err := m.Resolved.ValidateIntegrity(); err != nil {
		return MaterializationPlan{}, err
	}
	if err := m.requireSameRepository(ctx); err != nil {
		return MaterializationPlan{}, err
	}
	snapshot, err := m.Production.ActiveVectorPlanningSnapshot(ctx)
	if err != nil {
		return MaterializationPlan{}, err
	}
	if err := m.requireCurrentProfile(snapshot.Applied); err != nil {
		return MaterializationPlan{}, err
	}
	required := append([]string(nil), snapshot.CanonicalInputs...)
	if !validDigest(snapshot.Applied.ManifestSHA256) {
		return MaterializationPlan{}, fmt.Errorf("invalid active manifest")
	}
	for _, hash := range required {
		if !validDigest(hash) {
			return MaterializationPlan{}, fmt.Errorf("active segment missing canonical input")
		}
	}
	hits, err := m.Lab.ExistingKeys(ctx, string(m.Resolved.Profiles.Fingerprints.Source), required)
	if err != nil {
		return MaterializationPlan{}, err
	}
	plan := MaterializationPlan{Generation: snapshot.Applied.ActiveGeneration, ManifestSHA256: snapshot.Applied.ManifestSHA256, ServingProfile: string(m.Resolved.Profiles.Fingerprints.VectorStorage), RequiredRaw: len(required), required: required}
	for _, hash := range required {
		if hits[hash] {
			plan.RawHits++
		} else {
			plan.MissingRaw++
		}
	}
	if m.Resolved.Embedding.StorageCodec == config.StorageCodecBinary {
		plan.ExpectedBytes = plan.RawHits * ((m.Resolved.Embedding.TargetDimensions + 7) / 8)
	} else {
		plan.ExpectedBytes = plan.RawHits * m.Resolved.Embedding.TargetDimensions
	}
	return plan, nil
}

// Activate performs only local transformations. It never constructs a
// provider client or reads an API key.
func (m DevMaterialize) Activate(ctx context.Context, plan MaterializationPlan) (result MaterializationResult, err error) {
	if m.Production == nil || m.Lab == nil {
		return result, fmt.Errorf("production and lab stores are required")
	}
	if err := m.Resolved.ValidateIntegrity(); err != nil {
		return result, err
	}
	if plan.MissingRaw != 0 {
		return result, fmt.Errorf("RAW_COVERAGE_INCOMPLETE")
	}
	release, err := embedlock.Acquire(ctx, m.Production.Root)
	if err != nil {
		return result, err
	}
	defer release()
	if err := m.requireSameRepository(ctx); err != nil {
		return result, err
	}
	snapshot, err := m.Production.ActiveVectorPlanningSnapshot(ctx)
	if err != nil {
		return result, err
	}
	if err := m.requireCurrentProfile(snapshot.Applied); err != nil {
		return result, err
	}
	if snapshot.Applied.ActiveGeneration != plan.Generation || snapshot.Applied.ManifestSHA256 != plan.ManifestSHA256 || string(snapshot.Applied.ActiveServingProfile) != plan.ServingProfile || !sameStrings(snapshot.CanonicalInputs, plan.required) {
		return result, fmt.Errorf("VECTOR_BUILD_STATE_CHANGED")
	}
	codec, err := vector.CodecForID(m.Resolved.Profiles.VectorStorage.StorageCodecID)
	if err != nil {
		return result, err
	}
	buildID := fmt.Sprintf("materialize-%d", time.Now().UTC().UnixNano())
	runID, err := m.Lab.StartMaterialization(ctx, lab.MaterializationRun{BuildID: buildID, Generation: plan.Generation, ManifestSHA256: plan.ManifestSHA256, SourceProfile: string(m.Resolved.Profiles.Fingerprints.Source), VectorSpaceProfile: string(m.Resolved.Profiles.Fingerprints.VectorSpace), StorageProfile: plan.ServingProfile, Planned: plan.RequiredRaw})
	if err != nil {
		return result, err
	}
	result.BuildID = buildID
	published := false
	defer func() {
		if published {
			return
		}
		_ = m.Lab.FinishMaterialization(context.Background(), runID, "failed", result.Staged, plan.MissingRaw, 0, "materialization failed")
	}()
	raws, err := m.Lab.RawDocuments(ctx, string(m.Resolved.Profiles.Fingerprints.Source), plan.required)
	if err != nil {
		return result, err
	}
	if len(raws) != plan.RequiredRaw {
		return result, fmt.Errorf("RAW_COVERAGE_INCOMPLETE")
	}
	variants := make([]lab.MaterializedVariant, 0, plan.RequiredRaw)
	transformer := vector.Transformer{Spec: m.Resolved.Embedding.TransformSpec()}
	for _, hash := range plan.required {
		raw := raws[hash]
		if raw.Dimensions != m.Resolved.Embedding.Model.SourceDimensions {
			return result, fmt.Errorf("raw source dimension mismatch")
		}
		decoded, decodeErr := lab.DecodeF32(raw.VectorF32LE, raw.Dimensions, raw.Checksum)
		if decodeErr != nil {
			return result, decodeErr
		}
		space, transformErr := transformer.Transform(decoded.Values)
		if transformErr != nil {
			return result, transformErr
		}
		stored, encodeErr := codec.Encode(space)
		if encodeErr != nil {
			return result, encodeErr
		}
		if validateErr := store.ValidateServingVector(m.Resolved, stored); validateErr != nil {
			return result, validateErr
		}
		variants = append(variants, lab.MaterializedVariant{InputHash: hash, RawVectorSHA256: raw.VectorSHA256, Stored: stored})
	}
	if len(variants) > 0 {
		if err := m.Lab.PutMaterializedVariants(ctx, runID, variants); err != nil {
			return result, err
		}
	}
	result.Staged = len(variants)
	if err := m.Lab.FinishMaterialization(ctx, runID, "ready", result.Staged, 0, 0, ""); err != nil {
		return result, err
	}
	staged, err := m.Lab.MaterializedVariants(ctx, runID, plan.ServingProfile)
	if err != nil {
		return result, err
	}
	if len(staged) != plan.RequiredRaw {
		return result, fmt.Errorf("RAW_COVERAGE_INCOMPLETE")
	}
	rows := make([]store.MaterializedVector, 0, len(staged))
	for _, item := range staged {
		rows = append(rows, store.MaterializedVector{CanonicalInputSHA256: item.InputHash, RawVectorSHA256: item.RawVectorSHA256, MaterializationFingerprint: plan.ServingProfile, Stored: item.Stored})
	}
	if err := m.Production.PublishMaterializedVectors(ctx, m.Resolved, store.VectorPublishExpectation{Generation: plan.Generation, ManifestSHA256: plan.ManifestSHA256, ServingProfile: plan.ServingProfile}, rows); err != nil {
		return result, err
	}
	result.Published = true
	published = true
	if err := m.Lab.FinishMaterialization(ctx, runID, "published", result.Staged, 0, 0, ""); err != nil {
		// Serving publication already committed; preserve that truth and let the
		// run remain ready for an operator-visible provenance repair.
		return result, err
	}
	return result, nil
}

func (m DevMaterialize) requireCurrentProfile(applied config.AppliedProfiles) error {
	if applied.Fingerprints.Source != m.Resolved.Profiles.Fingerprints.Source || applied.Fingerprints.VectorSpace != m.Resolved.Profiles.Fingerprints.VectorSpace || applied.Fingerprints.VectorStorage != m.Resolved.Profiles.Fingerprints.VectorStorage || applied.ActiveServingProfile != m.Resolved.Profiles.Fingerprints.VectorStorage {
		return fmt.Errorf("PROFILE_RECONCILIATION_REQUIRED")
	}
	return nil
}

func (m DevMaterialize) requireSameRepository(ctx context.Context) error {
	root, err := m.Lab.CanonicalRoot(ctx)
	if err != nil {
		return err
	}
	if root != m.Production.Root {
		return fmt.Errorf("lab database belongs to different root")
	}
	return nil
}

func validDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
