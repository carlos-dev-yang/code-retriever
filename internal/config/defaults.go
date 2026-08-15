package config

import "fmt"

// ErrDefaultRawPending marks the single deliberately unresolved init boundary.
// The canonical design requires measured operational defaults before cidx may
// create a config file, so callers must not scatter provisional numbers.
var ErrDefaultRawPending = fmt.Errorf("DEFAULT_CONFIG_VALUES_PENDING_DECISION")

func DefaultRaw(targetDimensions int, codec string) (RawConfig, error) {
	spec := VoyageCode4()
	if !spec.SupportsTarget(targetDimensions) {
		return RawConfig{}, fmt.Errorf("unsupported target dimension")
	}
	if codec != StorageCodecBinary && codec != StorageCodecInt8 {
		return RawConfig{}, fmt.Errorf("unsupported storage codec")
	}
	return RawConfig{}, ErrDefaultRawPending
}
