package vector

import (
	"fmt"
	"math"
)

const int8CodecVersion uint16 = 1

// EncodeInt8 uses symmetric per-vector quantization. Persisted Scale is the
// float32 encoding of max(abs(x))/127. Rounding is nearest integer away from
// zero at half, values clamp to [-127, 127], and Norm is recomputed from that
// exact persisted scale before it is stored.
func EncodeInt8(values []float32) (StoredVector, error) {
	if err := ValidateF32(values, len(values)); err != nil || len(values) == 0 {
		if err != nil {
			return StoredVector{}, err
		}
		return StoredVector{}, ErrInvalidDimensions
	}
	maxAbs := float64(0)
	for _, value := range values {
		maxAbs = math.Max(maxAbs, math.Abs(float64(value)))
	}
	if maxAbs == 0 {
		return StoredVector{}, ErrZeroVector
	}
	scale := float32(maxAbs / 127)
	if !finitePositive(scale) {
		return StoredVector{}, ErrZeroVector
	}
	blob := make([]byte, len(values))
	for index, value := range values {
		quantized := math.Round(float64(value) / float64(scale))
		quantized = math.Max(-127, math.Min(127, quantized))
		blob[index] = byte(int8(quantized))
	}
	norm := reconstructedInt8Norm(blob, scale)
	if !finitePositive(norm) {
		return StoredVector{}, ErrZeroVector
	}
	return StoredVector{
		CodecID: Int8CodecID, CodecVersion: int8CodecVersion, Dimensions: len(values),
		Blob: blob, Scale: scale, Norm: norm,
	}, nil
}

func ValidateInt8(stored StoredVector) error {
	if stored.CodecVersion != int8CodecVersion || stored.Dimensions <= 0 {
		return fmt.Errorf("invalid int8 codec metadata")
	}
	if len(stored.Blob) != stored.Dimensions {
		return fmt.Errorf("invalid int8 blob length: got %d, want %d", len(stored.Blob), stored.Dimensions)
	}
	if !finitePositive(stored.Scale) || !finitePositive(stored.Norm) {
		return fmt.Errorf("invalid int8 scale or norm metadata")
	}
	for _, encoded := range stored.Blob {
		if int8(encoded) == -128 {
			return fmt.Errorf("int8 blob contains value outside [-127,127]")
		}
	}
	if stored.Norm != reconstructedInt8Norm(stored.Blob, stored.Scale) {
		return fmt.Errorf("int8 norm does not match persisted scale and blob")
	}
	return nil
}

func reconstructedInt8Norm(blob []byte, scale float32) float32 {
	var sum float64
	for _, encoded := range blob {
		reconstructed := float64(int8(encoded)) * float64(scale)
		sum += reconstructed * reconstructed
	}
	return float32(math.Sqrt(sum))
}

func finitePositive(value float32) bool {
	return value > 0 && !math.IsInf(float64(value), 0) && !math.IsNaN(float64(value))
}

// ScoreInt8 calculates cosine of the reconstructed query and document
// approximations. The result is a codec score, not exact target-f32 cosine.
func ScoreInt8(query []float32, stored StoredVector) (float64, error) {
	if err := ValidateInt8(stored); err != nil {
		return 0, err
	}
	if len(query) != stored.Dimensions {
		return 0, ErrInvalidDimensions
	}
	prepared, err := EncodeInt8(query)
	if err != nil {
		return 0, err
	}
	var dot float64
	for index, value := range prepared.Blob {
		dot += float64(int8(value)) * float64(int8(stored.Blob[index]))
	}
	dot *= float64(prepared.Scale) * float64(stored.Scale)
	score := dot / (float64(prepared.Norm) * float64(stored.Norm))
	return math.Max(-1, math.Min(1, score)), nil
}
