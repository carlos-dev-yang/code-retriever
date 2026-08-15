package embedclient

import (
	"context"
	"errors"
	"time"
)

// InputRole is deliberately explicit so document and query semantics cannot
// be accidentally interchanged. Only document calls are used by the lab.
type InputRole string

const (
	DocumentRole InputRole = "document"
	QueryRole    InputRole = "query"
)

type EmbeddingRequest struct {
	Source EmbeddingSourceSpec
	Role   InputRole
	Inputs []string
}
type EmbeddingResponse struct {
	Model, RequestID string
	Data             []EmbeddingDatum
	TotalTokens      int
}
type EmbeddingDatum struct {
	Index        int
	IndexPresent bool
	Values       []float32
}
type EmbeddingClient interface {
	Embed(context.Context, EmbeddingRequest) (EmbeddingResponse, error)
}
type ProviderError struct {
	// Class is code-owned and safe to retain in diagnostics. It never includes
	// provider response bodies, request text, or authorization data.
	Class      string
	StatusCode int
	Retryable  bool
	RetryAfter time.Duration
	Cause      error
}

func (e ProviderError) Error() string { return e.Class }
func (e ProviderError) Unwrap() error { return e.Cause }
func IsRetryable(err error) bool {
	provider, ok := providerError(err)
	return ok && provider.Retryable
}

// RetryAfter returns only a parsed, positive provider hint. Scheduling stays
// in the shared executor so adapters make exactly one HTTP call.
func RetryAfter(err error) (time.Duration, bool) {
	provider, ok := providerError(err)
	if !ok || provider.RetryAfter <= 0 {
		return 0, false
	}
	return provider.RetryAfter, true
}

func providerError(err error) (ProviderError, bool) {
	var value ProviderError
	if errors.As(err, &value) {
		return value, true
	}
	var pointer *ProviderError
	if errors.As(err, &pointer) && pointer != nil {
		return *pointer, true
	}
	return ProviderError{}, false
}
