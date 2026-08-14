# Phase 01 Runtime and Storage Spike Evidence

- Phase: `01-runtime-storage-spike`
- State: `done; main-agent commit-boundary validation passed`
- Date: 2026-08-15
- Scope: executable technology and codec contracts only; no final production schema, chunking rule, MCP transport, corpus, or paid embedding operation.

## Selected build contracts

| Concern | Selected contract | Evidence/result |
| --- | --- | --- |
| Go module | `cidx` | A local module name; it does not imply remote ownership. |
| SQLite | `modernc.org/sqlite` v1.47.0, BSD-3-Clause | Pure-Go binding. Its startup probe creates a temporary FTS5 table; opening fails when FTS5 or WAL is unavailable. The exact linked module version is read from Go build metadata and returned in `SQLiteCapabilities`/the spike runner. |
| Parser | `github.com/tree-sitter/go-tree-sitter` v0.25.0, MIT; Go grammar v0.25.0, MIT; TypeScript/TSX grammar v0.23.2, MIT | Grammars' generated C sources are compiled from Go modules; `internal/chunk` has no network/runtime-download path. |
| CGO policy | Required for builds that include Go/TypeScript/TSX parsing | `CGO_ENABLED=0 go build ./...` fails specifically because the embedded grammar packages are excluded. SQLite itself remains pure Go. |
| Verified platform | `darwin/arm64` with Apple clang and CGO enabled | `go build ./...`, parser core test, and `cidx-spike` FTS5/WAL run passed. |
| Unverified platforms | all other OS/architecture/toolchain combinations | No distribution-support claim is made yet; later packaging must build and test each declared target with a compatible C toolchain. |

## Storage and generation result

`internal/store` creates separately scoped reader and writer `database/sql` handles, enables WAL, uses a 2-second `busy_timeout`, and limits the writer handle to one connection. Modernc's DSN `_pragma` mechanism applies `busy_timeout` and `foreign_keys=ON` as each physical pool connection opens; a core test concurrently holds all four configured reader connections and checks both connection-local pragmas. `PublishPlan` contains work prepared outside SQLite. Its selected `ApplyPreparedInPlace` path replaces the prepared current snapshot, contentless FTS rows, and `active_generation` in one transaction.

The core test holds an old read transaction across publication. It observes generation 1 and its old FTS candidate after generation 2 commits; a new read transaction observes only generation 2. Injected failure rolls back the chunk, contentless FTS, and active-generation update together. Insert, same-row update, and deletion are all exercised.

Generation-scoped staging was not retained as a second implementation: SQLite read snapshots already give the required old-or-new visibility with the preferred in-place active tables, while retaining old and new contentless FTS copies would create an unnecessary alternate serving path. This spike did not measure a large-delta write duration; Phase 05 must reopen that choice only if a measured prepared delta cannot remain short.

## Vector and lab result

| Item | Fixed contract |
| --- | --- |
| Source/targets | source exactly 1024; targets limited centrally to 256, 512, 1024. |
| Transform | `prefix-l2-v1`: choose leading target components, then L2 normalize. `cosine` remains the target-f32 reference metric. |
| Binary | `cidx-binary-sign-lsb-v1`: non-negative=1, negative=0; component order is least-significant-bit first; last-byte high padding is zero; zero/non-finite vectors rejected. Scorer is normalized sign agreement in `[-1,1]`, an approximation and not exact cosine. |
| Int8 | `cidx-int8-symmetric-v1`: persist the float32 encoding of per-vector `max_abs/127`, then quantize with that exact persisted scale; nearest rounding, clamp `[-127,127]`; recompute and persist the float32 reconstructed norm from that scale/blob; validator recomputes it. Scorer is bounded reconstructed-vector cosine approximation. Zero/non-finite vectors and invalid scale/norm/length reject. |
| Lab f32 | `cidx-lab-f32-le-v1`: headerless IEEE-754 binary32 little-endian, exact `4*dimensions`, CRC-32 checksum. A separate development-only SQLite factory permits document-source artifacts only and refuses `.cidx/index.db`. |
| Production/lab boundary | `internal/store` does not import `internal/lab`; production vectors use only `vector.StoredVector`. Lab f32 is a distinct Go type and only `internal/lab` opens the lab schema. |

No paid source vector was captured, so no lab raw artifact, checksum, API input manifest, or direct-target observation exists. The temporary lab schema is explicitly a Phase 02 import/migration handoff, not a runtime dependency.

## Voyage direct-target decision

The official source contract is represented in `internal/embedclient`: provider `voyage-official`, model `voyage-code-4`, explicit source 1024, dtype `float`, `document`/`query` role mapping, `truncation=false`, and adapter version 1. It performs no HTTP call.

Direct 512/256 provider comparisons are **NOT RUN** because no paid operation was approved. `DirectTargetComparison=false`, and any spec that enables it is rejected. The production reference remains local source-1024 prefix-plus-L2 for documents and queries; query f32 has no persistence API.

## Checks actually run

```text
gofmt -w cmd/cidx-spike/main.go internal/vector/*.go internal/lab/*.go internal/embedclient/*.go internal/store/*.go internal/chunk/*.go
go mod tidy
go test ./internal/... ./cmd/cidx-spike
go build ./...
go run ./cmd/cidx-spike -input internal/vector/transform.go -language go -db /tmp/cidx-spike-phase01.db
CGO_ENABLED=0 go build ./...

# focused review-fix recheck
go test ./internal/store ./internal/vector ./cmd/cidx-spike
```

Results:

- Core tests passed for Tree-sitter Go/TypeScript/TSX parsing, production SQLite FTS5/WAL contentless FTS CRUD, held-reader old/new publication visibility, rollback, f32 lab integrity, and binary/int8 validity/scoring.
- The standalone runner reported `root=source_file`, `has_error=false`, `sqlite=modernc.org/sqlite@v1.47.0`, `fts5=true`, `wal=true`, `platform=darwin/arm64`.
- The focused recheck forced all four reader-pool connections and verified `busy_timeout=2000` plus `foreign_keys=1` on each; it also verifies the robust spike-path production guard and int8 persisted-scale/norm self-score bounds.
- The CGO-disabled build failed as expected with grammar packages excluded by build constraints, establishing the explicit parser CGO requirement.

## Checks not run

- No Voyage API request, direct-target compatibility comparison, paid document capture, query embedding, corpus selection, or provider batch-cap check; all require explicit approval or later scope.
- No network-firewalled parser run. Static package inspection and successful local parse establish no runtime downloader, but a packaged offline-host test remains for Phase 14.
- No non-darwin/arm64 build or runtime test.
- No large-delta write-duration measurement, numeric retrieval comparison, or f32-versus-codec ranking evaluation; those belong to Phases 05, 09, and 12.

## Main-agent commit-boundary validation

The main agent reviewed the complete Phase 01 diff and ran this validation once after the focused implementation fixes:

```text
go test -count=1 ./internal/... ./cmd/cidx-spike
go test -count=1 -race ./internal/store ./internal/vector ./cmd/cidx-spike
go vet ./internal/... ./cmd/cidx-spike
go build ./...
go run ./cmd/cidx-spike -input internal/vector/transform.go -language go -db <temporary-directory>/spike.db
go list -deps ./cmd/cidx-spike
CGO_ENABLED=0 go build ./...
gofmt -d cmd internal
go mod tidy -diff
git diff --check
```

All positive checks passed. The dependency list contained no `cidx/internal/lab`. The runner reported `modernc.org/sqlite@v1.47.0`, FTS5, WAL, and `darwin/arm64`. The CGO-disabled build failed only at the embedded Go and TypeScript grammar packages, which is the expected and documented parser build constraint.

## Phase 02 handoff

Phase 02 may consume the selected package boundaries, IDs, blob validators, source spec, `PublishPlan`/reader-writer pattern, and separate lab factory. It must supply formal migrations, `ResolvedConfig`, semantic fingerprints, final schema names, and profile injection. It must not enable direct-target API dimensions or make the lab a runtime dependency.
