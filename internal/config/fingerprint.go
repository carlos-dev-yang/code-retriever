package config

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"cidx/internal/chunk"
	"cidx/internal/profile"
	"cidx/internal/vector"
)

func CanonicalJSON(value any) ([]byte, error) {
	if err := rejectInvalidStrings(value); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := writeJCS(&output, decoded); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

// rejectInvalidStrings keeps JCS's Unicode precondition explicit. Config
// decoding separately rejects lone surrogate escapes before Go can replace
// them with U+FFFD; this check covers direct profile callers as well.
func rejectInvalidStrings(value any) error {
	return rejectInvalidStringsValue(reflect.ValueOf(value), map[visit]bool{})
}

type visit struct {
	typ reflect.Type
	ptr unsafePointer
}
type unsafePointer uintptr

func rejectInvalidStringsValue(value reflect.Value, seen map[visit]bool) error {
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Interface {
		if value.IsNil() {
			return nil
		}
		return rejectInvalidStringsValue(value.Elem(), seen)
	}
	switch value.Kind() {
	case reflect.String:
		if !utf8.ValidString(value.String()) {
			return fmt.Errorf("canonical JSON string is not valid UTF-8")
		}
	case reflect.Pointer:
		if value.IsNil() {
			return nil
		}
		key := visit{typ: value.Type(), ptr: unsafePointer(value.Pointer())}
		if seen[key] {
			return nil
		}
		seen[key] = true
		return rejectInvalidStringsValue(value.Elem(), seen)
	case reflect.Struct:
		for index := 0; index < value.NumField(); index++ {
			if value.Type().Field(index).PkgPath == "" {
				if err := rejectInvalidStringsValue(value.Field(index), seen); err != nil {
					return err
				}
			}
		}
	case reflect.Array, reflect.Slice:
		for index := 0; index < value.Len(); index++ {
			if err := rejectInvalidStringsValue(value.Index(index), seen); err != nil {
				return err
			}
		}
	case reflect.Map:
		iter := value.MapRange()
		for iter.Next() {
			if iter.Key().Kind() == reflect.String && !utf8.ValidString(iter.Key().String()) {
				return fmt.Errorf("canonical JSON key is not valid UTF-8")
			}
			if err := rejectInvalidStringsValue(iter.Value(), seen); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeJCS(output *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case nil:
		output.WriteString("null")
	case bool:
		output.WriteString(strconv.FormatBool(typed))
	case string:
		writeJSONString(output, typed)
	case json.Number:
		formatted, err := canonicalNumber(string(typed))
		if err != nil {
			return err
		}
		output.WriteString(formatted)
	case []any:
		output.WriteByte('[')
		for index, item := range typed {
			if index > 0 {
				output.WriteByte(',')
			}
			if err := writeJCS(output, item); err != nil {
				return err
			}
		}
		output.WriteByte(']')
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool { return utf16Less(keys[i], keys[j]) })
		output.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				output.WriteByte(',')
			}
			writeJSONString(output, key)
			output.WriteByte(':')
			if err := writeJCS(output, typed[key]); err != nil {
				return err
			}
		}
		output.WriteByte('}')
	default:
		return fmt.Errorf("unsupported canonical JSON value %T", value)
	}
	return nil
}

// canonicalNumber follows RFC 8785's ECMAScript-number rendering for every
// finite IEEE-754 number: shortest round-trippable digits, decimal notation
// for [1e-6,1e21), and normalized scientific exponents otherwise.
func canonicalNumber(raw string) (string, error) {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return "", fmt.Errorf("invalid canonical JSON number %q", raw)
	}
	if value == 0 {
		return "0", nil
	}
	abs := math.Abs(value)
	if abs >= 1e-6 && abs < 1e21 {
		return strconv.FormatFloat(value, 'f', -1, 64), nil
	}
	formatted := strconv.FormatFloat(value, 'e', -1, 64)
	exponent := strings.IndexByte(formatted, 'e')
	if exponent < 0 {
		return formatted, nil
	}
	sign := formatted[exponent+1]
	digits := strings.TrimLeft(formatted[exponent+2:], "0")
	if digits == "" {
		digits = "0"
	}
	return formatted[:exponent] + "e" + string(sign) + digits, nil
}

func utf16Less(left, right string) bool {
	a, b := utf16.Encode([]rune(left)), utf16.Encode([]rune(right))
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}
func writeJSONString(output *bytes.Buffer, value string) {
	output.WriteByte('"')
	for _, r := range value {
		switch r {
		case '"':
			output.WriteString(`\"`)
		case '\\':
			output.WriteString(`\\`)
		case '\b':
			output.WriteString(`\b`)
		case '\f':
			output.WriteString(`\f`)
		case '\n':
			output.WriteString(`\n`)
		case '\r':
			output.WriteString(`\r`)
		case '\t':
			output.WriteString(`\t`)
		default:
			if r < 0x20 {
				output.WriteString(fmt.Sprintf(`\u%04x`, r))
			} else {
				output.WriteRune(r)
			}
		}
	}
	output.WriteByte('"')
}

func DomainSHA256(domain string, payload []byte) string {
	hash := sha256.New()
	hash.Write([]byte(domain))
	hash.Write([]byte{0})
	hash.Write(payload)
	return hex.EncodeToString(hash.Sum(nil))
}
func Fingerprint(value any, domain string) (profile.Fingerprint, error) {
	payload, err := CanonicalJSON(value)
	if err != nil {
		return "", err
	}
	return profile.Fingerprint(DomainSHA256(domain, payload)), nil
}
func CanonicalInputSHA256(canonicalInput []byte) string {
	return DomainSHA256(CanonicalInputDomain, canonicalInput)
}

type CanonicalInputHash string

type PaidSourceKey struct {
	SourceProfileFingerprint profile.Fingerprint
	CanonicalInputSHA256     CanonicalInputHash
}

type ServingVectorKey struct {
	ServingProfileFingerprint profile.Fingerprint
	CanonicalInputSHA256      CanonicalInputHash
}

func FingerprintProfiles(resolved ResolvedConfig) (DesiredProfiles, error) {
	index := profile.IndexProfile{Languages: append([]chunk.Language(nil), resolved.Index.Languages...), ChunkerVersion: IndexChunkerVersion, ProjectionVersion: IndexProjectionVersion, SegmentVersion: IndexSegmentVersion, SymbolNormalizerID: SymbolNormalizerID, FTSSchemaVersion: FTSSchemaVersion, FTSTokenizerID: FTSTokenizerID, MaxSourceFileBytes: resolved.Index.MaxSourceFileBytes, TargetSegmentBytes: resolved.Index.TargetSegmentBytes}
	canonical := profile.CanonicalTextProfile{FormatterID: CanonicalTextFormatterID, FormatterVersion: CanonicalTextFormatterVer, ProjectionOrder: []string{"path", "kind", "qualified_symbol", "signature", "body"}}
	source := profile.EmbeddingSourceProfile{Provider: resolved.Embedding.Model.Provider, Model: resolved.Embedding.Model.Model, SourceDimensions: resolved.Embedding.Model.SourceDimensions, OutputDType: resolved.Embedding.Model.OutputDType, InputTypeMapping: profile.InputTypeMapping{Document: resolved.Embedding.Model.DocumentInputType, Query: resolved.Embedding.Model.QueryInputType}, Truncation: resolved.Embedding.Model.Truncation, AdapterVersion: resolved.Embedding.Model.AdapterVersion}
	indexFingerprint, err := Fingerprint(index, IndexProfileDomain)
	if err != nil {
		return DesiredProfiles{}, err
	}
	canonicalFingerprint, err := Fingerprint(canonical, CanonicalTextDomain)
	if err != nil {
		return DesiredProfiles{}, err
	}
	sourceFingerprint, err := Fingerprint(source, SourceProfileDomain)
	if err != nil {
		return DesiredProfiles{}, err
	}
	space := profile.VectorSpaceProfile{SourceProfileFingerprint: sourceFingerprint, ServingDimensions: resolved.Embedding.ServingDimensions, ReducerID: resolved.Embedding.ReducerID, NormalizerID: resolved.Embedding.NormalizerID, Metric: resolved.Embedding.Metric}
	spaceFingerprint, err := Fingerprint(space, VectorSpaceDomain)
	if err != nil {
		return DesiredProfiles{}, err
	}
	codecID, err := storageCodecID(resolved.Embedding.StorageCodec)
	if err != nil {
		return DesiredProfiles{}, err
	}
	storage := profile.VectorStorageProfile{VectorSpaceProfileFingerprint: spaceFingerprint, StorageCodecID: codecID}
	storageFingerprint, err := Fingerprint(storage, VectorStorageDomain)
	if err != nil {
		return DesiredProfiles{}, err
	}
	policy := profile.ServingPolicyProfile{DefaultMode: resolved.Search.DefaultMode, AllowPaidQueryEmbedding: resolved.Search.AllowPaidQueryEmbedding, ReturnK: resolved.Search.ReturnK, CandidateK: resolved.Search.CandidateK, RRFK: resolved.Search.RRFK, QueryTextFormatVersion: resolved.Search.QueryTextFormatVersion, MaxQueryBytes: resolved.Search.QueryLimits.MaxBytes, MaxQueryTokens: resolved.Search.QueryLimits.MaxTokens, MaxQueryTokenRunes: resolved.Search.QueryLimits.MaxTokenRunes, FTSSymbolWeight: resolved.Search.FTSSymbolWeight, FTSBodyWeight: resolved.Search.FTSBodyWeight, HardMaxInlineBytes: resolved.MCP.HardMaxInlineBytes}
	policyFingerprint, err := Fingerprint(policy, ServingPolicyDomain)
	if err != nil {
		return DesiredProfiles{}, err
	}
	return DesiredProfiles{Index: index, CanonicalText: canonical, Source: source, VectorSpace: space, VectorStorage: storage, Fingerprints: profile.ProfileFingerprints{Index: indexFingerprint, CanonicalText: canonicalFingerprint, Source: sourceFingerprint, VectorSpace: spaceFingerprint, VectorStorage: storageFingerprint, Policy: policyFingerprint}}, nil
}

func storageCodecID(codec string) (string, error) {
	if codec == StorageCodecInt8 {
		return vector.Int8CodecID, nil
	}
	return "", fmt.Errorf("unsupported storage codec")
}
