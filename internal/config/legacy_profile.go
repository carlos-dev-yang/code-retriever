package config

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"

	"cidx/internal/profile"
)

// LegacyServingProfileMetadata is the persisted pre-R4 profile record used
// only to decide whether an old local serving vector can be copied to the R4
// serving key. It is not a configuration input and never accepts aliases.
type LegacyServingProfileMetadata struct {
	CanonicalTextFingerprint profile.Fingerprint
	CanonicalTextJSON        []byte
	SourceFingerprint        profile.Fingerprint
	SourceJSON               []byte
	VectorSpaceFingerprint   profile.Fingerprint
	VectorSpaceJSON          []byte
	VectorStorageFingerprint profile.Fingerprint
	VectorStorageJSON        []byte
	ActiveServingProfile     profile.Fingerprint
}

type legacyVectorSpaceProfile struct {
	SourceProfileFingerprint profile.Fingerprint `json:"source_profile_fingerprint"`
	TargetDimensions         int                 `json:"target_dimensions"`
	ReducerID                string              `json:"reducer_id"`
	NormalizerID             string              `json:"normalizer_id"`
	Metric                   string              `json:"metric"`
}

// LegacyServingProfileEquivalent fails closed unless persisted canonical
// pre-R4 JSON reproduces every recorded fingerprint and its vector semantics
// exactly match the requested R4 serving representation.
func LegacyServingProfileEquivalent(desired ResolvedConfig, stored LegacyServingProfileMetadata) bool {
	if desired.ValidateIntegrity() != nil || stored.ActiveServingProfile == "" || stored.ActiveServingProfile != stored.VectorStorageFingerprint {
		return false
	}
	var canonical profile.CanonicalTextProfile
	if !decodeCanonicalLegacy(stored.CanonicalTextJSON, &canonical) || !reflect.DeepEqual(canonical, desired.Profiles.CanonicalText) {
		return false
	}
	if fingerprintMatches(canonical, CanonicalTextDomain, stored.CanonicalTextFingerprint) == false || stored.CanonicalTextFingerprint != desired.Profiles.Fingerprints.CanonicalText {
		return false
	}
	var source profile.EmbeddingSourceProfile
	if !decodeCanonicalLegacy(stored.SourceJSON, &source) || !reflect.DeepEqual(source, desired.Profiles.Source) {
		return false
	}
	if !fingerprintMatches(source, SourceProfileDomain, stored.SourceFingerprint) || stored.SourceFingerprint != desired.Profiles.Fingerprints.Source {
		return false
	}
	var space legacyVectorSpaceProfile
	if !decodeCanonicalLegacy(stored.VectorSpaceJSON, &space) || space.SourceProfileFingerprint != stored.SourceFingerprint || space.TargetDimensions != desired.Embedding.ServingDimensions || space.ReducerID != desired.Embedding.ReducerID || space.NormalizerID != desired.Embedding.NormalizerID || space.Metric != desired.Embedding.Metric {
		return false
	}
	if !fingerprintMatches(space, VectorSpaceDomain, stored.VectorSpaceFingerprint) {
		return false
	}
	var storage profile.VectorStorageProfile
	if !decodeCanonicalLegacy(stored.VectorStorageJSON, &storage) || storage.VectorSpaceProfileFingerprint != stored.VectorSpaceFingerprint || storage.StorageCodecID != desired.Profiles.VectorStorage.StorageCodecID {
		return false
	}
	return fingerprintMatches(storage, VectorStorageDomain, stored.VectorStorageFingerprint)
}

func decodeCanonicalLegacy(data []byte, target any) bool {
	if len(data) == 0 {
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return false
	}
	canonical, err := CanonicalJSON(target)
	return err == nil && bytes.Equal(canonical, data)
}

func fingerprintMatches(value any, domain string, expected profile.Fingerprint) bool {
	actual, err := Fingerprint(value, domain)
	return err == nil && expected != "" && actual == expected
}
