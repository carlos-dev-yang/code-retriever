# Phase 10 Embedding Orchestration Evidence

- Revision 4 reconciliation evidence: [revision-4.md](revision-4.md)
- Historical state: accepted at the pre-Revision-4 main-agent commit boundary; it is not evidence of the current request-policy reconciliation.
- No credential was read and no network, provider, corpus, or paid operation was used.

## Executed focused checks

```text
go test -count=1 ./internal/app ./internal/embed ./internal/lab ./internal/store
go test -count=1 -race ./internal/app ./internal/embed ./internal/store
go vet ./internal/app ./internal/embed ./internal/lab ./internal/store
go build ./internal/app ./internal/embed ./internal/lab ./internal/store
go list -deps ./internal/embed ./internal/store
git diff --check
```

The fake-provider checks prove free planning makes no provider call, apply
requires approval, document request fields are exact, production output matches
the shared transform/codec, FTS remains available while a fake provider blocks,
completed plans cannot be reused, and stale plans fail before provider calls.
They also cover multi-batch partial completion, active-key removal during a
blocked request, and generation change during a blocked request.

The main agent reran the focused race suite, vet, build, formatting,
production dependency-boundary inspection, and diff validation before
acceptance. All passed.

`requested_count` records provider input attempts, including retries.
`succeeded_count`, `failed_count`, and `discarded_count` record distinct
approved input outcomes.

## Not run

Live Voyage/API-key behavior, CLI/MCP wiring, query/hybrid integration,
corpus/evaluation work, broad validation, and load testing.
