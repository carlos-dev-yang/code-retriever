# Phase 08 Raw Embedding Lab Evidence

- Status: accepted at the main-agent commit boundary.
- Paid/provider evidence: **NOT RUN**. No Voyage request, API-key read, corpus selection, corpus checkout, or embedding submission occurred.

## Executed focused checks

Main commit-boundary validation:

- `go test -count=1 -race ./internal/config ./internal/embedclient ./internal/embedlock ./internal/lab ./internal/app`
- `go vet ./internal/config ./internal/embedclient ./internal/embedlock ./internal/lab ./internal/app`
- `go build ./internal/config ./internal/embedclient ./internal/embedlock ./internal/lab ./internal/app`
- changed-Go-file formatting, production-path dependency, provider-residue, and `git diff --check` checks

Implementation-agent focused evidence:

- `go test ./internal/lab ./internal/embedclient ./internal/embedlock ./internal/app`
  - f32 round-trip and immutable-key behavior;
  - lab root isolation and schema migration fail-closed behavior;
  - v1-to-v2 migration preserves raw bytes, vector SHA-256, and snapshot-only input provenance;
  - plan performs zero provider calls, first fake-backed apply persists a raw row, and rerun plans a cache hit;
  - malformed provider dimensions reject the complete response batch before a raw row is written.
  - terminal versus transient retry bounds, default/explicit/latest-state failed-input handling, foreign-key failure integrity, atomic conflict rollback, and no-follow embed-lock rejection.
  - partial resume: an earlier one-input batch commits, a later batch fails, and the next plan reuses the committed row while retaining only the failed input as payable.
  - per-attempt deadline expiry is classified as retryable while a canceled parent context stops before another provider call.
  - injected RoundTripper proves the code-owned Voyage endpoint, bearer header, explicit document/1024/float/non-truncating request fields, and absence of `encoding_format`; invalid source specifications make zero transport calls.
- `go vet ./internal/lab ./internal/embedclient ./internal/embedlock ./internal/app`
- `go build ./internal/lab ./internal/embedclient ./internal/embedlock ./internal/app`

## Not run

- Live Voyage API request/response validation, pricing/token estimates, retries against provider statuses, and actual request IDs.
- Paid capture, corpus/evaluation work, Phase 09 materialization, MCP/CLI wiring, and production-server runtime evidence.

## Known operational boundary

An external request can succeed before its local SQLite transaction completes. A later capture reuses committed rows, but that response-loss window can cause a duplicate provider charge; it cannot be made exactly-once across Voyage and SQLite.
