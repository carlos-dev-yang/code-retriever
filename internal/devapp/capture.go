// Package devapp owns unstable lab-backed application use cases. Production
// app, MCP, and serving packages do not import it.
package devapp

import (
	"context"
	"fmt"
	"time"

	"cidx/internal/config"
	"cidx/internal/embed"
	"cidx/internal/embedclient"
	"cidx/internal/embedlock"
	"cidx/internal/index/canonicaltext"
	"cidx/internal/lab"
	"cidx/internal/store"
)

type EmbeddingCapture struct {
	Production *store.ProductionStore
	Lab        *lab.Store
	Resolved   config.ResolvedConfig
	Client     embedclient.EmbeddingClient
}
type CapturePlan struct {
	Generation     int64
	ManifestSHA256 string
	Raw            lab.CapturePlan
}
type CaptureOptions struct{ RetryFailed bool }

func (c EmbeddingCapture) Plan(ctx context.Context) (CapturePlan, error) {
	return c.PlanWithOptions(ctx, CaptureOptions{})
}
func (c EmbeddingCapture) PlanWithOptions(ctx context.Context, options CaptureOptions) (CapturePlan, error) {
	if err := c.Resolved.ValidateIntegrity(); err != nil {
		return CapturePlan{}, err
	}
	if c.Production == nil || c.Lab == nil {
		return CapturePlan{}, fmt.Errorf("production and lab stores are required")
	}
	snapshot, err := c.Production.IndexSnapshot(ctx)
	if err != nil {
		return CapturePlan{}, err
	}
	segments, err := c.Production.ReconciliationSegments(ctx, snapshot.Applied.ActiveGeneration)
	if err != nil {
		return CapturePlan{}, err
	}
	inputs := make([]lab.CaptureInput, 0, len(segments))
	for _, segment := range segments {
		parts := make([][]byte, 0, len(segment.Projections))
		for _, projection := range segment.Projections {
			if projection.StartByte < 0 || projection.EndByte > len(segment.SourceBody) || projection.StartByte > projection.EndByte {
				return CapturePlan{}, fmt.Errorf("stored segment projection is invalid")
			}
			parts = append(parts, segment.SourceBody[projection.StartByte:projection.EndByte])
		}
		canonical, err := canonicaltext.Format(canonicaltext.Input{Path: segment.Path, Kind: segment.Kind, QualifiedSymbol: segment.QualifiedSymbol, Signature: segment.Signature, BodyParts: parts})
		if err != nil {
			return CapturePlan{}, err
		}
		if config.CanonicalInputSHA256(canonical) != segment.CanonicalInputSHA256 {
			return CapturePlan{}, fmt.Errorf("stored canonical input hash mismatch")
		}
		inputs = append(inputs, lab.CaptureInput{InputRecord: lab.InputRecord{InputHash: segment.CanonicalInputSHA256, CanonicalTextProfile: string(snapshot.Applied.Fingerprints.CanonicalText), CanonicalBytes: canonical, Generation: snapshot.Applied.ActiveGeneration, ManifestSHA256: snapshot.Applied.ManifestSHA256, SegmentID: segment.ID}})
	}
	plan, err := c.collector().PlanWithOptions(ctx, inputs, lab.PlanOptions{RetryFailed: options.RetryFailed})
	if err != nil {
		return CapturePlan{}, err
	}
	return CapturePlan{Generation: snapshot.Applied.ActiveGeneration, ManifestSHA256: snapshot.Applied.ManifestSHA256, Raw: plan}, nil
}
func (c EmbeddingCapture) Apply(ctx context.Context, plan CapturePlan) (lab.CaptureResult, error) {
	if err := c.Resolved.ValidateIntegrity(); err != nil {
		return lab.CaptureResult{}, err
	}
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
func (c EmbeddingCapture) collector() lab.Collector {
	waits := make([]time.Duration, len(c.Resolved.Embedding.Retry.WaitSeconds))
	for i, seconds := range c.Resolved.Embedding.Retry.WaitSeconds {
		waits[i] = time.Duration(seconds) * time.Second
	}
	return lab.Collector{
		Store: c.Lab, Source: c.Resolved.Embedding.EmbeddingSourceSpec(), SourceProfile: string(c.Resolved.Profiles.Fingerprints.Source),
		RequestLimits:  embed.RequestLimits{MaxInputs: c.Resolved.Embedding.Request.MaxInputs, MaxTotalBytes: c.Resolved.Embedding.Request.MaxTotalInputBytes},
		MaxConcurrency: c.Resolved.Embedding.Request.MaxConcurrency, AttemptTimeout: time.Duration(c.Resolved.Embedding.Request.TimeoutSeconds) * time.Second,
		MaxRetries: c.Resolved.Embedding.Retry.MaxRetries, RetryWaits: waits,
	}
}
