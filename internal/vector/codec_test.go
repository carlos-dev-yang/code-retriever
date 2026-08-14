package vector

import (
	"math"
	"testing"
)

func TestTransformAndCodecsRejectInvalidAndPreserveContracts(t *testing.T) {
	source := make([]float32, SourceDimensions)
	for index := range source {
		source[index] = float32(index + 1)
	}
	target, err := ReduceAndNormalize(DefaultTransformSpec(256), source)
	if err != nil {
		t.Fatal(err)
	}
	if len(target) != 256 {
		t.Fatalf("target dimensions = %d", len(target))
	}
	if cosine, err := Cosine(target, target); err != nil || math.Abs(cosine-1) > 1e-5 {
		t.Fatalf("normalized target cosine = %v, %v", cosine, err)
	}
	binaryVector, err := EncodeBinary([]float32{1, -1, 1, -1, 1, -1, 1, -1, 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := binaryVector.Validate(); err != nil {
		t.Fatal(err)
	}
	if binaryVector.Blob[1] != 1 {
		t.Fatalf("binary LSB packing = %08b, want 00000001", binaryVector.Blob[1])
	}
	invalidPadding := binaryVector.Clone()
	invalidPadding.Blob[1] |= 0x80
	if err := invalidPadding.Validate(); err == nil {
		t.Fatal("expected binary padding rejection")
	}
	if _, err := EncodeBinary(make([]float32, 9)); err != ErrZeroVector {
		t.Fatalf("zero binary error = %v", err)
	}
	int8Vector, err := EncodeInt8(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := int8Vector.Validate(); err != nil {
		t.Fatal(err)
	}
	if score, err := ScoreInt8(target, int8Vector); err != nil || score < 0.99 || score > 1 {
		t.Fatalf("int8 self score = %v, %v", score, err)
	}
	badMetadata := int8Vector.Clone()
	badMetadata.Scale = 0
	if err := badMetadata.Validate(); err == nil {
		t.Fatal("expected int8 metadata rejection")
	}
	badNorm := int8Vector.Clone()
	badNorm.Norm *= 0.5
	if err := badNorm.Validate(); err == nil {
		t.Fatal("expected int8 norm mismatch rejection")
	}
}
