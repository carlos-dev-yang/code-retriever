# Phase 06 Free FTS5 Search Evidence

- Phase: `06-fts-search`
- State: `done`
- Original evidence date: 2026-08-15
- Natural-language remediation date: 2026-08-20

The checks below still prove safe MATCH construction, snapshot integrity,
deterministic BM25, and provider-free execution. They do not prove a suitable
natural-language query policy. The 44-question v2 diagnostic later showed that
global all-token `AND` returned zero candidates for every semantic/mixed query.
Current authority:
[`natural-language-fts-query-planner-review-r4.md`](../phase-07/natural-language-fts-query-planner-review-r4.md).

## 2026-08-20 remediation completion

- `LexicalQueryPlannerVersion=2` is code-owned, validated in the immutable
  resolved serving policy, and included in its semantic fingerprint. Changing
  the planner changes serving identity but does not reindex or rematerialize.
- The query planner infers `anchor`, `descriptive`, or `mixed`, deduplicates
  normalized safe tokens, and emits only double-quoted letter/digit terms
  joined by bounded `OR`. Raw text is never executable FTS grammar. No public
  expert MATCH syntax or structured MUST field was added.
- Exact/normalized qualified and short symbols, repository-relative paths,
  and descriptive FTS now produce independent candidate orders inside the same
  pinned SQLite read transaction. Each lane copies authoritative parent
  metadata before the transaction closes.
- Local lanes deduplicate by canonical chunk ID and fuse ordinal ranks using
  the resolved RRF constant. BM25 remains descriptive-lane diagnostics only;
  raw BM25 and vector similarity are never compared.
- Hybrid retrieval consumes the same locally fused lexical order while dense
  remains an independent exhaustive provider. FTS zero candidates therefore
  do not cap dense recall.
- The lexical evaluator records the planner version, inferred shape, safe
  boolean form, selected terms, per-lane candidates and ranks, term coverage,
  candidate-zero state, and local union size. MCP keeps exactly four tools and
  exposes the same bounded diagnostics in `search` responses.

Focused checks actually run from the working tree:

```text
gofmt -w <all directly changed Go paths>
go test -count=1 ./internal/config ./internal/profile ./internal/symbol ./internal/store ./internal/search/lexical ./internal/search ./internal/eval ./internal/devlab ./internal/mcp
go test -count=1 -race ./internal/config ./internal/symbol ./internal/store ./internal/search/lexical ./internal/search ./internal/eval ./internal/devlab ./internal/mcp
go vet ./internal/config ./internal/profile ./internal/symbol ./internal/store ./internal/search/lexical ./internal/search ./internal/eval ./internal/devlab ./internal/mcp
go build -o /tmp/cidx-phase06-review ./cmd/cidx
gofmt -l internal/config internal/profile internal/symbol internal/store internal/search/lexical internal/search internal/eval internal/devlab internal/mcp
git diff --check
```

All commands passed. Existing tests cover safe OR grammar, single lowercase
and code-shaped symbol anchors, a path-only hit, absent-term tolerance,
matched-term coverage, deterministic ordering, FTS corruption, the shared
hybrid RRF input, inline-body rank invariance, evaluator diagnostics, and MCP
serialization. No corpus evaluation, Voyage request, API-key access, paid
operation, new repository, or broad full-project suite ran at this boundary.

The lexical package has no direct provider or lab import. Its store dependency
also owns vector snapshot types for the already-existing hybrid path, so a
transitive package listing includes `internal/vector`; this is not evidence of
a vector scan or provider operation in FTS mode.

Phase 07 subsequently built the committed source with `vcs.modified=false` and
created three preserved run pairs against the unchanged v2 questions. The
first real preflight caught and rejected a missing indexed-source hash in the
lexical artifact path; commit `dcdfd78` fixed that integration defect before
any artifact was published. The next pair exposed missing PascalCase anchor
recognition, and the final index-resolved weak-anchor correction at `2e1a270`
avoided treating ordinary sentence-initial words as anchors. Final planner v2
CompleteRequirementHit@5 is `30/44` versus the historical `10/44`, with zero
candidate-zero queries and no prior-pass regression. Exact evidence is in
[the sequential rerun report](../phase-07/natural-language-lexical-rerun-v2.md).

## Historical 2026-08-15 implementation contract

## Implemented contract

- `internal/symbol/query_tokens.go` reuses the Phase 02 `IdentifierNormalizer` for camelCase, PascalCase, snake_case, qualified-name, path, Unicode, and ordinary-text token classification. It does not create a second normalization implementation.
- `internal/search/lexical` validates UTF-8 and the injected resolved query limits, rejects tokenless input with `EMPTY_QUERY`, and converts only normalized letter/digit tokens into double-quoted FTS phrases joined by internal `AND` grammar. Raw user input is never passed to `MATCH`.
- The lexical service obtains candidate limits and BM25 field weights only from the injected immutable `ResolvedConfig.Search` serving policy. A per-call candidate limit may narrow, but never exceed, that resolved limit.
- `internal/store/search_snapshot.go` opens one independent read transaction, reads `meta.active_generation` plus manifest, and ranks contentless `chunk_fts` with its authoritative chunk/file join before applying the candidate limit. An FTS row lacking a chunk returns `ErrIndexCorrupt`; it is not silently dropped.
- Final rank ordering is BM25 descending, then qualified-symbol exact normalized equality only when BM25 ties, then path, qualified symbol, and chunk ID. The full ordering precedes `LIMIT`, so a boundary tie cannot discard the exact/stable winner. Lexical ranks are one-based and stable for Phase 07 and Phase 11.
- The implementation imports no provider, API-key, network, vector, or lab package. It does not scan the filesystem, acquire `index.lock`, reindex, or expose an MCP/CLI surface.

## Focused core checks actually run

```text
gofmt -w <Phase 06 and directly changed config/profile Go paths>
go test -count=1 ./internal/config ./internal/profile ./internal/symbol ./internal/store ./internal/search/lexical
go vet ./internal/config ./internal/profile ./internal/symbol ./internal/store ./internal/search/lexical
go build ./internal/config ./internal/profile ./internal/symbol ./internal/store ./internal/search/lexical
gofmt -l <Phase 06 and directly changed config/profile Go paths>
rg -n '"cidx/internal/(embedclient|lab|vector)"' <Phase 06 runtime paths>
git diff --check
```

The lexical tests create a temporary live Git worktree, use the Phase 05 indexer to publish Go, TypeScript, and TSX chunks, then verify identifier/body retrieval, generation/manifest metadata, one-based deterministic ranks, safe handling of FTS-like quotes/operators/wildcards, empty/invalid-UTF-8 rejection, negative candidate rejection, and fail-closed detection of an injected orphan contentless FTS row. Store fixtures verify that equal-BM25 candidates are fully ordered by exact/stable keys before `candidate_k=1` applies. Config fixtures verify resolved defaults, explicit values, zero rejection, and the query-byte ceiling. Query-builder fixtures record `GetUserByID -> "get" AND "user" AND "by" AND "id"` and preserve FTS keywords as quoted literal tokens.

## Checks not run

- No race, broad-project, corpus, lexical-evaluation, provider, paid API, embedding, vector, hybrid, MCP, CLI, load, or platform-concurrency benchmark was run by the implementation agent.
- A concurrent publish-versus-complete-lexical-search harness was not added. The implementation uses the same short reader transaction for metadata, FTS candidates, and authoritative chunk rows; Phase 05 already proved SQLite old-reader/new-publish snapshot behavior, but Phase 06-specific concurrent execution remains for commit-boundary review if desired.

## Main-agent commit-boundary validation

```text
go test -count=1 -race ./internal/config ./internal/profile ./internal/symbol ./internal/store ./internal/search/lexical
go vet ./internal/config ./internal/profile ./internal/symbol ./internal/store ./internal/search/lexical
go build ./internal/config ./internal/profile ./internal/symbol ./internal/store ./internal/search/lexical
gofmt -l <Phase 06 and directly changed config/profile Go paths>
rg -n '"cidx/internal/(embedclient|lab|vector)"' <Phase 06 runtime paths>
git diff --check
```

All checks passed. Validation remained scoped to Phase 06 and the directly extended serving-policy config/profile packages; it did not run provider, vector, hybrid, MCP, corpus, or evaluation workflows.

## Handoff and remaining risks

- Phase 07 can call `lexical.Searcher.Search` for normalized diagnostics, manifest/generation identifiers, distinct no-hit versus error outcomes, source-chunk IDs, and deterministic ordinal lexical ranks.
- Phase 11 can consume the same ordinal rank as the FTS lane for RRF without duplicating query construction or FTS SQL.
- Query limits are optional `search.max_query_bytes`, `search.max_query_tokens`, and `search.max_query_token_runes` settings resolved through the central immutable `ServingPolicy`; named defaults remain bounded by code-owned absolute ceilings. They affect the serving-policy fingerprint only and require no reindex, rematerialization, or migration.
- Absolute BM25 scores remain diagnostic/internal and are not a persistence or MCP compatibility promise.
