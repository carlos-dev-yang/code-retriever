package vector

import (
	"fmt"
	"math/bits"
)

const binaryCodecVersion uint16 = 1

// EncodeBinary maps non-negative components to bit 1 and negative components
// to bit 0. Components are packed least-significant-bit first in ascending
// component order. The unused high bits in the final byte are always zero.
func EncodeBinary(values []float32) (StoredVector, error) {
	if err := ValidateF32(values, len(values)); err != nil || len(values) == 0 {
		if err != nil {
			return StoredVector{}, err
		}
		return StoredVector{}, ErrInvalidDimensions
	}
	if _, err := Cosine(values, values); err != nil {
		return StoredVector{}, err
	}
	blob := make([]byte, (len(values)+7)/8)
	for index, value := range values {
		if value >= 0 {
			blob[index/8] |= 1 << uint(index%8)
		}
	}
	return StoredVector{CodecID: BinaryCodecID, CodecVersion: binaryCodecVersion, Dimensions: len(values), Blob: blob}, nil
}

func ValidateBinary(stored StoredVector) error {
	if stored.CodecVersion != binaryCodecVersion || stored.Dimensions <= 0 {
		return fmt.Errorf("invalid binary codec metadata")
	}
	if stored.Scale != 0 || stored.Norm != 0 {
		return fmt.Errorf("binary codec does not accept scale or norm metadata")
	}
	expectedLength := (stored.Dimensions + 7) / 8
	if len(stored.Blob) != expectedLength {
		return fmt.Errorf("invalid binary blob length: got %d, want %d", len(stored.Blob), expectedLength)
	}
	if remainder := stored.Dimensions % 8; remainder != 0 {
		unusedMask := byte(0xff << uint(remainder))
		if stored.Blob[len(stored.Blob)-1]&unusedMask != 0 {
			return fmt.Errorf("binary blob has non-zero padding bits")
		}
	}
	return nil
}

func binaryQuery(values []float32) ([]byte, error) {
	query, err := EncodeBinary(values)
	if err != nil {
		return nil, err
	}
	return query.Blob, nil
}

// ScoreBinary returns a normalized sign-agreement score in [-1, 1]. It is a
// codec approximation of target-space cosine, never an exact cosine value.
func ScoreBinary(query []float32, stored StoredVector) (float64, error) {
	if err := ValidateBinary(stored); err != nil {
		return 0, err
	}
	if len(query) != stored.Dimensions {
		return 0, ErrInvalidDimensions
	}
	queryBlob, err := binaryQuery(query)
	if err != nil {
		return 0, err
	}
	matches := 0
	for index := range queryBlob {
		matches += 8 - bits.OnesCount8(queryBlob[index]^stored.Blob[index])
	}
	if remainder := stored.Dimensions % 8; remainder != 0 {
		matches -= 8 - remainder // padding bits agree by construction but are excluded
	}
	return (2*float64(matches))/float64(stored.Dimensions) - 1, nil
}
