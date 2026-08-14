// Package vector contains the cidx-owned vector-space and storage-codec
// contracts. It deliberately receives typed values; it does not read config.
package vector

import (
	"errors"
	"fmt"
	"math"
)

const (
	SourceDimensions = 1024
	ReducerID        = "prefix-l2-v1"
	NormalizerID     = "l2-v1"
	MetricID         = "cosine"
)

var AllowedTargetDimensions = map[int]struct{}{256: {}, 512: {}, 1024: {}}

var (
	ErrInvalidDimensions = errors.New("invalid vector dimensions")
	ErrNonFiniteVector   = errors.New("vector contains non-finite value")
	ErrZeroVector        = errors.New("zero vector is not valid")
)

// TransformSpec is the future typed profile input for a vector transformation.
// It is intentionally independent of configuration parsing.
type TransformSpec struct {
	SourceDimensions int
	TargetDimensions int
	ReducerID        string
	NormalizerID     string
	MetricID         string
}

func DefaultTransformSpec(targetDimensions int) TransformSpec {
	return TransformSpec{
		SourceDimensions: SourceDimensions,
		TargetDimensions: targetDimensions,
		ReducerID:        ReducerID,
		NormalizerID:     NormalizerID,
		MetricID:         MetricID,
	}
}

func (s TransformSpec) Validate() error {
	if s.SourceDimensions != SourceDimensions {
		return fmt.Errorf("%w: source dimensions must be %d", ErrInvalidDimensions, SourceDimensions)
	}
	if _, ok := AllowedTargetDimensions[s.TargetDimensions]; !ok {
		return fmt.Errorf("%w: target dimensions %d", ErrInvalidDimensions, s.TargetDimensions)
	}
	if s.ReducerID != ReducerID || s.NormalizerID != NormalizerID || s.MetricID != MetricID {
		return fmt.Errorf("unsupported transform contract %q/%q/%q", s.ReducerID, s.NormalizerID, s.MetricID)
	}
	return nil
}

// ValidateF32 verifies a source or target float vector before any transform or
// codec accepts it.
func ValidateF32(values []float32, dimensions int) error {
	if len(values) != dimensions {
		return fmt.Errorf("%w: got %d, want %d", ErrInvalidDimensions, len(values), dimensions)
	}
	for _, value := range values {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return ErrNonFiniteVector
		}
	}
	return nil
}

// ReduceAndNormalize applies the v1 reference path: prefix selection followed
// by L2 normalization. Its returned storage never aliases the source slice.
func ReduceAndNormalize(spec TransformSpec, source []float32) ([]float32, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	if err := ValidateF32(source, spec.SourceDimensions); err != nil {
		return nil, err
	}
	result := append([]float32(nil), source[:spec.TargetDimensions]...)
	var sum float64
	for _, value := range result {
		sum += float64(value) * float64(value)
	}
	if sum == 0 {
		return nil, ErrZeroVector
	}
	norm := math.Sqrt(sum)
	for index := range result {
		result[index] = float32(float64(result[index]) / norm)
	}
	return result, nil
}

func Cosine(left, right []float32) (float64, error) {
	if len(left) != len(right) || len(left) == 0 {
		return 0, ErrInvalidDimensions
	}
	if err := ValidateF32(left, len(left)); err != nil {
		return 0, err
	}
	if err := ValidateF32(right, len(right)); err != nil {
		return 0, err
	}
	var dot, leftNorm, rightNorm float64
	for index, value := range left {
		dot += float64(value) * float64(right[index])
		leftNorm += float64(value) * float64(value)
		rightNorm += float64(right[index]) * float64(right[index])
	}
	if leftNorm == 0 || rightNorm == 0 {
		return 0, ErrZeroVector
	}
	return dot / math.Sqrt(leftNorm*rightNorm), nil
}
