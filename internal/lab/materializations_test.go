package lab

import (
	"context"
	"testing"

	"cidx/internal/vector"
)

func TestMaterializationRunTransitionsReadyToPublished(t *testing.T) {
	ctx := context.Background()
	store, err := OpenStore(ctx, Options{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	digest := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	id, err := store.StartMaterialization(ctx, MaterializationRun{BuildID: "build", Generation: 1, ManifestSHA256: digest, SourceProfile: digest, VectorSpaceProfile: digest, StorageProfile: digest, Planned: 1})
	if err != nil {
		t.Fatal(err)
	}
	stored, err := vector.EncodeBinary([]float32{1, -1, 1, -1, 1, -1, 1, -1})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutMaterializedVariants(ctx, id, []MaterializedVariant{{InputHash: digest, RawVectorSHA256: digest, Stored: stored}}); err != nil {
		t.Fatal(err)
	}
	if err := store.FinishMaterialization(ctx, id, "ready", 1, 0, 0, ""); err != nil {
		t.Fatal(err)
	}
	if err := store.PutMaterializedVariants(ctx, id, []MaterializedVariant{{InputHash: digest, RawVectorSHA256: digest, Stored: stored}}); err == nil {
		t.Fatal("ready materialization accepted a candidate mutation")
	}
	if err := store.FinishMaterialization(ctx, id, "published", 1, 0, 0, ""); err != nil {
		t.Fatal(err)
	}
	var status, checksum string
	var coverage float64
	if err := store.db.QueryRowContext(ctx, `SELECT status,raw_coverage,output_checksum FROM materialization_runs WHERE id=?`, id).Scan(&status, &coverage, &checksum); err != nil || status != "published" || coverage != 1 || !validDigest(checksum) {
		t.Fatalf("status=%q coverage=%v checksum=%q err=%v", status, coverage, checksum, err)
	}
}
