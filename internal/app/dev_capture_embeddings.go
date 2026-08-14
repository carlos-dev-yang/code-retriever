package app

import (
	"context"
	"fmt"

	"cidx/internal/config"
	"cidx/internal/embedclient"
	"cidx/internal/embedlock"
	"cidx/internal/index/canonicaltext"
	"cidx/internal/lab"
	"cidx/internal/store"
)

// DevEmbeddingCapture is the unstable Phase 13 command-use-case boundary.
// Its caller supplies an already-authorized client for apply; planning never
// constructs a provider client and therefore cannot perform paid work.
type DevEmbeddingCapture struct {
	Production *store.ProductionStore
	Lab        *lab.Store
	Resolved   config.ResolvedConfig
	Client     embedclient.EmbeddingClient
}
type DevCapturePlan struct {
	Generation     int64
	ManifestSHA256 string
	Raw            lab.CapturePlan
}
type DevCaptureOptions struct{ RetryFailed bool }

func (c DevEmbeddingCapture) Plan(ctx context.Context) (DevCapturePlan, error) {
	return c.PlanWithOptions(ctx, DevCaptureOptions{})
}
func (c DevEmbeddingCapture) PlanWithOptions(ctx context.Context, options DevCaptureOptions) (DevCapturePlan, error) {
	if c.Production == nil || c.Lab == nil {
		return DevCapturePlan{}, fmt.Errorf("production and lab stores are required")
	}
	snapshot, err := c.Production.IndexSnapshot(ctx)
	if err != nil {
		return DevCapturePlan{}, err
	}
	segments, err := c.Production.ReconciliationSegments(ctx, snapshot.Applied.ActiveGeneration)
	if err != nil {
		return DevCapturePlan{}, err
	}
	inputs := make([]lab.CaptureInput, 0, len(segments))
	for _, segment := range segments {
		parts := make([][]byte, 0, len(segment.Projections))
		for _, projection := range segment.Projections {
			if projection.StartByte < 0 || projection.EndByte > len(segment.SourceBody) || projection.StartByte > projection.EndByte {
				return DevCapturePlan{}, fmt.Errorf("stored segment projection is invalid")
			}
			parts = append(parts, segment.SourceBody[projection.StartByte:projection.EndByte])
		}
		canonical, err := canonicaltext.Format(canonicaltext.Input{Path: segment.Path, Kind: segment.Kind, QualifiedSymbol: segment.QualifiedSymbol, Signature: segment.Signature, BodyParts: parts})
		if err != nil {
			return DevCapturePlan{}, err
		}
		if hash := config.CanonicalInputSHA256(canonical); hash != segment.CanonicalInputSHA256 {
			return DevCapturePlan{}, fmt.Errorf("stored canonical input hash mismatch")
		}
		inputs = append(inputs, lab.CaptureInput{InputRecord: lab.InputRecord{InputHash: segment.CanonicalInputSHA256, CanonicalTextProfile: string(snapshot.Applied.Fingerprints.CanonicalText), CanonicalBytes: canonical, Generation: snapshot.Applied.ActiveGeneration, ManifestSHA256: snapshot.Applied.ManifestSHA256, SegmentID: segment.ID}})
	}
	collector := c.collector()
	plan, err := collector.PlanWithOptions(ctx, inputs, lab.PlanOptions{RetryFailed: options.RetryFailed})
	if err != nil {
		return DevCapturePlan{}, err
	}
	return DevCapturePlan{Generation: snapshot.Applied.ActiveGeneration, ManifestSHA256: snapshot.Applied.ManifestSHA256, Raw: plan}, nil
}
func (c DevEmbeddingCapture) Apply(ctx context.Context, plan DevCapturePlan) (lab.CaptureResult, error) {
	if c.Client == nil {
		return lab.CaptureResult{}, fmt.Errorf("embedding client is required for paid apply")
	}
	if c.Production == nil || c.Lab == nil {
		return lab.CaptureResult{}, fmt.Errorf("production and lab stores are required")
	}
	release, err := embedlock.Acquire(ctx, c.Production.Root)
	if err != nil {
		return lab.CaptureResult{}, err
	}
	defer release()
	collector := c.collector()
	collector.Client = c.Client
	return collector.Apply(ctx, plan.Raw, plan.Generation, plan.ManifestSHA256)
}
func (c DevEmbeddingCapture) collector() lab.Collector {
	return lab.Collector{Store: c.Lab, Source: c.Resolved.Embedding.EmbeddingSourceSpec(), SourceProfile: string(c.Resolved.Profiles.Fingerprints.Source), MaxInputs: c.Resolved.Embedding.Batch.MaxInputs, MaxInputTokens: c.Resolved.Embedding.Batch.MaxInputTokens, MaxRetries: c.Resolved.Embedding.Batch.MaxRetries, RequestTimeoutMS: c.Resolved.Embedding.Batch.RequestTimeoutMS}
}
