package search

import (
	"context"
	"fmt"
	"time"
	"unicode/utf8"

	"cidx/internal/config"
	"cidx/internal/embedclient"
	"cidx/internal/vector"
)

// QueryEmbeddingProviderError marks a failure returned by the injected
// embedding client itself. Response validation and local transformation
// failures deliberately retain their original error type: they are internal
// invariants, not a provider observation that an evaluation may reduce to a
// denominator failure.
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
	formatted, err := formatQueryText(resolved.Search.QueryTextFormatVersion, query)
	if err != nil {
		return nil, err
	}
	request := embedclient.EmbeddingRequest{Source: resolved.Embedding.EmbeddingSourceSpec(), Role: embedclient.QueryRole, Inputs: []string{formatted}}
	requestCtx, cancel := context.WithTimeout(ctx, time.Duration(resolved.Embedding.Request.TimeoutSeconds)*time.Second)
	defer cancel()
	response, err := client.Embed(requestCtx, request)
	if err != nil {
		return nil, QueryEmbeddingProviderError{Err: err}
	}
	ordered, err := embedclient.ValidateResponse(request, response)
	if err != nil {
		return nil, err
	}
	if len(ordered) != 1 {
		return nil, fmt.Errorf("query embedding response count invalid")
	}
	return (vector.Transformer{Spec: resolved.Embedding.TransformSpec()}).Transform(ordered[0])
}
