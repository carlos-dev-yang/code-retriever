package app

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"cidx/internal/embedclient"
	"cidx/internal/index"
	"cidx/internal/search/lexical"
	"cidx/internal/store"
	"cidx/internal/vector"
)

type fakeDocumentEmbedder struct {
	mu      sync.Mutex
	calls   int
	request embedclient.EmbeddingRequest
}

type blockingDocumentEmbedder struct {
	fakeDocumentEmbedder
	started chan struct{}
	release chan struct{}
}

type sequenceDocumentEmbedder struct {
	mu    sync.Mutex
	calls int
}

type outOfOrderDocumentEmbedder struct {
	failureStarted chan struct{}
	releaseFailure chan struct{}
}

func (f *outOfOrderDocumentEmbedder) Embed(ctx context.Context, request embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
	if strings.Contains(request.Inputs[0], "First") {
		return documentResponse(request, 11, false), nil
	}
	select {
	case f.failureStarted <- struct{}{}:
	default:
	}
	select {
	case <-f.releaseFailure:
	case <-ctx.Done():
		return embedclient.EmbeddingResponse{}, ctx.Err()
	}
	return embedclient.EmbeddingResponse{}, embedclient.ProviderError{Class: "rejected", Retryable: false}
}

type cancellingDocumentEmbedder struct{ cancel context.CancelFunc }

func (f cancellingDocumentEmbedder) Embed(_ context.Context, _ embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
	f.cancel()
	return embedclient.EmbeddingResponse{}, context.Canceled
}

type documentEmbedderFunc func(context.Context, embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error)

func (f documentEmbedderFunc) Embed(ctx context.Context, request embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
	return f(ctx, request)
}

func documentResponse(request embedclient.EmbeddingRequest, tokens int, zeroVectors bool) embedclient.EmbeddingResponse {
	data := make([]embedclient.EmbeddingDatum, len(request.Inputs))
	for i := range data {
		values := make([]float32, 1024)
		if zeroVectors {
			// This is source-valid but becomes a zero vector after the active
			// serving-prefix reduction.
			values[700] = 1
		} else {
			values[0] = 1
		}
		data[i] = embedclient.EmbeddingDatum{Index: i, IndexPresent: true, Values: values}
	}
	return embedclient.EmbeddingResponse{Model: embedclient.Model, Data: data, TotalTokens: tokens}
}

func (f *sequenceDocumentEmbedder) Embed(_ context.Context, request embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	if strings.Contains(request.Inputs[0], "Second") {
		return embedclient.EmbeddingResponse{}, embedclient.ProviderError{Class: "rejected", Retryable: false}
	}
	data := make([]embedclient.EmbeddingDatum, len(request.Inputs))
	for i := range data {
		values := make([]float32, 1024)
		values[0] = 1
		data[i] = embedclient.EmbeddingDatum{Index: i, IndexPresent: true, Values: values}
	}
	return embedclient.EmbeddingResponse{Model: embedclient.Model, Data: data, TotalTokens: len(data)}, nil
}

func (f *blockingDocumentEmbedder) Embed(ctx context.Context, request embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
	select {
	case f.started <- struct{}{}:
	default:
	}
	select {
	case <-f.release:
	case <-ctx.Done():
		return embedclient.EmbeddingResponse{}, ctx.Err()
	}
	return f.fakeDocumentEmbedder.Embed(ctx, request)
}

func TestPublicEmbeddingRetainsFirstBatchAfterLaterFailure(t *testing.T) {
	ctx, root := context.Background(), t.TempDir()
	runGit(t, root, "init")
	mustWriteFile(t, filepath.Join(root, ".cidx", "config.json"), "{}")
	mustWriteFile(t, filepath.Join(root, "a.go"), "package p\nfunc First() int { return 1 }\nfunc Second() int { return 2 }\n")
	runGit(t, root, "add", "a.go")
	resolved := materializeResolvedWithBatch(t, 1)
	production, err := store.OpenProduction(ctx, root, resolved)
	if err != nil {
		t.Fatal(err)
	}
	defer production.Close()
	if _, err := index.New(production).Execute(ctx, index.Request{Root: root, Reason: index.ReasonManual, Config: resolved}); err != nil {
		t.Fatal(err)
	}
	service := PublicEmbedding{Production: production, Resolved: resolved}
	plan, err := service.Plan(ctx)
	if err != nil || plan.PaidInputs != 2 {
		t.Fatalf("plan=%#v err=%v", plan, err)
	}
	fake := &sequenceDocumentEmbedder{}
	result, err := service.Apply(ctx, plan, PublicEmbeddingApply{Approved: true, Client: fake})
	if err != nil || result.Requested != 2 || result.Succeeded != 1 || result.Failed != 1 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	states, err := production.ActiveSegmentStates(ctx, resolved)
	if err != nil {
		t.Fatal(err)
	}
	ready, failed := 0, 0
	for _, state := range states {
		if state.State == store.EmbeddingReady {
			ready++
		}
		if state.State == store.EmbeddingFailed {
			failed++
		}
	}
	if ready != 1 || failed != 1 {
		t.Fatalf("states ready=%d failed=%d", ready, failed)
	}
	next, err := service.Plan(ctx)
	if err != nil || next.Ready != 1 || next.SkippedTerminal != 1 || next.PaidInputs != 0 {
		t.Fatalf("next=%#v err=%v", next, err)
	}
	db, err := sql.Open("sqlite", filepath.Join(root, ".cidx", "db", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var requested, succeeded, failures int
	var status string
	if err := db.QueryRowContext(ctx, `SELECT requested_count,succeeded_count,failed_count,status FROM embedding_runs ORDER BY id DESC LIMIT 1`).Scan(&requested, &succeeded, &failures, &status); err != nil || requested != 2 || succeeded != 1 || failures != 1 || status != "partially_succeeded" {
		t.Fatalf("run=%d/%d/%d %q err=%v", requested, succeeded, failures, status, err)
	}
}

func TestPublicEmbeddingOutOfOrderConcurrentPartialFailurePreservesSuccess(t *testing.T) {
	ctx, root := context.Background(), t.TempDir()
	runGit(t, root, "init")
	mustWriteFile(t, filepath.Join(root, ".cidx", "config.json"), "{}")
	mustWriteFile(t, filepath.Join(root, "a.go"), "package p\nfunc First() int { return 1 }\nfunc Second() int { return 2 }\n")
	runGit(t, root, "add", "a.go")
	resolved := materializeResolvedWithBatch(t, 1)
	production, err := store.OpenProduction(ctx, root, resolved)
	if err != nil {
		t.Fatal(err)
	}
	defer production.Close()
	if _, err := index.New(production).Execute(ctx, index.Request{Root: root, Reason: index.ReasonManual, Config: resolved}); err != nil {
		t.Fatal(err)
	}
	service := PublicEmbedding{Production: production, Resolved: resolved}
	plan, err := service.Plan(ctx)
	if err != nil || plan.PaidInputs != 2 {
		t.Fatalf("plan=%#v err=%v", plan, err)
	}
	client := &outOfOrderDocumentEmbedder{failureStarted: make(chan struct{}, 1), releaseFailure: make(chan struct{})}
	done := make(chan struct {
		result PublicEmbeddingResult
		err    error
	}, 1)
	go func() {
		result, err := service.Apply(ctx, plan, PublicEmbeddingApply{Approved: true, Client: client})
		done <- struct {
			result PublicEmbeddingResult
			err    error
		}{result, err}
	}()
	<-client.failureStarted
	deadline := time.After(time.Second)
	for {
		states, stateErr := production.ActiveSegmentStates(ctx, resolved)
		if stateErr != nil {
			t.Fatal(stateErr)
		}
		ready := 0
		for _, state := range states {
			if state.State == store.EmbeddingReady {
				ready++
			}
		}
		if ready == 1 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("successful group was not published while failing request remained blocked")
		case <-time.After(time.Millisecond):
		}
	}
	close(client.releaseFailure)
	completed := <-done
	if completed.err != nil || completed.result.Requested != 2 || completed.result.ActualTokens != 11 || completed.result.Succeeded != 1 || completed.result.Failed != 1 || completed.result.Discarded != 0 {
		t.Fatalf("result=%#v err=%v", completed.result, completed.err)
	}
	states, err := production.ActiveSegmentStates(ctx, resolved)
	if err != nil {
		t.Fatal(err)
	}
	ready, failed := 0, 0
	for _, state := range states {
		if state.State == store.EmbeddingReady {
			ready++
		}
		if state.State == store.EmbeddingFailed {
			failed++
		}
	}
	if ready != 1 || failed != 1 {
		t.Fatalf("states ready=%d failed=%d", ready, failed)
	}
}

func TestPublicEmbeddingCancellationRecordsRetryableFailure(t *testing.T) {
	ctx, root := context.Background(), t.TempDir()
	runGit(t, root, "init")
	mustWriteFile(t, filepath.Join(root, ".cidx", "config.json"), "{}")
	mustWriteFile(t, filepath.Join(root, "a.go"), "package p\nfunc Cancelled() int { return 1 }\n")
	runGit(t, root, "add", "a.go")
	resolved := materializeResolved(t)
	production, err := store.OpenProduction(ctx, root, resolved)
	if err != nil {
		t.Fatal(err)
	}
	defer production.Close()
	if _, err := index.New(production).Execute(ctx, index.Request{Root: root, Reason: index.ReasonManual, Config: resolved}); err != nil {
		t.Fatal(err)
	}
	service := PublicEmbedding{Production: production, Resolved: resolved}
	plan, err := service.Plan(ctx)
	if err != nil || plan.PaidInputs != 1 {
		t.Fatalf("plan=%#v err=%v", plan, err)
	}
	applyCtx, cancel := context.WithCancel(ctx)
	result, err := service.Apply(applyCtx, plan, PublicEmbeddingApply{Approved: true, Client: cancellingDocumentEmbedder{cancel: cancel}})
	if !errors.Is(err, context.Canceled) || result.Requested != 1 || result.ActualTokens != 0 || result.Succeeded != 0 || result.Failed != 1 || result.Discarded != 0 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	db, err := sql.Open("sqlite", filepath.Join(root, ".cidx", "db", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var classification, class, status string
	if err := db.QueryRowContext(ctx, `SELECT classification,error_class FROM embedding_failures ORDER BY id DESC LIMIT 1`).Scan(&classification, &class); err != nil || classification != "retryable" || class != "cancelled" {
		t.Fatalf("failure=%q/%q err=%v", classification, class, err)
	}
	if err := db.QueryRowContext(ctx, `SELECT status FROM embedding_runs ORDER BY id DESC LIMIT 1`).Scan(&status); err != nil || status != "cancelled" {
		t.Fatalf("status=%q err=%v", status, err)
	}
	next, err := service.Plan(ctx)
	if err != nil || next.PaidInputs != 1 || next.SkippedTerminal != 0 {
		t.Fatalf("next=%#v err=%v", next, err)
	}
}

func TestPublicEmbeddingClassifiesResponseAndTransformFailures(t *testing.T) {
	for _, scenario := range []struct {
		name, wantClass, wantErrorClass string
		wantTokens                      int
		response                        func(embedclient.EmbeddingRequest) embedclient.EmbeddingResponse
	}{
		{
			name: "response validation", wantClass: "terminal", wantErrorClass: "response_validation", wantTokens: 0,
			response: func(request embedclient.EmbeddingRequest) embedclient.EmbeddingResponse {
				response := documentResponse(request, 7, false)
				response.Data[0].Values = []float32{1}
				return response
			},
		},
		{
			name: "zero vector transform", wantClass: "terminal", wantErrorClass: "transform", wantTokens: 9,
			response: func(request embedclient.EmbeddingRequest) embedclient.EmbeddingResponse {
				return documentResponse(request, 9, true)
			},
		},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			ctx, root := context.Background(), t.TempDir()
			runGit(t, root, "init")
			mustWriteFile(t, filepath.Join(root, ".cidx", "config.json"), "{}")
			mustWriteFile(t, filepath.Join(root, "a.go"), "package p\nfunc Failure() int { return 1 }\n")
			runGit(t, root, "add", "a.go")
			resolved := materializeResolved(t)
			production, err := store.OpenProduction(ctx, root, resolved)
			if err != nil {
				t.Fatal(err)
			}
			defer production.Close()
			if _, err := index.New(production).Execute(ctx, index.Request{Root: root, Reason: index.ReasonManual, Config: resolved}); err != nil {
				t.Fatal(err)
			}
			service := PublicEmbedding{Production: production, Resolved: resolved}
			plan, err := service.Plan(ctx)
			if err != nil || plan.PaidInputs != 1 {
				t.Fatalf("plan=%#v err=%v", plan, err)
			}
			result, err := service.Apply(ctx, plan, PublicEmbeddingApply{Approved: true, Client: documentEmbedderFunc(func(_ context.Context, request embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
				return scenario.response(request), nil
			})})
			if err != nil || result.Requested != 1 || result.ActualTokens != scenario.wantTokens || result.Succeeded != 0 || result.Failed != 1 || result.Discarded != 0 {
				t.Fatalf("result=%#v err=%v", result, err)
			}
			db, err := sql.Open("sqlite", filepath.Join(root, ".cidx", "db", "index.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			var classification, class, status string
			var attempts, requested, actualTokens, failed, discarded int
			if err := db.QueryRowContext(ctx, `SELECT classification,error_class,attempts FROM embedding_failures ORDER BY id DESC LIMIT 1`).Scan(&classification, &class, &attempts); err != nil || classification != scenario.wantClass || class != scenario.wantErrorClass || attempts != 1 {
				t.Fatalf("failure=%q/%q attempts=%d err=%v", classification, class, attempts, err)
			}
			if err := db.QueryRowContext(ctx, `SELECT requested_count,actual_tokens,failed_count,discarded_count,status FROM embedding_runs ORDER BY id DESC LIMIT 1`).Scan(&requested, &actualTokens, &failed, &discarded, &status); err != nil || requested != 1 || actualTokens != scenario.wantTokens || failed != 1 || discarded != 0 || status != "failed" {
				t.Fatalf("run=%d/%d/%d/%d/%q err=%v", requested, actualTokens, failed, discarded, status, err)
			}
		})
	}
}

func TestPublicEmbeddingRejectsStalePlanBeforeProviderCall(t *testing.T) {
	ctx, root := context.Background(), t.TempDir()
	runGit(t, root, "init")
	mustWriteFile(t, filepath.Join(root, ".cidx", "config.json"), "{}")
	mustWriteFile(t, filepath.Join(root, "a.go"), "package p\nfunc Stale() int { return 1 }\n")
	runGit(t, root, "add", "a.go")
	resolved := materializeResolved(t)
	production, err := store.OpenProduction(ctx, root, resolved)
	if err != nil {
		t.Fatal(err)
	}
	defer production.Close()
	if _, err := index.New(production).Execute(ctx, index.Request{Root: root, Reason: index.ReasonManual, Config: resolved}); err != nil {
		t.Fatal(err)
	}
	service := PublicEmbedding{Production: production, Resolved: resolved}
	plan, err := service.Plan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(root, ".cidx", "db", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `UPDATE meta SET active_generation=active_generation+1`); err != nil {
		t.Fatal(err)
	}
	fake := &fakeDocumentEmbedder{}
	if _, err := service.Apply(ctx, plan, PublicEmbeddingApply{Approved: true, Client: fake}); !errors.Is(err, ErrEmbeddingPlanStale) || fake.calls != 0 {
		t.Fatalf("stale calls=%d err=%v", fake.calls, err)
	}
}

func TestPublicEmbeddingDiscardsResponseForRemovedActiveKey(t *testing.T) {
	ctx, root := context.Background(), t.TempDir()
	runGit(t, root, "init")
	mustWriteFile(t, filepath.Join(root, ".cidx", "config.json"), "{}")
	mustWriteFile(t, filepath.Join(root, "a.go"), "package p\nfunc Removed() int { return 1 }\n")
	runGit(t, root, "add", "a.go")
	resolved := materializeResolved(t)
	production, err := store.OpenProduction(ctx, root, resolved)
	if err != nil {
		t.Fatal(err)
	}
	defer production.Close()
	if _, err := index.New(production).Execute(ctx, index.Request{Root: root, Reason: index.ReasonManual, Config: resolved}); err != nil {
		t.Fatal(err)
	}
	service := PublicEmbedding{Production: production, Resolved: resolved}
	plan, err := service.Plan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	blocking := &blockingDocumentEmbedder{started: make(chan struct{}, 1), release: make(chan struct{})}
	done := make(chan struct {
		result PublicEmbeddingResult
		err    error
	}, 1)
	go func() {
		result, err := service.Apply(ctx, plan, PublicEmbeddingApply{Approved: true, Client: blocking})
		done <- struct {
			result PublicEmbeddingResult
			err    error
		}{result, err}
	}()
	<-blocking.started
	db, err := sql.Open("sqlite", filepath.Join(root, ".cidx", "db", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `UPDATE embedding_segments SET serving_profile='removed'`); err != nil {
		t.Fatal(err)
	}
	close(blocking.release)
	completed := <-done
	if completed.err != nil || completed.result.Discarded != plan.PaidInputs {
		t.Fatalf("result=%#v err=%v", completed.result, completed.err)
	}
	var vectors, failures int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM vector_cache`).Scan(&vectors); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM embedding_failures`).Scan(&failures); err != nil {
		t.Fatal(err)
	}
	if vectors != 0 || failures != 0 {
		t.Fatalf("late inactive wrote vectors=%d failures=%d", vectors, failures)
	}
	var discarded int
	var status string
	if err := db.QueryRowContext(ctx, `SELECT discarded_count,status FROM embedding_runs ORDER BY id DESC LIMIT 1`).Scan(&discarded, &status); err != nil || discarded != plan.PaidInputs || status != "failed" {
		t.Fatalf("late inactive run discarded=%d status=%q err=%v", discarded, status, err)
	}
}

func TestPublicEmbeddingRejectsBlockedResponseAfterGenerationChange(t *testing.T) {
	ctx, root := context.Background(), t.TempDir()
	runGit(t, root, "init")
	mustWriteFile(t, filepath.Join(root, ".cidx", "config.json"), "{}")
	mustWriteFile(t, filepath.Join(root, "a.go"), "package p\nfunc Generation() int { return 1 }\n")
	runGit(t, root, "add", "a.go")
	resolved := materializeResolved(t)
	production, err := store.OpenProduction(ctx, root, resolved)
	if err != nil {
		t.Fatal(err)
	}
	defer production.Close()
	if _, err := index.New(production).Execute(ctx, index.Request{Root: root, Reason: index.ReasonManual, Config: resolved}); err != nil {
		t.Fatal(err)
	}
	service := PublicEmbedding{Production: production, Resolved: resolved}
	plan, err := service.Plan(ctx)
	if err != nil {
		t.Fatal(err)
	}
	blocking := &blockingDocumentEmbedder{started: make(chan struct{}, 1), release: make(chan struct{})}
	done := make(chan struct {
		result PublicEmbeddingResult
		err    error
	}, 1)
	go func() {
		result, err := service.Apply(ctx, plan, PublicEmbeddingApply{Approved: true, Client: blocking})
		done <- struct {
			result PublicEmbeddingResult
			err    error
		}{result, err}
	}()
	<-blocking.started
	db, err := sql.Open("sqlite", filepath.Join(root, ".cidx", "db", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `UPDATE meta SET active_generation=active_generation+1`); err != nil {
		t.Fatal(err)
	}
	close(blocking.release)
	completed := <-done
	if !errors.Is(completed.err, store.ErrEmbeddingStateChanged) || completed.result.Requested != 1 || completed.result.ActualTokens != 1 || completed.result.Succeeded != 0 || completed.result.Failed != 0 || completed.result.Discarded != 0 {
		t.Fatalf("generation-change result=%#v err=%v", completed.result, completed.err)
	}
	var vectors int
	var status string
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM vector_cache`).Scan(&vectors); err != nil {
		t.Fatal(err)
	}
	var requested, tokens int
	if err := db.QueryRowContext(ctx, `SELECT requested_count,actual_tokens,status FROM embedding_runs ORDER BY id DESC LIMIT 1`).Scan(&requested, &tokens, &status); err != nil || requested != 1 || tokens != 1 || status != "failed" || vectors != 0 {
		t.Fatalf("requested=%d tokens=%d status=%q vectors=%d err=%v", requested, tokens, status, vectors, err)
	}
}

func (f *fakeDocumentEmbedder) Embed(_ context.Context, request embedclient.EmbeddingRequest) (embedclient.EmbeddingResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.request = request
	data := make([]embedclient.EmbeddingDatum, len(request.Inputs))
	for i := range data {
		values := make([]float32, 1024)
		values[i] = 1
		data[i] = embedclient.EmbeddingDatum{Index: i, IndexPresent: true, Values: values}
	}
	return embedclient.EmbeddingResponse{Model: embedclient.Model, Data: data, TotalTokens: len(request.Inputs)}, nil
}

func TestPublicEmbeddingPlansWithoutProviderAndAppliesDirectlyToProduction(t *testing.T) {
	ctx, root := context.Background(), t.TempDir()
	runGit(t, root, "init")
	mustWriteFile(t, filepath.Join(root, ".cidx", "config.json"), "{}")
	mustWriteFile(t, filepath.Join(root, "a.go"), "package p\nfunc Indexed() int { return 1 }\n")
	runGit(t, root, "add", "a.go")
	resolved := materializeResolved(t)
	production, err := store.OpenProduction(ctx, root, resolved)
	if err != nil {
		t.Fatal(err)
	}
	defer production.Close()
	if _, err := index.New(production).Execute(ctx, index.Request{Root: root, Reason: index.ReasonManual, Config: resolved}); err != nil {
		t.Fatal(err)
	}
	service := PublicEmbedding{Production: production, Resolved: resolved}
	plan, err := service.Plan(ctx)
	if err != nil || plan.ActiveDistinct == 0 || plan.PaidInputs == 0 {
		t.Fatalf("plan=%#v err=%v", plan, err)
	}
	fake := &fakeDocumentEmbedder{}
	if _, err := service.Apply(ctx, plan, PublicEmbeddingApply{Client: fake}); err == nil || fake.calls != 0 {
		t.Fatalf("unapproved apply calls=%d err=%v", fake.calls, err)
	}
	blocking := &blockingDocumentEmbedder{started: make(chan struct{}, 1), release: make(chan struct{})}
	done := make(chan struct {
		result PublicEmbeddingResult
		err    error
	}, 1)
	go func() {
		result, err := service.Apply(ctx, plan, PublicEmbeddingApply{Approved: true, Client: blocking})
		done <- struct {
			result PublicEmbeddingResult
			err    error
		}{result, err}
	}()
	select {
	case <-blocking.started:
	case <-time.After(time.Second):
		t.Fatal("fake provider did not block")
	}
	searcher, err := lexical.New(production, resolved)
	if err != nil {
		t.Fatal(err)
	}
	if result, err := searcher.Search(ctx, lexical.Request{Query: "Indexed"}); err != nil || len(result.Hits) == 0 {
		t.Fatalf("FTS blocked by provider: result=%#v err=%v", result, err)
	}
	close(blocking.release)
	completed := <-done
	result, err := completed.result, completed.err
	if err != nil || result.Succeeded != plan.PaidInputs {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	if blocking.request.Role != embedclient.DocumentRole || blocking.request.Source.SourceDimensions != 1024 || blocking.request.Source.OutputDType != "float" || blocking.request.Source.Truncation || blocking.request.Source.DocumentInputType != "document" {
		t.Fatalf("request=%#v", blocking.request)
	}
	if _, err := service.Apply(ctx, plan, PublicEmbeddingApply{Approved: true, Client: fake}); !errors.Is(err, ErrEmbeddingPlanStale) || fake.calls != 0 {
		t.Fatalf("completed plan reused: calls=%d err=%v", fake.calls, err)
	}
	again, err := service.Plan(ctx)
	if err != nil || again.Ready != again.ActiveDistinct || again.PaidInputs != 0 {
		t.Fatalf("ready plan=%#v err=%v", again, err)
	}
	db, err := sql.Open("sqlite", filepath.Join(root, ".cidx", "db", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var blob []byte
	if err := db.QueryRowContext(ctx, `SELECT blob FROM vector_cache LIMIT 1`).Scan(&blob); err != nil {
		t.Fatal(err)
	}
	values := make([]float32, 1024)
	values[0] = 1
	space, err := vector.Transformer{Spec: resolved.Embedding.TransformSpec()}.Transform(values)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := vector.EncodeInt8(space)
	if err != nil {
		t.Fatal(err)
	}
	if string(blob) != string(expected.Blob) {
		t.Fatalf("public vector differs from shared transform/codec")
	}
	var rawColumns int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM pragma_table_info('vector_cache') WHERE name LIKE '%f32%' OR name LIKE '%float%'`).Scan(&rawColumns); err != nil {
		t.Fatal(err)
	}
	if rawColumns != 0 {
		t.Fatal("production stored f32 column")
	}
	var runs int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM embedding_runs WHERE status='succeeded'`).Scan(&runs); err != nil || runs != 1 {
		t.Fatalf("runs=%d err=%v", runs, err)
	}
}
