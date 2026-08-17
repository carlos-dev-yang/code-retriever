package devapp

import (
	"context"
	"testing"

	"cidx/internal/config"
)

func TestEmbeddingCaptureRejectsMutatedResolvedRequestPolicy(t *testing.T) {
	raw, err := config.DefaultRaw(1024)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := config.Resolve(raw)
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*config.ResolvedConfig){
		"max inputs":  func(c *config.ResolvedConfig) { c.Embedding.Request.MaxInputs = 129 },
		"max bytes":   func(c *config.ResolvedConfig) { c.Embedding.Request.MaxTotalInputBytes = (256 << 10) + 1 },
		"concurrency": func(c *config.ResolvedConfig) { c.Embedding.Request.MaxConcurrency = 5 },
		"timeout":     func(c *config.ResolvedConfig) { c.Embedding.Request.TimeoutSeconds = 31 },
		"waits":       func(c *config.ResolvedConfig) { c.Embedding.Retry.WaitSeconds = []int{1, 2, 3} },
	} {
		t.Run(name, func(t *testing.T) {
			mutated := resolved
			mutated.Embedding.Retry.WaitSeconds = append([]int(nil), resolved.Embedding.Retry.WaitSeconds...)
			mutate(&mutated)
			capture := EmbeddingCapture{Resolved: mutated}
			if _, err := capture.PlanWithOptions(context.Background(), CaptureOptions{}); err == nil {
				t.Fatal("plan accepted invalid resolved config")
			}
			if _, err := capture.Apply(context.Background(), CapturePlan{}); err == nil {
				t.Fatal("apply accepted invalid resolved config")
			}
		})
	}
}
