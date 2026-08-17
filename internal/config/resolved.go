package config

import (
	"fmt"
	"os"
	"reflect"

	"cidx/internal/chunk"
	"cidx/internal/embedclient"
	"cidx/internal/profile"
	"cidx/internal/vector"
)

type ResolvedConfig struct {
	Version   int
	Index     ResolvedIndex
	Embedding ResolvedEmbedding
	Search    ServingPolicy
	MCP       ResolvedMCP
	Profiles  DesiredProfiles
}

// ValidateIntegrity detects accidental mutation after resolution. Resolved
// config is an immutable injection contract, but Go callers can still alter
// exported fields; profile fingerprints must never be trusted after that.
func (value ResolvedConfig) ValidateIntegrity() error {
	copy := value
	if err := Validate(&copy); err != nil {
		return err
	}
	expected, err := FingerprintProfiles(copy)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(expected, value.Profiles) {
		return fmt.Errorf("resolved config profiles do not match resolved values")
	}
	return nil
}

// EmbeddingSourceSpec and TransformSpec are the only adapters from resolved
// configuration into provider/vector code. Those packages retain no runtime
// source-dimension or allowed-target registry.
func (value ResolvedEmbedding) EmbeddingSourceSpec() embedclient.EmbeddingSourceSpec {
	return embedclient.EmbeddingSourceSpec{Provider: value.Model.Provider, Model: value.Model.Model, SourceDimensions: value.Model.SourceDimensions, OutputDType: value.Model.OutputDType, DocumentInputType: value.Model.DocumentInputType, QueryInputType: value.Model.QueryInputType, Truncation: value.Model.Truncation, AdapterVersion: value.Model.AdapterVersion, AllowDirectTargetCompare: embedclient.DirectTargetComparison}
}

func (value ResolvedEmbedding) TransformSpec() vector.TransformSpec {
	return vector.TransformSpec{SourceDimensions: value.Model.SourceDimensions, TargetDimensions: value.ServingDimensions, ReducerID: value.ReducerID, NormalizerID: value.NormalizerID, MetricID: value.Metric}
}

type ResolvedIndex struct {
	Languages                              []chunk.Language
	MaxSourceFileBytes, TargetSegmentBytes int
}
type ResolvedEmbedding struct {
	Model                                         ModelSpec
	ServingDimensions                             int
	ReducerID, NormalizerID, Metric, StorageCodec string
	Request                                       ResolvedRequest
	Retry                                         ResolvedRetry
}
type ResolvedRequest struct{ MaxInputs, MaxTotalInputBytes, MaxConcurrency, TimeoutSeconds int }
type ResolvedRetry struct {
	MaxRetries  int
	WaitSeconds []int
}
type ServingPolicy struct {
	DefaultMode                    string
	AllowPaidQueryEmbedding        bool
	ReturnK, CandidateK, RRFK      int
	QueryTextFormatVersion         int
	QueryLimits                    QueryLimits
	FTSSymbolWeight, FTSBodyWeight float64
}
type QueryLimits struct{ MaxBytes, MaxTokens, MaxTokenRunes int }
type ResolvedMCP struct{ HardMaxInlineBytes int }
type DesiredProfiles struct {
	Index         profile.IndexProfile
	CanonicalText profile.CanonicalTextProfile
	Source        profile.EmbeddingSourceProfile
	VectorSpace   profile.VectorSpaceProfile
	VectorStorage profile.VectorStorageProfile
	Fingerprints  profile.ProfileFingerprints
}

func Load(path string) (ResolvedConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ResolvedConfig{}, err
	}
	return LoadBytes(data)
}

func LoadBytes(data []byte) (ResolvedConfig, error) {
	raw, err := DecodeRaw(data)
	if err != nil {
		return ResolvedConfig{}, err
	}
	return Resolve(raw)
}

// LoadBytes makes testing and startup decoding share one strict path. Load is
// implemented in validate.go to keep file I/O inside this package.
func Resolve(raw RawConfig) (ResolvedConfig, error) {
	if raw.Version != SchemaVersion {
		return ResolvedConfig{}, fmt.Errorf("unsupported config version %d", raw.Version)
	}
	model, err := ResolveModel(raw.Embedding.Model)
	if err != nil {
		return ResolvedConfig{}, err
	}
	servingDimensions := DefaultServingDimensions
	if raw.Embedding.ServingDimensions != nil {
		servingDimensions = *raw.Embedding.ServingDimensions
	}
	codec := DefaultStorageCodec
	if raw.Embedding.StorageCodec != nil {
		codec = *raw.Embedding.StorageCodec
	}
	reducer := raw.Embedding.Reducer
	if reducer == "" {
		reducer = vector.ReducerID
	}
	normalizer := raw.Embedding.Normalizer
	if normalizer == "" {
		normalizer = vector.NormalizerID
	}
	metric := raw.Embedding.Metric
	if metric == "" {
		metric = vector.MetricID
	}
	mode := DefaultSearchMode
	if raw.Search.DefaultMode != nil {
		mode = *raw.Search.DefaultMode
	}
	returnK := DefaultReturnK
	if raw.Search.ReturnK != nil {
		returnK = *raw.Search.ReturnK
	}
	candidateK := DefaultCandidateK
	if raw.Search.CandidateK != nil {
		candidateK = *raw.Search.CandidateK
	}
	rrfK := DefaultRRFK
	if raw.Search.RRFK != nil {
		rrfK = *raw.Search.RRFK
	}
	maxQueryBytes := DefaultMaxQueryBytes
	if raw.Search.MaxQueryBytes != nil {
		maxQueryBytes = *raw.Search.MaxQueryBytes
	}
	maxQueryTokens := DefaultMaxQueryTokens
	if raw.Search.MaxQueryTokens != nil {
		maxQueryTokens = *raw.Search.MaxQueryTokens
	}
	maxQueryTokenRunes := DefaultMaxQueryTokenRunes
	if raw.Search.MaxQueryTokenRunes != nil {
		maxQueryTokenRunes = *raw.Search.MaxQueryTokenRunes
	}
	symbolWeight := DefaultFTSSymbolWeight
	if raw.Search.FTSWeights.Symbols != nil {
		symbolWeight = *raw.Search.FTSWeights.Symbols
	}
	bodyWeight := DefaultFTSBodyWeight
	if raw.Search.FTSWeights.Body != nil {
		bodyWeight = *raw.Search.FTSWeights.Body
	}
	allowPaid := false
	if raw.Search.AllowPaidQueryEmbedding != nil {
		allowPaid = *raw.Search.AllowPaidQueryEmbedding
	}
	maxSourceFileBytes := defaultInt(raw.Index.MaxSourceFileBytes, DefaultMaxSourceFileBytes)
	targetSegmentBytes := defaultInt(raw.Index.TargetSegmentBytes, DefaultTargetSegmentBytes)
	request := ResolvedRequest{
		MaxInputs:          defaultInt(raw.Embedding.Request.MaxInputs, DefaultRequestMaxInputs),
		MaxTotalInputBytes: defaultInt(raw.Embedding.Request.MaxTotalInputBytes, DefaultRequestMaxTotalInputBytes),
		MaxConcurrency:     defaultInt(raw.Embedding.Request.MaxConcurrency, DefaultRequestMaxConcurrency),
		TimeoutSeconds:     defaultInt(raw.Embedding.Request.TimeoutSeconds, DefaultRequestTimeoutSeconds),
	}
	retry := ResolvedRetry{MaxRetries: DefaultRetryMaxRetries}
	if raw.Embedding.Retry.MaxRetries != nil {
		retry.MaxRetries = *raw.Embedding.Retry.MaxRetries
	}
	if retry.MaxRetries < 0 || retry.MaxRetries > AbsoluteRetryMaxRetries {
		return ResolvedConfig{}, fmt.Errorf("invalid embedding retry policy")
	}
	if raw.Embedding.Retry.WaitSeconds == nil {
		waits := defaultRetryWaitSchedule()
		retry.WaitSeconds = append([]int(nil), waits[:retry.MaxRetries]...)
	} else {
		if len(*raw.Embedding.Retry.WaitSeconds) == 0 {
			return ResolvedConfig{}, fmt.Errorf("embedding.retry.wait_seconds must be non-empty when specified")
		}
		retry.WaitSeconds = append([]int(nil), (*raw.Embedding.Retry.WaitSeconds)...)
	}
	hardMaxInlineBytes := defaultInt(raw.MCP.HardMaxInlineBytes, DefaultHardMaxInlineBytes)
	resolved := ResolvedConfig{Version: raw.Version,
		Index:     ResolvedIndex{MaxSourceFileBytes: maxSourceFileBytes, TargetSegmentBytes: targetSegmentBytes},
		Embedding: ResolvedEmbedding{Model: model, ServingDimensions: servingDimensions, ReducerID: reducer, NormalizerID: normalizer, Metric: metric, StorageCodec: codec, Request: request, Retry: retry},
		Search:    ServingPolicy{DefaultMode: mode, AllowPaidQueryEmbedding: allowPaid, ReturnK: returnK, CandidateK: candidateK, RRFK: rrfK, QueryTextFormatVersion: QueryTextFormatVersion, QueryLimits: QueryLimits{MaxBytes: maxQueryBytes, MaxTokens: maxQueryTokens, MaxTokenRunes: maxQueryTokenRunes}, FTSSymbolWeight: symbolWeight, FTSBodyWeight: bodyWeight},
		MCP:       ResolvedMCP{HardMaxInlineBytes: hardMaxInlineBytes},
	}
	for _, language := range raw.Index.Languages {
		resolved.Index.Languages = append(resolved.Index.Languages, chunk.Language(language))
	}
	if err := Validate(&resolved); err != nil {
		return ResolvedConfig{}, err
	}
	profiles, err := FingerprintProfiles(resolved)
	if err != nil {
		return ResolvedConfig{}, err
	}
	resolved.Profiles = profiles
	return resolved, nil
}

func defaultInt(value, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}
