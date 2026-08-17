// Package relationdiag implements the intentionally isolated relation/usage
// graph diagnostic.  It is not imported by production indexing, search, MCP,
// or vector packages.
package relationdiag

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"sort"
	"strings"

	"cidx/internal/store"
)

const (
	SchemaVersion      = 1
	ProtocolVersion    = "cidx.relation-diagnostic.v1"
	IdentityPolicyID   = "path-indexed-sha-language-kind-qualified-symbol-byte-range-v1"
	SelectionPolicyID  = "one-hop-kind-order-type-call-member-one-bundle-v2"
	BodyPolicyID       = "related-complete-parent-2x1024-v1"
	MaxDenseDepth      = 20
	ProtectedPrimaryK  = 5
	RelatedParentLimit = 2
	RelatedBodyLimit   = 1024
)

type RelationKind string

const (
	Calls    RelationKind = "CALLS"
	TypeRef  RelationKind = "TYPE_REF"
	MemberOf RelationKind = "MEMBER_OF"
)

func (v RelationKind) Valid() bool { return v == Calls || v == TypeRef || v == MemberOf }

type Outcome string

const (
	ResolvedUnique     Outcome = "RESOLVED_UNIQUE"
	Unresolved         Outcome = "UNRESOLVED"
	Ambiguous          Outcome = "AMBIGUOUS"
	OutOfCorpus        Outcome = "OUT_OF_CORPUS"
	OutOfResolverScope Outcome = "OUT_OF_RESOLVER_SCOPE"
	ParentMappingFail  Outcome = "PARENT_MAPPING_FAILED"
	NoEnclosingParent  Outcome = "NO_ENCLOSING_PARENT"
)

func (v Outcome) Valid() bool {
	return v == ResolvedUnique || v == Unresolved || v == Ambiguous || v == OutOfCorpus || v == OutOfResolverScope || v == ParentMappingFail || v == NoEnclosingParent
}

type Direction string

const (
	Forward Direction = "FORWARD"
	Reverse Direction = "REVERSE"
)

type Parent struct {
	ID              string `json:"parent_id"`
	Path            string `json:"path"`
	IndexedSHA256   string `json:"indexed_sha256"`
	Language        string `json:"language"`
	Kind            string `json:"kind"`
	Symbol          string `json:"symbol"`
	QualifiedSymbol string `json:"qualified_symbol"`
	StartByte       int    `json:"start_byte"`
	EndByte         int    `json:"end_byte"`
	SourceBody      string `json:"-"`
}

func ParentFromStored(value store.SemanticParent) (Parent, error) {
	if !validParentFields(value.Path, value.IndexedSHA256, value.Language, value.Kind, value.QualifiedSymbol, value.StartByte, value.EndByte) {
		return Parent{}, fmt.Errorf("invalid semantic parent")
	}
	parent := Parent{Path: value.Path, IndexedSHA256: value.IndexedSHA256, Language: value.Language, Kind: value.Kind, Symbol: value.Symbol, QualifiedSymbol: value.QualifiedSymbol, StartByte: value.StartByte, EndByte: value.EndByte, SourceBody: value.SourceBody}
	parent.ID = ParentID(parent)
	return parent, nil
}

func ParentID(parent Parent) string {
	canonical := strings.Join([]string{parent.Path, parent.IndexedSHA256, parent.Language, parent.Kind, parent.QualifiedSymbol, fmt.Sprintf("%d", parent.StartByte), fmt.Sprintf("%d", parent.EndByte)}, "\x00")
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}

func ParentInventory(parents []store.SemanticParent) ([]Parent, error) {
	result := make([]Parent, 0, len(parents))
	seen := map[string]bool{}
	for _, source := range parents {
		parent, err := ParentFromStored(source)
		if err != nil {
			return nil, err
		}
		if seen[parent.ID] {
			return nil, fmt.Errorf("duplicate semantic parent identity")
		}
		seen[parent.ID] = true
		result = append(result, parent)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

type Candidate struct {
	ID             string       `json:"id"`
	Path           string       `json:"path"`
	Language       string       `json:"language"`
	Kind           RelationKind `json:"kind"`
	StartByte      int          `json:"start_byte"`
	EndByte        int          `json:"end_byte"`
	SourceParentID string       `json:"source_parent_id,omitempty"`
}

func CandidateID(file, language, parentID string, kind RelationKind, start, end int) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{file, language, parentID, string(kind), fmt.Sprintf("%d", start), fmt.Sprintf("%d", end)}, "\x00")))
	return hex.EncodeToString(sum[:])
}

type Occurrence struct {
	ID             string       `json:"relation_id"`
	SourceParentID string       `json:"source_parent_id,omitempty"`
	TargetParentID string       `json:"target_parent_id,omitempty"`
	Path           string       `json:"path"`
	Language       string       `json:"language"`
	Kind           RelationKind `json:"relation_kind"`
	StartByte      int          `json:"start_byte"`
	EndByte        int          `json:"end_byte"`
	Outcome        Outcome      `json:"outcome"`
	Resolver       string       `json:"resolver"`
}

func (v Occurrence) Validate() error {
	if v.ID == "" || !v.Kind.Valid() || !v.Outcome.Valid() || !validRelative(v.Path) || v.StartByte < 0 || v.EndByte <= v.StartByte || v.Resolver == "" {
		return fmt.Errorf("invalid relation occurrence")
	}
	if v.Outcome == ResolvedUnique {
		if v.SourceParentID == "" || v.TargetParentID == "" {
			return fmt.Errorf("resolved occurrence is missing parent")
		}
	} else if v.TargetParentID != "" {
		return fmt.Errorf("unresolved occurrence has target")
	}
	return nil
}

func OccurrenceID(candidate Candidate) string {
	return CandidateID(candidate.Path, candidate.Language, candidate.SourceParentID, candidate.Kind, candidate.StartByte, candidate.EndByte)
}

func validRelative(value string) bool {
	return value != "" && !strings.Contains(value, "\\") && !path.IsAbs(value) && path.Clean(value) == value && value != "." && value != ".." && !strings.HasPrefix(value, "../")
}

func validParentFields(file, digest, language, kind, symbol string, start, end int) bool {
	if !validRelative(file) || len(digest) != 64 || language == "" || kind == "" || symbol == "" || start < 0 || end <= start {
		return false
	}
	for _, c := range digest {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func ParentContaining(parents []Parent, file string, start, end int) (Parent, bool) {
	var found *Parent
	for index := range parents {
		parent := &parents[index]
		if parent.Path != file || parent.StartByte > start || parent.EndByte < end {
			continue
		}
		if found == nil || parent.EndByte-parent.StartByte < found.EndByte-found.StartByte {
			found = parent
		}
	}
	if found == nil {
		return Parent{}, false
	}
	return *found, true
}
