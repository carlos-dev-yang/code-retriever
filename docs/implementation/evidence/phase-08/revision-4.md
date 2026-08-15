# Phase 08 Revision 4 Reconciliation Evidence

- Status: `accepted` at the main-agent commit boundary
- Owner: `/root/r4_phase08_executor` (terra/high); `/root/r4_phase08_review` (terra/high); Codex commit-boundary validation
- Entry commit: `79d729c`
- Paid/provider evidence: **NOT RUN**. No Voyage request, API-key read, corpus selection, corpus checkout, or embedding submission is authorized in this reconciliation.

## Entry evidence checked

- Full Phase 08 implementation document and historical Phase 08 evidence.
- Phase 02 Revision 4 config/profile evidence and Phase 05 Revision 4 index/profile evidence.
- Live `internal/embed`, `internal/embedclient`, `internal/lab`, and `internal/devapp` execution paths.
- Clean workspace at the Phase 08 entry commit.

## Active implementation boundary

- Add one provider-neutral byte-bounded synchronous request grouper and executor under `internal/embed`.
- Keep the regular Voyage HTTP endpoint as the only provider call; do not add asynchronous Batch API or polling behavior.
- Limit groups to the resolved maximum input count and total UTF-8 input bytes, with bounded concurrency and per-attempt timeout.
- Retry only transient transport/timeout/HTTP 408/429/5xx failures after the resolved 10/20/30-second schedule, honoring only a longer valid `Retry-After` and stopping waits immediately on cancellation.
- Validate a complete response group before exposing it as successful, persist each successful lab group immediately, and retain those commits if another group fails.
- Preserve token estimates only as diagnostics; they are not a grouping or rejection authority.
- Leave Phase 10 production publication and Phase 11 query integration to their owning phases.

## Implemented reconciliation

- `internal/embed` now owns provider-neutral keyed request inputs, exact UTF-8-byte grouping, and a bounded synchronous executor. It validates duplicate keys/ordinals, empty or invalid UTF-8 inputs, and oversize requests before a provider call; token estimates remain plan diagnostics only.
- The executor uses the regular `EmbeddingClient.Embed` operation, applies one configured timeout per attempt, validates each complete response before success, permits only transient transport/attempt-timeout/408/429/5xx retries, waits on the configured 10/20/30-second schedule (or a longer valid `Retry-After`), and joins active workers on cancellation or a handler invariant failure. Success handlers are serialized as groups complete; returned outcomes are ordinally ordered.
- `embedclient.ProviderError` now retains sanitized class/status/cause metadata and parsed positive `Retry-After` delta-seconds or HTTP-date values without retaining response bodies or authentication data. It still performs one HTTP call only; retry timing remains executor-owned.
- The raw-lab collector now receives byte/concurrency/timeout/retry semantics from `ResolvedConfig` through `internal/devapp`, plans with the shared byte grouper, and uses the shared executor. Every validated successful group is committed through the serialized handler immediately; independent failed groups are recorded without rolling back earlier commits, so a later plan resumes from those raw hits.
- Phase 10 production publication and Phase 11 query wiring remain unchanged. `internal/embed/plan.go` reuses the byte grouper for public plan counts only; no production apply executor was added here.

## Implementation and review checks

```text
gofmt -w internal/embed internal/embedclient internal/lab/collector.go internal/lab/collector_test.go internal/devapp/capture.go
go test -count=1 ./internal/embed ./internal/embedclient ./internal/lab ./internal/devapp ./internal/app
go test -count=1 -race ./internal/embed ./internal/embedclient ./internal/lab ./internal/devapp
go vet ./internal/embed ./internal/embedclient ./internal/lab ./internal/devapp
go build ./internal/embed ./internal/embedclient ./internal/lab ./internal/devapp ./internal/app
git diff --check
```

All commands passed. Focused offline coverage includes exact 128/129 count grouping; exact 256 KiB and oversize byte grouping; multibyte UTF-8, invalid input, and duplicate-key rejection; measured concurrency reaching but not exceeding four; initial plus three retries with 10/20/30 waits; shorter and longer `Retry-After` handling plus signed/decimal/malformed/overflow rejection; request and wait cancellation; actual per-attempt deadline retry; response reorder and complete-response validation; deterministic ordinal outcomes despite out-of-order completion; lab immediate persistence/partial resume; handler/store-failure provider accounting; cancellation retryability; and mutated resolved request-policy rejection at both devapp entrypoints.

The independent Terra re-review reported **No findings** after the required accounting, config-integrity, Phase 10 ownership, and `Retry-After` remediations.

## Main-agent commit-boundary validation

```text
go test -count=1 -race ./internal/config ./internal/embed ./internal/embedclient ./internal/embedlock ./internal/lab ./internal/devapp ./internal/app
go vet ./internal/config ./internal/embed ./internal/embedclient ./internal/embedlock ./internal/lab ./internal/devapp ./internal/app
go build ./internal/config ./internal/embed ./internal/embedclient ./internal/embedlock ./internal/lab ./internal/devapp ./internal/app
go test -count=1 ./...
go build ./...
changed-package gofmt check
production app/search dependency exclusion for internal/lab and internal/devapp
removed token-named request-policy and asynchronous Batch API residue checks
production/lab schema non-change check against 79d729c
git diff --check
```

All commit-boundary checks passed. No provider, network, API-key, corpus, or paid action was performed.

## Checks not run

- Live provider, corpus, paid embedding, evaluation, and promotion work.

## Independent Terra review remediation

- The review found that raw-capture run accounting returned on a local success-handler failure before counting provider attempts and validated-response token usage. `Collector.Apply` now aggregates every returned executor outcome before detecting `HandlerError`; successful validated response tokens are counted outside the persistence handler, and handler/store errors do not create provider-failure rows.
- Failed provider and response-validation groups now attempt every failure record through a bounded independent background context. Parent cancellation no longer prevents the remaining records, and parent cancellation is recorded as a retryable `cancelled` outcome rather than a terminal blacklist. Attempt-timeout provider errors remain retryable provider failures because transient classification occurs before generic context-error classification. If failure-record persistence also fails, it is joined with the original provider/context error so `errors.Is` retains the original return authority.
- `EmbeddingCapture.PlanWithOptions` and `Apply` now call `Resolved.ValidateIntegrity` before any policy derivation or lock/store work. Focused coverage mutates each request ceiling and the retry waits to prove both entrypoints fail closed.
- `embed.Batches` was restored to its Phase 10-owned pre-reconciliation behavior. Phase 08 uses `Group` only for `BuildPlan` request-group counts and lab capture; production apply remains unchanged.
- Retry-After delta-seconds are now accepted only when composed of ASCII digits before integer parsing. Signed, decimal, malformed, and overflowing values are rejected; valid long values and HTTP-dates remain accepted.

## Handoff and residual risks

- Phase 10 may now integrate production document publication with the accepted executor; Phase 11 remains responsible for request-local query integration.
- No provider, network, API-key, corpus, or paid action occurred. Live provider behavior and actual provider request IDs remain unobserved.
- A response can still succeed remotely before its local SQLite group transaction commits; a later resume may make a duplicate paid request. Exactly-once behavior cannot span Voyage and SQLite.
- Phase 10 may reuse `embed.RequestInput`, `RequestGroup`, `Execute`, and `ProviderError` metadata for production document publication. Phase 11 may reuse the same executor for query calls only when its own phase applies the query authorization boundary.

## Exact next action

Enter Phase 10 from this accepted boundary without performing provider or corpus actions.
