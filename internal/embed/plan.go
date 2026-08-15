package embed

import (
	"fmt"
	"sort"
	"unicode/utf8"

	"cidx/internal/store"
)

// Input is an already reconstructed and hash-verified active canonical input.
// The canonical bytes remain in memory only for the duration of a plan/apply.
type Input struct {
	Hash  string
	Bytes []byte
	State store.EmbeddingState
}

// RequestInput is provider-neutral canonical request data. Ordinal is assigned
// by the caller and remains stable through grouping and execution.
type RequestInput struct {
	Ordinal int
	Key     string
	Bytes   []byte
}

type RequestGroup struct {
	Ordinal    int
	Inputs     []RequestInput
	TotalBytes int
}

type RequestLimits struct {
	MaxInputs     int
	MaxTotalBytes int
}

// Group validates all canonical input before a provider call and groups in
// caller order. Limits are literal UTF-8 byte limits, never token estimates.
func Group(inputs []RequestInput, limits RequestLimits) ([]RequestGroup, error) {
	if limits.MaxInputs <= 0 || limits.MaxTotalBytes <= 0 {
		return nil, fmt.Errorf("embedding request limits are required")
	}
	groups := make([]RequestGroup, 0)
	seenOrdinals := make(map[int]struct{}, len(inputs))
	seenKeys := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		if input.Ordinal < 0 || input.Key == "" || len(input.Bytes) == 0 || !utf8.Valid(input.Bytes) {
			return nil, fmt.Errorf("canonical embedding input is invalid")
		}
		if _, ok := seenOrdinals[input.Ordinal]; ok {
			return nil, fmt.Errorf("duplicate embedding input ordinal")
		}
		if _, ok := seenKeys[input.Key]; ok {
			return nil, fmt.Errorf("duplicate embedding input key")
		}
		seenOrdinals[input.Ordinal] = struct{}{}
		seenKeys[input.Key] = struct{}{}
		if len(input.Bytes) > limits.MaxTotalBytes {
			return nil, fmt.Errorf("canonical input exceeds request byte limit")
		}
		if len(groups) == 0 || len(groups[len(groups)-1].Inputs) == limits.MaxInputs || groups[len(groups)-1].TotalBytes+len(input.Bytes) > limits.MaxTotalBytes {
			groups = append(groups, RequestGroup{Ordinal: len(groups)})
		}
		group := &groups[len(groups)-1]
		group.Inputs = append(group.Inputs, RequestInput{Ordinal: input.Ordinal, Key: input.Key, Bytes: append([]byte(nil), input.Bytes...)})
		group.TotalBytes += len(input.Bytes)
	}
	return groups, nil
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

// BuildPlan keeps token estimation as a diagnostic only. Its grouping counts
// use the actual request-byte policy shared by all document executors.
func BuildPlan(inputs []Input, retryFailed bool, maxInputs, maxTotalBytes int) (Plan, error) {
	if maxInputs <= 0 || maxTotalBytes <= 0 {
		return Plan{}, fmt.Errorf("embedding request limits are required")
	}
	byHash := make(map[string]Input, len(inputs))
	orderedHashes := make([]string, 0, len(inputs))
	for _, input := range inputs {
		if input.Hash == "" || len(input.Bytes) == 0 || !utf8.Valid(input.Bytes) {
			return Plan{}, fmt.Errorf("canonical embedding input is required")
		}
		if prior, ok := byHash[input.Hash]; ok {
			if string(prior.Bytes) != string(input.Bytes) {
				return Plan{}, fmt.Errorf("duplicate canonical hash has different bytes")
			}
			continue
		}
		byHash[input.Hash] = Input{Hash: input.Hash, Bytes: append([]byte(nil), input.Bytes...), State: input.State}
		orderedHashes = append(orderedHashes, input.Hash)
	}
	sort.Strings(orderedHashes)
	plan := Plan{RetryFailed: retryFailed, ActiveDistinct: len(orderedHashes)}
	for _, hash := range orderedHashes {
		input := byHash[hash]
		plan.Inputs = append(plan.Inputs, input)
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
			plan.EstimatedTokens += ConservativeInputTokenUpperBound(input.Bytes)
		default:
			return Plan{}, fmt.Errorf("unknown embedding state")
		}
	}
	groups, err := requestGroups(plan.PaidInputs, maxInputs, maxTotalBytes)
	if err != nil {
		return Plan{}, err
	}
	plan.BatchCount = len(groups)
	return plan, nil
}

func requestGroups(inputs []Input, maxInputs, maxTotalBytes int) ([]RequestGroup, error) {
	requestInputs := make([]RequestInput, len(inputs))
	for i, input := range inputs {
		requestInputs[i] = RequestInput{Ordinal: i, Key: input.Hash, Bytes: input.Bytes}
	}
	return Group(requestInputs, RequestLimits{MaxInputs: maxInputs, MaxTotalBytes: maxTotalBytes})
}
