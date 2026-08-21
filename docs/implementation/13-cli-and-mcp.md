# 13. CLI and MCP Surface Integration

- Status: `done` — default 1024, explicit compact 512, fixed int8,
  source-bank reuse, and exactly four tools remain accepted. The revised
  search-result wire is locator-only and structured-only by default; its
  corrected reducer, budget invariance, adapter projection, and real Codex
  search-to-read journey are recorded.
- Prerequisites: reconciled `05-worktree-index-pipeline`, `10-embedding-orchestration-and-reconciliation`, and `11-vector-and-hybrid-search`; completed `06-fts-search`; Phase 12 corpus-independent core/API
- Followed by: `14-packaging-and-host-integration`
- Design source: `local-code-search-mcp-v1-design-r4.md` sections 3, 4, 8, and 10
- Evaluation authority: [EVALUATION-CONTRACT.md](EVALUATION-CONTRACT.md)

## Context Recovery Checklist

Read the [implementation index](README.md), [execution guide](EXECUTION-GUIDE.md), [evaluation contract](EVALUATION-CONTRACT.md), and [project status](STATUS.md) before resuming.

- Confirm the Phase 05/06/10/11 application services are stable and the Phase 12 corpus-independent core/API plus synthetic adapter parity are available; this phase adapts them rather than inventing new indexing or ranking logic. An official corpus run is not an entry gate.
- Re-check the exact MCP registry: `status`, `search`, `read_span`, and `reindex`, with no fifth tool and no lab/config/document-embedding tool.
- Re-check that caller-required `max_inline_bytes` is retained as a validated
  v1 compatibility input. Locator search returns no body for any value and the
  value cannot change result rank, IDs, order, or count.
- Re-check stdio purity, bounded concurrent dispatch, request cancellation, one explicit root per process, and the rule that FTS works without `VOYAGE_API_KEY`.
- Re-check that MCP/search never opens product source-bank or lab state, query
  f32 is nonpersistent, and production vectors use the fixed cidx-owned int8
  codec. The single CLI binary also owns public `embed`, so package linkage
  alone is not the runtime-access boundary.
- Stop if a tool schema, max-byte semantics, error code, root/freshness boundary, or paid-query disclosure is unresolved. Do not expand the public contract implicitly.
- Before pausing, update schemas/examples, this phase's evidence and decision log, then update [STATUS.md](STATUS.md) with validated transport behavior, open risks, and the exact next action.

## 2026-08-17 product-profile supersession

Public `cidx init` defaults to 1024, accepts only explicit 1024 or 512, and
exposes no codec flag. Config still records fixed int8 identity. Binary/256
code paths are removed; only historical evidence remains under
[`RETIRED-VECTOR-PROFILES.md`](RETIRED-VECTOR-PROFILES.md).

## 2026-08-21 locator-result supersession

The owner approved a search/read responsibility split after the paired V3
assistant diagnostic. `search` now returns only compact, deduplicated candidate
locators; `read_span` remains the only source-bearing cidx tool. This
supersedes the source-bearing search-response statements below wherever they
conflict, without changing the four tool names, search input fields, ranking,
caller-selected `k`, paid-query guard, or `read_span` safety contract.

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
- The caller supplies required `search.max_inline_bytes` on every request for
  v1 wire compatibility; every accepted value produces the same body-free
  locator projection.
- The server hard maximum governs complete `read_span` source transfer and does
  not alter search ranking or result count.
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
- MCP locator projection of the Phase 11 ranked results without exposing its
  source packaging or diagnostics.
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
7. MCP and search handlers never open or use the product source bank or lab
   DB. Source-bank mutation remains exclusive to explicit CLI embedding paths.

### Search locator and source-response maximum

1. `search.max_inline_bytes` is a required integer at least zero.
2. Search validates the value for compatibility but requests and returns zero
   inline source bytes.
3. The value cannot alter candidate selection, scores, ranks, or the
   IDs/order/count of up to `k` results.
4. `read_span` is the only source-bearing cidx MCP result and applies
   `config.mcp.hard_max_inline_bytes` as an all-or-nothing response ceiling.
5. JSON metadata, escaping overhead, and token counts are excluded from that
   source-byte ceiling.

### Transport and concurrency

1. stdout contains no bytes outside MCP JSON-RPC frames.
2. Responses correlate by request ID and may complete out of request order.
3. One handler's scan, parse, or API wait cannot block dispatch of independent handlers.
4. Cancellation propagates to the application service.
5. Short SQLite writer serialization for index/vector publication is allowed; a dispatcher-wide mutex is not.
6. Search does not duplicate the complete locator JSON across text and
   structured result channels. The verified Codex-compatible representation is
   fixed before the scored V4 run.

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
  Results[]

SearchLocator
  ChunkID / Path / Language / Kind / QualifiedSymbol
  StartLine / EndLine / IndexedSHA256 / MatchSources[]

ReadSpanRequest
  RelativePath / StartLine / EndLine / ExpectedSHA256
```

CLI/MCP parsers perform syntax validation and convert to shared application
request types. MCP invokes the shared search with zero body budget and projects
the already-ranked results into canonical locators. It owns neither candidate
selection nor ranking. Match sources are stable enums derived from the shared
ranked hit; full scores and planner diagnostics stay in local traces.

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

Top-level output contains only `results`. Each result includes `chunk_id`, path,
language, kind, qualified symbol, parent `start_line`/`end_line`,
`indexed_sha256`, and compact `match_sources`. Array order is rank, so there is
no separate rank field.

Canonicalize and deduplicate by indexed content identity, path, parent range,
and qualified symbol while preserving the first ranked occurrence. Merge
stable match sources for duplicates. Do not expose source bodies, signatures,
matched snippets, score values, planner diagnostics, profile fingerprints,
coverage counters, timing, or evaluation metadata. These remain available to
application/evaluation traces. Search requests zero inline source from the
shared response model; `read_span` supplies selected current source after its
hash check.

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
| `mcp.hard_max_inline_bytes` | `read_span` source responses; retained search input validation | Applies on next serve; no profile change |
| Active index/serving profile | Status/search validation | Mismatch causes policy-defined fallback/reconciliation |
| Model/serving-dimension/codec | Embed/search core | Read only through the one Phase 02 profile and `voyage-code-4` spec |

`mcp.hard_max_inline_bytes` defaults to 64 KiB and is a positive server
source-response safety ceiling. Reject startup if it is invalid or exceeds the
code-owned absolute ceiling of 1 MiB. The retained search input accepts any
nonnegative integer allowed by the wire but never causes a body to be returned.
Do not estimate tokenizer counts or host context size to decide source transfer.

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
11. Invoke Phase 11 search with zero inline-body budget, preserve its ranked
    hit order, and project only canonical locator fields.
12. Implement root/path/symlink/hash/range/max checks for `read_span`.
13. Connect `reindex` to the same service as CLI index.
14. Preserve paid guard, missing-key, profile-mismatch, and fallback metadata on the wire.
15. Verify `serve` and MCP handlers do not depend on `internal/lab`; only development bootstrap may do so.
16. On SIGINT/EOF, stop accepting work and cancel in-flight contexts.
17. Generate help/schema examples from one definition or otherwise keep them verifiably synchronized.
18. Probe text-only and structured-only search results through Codex CLI and
    retain exactly one compatible semantic representation.

## 9. Failure, Rollback, Concurrency, and Security

### Failure and rollback

- Config/root/schema mismatch fails before serving requests.
- CLI index and MCP reindex follow Phase 05 rollback on cancellation/failure.
- Locator projection or accounting failure does not rerank; it returns an internal error.
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
3. Requests with max 0, small, sufficient, or above hard max return byte-identical locator IDs/order/count and zero search source bytes.
4. `read_span` source bytes never exceed the configured hard maximum and no source is cut arbitrarily.
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

Assistant reducer v2 checkpoint (2026-08-21):

- quoted shell wrappers are normalized before repository-command
  classification;
- shell, MCP structured, MCP text, result-envelope, and source bytes are
  recorded separately;
- locator and evidence stage metrics have explicit denominators; and
- a disposable complete V3 replay preserved official outcomes/tokens while
  reproducing the corrected event accounting.

Exact evidence: [assistant reducer v2](evidence/phase-13/assistant-reducer-v2.md).

Codex result-representation checkpoint (2026-08-22):

- structured-only and text-only cidx results each completed an isolated real
  search-to-`read_span` journey;
- structured-only was selected because it preserves the typed object without
  duplicating JSON text; and
- the result does not claim compatibility for another host or compare the two
  stochastic turns as an efficiency experiment.

Exact evidence:
[Codex result representation probe](evidence/phase-13/codex-result-representation-probe.md).

Locator-only MCP completion checkpoint (2026-08-22):

- `search` requests zero source bytes from the shared core and emits only nine
  canonical locator fields in one structured representation;
- the adapter preserves ranked identity/parent ranges, deduplicates canonical
  locators, and merges compact match-source labels without exposing ranking or
  planner diagnostics;
- `max_inline_bytes` values 0, 12000, and 65537 produced byte-identical
  10-locator responses with zero source bytes; and
- one real Codex turn used the compact search result, selected two
  `read_span` calls, made no repository shell search, and completed without a
  control violation.

Exact evidence:
[locator-only MCP reconciliation](evidence/phase-13/locator-only-mcp-reconciliation.md).

Current int8-only CLI/MCP acceptance (2026-08-17):

- `cidx init` defaults to 1024/int8, accepts only explicit 1024 or 512, and
  exposes no codec selector. Help presents the default first.
- `cidx embed --apply` publishes compatible source-bank hits locally and
  constructs a Voyage client only when the plan contains missing source
  inputs. This is the provider-free 1024-to-512 rematerialization entry point
  after config edit and index reconciliation.
- MCP still exposes exactly `status`, `search`, `read_span`, and `reindex`.
  MCP/search do not open product source-bank or lab state; the single binary's
  separate public `embed` command owns source reuse.
- Current evidence, including focused normal/race/vet/build and static checks,
  is recorded in
  [int8-only CLI/MCP evidence](evidence/phase-13/int8-only-cli-mcp-reconciliation.md).
  No provider, key, network, corpus, or metric action was performed. That
  evidence remains valid for CLI behavior, tool count, isolation, and
  transport, but its source-bearing search-result wire is superseded and must
  be revalidated before this phase returns to `done`.

Official Phase 12 corpus/usefulness or `core_retrieval` promotion evidence is not a Phase 13 completion condition. Phase 13 must prove adapter/core parity with the corpus-independent core; Phase 14 later references official core and assistant/host evidence for `release_candidate` scope.

- Public and development CLI help snapshots.
- Versioned schemas for exactly four tools.
- Request, response, and typed-error examples.
- Byte-identical locator comparison across `max_inline_bytes` values and zero search source-byte evidence.
- Adapter/core parity evidence proving MCP serialization preserves ranked
  chunk identity and parent ranges while deliberately omitting evaluated body
  packages and diagnostics.
- Concurrent status/reindex/search dispatch trace.
- Cancellation and graceful-shutdown trace.
- stdout protocol-purity capture and stderr-log sample.
- FTS-only operation without an API key.
- Dependency/source inspection proving MCP/search handlers have no source-bank
  or lab access, while the explicit public `embed` path owns source reuse.

Completion reports distinguish the transport, OS, and host actually checked from unverified items.

## 12. Handoff

Phase 14 receives one `cidx` binary and public help, the `cidx serve --root <repository-root>` stdio contract, four tool schemas and example frames, environment/stderr/stdout requirements, project-scoped host command/args/env guidance, an offline FTS smoke procedure, and SQLite/schema/grammar runtime requirements.

## 13. Decision Log

| Decision | Rationale | Revisit when |
| --- | --- | --- |
| Exactly four MCP tools | Keep this auxiliary tool and host context surface small | A measured independent use case exists |
| Retain the required maximum temporarily; no detail enum | Preserve the v1 request wire while the response becomes locator-only | A later explicit MCP schema revision removes the compatibility field |
| Search body removal cannot affect rank | Response compaction must preserve retrieval identity and result order | Core v1 invariant; no planned revisit |
| Project the Phase 11 ranked hits rather than its body package | Offline retrieval retains full diagnostics while the assistant sees only navigation fields | Evaluation proves another locator field is required |
| Separate product source bank from evaluation state | Public embedding retains reusable document f32 without making lab metadata or raw vectors a search dependency | The product source-retention contract changes |
| Expose no source-bank MCP tool | Dimension changes use explicit CLI plan/apply and preserve the four-tool MCP surface | A measured MCP mutation use case is accepted |
| Concurrent dispatcher | Long management calls must not block search at application level | Core concurrency invariant |
| stdout is protocol-only | Prevent stdio frame corruption | Transport changes |
| Do not estimate token budgets | Caller owns tokenizer and host-context composition | Host provides a standard token contract |
| Init discovers Git before config | A new repository has no config yet, while normal serving still needs a configured worktree root | Repository ownership becomes multi-root |
| Search is locator-only; read_span owns source | V3 preserved correctness but expanded 176 source-bearing candidates before 29 selected reads; compact navigation isolates recall from evidence volume | A measured compact assistant run shows source-bearing search is necessary |
