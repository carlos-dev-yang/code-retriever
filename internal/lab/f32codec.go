// Package lab owns vector-free development and evaluation metadata. The f32
// aliases below are a compatibility surface for development callers while the
// bytes themselves live exclusively in internal/sourcebank.
package lab

import "cidx/internal/sourcebank"

const F32CodecID = sourcebank.EncodingID

type F32Vector = sourcebank.F32Vector

func NewF32Vector(values []float32, dimensions int) (F32Vector, error) {
	return sourcebank.NewF32Vector(values, dimensions)
}

func EncodeF32(values []float32) []byte { return sourcebank.EncodeF32(values) }

func VectorSHA256(blob []byte) string { return sourcebank.VectorSHA256(blob) }

func DecodeF32(blob []byte, dimensions int, checksum uint32) (F32Vector, error) {
	return sourcebank.DecodeF32(blob, dimensions, checksum)
}
