# Relation Evidence Completion Stage A — Revision 4

- State: accepted corpus-independent implementation boundary
- Date: 2026-08-18
- Implementation commit: `c863c049128470a190639f5e74b28a4b16a7f0f7`
- Authority: [`RELATION-EVIDENCE-COMPLETION-PLAN.md`](../../RELATION-EVIDENCE-COMPLETION-PLAN.md)
- Scope: Phase 07 relation sidecar plus the narrow Phase 12 retrieval-artifact producer contract
- Promotion status: not calibration results, not confirmation, and not product evidence

## Accepted implementation

`cidx dev relations complete` now consumes one immutable relation graph, one
immutable Phase 12 retrieval artifact, the exact dataset file as an opaque
digest plus an ID/text-only projection, and the current semantic-parent
snapshot. It publishes evaluation-only Stage A artifacts containing:

- full active-int8 segment-to-parent collapse and global parent score/rank/tie
  evidence;
- semantic features only for endpoints in the accepted bounded relation
  frontier;
- one-hop interpretive contract-closure candidates with explicit omission
  reasons and predeclared parent-count/body-byte grids; and
- body-free relation-hint disclosures with count and exact serialized-byte
  grids for the later independent assistant experiment.

The completion pass calls no provider, persists no query or document vector,
does not modify the protected primary ranking, and is not imported by
production search, MCP, store, or vector packages. It adds no MCP tool.

The Phase 12 producer now supports a distinct
`RELATION_CALIBRATION_POOL_BUILDING` evidence class. That class is rejected
unless it uses production FTS, a clean known build, an explicit series
authority and budget, source dimension 1024, serving dimension 1024 or 512,
and int8 storage. The historical `CALIBRATION_POOL_BUILDING` path retains its
safe-token FTS and exact 32-operation rules.

## Binding and safety proof

Before publication the consumer verifies:

- the exact graph and 16-file retrieval checksum sets;
- clean graph, retrieval, and current executable provenance;
- corpus, manifest, content, generation, profile, canonical/raw dataset,
  query ID/order, and query-text digest equality;
- the exact request-local query-vector digest without persisting the vector;
- one active-codec row per query and its exact occurrence count;
- distinct canonical-input coverage equal to the independently recorded raw
  document-input universe while allowing valid repeated occurrences;
- unique composite segment occurrences and one current semantic-parent mapping
  for every observation;
- distinct authoritative collapsed-parent identities, exact scores, and
  explicit tie ranges; and
- a final live snapshot, graph, retrieval artifact, and dataset reproof before
  immutable publication.

Candidate generation decodes only `corpus_id`, query ID, and query text. It
does not instantiate the frozen label schema. Portable hint payloads contain
relative source identities and line ranges but no source body, raw score,
confidence claim, vector, credential, absolute path, or provider data.

## Review and validation

Terra reviewed the new scope independently. The first review found producer
compatibility, full-universe cardinality, collapsed-parent uniqueness,
label-loading, and provenance gaps. The implementation repaired each item.
The final review also distinguished unique canonical vector keys from repeated
segment occurrences and reported `CLEAR` after the corrected proof.

The main agent then ran one offline commit-boundary validation with
`VOYAGE_API_KEY` removed and `GOPROXY=off` where applicable:

```text
go test -count=1 ./...
go test -count=1 -race ./internal/relationdiag ./internal/devlab
go vet ./internal/relationdiag ./internal/devlab
go build -o /tmp/cidx-relation-completion-boundary ./cmd/cidx
go mod tidy -diff
gofmt -l internal/relationdiag internal/devlab
git diff --check
go list -deps ./internal/search ./internal/mcp ./internal/store ./internal/vector
```

All checks passed. The dependency listing contained no
`cidx/internal/relationdiag` entry.

## Work deliberately not performed

- No new repository was selected, cloned, downloaded, indexed, or parsed.
- No existing chi/RHF score was inspected by this new policy.
- No document or query embedding was requested.
- No label pass, calibration result, confirmation result, assistant run, or
  product integration was produced.

The exposed 32-case chi/RHF set remains closed to tuning. The next step is
blocked on the owner's explicit selection and approval of every new
calibration repository. Acquisition, document capture, calibration query
execution, confirmation query execution, and assistant execution remain
separate approval boundaries.
