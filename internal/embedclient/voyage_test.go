package embedclient

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
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

func TestRetryAfterAcceptsDeltaAndHTTPDateWithoutArbitraryCeiling(t *testing.T) {
	now := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	if got := parseRetryAfter("172800", now); got != 48*time.Hour {
		t.Fatalf("delta=%v", got)
	}
	if got := parseRetryAfter(now.Add(90*time.Second).Format(http.TimeFormat), now); got != 90*time.Second {
		t.Fatalf("date=%v", got)
	}
	if got := parseRetryAfter("0", now); got != 0 {
		t.Fatalf("zero=%v", got)
	}
	for _, invalid := range []string{"+30", "-1", "1.5", "thirty", "999999999999999999999999999999"} {
		if got := parseRetryAfter(invalid, now); got != 0 {
			t.Fatalf("invalid %q=%v", invalid, got)
		}
	}
}

func TestVoyageClassifiesTerminalAndTransientHTTP(t *testing.T) {
	spec := EmbeddingSourceSpec{Provider: ProviderID, Model: Model, SourceDimensions: SourceDimensions, OutputDType: OutputDType, DocumentInputType: "document", QueryInputType: "query", AdapterVersion: AdapterVersion}
	for _, status := range []int{http.StatusBadRequest, http.StatusRequestTimeout, http.StatusTooManyRequests, http.StatusInternalServerError} {
		calls := 0
		client := VoyageClient{APIKey: "x", HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("{}")), Header: make(http.Header)}, nil
		})}}
		_, err := client.Embed(context.Background(), EmbeddingRequest{Source: spec, Role: DocumentRole, Inputs: []string{"x"}})
		if err == nil || calls != 1 {
			t.Fatalf("status %d calls=%d err=%v", status, calls, err)
		}
		if IsRetryable(err) != (status == 408 || status == 429 || status >= 500) {
			t.Fatalf("status classification %d", status)
		}
	}
}
