// Package evalcontract owns portable evaluation wire values only. It contains
// no retrieval, metric, or promotion-threshold implementation.
package evalcontract

type Language string

const (
	Go         Language = "go"
	TypeScript Language = "typescript"
	TSX        Language = "tsx"
	Mixed      Language = "mixed"
)

type AnswerMode string

const (
	Single      AnswerMode = "SINGLE"
	BestN       AnswerMode = "BEST_N"
	Exhaustive  AnswerMode = "EXHAUSTIVE"
	Abstainable AnswerMode = "ABSTAINABLE"
)

type DatasetSplit string

const (
	Calibration  DatasetSplit = "calibration"
	Confirmation DatasetSplit = "confirmation"
)

type Relevance int

const (
	Irrelevant        Relevance = 0
	UsefulSupport     Relevance = 1
	DirectRequirement Relevance = 2
)

type FirstLoss string

const (
	SourceDiscovery       FirstLoss = "SOURCE_DISCOVERY"
	ParseOrChunk          FirstLoss = "PARSE_OR_CHUNK"
	FTSCandidateMiss      FirstLoss = "FTS_CANDIDATE_MISS"
	DenseSegmentMiss      FirstLoss = "DENSE_SEGMENT_MISS"
	ProviderUnionMiss     FirstLoss = "PROVIDER_UNION_MISS"
	SegmentParentCollapse FirstLoss = "SEGMENT_PARENT_COLLAPSE"
	RRFFusion             FirstLoss = "RRF_FUSION"
	BodyPackaging         FirstLoss = "BODY_PACKAGING"
	AssistantUse          FirstLoss = "ASSISTANT_USE"
	AssistantResolution   FirstLoss = "ASSISTANT_RESOLUTION"
	NotObserved           FirstLoss = "NOT_OBSERVED"
	NoLoss                FirstLoss = "NO_LOSS"
)

type Stage string

const (
	StageSourceDiscovery     Stage = "source_discovery"
	StageParserChunker       Stage = "parser_chunker"
	StageFTSCandidate        Stage = "fts_candidate"
	StageDenseSegment        Stage = "dense_segment"
	StageProviderUnion       Stage = "provider_union"
	StageParentCollapse      Stage = "segment_parent_collapse"
	StageRRFFusion           Stage = "rrf_fusion"
	StageBodyPackaging       Stage = "body_packaging"
	StageAssistantUse        Stage = "assistant_use"
	StageAssistantResolution Stage = "assistant_resolution"
	StageOperational         Stage = "operational"
)

var PlannedStages = []Stage{StageSourceDiscovery, StageParserChunker, StageFTSCandidate, StageDenseSegment, StageProviderUnion, StageParentCollapse, StageRRFFusion, StageBodyPackaging, StageAssistantUse, StageAssistantResolution, StageOperational}

type TerminalState string

const (
	TerminalComplete TerminalState = "complete"
	TerminalFailed   TerminalState = "failed"
)

type ObservationStatus string

const (
	Observed               ObservationStatus = "OBSERVED"
	ObservationNotObserved ObservationStatus = "NOT_OBSERVED"
)

type PromotionScope string

const (
	CoreRetrieval    PromotionScope = "core_retrieval"
	ReleaseCandidate PromotionScope = "release_candidate"
)

type PromotionStatus string

const (
	PromotionEvidenceReady PromotionStatus = "PROMOTION_EVIDENCE_READY"
	NotPromotionReady      PromotionStatus = "NOT_PROMOTION_READY"
)

type SourceSpan struct {
	Path            string `json:"path"`
	ContentSHA256   string `json:"content_sha256"`
	QualifiedSymbol string `json:"qualified_symbol"`
	StartByte       int    `json:"start_byte"`
	EndByte         int    `json:"end_byte"`
}
type RequiredConstraints struct {
	Identifiers []string   `json:"identifiers"`
	Paths       []string   `json:"paths"`
	Languages   []Language `json:"languages"`
	Scopes      []string   `json:"scopes"`
}
type ExpectedAlternative struct {
	Spans []SourceSpan `json:"spans"`
}
type RequiredGroup struct {
	ID           string                `json:"id"`
	Alternatives []ExpectedAlternative `json:"alternatives"`
}
type HardNegative struct {
	Span   SourceSpan `json:"span"`
	Reason string     `json:"reason"`
}
type ReviewState string

const (
	ReviewDraft  ReviewState = "draft"
	ReviewFrozen ReviewState = "frozen"
)

type ReviewRecord struct {
	State                ReviewState  `json:"state"`
	Passes               []ReviewPass `json:"passes"`
	Rationale            string       `json:"rationale"`
	SoloReviewLimitation string       `json:"solo_review_limitation,omitempty"`
}
type ReviewPass struct {
	ID       string `json:"id"`
	Reviewer string `json:"reviewer"`
}
type RelevanceJudgment struct {
	Span      SourceSpan `json:"span"`
	Grade     Relevance  `json:"grade"`
	Rationale string     `json:"rationale"`
}
type AssistantTaskRequirements struct {
	Requirements         []string `json:"requirements"`
	ExpectedTestOutcomes []string `json:"expected_test_outcomes"`
}
type EvaluationCase struct {
	SchemaVersion       int                        `json:"schema_version"`
	ID                  string                     `json:"id"`
	Text                string                     `json:"text"`
	Language            Language                   `json:"language"`
	Cohorts             []string                   `json:"cohorts"`
	AnswerMode          AnswerMode                 `json:"answer_mode"`
	ExpectedCardinality *int                       `json:"expected_cardinality,omitempty"`
	Split               DatasetSplit               `json:"split"`
	RequiredConstraints RequiredConstraints        `json:"required_constraints"`
	RequiredGroups      []RequiredGroup            `json:"required_groups"`
	HardNegatives       []HardNegative             `json:"hard_negatives"`
	Judgments           []RelevanceJudgment        `json:"judgments"`
	Review              ReviewRecord               `json:"review"`
	AssistantTask       *AssistantTaskRequirements `json:"assistant_task_requirements,omitempty"`
	Digest              string                     `json:"digest"`
}

type FailureStage Stage
type DenominatorRecord struct {
	Name      string `json:"name"`
	TruthUnit string `json:"truth_unit"`
	Count     int    `json:"count"`
}
type GroupObservation struct {
	GroupID   string    `json:"group_id"`
	Present   bool      `json:"present"`
	FirstLoss FirstLoss `json:"first_loss"`
}
type StageObservation struct {
	Stage             Stage               `json:"stage"`
	Required          bool                `json:"required"`
	Status            ObservationStatus   `json:"status"`
	Denominators      []DenominatorRecord `json:"denominators"`
	GroupObservations []GroupObservation  `json:"group_observations"`
	FailureStage      FailureStage        `json:"failure_stage,omitempty"`
	CandidateCount    int                 `json:"candidate_count"`
	QueryVectorSHA256 string              `json:"query_vector_sha256,omitempty"`
}
type StageTrace struct {
	SchemaVersion    int                `json:"schema_version"`
	QueryID          string             `json:"query_id"`
	RequiredGroupIDs []string           `json:"required_group_ids"`
	Observations     []StageObservation `json:"observations"`
	TerminalState    TerminalState      `json:"terminal_state"`
}

type EvaluationRunManifest struct {
	SchemaVersion        int               `json:"schema_version"`
	CorpusManifestSHA256 string            `json:"corpus_manifest_sha256"`
	QueryManifestSHA256  string            `json:"query_manifest_sha256"`
	CodeCommit           string            `json:"code_commit"`
	ProfileFingerprint   string            `json:"profile_fingerprint"`
	Generation           int64             `json:"generation"`
	CandidatePolicy      string            `json:"candidate_policy"`
	Platform             string            `json:"platform"`
	PairedControls       PairedRunControls `json:"paired_controls"`
}
type PairedRunControls struct {
	CorpusStateSHA256 string `json:"corpus_state_sha256"`
	LabelDigestSHA256 string `json:"label_digest_sha256"`
	ParserVersion     string `json:"parser_version"`
	ChunkerVersion    string `json:"chunker_version"`
	FTSSchemaVersion  string `json:"fts_schema_version"`
	SourceModel       string `json:"source_model"`
	SourceDimensions  int    `json:"source_dimensions"`
	ReducerID         string `json:"reducer_id"`
	ServingDimensions int    `json:"serving_dimensions"`
	CandidatePolicy   string `json:"candidate_policy"`
	BodyBudget        string `json:"body_budget"`
	MCPVersion        string `json:"mcp_version"`
}
type ArtifactEntry struct {
	Path      string `json:"path"`
	MediaType string `json:"media_type"`
	ByteSize  int64  `json:"byte_size"`
	SHA256    string `json:"sha256"`
}
type ArtifactManifest struct {
	SchemaVersion int             `json:"schema_version"`
	Entries       []ArtifactEntry `json:"entries"`
	Complete      bool            `json:"complete"`
}
type PromotionContract struct {
	SchemaVersion             int               `json:"schema_version"`
	Scope                     PromotionScope    `json:"scope"`
	CalibrationEvidenceSHA256 []string          `json:"calibration_evidence_sha256"`
	FrozenGates               []string          `json:"frozen_gates"`
	ConfirmationDatasetSHA256 string            `json:"confirmation_dataset_sha256"`
	PairedControls            PairedRunControls `json:"paired_controls"`
}
type PromotionResult struct {
	SchemaVersion      int             `json:"schema_version"`
	Scope              PromotionScope  `json:"scope"`
	Status             PromotionStatus `json:"status"`
	PrerequisiteSHA256 []string        `json:"prerequisite_sha256"`
	PassedGates        []string        `json:"passed_gates"`
	FailedGates        []string        `json:"failed_gates"`
	IncompleteReason   string          `json:"incomplete_reason,omitempty"`
	ApplicableGates    []string        `json:"applicable_gates"`
}
