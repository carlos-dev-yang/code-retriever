# Phase 00 Config-Field Catalog

This catalog fixes semantic ownership and invalidation class. It does not select experimental numeric values. Only `internal/config` may read raw JSON; consumers receive immutable typed views from `ResolvedConfig`.

## Root and index

| JSON field | Resolved owner | Required validation | Fingerprint/impact |
| --- | --- | --- | --- |
| `version` | `RawConfig` schema decoder | exactly one supported config schema version | schema compatibility; not a semantic profile field |
| `index.languages[]` | `IndexProfile` | nonempty subset of code-owned `go|typescript|tsx`; no duplicates | local reindex |
| `index.max_source_file_bytes` | `IndexProfile` | positive and at or below the code-owned absolute ceiling | local reindex/file eligibility |
| `index.max_chunk_bytes` | `IndexProfile` | positive, internally consistent with source/segment limits | local reindex |
| `index.max_segment_input_bytes` | `IndexProfile` | positive and supported by the canonical formatter/provider policy | local reindex and active segment-key reconciliation |

## Embedding and serving vector

| JSON field | Resolved owner | Required validation | Fingerprint/impact |
| --- | --- | --- | --- |
| `embedding.model` | `EmbeddingSourceProfile` via `ModelSpec` | v1 accepts only `voyage-code-4` | paid document embedding required on change |
| `embedding.target_dimensions` | `VectorSpaceProfile` | member of `{256,512,1024}` and not above source 1024 | local rematerialization from compatible raw, otherwise paid embed |
| `embedding.reducer` | `VectorSpaceProfile` | code-owned supported reducer ID | local rematerialization from compatible raw |
| `embedding.normalizer` | `VectorSpaceProfile` | code-owned supported normalizer ID | local rematerialization from compatible raw |
| `embedding.metric` | `VectorSpaceProfile` | v1 accepts only `cosine` | local rematerialization and scorer compatibility check |
| `embedding.storage_codec` | `VectorStorageProfile` | `binary|int8`; resolves to `binary` when omitted | local rematerialization; changes serving fingerprint only |
| `embedding.batch.max_inputs` | embedding operation policy | positive and no higher than a verified provider/project ceiling | restart/reload only; no semantic fingerprint |
| `embedding.batch.max_input_tokens` | embedding operation policy | positive; must not claim an unverified `voyage-code-4` batch cap | restart/reload only |
| `embedding.batch.max_retries` | embedding operation policy | nonnegative and bounded | restart/reload only |
| `embedding.batch.request_timeout_ms` | embedding operation policy | positive and bounded | restart/reload only |

The raw config does not accept provider, endpoint, source/output dimensions, output dtype, input type, truncation, encoding format, codec implementation version, or API-key fields. Those are code-owned or environment-owned.

## Search and MCP

| JSON field | Resolved owner | Required validation | Fingerprint/impact |
| --- | --- | --- | --- |
| `search.default_mode` | `ServingPolicy` | `fts|hybrid` | restart/reload only |
| `search.allow_paid_query_embedding` | `ServingPolicy` | boolean | restart/reload only; false forces free FTS behavior |
| `search.return_k` | `ServingPolicy` | positive and at or below absolute v1 result maximum | restart/reload only |
| `search.candidate_k` | `ServingPolicy` | integer at least `return_k` and under absolute ceiling | restart/reload only |
| `search.rrf_k` | `ServingPolicy` | positive finite supported range | restart/reload only |
| `search.fts_weights.symbols` | lexical search policy | positive finite number | restart/reload or FTS-policy comparison; no reindex |
| `search.fts_weights.body` | lexical search policy | positive finite number | restart/reload or FTS-policy comparison; no reindex |
| `mcp.hard_max_inline_bytes` | MCP serving safety policy | positive, at or below executable absolute ceiling | restart/reload only; body packaging cannot rerank |
| `mcp.max_read_span_lines` | MCP read safety policy | positive, at or below executable absolute ceiling | restart/reload only |

`search.max_inline_bytes` is a required per-request MCP input, not a config default. Its effective value is clamped by `mcp.hard_max_inline_bytes` without changing result identities, order, or count.

## Intentionally absent or deferred

- No environment-specific repository path, lab path override, endpoint, credential, or host config value.
- No arbitrary algorithm/version number supplied by the user.
- No second query dimension or query codec setting.
- No HNSW/ANN settings.
- No user-scoped multi-repository routing.
- No evaluation candidate override file; evaluation uses current project config sequentially.
- A log-level field is not part of the normative v1 schema yet. Adding it later is a restart-only schema change, not permission to improvise it in a package.
