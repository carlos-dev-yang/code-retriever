package config

import (
	"fmt"
	"math"

	"cidx/internal/vector"
)

func Validate(resolved *ResolvedConfig) error {
	if len(resolved.Index.Languages) == 0 || resolved.Index.MaxSourceFileBytes <= 0 || resolved.Index.MaxChunkBytes <= 0 || resolved.Index.MaxSegmentInputBytes <= 0 || resolved.Index.MaxChunkBytes > resolved.Index.MaxSourceFileBytes || resolved.Index.MaxSegmentInputBytes > resolved.Index.MaxChunkBytes {
		return fmt.Errorf("invalid index limits")
	}
	seen := map[string]struct{}{}
	for _, language := range resolved.Index.Languages {
		if !language.Valid() {
			return fmt.Errorf("unsupported index language %q", language)
		}
		if _, exists := seen[string(language)]; exists {
			return fmt.Errorf("duplicate index language %q", language)
		}
		seen[string(language)] = struct{}{}
	}
	if !resolved.Embedding.Model.SupportsTarget(resolved.Embedding.TargetDimensions) || resolved.Embedding.ReducerID != vector.ReducerID || resolved.Embedding.NormalizerID != vector.NormalizerID || resolved.Embedding.Metric != vector.MetricID {
		return fmt.Errorf("unsupported embedding transform")
	}
	if resolved.Embedding.StorageCodec != StorageCodecBinary && resolved.Embedding.StorageCodec != StorageCodecInt8 {
		return fmt.Errorf("unsupported storage codec %q", resolved.Embedding.StorageCodec)
	}
	if resolved.Embedding.Batch.MaxInputs <= 0 || resolved.Embedding.Batch.MaxInputTokens <= 0 || resolved.Embedding.Batch.MaxRetries < 0 || resolved.Embedding.Batch.RequestTimeoutMS <= 0 {
		return fmt.Errorf("invalid embedding batch policy")
	}
	if resolved.Search.DefaultMode != "fts" && resolved.Search.DefaultMode != "hybrid" {
		return fmt.Errorf("unsupported search mode")
	}
	if resolved.Search.DefaultMode == "hybrid" && !resolved.Search.AllowPaidQueryEmbedding {
		return fmt.Errorf("hybrid default requires paid query embedding permission")
	}
	if resolved.Search.ReturnK <= 0 || resolved.Search.CandidateK < resolved.Search.ReturnK || resolved.Search.RRFK <= 0 || resolved.Search.QueryTextFormatVersion != QueryTextFormatVersion || resolved.Search.QueryLimits.MaxBytes <= 0 || resolved.Search.QueryLimits.MaxBytes > AbsoluteMaxQueryBytes || resolved.Search.QueryLimits.MaxTokens <= 0 || resolved.Search.QueryLimits.MaxTokens > AbsoluteMaxQueryTokens || resolved.Search.QueryLimits.MaxTokenRunes <= 0 || resolved.Search.QueryLimits.MaxTokenRunes > AbsoluteMaxQueryTokenRunes || !finitePositive(resolved.Search.FTSSymbolWeight) || !finitePositive(resolved.Search.FTSBodyWeight) {
		return fmt.Errorf("invalid search policy")
	}
	if resolved.MCP.HardMaxInlineBytes <= 0 || resolved.MCP.MaxReadSpanLines <= 0 {
		return fmt.Errorf("invalid MCP limits")
	}
	return nil
}

func finitePositive(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

const (
	StorageCodecBinary = "binary"
	StorageCodecInt8   = "int8"
)
