package vector

import "fmt"

const (
	BinaryCodecID      = "cidx-binary-sign-lsb-v1"
	Int8CodecID        = "cidx-int8-symmetric-v1"
	StorageCodecBinary = "binary"
	StorageCodecInt8   = "int8"
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
