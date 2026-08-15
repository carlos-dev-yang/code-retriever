# 11. Vector Scan and Hybrid Search

- Status: `done`
- Prerequisites: `06-fts-search`, `09-vector-materialization`, `10-embedding-orchestration-and-reconciliation`
- Followed by: `12-retrieval-evaluation`, `13-cli-and-mcp`
- Design source: `local-code-search-mcp-v1-design-r4.md` sections 8 and 9.2

## Context Recovery Checklist

Read the [implementation index](README.md), [execution guide](EXECUTION-GUIDE.md), and [project status](STATUS.md) before resuming.

- Confirm Phase 06 provides deterministic FTS results, Phase 09 provides the shared transform/codec/scorer, and Phase 10 provides one active serving profile plus coverage and reconciliation state.
- Re-check that `mode=fts` never calls Voyage AI and hybrid preflight runs before any paid query request.
- Re-check the query contract: `voyage-code-4`, `input_type=query`, explicit 1024-dimensional float output, `truncation=false`, no `encoding_format`, and the same serving-dimension reduction/normalization as documents.
- Re-check that query text and query f32 are nonpersistent, production search reads one active codec profile (`binary` by default or `int8`), and the shared body packager is transport-independent. Phase 13 owns only MCP schema and transport adaptation.
- Re-check that one search result uses one committed generation/profile snapshot and deterministic scan, aggregation, tie-break, and RRF rules.
- Stop if profile consistency, coverage semantics, query privacy, or snapshot materialization is unresolved. Do not introduce ANN, graph search, caching, or multi-profile serving.
- Before pausing, update this phase's evidence and decision log, then update [STATUS.md](STATUS.md) with checked scenarios, remaining risks, and the next action.

## Entry Gate Record

- Entered: 2026-08-15
- Owner: Codex (`/root/phase11_hybrid_search`)
- Prerequisites checked: Phase 06 deterministic lexical search and pinned snapshot handoff; Phase 09 shared transformer/codecs and production-vector validation; Phase 10 active-profile reconciliation and no-transaction provider-wait boundary. Their recorded evidence is [Phase 06](evidence/phase-06/README.md), [Phase 09](evidence/phase-09/README.md), and [Phase 10](evidence/phase-10/README.md).
- Workspace state: clean before entry; no existing implementation changes were claimed.
- Intended evidence: focused fake-client/core tests for free FTS isolation, preflight fallbacks, exact query request/transform, snapshot consistency, codec scans, deterministic RRF, and body-packaging fidelity. No real provider, credential, corpus, lab runtime access, or paid operation is permitted.

## 1. Objective

Return assistant-oriented code-search results by combining FTS results from the active index snapshot with semantic results from the single codec-specific serving profile selected by current config.

A hybrid query explicitly requests a 1024-dimensional f32 embedding from Voyage AI `voyage-code-4`, applies the same `prefix(serving_dimensions) -> L2` transform as documents, and keeps query f32 out of the production DB, raw lab, and evaluation artifacts.

## 2. Scope and Non-goals

### In scope

- Explicit paid boundary between `fts` and `hybrid` modes.
- Hybrid preflight and fallback before an API call.
- Shared serving-space transform for 1024-dimensional query f32.
- Full scan of current active binary or int8 vectors using the matching scorer.
- Segment-score aggregation to source chunks.
- Deterministic RRF across FTS and vector rankings.
- Rank-invariant indexed-body packaging under an already validated effective byte maximum.
- Partial coverage, fallback reason, and profile metadata.
- Generation-consistent ranked results, packaged indexed bodies, and freshness inputs.

### Out of scope

- Raw-lab access, query/result caches, or automatic document embedding.
- Any API source output other than 1024 dimensions.
- HNSW, graph traversal, or remote vector databases.
- Learned reranking or generated summaries.
- Numeric latency/hit-rate release gates.
- Concurrent serving profiles or A/B traffic splits.

## 3. Prerequisites

- FTS search, source chunks, and segment mappings are available from one active-generation snapshot.
- Phase 09 implements the shared `VectorTransformer`, `VectorCodec`, and codec-aware scoring.
- Phase 10 exposes active `ServingVectorProfile`, valid coverage, and reconciliation state.
- The Voyage query client is injected separately from document request orchestration.
- The r3 design defines `max_inline_bytes`, current-source hash checks, and `read_span`. This phase owns the transport-independent body packager so Phase 12 evaluates the same path that MCP later exposes. Phase 13 validates the caller/server maximum, annotates live freshness, and owns `read_span` plus MCP serialization.

## 4. Invariants

1. `mode=fts` never calls Voyage AI, regardless of credentials.
2. `mode=hybrid` calls no API until paid-query permission, profile agreement, and at least one active valid vector are confirmed.
3. Query requests require f32 matching `EmbeddingSourceProfile.SourceDimensions=1024`.
4. Queries use the document `VectorSpaceProfile.ServingDimensions`, never a separate setting.
5. Query and document use the same source profile, reducer, normalizer, serving dimensions, and metric.
6. Query vectors remain serving-space f32 or a transient codec-specific query representation in request memory and are never persisted.
7. Query text and vectors never enter caches, DBs, lab storage, or debug dumps.
8. The scanner accepts only rows passing active storage fingerprint, dimensions, codec, blob, scale/norm, and finite checks.
9. Manifest, FTS state, chunks, segments, vectors, coverage, and response metadata for one search come from one committed generation/profile.
10. If generation or profile changes after the API call, discard the query vector instead of forcing it into a new space.
11. Hybrid failure does not fail FTS; it returns an explicit fallback reason.
12. Map order, SQLite row order, and scheduling cannot affect RRF output.
13. Coverage counts active segment references, not unreferenced cache rows.
14. Raw-lab state cannot influence runtime results.
15. Explicitly send `input_type=query`, `output_dimension=1024`, `output_dtype=float`, and `truncation=false`; omit `encoding_format`.
16. Use a response only after model, count, unique in-range indexes, 1024 dimensions, and finite values validate.
17. Voyage-native quantization is neither the document codec nor a query shortcut.
18. Body packaging runs only after final ranking and cannot change candidate selection, rank, result identity, order, or the up-to-k result count.
19. The body packager consumes indexed-snapshot bytes only. It returns a complete parent chunk, a complete matched segment, or no body; it never cuts arbitrary UTF-8 bytes.

## 5. Implementation Packages, Files, and Types

| Package/file | Responsibility |
| --- | --- |
| `internal/embedclient/query.go` | Query-embedding port and Voyage adapter boundary |
| `internal/search/query_embedding.go` | Query formatting, response validation, and shared transform |
| `internal/search/preflight.go` | Mode, paid permission, profile, and coverage decision |
| `internal/search/vector_scan.go` | Validated codec-aware scan and top-candidate retention |
| `internal/search/segment_score.go` | Shared-key fan-out and chunk aggregation |
| `internal/search/rrf.go` | Deterministic reciprocal-rank fusion |
| `internal/search/inline_body.go` | Transport-independent, rank-invariant indexed-body allocation |
| `internal/search/service.go` | Snapshot boundaries and fallback orchestration |
| `internal/store/search_snapshot.go` | Consistent FTS/vector/chunk materialization |
| `internal/app/search.go` | Request validation and response model |

Core types are `SearchMode(fts|hybrid)`, `HybridPreflight`, ephemeral `QueryEmbedding`, immutable `SearchSnapshot`, `VectorCandidate`, `ChunkVectorScore`, `RankedChunk`, `BodyBudget`, `PackagedSearchResult`, and stable `FallbackReason`.

## 6. Schema, API, and CLI

### Hybrid preflight

Before calling the API, a short production read verifies:

1. Requested mode is hybrid.
2. `allow_paid_query_embedding` is true.
3. Desired and active source/space/storage fingerprints match.
4. At least one valid vector is referenced by an active segment.
5. The API key and client can initialize.

Any failure continues with FTS only and makes no query API call.

### Query embedding API

- Format query text deterministically under the query-formatter version.
- Use code-owned `https://api.voyageai.com/v1/embeddings`, `Authorization: Bearer $VOYAGE_API_KEY`, and active `voyage-code-4`.
- Explicitly send `input_type="query"`, `output_dimension=1024`, `output_dtype="float"`, and `truncation=false`; omit `encoding_format` and never rely on provider defaults.
- Validate response model/count/index uniqueness and range, dimensions, and NaN/Inf; restore input order by index.
- Use `VectorTransformer` for prefix reduction and L2 normalization into the active serving space.
- Keep serving f32 in request memory only and always request source 1024 regardless of the active serving dimension.
- Treat a context overflow under `truncation=false` as `QUERY_EMBEDDING_FAILED`; never use a partial query.

### Snapshot and scoring order

1. Make the query API call outside SQLite transactions.
2. After success or fallback decision, open the search read transaction.
3. Read generation, manifest, and serving fingerprints again.
4. If query fingerprints no longer match, discard it and materialize an FTS-only snapshot in the same transaction.
5. Read FTS `candidate_k` and BM25 metadata.
6. Materialize active segment references, valid vectors, and chunks into one immutable in-memory representation.
7. Compute coverage from that snapshot and close the transaction.
8. Run CPU-heavy codec-aware scanning, aggregation, and RRF outside the transaction.
9. After ranking, package indexed-snapshot bodies under the effective byte maximum.
10. Close the read snapshot after every source byte needed by the packaged results has been copied. Check current hashes only after that, outside the core ranking/body snapshot.

Never supplement a closed snapshot by reading chunks or vectors from a newer generation.

### Vector scan and aggregation

- Score a canonical-input vector once and fan it out when multiple segments share it.
- Use codec scale/norm and the configured metric.
- A source chunk's semantic score is its highest segment score.
- Resolve ties by an explicit stable order such as score, normalized path, start byte, then chunk ID.
- Read vector candidate count from resolved config; do not hard-code it in the scan loop.

### RRF and MCP handoff

- FTS and vector ranks start at 1; chunks appearing in only one list remain eligible.
- Read the RRF constant from config. Break fused-score ties by lexical rank, vector rank, then stable chunk key.
- Do not introduce arbitrary symbol multipliers.
- MCP inputs remain `query`, optional `k`, optional `mode`, and required `max_inline_bytes`; expose no model/dimension/codec override.
- Return requested/effective mode, `query_embedding_used`, stable fallback reason, active fingerprints, coverage numerator/denominator, generation/manifest, and per-hit lexical/vector ranks plus fused score. If API usage is exposed, include query input tokens but never vector values.
- Treat codec scores as codec-specific ranking inputs. Do not expose binary Hamming/asymmetric or int8 reconstructed scores as exact cosine; RRF consumes ranks, so cross-codec raw-score calibration is not required.

### Indexed-body packaging

- The caller-facing layer supplies an already validated `effective_max_inline_bytes`; the shared packager does not clamp configuration or parse MCP input.
- Allocate body bytes in final rank order. Include the full parent chunk when it fits; otherwise include the complete winning matched segment when one exists and fits; otherwise omit the body.
- FTS-only results have no guaranteed matched segment. If the full parent chunk does not fit, omit the body rather than inventing an arbitrary excerpt.
- Count actual raw UTF-8 source bytes placed in result bodies. JSON metadata, escaping overhead, and tokens are not part of this budget.
- A zero budget returns the identical up-to-k ranked result identities/order/count with metadata and no bodies.
- Return body range, completeness, bytes used, and a stable omission reason so Phase 12 can attribute `BODY_PACKAGING` loss and Phase 13 can serialize without reimplementing policy.

Stable fallback values are `PAID_QUERY_DISABLED`, `PROFILE_RECONCILIATION_REQUIRED`, `NO_VALID_DOCUMENT_VECTORS`, `API_KEY_MISSING`, `QUERY_EMBEDDING_FAILED`, `QUERY_PROFILE_CHANGED_DURING_REQUEST`, and `VECTOR_SNAPSHOT_INVALID`. Never infer them by parsing error text.

## 7. Configuration and Change Impact

Embedding inputs are model, serving dimensions, reducer, normalizer, metric, storage codec, and code-defined query-formatter version. `voyage-code-4` is the v1 default and only initially validated model. `ModelSpec` resolves source 1024 and allowed serving dimensions `{256,512,1024}`; source dimensions are not configurable.

Search defaults to `fts`; hybrid is explicit or selected only by a later user configuration change. Search inputs are paid-query permission, default/maximum `k`, FTS/vector `candidate_k`, RRF constant, and caller/server inline-source limits.

| Change | Document vectors | Query behavior | FTS |
| --- | --- | --- | --- |
| Source/space/storage profile | Reconciliation required | No paid call until matching profile is active | Remains available |
| Query formatter | No rebuild | Applies to the next query | No effect |
| Candidate/RRF policy | No rebuild | Ranking changes | Reuses index |
| Paid permission | No rebuild | API gate changes | No effect |
| Inline byte maximum | No rebuild | Shared body packaging only | Ranking unchanged |

The service uses immutable `ResolvedConfig` created at server start rather than rereading config per request, and compares it with active DB fingerprints.

## 8. Ordered Implementation Checklist

1. Implement `HybridPreflight` and stable fallback enums.
2. Keep FTS operational with no embedding-client dependency.
3. Implement the query formatter and Voyage 1024 request builder.
4. Share response validation with the document path.
5. Connect the shared reducer and L2 normalizer.
6. Revalidate the active profile after the API call.
7. Implement generation-consistent `SearchSnapshot` loading.
8. Validate vector fingerprint, dimensions, codec, blob, and scale/norm during load.
9. Implement distinct-vector scoring and segment fan-out.
10. Implement max-segment aggregation and deterministic top-k.
11. Implement deterministic RRF and tie-breaks.
12. Add partial-coverage and response metadata.
13. Ensure API failure or vector corruption degrades hybrid rather than FTS.
14. Close read transactions before CPU scanning.
15. Expose the existing search contract through MCP without profile overrides.
16. Implement the shared body packager and stable omission reasons after final ranking.
17. Hand packaged indexed results and freshness inputs to Phase 13 without MCP-specific types.
18. Propagate cancellation through API, snapshot read, vector scan, and body copying.

## 9. Failure, Rollback, Concurrency, and Security

### Failure and fallback

- Invalid query input returns a request error without calling the API.
- Permission, profile, coverage, or credential preflight failures fall back to FTS.
- Timeout, invalid response, and transform failures also fall back; never use a partial query vector.
- Do not silently skip corrupt rows and inflate coverage. The default integrity policy falls back the whole vector path and records diagnostics.
- Only an FTS failure fails the overall search. Search writes nothing and has no rollback.

### Concurrency

- Hold no SQLite read/write transaction during the query API call.
- A slow API request does not serialize other FTS handlers.
- Discard the paid result and fall back if the profile changes after the API call.
- Close the snapshot transaction before a long CPU scan so it does not retain a WAL checkpoint.
- Cancellation stops the API and scan without mutating shared server state.

### Security

- Configuration, help, and tool descriptions disclose that hybrid sends query text to Voyage AI.
- Never persist query text or vectors in DBs, lab data, metrics artifacts, or debug logs.
- Never expose keys or provider error bodies in MCP responses.
- Body output obeys caller and server hard limits; the vector path cannot bypass them.
- Body allocation reads only the indexed snapshot and never substitutes live worktree bytes.

## 10. Validation Scenarios

- FTS results are identical when no embedding client is injected.
- Disabling paid queries produces a stable fallback and zero API calls.
- Zero valid vectors prevents an API call.
- A 1024-dimensional query transforms to exactly the active document serving dimensions.
- Dimension or codec mismatch reads no old vector and calls no query API.
- A profile change during the API call discards the query and does not mix profiles.
- Partial-coverage numerator and denominator agree with active segment references.
- A shared vector is scored once and produces the same chunk aggregation.
- SQLite row reordering does not change top-k or RRF ties.
- Removing or locking the raw lab does not alter search.
- No query f32 row appears in either DB before or after a query.
- Concurrent publication during CPU scanning never mixes generations within one response.
- Zero, small, sufficient, and clamped-effective body maxima leave ranked IDs/order/count unchanged.
- Every non-null body exactly matches its indexed parent or matched-segment range, and the aggregate raw UTF-8 byte count does not exceed the effective maximum.
- FTS-only hits whose full parent does not fit have no invented matched excerpt.

## 11. Completion Evidence

Implementation and main-agent acceptance record (2026-08-15; details in [Phase 11 evidence](evidence/phase-11/README.md)):

- Added the transport-independent `internal/search` core and one production-only pinned snapshot loader in `internal/store/hybrid_snapshot.go`. Runtime search neither imports nor opens the lab database, and it has no write path.
- Hybrid preflight uses one pinned read-only snapshot to check paid permission, desired versus active source/space/storage fingerprints, and every present active referenced vector row before a provider request. Missing rows remain partial coverage; zero valid rows and malformed rows respectively yield `NO_VALID_DOCUMENT_VECTORS` and `VECTOR_SNAPSHOT_INVALID` with zero provider calls.
- Query embedding uses the existing Voyage response validator and a single injected fake client in tests. The request is the source-1024 float query role, then the existing shared transformer produces only request-local serving f32. No provider-native quantization path is used.
- The post-request snapshot copies FTS candidates, active segment references, validated stored vectors, bodies, coverage, generation, manifest, and fingerprints from one read transaction; CPU scan, RRF, and body packaging run after it closes. Generation/manifest/profile drift discards the query result and returns FTS with `QUERY_PROFILE_CHANGED_DURING_REQUEST`.
- Binary and int8 scans score each distinct canonical input key once, fan scores out to segments, choose the maximum parent score, apply stable candidate ordering, and use deterministic one-based-rank RRF. Corruption falls back the full vector lane rather than inflating coverage.
- The shared body packager preserves result identity/order/count for zero, small, and sufficient raw-UTF-8 budgets. It returns only complete indexed parents, complete vector-winning segments, or no body; FTS-only does not create an excerpt.
- Acceptance follow-up: explicit FTS and every fallback now use a separate lexical-only snapshot that reads metadata, FTS candidates, and their indexed bodies only. It does not materialize vectors, segments, coverage, or vector payloads; response metadata records that vector coverage was not observed.
- Acceptance follow-up: the hybrid snapshot stores each parent chunk/body once and each active canonical vector key once. Segment references contain only a parent/key/display-range link; one score is computed per key and then fanned out.
- Acceptance follow-up: `QueryTextFormatVersion=1` is code-owned runtime policy, included in the serving-policy fingerprint and core response. v1 strictly validates then preserves query UTF-8 bytes. Query calls use the resolved request timeout; provider timeout, provider error, response validation, or transform failure uses the ordinary FTS fallback while caller cancellation remains cancellation.

Focused checks actually run:

```text
gofmt -w internal/search internal/store/hybrid_snapshot.go
go test -count=1 ./internal/search
go test -count=1 -race ./internal/search ./internal/store ./internal/config ./internal/vector ./internal/embedclient ./internal/search/lexical
go vet ./internal/search ./internal/store ./internal/config ./internal/vector ./internal/embedclient ./internal/search/lexical
go build ./internal/search ./internal/store ./internal/config ./internal/vector ./internal/embedclient ./internal/search/lexical
go list -deps ./internal/search ./internal/store | rg 'cidx/internal/lab'   # no matches
git diff --check
```

The focused fake-client tests cover FTS zero calls and lexical-only isolation from corrupt vector/segment rows; disabled, unready, missing-client, profile-mismatch, provider-failure, timeout, and corrupt-row fallbacks; exact query request fields and 1024-to-serving transform; partial coverage; binary/int8 deterministic shared-key collapse and RRF ties; post-request generation drift; independent FTS while a query client blocks; no query write to `vector_cache` or lab state; and zero/small/full aggregate body budgets with exact UTF-8 segment bytes/ranges and no FTS-only invented excerpt.

Not run: a real Voyage request, API-key retrieval, network access, corpus selection/access, paid operation, raw-lab runtime access, CLI/MCP transport, `read_span`, Phase 12 evaluation, full-project validation, load testing, or a live publish-concurrency benchmark. Deduplicated parent-body copying and brute-force scan have no latency/memory guarantee; Phase 12 must measure them. The main-agent acceptance review inspected the Phase 11 diff and reran the focused race, vet, build, format, dependency-boundary, and diff checks before changing this phase to `done`.

- API-call count table by mode and preflight condition.
- Query 1024 -> serving dimensions/norm validation.
- Corrupt-row fallback and coverage result.
- Deterministic vector top-k and RRF checksum.
- Evidence of no generation/profile mixing during concurrent reconciliation and publication.
- An independent FTS request completing during a query API wait.
- Schema/file inspection proving no stored query vector.
- Comparable FTS-only and hybrid response metadata for one query.
- Same-rank/body-budget evidence plus exact indexed-body fidelity, byte accounting, and omission-reason records.

## 12. Handoff

Phase 12 calls this exact transform, scorer, aggregation, RRF, and body-packaging implementation. It must not create a second evaluation-only ranker or packager. Its only additional path is a development-only in-memory serving-f32 document baseline sourced from the raw bank; production search cannot import or select that path.

## 13. Decision Log

- Hybrid is auxiliary; FTS always works independently.
- Every query gets a fresh nonpersistent `voyage-code-4` 1024 f32 embedding.
- Fixed evaluation questions do not justify a runtime query cache.
- Documents and queries do not have separate serving dimensions or source-output paths.
- Production reads one active binary or int8 profile and uses shared Voyage Matryoshka prefix plus L2 followed by the active codec's scorer contract.
- Start with brute-force scan; decide on ANN or graphs only after measurement.
- Observe latency and hit rate, but do not make them numeric completion gates here.
- Own body allocation in the search core so offline evaluation and MCP transport observe one policy; Phase 13 only validates transport input and serializes the packaged result.
- Query-client availability is an injected capability: the core neither reads an API key nor constructs a real provider client. A missing injected client maps to `API_KEY_MISSING` without a provider call.
- A corrupt active vector row invalidates the whole vector lane for that search, including partial-coverage searches. Missing rows remain ordinary partial coverage.
- Profile, generation, or manifest movement between an approved query request and the pinned materialization snapshot discards the transient query vector rather than attempting cross-snapshot reuse.
