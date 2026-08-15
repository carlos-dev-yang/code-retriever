# Phase 11 Revision 4 Reconciliation Evidence

- Status: `accepted`
- Owner: `/root/r4_phase11_executor` (terra/high); `/root/r4_phase11_review` (terra/high); Codex commit-boundary validation
- Entry commit: `499df39`
- Provider/paid evidence: **NOT RUN**. No Voyage request, API-key read, corpus selection/access, paid query, or metric-measurement action is authorized in this reconciliation.

## Entry evidence checked

- Full canonical Revision 4 design, implementation execution guide, evaluation contract, implementation index, status ledger, and Phase 11 document.
- Accepted Phase 06, Phase 09, and Phase 10 Revision 4 evidence plus the historical Phase 11 evidence.
- Live query adapter, search fallback/snapshot path, shared executor, vector scan/RRF/body core, and Phase 12 evaluation adapter call site.
- Clean workspace at the Phase 11 entry commit.

## Active reconciliation boundary

- Replace only the direct single query embedding call and local timeout with the accepted shared synchronous executor.
- Inject request grouping limits, concurrency, per-attempt timeout, retry count, and 10/20/30-second waits from the one immutable `ResolvedConfig`.
- Preserve preflight before paid work, lexical-only fallback, caller cancellation, response validation, shared transform, post-request profile reproof, and query/vector nonpersistence.
- Preserve the existing snapshot, codec scan, segment collapse, RRF, body packaging, coverage, and deterministic tie algorithms unchanged.
- Let the existing `EmbedEvaluationQuery` consume the same reconciled adapter without creating a second Phase 12 query executor.

## Implemented reconciliation

- `internal/search/query_embedding.go` now formats one query into exactly one
  ephemeral `embed.RequestInput` keyed `query-0`, then invokes
  `embed.Execute` with query role and every grouping, concurrency, timeout,
  retry, and wait value derived from the immutable resolved configuration.
  Query text and vectors remain request-local and are neither persisted nor
  logged.
- The accepted executor remains the response-validation authority. Its ordered
  single outcome is transformed through the existing shared transformer only
  after validation. Validation and transform failures retain their invariant
  errors; only exhausted provider attempts become
  `QueryEmbeddingProviderError`; caller cancellation/deadline returns the
  caller context error.
- `EmbedEvaluationQuery` still calls this unchanged adapter, so it receives
  the same executor policy without a Phase 12-specific request path. Existing
  service fallback behavior continues to map non-cancellation query execution
  failures to lexical-only `QUERY_EMBEDDING_FAILED` results.

## Checks run

```text
gofmt -w internal/search/query_embedding.go internal/search/service_test.go
go test -count=1 ./internal/search
go test -count=1 -race ./internal/search ./internal/store ./internal/config ./internal/vector ./internal/embed ./internal/embedclient ./internal/search/lexical
go vet ./internal/search ./internal/store ./internal/config ./internal/vector ./internal/embed ./internal/embedclient ./internal/search/lexical
go build ./internal/search ./internal/store ./internal/config ./internal/vector ./internal/embed ./internal/embedclient ./internal/search/lexical
gofmt -l internal/search/query_embedding.go internal/search/service_test.go  # no output
go list -deps ./internal/search ./internal/store | rg 'cidx/internal/(lab|devapp)'  # no matches
go mod tidy -diff
git diff --exit-code 499df39 -- internal/search/vector_scan.go internal/search/rrf.go internal/search/inline_body.go internal/search/evaluation.go internal/store/hybrid_snapshot.go internal/eval/retrieval.go internal/embed internal/embedclient internal/config
git diff --check

# independent review
# /root/r4_phase11_review: no findings

# one-time main-agent commit-boundary validation
go test -count=1 -race ./internal/search ./internal/embed ./internal/embedclient ./internal/config ./internal/vector ./internal/store ./internal/search/lexical ./internal/devlab
go vet ./internal/search ./internal/embed ./internal/embedclient ./internal/config ./internal/vector ./internal/store ./internal/search/lexical ./internal/devlab
go build ./internal/search ./internal/embed ./internal/embedclient ./internal/config ./internal/vector ./internal/store ./internal/search/lexical ./internal/devlab
go test -count=1 ./...
go build ./...
go mod tidy -diff
gofmt -l internal/search/query_embedding.go internal/search/service_test.go  # no output
go list -deps ./internal/search ./internal/store  # no internal/lab, internal/devlab, or internal/devapp
rg '\.Embed\(' internal/search/query_embedding.go  # no output
git diff --exit-code 499df39 -- internal/search/vector_scan.go internal/search/rrf.go internal/search/inline_body.go internal/search/evaluation.go internal/store/hybrid_snapshot.go internal/eval/retrieval.go internal/embed internal/embedclient internal/config schemas
git diff --check
```

All passed. The independent Terra review reported no findings. The main suite
completed its tests, race, vet, builds, module, format, and dependency checks;
the first residue-search expression matched the `EmbeddingClient` type name,
so only that inspection was corrected to the literal direct-call pattern
`\.Embed\(` before the remaining frozen-file, schema, and diff checks ran.
No test suite was repeated. Focused tests cover one transient provider retry followed by
success with the resolved 10-second wait through the narrow waiter seam;
query role/request and shared transform; validation and zero-prefix transform
failures retaining invariant errors; exhausted provider error classification;
caller cancellation during a request and retry wait; and existing
lexical-only `QUERY_EMBEDDING_FAILED` metadata. The executor's broader
grouping/retry matrix remains Phase 08 evidence and was not duplicated.

## Checks not run

- Live provider, API-key, network, corpus, paid query, evaluation metrics, promotion, MCP/CLI, and load work.

## Handoff

Phase 12 receives the unchanged `EvaluationSession`, target-f32/active-codec
scan, collapse, RRF, and body-packaging path plus the reconciled
`EmbedEvaluationQuery`. It must distinguish logical query count from actual
provider attempts and reuse this path rather than create a second executor.
Corpus selection/access, provider calls, paid queries, metric measurement, and
promotion remain outside this accepted boundary.
