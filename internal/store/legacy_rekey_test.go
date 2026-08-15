package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"cidx/internal/config"
	"cidx/internal/profile"
)

type legacySpaceWire struct {
	SourceProfileFingerprint profile.Fingerprint `json:"source_profile_fingerprint"`
	TargetDimensions         int                 `json:"target_dimensions"`
	ReducerID                string              `json:"reducer_id"`
	NormalizerID             string              `json:"normalizer_id"`
	Metric                   string              `json:"metric"`
}

func TestLegacyServingVectorRekeyRequiresCompleteProofAndPublishesAtomically(t *testing.T) {
	ctx := context.Background()
	p, desired, inputHash, rekeys := legacyRekeyFixture(t)
	defer p.Close()
	if len(rekeys) != 1 {
		t.Fatalf("equivalent legacy row plan=%#v", rekeys)
	}
	snapshot, err := p.IndexSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	segments, err := p.ReconciliationSegments(ctx, snapshot.Applied.ActiveGeneration)
	if err != nil || len(segments) != 1 {
		t.Fatalf("segments=%#v err=%v", segments, err)
	}
	update := SegmentUpdate{ID: segments[0].ID, CanonicalInputSHA256: inputHash, CanonicalTextProfile: string(desired.Profiles.Fingerprints.CanonicalText), ServingProfile: string(desired.Profiles.Fingerprints.VectorStorage)}
	if err := p.PublishIndexGeneration(ctx, IndexPublishPlan{BaseGeneration: 1, NextGeneration: 2, ManifestSHA256: fixtureHash("rekey"), Reason: "manual", Desired: desired, SegmentUpdates: []SegmentUpdate{update}, ServingVectorRekeys: rekeys}); err != nil {
		t.Fatal(err)
	}
	var copied int
	var profile string
	if err := p.Read.db.QueryRowContext(ctx, `SELECT count(*),serving_profile FROM vector_cache WHERE canonical_input_sha256=? AND serving_profile=?`, inputHash, string(desired.Profiles.Fingerprints.VectorStorage)).Scan(&copied, &profile); err != nil || copied != 1 || profile != string(desired.Profiles.Fingerprints.VectorStorage) {
		t.Fatalf("copied=%d profile=%q err=%v", copied, profile, err)
	}
	if err := p.PublishIndexGeneration(ctx, IndexPublishPlan{BaseGeneration: 1, NextGeneration: 2, ManifestSHA256: fixtureHash("stale"), Reason: "manual", Desired: desired, SegmentUpdates: []SegmentUpdate{update}, ServingVectorRekeys: rekeys}); err == nil || !strings.Contains(err.Error(), "BASE_GENERATION_CHANGED") {
		t.Fatalf("stale generation err=%v", err)
	}
}

func TestLegacyServingVectorRekeyRejectsUnprovenRows(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(t *testing.T, p *ProductionStore, desired config.ResolvedConfig, inputHash string)
	}{
		{"dimension", func(t *testing.T, p *ProductionStore, desired config.ResolvedConfig, inputHash string) {
			t.Helper()
			mutateLegacySpace(t, p, desired, func(space *legacySpaceWire) { space.TargetDimensions = 512 })
		}},
		{"reducer", func(t *testing.T, p *ProductionStore, desired config.ResolvedConfig, inputHash string) {
			t.Helper()
			mutateLegacySpace(t, p, desired, func(space *legacySpaceWire) { space.ReducerID = "forged-reducer" })
		}},
		{"source", func(t *testing.T, p *ProductionStore, desired config.ResolvedConfig, inputHash string) {
			t.Helper()
			if _, err := p.Write.db.Exec(`UPDATE meta SET source_profile='forged'`); err != nil {
				t.Fatal(err)
			}
		}},
		{"codec", func(t *testing.T, p *ProductionStore, desired config.ResolvedConfig, inputHash string) {
			t.Helper()
			if _, err := p.Write.db.Exec(`UPDATE meta SET vector_storage_profile='forged'`); err != nil {
				t.Fatal(err)
			}
		}},
		{"canonical-key", func(t *testing.T, p *ProductionStore, desired config.ResolvedConfig, inputHash string) {
			t.Helper()
			if _, err := p.Write.db.Exec(`UPDATE vector_cache SET canonical_input_sha256=?`, fixtureHash("other")); err != nil {
				t.Fatal(err)
			}
		}},
		{"forged-json", func(t *testing.T, p *ProductionStore, desired config.ResolvedConfig, inputHash string) {
			t.Helper()
			if _, err := p.Write.db.Exec(`UPDATE meta SET vector_space_profile_json=x'7b7d'`); err != nil {
				t.Fatal(err)
			}
		}},
		{"invalid-blob", func(t *testing.T, p *ProductionStore, desired config.ResolvedConfig, inputHash string) {
			t.Helper()
			if _, err := p.Write.db.Exec(`UPDATE vector_cache SET blob=x'00'`); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			p, desired, inputHash, _ := legacyRekeyFixture(t)
			defer p.Close()
			test.mutate(t, p, desired, inputHash)
			snapshot, err := p.IndexSnapshot(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			plan, err := p.PlanLegacyServingVectorRekeys(context.Background(), snapshot, desired, []string{inputHash})
			if err != nil || len(plan) != 0 {
				t.Fatalf("unproven vector plan=%#v err=%v", plan, err)
			}
		})
	}
}

func TestLegacyServingVectorRekeyRollsBackWithPublishFailure(t *testing.T) {
	ctx := context.Background()
	p, desired, inputHash, rekeys := legacyRekeyFixture(t)
	defer p.Close()
	snapshot, err := p.IndexSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	segments, err := p.ReconciliationSegments(ctx, snapshot.Applied.ActiveGeneration)
	if err != nil || len(segments) != 1 {
		t.Fatal(err)
	}
	if _, err := p.Write.db.ExecContext(ctx, `CREATE TRIGGER fail_rekey_publish BEFORE UPDATE ON meta BEGIN SELECT RAISE(ABORT, 'injected publish failure'); END`); err != nil {
		t.Fatal(err)
	}
	update := SegmentUpdate{ID: segments[0].ID, CanonicalInputSHA256: inputHash, CanonicalTextProfile: string(desired.Profiles.Fingerprints.CanonicalText), ServingProfile: string(desired.Profiles.Fingerprints.VectorStorage)}
	if err := p.PublishIndexGeneration(ctx, IndexPublishPlan{BaseGeneration: 1, NextGeneration: 2, ManifestSHA256: fixtureHash("rollback"), Reason: "manual", Desired: desired, SegmentUpdates: []SegmentUpdate{update}, ServingVectorRekeys: rekeys}); err == nil {
		t.Fatal("injected publish failure accepted")
	}
	var generation, copied int
	var serving string
	if err := p.Read.db.QueryRowContext(ctx, `SELECT active_generation,active_serving_profile FROM meta`).Scan(&generation, &serving); err != nil || generation != 1 || serving == string(desired.Profiles.Fingerprints.VectorStorage) {
		t.Fatalf("rollback metadata=%d/%q err=%v", generation, serving, err)
	}
	if err := p.Read.db.QueryRowContext(ctx, `SELECT count(*) FROM vector_cache WHERE serving_profile=? AND canonical_input_sha256=?`, string(desired.Profiles.Fingerprints.VectorStorage), inputHash).Scan(&copied); err != nil || copied != 0 {
		t.Fatalf("rollback copied=%d err=%v", copied, err)
	}
}

func TestLegacyServingVectorRekeyRechecksItsSourceInsidePublish(t *testing.T) {
	ctx := context.Background()
	p, desired, inputHash, rekeys := legacyRekeyFixture(t)
	defer p.Close()
	if _, err := p.Write.db.ExecContext(ctx, `UPDATE vector_cache SET blob=x'00' WHERE serving_profile=? AND canonical_input_sha256=?`, rekeys[0].LegacyServingID, inputHash); err != nil {
		t.Fatal(err)
	}
	snapshot, err := p.IndexSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	segments, err := p.ReconciliationSegments(ctx, snapshot.Applied.ActiveGeneration)
	if err != nil || len(segments) != 1 {
		t.Fatalf("segments=%#v err=%v", segments, err)
	}
	update := SegmentUpdate{ID: segments[0].ID, CanonicalInputSHA256: inputHash, CanonicalTextProfile: string(desired.Profiles.Fingerprints.CanonicalText), ServingProfile: string(desired.Profiles.Fingerprints.VectorStorage)}
	if err := p.PublishIndexGeneration(ctx, IndexPublishPlan{BaseGeneration: 1, NextGeneration: 2, ManifestSHA256: fixtureHash("changed-source"), Reason: "manual", Desired: desired, SegmentUpdates: []SegmentUpdate{update}, ServingVectorRekeys: rekeys}); err == nil || !strings.Contains(err.Error(), "source changed") {
		t.Fatalf("mutated source err=%v", err)
	}
	var generation, copied int
	if err := p.Read.db.QueryRowContext(ctx, `SELECT active_generation FROM meta`).Scan(&generation); err != nil || generation != 1 {
		t.Fatalf("generation=%d err=%v", generation, err)
	}
	if err := p.Read.db.QueryRowContext(ctx, `SELECT count(*) FROM vector_cache WHERE serving_profile=? AND canonical_input_sha256=?`, string(desired.Profiles.Fingerprints.VectorStorage), inputHash).Scan(&copied); err != nil || copied != 0 {
		t.Fatalf("copied=%d err=%v", copied, err)
	}
}

func legacyRekeyFixture(t *testing.T) (*ProductionStore, config.ResolvedConfig, string, []ServingVectorRekey) {
	t.Helper()
	ctx := context.Background()
	desired := testResolvedConfig(t)
	p, err := OpenProduction(ctx, t.TempDir(), desired)
	if err != nil {
		t.Fatal(err)
	}
	file := indexFile("a.go", "A", "package p\nfunc A() {}\n", desired)
	if err := p.PublishIndexGeneration(ctx, IndexPublishPlan{BaseGeneration: 0, NextGeneration: 1, ManifestSHA256: fixtureHash("legacy-base"), Reason: "manual", Desired: desired, Changed: []PreparedIndexFile{file}}); err != nil {
		p.Close()
		t.Fatal(err)
	}
	var inputHash string
	if err := p.Read.db.QueryRowContext(ctx, `SELECT canonical_input_sha256 FROM embedding_segments`).Scan(&inputHash); err != nil {
		p.Close()
		t.Fatal(err)
	}
	writeLegacyMeta(t, p, desired)
	legacy := legacyMetadata(t, desired)
	stored := validBinary(t, desired.Embedding.ServingDimensions)
	if _, err := p.Write.db.ExecContext(ctx, `INSERT INTO vector_cache(serving_profile,canonical_input_sha256,dimensions,codec_id,codec_version,blob,scale,norm,materialization_fingerprint,source_profile,vector_space_profile,raw_vector_sha256,materialized_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, legacy.storageFingerprint, inputHash, stored.Dimensions, stored.CodecID, stored.CodecVersion, stored.Blob, nil, nil, legacy.storageFingerprint, legacy.sourceFingerprint, legacy.spaceFingerprint, inputHash, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		p.Close()
		t.Fatal(err)
	}
	snapshot, err := p.IndexSnapshot(ctx)
	if err != nil {
		p.Close()
		t.Fatal(err)
	}
	plan, err := p.PlanLegacyServingVectorRekeys(ctx, snapshot, desired, []string{inputHash})
	if err != nil {
		p.Close()
		t.Fatal(err)
	}
	return p, desired, inputHash, plan
}

type legacyMetadataRecord struct {
	sourceFingerprint, spaceFingerprint, storageFingerprint profile.Fingerprint
	sourceJSON, spaceJSON, storageJSON                      []byte
}

func legacyMetadata(t *testing.T, desired config.ResolvedConfig) legacyMetadataRecord {
	t.Helper()
	sourceJSON, err := config.CanonicalJSON(desired.Profiles.Source)
	if err != nil {
		t.Fatal(err)
	}
	sourceFingerprint, err := config.Fingerprint(desired.Profiles.Source, config.SourceProfileDomain)
	if err != nil {
		t.Fatal(err)
	}
	space := legacySpaceWire{SourceProfileFingerprint: sourceFingerprint, TargetDimensions: desired.Embedding.ServingDimensions, ReducerID: desired.Embedding.ReducerID, NormalizerID: desired.Embedding.NormalizerID, Metric: desired.Embedding.Metric}
	spaceJSON, err := config.CanonicalJSON(space)
	if err != nil {
		t.Fatal(err)
	}
	spaceFingerprint, err := config.Fingerprint(space, config.VectorSpaceDomain)
	if err != nil {
		t.Fatal(err)
	}
	storage := profile.VectorStorageProfile{VectorSpaceProfileFingerprint: spaceFingerprint, StorageCodecID: desired.Profiles.VectorStorage.StorageCodecID}
	storageJSON, err := config.CanonicalJSON(storage)
	if err != nil {
		t.Fatal(err)
	}
	storageFingerprint, err := config.Fingerprint(storage, config.VectorStorageDomain)
	if err != nil {
		t.Fatal(err)
	}
	return legacyMetadataRecord{sourceFingerprint: sourceFingerprint, spaceFingerprint: spaceFingerprint, storageFingerprint: storageFingerprint, sourceJSON: sourceJSON, spaceJSON: spaceJSON, storageJSON: storageJSON}
}

func writeLegacyMeta(t *testing.T, p *ProductionStore, desired config.ResolvedConfig) {
	t.Helper()
	legacy := legacyMetadata(t, desired)
	canonicalJSON, err := config.CanonicalJSON(desired.Profiles.CanonicalText)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Write.db.Exec(`UPDATE meta SET index_profile='legacy-index',canonical_text_profile=?,canonical_text_profile_json=?,source_profile=?,source_profile_json=?,vector_space_profile=?,vector_space_profile_json=?,vector_storage_profile=?,vector_storage_profile_json=?,active_serving_profile=?`, desired.Profiles.Fingerprints.CanonicalText, canonicalJSON, legacy.sourceFingerprint, legacy.sourceJSON, legacy.spaceFingerprint, legacy.spaceJSON, legacy.storageFingerprint, legacy.storageJSON, legacy.storageFingerprint); err != nil {
		t.Fatal(err)
	}
}

func mutateLegacySpace(t *testing.T, p *ProductionStore, desired config.ResolvedConfig, mutate func(*legacySpaceWire)) {
	t.Helper()
	legacy := legacyMetadata(t, desired)
	space := legacySpaceWire{SourceProfileFingerprint: legacy.sourceFingerprint, TargetDimensions: desired.Embedding.ServingDimensions, ReducerID: desired.Embedding.ReducerID, NormalizerID: desired.Embedding.NormalizerID, Metric: desired.Embedding.Metric}
	mutate(&space)
	json, err := config.CanonicalJSON(space)
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := config.Fingerprint(space, config.VectorSpaceDomain)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Write.db.Exec(`UPDATE meta SET vector_space_profile=?,vector_space_profile_json=?`, fingerprint, json); err != nil {
		t.Fatal(err)
	}
}
