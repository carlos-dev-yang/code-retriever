package embedclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"unicode/utf8"
)

// VoyageClient has a code-owned endpoint. Credential acquisition stays at the
// application boundary; tests inject a transport and never use a real key.
type VoyageClient struct {
	HTTPClient *http.Client
	APIKey     string
}

func (c VoyageClient) Embed(ctx context.Context, request EmbeddingRequest) (EmbeddingResponse, error) {
	if err := request.Source.Validate(); err != nil {
		return EmbeddingResponse{}, err
	}
	if c.APIKey == "" {
		return EmbeddingResponse{}, fmt.Errorf("Voyage API key is required")
	}
	if request.Role != DocumentRole && request.Role != QueryRole {
		return EmbeddingResponse{}, fmt.Errorf("unsupported embedding input role")
	}
	if len(request.Inputs) == 0 {
		return EmbeddingResponse{}, fmt.Errorf("embedding inputs are required")
	}
	for _, input := range request.Inputs {
		if input == "" || !utf8.ValidString(input) {
			return EmbeddingResponse{}, fmt.Errorf("embedding input is invalid")
		}
	}
	inputType := request.Source.DocumentInputType
	if request.Role == QueryRole {
		inputType = request.Source.QueryInputType
	}
	body, err := json.Marshal(struct {
		Input           []string `json:"input"`
		Model           string   `json:"model"`
		InputType       string   `json:"input_type"`
		OutputDimension int      `json:"output_dimension"`
		OutputDType     string   `json:"output_dtype"`
		Truncation      bool     `json:"truncation"`
	}{request.Inputs, request.Source.Model, inputType, request.Source.SourceDimensions, request.Source.OutputDType, request.Source.Truncation})
	if err != nil {
		return EmbeddingResponse{}, ProviderError{Class: "transport", Retryable: true}
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, Endpoint, bytes.NewReader(body))
	if err != nil {
		return EmbeddingResponse{}, err
	}
	httpRequest.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := httpClient.Do(httpRequest)
	if err != nil {
		if ctx.Err() != nil {
			return EmbeddingResponse{}, ctx.Err()
		}
		return EmbeddingResponse{}, ProviderError{Class: "transport", Retryable: true}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
		retry := response.StatusCode == http.StatusRequestTimeout || response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500
		return EmbeddingResponse{}, ProviderError{Class: fmt.Sprintf("http_%d", response.StatusCode), Retryable: retry}
	}
	var wire struct {
		Model string `json:"model"`
		Data  []struct {
			Embedding []float32 `json:"embedding"`
			Index     *int      `json:"index"`
		} `json:"data"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}
	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&wire); err != nil {
		return EmbeddingResponse{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return EmbeddingResponse{}, fmt.Errorf("embedding response has trailing JSON")
	}
	out := EmbeddingResponse{Model: wire.Model, RequestID: response.Header.Get("x-request-id"), TotalTokens: wire.Usage.TotalTokens}
	for _, datum := range wire.Data {
		item := EmbeddingDatum{Values: datum.Embedding}
		if datum.Index != nil {
			item.Index = *datum.Index
			item.IndexPresent = true
		}
		out.Data = append(out.Data, item)
	}
	return out, nil
}
