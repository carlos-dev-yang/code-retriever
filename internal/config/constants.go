package config

const (
	SchemaVersion                     = 1
	CanonicalTextFormatterID          = "cidx-canonical-text"
	CanonicalTextFormatterVer         = 1
	IndexChunkerVersion               = 3
	IndexProjectionVersion            = 1
	IndexSegmentVersion               = 1
	SymbolNormalizerID                = "identifier-split-lower-v1"
	FTSSchemaVersion                  = 1
	FTSTokenizerID                    = "unicode61-v1"
	DefaultServingDimensions          = 1024
	StorageCodecInt8                  = "int8"
	DefaultStorageCodec               = StorageCodecInt8
	DefaultSearchMode                 = "fts"
	DefaultReturnK                    = 5
	AbsoluteMaxReturnK                = 20
	DefaultCandidateK                 = 20
	DefaultRRFK                       = 60
	DefaultFTSSymbolWeight            = 5.0
	DefaultFTSBodyWeight              = 1.0
	DefaultMaxSourceFileBytes         = 1 << 20
	AbsoluteMaxSourceFileBytes        = 1 << 20
	DefaultTargetSegmentBytes         = 1024
	DefaultHardMaxInlineBytes         = 64 << 10
	AbsoluteMaxInlineBytes            = 1 << 20
	DefaultRequestMaxInputs           = 128
	AbsoluteRequestMaxInputs          = 128
	DefaultRequestMaxTotalInputBytes  = 256 << 10
	AbsoluteRequestMaxTotalInputBytes = 256 << 10
	DefaultRequestMaxConcurrency      = 4
	AbsoluteRequestMaxConcurrency     = 4
	DefaultRequestTimeoutSeconds      = 30
	AbsoluteRequestTimeoutSeconds     = 30
	DefaultRetryMaxRetries            = 3
	AbsoluteRetryMaxRetries           = 3
	DefaultMaxQueryBytes              = 8 << 10
	DefaultMaxQueryTokens             = 64
	DefaultMaxQueryTokenRunes         = 128
	// LexicalQueryPlannerVersion identifies the deterministic natural-language
	// lexical planning policy. It is runtime-only serving policy and never
	// invalidates the local index or vectors.
	LexicalQueryPlannerVersion = 2
	// QueryTextFormatVersion is code-owned runtime policy. It identifies the
	// deterministic bytes sent to the query embedding provider.
	QueryTextFormatVersion = 1

	// Query limits are runtime policy. The defaults are intentionally modest,
	// while these ceilings leave room for local use without permitting an
	// unbounded FTS expression or diagnostic payload.
	AbsoluteMaxQueryBytes      = 1 << 20
	AbsoluteMaxQueryTokens     = 4096
	AbsoluteMaxQueryTokenRunes = 4096
)

func defaultRetryWaitSchedule() []int { return []int{10, 20, 30} }

const (
	CanonicalInputDomain = "cidx/input/v1"
	IndexProfileDomain   = "cidx/index-profile/v1"
	CanonicalTextDomain  = "cidx/canonical-text-profile/v1"
	SourceProfileDomain  = "cidx/embedding-source-profile/v1"
	VectorSpaceDomain    = "cidx/vector-space-profile/v1"
	VectorStorageDomain  = "cidx/vector-storage-profile/v1"
	ServingPolicyDomain  = "cidx/serving-policy-profile/v1"
)
