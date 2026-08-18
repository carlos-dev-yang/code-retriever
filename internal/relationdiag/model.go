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
	SchemaVersion                         = 3
	ProbeSchemaVersion                    = 1
	ProtocolVersion                       = "cidx.relation-diagnostic.v3"
	IdentityPolicyID                      = "path-indexed-sha-language-kind-qualified-symbol-byte-range-v1"
	MetadataPolicyID                      = "occurrence-context-ast-compiler-v2"
	DenseFirstPolicyID                    = "query-edge-metadata-dense-first-v1"
	ValueParameterDenseFirstPolicyID      = "query-edge-value-parameter-dense-first-v1"
	GraphFirstPolicyID                    = "query-edge-metadata-graph-first-dense-crossover-v1"
	AnchorEdgeRawFrequencyPolicyID        = "anchor-edge-raw-frequency-control-v1"
	AnchorEdgeSourceNormalizedPolicyID    = "anchor-edge-source-normalized-focus-v1"
	AnchorEdgeBidirectionalPolicyID       = "anchor-edge-bidirectional-specificity-v1"
	AnchorEdgeIncomingPopularityPolicyID  = "anchor-edge-incoming-popularity-control-v1"
	AnchorFrontierCapOnlyPolicyID         = "anchor-frontier-cap-only-v1"
	AnchorFrontierBridgePolicyID          = "anchor-frontier-bridge-abstention-v1"
	AnchorFrontierGraphOnlyParetoPolicyID = "anchor-frontier-graph-only-pareto-v1"
	// SelectionPolicyID remains the default only for legacy focused fixtures.
	SelectionPolicyID   = DenseFirstPolicyID
	BodyPolicyID        = "related-complete-parent-2x1024-v1"
	MaxDenseDepth       = 20
	ProtectedPrimaryK   = 5
	RelatedParentLimit  = 2
	RelatedBodyLimit    = 1024
	FrontierGlobalLimit = 32
	FrontierBucketLimit = 2
)

type OccurrenceZone string

const (
	SignatureZone   OccurrenceZone = "SIGNATURE"
	BodyZone        OccurrenceZone = "BODY"
	TypeBodyZone    OccurrenceZone = "TYPE_BODY"
	InitializerZone OccurrenceZone = "INITIALIZER"
)

func (v OccurrenceZone) Valid() bool {
	return v == SignatureZone || v == BodyZone || v == TypeBodyZone || v == InitializerZone
}

type OccurrenceRole string

const (
	CallFreeFunctionRole   OccurrenceRole = "CALL_FREE_FUNCTION"
	CallMethodRole         OccurrenceRole = "CALL_METHOD"
	CallableValueRole      OccurrenceRole = "CALLABLE_VALUE"
	TypeParameterRole      OccurrenceRole = "TYPE_PARAMETER"
	TypeValueParameterRole OccurrenceRole = "TYPE_VALUE_PARAMETER"
	TypeReturnRole         OccurrenceRole = "TYPE_RETURN"
	TypeFieldRole          OccurrenceRole = "TYPE_FIELD"
	TypeAliasRole          OccurrenceRole = "TYPE_ALIAS"
	TypeHeritageRole       OccurrenceRole = "TYPE_HERITAGE"
	TypeArgumentRole       OccurrenceRole = "TYPE_ARGUMENT"
	TypeLocalRole          OccurrenceRole = "TYPE_LOCAL"
	TypeOtherRole          OccurrenceRole = "TYPE_OTHER"
	MemberReceiverRole     OccurrenceRole = "MEMBER_RECEIVER"
	MemberDeclarationRole  OccurrenceRole = "MEMBER_DECLARATION"
)

func (v OccurrenceRole) Valid() bool {
	switch v {
	case CallFreeFunctionRole, CallMethodRole, CallableValueRole, TypeParameterRole, TypeValueParameterRole, TypeReturnRole, TypeFieldRole, TypeAliasRole, TypeHeritageRole, TypeArgumentRole, TypeLocalRole, TypeOtherRole, MemberReceiverRole, MemberDeclarationRole:
		return true
	}
	return false
}

type FlowRole string

const (
	FlowNone        FlowRole = "NONE"
	FlowReturn      FlowRole = "RETURN"
	FlowAssignment  FlowRole = "ASSIGNMENT"
	FlowCondition   FlowRole = "CONDITION"
	FlowArgument    FlowRole = "ARGUMENT"
	FlowDeclaration FlowRole = "DECLARATION"
)

func (v FlowRole) Valid() bool {
	return v == FlowNone || v == FlowReturn || v == FlowAssignment || v == FlowCondition || v == FlowArgument || v == FlowDeclaration
}

type FileRole string

const (
	ProductionFileRole FileRole = "PRODUCTION"
	TestFileRole       FileRole = "TEST"
	ExampleFileRole    FileRole = "EXAMPLE"
	BenchmarkFileRole  FileRole = "BENCHMARK"
)

func (v FileRole) Valid() bool {
	return v == ProductionFileRole || v == TestFileRole || v == ExampleFileRole || v == BenchmarkFileRole
}

type ExecutionMode string

const (
	DirectExecution     ExecutionMode = "DIRECT"
	DeferredExecution   ExecutionMode = "DEFERRED"
	ConcurrentExecution ExecutionMode = "CONCURRENT"
	AwaitedExecution    ExecutionMode = "AWAITED"
)

func (v ExecutionMode) Valid() bool {
	return v == DirectExecution || v == DeferredExecution || v == ConcurrentExecution || v == AwaitedExecution
}

type ControlRole string

const (
	ControlNone     ControlRole = "NONE"
	ControlBranch   ControlRole = "BRANCH"
	ControlLoop     ControlRole = "LOOP"
	ControlSwitch   ControlRole = "SWITCH"
	ControlTryCatch ControlRole = "TRY_CATCH"
)

func (v ControlRole) Valid() bool {
	return v == ControlNone || v == ControlBranch || v == ControlLoop || v == ControlSwitch || v == ControlTryCatch
}

type OccurrenceMetadata struct {
	Zone               OccurrenceZone `json:"zone"`
	Role               OccurrenceRole `json:"role"`
	Flow               FlowRole       `json:"flow_role"`
	FileRole           FileRole       `json:"file_role"`
	Execution          ExecutionMode  `json:"execution_mode"`
	Control            ControlRole    `json:"control_role"`
	ContextIdentifiers []string       `json:"context_identifiers"`
	SourceOrdinal      int            `json:"source_ordinal"`
}

func DefaultOccurrenceMetadata(file string, ordinal int) OccurrenceMetadata {
	if ordinal < 1 {
		ordinal = 1
	}
	return OccurrenceMetadata{Zone: BodyZone, Role: TypeOtherRole, Flow: FlowNone, FileRole: FileRoleForPath(file), Execution: DirectExecution, Control: ControlNone, ContextIdentifiers: []string{}, SourceOrdinal: ordinal}
}

func (v OccurrenceMetadata) Validate() error {
	if !v.Zone.Valid() || !v.Role.Valid() || !v.Flow.Valid() || !v.FileRole.Valid() || !v.Execution.Valid() || !v.Control.Valid() || v.ContextIdentifiers == nil || v.SourceOrdinal < 1 || len(v.ContextIdentifiers) > 8 {
		return fmt.Errorf("invalid occurrence metadata")
	}
	seen := map[string]bool{}
	for _, token := range v.ContextIdentifiers {
		if token == "" || strings.TrimSpace(token) != token || strings.ContainsAny(token, "\\/\x00") || seen[token] {
			return fmt.Errorf("invalid occurrence context identifier")
		}
		seen[token] = true
	}
	return nil
}

func FileRoleForPath(value string) FileRole {
	lower := strings.ToLower(path.Clean(value))
	base := path.Base(lower)
	if strings.HasSuffix(base, "_example_test.go") || strings.Contains(base, ".example.") || strings.Contains(lower, "/example/") || strings.Contains(lower, "/examples/") || strings.HasPrefix(base, "example_") {
		return ExampleFileRole
	}
	if strings.HasSuffix(base, "_bench_test.go") || strings.Contains(base, "benchmark") || strings.Contains(base, ".bench.") || strings.HasPrefix(base, "bench_") || strings.Contains(lower, "/benchmark/") || strings.Contains(lower, "/benchmarks/") {
		return BenchmarkFileRole
	}
	if strings.HasSuffix(base, "_test.go") || strings.Contains(base, ".test.") || strings.Contains(lower, "/test/") || strings.Contains(lower, "/tests/") {
		return TestFileRole
	}
	return ProductionFileRole
}

type RelationKind string

const (
	Calls    RelationKind = "CALLS"
	TypeRef  RelationKind = "TYPE_REF"
	MemberOf RelationKind = "MEMBER_OF"
)

func (v RelationKind) Valid() bool { return v == Calls || v == TypeRef || v == MemberOf }

// StructuralTier is a mechanical classification of an already resolved graph
// occurrence. It is evaluation-only and deliberately independent of labels.
type StructuralTier string

const (
	DeclarationContractTier  StructuralTier = "DECLARATION_CONTRACT"
	ExecutableDependencyTier StructuralTier = "EXECUTABLE_DEPENDENCY"
	BodyReferenceTier        StructuralTier = "BODY_REFERENCE"
	DeclarationStructureTier StructuralTier = "DECLARATION_STRUCTURE"
)

func (v StructuralTier) Valid() bool {
	return v == DeclarationContractTier || v == ExecutableDependencyTier || v == BodyReferenceTier || v == DeclarationStructureTier
}

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
	ID              string   `json:"parent_id"`
	Path            string   `json:"path"`
	IndexedSHA256   string   `json:"indexed_sha256"`
	Language        string   `json:"language"`
	Kind            string   `json:"kind"`
	Symbol          string   `json:"symbol"`
	QualifiedSymbol string   `json:"qualified_symbol"`
	StartByte       int      `json:"start_byte"`
	EndByte         int      `json:"end_byte"`
	SourceBody      string   `json:"-"`
	FileRole        FileRole `json:"file_role"`
	Deprecated      bool     `json:"deprecated"`
}

func ParentFromStored(value store.SemanticParent) (Parent, error) {
	if !validParentFields(value.Path, value.IndexedSHA256, value.Language, value.Kind, value.QualifiedSymbol, value.StartByte, value.EndByte) {
		return Parent{}, fmt.Errorf("invalid semantic parent")
	}
	parent := Parent{Path: value.Path, IndexedSHA256: value.IndexedSHA256, Language: value.Language, Kind: value.Kind, Symbol: value.Symbol, QualifiedSymbol: value.QualifiedSymbol, StartByte: value.StartByte, EndByte: value.EndByte, SourceBody: value.SourceBody, FileRole: FileRoleForPath(value.Path), Deprecated: deprecatedParent(value.Language, value.SourceBody)}
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
	ID             string             `json:"id"`
	Path           string             `json:"path"`
	Language       string             `json:"language"`
	Kind           RelationKind       `json:"kind"`
	StartByte      int                `json:"start_byte"`
	EndByte        int                `json:"end_byte"`
	SourceParentID string             `json:"source_parent_id,omitempty"`
	Metadata       OccurrenceMetadata `json:"metadata"`
}

func CandidateID(file, language, parentID string, kind RelationKind, start, end int) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{file, language, parentID, string(kind), fmt.Sprintf("%d", start), fmt.Sprintf("%d", end)}, "\x00")))
	return hex.EncodeToString(sum[:])
}

type Occurrence struct {
	ID             string             `json:"relation_id"`
	SourceParentID string             `json:"source_parent_id,omitempty"`
	TargetParentID string             `json:"target_parent_id,omitempty"`
	Path           string             `json:"path"`
	Language       string             `json:"language"`
	Kind           RelationKind       `json:"relation_kind"`
	StartByte      int                `json:"start_byte"`
	EndByte        int                `json:"end_byte"`
	Outcome        Outcome            `json:"outcome"`
	Resolver       string             `json:"resolver"`
	Metadata       OccurrenceMetadata `json:"metadata"`
}

func (v Occurrence) Validate() error {
	if v.ID == "" || !v.Kind.Valid() || !v.Outcome.Valid() || !validRelative(v.Path) || v.StartByte < 0 || v.EndByte <= v.StartByte || v.Resolver == "" || v.Metadata.Validate() != nil {
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
