package vector

import "fmt"

const Int8CodecID = "cidx-int8-symmetric-v1"

// StoredVector is the only vector payload accepted by the future production
// store. F32Vector is intentionally defined in internal/lab and cannot be
// substituted for this type.
type StoredVector struct {
	CodecID      string
	CodecVersion uint16
	Dimensions   int
	Blob         []byte
	Scale        float32
	Norm         float32
}

func (v StoredVector) Clone() StoredVector {
	v.Blob = append([]byte(nil), v.Blob...)
	return v
}

func (v StoredVector) Validate() error {
	if v.CodecID != Int8CodecID {
		return fmt.Errorf("unknown cidx storage codec %q", v.CodecID)
	}
	return ValidateInt8(v)
}
