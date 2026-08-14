package embedclient

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestVoyageRequestContractAndInvalidSpecDoesNotCallTransport(t *testing.T) {
	calls := 0
	client := VoyageClient{APIKey: "test", HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != Endpoint || r.Header.Get("Authorization") != "Bearer test" {
			t.Fatal("endpoint/auth")
		}
		body, _ := io.ReadAll(r.Body)
		text := string(body)
		for _, field := range []string{`"model":"voyage-code-4"`, `"input_type":"document"`, `"output_dimension":1024`, `"output_dtype":"float"`, `"truncation":false`} {
			if !strings.Contains(text, field) {
				t.Fatalf("missing %s: %s", field, text)
			}
		}
		if strings.Contains(text, "encoding_format") {
			t.Fatal("encoding format present")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"model":"voyage-code-4","data":[{"index":0,"embedding":[1]}]}`)), Header: make(http.Header)}, nil
	})}}
	spec := EmbeddingSourceSpec{Provider: ProviderID, Model: Model, SourceDimensions: SourceDimensions, OutputDType: OutputDType, DocumentInputType: "document", QueryInputType: "query", AdapterVersion: AdapterVersion}
	if _, err := client.Embed(context.Background(), EmbeddingRequest{Source: spec, Role: DocumentRole, Inputs: []string{"x"}}); err != nil {
		t.Fatal(err)
	}
	bad := spec
	bad.SourceDimensions = 1
	if _, err := client.Embed(context.Background(), EmbeddingRequest{Source: bad, Role: DocumentRole, Inputs: []string{"x"}}); err == nil || calls != 1 {
		t.Fatalf("invalid spec transport calls=%d err=%v", calls, err)
	}
}

func TestVoyageClassifiesTerminalAndTransientHTTP(t *testing.T) {
	spec := EmbeddingSourceSpec{Provider: ProviderID, Model: Model, SourceDimensions: SourceDimensions, OutputDType: OutputDType, DocumentInputType: "document", QueryInputType: "query", AdapterVersion: AdapterVersion}
	for _, status := range []int{http.StatusBadRequest, http.StatusTooManyRequests, http.StatusInternalServerError} {
		calls := 0
		client := VoyageClient{APIKey: "x", HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("{}")), Header: make(http.Header)}, nil
		})}}
		_, err := client.Embed(context.Background(), EmbeddingRequest{Source: spec, Role: DocumentRole, Inputs: []string{"x"}})
		if err == nil || calls != 1 {
			t.Fatalf("status %d calls=%d err=%v", status, calls, err)
		}
		if IsRetryable(err) != (status == 429 || status >= 500) {
			t.Fatalf("status classification %d", status)
		}
	}
}
