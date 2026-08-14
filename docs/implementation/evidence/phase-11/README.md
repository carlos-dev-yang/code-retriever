# Phase 11 Vector and Hybrid Search Evidence

- State: accepted at the Phase 11 commit boundary by the main agent.
- Date: 2026-08-15
- Scope: transport-independent core search only. No MCP/CLI adapter, `read_span`, evaluation runner, real provider, API key, network, corpus, lab runtime access, or paid action was used.

## Implemented evidence

- Explicit FTS and every fallback use the lexical-only production snapshot. It reads active metadata plus ordered FTS candidates and their indexed parent bodies; it does not query `embedding_segments`, `vector_cache`, coverage, or vector payloads.
- Hybrid preflight uses one read-only transaction. It reads applied metadata and a deterministic `DISTINCT` active canonical-key CTE, validates every present current-profile stored row with the production codec validator, and performs no FTS/chunk/body/projection read. Missing vector rows are ordinary partial coverage; no valid row yields `NO_VALID_DOCUMENT_VECTORS`; any present invalid row yields `VECTOR_SNAPSHOT_INVALID`; both occur before a provider call.
- The post-query hybrid snapshot is one pinned transaction. It materializes FTS candidates, deduplicated active parent chunks/bodies, deduplicated active canonical vectors, and segment references. It detects generation/manifest/profile movement and drops the transient query vector.
- Query input uses code-owned `QueryTextFormatVersion=1`, included in the runtime-only serving-policy fingerprint and returned in the core response. The v1 formatter strictly accepts nonempty UTF-8 and preserves those bytes. Calls use the resolved embedding request timeout.
- Body packaging uses exact indexed UTF-8 byte counts and complete parent/segment bodies only. `inline_limited` is true whenever a complete parent cannot be returned, including a complete matched-segment fallback or an omission. It returns stable score sources (`fts`, `vector`, `both`) and partial-coverage metadata.

The deduplicated active parent-body copy is intentionally a bounded per-request core representation, not a latency guarantee. Phase 12/load measurement must quantify its memory and scan cost before any operational budget is claimed.

## Focused checks run

```text
go test -count=1 -race ./internal/search ./internal/store ./internal/config ./internal/vector ./internal/embedclient ./internal/search/lexical
go vet ./internal/search ./internal/store ./internal/config ./internal/vector ./internal/embedclient ./internal/search/lexical
go build ./internal/search ./internal/store ./internal/config ./internal/vector ./internal/embedclient ./internal/search/lexical
go list -deps ./internal/search ./internal/store | rg 'cidx/internal/lab'   # no matches
git diff --check
```

Focused fake/core checks cover lexical isolation from corrupt vector/segment state, preflight fallback call counts, exact query request/transform/timeout, corrupt-row zero-call fallback, partial coverage, deterministic binary/int8 shared-key collapse, generation drift, independent FTS during a blocked query, no query-vector persistence, and exact UTF-8/aggregate body budgets.

The main agent independently reran the listed race, vet, build, format, dependency-boundary, and diff checks before accepting the phase. The `inline_limited` contract was also checked for full-parent, matched-segment-only, and omitted-body outcomes.

## Remaining risks

- No real Voyage/API-key/network behavior, corpus/evaluation, MCP/CLI integration, full-project validation, load test, or live publish-concurrency benchmark was run.
- Active parent-body copying and brute-force vector scanning have no latency/memory budget yet; Phase 12 owns measurement, not a quality or readiness claim.
