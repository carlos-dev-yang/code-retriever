# Phase 09 int8-only materialization reconciliation

- State: accepted at the Phase 09 commit boundary.
- Date: 2026-08-17.
- Scope: offline product configuration, transformation, materialization,
  production-schema, search-scan, development-evaluation, and CLI fixtures.
- External actions: no corpus read, credential read, provider request, or
  network operation was performed.

## Accepted product contract

- The ordinary serving profile is `1024/int8`; `512/int8` is the only compact
  profile.
- Both profiles derive locally from the product-owned, validated 1024-f32
  document source bank using `prefix -> L2 -> int8`. Switching profiles does
  not require another Voyage document request while compatible source rows
  exist.
- The vector package has one stored-vector implementation and one exhaustive
  scorer: cidx symmetric int8. Query quantization is prepared once per scan.
- Public and development initialization expose serving dimension only. Codec
  selection is absent; omitted dimension defaults to 1024.
- Production schema v5 accepts only int8 cache rows with positive scale and
  norm. Migration from v4 copies valid int8 rows and does not copy retired
  representations.
- Current config, materialization, search, evaluation CLI, and schemas contain
  no executable retired-profile identifier or 256-dimensional serving path.
  Historical comparison evidence remains only in documentation and immutable
  artifacts.

## Implementation boundary

- Removed the retired stored-vector implementation, codec registry, and
  development codec-comparison command/artifact path.
- Made `EncodeInt8`, `PrepareInt8Query`, and `ScorePreparedInt8` the direct
  materialization and search contracts.
- Reconciled existing core fixtures to 512/1024 int8 without adding a broad new
  test matrix.
- Preserved one active serving profile, generation/manifest reproof, atomic
  complete-set publication, and the rule that production search never opens
  the source bank or evaluation database.

## Commit-boundary validation

The first focused run exposed three obsolete fixture assumptions: explicit
`storage_codec:null`, a no-op wrong-dimension mutation, and a zero-vector value
inside the new 512 prefix. Those fixtures were corrected. The final boundary
passed:

```text
go test -count=1 ./internal/config ./internal/vector ./internal/sourcebank ./internal/lab ./internal/store ./internal/devapp ./internal/app ./internal/search ./internal/devlab ./internal/cli ./internal/eval ./internal/evalcontract
go test -count=1 -race ./internal/vector ./internal/sourcebank ./internal/lab ./internal/store ./internal/devapp ./internal/app ./internal/search ./internal/devlab
go vet ./internal/config ./internal/vector ./internal/sourcebank ./internal/lab ./internal/store ./internal/devapp ./internal/app ./internal/search ./internal/devlab ./internal/cli ./internal/eval ./internal/evalcontract
go build ./...
gofmt -l cmd internal
go mod tidy -diff
git diff --check
```

Static boundary checks also found no retired vector identifier and no
256-dimensional serving configuration in current Go or evaluation-schema
paths.

## Handoff

Phase 10 must write each validated provider 1024-f32 document result to the
product source bank before publishing its int8 serving row, reuse a compatible
source row without a provider call, and retain the accepted executor,
authorization, accounting, and transactional reproof behavior.
