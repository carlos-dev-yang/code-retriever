# Phase 09 Vector Materialization Evidence

> Historical pre-int8-only evidence. The current product authority is
> [`int8-only-materialization-reconciliation.md`](int8-only-materialization-reconciliation.md).

- State: done; main-agent commit-boundary validation passed.
- Scope: synthetic/offline f32 only. No provider client, API key, network, corpus, or paid request was used.

## Implemented contract

- The shared transform is source f32 validation, leading-prefix reduction, L2 normalization, then the selected cidx `binary` or `int8` codec.
- The lab v3 schema keeps search-invisible materialization runs and variants. Its v2-to-v3 migration preserves raw/capture data.
- The production v2 schema retains vector rows and adds source-profile, vector-space-profile, raw-SHA, and materialization-time lineage. Rows without valid lineage fail readiness until rebuilt.
- Publication rechecks active generation, manifest, source/vector-space/storage profile, active serving key, and exact active-hash coverage, then replaces the one current-profile set inside a single SQLite transaction.
- The materialization application service is the only path opening both lab and production stores. It uses the shared embed lock and has no provider dependency.

## Implementation-agent focused checks

```text
go test -count=1 ./internal/config ./internal/vector ./internal/lab ./internal/store ./internal/app
go test -count=1 -race ./internal/config ./internal/vector ./internal/lab ./internal/store ./internal/app
go vet ./internal/config ./internal/vector ./internal/lab ./internal/store ./internal/app
go build ./internal/config ./internal/vector ./internal/lab ./internal/store ./internal/app
go list -deps ./internal/store  # no cidx/internal/lab
go test -count=1 -run TestSyntheticCodecRankObservation -v ./internal/vector
```

Passed. The store test covers incomplete-coverage rejection, old-set preservation, pinned old-reader/new-reader visibility, and generation-change rejection. Migration tests cover exact historical lab v2-to-v3 raw/capture/evaluation/materialization preservation and exact historical production v1-to-v2 metadata/vector preservation. Lab run-transition tests derive and retain exact staged coverage and deterministic candidate checksum evidence. The application integration test covers plan, mismatched-root and missing-raw rejection, staging, activation, no-f32 production storage, and active-key recheck. Config tests reject post-resolution mutation. Existing lab/vector tests continue to cover f32 checksum/finite validation and deterministic binary/int8 codec contracts. The [synthetic codec observation](synthetic-codec-observation.md) records 256/512 binary/int8 ranking behavior without creating a release threshold.

## Not run

- Real Voyage/network/API-key/corpus/paid work.
- Phase 13 CLI/MCP wiring, query vectors, hybrid search, HNSW, raw garbage collection, and evaluation metrics.

## Main-agent commit-boundary validation

The main agent reviewed the complete Phase 09 diff, including the real historical-schema migrations, resolved-profile integrity checks, active-key snapshot, batched raw reads, immutable lab evidence, production lineage/readiness, and atomic publication. It then ran:

```text
go test -count=1 -race ./internal/config ./internal/vector ./internal/lab ./internal/store ./internal/app ./internal/index
go vet ./internal/config ./internal/vector ./internal/lab ./internal/store ./internal/app ./internal/index
go build ./internal/config ./internal/vector ./internal/lab ./internal/store ./internal/app ./internal/index
go test -count=1 -run TestSyntheticCodecRankObservation -v ./internal/vector
gofmt -l internal/config internal/vector internal/lab internal/store internal/app
go list -deps ./internal/store
go list -deps ./internal/index
go mod tidy -diff
git diff --check
```

All checks passed. Neither production `store` nor `index` depends on `internal/lab`. The synthetic observation reproduced the committed 256/512 binary/int8 score-error and pair-inversion values. No external provider, credential, corpus, or paid operation was used.
