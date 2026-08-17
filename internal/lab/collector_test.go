package lab

import (
	"context"
	"errors"
	"testing"
	"time"

	"cidx/internal/embed"
	"cidx/internal/embedclient"
)

type fakeEmbeddingClient struct {
	calls    int
	response embedclient.EmbeddingResponse
	errs     []error
}
type sequenceClient struct {
	calls    int
	response embedclient.EmbeddingResponse
}

type collectorClientFunc func(context.Context, embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error)

func (f collectorClientFunc) Embed(ctx context.Context, request embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
	return f(ctx, request)
}

func (f *sequenceClient) Embed(_ context.Context, _ embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
	f.calls++
	if f.calls == 2 {
		return embedclient.EmbeddingResponse{}, embedclient.ProviderError{Class: "http_400"}
	}
	return f.response, nil
}

func (f *fakeEmbeddingClient) Embed(_ context.Context, request embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
	f.calls++
	if f.calls <= len(f.errs) {
		return embedclient.EmbeddingResponse{}, f.errs[f.calls-1]
	}
	return f.response, nil
}

func TestRetryClassificationBounded(t *testing.T) {
	ctx := context.Background()
	s, err := OpenStore(ctx, Options{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, retry := range []bool{false, true} {
		f := &fakeEmbeddingClient{errs: []error{embedclient.ProviderError{Class: "http", Retryable: retry}, embedclient.ProviderError{Class: "http", Retryable: retry}, embedclient.ProviderError{Class: "http", Retryable: retry}}}
		c := testCollector(t, s, f)
		c.MaxRetries = 2
		p, err := c.PlanWithOptions(ctx, []CaptureInput{testInput()}, PlanOptions{RetryFailed: true})
		if err != nil {
			t.Fatal(err)
		}
		_, err = c.Apply(ctx, p, 1, "m")
		if err == nil {
			t.Fatal("missing error")
		}
		want := 1
		if retry {
			want = 3
		}
		if f.calls != want {
			t.Fatalf("retry=%v calls=%d", retry, f.calls)
		}
	}
}
func TestAttemptTimeoutRetriesButParentCancellationStops(t *testing.T) {
	ctx := context.Background()
	s, err := OpenStore(ctx, Options{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	f := &fakeEmbeddingClient{errs: []error{context.DeadlineExceeded, context.DeadlineExceeded}}
	c := testCollector(t, s, f)
	c.MaxRetries = 1
	p, err := c.Plan(ctx, []CaptureInput{testInput()})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Apply(ctx, p, 1, "m")
	if err == nil || f.calls != 2 {
		t.Fatalf("timeout calls=%d err=%v", f.calls, err)
	}
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()
	f = &fakeEmbeddingClient{}
	c = testCollector(t, s, f)
	_, err = c.Apply(cancelCtx, p, 1, "m")
	if !errors.Is(err, context.Canceled) || f.calls != 0 {
		t.Fatalf("cancel calls=%d err=%v", f.calls, err)
	}
}
func testCollector(t *testing.T, store *Store, client embedclient.EmbeddingClient) Collector {
	t.Helper()
	return Collector{Store: store, Client: client, Source: embedclient.EmbeddingSourceSpec{Provider: embedclient.ProviderID, Model: embedclient.Model, SourceDimensions: embedclient.SourceDimensions, OutputDType: embedclient.OutputDType, DocumentInputType: "document", QueryInputType: "query", AdapterVersion: embedclient.AdapterVersion}, SourceProfile: testDigest, RequestLimits: embed.RequestLimits{MaxInputs: 10, MaxTotalBytes: 100000}, MaxConcurrency: 1, AttemptTimeout: time.Second, MaxRetries: 3, RetryWaits: []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}, Wait: func(context.Context, time.Duration) error { return nil }}
}
func testInput() CaptureInput {
	return CaptureInput{InputRecord: InputRecord{InputHash: testDigest, CanonicalTextProfile: "text", CanonicalBytes: []byte("path: a.go\nbody:\nfunc A() {}\n"), Generation: 1, ManifestSHA256: "manifest", SegmentID: 1}}
}

func TestCapturePlanApplyAndResume(t *testing.T) {
	ctx := context.Background()
	store, err := OpenStore(ctx, Options{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	values := make([]float32, 1024)
	values[0] = 1
	client := &fakeEmbeddingClient{response: embedclient.EmbeddingResponse{Model: embedclient.Model, Data: []embedclient.EmbeddingDatum{{Index: 0, IndexPresent: true, Values: values}}}}
	collector := testCollector(t, store, client)
	plan, err := collector.Plan(ctx, []CaptureInput{testInput()})
	if err != nil {
		t.Fatal(err)
	}
	if client.calls != 0 || plan.PaidMisses != 1 {
		t.Fatalf("plan calls/misses=%d/%d", client.calls, plan.PaidMisses)
	}
	if _, err := collector.Apply(ctx, plan, 1, "manifest"); err != nil {
		t.Fatal(err)
	}
	if client.calls != 1 {
		t.Fatalf("apply calls=%d", client.calls)
	}
	plan, err = collector.Plan(ctx, []CaptureInput{testInput()})
	if err != nil {
		t.Fatal(err)
	}
	if plan.PaidMisses != 0 || plan.RawHits != 1 {
		t.Fatalf("resume plan=%#v", plan)
	}
}
func TestPartialResumeOnlyRequestsUnpersistedBatch(t *testing.T) {
	ctx := context.Background()
	s, err := OpenStore(ctx, Options{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	v := make([]float32, 1024)
	v[0] = 1
	client := &sequenceClient{response: embedclient.EmbeddingResponse{Model: embedclient.Model, Data: []embedclient.EmbeddingDatum{{Index: 0, IndexPresent: true, Values: v}}}}
	c := testCollector(t, s, client)
	c.RequestLimits.MaxInputs = 1
	second := testInput()
	second.InputHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	second.CanonicalBytes = []byte("second\n")
	p, err := c.Plan(ctx, []CaptureInput{testInput(), second})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.Apply(ctx, p, 1, "m"); err == nil {
		t.Fatal("later batch failure missing")
	}
	p, err = c.PlanWithOptions(ctx, []CaptureInput{testInput(), second}, PlanOptions{RetryFailed: true})
	if err != nil || p.RawHits != 1 || p.PaidMisses != 1 {
		t.Fatalf("resume %#v %v", p, err)
	}
}
func TestCaptureRejectsInvalidBatchBeforeWrite(t *testing.T) {
	ctx := context.Background()
	store, err := OpenStore(ctx, Options{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	client := &fakeEmbeddingClient{response: embedclient.EmbeddingResponse{Model: embedclient.Model, Data: []embedclient.EmbeddingDatum{{Index: 0, IndexPresent: true, Values: make([]float32, 3)}}}}
	collector := testCollector(t, store, client)
	plan, err := collector.Plan(ctx, []CaptureInput{testInput()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = collector.Apply(ctx, plan, 1, "manifest"); err == nil {
		t.Fatal("invalid response accepted")
	}
	hits, err := store.ExistingKeys(ctx, testDigest, []string{testDigest})
	if err != nil || hits[testDigest] {
		t.Fatalf("invalid batch persisted: %#v %v", hits, err)
	}
}

func TestCaptureHandlerFailureStillAccountsSuccessfulProviderResponse(t *testing.T) {
	ctx := context.Background()
	store, err := OpenStore(ctx, Options{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	values := make([]float32, 1024)
	values[0] = 1
	collector := testCollector(t, store, &fakeEmbeddingClient{response: embedclient.EmbeddingResponse{Model: embedclient.Model, TotalTokens: 7, Data: []embedclient.EmbeddingDatum{{Index: 0, IndexPresent: true, Values: values}}}})
	commitErr := errors.New("durable store unavailable")
	collector.putSources = func(context.Context, []DocumentRaw, int) error { return commitErr }
	plan, err := collector.Plan(ctx, []CaptureInput{testInput()})
	if err != nil {
		t.Fatal(err)
	}
	result, err := collector.Apply(ctx, plan, 1, "m")
	if !errors.Is(err, commitErr) || result.Requested != 1 || result.ActualTokens != 7 || result.Persisted != 0 || result.Failed != 0 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestCaptureCancellationFailureRemainsRetryable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store, err := OpenStore(context.Background(), Options{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	collector := testCollector(t, store, collectorClientFunc(func(context.Context, embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
		cancel()
		return embedclient.EmbeddingResponse{}, context.Canceled
	}))
	collector.MaxRetries = 0
	plan, err := collector.Plan(context.Background(), []CaptureInput{testInput()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := collector.Apply(ctx, plan, 1, "m"); !errors.Is(err, context.Canceled) {
		t.Fatalf("apply error=%v", err)
	}
	retryPlan, err := collector.Plan(context.Background(), []CaptureInput{testInput()})
	if err != nil || retryPlan.PaidMisses != 1 || retryPlan.SkippedTerminal != 0 {
		t.Fatalf("retry plan=%#v err=%v", retryPlan, err)
	}
}

func TestCapturePlanSkipsTerminalFailureUnlessRequested(t *testing.T) {
	ctx := context.Background()
	store, err := OpenStore(ctx, Options{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	id, err := store.StartCapture(ctx, CaptureRun{Generation: 1, ManifestSHA256: "m", SourceProfile: testDigest})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RecordFailure(ctx, id, RawEmbeddingKey{SourceProfile: testDigest, InputHash: testDigest}, "terminal", "provider", "safe", 1); err != nil {
		t.Fatal(err)
	}
	collector := testCollector(t, store, &fakeEmbeddingClient{})
	plan, err := collector.Plan(ctx, []CaptureInput{testInput()})
	if err != nil || plan.SkippedTerminal != 1 || plan.PaidMisses != 0 {
		t.Fatalf("default %#v %v", plan, err)
	}
	plan, err = collector.PlanWithOptions(ctx, []CaptureInput{testInput()}, PlanOptions{RetryFailed: true})
	if err != nil || plan.PaidMisses != 1 {
		t.Fatalf("override %#v %v", plan, err)
	}
	id2, err := store.StartCapture(ctx, CaptureRun{Generation: 1, ManifestSHA256: "m", SourceProfile: testDigest})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RecordFailure(ctx, id2, RawEmbeddingKey{SourceProfile: testDigest, InputHash: testDigest}, "retryable", "provider", "safe", 1); err != nil {
		t.Fatal(err)
	}
	plan, err = collector.Plan(ctx, []CaptureInput{testInput()})
	if err != nil || plan.PaidMisses != 1 {
		t.Fatalf("latest retryable %#v %v", plan, err)
	}
}

func TestCaptureFailureForeignKey(t *testing.T) {
	ctx := context.Background()
	s, err := OpenStore(ctx, Options{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.RecordFailure(ctx, 999, RawEmbeddingKey{SourceProfile: "s", InputHash: "h"}, "terminal", "x", "safe", 1); err == nil {
		t.Fatal("invalid run accepted")
	}
}

func TestAtomicRawBatchConflictRollsBack(t *testing.T) {
	ctx := context.Background()
	s, err := OpenStore(ctx, Options{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	v := make([]float32, 1024)
	v[0] = 1
	fv, err := NewF32Vector(v, 1024)
	if err != nil {
		t.Fatal(err)
	}
	base := DocumentRaw{SourceProfile: testDigest, InputHash: testDigest, ResponseModel: "voyage-code-4", Vector: fv}
	if err := s.PutDocumentSource(ctx, base, 1024); err != nil {
		t.Fatal(err)
	}
	changed := append([]float32(nil), v...)
	changed[1] = 1
	bad, _ := NewF32Vector(changed, 1024)
	newRaw := base
	newRaw.InputHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	conflict := base
	conflict.Vector = bad
	if err := s.PutDocumentSources(ctx, []DocumentRaw{newRaw, conflict}, 1024); err == nil {
		t.Fatal("conflict accepted")
	}
	hits, err := s.ExistingKeys(ctx, testDigest, []string{newRaw.InputHash})
	if err != nil || hits[newRaw.InputHash] {
		t.Fatal("partial batch persisted")
	}
}
