# Phase 13 Revision 4 Provider-Free Initialization Evidence

- Status: `accepted`
- Owner: `/root/r4_phase13_executor` (terra/high), `/root/r4_phase13_review`
  (terra/high), and Codex commit-boundary validation
- Entry commit: `6797544`
- External actions: **NOT RUN**. No Voyage provider/API-key/network access,
  corpus selection/binding/access, paid operation, indexing operation, lab
  action, or evaluation action is authorized or performed.

## Entry evidence checked

- Full Revision 4 canonical contract, execution guide, implementation index,
  status ledger, evaluation contract, and Phase 13 document.
- Phase 05 Revision 4, Phase 06, Phase 10 Revision 4, Phase 11 Revision 4,
  and accepted corpus-independent Phase 12 Revision 4 evidence.
- Historical Phase 13 evidence and the live production bootstrap, CLI, MCP,
  search, index, store, and read-span boundaries at `6797544`.

## Implemented narrow boundary

- `root.GitRoot` discovers and canonicalizes the containing Git worktree with
  no configuration dependency, so `init` works when invoked from a nested
  directory. `root.Repository` still requires an explicit configured Git
  worktree root for normal production opening.
- `app.Initialize` constructs `config.DefaultRaw(servingDimensions, codec)`,
  resolves it before writing, stages the complete owner-only JSON through an
  exclusive `.cidx/.config.json.init` file, writes all bytes, syncs, and
  closes it. Only after `store.OpenProduction` successfully opens and closes
  production SQLite does it atomically link the final config without replacing
  a concurrently created one.
- Preflight rejects an existing config or configless `index.db`/WAL/SHM/journal
  production state before mutation. After staging, init exclusively claims an
  absent `index.db` and retains its file identity before calling
  `store.OpenProduction`. Failed staging/DB initialization/final publication
  removes staging plus DB/ancillary artifacts only if the current DB remains
  that claimed file; an external claim loss or path replacement preserves all
  unverified production artifacts. It removes `.cidx` only if that invocation
  created it and it is empty. After the no-replace hard link succeeds, init is
  committed: removal of the redundant staging link is best-effort and cannot
  report failure or remove the final config/DB. It does not construct a Voyage
  client or open lab state.
- Stable CLI `init` now requires `--serving-dim`, accepts only the shared
  default factory's 256/512/1024 values, defaults codec to `binary`, accepts
  only `int8` as the alternative, has an injected initializer seam, and no
  longer has the pending-default sentinel or its process-entry special case.
- The existing read-span implementation is unchanged. Focused tests re-prove
  that a 500-line, byte-fitting range succeeds and an oversized range returns
  typed `SPAN_TOO_LARGE` with `max_bytes` and no partial body.

## Checks actually run

All checks ran offline with `VOYAGE_API_KEY` unset and `GOPROXY=off`:

```text
gofmt -w internal/root/repository.go internal/root/repository_test.go internal/app/bootstrap.go internal/app/bootstrap_test.go internal/app/readspan_test.go internal/cli/cli.go internal/cli/cli_test.go cmd/cidx/main.go
go test -count=1 ./internal/root ./internal/app ./internal/cli ./internal/mcp
go test -count=1 -race ./internal/root ./internal/app ./internal/cli ./internal/mcp
go vet ./internal/root ./internal/app ./internal/cli ./internal/mcp ./cmd/cidx
go build ./internal/root ./internal/app ./internal/cli ./internal/mcp ./cmd/cidx
gofmt -l <owned Go files>  # no output
go list -deps ./internal/app ./internal/mcp ./internal/search ./internal/store | rg 'cidx/internal/(lab|devlab|eval)'  # no matches
go mod tidy -diff  # no output
git diff --exit-code 6797544 -- internal/config internal/mcp internal/search internal/index internal/store internal/embed internal/embedclient internal/vector internal/eval internal/evalcontract internal/lab internal/devlab schemas  # no output
git diff --check
```

Focused fixtures cover nested-directory Git-root init; exact Revision 4
defaults and reload; production DB/meta initialization; absence of lab state;
existing-config byte preservation with no DB mutation; configless orphan-DB
preflight preservation; injected DB-open cleanup followed by successful retry;
no-replace preservation of a config created before final publication; CLI
dimensions, default codec, invalid codec/dimension, rejected legacy
`--target-dim`, and help; plus long line-cap-free and all-or-nothing oversized
`read_span` behavior. A deterministic post-link staging-removal failure proves
that init returns success, retains a usable final config/DB, and treats a
leftover staging hard link as non-semantic cleanup residue. A deterministic
post-claim DB-path replacement verifies that rollback preserves the replacement
and its WAL while removing only this invocation's staging residue.

## Review remediation

Terra reported P1: publishing `config.json` before `store.OpenProduction`
could leave an unretryable config-only repository if SQLite initialization
failed. The implementation now stages and syncs private config first,
preflights existing production artifacts, initializes/closes SQLite before
no-replace publication, and cleans only attempt-owned state on failure. The
new focused tests prove the failure, no-mutation, retry, and concurrent-config
paths.

Re-review found a second P1 at the commit boundary: a failed best-effort
unlink after the no-replace hard link must not turn the already-published
config/DB into a reported failure or trigger rollback. Initialization now marks
the link as committed before attempting cleanup, ignores cleanup failure, and
uses per-call dependencies in focused tests rather than mutable package-global
test hooks.

The next re-review found rollback could still delete an external DB path that
appeared after preflight. Init now claims `index.db` with `O_CREATE|O_EXCL`
after exclusive staging, records its `FileInfo`, and removes DB/ancillary files
only when `os.SameFile` proves the current DB is that claim. The replacement
fixture proves a failed attempt preserves a newer DB and WAL verbatim.

The final independent re-review reported no findings after all remediations.

## Main commit-boundary validation

Codex ran the one-time boundary offline with `VOYAGE_API_KEY` unset and
`GOPROXY=off`:

```text
go test -count=1 ./internal/config ./internal/root ./internal/app ./internal/cli ./internal/mcp ./internal/index ./internal/store ./internal/search
go test -count=1 -race ./internal/root ./internal/app ./internal/cli ./internal/mcp ./internal/index ./internal/store ./internal/search
go vet ./cmd/cidx ./internal/config ./internal/root ./internal/app ./internal/cli ./internal/mcp ./internal/index ./internal/store ./internal/search
go build ./cmd/cidx
go mod tidy -diff
gofmt -l <owned Go files>  # no output
go list -deps ./internal/app ./internal/mcp ./internal/search ./internal/store  # no lab/devlab/devapp/eval dependency
rg -n --glob '!**/*_test.go' 'target-dim|ErrInitDefaultsPending|INIT_DEFAULTS_PENDING' cmd/cidx internal/cli  # no output
rg -n 'MaxReadSpan|max_read_span_lines|max_chunk_bytes' internal/app internal/cli internal/mcp internal/index internal/chunk  # no output
git diff --exit-code 6797544 -- <frozen core and schema paths>
git diff --check
```

All passed. The first legacy-name inspection included test fixtures and found
the intentional assertion that `--target-dim` is rejected. Codex corrected
only that inspection to exclude `*_test.go`, completed the remaining checks,
and did not rerun tests, race, vet, or build.

## Frozen boundary

No MCP, search, index, store, config, embedding, vector, evaluation, lab, or
schema implementation was changed. The frozen core's existing MCP/search
coverage was rerun through `./internal/mcp`; only root/app/CLI assembly and
focused regression tests changed.

Independent review, remediation, no-findings re-review, and the one main
commit-boundary validation are complete. No official Phase 12
corpus/usefulness/promotion claim is made or unblocked here.

## Phase 14 handoff

Phase 14 receives a provider-free `cidx init`
contract that can be invoked from a repository subdirectory, a validated
R4-configured production SQLite root, and the unchanged four-tool MCP/stdio
surface. Phase 14 still needs its own packaging, host, assistant-use, and
scoped `release_candidate` evidence; this accepted Phase 13 boundary
establishes none of those claims.
