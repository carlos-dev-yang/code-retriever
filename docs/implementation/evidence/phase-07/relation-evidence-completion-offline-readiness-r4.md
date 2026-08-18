# Relation Evidence Completion Offline Readiness — Revision 4

- Date: 2026-08-19
- State: accepted offline readiness checkpoint; no new corpus or provider operation
- Stage A implementation: `c863c049128470a190639f5e74b28a4b16a7f0f7`
- Documentation acceptance: `da5e0bd`
- Packaging dependency reconciliation: `31d37fe2035c0d39c44b7b8a598028f5041307f7`
- Authority: [`RELATION-EVIDENCE-COMPLETION-PLAN.md`](../../RELATION-EVIDENCE-COMPLETION-PLAN.md)
- Promotion status: regression and readiness evidence only; not calibration, confirmation, or promotion

## Scope and authorization

This checkpoint followed the `kb-guide` offline matrix with
`VOYAGE_API_KEY` absent and module/provider network access disabled. It did
not select, clone, update, index, or embed a new repository. It made zero
Voyage document/query operations and performed no assistant or promotion run.

The exposed chi/RHF 32-case set was used only to replay the already frozen and
rejected `anchor-frontier-graph-only-pareto-v1` policy. No question, label,
margin, feature family, closure role, hint budget, or product policy was
changed or selected from that replay.

## Full implementation boundary

The following checks passed:

```text
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 ./...
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 -race ./...
env -u VOYAGE_API_KEY GOPROXY=off go vet ./...
env -u VOYAGE_API_KEY GOPROXY=off go build ./...
go mod tidy -diff
gofmt -l <all tracked Go files>
git diff --check
node --check tools/relationdiag/typescript-resolver.mjs
bash -n <every tracked scripts/*.sh file>
jq -e . <every tracked schemas/testdata JSON file>
```

Focused Stage A negative and integrity tests also passed. They cover:

- approval/preflight rejection before runtime-state creation;
- strict relation-series evidence class, authority, budget, FTS, dimension,
  codec, reuse, and persistence controls;
- provider failure, retry, cancellation, timeout, and operation accounting;
- label-free completion inputs and exact query-text binding;
- full active-int8 canonical-input and repeated-occurrence coverage;
- stable parent collapse, explicit ties, protected top-20 identity, and final
  live reproof;
- endpoint/hint deduplication and closure omission reasons; and
- the exact four-tool MCP registry plus legacy dimension/flag rejection.

Production dependency enumeration for `internal/search`, `internal/mcp`,
`internal/store`, and `internal/vector` contained no
`cidx/internal/relationdiag` dependency.

## Toolchain and module evidence

The host `go` shim is 1.26.3 while `go.mod` requires 1.26.4. Therefore the
first `GOTOOLCHAIN=local` attempt stopped before module verification or product
execution with `HOST_TOOLCHAIN_PRECONDITION`. This is a host mismatch, not a
failed product check.

The exact Go 1.26.4 toolchain was already present in the local toolchain cache.
With `GOTOOLCHAIN=go1.26.4` and `GOPROXY=off`:

- `go version` reported `go1.26.4 darwin/arm64`;
- `go mod verify` passed;
- the trimpath CLI build passed and reported clean commit provenance, FTS5,
  WAL, Go/TypeScript/TSX grammars, and production schema range 1 through 5;
  and
- no network or provider operation occurred.

`go list -m all` is `NOT_RUN — OFFLINE_MODULE_GRAPH_CACHE_INCOMPLETE` because
the local proxy cache lacks metadata files for unused transitive
`golang.org/x/net@v0.58.0` and
`golang.org/x/telemetry@v0.0.0-20260811182544-a038080d80e5`. No dependency was
downloaded to satisfy this diagnostic-only enumeration. The exact linked
`cmd/cidx` module set was independently enumerated offline and accepted by the
package license allowlist and installed-binary verifier.

## Artifact integrity and historical replay

Before replay, 62 retained artifact manifests covering 699 files matched
their recorded byte sizes and SHA-256 digests. After the two regression runs,
64 manifests covering 727 files passed the same direct file verification.

Fresh ignored regression artifacts:

```text
.cidx/test/states/chi-1024-int8/evaluations/relation-diagnostic-regression-chi-pareto-20260819
.cidx/test/states/react-hook-form-1024-int8/evaluations/relation-diagnostic-regression-rhf-pareto-20260819
```

The preserved clean `497c000` evaluator reproduced all ten documented key
hashes for primary top-five proof, Pareto trace, Pareto denominators, bundles,
and aggregate metrics. The measured result remained:

| Corpus | Baseline complete | Augmented complete | Queries | Provider operations | `walkXFF` |
| --- | ---: | ---: | ---: | ---: | ---: |
| chi | 11 | 12 | 12 | 0 | 0 |
| react-hook-form | 19 | 20 | 20 | 0 | 0 |

This proves historical determinism only. The policy remains rejected because
the accepted result emitted 10 noise-only bundles among 17 emissions.

## Packaging regression and repair

The first clean package attempt correctly failed because the relation resolver
introduced `golang.org/x/tools` transitive modules that were absent from the
third-party source-file allowlist. Commit `31d37fe` reconciles the allowlist to
the exact linked module set and copies the original `LICENSE` and `PATENTS`
files for `x/mod`, `x/sync`, `x/sys`, and `x/tools`.

A second clean, offline clone at `31d37fe` passed package and installed-release
verification. It covered archive checksum and deliberate corruption, neutral
archive metadata, license inputs, default 1024/int8, provider-free 512/int8
rematerialization, Binary/256 rejection, Go/TypeScript/TSX FTS, exactly four
MCP tools, relocated source-bank-free serving, and isolated Codex project
configuration. It invoked no model, assistant, provider, or external corpus.

The ephemeral verification archive SHA-256 was
`dcc781cc2e2d970d63823fec0d4f723e03c89446278103eeed6ebb741273e204`;
the checksum-manifest SHA-256 was
`50c816455a9430d15b566484bced72bafdb3d11ce6b01e573dd4a4cbf2112d03`.
These hashes record this validation run and are not a published release.

## Exact stop boundary

All corpus-independent Stage A readiness gates now pass. The next action is
still blocked on explicit owner selection and approval of every new
calibration repository. Only after that selection may Stage B create portable
manifests and ignored bindings, acquire the repository, and run provider-free
index/parser/graph checks. Document capture and calibration query execution
remain later, separate approval boundaries.
