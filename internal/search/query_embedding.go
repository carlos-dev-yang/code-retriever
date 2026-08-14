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
	requestCtx, cancel := context.WithTimeout(ctx, time.Duration(resolved.Embedding.Batch.RequestTimeoutMS)*time.Millisecond)
	defer cancel()
	response, err := client.Embed(requestCtx, request)
	if err != nil {
		return nil, err
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
