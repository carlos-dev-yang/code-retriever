# Phase 12 Revision 4 Corpus-Independent Reconciliation Evidence

- Status: corpus-independent Revision 4 boundary `accepted`; official Phase 12 evaluation/promotion remains `blocked`
- Owner: `/root/r4_phase12_executor` (terra/high), `/root/r4_phase12_review` (terra/high), and Codex commit-boundary validation
- Entry commit: `0258872`
- External actions: **NOT RUN**. No corpus selection/access/binding, provider/API-key/network request, paid query, retrieval metric measurement, candidate selection, or promotion action is authorized.

## Entry evidence checked

- Full canonical Revision 4 design, execution guide, evaluation contract, implementation index, status ledger, and Phase 12 document.
- Phase 07, Phase 08 Revision 4, Phase 09, Phase 11 Revision 4, and historical Phase 12 evidence.
- Live corpus-independent retrieval core, development adapter, immutable artifact publisher, lab v4 evaluation provenance, Phase 11 query executor, and focused synthetic tests.
- User-directed English advice from `kb-guide` on logical-operation versus provider-attempt denominators and honest partial token observability.
- Clean workspace at the Phase 12 entry commit.

## Active reconciliation boundary

- Preserve provider attempts as operational lineage, never retrieval-query denominators.
- Record exactly one logical operation per required query plus actual attempts, retries, validated/failed counts, conservative terminal state, nullable observed provider tokens, token-observed attempts, and completeness.
- Record configured retry/source/timeout identity without claiming observed backoff, `Retry-After`, per-attempt latency/status, or complete token usage when the provider did not expose it.
- Add a preserving isolated lab v4-to-v5 migration because the old non-null token total cannot distinguish an observed zero from unobserved failed-attempt usage.
- Freeze `internal/eval` retrieval calculations, Phase 11 search/ranking/body algorithms, shared execution, `internal/evalcontract`, and `schemas/evaluation`.

## Implemented corpus-independent boundary

- The plan now names `logical_query_operations_planned`, while retaining the
  separate dataset `query_count` provenance.
- Each first Phase 11 query execution receives one transparent, operation-scoped
  recording client. It records dataset-order logical-operation IDs and actual
  attempts, then finalizes only after Phase 11 validation/transform/error
  classification returns. Cached vector arms and cached provider failures do
  not create duplicate records.
- `provider-usage.json` records the local wire kind/version, configured retry
  policy identity and wait prefix, timeout, source/provider/model/adapter
  identity, ordered operations, and a derived aggregate. The run manifest
  embeds only that aggregate. Failed attempts have no invented token value;
  immediate observed zero is retained; retry-success remains incomplete; and
  generated-response tokens are `NOT_APPLICABLE`.
- The isolated lab schema is now v5. v4 flat calls/tokens migrate to nullable
  legacy columns, every historical v4 row becomes `legacy`, and current rows
  persist only the validated aggregate accounting fields.

## Checks run

```text
gofmt -w internal/devlab/evaluate.go internal/devlab/retrieval_artifact.go internal/devlab/evaluate_test.go internal/lab/schema.go internal/lab/store.go internal/lab/f32codec_test.go
go test -count=1 ./internal/devlab ./internal/lab
go test -count=1 -race ./internal/devlab ./internal/lab
go vet ./internal/devlab ./internal/lab
go build ./internal/devlab ./internal/lab
go mod tidy -diff
gofmt -l internal/devlab/evaluate.go internal/devlab/retrieval_artifact.go internal/devlab/evaluate_test.go internal/lab/schema.go internal/lab/store.go internal/lab/f32codec_test.go  # no output
git diff --check
```

All passed. Focused synthetic checks cover immediate observed-zero success,
retry-success, terminal provider failure denominator treatment, mixed derived
aggregates, null-versus-zero tokens, malformed/forged/reordered accounting,
post-success attempts, cancellation and malformed-response non-publication,
deterministic dataset order, v4-to-v5 preservation, current v5 insertion, and
the existing older migration chains. A Phase 11 provider failure that wraps an
exhausted per-attempt deadline remains `FAILED` operational denominator
evidence and publishes its artifact/row; only the caller context itself
aborts publication. The checks do not repeat the shared executor's retry-
timing matrix.

Independent Terra review found one error-authority defect: an exhausted
per-attempt provider timeout was initially mistaken for caller cancellation.
The remediation makes the caller context authoritative for cancellation while
retaining a Phase 11 `QueryEmbeddingProviderError` that wraps an attempt
deadline as failed operational evidence. The same reviewer rechecked the
remediation and reported no findings.

Codex then ran the one-time main commit-boundary validation:

```text
go test -count=1 -race ./internal/devlab ./internal/lab ./internal/eval ./internal/evalcontract ./internal/search ./internal/embed ./internal/embedclient
go vet ./internal/devlab ./internal/lab ./internal/eval ./internal/evalcontract ./internal/search ./internal/embed ./internal/embedclient
go build ./...
go test -count=1 ./...
go mod tidy -diff
gofmt -l <changed Go files>  # no output
go list -deps ./internal/search ./internal/store  # no lab/devlab/devapp/eval dependency
git diff --exit-code 0258872 -- <frozen evaluation/search/executor/schema paths>
git diff --check
```

All passed. Production relational schemas, normative evaluation schemas, and
the frozen retrieval, ranking, packaging, query-executor, shared-executor, and
evaluation-contract implementations are unchanged.

## Checks not run

- Corpus/provider/API-key/network/paid work, real quality/latency/cost measurement, candidate selection, confirmation, promotion, MCP/CLI, and load work.

## Exact next action

Commit this accepted corpus-independent boundary separately, then enter Phase
13. Official Phase 12 remains blocked on the documented user-controlled corpus,
labels, raw coverage, and separate paid-query approval; none is a gate for the
corpus-independent Phase 13 adapter.
