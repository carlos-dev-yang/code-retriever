package vector

import (
	"math"
	"testing"
)

func TestTransformAndCodecsRejectInvalidAndPreserveContracts(t *testing.T) {
	source := make([]float32, 1024)
	for index := range source {
		source[index] = float32(index + 1)
	}
	target, err := ReduceAndNormalize(TransformSpec{SourceDimensions: 1024, TargetDimensions: 256, ReducerID: ReducerID, NormalizerID: NormalizerID, MetricID: MetricID}, source)
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

func TestTransformerAndBothCodecsAreDeterministic(t *testing.T) {
	source := make([]float32, 1024)
	for i := range source {
		source[i] = float32((i % 17) - 8)
	}
	transformer := Transformer{Spec: TransformSpec{SourceDimensions: 1024, TargetDimensions: 256, ReducerID: ReducerID, NormalizerID: NormalizerID, MetricID: MetricID}}
	first, err := transformer.Transform(source)
	if err != nil {
		t.Fatal(err)
	}
	second, err := transformer.Transform(source)
	if err != nil || !equalF32(first, second) {
		t.Fatalf("non-deterministic transform: %v", err)
	}
	for _, id := range []string{BinaryCodecID, Int8CodecID} {
		codec, err := CodecForID(id)
		if err != nil {
			t.Fatal(err)
		}
		a, err := codec.Encode(first)
		if err != nil {
			t.Fatal(err)
		}
		b, err := codec.Encode(second)
		if err != nil || string(a.Blob) != string(b.Blob) || a.Scale != b.Scale || a.Norm != b.Norm {
			t.Fatalf("%s encoding not deterministic: %v", id, err)
		}
	}
}

func equalF32(left, right []float32) bool {
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
