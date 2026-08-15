# Phase 10 Revision 4 Reconciliation Evidence

- Status: `accepted`
- Owner: `/root/r4_phase10_executor` (terra/high); `/root/r4_phase10_review` (terra/high); Codex commit-boundary validation
- Entry commit: `317d23c`
- Paid/provider evidence: **NOT RUN**. No Voyage request, API-key read, corpus selection, corpus checkout, or embedding submission is authorized in this reconciliation.

## Entry evidence checked

- Full Phase 10 implementation document and historical Phase 10 evidence.
- Phase 05 and Phase 08 Revision 4 evidence plus accepted Phase 09 materialization evidence.
- Live public plan/apply, shared executor, transform/codec, production run/failure/publication, and focused fake-provider tests.
- Clean workspace at the Phase 10 entry commit.

## Active implementation boundary

- Preserve API-free planning, explicit apply approval, plan freshness, the repository embedding lock, direct source-f32 transformation, current-profile incremental publication, and transaction-local generation/profile/key reproof.
- Replace only the production serial grouping/retry loop with the accepted Phase 08 byte-bounded synchronous executor and resolved request/retry policy.
- Transform, encode, and publish validated groups as they complete without holding a provider wait transaction or importing the raw lab.
- Preserve successful publications when another group fails; record provider/response/transform failures safely and classify cancellation as non-terminal.
- Keep requested, succeeded, failed, discarded, actual-token, and run-status accounting exact under retry, out-of-order completion, cancellation, stale keys, and local handler failures.
- Leave query embedding and FTS fallback integration to Phase 11, and leave CLI presentation/wiring to Phase 13.

## Checks run

```text
gofmt -w internal/app/embed.go internal/app/embed_test.go internal/embed/plan.go
go test -count=1 ./internal/app ./internal/embed ./internal/embedclient ./internal/store
go test -count=1 -race ./internal/app ./internal/embed ./internal/store
go vet ./internal/app ./internal/embed ./internal/embedclient ./internal/store
go build ./internal/app ./internal/embed ./internal/embedclient ./internal/store
gofmt -l internal/app/embed.go internal/app/embed_test.go internal/embed/plan.go  # no output
go list -deps ./internal/app ./internal/embed ./internal/store  # no internal/lab or internal/devapp
git diff --check

# independent review
# /root/r4_phase10_review: no findings

# one-time main-agent commit-boundary validation
go test -count=1 -race ./internal/config ./internal/embed ./internal/embedclient ./internal/embedlock ./internal/app ./internal/store ./internal/vector
go vet ./internal/config ./internal/embed ./internal/embedclient ./internal/embedlock ./internal/app ./internal/store ./internal/vector
go build ./internal/config ./internal/embed ./internal/embedclient ./internal/embedlock ./internal/app ./internal/store ./internal/vector
go test -count=1 ./...
go build ./...
gofmt -l internal/app/embed.go internal/app/embed_test.go internal/embed/plan.go  # no output
go list -deps ./internal/app ./internal/embed ./internal/store  # no internal/lab or internal/devapp
rg '\\bBatches\\(' internal/app internal/embed  # no output
git diff --exit-code 317d23c -- schemas  # no output
git diff --check
```

All passed locally. The independent Terra review reported no findings. The
focused fake-provider coverage verifies exact document
request fields and codec parity, API-free/stale-plan behavior, FTS while a
request is blocked, removed-key discard, immediate durable success while a
concurrent failure remains blocked, generation-change handler accounting,
cancellation failure durability, and response-validation versus source-valid,
prefix-zero transform failure accounting. No
shared-executor grouping/retry/`Retry-After` scenarios were duplicated here;
they remain Phase 08 coverage.

## Implemented reconciliation

- `PublicEmbedding.Apply` converts the fresh, sorted paid inputs to keyed
  `embed.RequestInput` values and invokes `embed.Execute` with the one
  validated resolved request/retry policy and document role.
- Its serialized success handler transforms, encodes, and immediately
  publishes each valid vector through the existing short-transaction reproof.
  Transform rejection records a terminal `transform` failure; encode, publish,
  and durable handler errors stop execution without fabricating provider rows.
- Returned outcomes are accounted before handler-error detection. Provider and
  response failures record independently under a bounded background context;
  cancellation remains retryable/`cancelled`, provider attempt timeouts remain
  retryable/`provider`, and provider/validation outcomes do not themselves
  turn apply into an application error.
- The dead serial token-budget `embed.Batches` helper is removed. Production
  request grouping is byte-bounded only through `embed.Group`/`embed.Execute`.

## Checks not run

- Live provider, corpus, paid embedding, evaluation, and promotion work.

## Handoff

Phase 11 receives the accepted active serving profile and fingerprint,
codec-tagged rows and coverage, the shared transformer and scorer inputs, and
stable fallback reasons. It owns request-local query embedding through the
shared executor and the affected fallback/body boundary. No document-provider,
corpus, or paid action is required to enter it.
