# Phase 10 source-bank-first document publication

- State: accepted at the Phase 10 commit boundary.
- Date: 2026-08-17.
- External actions: no credential read, corpus read, provider request, or
  network operation was performed; provider behavior used injected fakes.

## Accepted flow

```text
verified active canonical inputs
  -> existing compatible 1024-f32 source rows
       -> local prefix/L2/int8 materialization
  -> missing source rows
       -> explicitly approved Voyage document request
       -> validate response 1024-f32
       -> commit immutable product source row
       -> prefix/L2/int8
  -> generation/profile/key-reproved production publication
```

- Planning opens an existing source bank read-only and does not create one.
- The public plan reports `source_input_count` and `voyage_input_count`; the
  retired `paid_input_count` wording is absent.
- A source-only apply needs neither `VOYAGE_API_KEY` nor an embedding client.
- A Voyage input still requires explicit approval and an injected client.
- Provider success handling commits the immutable document source before any
  transform or serving publication. If a later local invariant or production
  reproof fails, the compatible source remains reusable on the next plan.
- The source-profile/input-hash key is serving-dimension independent, so a
  1024 provider result can locally materialize either the 1024/int8 default or
  the 512/int8 compact target.
- `requested_count` remains provider-attempt accounting. Local source-bank
  outcomes count toward succeeded/failed/discarded without manufacturing a
  provider request.
- The public embedding path has no evaluation-store dependency, and search has
  no source-bank dependency.

## Existing core proof adapted

The existing public embedding integration now proves that a fake Voyage
response creates both the int8 serving row and the separate source-bank row.
It then removes the serving row, replans as one source input and zero Voyage
inputs, and republishes locally with zero requests and zero tokens. Existing
tests continue to cover plan freshness, cancellation, response validation,
transform failures, stale-key discard, FTS availability during a blocked
request, retry classification, request fields, and run accounting.

## Commit-boundary validation

The first focused run exposed that the old run validator assumed every local
success had a provider request. The validator was reconciled so outcome counts
are bounded by payable inputs while `requested_count` remains provider-only.
The final boundary passed:

```text
go test -count=1 ./internal/sourcebank ./internal/embed ./internal/embedclient ./internal/store ./internal/app ./internal/cli ./internal/devapp ./internal/devlab
go test -count=1 -race ./internal/sourcebank ./internal/embed ./internal/embedclient ./internal/store ./internal/app ./internal/cli
go vet ./internal/sourcebank ./internal/embed ./internal/embedclient ./internal/store ./internal/app ./internal/cli ./internal/devapp ./internal/devlab
go build ./...
gofmt -l internal/app internal/cli internal/embed internal/sourcebank internal/store
go mod tidy -diff
git diff --check
```

Static dependency checks confirmed that `internal/search` does not import the
source bank and the public embedding/source-bank path does not import
`internal/lab`.

## Handoff

Phase 11 receives one current int8 profile, query-side 1024-f32 transformation,
the request-local int8 query preparation/scorer, and unchanged snapshot/RRF/
fallback/body contracts. It does not receive source-bank access.
