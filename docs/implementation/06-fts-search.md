# 06. Free FTS5 Search

- Status: `done`
- Prerequisite phase: `05-worktree-index-pipeline`
- Follow-up phases: `07-lexical-evaluation`, `11-vector-and-hybrid-search`
- Design basis: `local-code-search-mcp-v1-design-r4.md` §5, §8

## Context Recovery Checklist

- Reopen the [implementation index](README.md), [execution guide](EXECUTION-GUIDE.md), and [status ledger](STATUS.md) before continuing.
- Confirm the Phase 05 active-generation schema, authoritative `chunks` rows, contentless-first `chunk_fts`, FTS rowid-to-chunk mapping, shared identifier normalizer, and generation-pinned read-snapshot API are present.
- Re-check these invariants after any context compaction: FTS-only search is free and creates no external client; raw user text never becomes FTS grammar; candidates and chunk metadata come from one generation and one read transaction; ranking, deduplication, and ties are deterministic; an empty normalized query never becomes a table scan.
- Stop if SQLite lacks the selected FTS5 tokenizer, the Phase 05 FTS/chunk relationship is ambiguous or corrupt, safe query construction cannot represent an input, or a single-generation read cannot be guaranteed.
- Before pausing, record executed evidence in §11, capture new architectural choices in §13, and update [STATUS.md](STATUS.md) with the exact next checklist item and unresolved stop condition.

## 1. Goal

Implement a fully local lexical path that searches the functions, methods, and types in the Phase 05 active generation with SQLite FTS5 and BM25.

The result is the default search engine that works without an API key or network access. Later hybrid search combines this output with vector rankings rather than replacing it, so FTS must be a useful first-class retrieval layer rather than merely a fallback.

## 2. Scope

### Included

- Normalize identifier and general-text queries.
- Split camelCase, PascalCase, and snake_case identifiers.
- Convert user input into a safe FTS `MATCH` expression.
- Search the `symbols` and `body` fields with BM25.
- Apply a small qualified-symbol exact-match tie-break.
- Enforce deterministic ranking and candidate limits.
- Read candidates and chunk metadata from the same snapshot as the active generation.
- Provide an FTS-only application search API.
- Return lexical-search diagnostics.

### Out of scope

- Query or document embedding.
- vector scan and RRF.
- Voyage API calls.
- Automatic reindexing or a full-file freshness scan on search.
- Search-result body-byte allocation and the MCP wire schema.
- Semantic language analysis, call graphs, or LSP symbol lookup.
- Per-query boosts tuned to the evaluation set.
- Numeric hit-rate or latency acceptance thresholds.

## 3. Prerequisites

- Phase 01 confirmed that the distributed SQLite contains FTS5 and supports the selected tokenizer.
- The Phase 02 config, profile, and store reader are available.
- The Phase 02 shared identifier normalizer and lexical settings, including `search.fts_weights`, are available.
- Phase 05 publishes contentless-first `chunk_fts` rows and authoritative chunk rows in the same active snapshot.
- The symbol/body projections have already removed duplicate nested-method bodies according to the Phase 03 and Phase 04 rules.
- A store API can pin the read transaction and generation used by one search.

## 4. Invariants

1. FTS-only search does not reference the Voyage client, an API key, or the network.
2. A raw user query is never executed directly as `MATCH` syntax.
3. Parameter binding is necessary but not sufficient: the bound MATCH value must be an allowed token-and-phrase expression produced only by the internal query builder.
4. Search units are source chunks for functions, methods, and types. Exported const/var declarations are not independent results.
5. An indexed type body does not duplicate the implementation bodies of its nested methods.
6. One search reads the active generation, FTS statistics and candidates, and chunk metadata within the same SQLite read transaction.
7. `candidate_k` limits only the number of candidates; it does not control how many indexed-body bytes are returned.
8. Absolute BM25 values are not an external compatibility contract. Stable rank and score source are the supported outputs.
9. Ties use stable keys such as path, qualified symbol, and chunk ID so result order is reproducible.
10. An empty query, or a query with no tokens after normalization, returns an explicit input error rather than becoming a full-table scan.

## 5. Packages, Files, and Types to Implement

```text
internal/
  symbol/
    query_tokens.go          # query token classes using the Phase 02 normalizer
  search/
    lexical/
      service.go             # LexicalSearcher implementation
      query.go               # safe MATCH-expression builder
      rank.go                # BM25 plus exact-symbol tie-break
      result.go              # LexicalHit and diagnostics
  store/
    search_snapshot.go       # generation-pinned read scope
    fts_query.go             # FTS SQL and chunk materialization
```

Recommended core types:

```text
NormalizedQuery
  Original
  IdentifierTokens
  TextTokens
  ExactSymbolCandidate(optional)
  MatchExpression

LexicalSearchRequest
  Query
  CandidateK
  SnapshotConstraints

LexicalHit
  ChunkID
  Path / Kind / Symbol / QualifiedSymbol / Signature
  Parent line/byte range
  BM25Rank
  ExactSymbolMatched

LexicalSearchResult
  IndexGeneration
  ManifestSHA256
  Hits
  CandidateCount
  Diagnostics

LexicalSearcher
  Search(ctx, snapshot, request) -> LexicalSearchResult
```

`LexicalSearcher` knows nothing about MCP requests or vectors. The Phase 11 hybrid orchestrator and Phase 13 MCP handler consume the same interface.

## 6. Schema, Internal API, and CLI Contract

### 6.1 FTS schema

Fix the logical fields to these two:

- `symbols`: original symbol, qualified symbol, and decomposed identifier forms
- `body`: signature and retrieval projection body

The FTS table is not the authoritative store for source bodies. Prefer contentless FTS5 and connect the FTS rowid to the authoritative `chunks` row. Phase 05 must delete or replace FTS rows within its publish transaction so no orphan remains.

Include the tokenizer name and options plus the FTS schema version in the index profile. Values used only at query time, such as BM25 field weights and candidate count, belong to serving config.

### 6.2 Query builder

Process a query in this order:

1. Validate UTF-8 and the maximum query-byte policy.
2. Separate identifier-like tokens from general text tokens.
3. Produce the original identifier form and its case/separator decomposition.
4. Validate allowed characters and token lengths, then remove empty tokens.
5. Escape or treat as ordinary text any character that could be interpreted as an FTS operator, quote, wildcard, or column selector.
6. Build the phrase/token combination under internal policy and pass it as one bound MATCH parameter.
7. Use the exact-symbol candidate only in a separate normalized-equality comparison.

Do not convert unsupported syntax automatically into a broad OR query. Diagnostics may retain normalized tokens and the selected internal query shape, but logs must not contain source bodies or secrets.

### 6.3 SQL and ranking

- Call FTS5 `bm25()` with the resolved field weights.
- Normalize BM25 direction and the SQLite return representation once in the store adapter.
- Apply only the small deterministic qualified-symbol exact-match tie-break allowed by config.
- Produce an ordinal lexical rank for later RRF consumption.
- Return each source chunk at most once.
- Include fixed stable sort keys in the final SQL.

### 6.4 Call surface

- An internal CLI or development evaluator may call `LexicalSearcher` directly.
- MCP `search(mode=fts)` is connected in Phase 13 through the application search service.
- Adding a separate public `cidx search` command is not part of the v1 contract.

## 7. Config Used and Change Impact

Pass every value through a `LexicalSearchConfig` derived from the Phase 02 immutable `ResolvedConfig`.

| Setting | Classification | Change impact |
| --- | --- | --- |
| FTS schema/tokenizer implementation ID | index profile | requires a full local reindex |
| symbol-normalization implementation ID | index profile | requires a full local reindex |
| `search.candidate_k` | serving policy | applies on the next search; no reindex |
| `search.return_k` | serving policy | upper application default; no reindex |
| BM25 symbol/body weights | serving policy | apply on the next search; no reindex |
| exact-symbol tie-break | serving policy | applies on the next search; no reindex |
| query byte/token safety limits | operational/safety | apply on the next serve; no reindex |

Implementation packages must not define arbitrary default numbers. Use only named defaults from the config resolver and hard safety caps. Keep evaluation overrides as an explicit development-command profile; never mutate production config silently during a run.

## 8. Ordered Implementation Checklist

1. Verify Phase 05 FTS rowid-to-chunk linkage and delete/update behavior.
2. Observe `symbols` and `body` tokenizer behavior directly on a small Go/TypeScript/TSX corpus.
3. Connect the Phase 02 shared identifier normalizer to query-token classification and verify it produces the same fixture results as the Phase 05 index input.
4. Consume the shared normalizer contract for identifiers containing Unicode, acronyms, numbers, and separators; do not create another implementation here.
5. Implement query-length, UTF-8, and empty-query validation.
6. Implement a safe query builder that does not expose raw FTS syntax.
7. Implement the FTS candidate query inside the store's generation-pinned read scope.
8. Inject BM25 field weights and the exact-qualified-symbol tie-break from resolved config.
9. Implement stable secondary sorting and deduplication.
10. Copy `LexicalHit` and authoritative chunk metadata into memory within the same read transaction.
11. Close the transaction before returning the immutable result to external callers.
12. Define the error taxonomy for empty input, no hits, and malformed internal state.
13. Preserve rank and diagnostics for the Phase 07 evaluator.
14. Stabilize the interface that supplies ordinal lexical ranks to Phase 11 RRF.

## 9. Failure, Rollback, Concurrency, and Security

### Failure and rollback

- Search in this phase is read-only, so there is no DB state to roll back.
- Distinguish user-correctable errors such as `INVALID_QUERY` and `EMPTY_QUERY` after normalization.
- If an internally built MATCH expression causes a SQLite syntax error, record it as an implementation/index error rather than disguising it as user input failure.
- If an FTS row cannot resolve to its chunk row, fail the search with `INDEX_CORRUPT` diagnostics rather than silently dropping the hit.
- When profile reconciliation is required, the upper service decides whether the active FTS remains usable and returns mismatch metadata.

### Concurrency

- Each search uses an independent reader connection and a short read transaction.
- A search that starts concurrently with index publish sees either the old or the new snapshot under SQLite snapshot isolation.
- Copy candidates and chunk metadata into memory, then close the transaction.
- Perform query normalization and post-ranking outside the DB transaction.
- Search does not acquire `index.lock` or the embedding lock.

### Security

- Bind all SQL values as parameters.
- Generate FTS grammar only through the allowlisted builder.
- Never pass the query to a shell, Git command, or file path.
- Enforce executable-owned absolute caps below configurable excessive query-length and token-count limits.
- Do not write full source bodies to diagnostic logs.
- Preserve package dependencies that prevent the FTS-only path from creating an external client.

## 10. Validation Scenarios

1. Find the correct chunk by function, method, type name, and qualified symbol.
2. Map `GetUserByID`, `get user by id`, and `get_user_by_id` through the intended decomposed forms.
3. Find relevant functions for a natural-language body query through signatures and projections.
4. Ensure type search does not over-rank the same implementation because nested method bodies were duplicated.
5. Ensure FTS-like input such as quotes, `*`, `:`, `NEAR`, and parentheses causes neither injection nor a syntax error.
6. Handle Unicode symbols and valid UTF-8 queries deterministically.
7. Reject empty strings, whitespace, punctuation-only input, and excessive queries without a full scan.
8. Reproduce the same rank order for identical DB, config, and query inputs using stable tie-breaks.
9. Ensure concurrent searches before and after active-generation publish never observe mixed FTS/chunk state.
10. Complete search without an API key or network access.
11. Confirm the exact-symbol tie-break is not an arbitrary multiplicative boost that overwhelms ordinary BM25 results.
12. Fail closed when an intentionally corrupted diagnostic database breaks FTS-row-to-chunk linkage.

## 11. Completion Evidence

- Representative identifier-normalization table.
- Safe-query-builder input-to-internal-MATCH transformation table.
- Lexical results and ranking diagnostics for Go, TypeScript, and TSX corpora.
- Stable-ordering record from repeated identical queries.
- Dependency and execution evidence that no API client was loaded or called by FTS-only search.
- Generation-consistency record from concurrent publish and search.
- Failure records for query injection, length caps, and a corrupt index.
- Package-wiring evidence that one resolved-config path supplies every search constant.

Do not use numeric hit rate or latency as this phase's completion condition. Pass observed measurements to Phase 07 as a baseline.

## 12. Follow-up Handoff

Provide Phase 07 with:

- normalized-query diagnostics;
- lexical ordinal ranks and hit metadata;
- deterministic config, profile, and manifest identifiers;
- a distinction between no-hit and error outcomes; and
- a read-only API with which the evaluator can repeat FTS-only searches.

Provide Phase 11 with `LexicalSearcher` and the ordinal lexical rank for each source chunk. Hybrid implementation consumes this result as an RRF input rather than duplicating the lexical path.

## 13. Decision Log

| Decision | Reason | Revisit when |
| --- | --- | --- |
| Keep FTS-only as the default independent search path. | It is free and must search the latest local index even when vectors are unavailable. | Never for the v1 core contract. |
| Use source chunks as FTS units. | This matches the product unit returned to users: function, method, or type source. | Evaluation demonstrates a need for a smaller lexical unit. |
| Do not expose raw MATCH syntax. | This reduces injection, syntax errors, and host-specific query behavior. | A separately designed expert API is requested. |
| Use separate `symbols` and `body` fields. | This permits distinct weights for name and implementation-body search. | Corpus evaluation shows that the field design is insufficient. |
| Use exact symbol only as a small tie-break. | This avoids evaluation-set overfitting and distortion of general natural-language results. | A change is supported by measurements and documented evidence. |
| Do not make absolute BM25 score an external contract. | It is fragile across tokenizer, config, and SQLite changes. | Score calibration is designed separately. |
| Define no numeric SLA in this phase. | The current goal is correctness and baseline collection. | Product requirements and representative corpora are settled. |
| Build `MATCH` only from normalized, double-quoted Unicode letter/digit tokens joined by `AND`. | This preserves the Phase 02 normalizer while preventing user punctuation, operators, prefixes, columns, or quotes from becoming FTS grammar. | An explicitly designed expert query language is requested. |
| Use the qualified-symbol normalized equality only after BM25 score comparison. | It is a deterministic tie-break, not a multiplicative score boost or corpus-tuned ranking rule. | Evaluation supports a separately designed and documented ranking change. |
| Resolve query byte, token, and token-rune caps through `ServingPolicy` below code-owned absolute ceilings. | They are adjustable local operational limits, but must remain bounded and must not alter index/vector identity. | A request needs a new query-language or resource-limit dimension. |
