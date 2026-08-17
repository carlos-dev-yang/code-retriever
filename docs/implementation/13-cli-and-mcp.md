# 13. CLI and MCP Surface Integration

- Status: `blocked` — public init/help/package smoke must be reconciled to
  default 1024, `--serving-dim <1024|512>`, fixed int8, and source-bank reuse after Phase 02/08/11
- Prerequisites: reconciled `05-worktree-index-pipeline`, `10-embedding-orchestration-and-reconciliation`, and `11-vector-and-hybrid-search`; completed `06-fts-search`; Phase 12 corpus-independent core/API
- Followed by: `14-packaging-and-host-integration`
- Design source: `local-code-search-mcp-v1-design-r4.md` sections 3, 4, 8, and 10
- Evaluation authority: [EVALUATION-CONTRACT.md](EVALUATION-CONTRACT.md)

## Context Recovery Checklist

Read the [implementation index](README.md), [execution guide](EXECUTION-GUIDE.md), [evaluation contract](EVALUATION-CONTRACT.md), and [project status](STATUS.md) before resuming.

- Confirm the Phase 05/06/10/11 application services are stable and the Phase 12 corpus-independent core/API plus synthetic adapter parity are available; this phase adapts them rather than inventing new indexing or ranking logic. An official corpus run is not an entry gate.
- Re-check the exact MCP registry: `status`, `search`, `read_span`, and `reindex`, with no fifth tool and no lab/config/document-embedding tool.
- Re-check that caller-required `max_inline_bytes` limits bodies only; it cannot change result rank, IDs, order, or count. This phase validates/clamps the request and passes the effective maximum to the Phase 11 shared packager.
- Re-check stdio purity, bounded concurrent dispatch, request cancellation, one explicit root per process, and the rule that FTS works without `VOYAGE_API_KEY`.
- Re-check that public/lab dependency graphs remain separate, query f32 is nonpersistent, and production vectors use the fixed cidx-owned int8 codec.
- Stop if a tool schema, max-byte semantics, error code, root/freshness boundary, or paid-query disclosure is unresolved. Do not expand the public contract implicitly.
- Before pausing, update schemas/examples, this phase's evidence and decision log, then update [STATUS.md](STATUS.md) with validated transport behavior, open risks, and the exact next action.

## 2026-08-17 product-profile supersession

Public `cidx init` defaults to 1024, accepts only explicit 1024 or 512, and
exposes no codec flag. Config still records fixed int8 identity. Binary/256
code paths are removed; only historical evidence remains under
[`RETIRED-VECTOR-PROFILES.md`](RETIRED-VECTOR-PROFILES.md).

## Revision 4 initialization checkpoint

The narrow provider-free initialization reconciliation entered from `6797544`
is implemented, independently reviewed, validated at the one main commit
boundary, and accepted in [Phase 13 Revision 4 evidence](evidence/phase-13/revision-4.md).

- `root.GitRoot` discovers the containing Git worktree without requiring
  `.cidx/config.json`; `root.Repository` retains its configured explicit-root
  behavior for normal serving.
- `cidx init [--serving-dim <1024|512>]` uses the
  complete `config.DefaultRaw` factory and resolves it before any write. It
  stages owner-only configuration under an exclusive temporary name, opens and
  closes production SQLite through `store.OpenProduction`, then atomically
  publishes `.cidx/config.json` without replacement. That hard link is the
  commit point; redundant staging-link cleanup is best-effort.
- Initialization has no provider client, key read, network, lab, corpus,
  index, or embedding action. Existing configuration or configless production
  DB state is rejected before mutation; a failed staged attempt removes only
  its own temporary and production artifacts so a retry can succeed. It claims
  the initially absent DB exclusively and rolls it back only when that exact
  file identity remains current, preserving externally replaced state.
- The existing MCP registry, search/ranking/body-packaging core, index/store
  algorithms, and read-span implementation remain frozen; the added
  read-span coverage only re-proves line-cap-free complete byte-bounded
  behavior.

## 1. Objective

Connect earlier application services to one `cidx` CLI and stdio MCP server. Keep the public surface small and stable, while keeping source-bank mutation out of MCP and isolating evaluation operations under explicitly unstable `cidx dev ...` commands.

Completion requires:

- CLI and MCP call the same index/search/status/read services.
- MCP exposes exactly `status`, `search`, `read_span`, and `reindex`.
- The caller supplies required `search.max_inline_bytes` on every request.
- The server hard maximum clamps body transfer only and does not alter ranking or result count.
- Long status/reindex work and search are not serialized by the dispatcher.
- stdout contains only stdio protocol frames; diagnostics use stderr.
- Source f32, target materialization, and evaluation remain outside MCP; ordinary CLI embed/rematerialization owns the product source-bank workflow.

## 2. Scope and Non-goals

### In scope

- Public CLI tree and exit/error contracts.
- Unstable development CLI namespace.
- stdio JSON-RPC/MCP transport.
- Strict schemas for exactly four MCP tools.
- Validation and typed application-error mapping.
- Concurrent dispatch, cancellation, and graceful shutdown.
- MCP adaptation of Phase 11 body packaging under `max_inline_bytes`.
- Hash-guarded live-file `read_span`.
- stdout/stderr separation and structured diagnostics.
- Cost visibility in help and response metadata.

### Out of scope

- New index or search algorithms, or a fifth MCP tool.
- HTTP, SSE, remote transport, GUI, installer, or daemon.
- Automatic host-config edits.
- MCP document embedding, source-bank/lab execution, evaluation, or config mutation.
- Server-side token-budget estimates or enforcement.
- Generated summaries or result rewriting.
- Long-term compatibility guarantees for development commands.

## 3. Prerequisites

- Phase 02 injects one immutable `ResolvedConfig`.
- Phase 05 `IndexService` is shared by CLI `index` and MCP `reindex`.
- Phase 06/11 search returns FTS/hybrid results plus fallback metadata.
- Production embedding and lab paths from Phases 08-11 are package-separated.
- Phase 11 returns final ranks, packaged indexed source bodies, omission metadata, and freshness inputs through a transport-independent response model.
- Phase 12 exposes the real production search/evaluation core and synthetic parity seam. Official corpus evaluation, profile promotion, and residual-risk evidence remain Phase 14 release-candidate inputs rather than a Phase 13 implementation blocker.
- Production and lab connection types are not interchangeable.

## 4. Invariants

### Public surface

1. MCP tool names are fixed to `status`, `search`, `read_span`, and `reindex` for v1.
2. `search` never auto-reindexes or embeds documents.
3. `reindex` never calls Voyage AI.
4. `search(mode=fts)` works without network or credentials.
5. Only `search(mode=hybrid)` may pay for query embedding, and cannot bypass the configured paid guard.
6. `status` returns neither source bodies nor a full file list.
7. The source bank, lab DB, `internal/sourcebank`, and `internal/lab` are absent from the `serve` dependency graph.

### Body maximum

1. `search.max_inline_bytes` is a required integer at least zero.
2. `effective_max_inline_bytes = min(request.max_inline_bytes, config.mcp.hard_max_inline_bytes)`.
3. The measured unit is the sum of actual indexed-source UTF-8 bytes placed in all `results[].body` fields.
4. JSON metadata, escaping overhead, and token counts are excluded.
5. The value cannot alter candidate selection, scores, ranks, or the IDs/order/count of up to `k` results.
6. A body is a complete source chunk, a complete matched segment, or absent; never cut at an arbitrary byte.
7. If an FTS-only hit's complete chunk does not fit, omit its body.

### Transport and concurrency

1. stdout contains no bytes outside MCP JSON-RPC frames.
2. Responses correlate by request ID and may complete out of request order.
3. One handler's scan, parse, or API wait cannot block dispatch of independent handlers.
4. Cancellation propagates to the application service.
5. Short SQLite writer serialization for index/vector publication is allowed; a dispatcher-wide mutex is not.

## 5. Implementation Packages, Files, and Types

```text
cmd/cidx/main.go                 # process entry and exit code
internal/app/bootstrap.go        # root/config/store/service assembly
internal/app/commands.go         # public application commands
internal/app/{status,search,readspan}.go
internal/cli/{root,init,status,index,embed,serve,output}.go
internal/devlab/cli.go           # unstable cidx dev namespace
internal/mcp/server.go           # lifecycle and bounded dispatcher
internal/mcp/transport_stdio.go  # frame I/O only
internal/mcp/schema.go           # exactly four strict tool schemas
internal/mcp/handlers.go         # schema to application adapters
internal/mcp/errors.go           # typed error mapping
internal/readspan/service.go     # hash-guarded live range
```

Key types:

```text
Application
  StatusService / SearchService / ReadSpanService / IndexService / EmbedService

MCPServer
  Dispatcher / ToolRegistry(exactly four) / RootContext / Shutdown

SearchToolRequest
  Query string
  K optional integer
  Mode optional fts|hybrid
  MaxInlineBytes required nonnegative integer

SearchToolResponse
  IndexGeneration / ManifestSHA256
  RequestedMax / EffectiveMax / InlineBytesUsed / Clamped / InlineLimited
  VectorCoverage / QueryEmbeddingUsed / FallbackReason / Results[]

ReadSpanRequest
  RelativePath / StartLine / EndLine / ExpectedSHA256
```

CLI/MCP parsers perform syntax validation, clamp the requested maximum to the configured hard maximum, and convert to shared application request types. Phase 11 applies the effective body budget once. The MCP package owns neither ranking nor body allocation.

## 6. CLI and MCP Contracts

### 6.1 Stable public CLI

```text
cidx init [--serving-dim <1024|512>]
cidx status [--json]
cidx index [--dry-run] [--reason manual|commit]
cidx embed [--dry-run|--apply] [--retry-failed]
cidx serve --root <repository-root>
```

- `init` creates production config/DB at the Git root without an API call. It records `voyage-code-4`, the selected serving dimension (1024 by default), `fts` as the default mode, and fixed int8 storage. It resolves source 1024 from `ModelSpec` and never silently overwrites existing config.
- `status` briefly copies the active DB snapshot, closes its transaction, then inspects the whole live worktree without writing.
- `index` uses the Phase 05 live-worktree AST+FTS pipeline; `--reason` is metadata.
- `embed` defaults to pending-input, reusable-source, and token/cost planning. `--apply` reuses compatible source rows locally and calls the paid document API only for missing sources; every new source row is durable before active int8 publication. It does not depend on a lab DB.
- `serve` starts one stdio MCP server for one explicit root.

Do not place temporary r2-style f32 preservation flags such as `--eval-f32-out` on stable public embed.

### 6.2 Unstable development CLI

```text
cidx dev embeddings capture [--apply] [--retry-failed]
cidx dev embeddings materialize [--activate]
cidx dev retrieval evaluate --corpus-manifest <path> --dataset <path> [--apply]
```

- `capture` reports compatible product-source `voyage-code-4` 1024-dimensional document f32 and pays only for misses under `--apply`; development run accounting is written separately.
- `materialize` locally transforms product source f32 into the one current project profile. Default is a plan; `--activate` verifies active segment-key agreement and atomically publishes that current-profile set. It does not edit config or require the lab DB.
- `evaluate` compares lexical, exhaustive serving-dimension f32, active int8, vector, and hybrid variants for current config and an explicitly approved corpus/dataset. Default is planning; `--apply` pays for queries. Query f32 stays in run memory.

There is no `promote` command. `cidx index` owns profile/key reconciliation; materialization publishes vectors for the already-current profile. Mark development commands unstable, omit them from MCP, and never make them required general installation steps.

### 6.3 MCP `status`

Input is empty. Output includes:

- Desired/applied index, embedding-source, vector-space, vector-storage, and serving fingerprints.
- `observed_generation` and `manifest_sha256`.
- File/chunk/segment counts.
- Dirty and stale/unindexed/deleted/index-error counts.
- Active-snapshot coverage and ready/pending/failed counts.
- Last successful/attempted index and embedding times.
- Whether generation changed during the complete filesystem inspection.

Return no body, full path list, or raw vector.

### 6.4 MCP `search`

Input fields:

- Required nonempty `query`.
- Optional integer `k`, using resolved default and absolute v1 maximum 20.
- Optional `mode: fts | hybrid`, using resolved default.
- Required nonnegative integer `max_inline_bytes`.

Do not add `detail`, `verbosity`, or `include_body`.

Top-level output includes generation/manifest, requested/effective byte maxima, actual inline bytes and clamp/limit flags, vector and partial coverage, query-embedding use and fallback reason, mismatch/freshness diagnostics, and at most `k` ranked results.

Each result includes path, kind, symbol, qualified symbol, signature, parent range, optional matched segment, conditional complete body/range, `content_source=indexed_snapshot`, `indexed_sha256`, `source_state`, and score sources.

Copy rank, metadata, and body from one DB read snapshot and close it. Deduplicate returned paths and check live hashes only after ranking. Freshness annotation neither changes rank nor starts reindexing. A body always comes from the indexed snapshot; never silently substitute live bytes.

### 6.5 MCP `read_span`

Input is repository-relative `path`, 1-based inclusive `start_line` and `end_line`, and required `expected_sha256` from search.

Read the current live file exactly once without following symlinks. Derive the whole-file SHA-256 and requested line bytes from that same read. Return `FILE_STALE` on hash mismatch and `FILE_NOT_FOUND` when absent.

Return the complete requested range only when it fits the server hard maximum; never truncate it. Otherwise return `SPAN_TOO_LARGE` and `max_bytes`. A single source line exceeding the cap cannot be split by v1 `read_span`.

There is no read-span line-count cap. The complete requested range is governed only by the byte limit and remains all-or-nothing.

### 6.6 MCP `reindex`

Input has one optional `dry_run` boolean. It calls the same Phase 05 `IndexService` and no external API.

- Apply result: scanned/updated/reused/deleted files, updated chunks, reused/pending embeddings, activated generation, and manifest.
- Dry run: planned file/chunk/reuse/pending counts with no production DB write.

## 7. Configuration and Change Impact

| Setting | Consumer | Effect |
| --- | --- | --- |
| `search.default_mode` | Search request default | Applies after restart/reload; no reindex |
| `search.allow_paid_query_embedding` | Hybrid paid guard | No document embedding; false means FTS fallback |
| `search.return_k` | Optional `k` default | No reindex |
| `search.candidate_k`, RRF | Search service | No reindex |
| `mcp.hard_max_inline_bytes` | Search/read responses | Applies on next serve; no profile change |
| Active index/serving profile | Status/search validation | Mismatch causes policy-defined fallback/reconciliation |
| Model/serving-dimension/codec | Embed/search core | Read only through the one Phase 02 profile and `voyage-code-4` spec |

`mcp.hard_max_inline_bytes` defaults to 64 KiB and is a positive server safety ceiling. Reject startup if it is invalid or exceeds the code-owned absolute ceiling of 1 MiB. A request may choose any lower nonnegative maximum. Do not estimate tokenizer counts or host context size to decide bodies.

Credentials come only from `VOYAGE_API_KEY`. The endpoint `https://api.voyageai.com/v1/embeddings` is code-owned and host config cannot provide a custom `base_url`. FTS-only startup succeeds without a key.

## 8. Ordered Implementation Checklist

1. Bootstrap all services through constructor injection.
2. Separate public commands from the development namespace and connect existing embed/capture/materialize/evaluate application handlers.
3. Define human output, `--json`, and exit codes for every public command.
4. Reserve stdout for stdio and stderr for all logs/progress.
5. Build an immutable registry containing exactly four MCP tools.
6. Implement strict schemas rejecting unknown fields and invalid types/ranges.
7. Map typed application errors to stable MCP codes and data.
8. Create per-request context and cancellation propagation.
9. Use bounded concurrent dispatch while preserving response IDs.
10. Make status copy a DB snapshot before filesystem scanning.
11. Pass the validated effective maximum into the Phase 11 packager and verify its returned byte accounting before serialization; do not reimplement allocation.
12. Implement root/path/symlink/hash/range/max checks for `read_span`.
13. Connect `reindex` to the same service as CLI index.
14. Preserve paid guard, missing-key, profile-mismatch, and fallback metadata on the wire.
15. Verify `serve` and MCP handlers do not depend on `internal/lab`; only development bootstrap may do so.
16. On SIGINT/EOF, stop accepting work and cancel in-flight contexts.
17. Generate help/schema examples from one definition or otherwise keep them verifiably synchronized.

## 9. Failure, Rollback, Concurrency, and Security

### Failure and rollback

- Config/root/schema mismatch fails before serving requests.
- CLI index and MCP reindex follow Phase 05 rollback on cancellation/failure.
- Shared-packager or accounting failure does not rerank; it returns an internal error.
- Hybrid API failure degrades to an FTS result with `fallback_reason` when FTS is available.
- Distinguish JSON parse errors, unknown tools, invalid params, and application failures.
- After a partial stdio write failure, terminate rather than retransmitting partial JSON and creating duplicate responses.

### Concurrency

- Each request gets a context and bounded task/goroutine; never spawn without a limit.
- Handlers may run concurrently, but one writer serializes complete stdout frames.
- Search, status, and read-span do not acquire index/embed file locks.
- Reindex writers serialize under the Phase 05 index lock.
- A long status scan or reindex preparation cannot stop search dispatch.
- Handlers do not enlarge service-owned SQLite transaction scopes.
- Shutdown never rolls back already committed generations or vectors.

### Security

- Canonicalize the explicit serve root and compare it with DB root metadata.
- Reject absolute, traversing, symlinked, or ignored `read_span` paths.
- Put no logs, stack traces, or progress bars on stdout.
- Errors/logs exclude API keys, raw vectors, and complete source bodies.
- Help and docs disclose that document embedding and hybrid queries send code/query text to Voyage AI.
- Do not expose config mutation, raw capture, materialization, or evaluation through MCP.

## 10. Validation Scenarios

This file defines an implementation plan and does not add test code.

1. Without `VOYAGE_API_KEY`, init, status, index, FTS search, read-span, and reindex work.
2. Tool discovery shows exactly four tools.
3. Requests with max 0, small, sufficient, or above hard max return the same result IDs/order/count.
4. Actual inline UTF-8 bytes never exceed effective max and no source is cut arbitrarily.
5. Stale/deleted results are not expanded from a different live range.
6. Hash mismatch and oversized range return exact typed errors.
7. Long status/reindex and multiple searches prove dispatch is not serialized.
8. Out-of-order completion retains correct JSON-RPC IDs.
9. Cancellation reaches scan/API/services without damaging the active snapshot.
10. stdout capture contains only MCP frames.
11. Disabled paid guard, missing key, profile mismatch, and API failure use the required no-call or post-failure FTS behavior.
12. Development commands are absent from MCP; production serve neither creates nor opens a source bank or lab DB.
13. Unknown fields, negative/fractional maxima, excessive `k`, and invalid mode are rejected strictly.
14. Root mismatch, traversal, and symlinks fail closed.

## 11. Completion Evidence

Official Phase 12 corpus/usefulness or `core_retrieval` promotion evidence is not a Phase 13 completion condition. Phase 13 must prove adapter/core parity with the corpus-independent core; Phase 14 later references official core and assistant/host evidence for `release_candidate` scope.

- Public and development CLI help snapshots.
- Versioned schemas for exactly four tools.
- Request, response, and typed-error examples.
- Same-rank comparison across `max_inline_bytes` boundaries and actual UTF-8 byte recalculation.
- Adapter/core parity evidence proving MCP serialization preserves Phase 12's evaluated package IDs, ranges, omission reasons, and bytes.
- Concurrent status/reindex/search dispatch trace.
- Cancellation and graceful-shutdown trace.
- stdout protocol-purity capture and stderr-log sample.
- FTS-only operation without an API key.
- Dependency inspection proving production serve excludes `internal/sourcebank` and `internal/lab`.

Completion reports distinguish the transport, OS, and host actually checked from unverified items.

## 12. Handoff

Phase 14 receives one `cidx` binary and public help, the `cidx serve --root <repository-root>` stdio contract, four tool schemas and example frames, environment/stderr/stdout requirements, project-scoped host command/args/env guidance, an offline FTS smoke procedure, and SQLite/schema/grammar runtime requirements.

## 13. Decision Log

| Decision | Rationale | Revisit when |
| --- | --- | --- |
| Exactly four MCP tools | Keep this auxiliary tool and host context surface small | A measured independent use case exists |
| Accept only a maximum; no detail enum | Caller controls data volume without option explosion | Another stable budget contract is required |
| Body bytes cannot affect rank | Smaller context requests must not change retrieval quality or result count | Core v1 invariant; no planned revisit |
| Reuse Phase 11 body packager | Offline evaluation and MCP must exercise one result-shaping policy | A new transport requires different serialization only |
| Separate product source bank from evaluation state | Public embedding retains reusable document f32 without making lab metadata or raw vectors a search dependency | The product source-retention contract changes |
| Expose no source-bank MCP tool | Dimension changes use explicit CLI plan/apply and preserve the four-tool MCP surface | A measured MCP mutation use case is accepted |
| Concurrent dispatcher | Long management calls must not block search at application level | Core concurrency invariant |
| stdout is protocol-only | Prevent stdio frame corruption | Transport changes |
| Do not estimate token budgets | Caller owns tokenizer and host-context composition | Host provides a standard token contract |
| Init discovers Git before config | A new repository has no config yet, while normal serving still needs a configured worktree root | Repository ownership becomes multi-root |
