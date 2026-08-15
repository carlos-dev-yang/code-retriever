package lab

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"cidx/internal/embed"
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

// Collector is the development-only persistence adapter for embed.Execute.
// Request execution policy is injected from the one validated ResolvedConfig.
type Collector struct {
	Store          *Store
	Client         embedclient.EmbeddingClient
	Source         embedclient.EmbeddingSourceSpec
	SourceProfile  string
	RequestLimits  embed.RequestLimits
	MaxConcurrency int
	AttemptTimeout time.Duration
	MaxRetries     int
	RetryWaits     []time.Duration
	Wait           embed.Waiter // test-only timing seam; production leaves it nil.
	putSources     func(context.Context, []DocumentRaw, int) error
}

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
	if err := c.validateExecutionPolicy(); err != nil {
		return CapturePlan{}, err
	}
	unique := make([]CaptureInput, 0, len(inputs))
	seen := map[string][]byte{}
	hashes := make([]string, 0, len(inputs))
	for _, input := range inputs {
		if input.InputHash == "" || len(input.CanonicalBytes) == 0 || !utf8.Valid(input.CanonicalBytes) {
			return CapturePlan{}, fmt.Errorf("canonical capture input is required")
		}
		if prior, ok := seen[input.InputHash]; ok {
			if string(prior) != string(input.CanonicalBytes) {
				return CapturePlan{}, fmt.Errorf("duplicate canonical hash has different bytes")
			}
			continue
		}
		seen[input.InputHash] = append([]byte(nil), input.CanonicalBytes...)
		unique = append(unique, input)
		hashes = append(hashes, input.InputHash)
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
		if hits[input.InputHash] {
			plan.RawHits++
		} else if terminal[input.InputHash] && !options.RetryFailed {
			plan.SkippedTerminal++
		} else {
			plan.PaidMisses++
			plan.EstimatedTokens += embed.ConservativeInputTokenUpperBound(input.CanonicalBytes)
			plan.eligible = append(plan.eligible, input)
		}
	}
	groups, err := captureGroups(plan.eligible, c.RequestLimits)
	if err != nil {
		return CapturePlan{}, err
	}
	plan.BatchCount = len(groups)
	return plan, nil
}

func (c Collector) Apply(ctx context.Context, plan CapturePlan, generation int64, manifest string) (result CaptureResult, err error) {
	if c.Store == nil || c.SourceProfile == "" || c.Client == nil {
		return result, fmt.Errorf("embedding client is required for apply")
	}
	if err := c.Source.Validate(); err != nil {
		return result, err
	}
	if err := c.validateExecutionPolicy(); err != nil {
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
			if err := c.Store.PutInput(ctx, input.InputRecord); err != nil {
				return result, err
			}
			misses = append(misses, input)
		}
	}
	requestInputs, byKey := requestInputs(misses)
	outcomes, executeErr := embed.Execute(ctx, c.Client, c.Source, embedclient.DocumentRole, requestInputs, embed.ExecuteOptions{
		Limits: c.RequestLimits, MaxConcurrency: c.MaxConcurrency, AttemptTimeout: c.AttemptTimeout,
		MaxRetries: c.MaxRetries, RetryWaits: append([]time.Duration(nil), c.RetryWaits...), Wait: c.Wait,
	}, func(handlerCtx context.Context, outcome embed.Outcome) error {
		raws := make([]DocumentRaw, 0, len(outcome.Group.Inputs))
		for i, requestInput := range outcome.Group.Inputs {
			input := byKey[requestInput.Key]
			vector, vectorErr := NewF32Vector(outcome.Vectors[i], c.Source.SourceDimensions)
			if vectorErr != nil {
				return vectorErr
			}
			raws = append(raws, DocumentRaw{SourceProfile: c.SourceProfile, InputHash: input.InputHash, RequestedModel: c.Source.Model, ResponseModel: outcome.Response.Model, RequestID: outcome.Response.RequestID, Vector: vector})
		}
		if err := c.putDocumentSources(handlerCtx, raws); err != nil {
			return err
		}
		result.Persisted += len(raws)
		return nil
	})
	for _, outcome := range outcomes {
		result.Requested += len(outcome.Group.Inputs) * outcome.Attempts
		if outcome.Err == nil {
			result.ActualTokens += outcome.Response.TotalTokens
		}
	}
	var handlerErr *embed.HandlerError
	if errors.As(executeErr, &handlerErr) {
		return result, executeErr
	}
	failureCtx, cancelFailures := context.WithTimeout(context.Background(), captureFailureRecordTimeout)
	defer cancelFailures()
	var failureRecordErr error
	for _, outcome := range outcomes {
		if outcome.Err == nil {
			continue
		}
		result.Failed += len(outcome.Group.Inputs)
		classification := "terminal"
		errorClass, message := "provider", "embedding request failed"
		if outcome.ResponseRejected {
			errorClass, message = "response_validation", "embedding response rejected"
		} else if embed.Transient(outcome.Err) {
			classification = "retryable"
		} else if errors.Is(outcome.Err, context.Canceled) || errors.Is(outcome.Err, context.DeadlineExceeded) {
			classification, errorClass, message = "retryable", "cancelled", "embedding request cancelled"
		}
		for _, requestInput := range outcome.Group.Inputs {
			input := byKey[requestInput.Key]
			if recordErr := c.Store.RecordFailure(failureCtx, run, RawEmbeddingKey{SourceProfile: c.SourceProfile, InputHash: input.InputHash}, classification, errorClass, message, outcome.Attempts); recordErr != nil && failureRecordErr == nil {
				failureRecordErr = recordErr
			}
		}
	}
	if executeErr != nil {
		return result, joinFailureRecordError(executeErr, failureRecordErr)
	}
	for _, outcome := range outcomes {
		if outcome.Err != nil {
			return result, joinFailureRecordError(outcome.Err, failureRecordErr)
		}
	}
	if failureRecordErr != nil {
		return result, failureRecordErr
	}
	return result, nil
}

const captureFailureRecordTimeout = 2 * time.Second

func joinFailureRecordError(primary, failureRecord error) error {
	if failureRecord == nil {
		return primary
	}
	return errors.Join(primary, failureRecord)
}

func (c Collector) putDocumentSources(ctx context.Context, raws []DocumentRaw) error {
	if c.putSources != nil {
		return c.putSources(ctx, raws, c.Source.SourceDimensions)
	}
	return c.Store.PutDocumentSources(ctx, raws, c.Source.SourceDimensions)
}

func (c Collector) validateExecutionPolicy() error {
	if c.RequestLimits.MaxInputs <= 0 || c.RequestLimits.MaxTotalBytes <= 0 || c.MaxConcurrency <= 0 || c.AttemptTimeout <= 0 || c.MaxRetries < 0 || len(c.RetryWaits) < c.MaxRetries {
		return fmt.Errorf("capture execution policy is required")
	}
	return nil
}

func captureGroups(inputs []CaptureInput, limits embed.RequestLimits) ([]embed.RequestGroup, error) {
	request, _ := requestInputs(inputs)
	return embed.Group(request, limits)
}

func requestInputs(inputs []CaptureInput) ([]embed.RequestInput, map[string]CaptureInput) {
	request := make([]embed.RequestInput, len(inputs))
	byKey := make(map[string]CaptureInput, len(inputs))
	for i, input := range inputs {
		request[i] = embed.RequestInput{Ordinal: i, Key: input.InputHash, Bytes: input.CanonicalBytes}
		byKey[input.InputHash] = input
	}
	return request, byKey
}
