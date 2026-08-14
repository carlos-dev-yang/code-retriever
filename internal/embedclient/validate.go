package embedclient

import (
	"fmt"
)

// ValidateResponse rejects a complete batch before storage and restores the
// provider's indexed data to request order.
func ValidateResponse(request EmbeddingRequest, response EmbeddingResponse) ([][]float32, error) {
	if err := request.Source.Validate(); err != nil {
		return nil, err
	}
	if request.Role != DocumentRole && request.Role != QueryRole {
		return nil, fmt.Errorf("unsupported embedding input role")
	}
	if request.Role == DocumentRole && request.Source.DocumentInputType != string(DocumentRole) || request.Role == QueryRole && request.Source.QueryInputType != string(QueryRole) {
		return nil, fmt.Errorf("source role mapping mismatch")
	}
	if response.Model != request.Source.Model {
		return nil, fmt.Errorf("embedding response model mismatch")
	}
	if response.TotalTokens < 0 {
		return nil, fmt.Errorf("embedding response total tokens invalid")
	}
	if len(response.Data) != len(request.Inputs) {
		return nil, fmt.Errorf("embedding response count mismatch")
	}
	ordered := make([][]float32, len(request.Inputs))
	for _, datum := range response.Data {
		if !datum.IndexPresent || datum.Index < 0 || datum.Index >= len(ordered) || ordered[datum.Index] != nil {
			return nil, fmt.Errorf("embedding response index invalid")
		}
		if err := ValidateSourceVector(request.Source, datum.Values); err != nil {
			return nil, fmt.Errorf("embedding response vector %d: %w", datum.Index, err)
		}
		ordered[datum.Index] = append([]float32(nil), datum.Values...)
	}
	return ordered, nil
}
