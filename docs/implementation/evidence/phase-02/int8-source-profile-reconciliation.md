# Phase 02 int8/source-profile reconciliation

- State: accepted
- Date: 2026-08-17
- Owner: `/root`
- Entry documentation commit: `8442749`

## Implemented boundary

- Omitted `embedding.serving_dimensions` resolves to 1024.
- The model registry accepts only serving dimensions 1024 and 512.
- `storage_codec` resolves to and validates only `int8`; explicit Binary is
  rejected before profile use.
- Vector-storage fingerprints map only an accepted int8 config to the current
  int8 codec ID.
- Evaluation run/promotion controls require source dimension 1024 and serving
  dimension 1024 or 512; 256 is rejected by both Go validation and JSON Schema.
- The impact class is named `local_rematerialize_if_source`, making the durable
  product source bank—not the historical lab raw DB—the reuse authority.
- CLI defaults were mechanically aligned to 1024/int8 so config validation and
  package build remain coherent; Phase 13 still removes the transitional codec
  flag entirely.

The exported historical Binary identifier remains temporarily only because
blocked Phase 09/11 source files still compile against it. Config resolution
cannot select it. The identifier and every encoder/scorer/evaluator reference
are deleted at their owning phase boundaries.

## Boundary validation

Executed once after the final Phase 02 changes:

```text
go test -count=1 ./internal/config ./internal/profile ./internal/evalcontract
go test -count=1 -race ./internal/config ./internal/profile ./internal/evalcontract
go vet ./internal/config ./internal/profile ./internal/evalcontract
go build ./...
jq -e . schemas/evaluation/*.json
gofmt -l internal/config internal/profile internal/evalcontract internal/cli
git diff --check
```

All checks passed; `gofmt -l` produced no paths. No provider, credential,
network, corpus, database migration, or paid operation ran. Broader package
tests are intentionally deferred to the phase that removes their historical
Binary fixtures and implementations rather than validating obsolete behavior.

## Handoff

Phase 05 consumes the 1024/512 int8-only desired profiles. Phase 08 implements
the product source bank at `<state_root>/db/embeddings.db` and vector-free lab
metadata store. Phase 09/11 remove the remaining Binary/256 executable code,
and Phase 13 removes the codec flag.
