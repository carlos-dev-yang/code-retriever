package embed

import (
	"fmt"
	"sort"

	"cidx/internal/store"
)

// Input is an already reconstructed and hash-verified active canonical input.
// The canonical bytes remain in memory only for the duration of a plan/apply.
type Input struct {
	Hash  string
	Bytes []byte
	State store.EmbeddingState
}

type Plan struct {
	Inputs          []Input
	PaidInputs      []Input
	RetryFailed     bool
	ActiveDistinct  int
	Ready           int
	SkippedTerminal int
	EstimatedTokens int
	BatchCount      int
}

func BuildPlan(inputs []Input, retryFailed bool, maxInputs, maxTokens int) (Plan, error) {
	if maxInputs <= 0 || maxTokens <= 0 {
		return Plan{}, fmt.Errorf("embedding batch limits are required")
	}
	byHash := make(map[string]Input, len(inputs))
	for _, input := range inputs {
		if input.Hash == "" || len(input.Bytes) == 0 {
			return Plan{}, fmt.Errorf("canonical embedding input is required")
		}
		if prior, ok := byHash[input.Hash]; ok && string(prior.Bytes) != string(input.Bytes) {
			return Plan{}, fmt.Errorf("duplicate canonical hash has different bytes")
		}
		byHash[input.Hash] = Input{Hash: input.Hash, Bytes: append([]byte(nil), input.Bytes...), State: input.State}
	}
	hashes := make([]string, 0, len(byHash))
	for hash := range byHash {
		hashes = append(hashes, hash)
	}
	sort.Strings(hashes)
	plan := Plan{RetryFailed: retryFailed, ActiveDistinct: len(hashes)}
	for _, hash := range hashes {
		input := byHash[hash]
		plan.Inputs = append(plan.Inputs, input)
		estimate := ConservativeInputTokenUpperBound(input.Bytes)
		if estimate > maxTokens {
			return Plan{}, fmt.Errorf("canonical input exceeds local batch token budget")
		}
		switch input.State {
		case store.EmbeddingReady:
			plan.Ready++
		case store.EmbeddingFailed:
			if !retryFailed {
				plan.SkippedTerminal++
				continue
			}
			fallthrough
		case store.EmbeddingPending:
			plan.PaidInputs = append(plan.PaidInputs, input)
			plan.EstimatedTokens += estimate
		default:
			return Plan{}, fmt.Errorf("unknown embedding state")
		}
	}
	count, tokens := 0, 0
	for _, input := range plan.PaidInputs {
		estimate := ConservativeInputTokenUpperBound(input.Bytes)
		if count == maxInputs || tokens+estimate > maxTokens {
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

func Batches(inputs []Input, maxInputs, maxTokens int) ([][]Input, error) {
	if maxInputs <= 0 || maxTokens <= 0 {
		return nil, fmt.Errorf("embedding batch limits are required")
	}
	var out [][]Input
	var batch []Input
	tokens := 0
	for _, input := range inputs {
		estimate := ConservativeInputTokenUpperBound(input.Bytes)
		if estimate > maxTokens {
			return nil, fmt.Errorf("canonical input exceeds local batch token budget")
		}
		if len(batch) == maxInputs || tokens+estimate > maxTokens {
			out = append(out, batch)
			batch = nil
			tokens = 0
		}
		batch = append(batch, input)
		tokens += estimate
	}
	if len(batch) > 0 {
		out = append(out, batch)
	}
	return out, nil
}
