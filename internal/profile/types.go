// Package profile contains immutable semantic profile values. It does not read
// files or database rows.
package profile

import "cidx/internal/chunk"

type Fingerprint string

type IndexProfile struct {
	Languages          []chunk.Language `json:"languages"`
	ChunkerVersion     int              `json:"chunker_version"`
	ProjectionVersion  int              `json:"projection_version"`
	SegmentVersion     int              `json:"segment_version"`
	SymbolNormalizerID string           `json:"symbol_normalizer_id"`
	FTSSchemaVersion   int              `json:"fts_schema_version"`
	FTSTokenizerID     string           `json:"fts_tokenizer_id"`
	MaxSourceFileBytes int              `json:"max_source_file_bytes"`
	TargetSegmentBytes int              `json:"target_segment_bytes"`
}

type CanonicalTextProfile struct {
	FormatterID      string   `json:"formatter_id"`
	FormatterVersion int      `json:"formatter_version"`
	ProjectionOrder  []string `json:"projection_order"`
}

type InputTypeMapping struct {
	Document string `json:"document"`
	Query    string `json:"query"`
}

type EmbeddingSourceProfile struct {
	Provider         string           `json:"provider"`
	Model            string           `json:"model"`
	SourceDimensions int              `json:"source_dimensions"`
	OutputDType      string           `json:"output_dtype"`
	InputTypeMapping InputTypeMapping `json:"input_type_mapping"`
	Truncation       bool             `json:"truncation"`
	AdapterVersion   int              `json:"adapter_version"`
}

type VectorSpaceProfile struct {
	SourceProfileFingerprint Fingerprint `json:"source_profile_fingerprint"`
	ServingDimensions        int         `json:"serving_dimensions"`
	ReducerID                string      `json:"reducer_id"`
	NormalizerID             string      `json:"normalizer_id"`
	Metric                   string      `json:"metric"`
}

type VectorStorageProfile struct {
	VectorSpaceProfileFingerprint Fingerprint `json:"vector_space_profile_fingerprint"`
	StorageCodecID                string      `json:"storage_codec_id"`
}

type ServingVectorProfile struct {
	EmbeddingSource EmbeddingSourceProfile
	VectorSpace     VectorSpaceProfile
	VectorStorage   VectorStorageProfile
	Fingerprint     Fingerprint
}

// ServingPolicyProfile is runtime-only policy. It deliberately is not an
// index or vector identity and therefore never causes materialization.
type ServingPolicyProfile struct {
	DefaultMode             string  `json:"default_mode"`
	AllowPaidQueryEmbedding bool    `json:"allow_paid_query_embedding"`
	ReturnK                 int     `json:"return_k"`
	CandidateK              int     `json:"candidate_k"`
	RRFK                    int     `json:"rrf_k"`
	QueryTextFormatVersion  int     `json:"query_text_format_version"`
	MaxQueryBytes           int     `json:"max_query_bytes"`
	MaxQueryTokens          int     `json:"max_query_tokens"`
	MaxQueryTokenRunes      int     `json:"max_query_token_runes"`
	FTSSymbolWeight         float64 `json:"fts_symbol_weight"`
	FTSBodyWeight           float64 `json:"fts_body_weight"`
	HardMaxInlineBytes      int     `json:"hard_max_inline_bytes"`
}

type ProfileFingerprints struct {
	Index         Fingerprint `json:"index"`
	CanonicalText Fingerprint `json:"canonical_text"`
	Source        Fingerprint `json:"source"`
	VectorSpace   Fingerprint `json:"vector_space"`
	VectorStorage Fingerprint `json:"vector_storage"`
	Policy        Fingerprint `json:"policy"`
}
