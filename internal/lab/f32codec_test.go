package lab

import (
	"context"
	"math"
	"testing"

	"cidx/internal/vector"
)

func TestF32LittleEndianRoundTripAndIntegrityRejection(t *testing.T) {
	original, err := NewF32Vector([]float32{1.25, -2.5}, 2)
	if err != nil {
		t.Fatal(err)
	}
	blob := EncodeF32(original.Values)
	decoded, err := DecodeF32(blob, original.Dimensions, original.Checksum)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Values[0] != original.Values[0] || decoded.Values[1] != original.Values[1] {
		t.Fatalf("round trip mismatch: %#v", decoded.Values)
	}
	if _, err := DecodeF32(blob[:len(blob)-1], original.Dimensions, original.Checksum); err == nil {
		t.Fatal("expected length rejection")
	}
	if _, err := NewF32Vector([]float32{float32(math.NaN())}, 1); err == nil {
		t.Fatal("expected NaN rejection")
	}
}

func TestLabStoreIsolatedDocumentOnlyF32Artifact(t *testing.T) {
	ctx := context.Background()
	store, err := OpenSpikeStore(ctx, t.TempDir()+"/embeddings.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	values := make([]float32, vector.SourceDimensions)
	values[0], values[1] = 1, 2
	vector, err := NewF32Vector(values, vector.SourceDimensions)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.PutDocumentSource(ctx, DocumentRaw{SourceProfile: "source-profile", InputHash: "input-hash", Vector: vector}); err != nil {
		t.Fatal(err)
	}
	stored, err := store.GetDocumentSource(ctx, "source-profile", "input-hash")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Checksum != vector.Checksum || stored.Values[1] != 2 {
		t.Fatalf("unexpected stored lab vector: %#v", stored)
	}
	if _, err := OpenSpikeStore(ctx, ".cidx/index.db"); err == nil {
		t.Fatal("lab factory accepted production path")
	}
}
