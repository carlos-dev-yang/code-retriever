# Phase 14 Revision 4 Corpus-Independent Working Evidence

- Status: `blocked`; the corpus-independent implementation handoff passed
  independent Terra review after remediation and the one main non-artifact
  commit-boundary validation. Artifact generation awaits an owner-selected
  cidx project license.
- Owner: `/root/r4_phase14_executor` (terra/high).
- Entry commit: `3030ddd`.
- External actions: **NOT RUN**. No corpus selection/binding, provider/API-key
  request, paid operation, model or assistant invocation, host-config mutation,
  or release publication occurred.

## Implemented working boundary

- `internal/buildinfo` emits stable schema-versioned JSON with ldflag `Version`/`Commit`
  overrides and `debug.ReadBuildInfo` fallback. It reports explicit unknown
  provenance rather than guessing, has no timestamp, and includes actual
target/CGO facts, linked SQLite and grammar IDs, per-language chunker IDs, FTS/production-schema
  IDs, and a non-static link policy.
- `cidx version [--json]` is a CLI-only reporting surface. It includes a fresh
  disposable runtime FTS5/WAL/grammar probe and production schema range, but
  does not alter MCP `serverInfo`, stdio behavior, or the exactly-four-tool
  registry.
- `internal/runtimecheck.Check` opens only a disposable temporary SQLite file
  through `store.OpenSQLiteStores`, then probes the embedded Go, TypeScript,
  and TSX grammars. It has no network/download/repair path. `Initialize` runs
  it before `.cidx` exists or mutates; production `Open` runs it after config
  load and before production DB opening or migration. Both test seams are
  per-call dependencies, not mutable globals.
- Local packaging is constrained to darwin/arm64 with `CGO_ENABLED=1` and
  `clang`. The script records ldflag provenance, build manifest, checksums,
  Mach-O linkage evidence, exact cached-module copied-source texts and notices
  with source paths/checksums, without inferred license classifications. It
  refuses a dirty source tree or absent project `LICENSE` rather than inventing
  redistribution terms.
- The local verifier is network-denied only when `sandbox-exec` is present. It
  is designed to verify checksum/corruption/executable paths, Go/TS/TSX FTS
  behavior in a Git path containing spaces and non-ASCII text, concurrent
  indexing, structurally validated four direct stdio tools, root/schema/unknown-config failures with
  empty stdout plus actionable stderr, no lab/API-key dependency, and isolated
  Codex get/list project-config parsing under the CLI's supported options. `CIDX_EVIDENCE_DIR` retains those local stdout/
  stderr and MCP transcripts when a user runs the verifier. It does not call a
  model or assistant.
- Install, Codex host, manual hook, and upgrade documentation explicitly limit
  claims to this local target and preserve the project-only/no-secret/no-host
  mutation policy.

## Checks run

```text
gofmt -w internal/buildinfo/*.go internal/runtimecheck/*.go internal/app/bootstrap.go internal/app/bootstrap_test.go internal/cli/cli.go internal/cli/cli_test.go
go test -count=1 ./internal/buildinfo ./internal/runtimecheck ./internal/app ./internal/cli
go test -count=1 -race ./internal/buildinfo ./internal/runtimecheck ./internal/app ./internal/cli
go vet ./internal/buildinfo ./internal/runtimecheck ./internal/app ./internal/cli ./cmd/cidx
go build -o <temporary-directory>/cidx ./cmd/cidx
go mod tidy -diff
bash -n scripts/package-local.sh scripts/verify-local-release.sh
go list -deps ./cmd/cidx compared exactly with packaging/third-party-licenses.tsv
CGO_ENABLED=1 CC=clang go build -trimpath -buildvcs=true -ldflags <test Version/Commit/CGO> ./cmd/cidx; cidx version --json
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 ./...
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 -race ./internal/buildinfo ./internal/runtimecheck ./internal/app ./internal/cli ./internal/mcp ./internal/index ./internal/store ./internal/search ./internal/chunk/...
env -u VOYAGE_API_KEY GOPROXY=off go vet ./...
env -u VOYAGE_API_KEY GOPROXY=off go build ./...
CGO_ENABLED=0 GOPROXY=off go build ./cmd/cidx # expected grammar build-constraint failure
frozen production-core and production-dependency checks
git diff --check
```

All listed checks passed. The ldflag smoke returned the supplied version,
commit, `darwin/arm64`, `CGO_ENABLED=1`, modernc SQLite `v1.47.0`, all three
grammar module versions, all three runtime grammar probes, FTS5/WAL true, and
production schema range `1..3`. The injected runtime-fail fixtures prove no
`.cidx` mutation before initialization failure and no production DB open after
configured-state runtime failure.

`scripts/package-local.sh` was invoked once and correctly stopped before any
artifact at the absent owner project `LICENSE`. No archive, checksum, host
transcript, or package/host success is claimed.

## Checks not run

- Package and verifier end-to-end execution: blocked by the missing owner
  project `LICENSE`; the source tree is also intentionally in implementation
  state until this phase is committed.
- Actual Codex CLI project-config parse under the packaged binary: part of the
  blocked verifier; no user/home config was read or changed by this work.
- Any non-darwin/arm64 target, code signing, notarization, model/assistant
  invocation, corpus/label/task work, paid query, Phase 12 official core run,
  or immutable `release_candidate` promotion result.

## Remaining blocker and next action

The implementation checkpoint is ready for its separate provenance commit.
Public package generation then needs an explicit owner project-license choice.
Official release-candidate scope also remains blocked on user-selected corpus/labels/bindings, a compatible Phase 12
`core_retrieval` result, frozen assistant controls/tasks with all three arms,
and separate paid-query approval. This record is not a promotion result and
does not make any `NOT_PROMOTION_READY` artifact appear immutable without its
required provenance.
