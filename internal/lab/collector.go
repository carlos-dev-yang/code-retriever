package lab

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cidx/internal/embedclient"
)

type CaptureInput struct{ InputRecord }
type CapturePlan struct {
	inputs                                                                            []CaptureInput
	eligible                                                                          []CaptureInput
	RetryFailed                                                                       bool
	ActiveDistinct, RawHits, PaidMisses, SkippedTerminal, EstimatedTokens, BatchCount int
}
type PlanOptions struct{ RetryFailed bool }
type CaptureResult struct {
	RunID                                              int64
	Persisted, Reused, Failed, Requested, ActualTokens int
}

type Collector struct {
	Store                                                   *Store
	Client                                                  embedclient.EmbeddingClient
	Source                                                  embedclient.EmbeddingSourceSpec
	SourceProfile                                           string
	MaxInputs, MaxInputTokens, MaxRetries, RequestTimeoutMS int
}

// localInputTokenUpperBound deliberately charges one token per UTF-8 byte. It
// is a safe local batching bound, not a claimed Voyage model token limit.
func localInputTokenUpperBound(input []byte) int { return len(input) }

func (c Collector) Plan(ctx context.Context, inputs []CaptureInput) (CapturePlan, error) {
	return c.PlanWithOptions(ctx, inputs, PlanOptions{})
}
func (c Collector) PlanWithOptions(ctx context.Context, inputs []CaptureInput, options PlanOptions) (CapturePlan, error) {
	if c.Store == nil || c.SourceProfile == "" {
		return CapturePlan{}, fmt.Errorf("lab store and source profile fingerprint are required")
	}
	if err := c.Source.Validate(); err != nil {
		return CapturePlan{}, err
	}
	if c.MaxInputs <= 0 || c.MaxInputTokens <= 0 || c.MaxRetries < 0 || c.RequestTimeoutMS <= 0 {
		return CapturePlan{}, fmt.Errorf("capture max inputs is required")
	}
	unique := make([]CaptureInput, 0, len(inputs))
	seen := map[string][]byte{}
	hashes := make([]string, 0, len(inputs))
	for _, input := range inputs {
		if input.InputHash == "" || len(input.CanonicalBytes) == 0 {
			return CapturePlan{}, fmt.Errorf("canonical capture input is required")
		}
		if prior, ok := seen[input.InputHash]; ok {
			if string(prior) != string(input.CanonicalBytes) {
				return CapturePlan{}, fmt.Errorf("duplicate canonical hash has different bytes")
			}
		} else {
			seen[input.InputHash] = append([]byte(nil), input.CanonicalBytes...)
			unique = append(unique, input)
			hashes = append(hashes, input.InputHash)
		}
	}
	hits, err := c.Store.ExistingKeys(ctx, c.SourceProfile, hashes)
	if err != nil {
		return CapturePlan{}, err
	}
	terminal, err := c.Store.TerminalFailures(ctx, c.SourceProfile, hashes)
	if err != nil {
		return CapturePlan{}, err
	}
	plan := CapturePlan{inputs: unique, ActiveDistinct: len(unique), RetryFailed: options.RetryFailed}
	for _, input := range unique {
		estimate := localInputTokenUpperBound(input.CanonicalBytes)
		if estimate > c.MaxInputTokens {
			return CapturePlan{}, fmt.Errorf("canonical input exceeds local batch token budget")
		}
		if hits[input.InputHash] {
			plan.RawHits++
		} else if terminal[input.InputHash] && !options.RetryFailed {
			plan.SkippedTerminal++
		} else {
			plan.PaidMisses++
			plan.EstimatedTokens += estimate
			plan.eligible = append(plan.eligible, input)
		}
	}
	count, tokens := 0, 0
	for _, input := range plan.eligible {
		estimate := localInputTokenUpperBound(input.CanonicalBytes)
		if count == c.MaxInputs || tokens+estimate > c.MaxInputTokens {
			plan.BatchCount++
			count, tokens = 0, 0
		}
		count++
		tokens += estimate
	}
	if count > 0 {
		plan.BatchCount++
	}
	return plan, nil
}

func (c Collector) Apply(ctx context.Context, plan CapturePlan, generation int64, manifest string) (result CaptureResult, err error) {
	if c.Store == nil || c.SourceProfile == "" || c.Client == nil || c.MaxInputs <= 0 || c.MaxInputTokens <= 0 || c.MaxRetries < 0 || c.RequestTimeoutMS <= 0 {
		return result, fmt.Errorf("embedding client is required for apply")
	}
	if err := c.Source.Validate(); err != nil {
		return result, err
	}
	run, err := c.Store.StartCapture(ctx, CaptureRun{Generation: generation, ManifestSHA256: manifest, SourceProfile: c.SourceProfile, Planned: plan.ActiveDistinct, Hits: plan.RawHits, Misses: plan.PaidMisses, EstimatedTokens: plan.EstimatedTokens})
	if err != nil {
		return result, err
	}
	result.RunID, result.Reused = run, plan.RawHits
	defer func() {
		status := "complete"
		if err != nil {
			status = "failed"
		}
		if finishErr := c.Store.FinishCapture(context.Background(), run, CaptureRun{Requested: result.Requested, Hits: result.Reused, Misses: plan.PaidMisses, Persisted: result.Persisted, Failed: result.Failed, ActualTokens: result.ActualTokens}, status); finishErr != nil && err == nil {
			err = finishErr
		}
	}()
	hashes := make([]string, 0, len(plan.eligible))
	for _, input := range plan.eligible {
		hashes = append(hashes, input.InputHash)
	}
	existing, err := c.Store.ExistingKeys(ctx, c.SourceProfile, hashes)
	if err != nil {
		return result, err
	}
	terminal, err := c.Store.TerminalFailures(ctx, c.SourceProfile, hashes)
	if err != nil {
		return result, err
	}
	misses := make([]CaptureInput, 0, plan.PaidMisses)
	for _, input := range plan.eligible {
		if !existing[input.InputHash] && (plan.RetryFailed || !terminal[input.InputHash]) {
			misses = append(misses, input)
		}
	}
	for start := 0; start < len(misses); {
		end, tokens := start, 0
		for end < len(misses) && end-start < c.MaxInputs {
			estimate := localInputTokenUpperBound(misses[end].CanonicalBytes)
			if tokens+estimate > c.MaxInputTokens {
				break
			}
			tokens += estimate
			end++
		}
		if end == start {
			return result, fmt.Errorf("canonical input exceeds local batch token budget")
		}
		batch := misses[start:end]
		texts := make([]string, len(batch))
		for i, input := range batch {
			texts[i] = string(input.CanonicalBytes)
			if err = c.Store.PutInput(ctx, input.InputRecord); err != nil {
				return result, err
			}
		}
		request := embedclient.EmbeddingRequest{Source: c.Source, Role: embedclient.DocumentRole, Inputs: texts}
		var response embedclient.EmbeddingResponse
		var callErr error
		attempts := 0
		for attempt := 0; attempt <= c.MaxRetries; attempt++ {
			attempts++
			result.Requested += len(batch)
			attemptCtx, cancel := context.WithTimeout(ctx, time.Duration(c.RequestTimeoutMS)*time.Millisecond)
			response, callErr = c.Client.Embed(attemptCtx, request)
			cancel()
			if errors.Is(callErr, context.DeadlineExceeded) && ctx.Err() == nil {
				callErr = embedclient.ProviderError{Class: "timeout", Retryable: true}
			}
			if ctx.Err() != nil {
				return result, ctx.Err()
			}
			if callErr == nil || !embedclient.IsRetryable(callErr) {
				break
			}
		}
		if callErr != nil {
			result.Failed += len(batch)
			classification := "terminal"
			if embedclient.IsRetryable(callErr) {
				classification = "retryable"
			}
			for _, input := range batch {
				if recordErr := c.Store.RecordFailure(ctx, run, RawEmbeddingKey{SourceProfile: c.SourceProfile, InputHash: input.InputHash}, classification, "provider", "embedding request failed", attempts); recordErr != nil {
					return result, recordErr
				}
			}
			return result, callErr
		}
		vectors, validateErr := embedclient.ValidateResponse(request, response)
		if validateErr != nil {
			result.Failed += len(batch)
			for _, input := range batch {
				if recordErr := c.Store.RecordFailure(ctx, run, RawEmbeddingKey{SourceProfile: c.SourceProfile, InputHash: input.InputHash}, "terminal", "response_validation", "embedding response rejected", 1); recordErr != nil {
					return result, recordErr
				}
			}
			return result, validateErr
		}
		result.ActualTokens += response.TotalTokens
		raws := make([]DocumentRaw, 0, len(batch))
		for i, input := range batch {
			vector, vectorErr := NewF32Vector(vectors[i], c.Source.SourceDimensions)
			if vectorErr != nil {
				return result, vectorErr
			}
			raws = append(raws, DocumentRaw{SourceProfile: c.SourceProfile, InputHash: input.InputHash, RequestedModel: c.Source.Model, ResponseModel: response.Model, RequestID: response.RequestID, Vector: vector})
		}
		if err = c.Store.PutDocumentSources(ctx, raws, c.Source.SourceDimensions); err != nil {
			return result, err
		}
		result.Persisted += len(raws)
		start = end
	}
	return result, nil
}
