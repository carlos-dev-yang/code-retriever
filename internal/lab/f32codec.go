// Package lab is development-only storage support. Production packages must
// not import this package or open its database.
package lab

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"math"

	"cidx/internal/vector"
)

const F32CodecID = "cidx-lab-f32-le-v1"

// F32Vector exists only for source-document raw preservation and never enters
// production store APIs.
type F32Vector struct {
	Dimensions int
	Values     []float32
	Checksum   uint32
}

func NewF32Vector(values []float32, dimensions int) (F32Vector, error) {
	if err := vector.ValidateF32(values, dimensions); err != nil {
		return F32Vector{}, err
	}
	encoded := EncodeF32(values)
	return F32Vector{Dimensions: dimensions, Values: append([]float32(nil), values...), Checksum: crc32.ChecksumIEEE(encoded)}, nil
}

// EncodeF32 is IEEE-754 binary32 little-endian with no header.
func EncodeF32(values []float32) []byte {
	blob := make([]byte, len(values)*4)
	for index, value := range values {
		binary.LittleEndian.PutUint32(blob[index*4:], math.Float32bits(value))
	}
	return blob
}

// VectorSHA256 identifies the exact persisted f32-le-v1 bytes. It is
// provenance only; it never selects a serving representation.
func VectorSHA256(blob []byte) string {
	sum := sha256.Sum256(blob)
	return hex.EncodeToString(sum[:])
}

func DecodeF32(blob []byte, dimensions int, checksum uint32) (F32Vector, error) {
	if dimensions <= 0 || len(blob) != dimensions*4 {
		return F32Vector{}, fmt.Errorf("invalid f32 blob length")
	}
	if crc32.ChecksumIEEE(blob) != checksum {
		return F32Vector{}, fmt.Errorf("f32 checksum mismatch")
	}
	values := make([]float32, dimensions)
	for index := range values {
		values[index] = math.Float32frombits(binary.LittleEndian.Uint32(blob[index*4:]))
	}
	return NewF32Vector(values, dimensions)
}
