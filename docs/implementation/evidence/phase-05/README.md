# Phase 05 Worktree Index Pipeline Evidence

- Phase: `05-worktree-index-pipeline`
- State: `complete; main-agent commit-boundary validation accepted`
- Date: 2026-08-15

## Implemented contract

- `internal/root` validates an explicit canonical Git worktree root with a local `.cidx/config.json`; Git receives fixed arguments and NUL-delimited output is consumed as data.
- `internal/ignore` enumerates tracked plus untracked nonignored paths, normalizes repository-relative `/` paths, and rejects/excludes traversal, state directories, dependency/build outputs, generated/minified and lock inputs, oversized inputs, symlinks, and non-regular files.
- `internal/index` serializes writers with `.cidx/index.lock`, obtains the authoritative snapshot/plan only after that lock, opens selected files through Go's root-relative `os.OpenRoot` API after rejecting symlinked path components, reads each accepted file once, hashes that byte slice, dispatches explicitly to Go, TypeScript, or TSX chunkers, rejects unsafe diagnostics, derives symbols, projections, canonical segment inputs, and deterministic manifests locally, and never imports an embedding client.
- `internal/index/canonicaltext` owns the version-1 labeled, newline-normalized, LF-terminated canonical input byte format. It rejects ambiguous paths and is the sole source of segment hashes.
- `internal/store/index_snapshot.go` adds the narrowly scoped planning snapshot API; `internal/store/index_publish.go` adds an active-table delta publisher. It validates the complete immutable publish payload before entering SQLite, verifies the base generation in the final transaction, deduplicates multi-segment deletions, removes/creates only changed FTS rows with exact contentless delete values, updates all profile fingerprint and canonical-JSON metadata columns plus manifest/run metadata, and commits the next generation together. No Phase 02 schema migration or profile contract changed.

## Checks actually run

Implementation-agent focused checks:

```text
gofmt -w internal/index internal/store/index_publish.go
go test -count=1 ./internal/index ./internal/store ./internal/ignore ./internal/root
git diff --check
```

Main-agent boundary checks:

```text
go test -count=1 -race ./internal/index/... ./internal/ignore ./internal/root ./internal/store
go test -count=1 -race ./internal/index/canonicaltext
go vet ./internal/index/... ./internal/ignore ./internal/root ./internal/store
go build ./internal/index/... ./internal/ignore ./internal/root ./internal/store
gofmt -l <Phase 05 owned Go paths>
rg -n '"cidx/internal/(embedclient|lab)"' <Phase 05 runtime paths>
git diff --check
```

The first race batch passed `internal/index`, `internal/ignore`, `internal/root`, and `internal/store`, and exposed one duplicate trailing LF in `internal/index/canonicaltext`. The formatter was corrected to preserve exactly one final LF, then the main agent reran only that failed package with race detection; it passed. Vet, focused build, format, forbidden direct-import, and diff checks then passed.

The focused index integration test initializes a local temporary Git repository, indexes tracked Go and untracked TypeScript from the live worktree, excludes Git-ignored and built-in-excluded files, verifies persisted files/chunks/segments and FTS replacement, proves unchanged files do not publish a new generation, and verifies changed-file plus deletion publication. Storage-profile and canonical-text reconciliation preserve chunk, segment, and FTS identities while updating only their intended keys. The filter test covers built-in exclusions, symlink/oversize exclusion, and traversal rejection; the reader test covers a symlinked parent escape. Canonical-text fixtures cover CRLF/CR normalization, labels, separators, a single final LF, hashes, and ambiguous-path rejection. Failed/cancelled paths leave generation zero. The production publisher pins an old read transaction through a new publish and confirms it still reads the old chunk, then injects an invalid next payload and confirms the active generation remains unchanged.

## Checks not run

- No broad project, prior-phase, FTS-search, CLI, MCP, corpus, network, provider, paid API, embedding, or evaluation run.
- No load/platform concurrency benchmark was run. The focused production-publisher reader test covers the SQLite snapshot invariant, but a long-running scan/parse reader harness remains for main-agent validation.

## Remaining risks and handoff

- The final publisher is intentionally active-table in-place and depends on SQLite WAL/MVCC for readers pinned before commit; Phase 06 must consume `IndexSnapshot`/the later FTS reader in one read transaction.
- Index-profile mismatch reparses files; canonical-text mismatch recomputes existing canonical input hashes from the stored source/projection snapshot without AST/FTS replacement; source/vector-space/storage mismatches relink existing segment serving keys without AST/FTS replacement or provider access. The metadata and segment updates publish in the same generation transaction.
- No automatic hook, provider, vector creation, ready-state bit, lab database, or external endpoint was added.
