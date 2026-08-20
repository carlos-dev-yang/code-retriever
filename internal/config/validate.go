package config

import (
	"fmt"
	"math"

	"cidx/internal/vector"
)

func Validate(resolved *ResolvedConfig) error {
	if len(resolved.Index.Languages) == 0 || resolved.Index.MaxSourceFileBytes <= 0 || resolved.Index.MaxSourceFileBytes > AbsoluteMaxSourceFileBytes || resolved.Index.TargetSegmentBytes <= 0 || resolved.Index.TargetSegmentBytes > resolved.Index.MaxSourceFileBytes {
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
	if !resolved.Embedding.Model.SupportsServingDimensions(resolved.Embedding.ServingDimensions) || resolved.Embedding.ReducerID != vector.ReducerID || resolved.Embedding.NormalizerID != vector.NormalizerID || resolved.Embedding.Metric != vector.MetricID {
		return fmt.Errorf("unsupported embedding transform")
	}
	if resolved.Embedding.StorageCodec != StorageCodecInt8 {
		return fmt.Errorf("unsupported storage codec %q", resolved.Embedding.StorageCodec)
	}
	if resolved.Embedding.Request.MaxInputs <= 0 || resolved.Embedding.Request.MaxInputs > AbsoluteRequestMaxInputs || resolved.Embedding.Request.MaxTotalInputBytes <= 0 || resolved.Embedding.Request.MaxTotalInputBytes > AbsoluteRequestMaxTotalInputBytes || resolved.Embedding.Request.MaxConcurrency <= 0 || resolved.Embedding.Request.MaxConcurrency > AbsoluteRequestMaxConcurrency || resolved.Embedding.Request.TimeoutSeconds <= 0 || resolved.Embedding.Request.TimeoutSeconds > AbsoluteRequestTimeoutSeconds {
		return fmt.Errorf("invalid embedding request policy")
	}
	if resolved.Embedding.Retry.MaxRetries < 0 || resolved.Embedding.Retry.MaxRetries > AbsoluteRetryMaxRetries || !validRetryWaits(resolved.Embedding.Retry) {
		return fmt.Errorf("invalid embedding retry policy")
	}
	if resolved.Search.DefaultMode != "fts" && resolved.Search.DefaultMode != "hybrid" {
		return fmt.Errorf("unsupported search mode")
	}
	if resolved.Search.DefaultMode == "hybrid" && !resolved.Search.AllowPaidQueryEmbedding {
		return fmt.Errorf("hybrid default requires paid query embedding permission")
	}
	if resolved.Search.ReturnK <= 0 || resolved.Search.ReturnK > AbsoluteMaxReturnK || resolved.Search.CandidateK < resolved.Search.ReturnK || resolved.Search.RRFK <= 0 || resolved.Search.LexicalQueryPlannerVersion != LexicalQueryPlannerVersion || resolved.Search.QueryTextFormatVersion != QueryTextFormatVersion || resolved.Search.QueryLimits.MaxBytes <= 0 || resolved.Search.QueryLimits.MaxBytes > AbsoluteMaxQueryBytes || resolved.Search.QueryLimits.MaxTokens <= 0 || resolved.Search.QueryLimits.MaxTokens > AbsoluteMaxQueryTokens || resolved.Search.QueryLimits.MaxTokenRunes <= 0 || resolved.Search.QueryLimits.MaxTokenRunes > AbsoluteMaxQueryTokenRunes || !finitePositive(resolved.Search.FTSSymbolWeight) || !finitePositive(resolved.Search.FTSBodyWeight) {
		return fmt.Errorf("invalid search policy")
	}
	if resolved.MCP.HardMaxInlineBytes <= 0 || resolved.MCP.HardMaxInlineBytes > AbsoluteMaxInlineBytes {
		return fmt.Errorf("invalid MCP limits")
	}
	return nil
}

func validRetryWaits(retry ResolvedRetry) bool {
	if retry.MaxRetries != len(retry.WaitSeconds) {
		return false
	}
	expected := defaultRetryWaitSchedule()
	for index, wait := range retry.WaitSeconds {
		if wait != expected[index] {
			return false
		}
	}
	return true
}

func finitePositive(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}
