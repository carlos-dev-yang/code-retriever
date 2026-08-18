// Package eval implements development-only, offline evaluation support. It
// deliberately has no provider, lab, or production-MCP dependency.
package eval

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"cidx/internal/config"
	"cidx/internal/evalcontract"
)

const CorpusManifestVersion = 1

const (
	maxManifestPatterns = 256
	maxPatternLength    = 512
	maxPatternSegments  = 64
)

type CorpusManifest struct {
	SchemaVersion         int                     `json:"schema_version"`
	CorpusID              string                  `json:"corpus_id"`
	UpstreamURL           string                  `json:"upstream_url"`
	PinnedCommit          string                  `json:"pinned_commit"`
	LicenseSPDX           string                  `json:"license_spdx"`
	LicenseEvidence       string                  `json:"license_evidence"`
	RedistributionNotice  string                  `json:"redistribution_notice,omitempty"`
	LanguageSlices        []evalcontract.Language `json:"language_slices"`
	RootSubdir            string                  `json:"root_subdir,omitempty"`
	TypeScriptConfig      string                  `json:"typescript_config,omitempty"`
	Include               []string                `json:"include"`
	Exclude               []string                `json:"exclude"`
	ExpectedContentSHA256 string                  `json:"expected_content_hash"`
	ExpectedTreeHash      string                  `json:"expected_tree_hash"`
	CleanTreeRequired     bool                    `json:"clean_tree_required"`
}

func LoadCorpusManifest(data []byte) (CorpusManifest, error) {
	var value CorpusManifest
	if err := decodeStrict(data, &value); err != nil {
		return CorpusManifest{}, fmt.Errorf("decode corpus manifest: %w", err)
	}
	if err := value.Validate(); err != nil {
		return CorpusManifest{}, err
	}
	return value, nil
}

func (value CorpusManifest) Validate() error {
	if value.SchemaVersion != CorpusManifestVersion || !validID(value.CorpusID) || !validUpstream(value.UpstreamURL) || !validCommit(value.PinnedCommit) || !validSPDX(value.LicenseSPDX) || !validRelative(value.LicenseEvidence, false) || !validRelative(value.RootSubdir, true) || !validRelative(value.TypeScriptConfig, true) || !validSHA256(value.ExpectedContentSHA256) || !validCommit(value.ExpectedTreeHash) {
		return fmt.Errorf("invalid corpus manifest")
	}
	if !validLanguages(value.LanguageSlices) || !validPatterns(value.Include, false) || !validPatterns(value.Exclude, true) {
		return fmt.Errorf("invalid corpus selection policy")
	}
	if value.TypeScriptConfig != "" && !containsTypeScript(value.LanguageSlices) {
		return fmt.Errorf("TypeScript config requires a TypeScript language slice")
	}
	return nil
}

func containsTypeScript(values []evalcontract.Language) bool {
	for _, value := range values {
		if value == evalcontract.TypeScript || value == evalcontract.TSX {
			return true
		}
	}
	return false
}

// Fingerprint is RFC 8785 canonical JSON followed by a plain SHA-256. The
// digest is portable and must not contain a local binding path.
func (value CorpusManifest) Fingerprint() (string, error) {
	if err := value.Validate(); err != nil {
		return "", err
	}
	canonical, err := config.CanonicalJSON(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

type EvaluationDataset struct {
	SchemaVersion int                           `json:"schema_version"`
	Version       string                        `json:"version"`
	CorpusID      string                        `json:"corpus_id"`
	Cases         []evalcontract.EvaluationCase `json:"cases"`
}

func LoadDataset(data []byte) (EvaluationDataset, error) {
	var value EvaluationDataset
	if err := decodeStrict(data, &value); err != nil {
		return EvaluationDataset{}, fmt.Errorf("decode evaluation dataset: %w", err)
	}
	if err := value.Validate(); err != nil {
		return EvaluationDataset{}, err
	}
	return value, nil
}

func (value EvaluationDataset) Validate() error {
	if value.SchemaVersion != evalcontract.SchemaVersion || !validID(value.Version) || !validID(value.CorpusID) || len(value.Cases) == 0 {
		return fmt.Errorf("invalid evaluation dataset")
	}
	seen := map[string]struct{}{}
	for _, c := range value.Cases {
		if err := c.Validate(); err != nil {
			return fmt.Errorf("invalid case %q: %w", c.ID, err)
		}
		if _, ok := seen[c.ID]; ok {
			return fmt.Errorf("duplicate evaluation case %q", c.ID)
		}
		seen[c.ID] = struct{}{}
	}
	return nil
}

func (value EvaluationDataset) Fingerprint() (string, error) {
	if err := value.Validate(); err != nil {
		return "", err
	}
	canonical, err := config.CanonicalJSON(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

func decodeStrict(data []byte, target any) error {
	if !utf8.Valid(data) {
		return fmt.Errorf("JSON is not valid UTF-8")
	}
	if err := rejectDuplicateKeys(data); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	if token, err := d.Token(); err != io.EOF || token != nil {
		return fmt.Errorf("JSON contains trailing data")
	}
	return nil
}

func rejectDuplicateKeys(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	var walk func() error
	walk = func() error {
		t, err := d.Token()
		if err != nil {
			return err
		}
		delim, ok := t.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]struct{}{}
			for d.More() {
				k, err := d.Token()
				if err != nil {
					return err
				}
				key, ok := k.(string)
				if !ok {
					return fmt.Errorf("invalid object key")
				}
				if _, exists := seen[key]; exists {
					return fmt.Errorf("duplicate JSON field %q", key)
				}
				seen[key] = struct{}{}
				if err := walk(); err != nil {
					return err
				}
			}
			_, err := d.Token()
			return err
		case '[':
			for d.More() {
				if err := walk(); err != nil {
					return err
				}
			}
			_, err := d.Token()
			return err
		default:
			return fmt.Errorf("invalid JSON delimiter")
		}
	}
	if err := walk(); err != nil {
		return err
	}
	_, err := d.Token()
	if err != io.EOF {
		return fmt.Errorf("JSON contains trailing data")
	}
	return nil
}

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
var commitPattern = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)
var spdxPattern = regexp.MustCompile(`^[A-Za-z0-9.+()\- ]+$`)
var hashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func validID(v string) bool     { return idPattern.MatchString(v) }
func validCommit(v string) bool { return commitPattern.MatchString(v) }
func validSHA256(v string) bool { return hashPattern.MatchString(v) }
func validSPDX(v string) bool {
	return v != "" && spdxPattern.MatchString(v) && !strings.HasPrefix(v, " ") && !strings.HasSuffix(v, " ")
}
func validUpstream(v string) bool {
	u, err := url.Parse(v)
	return err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil && u.Fragment == "" && !strings.Contains(u.Path, "..")
}
func validRelative(v string, allowEmpty bool) bool {
	if v == "" {
		return allowEmpty
	}
	return !strings.Contains(v, "\\") && !path.IsAbs(v) && path.Clean(v) == v && v != "." && !strings.HasPrefix(v, "../") && v != ".."
}
func validLanguages(values []evalcontract.Language) bool {
	seen := map[evalcontract.Language]struct{}{}
	for _, v := range values {
		if v != evalcontract.Go && v != evalcontract.TypeScript && v != evalcontract.TSX {
			return false
		}
		if _, ok := seen[v]; ok {
			return false
		}
		seen[v] = struct{}{}
	}
	return len(values) > 0
}
func validPatterns(values []string, allowEmpty bool) bool {
	if len(values) == 0 {
		return allowEmpty
	}
	if len(values) > maxManifestPatterns {
		return false
	}
	seen := map[string]struct{}{}
	for _, v := range values {
		if _, err := compileGlob(v); err != nil {
			return false
		}
		if _, ok := seen[v]; ok {
			return false
		}
		seen[v] = struct{}{}
	}
	return true
}

// compileGlob accepts slash-separated patterns and gives ** its only useful
// meaning: a complete path segment that matches zero or more segments. This
// avoids path.Match's non-recursive treatment and rejects ambiguous **foo
// forms before a manifest can become an official selection policy.
func compileGlob(value string) ([]string, error) {
	if value == "" || len(value) > maxPatternLength || strings.Contains(value, "\\") || path.IsAbs(value) || path.Clean(value) != value {
		return nil, fmt.Errorf("invalid glob")
	}
	segments := strings.Split(value, "/")
	if len(segments) > maxPatternSegments {
		return nil, fmt.Errorf("too many glob segments")
	}
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." || (strings.Contains(segment, "**") && segment != "**") {
			return nil, fmt.Errorf("invalid glob segment")
		}
		if segment != "**" {
			if _, err := path.Match(segment, ""); err != nil {
				return nil, fmt.Errorf("invalid glob segment")
			}
		}
	}
	return segments, nil
}

func sortedStrings(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}
