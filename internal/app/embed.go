package app

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"cidx/internal/config"
	"cidx/internal/embed"
	"cidx/internal/embedclient"
	"cidx/internal/embedlock"
	"cidx/internal/index/canonicaltext"
	"cidx/internal/store"
	"cidx/internal/vector"
)

// PublicEmbedding is the Phase 10 application boundary. It intentionally has
// no lab field and never reads credentials. An adapter must supply both an
// explicit approval and an already-authorized embedding client for apply.
type PublicEmbedding struct {
	Production *store.ProductionStore
	Resolved   config.ResolvedConfig
}
type PublicEmbeddingOptions struct{ RetryFailed bool }
type PublicEmbeddingApply struct {
	Approved bool
	Client   embedclient.EmbeddingClient
}

var ErrEmbeddingPlanStale = errors.New("EMBEDDING_PLAN_STALE")

// PublicEmbeddingPlan deliberately exposes only review-safe summary data.
// Its authorization payload and canonical inputs remain package-private.
type PublicEmbeddingPlan struct {
	Generation      int64  `json:"generation"`
	ActiveDistinct  int    `json:"active_distinct"`
	Ready           int    `json:"ready_count"`
	SkippedTerminal int    `json:"skipped_terminal_count"`
	PaidInputs      int    `json:"paid_input_count"`
	EstimatedTokens int    `json:"estimated_tokens"`
	BatchCount      int    `json:"batch_count"`
	ManifestSHA256  string `json:"manifest_sha256"`
	RetryFailed     bool   `json:"retry_failed"`
	authorization   embeddingPlanAuthorization
}
type embeddingPlanAuthorization struct {
	source, space, storage string
	paidHashes             []string
}
type PublicEmbeddingResult struct {
	RunID        int64 `json:"run_id"`
	Requested    int   `json:"requested_count"`
	Succeeded    int   `json:"succeeded_count"`
	Failed       int   `json:"failed_count"`
	Discarded    int   `json:"discarded_count"`
	ActualTokens int   `json:"actual_tokens"`
}

func (p PublicEmbedding) Plan(ctx context.Context) (PublicEmbeddingPlan, error) {
	return p.PlanWithOptions(ctx, PublicEmbeddingOptions{})
}
func (p PublicEmbedding) PlanWithOptions(ctx context.Context, options PublicEmbeddingOptions) (PublicEmbeddingPlan, error) {
	if p.Production == nil {
		return PublicEmbeddingPlan{}, fmt.Errorf("production store is required")
	}
	if err := p.Resolved.ValidateIntegrity(); err != nil {
		return PublicEmbeddingPlan{}, err
	}
	snapshot, inputs, err := p.reconstructInputs(ctx)
	if err != nil {
		return PublicEmbeddingPlan{}, err
	}
	plan, err := embed.BuildPlan(inputs, options.RetryFailed, p.Resolved.Embedding.Request.MaxInputs, p.Resolved.Embedding.Request.MaxTotalInputBytes)
	if err != nil {
		return PublicEmbeddingPlan{}, err
	}
	return publicPlan(snapshot.Applied, plan), nil
}

func (p PublicEmbedding) reconstructInputs(ctx context.Context) (store.EmbeddingPlanningSnapshot, []embed.Input, error) {
	snapshot, err := p.Production.EmbeddingPlanningSnapshot(ctx, p.Resolved)
	if err != nil {
		return store.EmbeddingPlanningSnapshot{}, nil, err
	}
	inputs := make([]embed.Input, 0, len(snapshot.Segments))
	seen := map[string]embed.Input{}
	for _, segment := range snapshot.Segments {
		parts := make([][]byte, 0, len(segment.Projections))
		for _, projection := range segment.Projections {
			if projection.StartByte < 0 || projection.EndByte > len(segment.SourceBody) || projection.StartByte > projection.EndByte {
				return store.EmbeddingPlanningSnapshot{}, nil, fmt.Errorf("stored segment projection is invalid")
			}
			parts = append(parts, segment.SourceBody[projection.StartByte:projection.EndByte])
		}
		canonical, err := canonicaltext.Format(canonicaltext.Input{Path: segment.Path, Kind: segment.Kind, QualifiedSymbol: segment.QualifiedSymbol, Signature: segment.Signature, BodyParts: parts})
		if err != nil {
			return store.EmbeddingPlanningSnapshot{}, nil, err
		}
		if config.CanonicalInputSHA256(canonical) != segment.CanonicalInputSHA256 {
			return store.EmbeddingPlanningSnapshot{}, nil, fmt.Errorf("stored canonical input hash mismatch")
		}
		input := embed.Input{Hash: segment.CanonicalInputSHA256, Bytes: canonical, State: segment.State}
		if prior, ok := seen[input.Hash]; ok {
			if string(prior.Bytes) != string(input.Bytes) || prior.State != input.State {
				return store.EmbeddingPlanningSnapshot{}, nil, fmt.Errorf("active canonical input state is inconsistent")
			}
		} else {
			seen[input.Hash] = input
			inputs = append(inputs, input)
		}
	}
	return snapshot, inputs, nil
}

func publicPlan(applied config.AppliedProfiles, plan embed.Plan) PublicEmbeddingPlan {
	hashes := make([]string, 0, len(plan.PaidInputs))
	for _, input := range plan.PaidInputs {
		hashes = append(hashes, input.Hash)
	}
	sort.Strings(hashes)
	return PublicEmbeddingPlan{Generation: applied.ActiveGeneration, ManifestSHA256: applied.ManifestSHA256, ActiveDistinct: plan.ActiveDistinct, Ready: plan.Ready, SkippedTerminal: plan.SkippedTerminal, PaidInputs: len(plan.PaidInputs), EstimatedTokens: plan.EstimatedTokens, BatchCount: plan.BatchCount, RetryFailed: plan.RetryFailed, authorization: embeddingPlanAuthorization{source: string(applied.Fingerprints.Source), space: string(applied.Fingerprints.VectorSpace), storage: string(applied.Fingerprints.VectorStorage), paidHashes: hashes}}
}

func (p PublicEmbedding) Apply(ctx context.Context, plan PublicEmbeddingPlan, apply PublicEmbeddingApply) (result PublicEmbeddingResult, err error) {
	if !apply.Approved {
		return result, fmt.Errorf("embedding apply requires explicit approval")
	}
	if apply.Client == nil {
		return result, fmt.Errorf("embedding client is required for paid apply")
	}
	if p.Production == nil {
		return result, fmt.Errorf("production store is required")
	}
	if err := p.Resolved.ValidateIntegrity(); err != nil {
		return result, err
	}
	if plan.Generation < 0 || plan.ManifestSHA256 == "" || len(plan.authorization.paidHashes) != plan.PaidInputs || plan.authorization.source == "" || plan.authorization.space == "" || plan.authorization.storage == "" {
		return result, fmt.Errorf("invalid embedding plan")
	}
	release, err := embedlock.AcquireState(ctx, p.Production.StateRoot)
	if err != nil {
		return result, err
	}
	defer release()
	// A plan is only an approval preview. Under the document lock, derive a
	// fresh immutable snapshot and require it to be exactly the approved paid
	// set; never trust caller-owned plan bytes or silently expand a charge.
	fresh, freshPlan, err := p.currentPlan(ctx, plan.RetryFailed)
	if err != nil {
		return result, err
	}
	if !sameApprovedPlan(plan, fresh) {
		return result, ErrEmbeddingPlanStale
	}
	run := store.EmbeddingRun{Generation: fresh.Generation, ManifestSHA256: fresh.ManifestSHA256, SourceProfile: string(p.Resolved.Profiles.Fingerprints.Source), VectorSpaceProfile: string(p.Resolved.Profiles.Fingerprints.VectorSpace), StorageProfile: string(p.Resolved.Profiles.Fingerprints.VectorStorage), Planned: fresh.ActiveDistinct, Ready: fresh.Ready, Skipped: fresh.SkippedTerminal, EstimatedTokens: fresh.EstimatedTokens}
	runID, err := p.Production.StartEmbeddingRun(ctx, p.Resolved, run)
	if err != nil {
		return result, err
	}
	result.RunID = runID
	defer func() {
		status := "succeeded"
		if err == nil && (result.Failed > 0 || result.Discarded > 0) {
			if result.Succeeded > 0 {
				status = "partially_succeeded"
			} else {
				status = "failed"
			}
		} else if err != nil {
			if result.Succeeded > 0 {
				status = "partially_succeeded"
			} else if errors.Is(err, context.Canceled) {
				status = "cancelled"
			} else {
				status = "failed"
			}
		}
		run.Requested, run.Succeeded, run.Failed, run.Discarded, run.ActualTokens = result.Requested, result.Succeeded, result.Failed, result.Discarded, result.ActualTokens
		finishCtx, cancel := context.WithTimeout(context.Background(), embeddingRunFinishTimeout)
		defer cancel()
		if finishErr := p.Production.FinishEmbeddingRun(finishCtx, runID, run, status); finishErr != nil && err == nil {
			err = finishErr
		}
	}()
	expected := store.EmbeddingWriteExpectation{Generation: fresh.Generation, ManifestSHA256: fresh.ManifestSHA256}
	source := p.Resolved.Embedding.EmbeddingSourceSpec()
	transformer := vector.Transformer{Spec: p.Resolved.Embedding.TransformSpec()}
	if p.Resolved.Embedding.StorageCodec != config.StorageCodecInt8 || p.Resolved.Profiles.VectorStorage.StorageCodecID != vector.Int8CodecID {
		return result, fmt.Errorf("unsupported serving codec")
	}
	requestInputs, byKey := embeddingRequestInputs(freshPlan.PaidInputs)
	outcomes, executeErr := embed.Execute(ctx, apply.Client, source, embedclient.DocumentRole, requestInputs, embed.ExecuteOptions{
		Limits:         embed.RequestLimits{MaxInputs: p.Resolved.Embedding.Request.MaxInputs, MaxTotalBytes: p.Resolved.Embedding.Request.MaxTotalInputBytes},
		MaxConcurrency: p.Resolved.Embedding.Request.MaxConcurrency,
		AttemptTimeout: time.Duration(p.Resolved.Embedding.Request.TimeoutSeconds) * time.Second,
		MaxRetries:     p.Resolved.Embedding.Retry.MaxRetries,
		RetryWaits:     resolvedRetryWaits(p.Resolved),
	}, func(handlerCtx context.Context, outcome embed.Outcome) error {
		for i, requestInput := range outcome.Group.Inputs {
			input := byKey[requestInput.Key]
			space, transformErr := transformer.Transform(outcome.Vectors[i])
			if transformErr != nil {
				written, writeErr := p.Production.RecordCurrentEmbeddingFailure(handlerCtx, p.Resolved, expected, input.Hash, "terminal", "transform", "embedding transform rejected", outcome.Attempts)
				if writeErr != nil {
					return writeErr
				}
				if written {
					result.Failed++
				} else {
					result.Discarded++
				}
				continue
			}
			stored, encodeErr := vector.EncodeInt8(space)
			if encodeErr != nil {
				return encodeErr
			}
			published, publishErr := p.Production.PublishEmbeddedVector(handlerCtx, p.Resolved, expected, input.Hash, rawF32SHA256(outcome.Vectors[i]), stored)
			if publishErr != nil {
				return publishErr
			}
			if published {
				result.Succeeded++
			} else {
				result.Discarded++
			}
		}
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
	failureCtx, cancelFailures := context.WithTimeout(context.Background(), publicEmbeddingFailureRecordTimeout)
	defer cancelFailures()
	var failureRecordErr error
	for _, outcome := range outcomes {
		if outcome.Err == nil {
			continue
		}
		classification, errorClass, message := publicEmbeddingFailure(outcome)
		for _, requestInput := range outcome.Group.Inputs {
			input := byKey[requestInput.Key]
			written, writeErr := p.Production.RecordCurrentEmbeddingFailure(failureCtx, p.Resolved, expected, input.Hash, classification, errorClass, message, outcome.Attempts)
			if writeErr != nil {
				failureRecordErr = errors.Join(failureRecordErr, writeErr)
				continue
			}
			if written {
				result.Failed++
			} else {
				result.Discarded++
			}
		}
	}
	if executeErr != nil {
		return result, errors.Join(executeErr, failureRecordErr)
	}
	if failureRecordErr != nil {
		return result, failureRecordErr
	}
	return result, nil
}

const embeddingRunFinishTimeout = 2 * time.Second
const publicEmbeddingFailureRecordTimeout = 2 * time.Second

func embeddingRequestInputs(inputs []embed.Input) ([]embed.RequestInput, map[string]embed.Input) {
	requestInputs := make([]embed.RequestInput, len(inputs))
	byKey := make(map[string]embed.Input, len(inputs))
	for i, input := range inputs {
		requestInputs[i] = embed.RequestInput{Ordinal: i, Key: input.Hash, Bytes: input.Bytes}
		byKey[input.Hash] = input
	}
	return requestInputs, byKey
}

func resolvedRetryWaits(resolved config.ResolvedConfig) []time.Duration {
	waits := make([]time.Duration, len(resolved.Embedding.Retry.WaitSeconds))
	for i, seconds := range resolved.Embedding.Retry.WaitSeconds {
		waits[i] = time.Duration(seconds) * time.Second
	}
	return waits
}

func publicEmbeddingFailure(outcome embed.Outcome) (classification, errorClass, message string) {
	if outcome.ResponseRejected {
		return "terminal", "response_validation", "embedding response rejected"
	}
	// A provider-created attempt timeout is transient provider evidence, even
	// though it wraps context.DeadlineExceeded. Parent cancellation is a
	// separate non-terminal local outcome.
	if embed.Transient(outcome.Err) {
		return "retryable", "provider", "embedding request failed"
	}
	if errors.Is(outcome.Err, context.Canceled) || errors.Is(outcome.Err, context.DeadlineExceeded) {
		return "retryable", "cancelled", "embedding request cancelled"
	}
	return "terminal", "provider", "embedding request failed"
}

func (p PublicEmbedding) currentPlan(ctx context.Context, retryFailed bool) (PublicEmbeddingPlan, embed.Plan, error) {
	snapshot, inputs, err := p.reconstructInputs(ctx)
	if err != nil {
		return PublicEmbeddingPlan{}, embed.Plan{}, err
	}
	plan, err := embed.BuildPlan(inputs, retryFailed, p.Resolved.Embedding.Request.MaxInputs, p.Resolved.Embedding.Request.MaxTotalInputBytes)
	if err != nil {
		return PublicEmbeddingPlan{}, embed.Plan{}, err
	}
	return publicPlan(snapshot.Applied, plan), plan, nil
}

func sameApprovedPlan(old, fresh PublicEmbeddingPlan) bool {
	return old.Generation == fresh.Generation && old.ManifestSHA256 == fresh.ManifestSHA256 && old.ActiveDistinct == fresh.ActiveDistinct && old.Ready == fresh.Ready && old.SkippedTerminal == fresh.SkippedTerminal && old.PaidInputs == fresh.PaidInputs && old.EstimatedTokens == fresh.EstimatedTokens && old.BatchCount == fresh.BatchCount && old.RetryFailed == fresh.RetryFailed && old.authorization.source == fresh.authorization.source && old.authorization.space == fresh.authorization.space && old.authorization.storage == fresh.authorization.storage && stringSliceEqual(old.authorization.paidHashes, fresh.authorization.paidHashes)
}
func stringSliceEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func rawF32SHA256(values []float32) string {
	bytes := make([]byte, len(values)*4)
	for i, value := range values {
		binary.LittleEndian.PutUint32(bytes[i*4:], math.Float32bits(value))
	}
	return fmt.Sprintf("%x", sha256.Sum256(bytes))
}
