package vector

import (
	"math"
	"testing"
)

func TestTransformAndInt8RejectInvalidAndPreserveContracts(t *testing.T) {
	source := make([]float32, 1024)
	for index := range source {
		source[index] = float32(index + 1)
	}
	target, err := ReduceAndNormalize(TransformSpec{SourceDimensions: 1024, TargetDimensions: 512, ReducerID: ReducerID, NormalizerID: NormalizerID, MetricID: MetricID}, source)
	if err != nil {
		t.Fatal(err)
	}
	if len(target) != 512 {
		t.Fatalf("target dimensions = %d", len(target))
	}
	if cosine, err := Cosine(target, target); err != nil || math.Abs(cosine-1) > 1e-5 {
		t.Fatalf("normalized target cosine = %v, %v", cosine, err)
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

func TestTransformerAndInt8AreDeterministic(t *testing.T) {
	source := make([]float32, 1024)
	for i := range source {
		source[i] = float32((i % 17) - 8)
	}
	transformer := Transformer{Spec: TransformSpec{SourceDimensions: 1024, TargetDimensions: 1024, ReducerID: ReducerID, NormalizerID: NormalizerID, MetricID: MetricID}}
	first, err := transformer.Transform(source)
	if err != nil {
		t.Fatal(err)
	}
	second, err := transformer.Transform(source)
	if err != nil || !equalF32(first, second) {
		t.Fatalf("non-deterministic transform: %v", err)
	}
	a, err := EncodeInt8(first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := EncodeInt8(second)
	if err != nil || string(a.Blob) != string(b.Blob) || a.Scale != b.Scale || a.Norm != b.Norm {
		t.Fatalf("int8 encoding not deterministic: %v", err)
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
