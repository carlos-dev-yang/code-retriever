package search

import (
	"context"
	"fmt"
	"time"
	"unicode/utf8"

	"cidx/internal/config"
	"cidx/internal/embed"
	"cidx/internal/embedclient"
	"cidx/internal/vector"
)

// QueryEmbeddingProviderError marks an exhausted provider attempt failure.
// Response validation and local transformation failures deliberately retain
// their original error type: they are internal invariants, not a provider
// observation that an evaluation may reduce to a denominator failure.
type QueryEmbeddingProviderError struct{ Err error }

func (value QueryEmbeddingProviderError) Error() string { return value.Err.Error() }
func (value QueryEmbeddingProviderError) Unwrap() error { return value.Err }

// queryEmbedding creates a fresh, request-local query vector. Neither its
// input nor either f32 representation is passed to storage or logging.
func formatQueryText(version int, query string) (string, error) {
	if version != config.QueryTextFormatVersion || !utf8.ValidString(query) || query == "" {
		return "", fmt.Errorf("invalid query text format input")
	}
	// v1 preserves validated UTF-8 bytes exactly. Keeping this explicit makes a
	// future format change a serving-policy fingerprint change rather than an
	// invisible provider-input change.
	return query, nil
}

func queryEmbedding(ctx context.Context, client embedclient.EmbeddingClient, resolved config.ResolvedConfig, query string) ([]float32, error) {
	return queryEmbeddingWithWait(ctx, client, resolved, query, nil)
}

// queryEmbeddingWithWait keeps the executor's cancellation-aware waiter
// injectable only for focused core tests. Production uses its default waiter.
func queryEmbeddingWithWait(ctx context.Context, client embedclient.EmbeddingClient, resolved config.ResolvedConfig, query string, wait embed.Waiter) ([]float32, error) {
	if err := resolved.ValidateIntegrity(); err != nil {
		return nil, err
	}
	formatted, err := formatQueryText(resolved.Search.QueryTextFormatVersion, query)
	if err != nil {
		return nil, err
	}
	inputs := []embed.RequestInput{{Ordinal: 0, Key: "query-0", Bytes: []byte(formatted)}}
	waits := make([]time.Duration, len(resolved.Embedding.Retry.WaitSeconds))
	for i, seconds := range resolved.Embedding.Retry.WaitSeconds {
		waits[i] = time.Duration(seconds) * time.Second
	}
	outcomes, err := embed.Execute(ctx, client, resolved.Embedding.EmbeddingSourceSpec(), embedclient.QueryRole, inputs, embed.ExecuteOptions{
		Limits:         embed.RequestLimits{MaxInputs: resolved.Embedding.Request.MaxInputs, MaxTotalBytes: resolved.Embedding.Request.MaxTotalInputBytes},
		MaxConcurrency: resolved.Embedding.Request.MaxConcurrency,
		AttemptTimeout: time.Duration(resolved.Embedding.Request.TimeoutSeconds) * time.Second,
		MaxRetries:     resolved.Embedding.Retry.MaxRetries,
		RetryWaits:     waits,
		Wait:           wait,
	}, nil)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	if err != nil {
		return nil, err
	}
	if len(outcomes) != 1 {
		return nil, fmt.Errorf("query embedding response count invalid")
	}
	outcome := outcomes[0]
	if outcome.Err != nil {
		if outcome.ResponseRejected {
			return nil, outcome.Err
		}
		return nil, QueryEmbeddingProviderError{Err: outcome.Err}
	}
	if len(outcome.Vectors) != 1 {
		return nil, fmt.Errorf("query embedding response count invalid")
	}
	return (vector.Transformer{Spec: resolved.Embedding.TransformSpec()}).Transform(outcome.Vectors[0])
}
