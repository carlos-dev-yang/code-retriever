package chunk

import "context"

// Language is shared by configuration, chunkers, storage, and evaluation.
type Language string

const (
	Go         Language = "go"
	TypeScript Language = "typescript"
	TSX        Language = "tsx"
)

func (language Language) Valid() bool {
	return language == Go || language == TypeScript || language == TSX
}

type ChunkKind string

const (
	Function ChunkKind = "function"
	Method   ChunkKind = "method"
	Type     ChunkKind = "type"
)

func (kind ChunkKind) Valid() bool {
	return kind == Function || kind == Method || kind == Type
}

type ChunkRequest struct {
	Path               string
	Source             []byte
	SegmentationPolicy SegmentationPolicy
}

// SegmentationPolicy is injected by the caller and contains no config/profile
// dependency, allowing Go and TypeScript chunkers to share one contract.
type SegmentationPolicy struct {
	Version          int
	BoundaryPolicyID string
	MaxSegmentBytes  int
}

func (value SegmentationPolicy) Valid() bool {
	return value.Version > 0 && value.BoundaryPolicyID != "" && value.MaxSegmentBytes > 0
}

func (value ChunkRequest) Clone() ChunkRequest {
	value.Source = append([]byte(nil), value.Source...)
	return value
}

func (value ChunkRequest) Validate() bool {
	return value.Path != "" && len(value.Source) > 0 && value.SegmentationPolicy.Valid()
}

type ChunkResult struct {
	Chunks      []SourceChunk
	Diagnostics []ChunkDiagnostic
	Parser      ParserMetadata
}

type DiagnosticSeverity string

const (
	DiagnosticWarning DiagnosticSeverity = "warning"
	DiagnosticError   DiagnosticSeverity = "error"
)

type ChunkDiagnostic struct {
	Code        string
	Severity    DiagnosticSeverity
	Range       *ByteRange
	SafeToIndex bool
}
type ParserMetadata struct {
	ParserID       string
	GrammarVersion string
	RootKind       string
	HasError       bool
}

// Chunker is deliberately language-neutral. Phases 03 and 04 supply language
// implementations without redefining persisted chunk values.
type Chunker interface {
	Language() Language
	Chunk(context.Context, ChunkRequest) (ChunkResult, error)
}
