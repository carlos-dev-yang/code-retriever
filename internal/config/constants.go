package config

const (
	SchemaVersion             = 1
	CanonicalTextFormatterID  = "cidx-canonical-text"
	CanonicalTextFormatterVer = 1
	IndexChunkerVersion       = 1
	IndexProjectionVersion    = 1
	IndexSegmentVersion       = 1
	SymbolNormalizerID        = "identifier-split-lower-v1"
	FTSSchemaVersion          = 1
	FTSTokenizerID            = "unicode61-v1"
	DefaultStorageCodec       = "binary"
	DefaultSearchMode         = "fts"
	DefaultReturnK            = 5
	DefaultCandidateK         = 20
	DefaultRRFK               = 60
	DefaultFTSSymbolWeight    = 5.0
	DefaultFTSBodyWeight      = 1.0
	DefaultMaxQueryBytes      = 8 << 10
	DefaultMaxQueryTokens     = 64
	DefaultMaxQueryTokenRunes = 128

	// Query limits are runtime policy. The defaults are intentionally modest,
	// while these ceilings leave room for local use without permitting an
	// unbounded FTS expression or diagnostic payload.
	AbsoluteMaxQueryBytes      = 1 << 20
	AbsoluteMaxQueryTokens     = 4096
	AbsoluteMaxQueryTokenRunes = 4096
)

const (
	CanonicalInputDomain = "cidx/input/v1"
	IndexProfileDomain   = "cidx/index-profile/v1"
	CanonicalTextDomain  = "cidx/canonical-text-profile/v1"
	SourceProfileDomain  = "cidx/embedding-source-profile/v1"
	VectorSpaceDomain    = "cidx/vector-space-profile/v1"
	VectorStorageDomain  = "cidx/vector-storage-profile/v1"
	ServingPolicyDomain  = "cidx/serving-policy-profile/v1"
)
