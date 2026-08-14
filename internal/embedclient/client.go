package embedclient

import (
	"context"
	"errors"
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
	Class     string
	Retryable bool
}

func (e ProviderError) Error() string { return e.Class }
func IsRetryable(err error) bool {
	var provider ProviderError
	return errors.As(err, &provider) && provider.Retryable
}
