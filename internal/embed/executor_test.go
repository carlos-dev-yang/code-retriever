package embed

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cidx/internal/embedclient"
)

type executorClient func(context.Context, embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error)

func (f executorClient) Embed(ctx context.Context, request embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
	return f(ctx, request)
}

func executorSource() embedclient.EmbeddingSourceSpec {
	return embedclient.EmbeddingSourceSpec{Provider: embedclient.ProviderID, Model: embedclient.Model, SourceDimensions: embedclient.SourceDimensions, OutputDType: embedclient.OutputDType, DocumentInputType: "document", QueryInputType: "query", AdapterVersion: embedclient.AdapterVersion}
}

func responseFor(request embedclient.EmbeddingRequest, reverse bool) embedclient.EmbeddingResponse {
	data := make([]embedclient.EmbeddingDatum, len(request.Inputs))
	for i := range request.Inputs {
		values := make([]float32, 1024)
		values[0] = float32(i + 1)
		index := i
		if reverse {
			index = len(request.Inputs) - i - 1
		}
		data[i] = embedclient.EmbeddingDatum{Index: index, IndexPresent: true, Values: values}
	}
	return embedclient.EmbeddingResponse{Model: embedclient.Model, Data: data}
}

func TestGroupUsesExactUTF8BytesAndStableKeys(t *testing.T) {
	inputs := []RequestInput{{Ordinal: 0, Key: "a", Bytes: []byte("é")}, {Ordinal: 1, Key: "b", Bytes: []byte("x")}, {Ordinal: 2, Key: "c", Bytes: []byte("yz")}}
	groups, err := Group(inputs, RequestLimits{MaxInputs: 2, MaxTotalBytes: 3})
	if err != nil || len(groups) != 2 || groups[0].TotalBytes != 3 || groups[1].TotalBytes != 2 || groups[0].Inputs[0].Key != "a" {
		t.Fatalf("groups=%#v err=%v", groups, err)
	}
	for _, bad := range [][]RequestInput{
		{{Ordinal: 0, Key: "a", Bytes: []byte{0xff}}},
		{{Ordinal: 0, Key: "a", Bytes: []byte("abcd")}},
		{{Ordinal: 0, Key: "a", Bytes: []byte("x")}, {Ordinal: 1, Key: "a", Bytes: []byte("y")}},
	} {
		if _, err := Group(bad, RequestLimits{MaxInputs: 2, MaxTotalBytes: 3}); err == nil {
			t.Fatalf("invalid group accepted: %#v", bad)
		}
	}
}

func TestGroupCanonicalCountAndByteCeilings(t *testing.T) {
	inputs := make([]RequestInput, 129)
	for i := range inputs {
		inputs[i] = RequestInput{Ordinal: i, Key: string(rune('a' + i)), Bytes: []byte("x")}
	}
	groups, err := Group(inputs[:128], RequestLimits{MaxInputs: 128, MaxTotalBytes: 256 << 10})
	if err != nil || len(groups) != 1 || len(groups[0].Inputs) != 128 {
		t.Fatalf("128 inputs: %#v %v", groups, err)
	}
	groups, err = Group(inputs, RequestLimits{MaxInputs: 128, MaxTotalBytes: 256 << 10})
	if err != nil || len(groups) != 2 || len(groups[1].Inputs) != 1 {
		t.Fatalf("129 inputs: %#v %v", groups, err)
	}
	exact := []RequestInput{{Ordinal: 0, Key: "exact", Bytes: make([]byte, 256<<10)}}
	if groups, err = Group(exact, RequestLimits{MaxInputs: 128, MaxTotalBytes: 256 << 10}); err != nil || len(groups) != 1 || groups[0].TotalBytes != 256<<10 {
		t.Fatalf("exact bytes: %#v %v", groups, err)
	}
	over := []RequestInput{{Ordinal: 0, Key: "over", Bytes: make([]byte, (256<<10)+1)}}
	if _, err := Group(over, RequestLimits{MaxInputs: 128, MaxTotalBytes: 256 << 10}); err == nil {
		t.Fatal("oversize input accepted")
	}
}

func TestExecuteBoundsConcurrencyRetriesAndRestoresResponseOrder(t *testing.T) {
	var active, maximum, calls int32
	var mu sync.Mutex
	var waits []time.Duration
	client := executorClient(func(_ context.Context, request embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
		current := atomic.AddInt32(&active, 1)
		for {
			prior := atomic.LoadInt32(&maximum)
			if current <= prior || atomic.CompareAndSwapInt32(&maximum, prior, current) {
				break
			}
		}
		defer atomic.AddInt32(&active, -1)
		call := atomic.AddInt32(&calls, 1)
		if call == 1 {
			return embedclient.EmbeddingResponse{}, embedclient.ProviderError{Class: "http_429", StatusCode: 429, Retryable: true, RetryAfter: 25 * time.Millisecond}
		}
		return responseFor(request, len(request.Inputs) == 2), nil
	})
	inputs := []RequestInput{{Ordinal: 0, Key: "a", Bytes: []byte("a")}, {Ordinal: 1, Key: "b", Bytes: []byte("b")}, {Ordinal: 2, Key: "c", Bytes: []byte("c")}}
	outcomes, err := Execute(context.Background(), client, executorSource(), embedclient.DocumentRole, inputs, ExecuteOptions{
		Limits: RequestLimits{MaxInputs: 2, MaxTotalBytes: 10}, MaxConcurrency: 2, AttemptTimeout: time.Second, MaxRetries: 1, RetryWaits: []time.Duration{10 * time.Millisecond},
		Wait: func(_ context.Context, wait time.Duration) error {
			mu.Lock()
			waits = append(waits, wait)
			mu.Unlock()
			return nil
		},
	}, nil)
	if err != nil || len(outcomes) != 2 || atomic.LoadInt32(&maximum) > 2 || atomic.LoadInt32(&calls) != 3 {
		t.Fatalf("outcomes=%#v maximum=%d calls=%d err=%v", outcomes, maximum, calls, err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(waits) != 1 || waits[0] != 25*time.Millisecond {
		t.Fatalf("waits=%v", waits)
	}
	if outcomes[0].Group.Ordinal != 0 || outcomes[0].Err != nil || outcomes[0].Vectors[0][0] != 2 || outcomes[0].Vectors[1][0] != 1 {
		t.Fatalf("ordered outcome=%#v", outcomes[0])
	}
}

func TestExecuteCancellationWaitsForActiveWorker(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	client := executorClient(func(context.Context, embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
		close(started)
		<-release // Deliberately ignores cancellation; Execute must join it.
		return embedclient.EmbeddingResponse{}, errors.New("late transport result")
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := Execute(ctx, client, executorSource(), embedclient.DocumentRole, []RequestInput{{Ordinal: 0, Key: "a", Bytes: []byte("a")}}, ExecuteOptions{Limits: RequestLimits{MaxInputs: 1, MaxTotalBytes: 1}, MaxConcurrency: 1, AttemptTimeout: time.Second}, nil)
		done <- err
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		t.Fatalf("returned before worker joined: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}

func TestExecuteReachesFourWorkersAndReturnsOrdinalOutcomes(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var current, maximum int32
	var once sync.Once
	client := executorClient(func(_ context.Context, request embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
		active := atomic.AddInt32(&current, 1)
		defer atomic.AddInt32(&current, -1)
		for {
			previous := atomic.LoadInt32(&maximum)
			if active <= previous || atomic.CompareAndSwapInt32(&maximum, previous, active) {
				break
			}
		}
		if active == 4 {
			once.Do(func() { close(started) })
		}
		<-release
		return responseFor(request, false), nil
	})
	inputs := make([]RequestInput, 8)
	for i := range inputs {
		inputs[i] = RequestInput{Ordinal: i, Key: string(rune('a' + i)), Bytes: []byte("x")}
	}
	done := make(chan []Outcome, 1)
	go func() {
		outcomes, err := Execute(context.Background(), client, executorSource(), embedclient.DocumentRole, inputs, ExecuteOptions{Limits: RequestLimits{MaxInputs: 1, MaxTotalBytes: 1}, MaxConcurrency: 4, AttemptTimeout: time.Second}, nil)
		if err != nil {
			t.Errorf("execute: %v", err)
		}
		done <- outcomes
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("four workers did not start")
	}
	close(release)
	outcomes := <-done
	if atomic.LoadInt32(&maximum) != 4 || len(outcomes) != 8 {
		t.Fatalf("maximum=%d outcomes=%d", maximum, len(outcomes))
	}
	for i, outcome := range outcomes {
		if outcome.Group.Ordinal != i {
			t.Fatalf("outcomes not ordinal: %#v", outcomes)
		}
	}
}

func TestExecuteRetryScheduleAndCancellation(t *testing.T) {
	input := []RequestInput{{Ordinal: 0, Key: "a", Bytes: []byte("a")}}
	t.Run("initial plus three retries", func(t *testing.T) {
		calls := 0
		var waits []time.Duration
		client := executorClient(func(context.Context, embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
			calls++
			return embedclient.EmbeddingResponse{}, embedclient.ProviderError{Class: "http_503", StatusCode: 503, Retryable: true, RetryAfter: 5 * time.Second}
		})
		outcomes, err := Execute(context.Background(), client, executorSource(), embedclient.DocumentRole, input, ExecuteOptions{
			Limits: RequestLimits{MaxInputs: 1, MaxTotalBytes: 1}, MaxConcurrency: 1, AttemptTimeout: time.Second, MaxRetries: 3,
			RetryWaits: []time.Duration{10 * time.Second, 20 * time.Second, 30 * time.Second},
			Wait:       func(_ context.Context, wait time.Duration) error { waits = append(waits, wait); return nil },
		}, nil)
		if err != nil || calls != 4 || len(outcomes) != 1 || outcomes[0].Attempts != 4 || len(waits) != 3 {
			t.Fatalf("calls=%d outcomes=%#v waits=%v err=%v", calls, outcomes, waits, err)
		}
		for i, want := range []time.Duration{10 * time.Second, 20 * time.Second, 30 * time.Second} {
			if waits[i] != want {
				t.Fatalf("waits=%v", waits)
			}
		}
	})
	t.Run("cancellation during wait", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		calls := 0
		client := executorClient(func(context.Context, embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
			calls++
			return embedclient.EmbeddingResponse{}, embedclient.ProviderError{Class: "transport", Retryable: true}
		})
		_, err := Execute(ctx, client, executorSource(), embedclient.DocumentRole, input, ExecuteOptions{
			Limits: RequestLimits{MaxInputs: 1, MaxTotalBytes: 1}, MaxConcurrency: 1, AttemptTimeout: time.Second, MaxRetries: 1, RetryWaits: []time.Duration{time.Hour},
			Wait: func(waitCtx context.Context, _ time.Duration) error { cancel(); <-waitCtx.Done(); return waitCtx.Err() },
		}, nil)
		if !errors.Is(err, context.Canceled) || calls != 1 {
			t.Fatalf("calls=%d err=%v", calls, err)
		}
	})
}

func TestExecuteAttemptTimeoutRetries(t *testing.T) {
	calls := 0
	client := executorClient(func(ctx context.Context, _ embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
		calls++
		<-ctx.Done()
		return embedclient.EmbeddingResponse{}, ctx.Err()
	})
	outcomes, err := Execute(context.Background(), client, executorSource(), embedclient.DocumentRole, []RequestInput{{Ordinal: 0, Key: "a", Bytes: []byte("a")}}, ExecuteOptions{
		Limits: RequestLimits{MaxInputs: 1, MaxTotalBytes: 1}, MaxConcurrency: 1, AttemptTimeout: time.Millisecond, MaxRetries: 1, RetryWaits: []time.Duration{0}, Wait: func(context.Context, time.Duration) error { return nil },
	}, nil)
	if err != nil || calls != 2 || len(outcomes) != 1 || outcomes[0].Attempts != 2 || !Transient(outcomes[0].Err) {
		t.Fatalf("calls=%d outcomes=%#v err=%v", calls, outcomes, err)
	}
}

func TestExecuteOutOfOrderCompletionHasOrdinalOutcomes(t *testing.T) {
	slowStarted, fastDone, releaseSlow := make(chan struct{}), make(chan struct{}), make(chan struct{})
	client := executorClient(func(_ context.Context, request embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
		if request.Inputs[0] == "slow" {
			close(slowStarted)
			<-releaseSlow
		} else {
			close(fastDone)
		}
		return responseFor(request, false), nil
	})
	done := make(chan []Outcome, 1)
	go func() {
		outcomes, err := Execute(context.Background(), client, executorSource(), embedclient.DocumentRole, []RequestInput{{Ordinal: 0, Key: "slow", Bytes: []byte("slow")}, {Ordinal: 1, Key: "fast", Bytes: []byte("fast")}}, ExecuteOptions{Limits: RequestLimits{MaxInputs: 1, MaxTotalBytes: 4}, MaxConcurrency: 2, AttemptTimeout: time.Second}, nil)
		if err != nil {
			t.Errorf("execute: %v", err)
		}
		done <- outcomes
	}()
	<-slowStarted
	<-fastDone
	close(releaseSlow)
	outcomes := <-done
	if len(outcomes) != 2 || outcomes[0].Group.Ordinal != 0 || outcomes[1].Group.Ordinal != 1 {
		t.Fatalf("outcomes=%#v", outcomes)
	}
}
