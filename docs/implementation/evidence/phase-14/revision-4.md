# Phase 14 Revision 4 Local Checkpoint Evidence

- Status: local darwin/arm64 package and operational checkpoint **accepted**;
  the phase is `blocked` for its official evaluation and release-candidate
  scope. No immutable `release_candidate` result was created.
- Provenance: clean commit
  `a5b2baef9a18e68d6c8b5d4fb62dc2e03727edb4`.
- Owner: `/root/r4_phase14_executor` (terra/high), with independent Terra
  artifact re-review reporting no findings.
- Entry commit: `3030ddd`.
- Implementation commit: `30748c1`.
- External actions: **NOT RUN**. No corpus selection/binding, provider/API-key
  request, paid operation, model or assistant invocation, host-config mutation,
  or release publication occurred.

## Accepted local artifacts

All artifacts below are ignored local outputs, not published releases.

| Artifact | Identifier |
| --- | --- |
| darwin/arm64 archive | `dist/cidx_dev-a5b2baef9a18_darwin_arm64.tar.gz`; SHA-256 `7da32f3852e5d14b00d4cbd9949d4cb73c2903696b097e6a1df75ec34869b90a` |
| checksum manifest | `dist/checksums.txt`; SHA-256 `62c53d862f4d3ae0fdfa6bb6eee5ef31344ebfa87d6d863f1337a2e6a2c404d7` |
| embedded build manifest | SHA-256 `301a46cd1bbfd1c21b33b3d855cb3f4641d2a8b2014625e3a7d0aa96284826a5` |
| retained evidence directory | `dist/evidence/phase-14-local-a5b2baef9a18` |
| direct MCP transcript | SHA-256 `0a65076960bb76eae432e326fd69f922c78c807de44a359f0001bf8b88f14b21` |
| Codex app-server transcript | SHA-256 `1df5f2a0320909fd43c2bc45535ef2d27fb8bc172a3b569769687efac7e5a278` |

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
  Codex app-server strict project-config parsing through `config/read` under
  the CLI's supported options. `CIDX_EVIDENCE_DIR` retains completed local
  stdout/stderr and MCP transcripts, including partial transcripts on failure
  without marking verification successful. It does not call a
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
scripts/package-local.sh
CIDX_EVIDENCE_DIR=dist/evidence/phase-14-local-a5b2baef9a18 scripts/verify-local-release.sh dist/cidx_dev-a5b2baef9a18_darwin_arm64.tar.gz dist/checksums.txt
```

All listed checks passed. The ldflag smoke returned the supplied version,
commit, `darwin/arm64`, `CGO_ENABLED=1`, modernc SQLite `v1.47.0`, all three
grammar module versions, all three runtime grammar probes, FTS5/WAL true, and
production schema range `1..3`. The injected runtime-fail fixtures prove no
`.cidx` mutation before initialization failure and no production DB open after
configured-state runtime failure.

The final package and verifier commands passed from the stated clean
provenance. The verifier accepted the archive checksum and rejected a deliberate
corruption. It also verified neutral archive uid/gid, owner names, and mtimes,
and confirmed that the linkage, binary-format, and Go-version diagnostics use
the literal target `cidx` rather than a staging path. The embedded manifest and
unpacked binary version JSON agreed on target, dependency, runtime FTS5/WAL,
registered Go/TypeScript/TSX grammars, and the production schema maximum.

The direct stdio transcript structurally validated exactly `status`, `search`,
`read_span`, and dry-run `reindex`; its provider-free FTS search returned the
fixture identity and `read_span` returned the requested complete range. The
same run verified concurrent indexing, no lab directory, no API key, and no
runtime grammar/model/download artifact. The separate root mismatch,
same-root newer-schema, and unsupported-config fixtures each failed with empty
stdout and actionable stderr. The Codex app-server check read the effective
configuration for the isolated trusted project, including the verifier-created
isolated `CODEX_HOME` trust layer, an empty system layer, and session flags
(`plugins=false`). Its only effective `cidx` MCP entry originated in the
project's `.codex` layer and confirmed the packaged command, arguments,
explicit root, stdio transport, and optional `env_vars` name. It did not run a
model or assistant, and it did not read or mutate existing user configuration.

## Checks not run

- Any non-darwin/arm64 target or host beyond the isolated Codex
  project-configuration read; code signing; notarization; or release
  publication.
- Model/assistant invocation, corpus/label/task work, provider/API-key action,
  paid query, Phase 07 official baseline, Phase 12 official core run, or an
  immutable `release_candidate` promotion result.

## Remaining blocker and next action

The local package/operational boundary is complete. Official release-candidate
scope remains blocked on
user-selected corpus manifests and bindings, reviewed labels, compatible raw
coverage, a compatible Phase 12 `core_retrieval` result, frozen assistant
controls/tasks with all three arms, and separate paid-query approval. The next
eligible work is Phase 07/12 official evaluation after those inputs and
authorizations exist. This record is not a promotion result and does not make
any `NOT_PROMOTION_READY` artifact immutable without its required provenance.
