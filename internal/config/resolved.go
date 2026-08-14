package config

import (
	"fmt"
	"os"

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

// EmbeddingSourceSpec and TransformSpec are the only adapters from resolved
// configuration into provider/vector code. Those packages retain no runtime
// source-dimension or allowed-target registry.
func (value ResolvedEmbedding) EmbeddingSourceSpec() embedclient.EmbeddingSourceSpec {
	return embedclient.EmbeddingSourceSpec{Provider: value.Model.Provider, Model: value.Model.Model, SourceDimensions: value.Model.SourceDimensions, OutputDType: value.Model.OutputDType, DocumentInputType: value.Model.DocumentInputType, QueryInputType: value.Model.QueryInputType, Truncation: value.Model.Truncation, AdapterVersion: value.Model.AdapterVersion, AllowDirectTargetCompare: embedclient.DirectTargetComparison}
}

func (value ResolvedEmbedding) TransformSpec() vector.TransformSpec {
	return vector.TransformSpec{SourceDimensions: value.Model.SourceDimensions, TargetDimensions: value.TargetDimensions, ReducerID: value.ReducerID, NormalizerID: value.NormalizerID, MetricID: value.Metric}
}

type ResolvedIndex struct {
	Languages                                               []chunk.Language
	MaxSourceFileBytes, MaxChunkBytes, MaxSegmentInputBytes int
}
type ResolvedEmbedding struct {
	Model                                         ModelSpec
	TargetDimensions                              int
	ReducerID, NormalizerID, Metric, StorageCodec string
	Batch                                         ResolvedBatch
}
type ResolvedBatch struct{ MaxInputs, MaxInputTokens, MaxRetries, RequestTimeoutMS int }
type ServingPolicy struct {
	DefaultMode                    string
	AllowPaidQueryEmbedding        bool
	ReturnK, CandidateK, RRFK      int
	FTSSymbolWeight, FTSBodyWeight float64
}
type ResolvedMCP struct{ HardMaxInlineBytes, MaxReadSpanLines int }
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
	if raw.Embedding.TargetDimensions == nil {
		return ResolvedConfig{}, fmt.Errorf("embedding.target_dimensions is required")
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
	resolved := ResolvedConfig{Version: raw.Version,
		Index:     ResolvedIndex{MaxSourceFileBytes: raw.Index.MaxSourceFileBytes, MaxChunkBytes: raw.Index.MaxChunkBytes, MaxSegmentInputBytes: raw.Index.MaxSegmentInputBytes},
		Embedding: ResolvedEmbedding{Model: model, TargetDimensions: *raw.Embedding.TargetDimensions, ReducerID: reducer, NormalizerID: normalizer, Metric: metric, StorageCodec: codec, Batch: ResolvedBatch{MaxInputs: raw.Embedding.Batch.MaxInputs, MaxInputTokens: raw.Embedding.Batch.MaxInputTokens, MaxRetries: raw.Embedding.Batch.MaxRetries, RequestTimeoutMS: raw.Embedding.Batch.RequestTimeoutMS}},
		Search:    ServingPolicy{DefaultMode: mode, AllowPaidQueryEmbedding: allowPaid, ReturnK: returnK, CandidateK: candidateK, RRFK: rrfK, FTSSymbolWeight: symbolWeight, FTSBodyWeight: bodyWeight},
		MCP:       ResolvedMCP{HardMaxInlineBytes: raw.MCP.HardMaxInlineBytes, MaxReadSpanLines: raw.MCP.MaxReadSpanLines},
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
