// Package embed contains the provider-neutral public document embedding
// orchestration primitives. It has no lab dependency.
package embed

// ConservativeInputTokenUpperBound deliberately charges one token per UTF-8
// byte. It is a safe local batching bound, not a claimed Voyage model limit.
func ConservativeInputTokenUpperBound(input []byte) int { return len(input) }
