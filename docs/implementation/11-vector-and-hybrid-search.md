# 11. Vector Scan and Hybrid Search

- Status: `planned`
- Prerequisites: `06-fts-search`, `09-vector-materialization`, `10-embedding-orchestration-and-reconciliation`
- Followed by: `12-retrieval-evaluation`, `13-cli-and-mcp`
- Design source: `local-code-search-mcp-v1-design-r3.md` sections 8 and 9.2

## Context Recovery Checklist

Read the [implementation index](README.md), [execution guide](EXECUTION-GUIDE.md), and [project status](STATUS.md) before resuming.

- Confirm Phase 06 provides deterministic FTS results, Phase 09 provides the shared transform/codec/scorer, and Phase 10 provides one active serving profile plus coverage and reconciliation state.
- Re-check that `mode=fts` never calls Voyage AI and hybrid preflight runs before any paid query request.
- Re-check the query contract: `voyage-code-4`, `input_type=query`, explicit 1024-dimensional float output, `truncation=false`, no `encoding_format`, and the same target reduction/normalization as documents.
- Re-check that query text and query f32 are nonpersistent, production search reads one active codec profile (`binary` by default or `int8`), and the shared body packager is transport-independent. Phase 13 owns only MCP schema and transport adaptation.
- Re-check that one search result uses one committed generation/profile snapshot and deterministic scan, aggregation, tie-break, and RRF rules.
- Stop if profile consistency, coverage semantics, query privacy, or snapshot materialization is unresolved. Do not introduce ANN, graph search, caching, or multi-profile serving.
- Before pausing, update this phase's evidence and decision log, then update [STATUS.md](STATUS.md) with checked scenarios, remaining risks, and the next action.

## 1. Objective

Return assistant-oriented code-search results by combining FTS results from the active index snapshot with semantic results from the single codec-specific serving profile selected by current config.

A hybrid query explicitly requests a 1024-dimensional f32 embedding from Voyage AI `voyage-code-4`, applies the same `prefix(target) -> L2` transform as documents, and keeps query f32 out of the production DB, raw lab, and evaluation artifacts.

## 2. Scope and Non-goals

### In scope

- Explicit paid boundary between `fts` and `hybrid` modes.
- Hybrid preflight and fallback before an API call.
- Shared target-space transform for 1024-dimensional query f32.
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
- The Voyage query client is injected separately from document batch orchestration.
- The r3 design defines `max_inline_bytes`, current-source hash checks, and `read_span`. This phase owns the transport-independent body packager so Phase 12 evaluates the same path that MCP later exposes. Phase 13 validates the caller/server maximum, annotates live freshness, and owns `read_span` plus MCP serialization.

## 4. Invariants

1. `mode=fts` never calls Voyage AI, regardless of credentials.
2. `mode=hybrid` calls no API until paid-query permission, profile agreement, and at least one active valid vector are confirmed.
3. Query requests require f32 matching `EmbeddingSourceProfile.SourceDimensions=1024`.
4. Queries use the document `VectorSpaceProfile.TargetDimensions`, never a separate setting.
5. Query and document use the same source profile, reducer, normalizer, target dimensions, and metric.
6. Query vectors remain target-space f32 or a transient codec-specific query representation in request memory and are never persisted.
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
- Use `VectorTransformer` for prefix reduction and L2 normalization into the active target space.
- Keep target f32 in request memory only and always request source 1024 regardless of active target.
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

Embedding inputs are model, target dimensions, reducer, normalizer, metric, storage codec, and code-defined query-formatter version. `voyage-code-4` is the v1 default and only initially validated model. `ModelSpec` resolves source 1024 and allowed targets `{256,512,1024}`; source dimensions are not configurable.

Search inputs are default mode, paid-query permission, default/maximum `k`, FTS/vector `candidate_k`, RRF constant, and caller/server inline-source limits.

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
- A 1024-dimensional query transforms to exactly the active document target dimensions.
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

- API-call count table by mode and preflight condition.
- Query 1024 -> target dimensions/norm validation.
- Corrupt-row fallback and coverage result.
- Deterministic vector top-k and RRF checksum.
- Evidence of no generation/profile mixing during concurrent reconciliation and publication.
- An independent FTS request completing during a query API wait.
- Schema/file inspection proving no stored query vector.
- Comparable FTS-only and hybrid response metadata for one query.
- Same-rank/body-budget evidence plus exact indexed-body fidelity, byte accounting, and omission-reason records.

## 12. Handoff

Phase 12 calls this exact transform, scorer, aggregation, RRF, and body-packaging implementation. It must not create a second evaluation-only ranker or packager. Its only additional path is a development-only in-memory target-f32 document baseline sourced from the raw bank; production search cannot import or select that path.

## 13. Decision Log

- Hybrid is auxiliary; FTS always works independently.
- Every query gets a fresh nonpersistent `voyage-code-4` 1024 f32 embedding.
- Fixed evaluation questions do not justify a runtime query cache.
- Documents and queries do not have separate target dimensions or source-output paths.
- Production reads one active binary or int8 profile and uses shared Voyage Matryoshka prefix plus L2 followed by the active codec's scorer contract.
- Start with brute-force scan; decide on ANN or graphs only after measurement.
- Observe latency and hit rate, but do not make them numeric completion gates here.
- Own body allocation in the search core so offline evaluation and MCP transport observe one policy; Phase 13 only validates transport input and serializes the packaged result.
