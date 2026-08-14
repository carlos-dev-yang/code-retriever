package embed

import (
	"testing"

	"cidx/internal/store"
)

func TestBuildPlanExcludesReadyAndTerminalUnlessRetried(t *testing.T) {
	inputs := []Input{{Hash: "ready", Bytes: []byte("a"), State: store.EmbeddingReady}, {Hash: "failed", Bytes: []byte("bb"), State: store.EmbeddingFailed}, {Hash: "pending", Bytes: []byte("ccc"), State: store.EmbeddingPending}}
	plan, err := BuildPlan(inputs, false, 2, 10)
	if err != nil || plan.ActiveDistinct != 3 || plan.Ready != 1 || plan.SkippedTerminal != 1 || len(plan.PaidInputs) != 1 || plan.PaidInputs[0].Hash != "pending" || plan.EstimatedTokens != 3 {
		t.Fatalf("plan=%#v err=%v", plan, err)
	}
	retry, err := BuildPlan(inputs, true, 2, 10)
	if err != nil || len(retry.PaidInputs) != 2 || retry.PaidInputs[0].Hash != "failed" || retry.PaidInputs[1].Hash != "pending" || retry.BatchCount != 1 {
		t.Fatalf("retry=%#v err=%v", retry, err)
	}
}
