// Package search provides the transport-independent Phase 11 retrieval core.
// It owns no MCP schema, filesystem freshness check, or lab dependency.
package search

import "cidx/internal/profile"

type SearchMode string

const (
	ModeFTS    SearchMode = "fts"
	ModeHybrid SearchMode = "hybrid"
)

type FallbackReason string

const (
	FallbackPaidQueryDisabled             FallbackReason = "PAID_QUERY_DISABLED"
	FallbackProfileReconciliationRequired FallbackReason = "PROFILE_RECONCILIATION_REQUIRED"
	FallbackNoValidDocumentVectors        FallbackReason = "NO_VALID_DOCUMENT_VECTORS"
	FallbackAPIKeyMissing                 FallbackReason = "API_KEY_MISSING"
	FallbackQueryEmbeddingFailed          FallbackReason = "QUERY_EMBEDDING_FAILED"
	FallbackQueryProfileChanged           FallbackReason = "QUERY_PROFILE_CHANGED_DURING_REQUEST"
	FallbackVectorSnapshotInvalid         FallbackReason = "VECTOR_SNAPSHOT_INVALID"
)

type Request struct {
	Query                   string
	Mode                    SearchMode
	K                       int
	EffectiveMaxInlineBytes int
}

type ByteLineRange struct {
	StartByte int `json:"start_byte"`
	EndByte   int `json:"end_byte"`
	StartLine int `json:"start_line"`
	EndLine   int `json:"end_line"`
}

type OmissionReason string

const (
	BodyIncluded                OmissionReason = ""
	BodyOmittedBudget           OmissionReason = "INLINE_BUDGET_EXCEEDED"
	BodyOmittedNoMatchedSegment OmissionReason = "NO_FITTING_INDEXED_BODY"
)

type ScoreSource string

const (
	ScoreSourceFTS    ScoreSource = "fts"
	ScoreSourceVector ScoreSource = "vector"
	ScoreSourceBoth   ScoreSource = "both"
)

type Hit struct {
	ChunkID                                       int64
	Path, Language, Kind, Symbol, QualifiedSymbol string
	Signature                                     string
	ParentRange                                   ByteLineRange
	IndexedSHA256                                 string
	LexicalRank, VectorRank                       int
	SymbolRank, PathRank, DescriptiveRank         int
	SymbolMatchTier, PathMatchTier                int
	MatchedTerms, SelectedTerms                   int
	LexicalScore                                  float64
	LexicalSources                                []string
	FusedScore                                    float64
	ScoreSource                                   ScoreSource
	MatchedSegment                                *ByteLineRange
	Body                                          []byte
	BodyRange                                     *ByteLineRange
	BodyComplete                                  bool
	BodyBytes                                     int
	BodyOmissionReason                            OmissionReason
	SourceState                                   string
}

type Response struct {
	RequestedMaxInlineBytes, EffectiveMaxInlineBytes  int
	MaxInlineBytesClamped                             bool
	RequestedMode, EffectiveMode                      SearchMode
	QueryEmbeddingUsed                                bool
	LexicalQueryPlannerVersion                        int
	QueryTextFormatVersion                            int
	QueryShape                                        string
	LexicalBooleanForm                                string
	ExplicitAnchors, PathAnchors                      []string
	SelectedDescriptiveTerms, DroppedDescriptiveTerms []string
	SymbolCandidateCount, PathCandidateCount          int
	DescriptiveCandidateCount, LexicalCandidateCount  int
	LexicalCandidateZero                              bool
	FallbackReason                                    FallbackReason
	Generation                                        int64
	ManifestSHA256                                    string
	SourceProfile                                     profile.Fingerprint
	VectorSpaceProfile                                profile.Fingerprint
	VectorStorageProfile                              profile.Fingerprint
	CoverageNumerator                                 int
	CoverageDenominator                               int
	PartialVectorCoverage                             bool
	VectorCoverageObserved                            bool
	InlineBytesUsed                                   int
	InlineLimited                                     bool
	Hits                                              []Hit
}
