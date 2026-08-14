package vector

import "fmt"

const (
	BinaryCodecID = "cidx-binary-sign-lsb-v1"
	Int8CodecID   = "cidx-int8-symmetric-v1"
)

// StoredVector is the only vector payload accepted by the future production
// store. F32Vector is intentionally defined in internal/lab and cannot be
// substituted for this type.
type StoredVector struct {
	CodecID      string
	CodecVersion uint16
	Dimensions   int
	Blob         []byte
	Scale        float32 // int8 only; zero for binary
	Norm         float32 // int8 reconstructed-vector norm; zero for binary
}

// Codec is the complete cidx-owned storage contract for one selected serving
// profile. It intentionally accepts only normalized target-space f32 values.
type Codec interface {
	ID() string
	Encode([]float32) (StoredVector, error)
}

type binaryCodec struct{}

func (binaryCodec) ID() string                                    { return BinaryCodecID }
func (binaryCodec) Encode(values []float32) (StoredVector, error) { return EncodeBinary(values) }

type int8Codec struct{}

func (int8Codec) ID() string                                    { return Int8CodecID }
func (int8Codec) Encode(values []float32) (StoredVector, error) { return EncodeInt8(values) }

func CodecForID(id string) (Codec, error) {
	switch id {
	case BinaryCodecID:
		return binaryCodec{}, nil
	case Int8CodecID:
		return int8Codec{}, nil
	default:
		return nil, fmt.Errorf("unknown cidx storage codec %q", id)
	}
}

func (v StoredVector) Clone() StoredVector {
	v.Blob = append([]byte(nil), v.Blob...)
	return v
}

func (v StoredVector) Validate() error {
	switch v.CodecID {
	case BinaryCodecID:
		return ValidateBinary(v)
	case Int8CodecID:
		return ValidateInt8(v)
	default:
		return fmt.Errorf("unknown cidx storage codec %q", v.CodecID)
	}
}
