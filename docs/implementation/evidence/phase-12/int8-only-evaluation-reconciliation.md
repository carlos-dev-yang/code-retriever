# Phase 12 int8-only evaluation reconciliation

- State: corpus-independent boundary accepted; official Phase 12 evaluation
  and promotion remain blocked.
- Date: 2026-08-17.
- External actions: no credential read, corpus read, provider request, network
  operation, paid query, metric run, or promotion action was performed.

## Accepted evaluation matrix

- One run evaluates the active fixed-int8 product profile at either default
  1024 dimensions or explicit compact 512 dimensions.
- The immutable experiment artifact now accepts both supported dimensions and
  records the actual `serving_dimensions` together with `storage_codec=int8`.
- The same request-local query vector feeds serving-f32 and active-int8 arms.
  Serving-f32 is a nonpersistent fidelity reference, not a production storage
  option.
- FTS-only, f32 dense, int8 dense, provider union, both RRF arms, and both lane
  ablations remain separate. Human usefulness and int8/f32 fidelity remain
  separate metrics.
- Current code cannot generate, activate, or score Binary/256 candidates.
  Their prior reports remain immutable historical evidence only.

The source bank remains document-only and read-only from the development
evaluation adapter. Runtime search remains independent of it. Switching the
active document target from 1024/int8 to 512/int8 can reuse compatible
1024-f32 document source rows without a document provider request; evaluation
queries remain fresh and nonpersistent per run.

## Commit-boundary validation

```text
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 ./internal/devlab ./internal/eval ./internal/evalcontract ./internal/search ./internal/lab ./internal/sourcebank
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 -race ./internal/devlab ./internal/eval ./internal/evalcontract ./internal/search ./internal/lab ./internal/sourcebank
env -u VOYAGE_API_KEY GOPROXY=off go vet ./internal/devlab ./internal/eval ./internal/evalcontract ./internal/search ./internal/lab ./internal/sourcebank
env -u VOYAGE_API_KEY GOPROXY=off go build ./...
gofmt -l cmd internal
jq -e '."$defs".controls.properties.serving_dimensions.enum == [1024,512]' schemas/evaluation/run-manifest.schema.json
jq -e '."$defs".controls.properties.serving_dimensions.enum == [1024,512]' schemas/evaluation/promotion-contract.schema.json
go mod tidy -diff
git diff --check
```

All passed. Static inspection found no retired comparison/codec identifier in
the current evaluation, lab, search, or schema execution surface.

## Remaining external gate

This boundary proves adapter/profile consistency only. The frozen-label,
current-corpus, provider-query, per-language/cohort, confirmation, and
`scope=core_retrieval` promotion evidence described by Phase 12 still must be
run separately after the user-controlled inputs are ready.

## Handoff

Phase 13 may expose the already validated 1024-default/512-optional int8
configuration and provider-free document rematerialization path. It must not
add a codec selector or evaluation-only search implementation.
