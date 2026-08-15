# 08. Raw Document Embedding Lab

- Status: `planned` — isolated raw storage remains valid; synchronous request grouping and retry execution require Revision 4 reconciliation
- Prerequisite phases: `02-config-profiles-and-schemas`, `05-worktree-index-pipeline`
- Follow-up phases: `09-vector-materialization`, `12-retrieval-evaluation`
- Design basis: `local-code-search-mcp-v1-design-r4.md` §7.1–§7.4, §9.1

## Context Recovery Checklist

- Reopen the [implementation index](README.md), [execution guide](EXECUTION-GUIDE.md), and [status ledger](STATUS.md) before continuing.
- Confirm the Phase 02 production/lab stores and profile schemas, the central `voyage-code-4` `ModelSpec`, the Phase 05 active canonical document inputs, the repository identity contract, and the shared embedding lock are available.
- Re-check these invariants after any context compaction: source requests explicitly use 1024-dimensional float output; the bank stores immutable document f32 only; document and query roles remain distinct; queries are never persisted; production serving neither opens nor imports the lab store; provider quantized output is not either cidx-owned `binary` or `int8` codec.
- Stop before a paid call if repository/profile identity is unresolved, active canonical inputs cannot be reconstructed exactly, `VOYAGE_API_KEY` is missing for apply, synchronous request limits rely on an undocumented model-specific token cap, or response model/count/index/dimension/finite validation fails.
- Before pausing, record executed evidence in §11, capture new architectural choices in §13, and update [STATUS.md](STATUS.md) with the exact next checklist item and unresolved stop condition.

## 1. Goal

Avoid paying repeatedly for the same document embeddings during initial development and retrieval evaluation by preserving the explicitly requested 1024-dimensional float32 document vectors from Voyage AI `voyage-code-4` in a store physically separate from production.

This phase produces a development- and evaluation-only raw document bank at `.cidx/lab/embeddings.db`. `cidx serve` and ordinary search paths neither open nor depend on this database.

## 2. Scope

### Included

- Collect the distinct canonical document inputs referenced by the current active index snapshot.
- Call the official Voyage AI Embeddings API with explicit `output_dimension=1024`, `output_dtype=float`, `input_type=document`, and `truncation=false`.
- Preserve validated responses as IEEE-754 float32 little-endian blobs.
- Provide plan/apply behavior that does not call the API again for an already preserved raw key.
- Record raw collection runs, successes, terminal failures, retries, and diagnostics.
- Provide the development-only helper command `cidx dev embeddings capture`.

### Out of scope

- An MCP tool or stable general-user CLI.
- Runtime vector search or updates to `index.db.vector_cache`.
- Query-embedding persistence or a query cache.
- Preserving vectors for fixed evaluation queries.
- Operating the raw DB as a permanent runtime feed.
- Permanently retaining or switching among multiple serving profiles as a product feature.
- Injecting externally created vectors, supporting arbitrary providers, or deciding a future user-facing model-pinning policy.
- Automatic raw-vector garbage collection, remote backup, or a shared cache.

## 3. Prerequisites

- Strictly validated `ResolvedConfig` and production/lab store factories from the config phase are implemented.
- The active index snapshot's `embedding_segments` provides profile-independent `canonical_input_sha256` values and enough provenance to reconstruct canonical inputs.
- `canonical_input_sha256` excludes serving dimension, reducer, normalization, metric, and quantization codec.
- The Voyage client reads `VOYAGE_API_KEY` only from the environment and separates the cost-free plan from paid apply.
- `.cidx/` is excluded from source enumeration and Git tracking.
- The central `ModelSpec` registry resolves and validates the sole initial v1 model, `voyage-code-4`, with `SourceDimensions=1024` and allowed targets `{256, 512, 1024}`.
- Indexed source files are capped at 1 MiB. A chunk is the complete semantic AST parent, never a user-configured byte slice; embedding segments are AST-derived and never arbitrarily split.

## 4. Invariants

1. A raw key is `(embedding_source_profile_fingerprint, canonical_input_sha256)`.
2. `EmbeddingSourceProfile` contains `provider=voyage-official`, `model=voyage-code-4`, resolved `SourceDimensions=1024`, `output_dtype=float`, document/query `input_type` mappings, `truncation=false`, and the provider-adapter contract version.
3. Changing serving dimension or switching between the cidx `binary` and `int8` codecs does not change the raw key.
4. Changing provider, model, resolved source dimension, dtype, input-role mapping, truncation policy, or adapter-contract version changes the source fingerprint and raw key.
5. If a valid row already exists for a raw key, a later **development capture** does not call the API again. Public document embedding does not read the lab DB, so this cache-hit contract does not apply there.
6. A raw row is immutable. Never overwrite the same key silently with different vector bytes.
7. Store an API response only after validating response model, `dimensions == 1024`, response-index uniqueness and range, response count, finite values, and blob length.
8. Preserve a successful response in the raw bank even if its active-segment reference disappears after the request. It is a paid development asset.
9. Do not store query text or query vectors in the raw DB, including evaluation queries.
10. The `cidx serve` import graph and runtime initialization exclude the raw-lab package.
11. An absent or corrupt raw DB does not prevent already materialized production search from operating.
12. A Voyage 1024-dimensional source vector permits only `prefix(serving) -> L2` for supported serving dimensions `{256, 512, 1024}`.
13. Document requests use `input_type=document`; live and evaluation query requests use `input_type=query`. Provider-added role prompts are identified by the source-profile contract, not by the canonical-input hash.
14. Never treat Voyage provider `int8`, `uint8`, `binary`, or `ubinary` output as the same encoding as either cidx-owned local codec.

## 5. Packages, Files, and Types to Implement

These paths identify implementation responsibilities. If an earlier phase already created a file with the same responsibility, extend that package rather than duplicating it.

| Package/file | Responsibility |
| --- | --- |
| `internal/profile/embedding_source.go` | `EmbeddingSourceProfile` fingerprint; consumes Phase 02 output |
| `internal/config/model_registry.go` | v1 source and allowed-serving-dimension capabilities by provider/model; consumes Phase 02 output |
| `internal/embedclient/client.go` | provider-independent `EmbeddingClient` interface |
| `internal/embedclient/voyage.go` | official Voyage AI request/response adapter and code-owned endpoint |
| `internal/embedclient/validate.go` | response model, count, index, dimension, and finite-value validation |
| Phase 02 `internal/lab/{schema,store}.go` | lab DB factory extended here with the additive v2 raw-capture provenance migration; production schema is unchanged |
| `internal/lab/inputs.go` | document canonical-input provenance |
| `internal/lab/raw_embeddings.go` | immutable raw-row lookup/insert plus run/failure records |
| `internal/lab/collector.go` | active-input planning, synchronous request grouping, and raw-first persistence |
| `internal/app/dev_capture_embeddings.go` | use case and output model consumed by the Phase 13 development command |

Core types carry these responsibilities:

- `ModelSpec`: provides code-owned `SourceDimensions=1024` and `AllowedServingDimensions={256,512,1024}` for `voyage-code-4`.
- `EmbeddingSourceProfile`: creates the canonical fingerprint from provider, model, resolved source dimensions, dtype, per-role input type, truncation policy, and adapter version.
- `RawEmbeddingKey`: contains the source-profile fingerprint and canonical-input hash.
- `RawEmbeddingRecord`: contains the key, actual dimensions, f32 blob, vector hash, and provenance.
- `CaptureRawPlan`: contains counts for active distinct inputs, raw hits, paid misses, and failures, plus token and cost estimates.
- `CaptureRawResult`: contains persisted, reused, failed, and skipped counts plus the run ID.
- `LabStore`: a read/write port separate from the production-store interface.
- `EmbeddingClient`: accepts a source profile, input role, and ordered inputs and returns ordered 1024-dimensional f32 results.

## 6. Schema, API, and CLI Contract

### Lab storage location

```text
.cidx/
  index.db
  config.json
  embed.lock
  lab/
    embeddings.db
```

The path is repository-local in v1. Do not expose a production-config override for the raw-DB path or promote it to a machine-global cache.

### Logical `.cidx/lab/embeddings.db` schema

`lab_meta`

- `schema_version`
- canonical repository identity and root hash
- creation time and last successful collection time

`lab_inputs`

- `canonical_input_sha256` primary key
- canonical-text-profile fingerprint
- captured generation, manifest, and source-segment provenance
- exact canonical bytes or the reconstruction information selected by the Phase 02 schema as a production-snapshot reference

`raw_document_embeddings`

- `source_profile_fingerprint`
- `canonical_input_sha256`
- `dimensions`
- `vector_f32_le`
- `vector_sha256`
- requested model and response model
- Voyage request ID or client request ID, when available, as provenance
- code-defined raw encoding `f32-le-v1`
- creation time
- primary key `(source_profile_fingerprint, canonical_input_sha256)`
- blob length exactly `dimensions * 4`

`capture_runs`

- run ID, captured active generation, and manifest
- source-profile fingerprint
- planned, hit, requested, persisted, and failed counts
- estimated and actual input tokens, start/end time, and status

`capture_failures`

- run ID and raw key
- `terminal | retryable` classification
- error class, sanitized message, attempts, and last-attempt time

Follow the Phase 02 schema's choice between exact bytes and a snapshot reference in `lab_inputs`. Neither option stores query input. The canonical-input hash and source profile remain authoritative for the raw-vector key.

### Voyage AI request contract

- Define request/response fields and authentication from [Voyage AI Text Embeddings](https://docs.voyageai.com/docs/embeddings) and the [Text Embedding API reference](https://docs.voyageai.com/reference/embeddings-api).
- The executable owns the official endpoint `https://api.voyageai.com/v1/embeddings`; config does not expose a custom `base_url`.
- Authenticate with `Authorization: Bearer $VOYAGE_API_KEY` and `Content-Type: application/json`.
- Use `voyage-code-4`, the default and sole initially validated v1 model, from `EmbeddingSourceProfile.Model`.
- Every raw-document request explicitly sends `input_type="document"`, `output_dimension=1024`, `output_dtype="float"`, and `truncation=false`.
- Omit `encoding_format` so the transport returns a JSON numeric array; keep that transport representation distinct from local `f32-le-v1` storage.
- Always send `output_dimension=1024`; never depend on the provider default.
- For every synchronous request group, validate response count, response model, uniqueness and range of `data[].index`, 1024 dimensions for every embedding, and finite values; restore request order from response indices.
- Fail before an API call if source/serving dimensions conflict with the `ModelSpec` capability. The serving dimension must be one of `{256,512,1024}`.
- Classify a context-limit error under `truncation=false` as a non-retryable input/segment failure.
- Do not use Voyage Batch Inference or asynchronous polling. Group regular synchronous Embeddings endpoint requests at most 128 inputs and 256 KiB total input bytes, with at most four concurrent requests and a 30-second request timeout.
- Retry an initial attempt at most three times after waits of 10, 20, and 30 seconds; a longer provider `Retry-After` wins. Context cancellation stops waiting and retrying.
- Convert API response numbers to float32 and store those exact bytes. Do not store f16.

### Development CLI

```text
cidx dev embeddings capture
cidx dev embeddings capture --apply
cidx dev embeddings capture --apply --retry-failed
```

- The default execution prints a plan and makes no API call.
- Only `--apply` performs paid requests.
- The plan displays raw hits and misses, estimated provider usage, cost or `unknown`, and synchronous request-group count.
- This command is an unstable development surface. Do not add it to MCP tools or general-user installation documentation.

## 7. Config Used and Change Impact

### Production `.cidx/config.json`

- `embedding.model`
- The resolved synchronous request policy: 128 inputs, 256 KiB total input bytes, concurrency 4, 30-second timeout, and the three staged retries (10/20/30 seconds, honoring a longer `Retry-After`). These are operational request/retry settings, not Voyage Batch Inference settings.

The v1 default and sole initially validated `embedding.model` value is `voyage-code-4`. Source output dimension is not user config. The central `ModelSpec` resolves it to 1024 and fixes it in `EmbeddingSourceProfile`. Do not duplicate a configurable lab path or f32 encoding: `.cidx/lab/embeddings.db` and `f32-le-v1` are code-defined contracts. The preservation policy during initial evaluation is no automatic deletion.

### Change impact

| Change | Raw API call required | Notes |
| --- | --- | --- |
| canonical input bytes | only for affected keys | produces a new `canonical_input_sha256` |
| formatter version only, with identical bytes | no | revalidate the hash only |
| provider, model, resolved source dimensions, or request-semantic contract | yes for every active input | produces a new source profile |
| serving dimensions | no | only Phase 09 local materialization |
| reducer, normalization, or metric | no | produces a new vector-space materialization |
| storage codec (`binary | int8`) | no | produces a new storage materialization |
| FTS, RRF, or response-byte settings | no | unrelated to raw vectors |

Validate every user-adjustable codec name once in `ResolvedConfig`. The central code-owned registry/adapter owns the official provider/model capabilities, endpoint, and synchronous request policy; config cannot replace them.

## 8. Ordered Implementation Checklist

1. Fix canonical JSON and fingerprint rules for `EmbeddingSourceProfile`.
2. Resolve `voyage-code-4` source dimension 1024 and its allowed serving-dimension set from the model-capability registry, and fail fast on invalid combinations.
3. Define float32 little-endian encode/decode and vector SHA-256 rules.
4. Open the Phase 02 lab DB schema and migration, and verify its tables and constraints match the raw input/embedding repository contract; do not create a competing schema here.
5. Fail closed when a lab DB has a different repository identity.
6. Implement a read service that gets distinct canonical inputs and exact canonical bytes from the captured active generation.
7. Use bounded raw-key lookup groups to separate hits, misses, and previous failures.
8. Implement the cost-free plan result and JSON output.
9. Implement ordered Voyage synchronous request groups plus response-model/count/index/dimension/finite validation.
10. Durably commit every successful request group to the lab DB immediately.
11. Immutably insert validated raw responses even when their active reference has disappeared.
12. Distinguish terminal from retryable failures and apply bounded retries.
13. Connect cancellation and the process-wide `embed.lock` shared with public document embedding.
14. Stabilize the development capture plan/apply request/result contract for the Phase 13 CLI without registering it as MCP.
15. Verify production `serve` can build without importing the raw package.

## 9. Failure, Rollback, Concurrency, and Security

### Failure and rollback

- Errors before an API call modify neither the raw DB nor the production DB.
- If any validation fails in a grouped response, store none of that request group.
- Do not roll back earlier successful request groups when a later group fails; a subsequent run reuses them as hits.
- A process can die after the API succeeds but before local commit. No exactly-once transaction spans an external API and SQLite.
- Completion documentation must disclose the possible duplicate charge in that response-loss window.
- If the same key already contains different vector bytes, do not overwrite; stop with `RAW_VECTOR_CONFLICT`.

### Concurrency

- Within one repository, allow only one combined development raw collection or public document-embedding apply operation under `embed.lock`.
- Do not hold a production SQLite write transaction during an API call.
- If the index generation changes during collection, preserve raw results for captured inputs as valid content-addressed development assets.
- Collection does not hold a search handler, index read transaction, or production writer gate.

### Security

- Pass the API key only through the environment; never write it to config, run rows, or error messages.
- The `--apply` confirmation must state that canonical document inputs are sent to Voyage AI.
- Treat raw vectors as sensitive source-derived data and restrict lab-directory permissions at least as strongly as production-DB permissions.
- Do not leave canonical source text in error bodies or request logs.
- Never add the lab DB or evaluation artifacts to Git.

## 10. Validation Scenarios

- Run plan twice for the same snapshot and source profile; after the first apply, the second apply has zero API misses.
- Change only serving dimension and codec and retain the same raw-hit count.
- Change the model and plan new raw keys without overwriting existing rows.
- Reject an entire request group with an invalid response dimension, index, count, NaN/Inf value, or blob length.
- Interrupt after some successful request groups, resume, and do not request the persisted groups again.
- Preserve a received raw row even if its active reference disappears while collection is in progress.
- Start `cidx serve` with a missing or inaccessible lab DB and still provide FTS and existing vector search.
- Run evaluation queries without increasing the `raw_document_embeddings` row count.
- Copy a lab DB from another repository and fail repository-identity validation.

Do not create validation code in this planning-document phase. During implementation, preserve evidence by using the existing harness or phase-specific validation commands that cover these scenarios.

## 11. Completion Evidence

Implementation evidence is recorded in [Phase 08 evidence](evidence/phase-08/README.md). The focused fake-backed checks validate cache-first plan/apply, all-or-nothing malformed-response rejection, immutable f32 persistence, root isolation, and v1-to-v2 lab migration preservation. Live provider/cost/corpus evidence is **NOT RUN**; no paid operation was authorized or executed.

- Raw-DB schema dump and migration version.
- Plan/apply log showing that only the first of two identical inputs incurs a paid call.
- `voyage-code-4` 1024-dimensional source response and model/count/index/finite-validation record.
- Report showing raw-row dimensions, blob length, and vector SHA-256 agreement.
- Resume record demonstrating reuse of already persisted request groups after partial failure.
- Confirmation that the production-server dependency graph excludes `internal/lab`.
- Review showing that neither API keys nor canonical source appears in logs or DB diagnostic columns.

Separate validations actually executed from validations not yet run in the completion report.

## 12. Follow-up Handoff

Provide Phase 09 with only:

- the captured manifest and active `canonical_input_sha256` set;
- the `EmbeddingSourceProfile` fingerprint;
- immutable 1024-dimensional f32 raw rows; and
- raw coverage and the list of missing keys.

Phase 09 does not connect the raw DB to search. It uses the bank only to create the single runtime vector representation selected by config.

## 13. Decision Log

- Preserve raw f32 document embeddings only to prevent repeat charges during initial development and retrieval evaluation.
- Keep the raw bank physically separate from the production DB.
- Do not turn the raw bank into a permanent runtime feed or general-user feature.
- Do not store query f32, including fixed evaluation queries, because product questions can continue to change.
- Explicitly request `output_dimension=1024` and `output_dtype=float` from `voyage-code-4` for the v1 source float response.
- Follow the Matryoshka guidance in [Voyage AI Flexible Dimensions and Quantization](https://docs.voyageai.com/docs/flexible-dimensions-and-quantization): select the supported serving prefix, then L2-normalize it.
- Keep provider-supplied int8/binary output separate from both cidx-owned local codecs; do not mix it into v1 materialization.
- The provider may offer a 2048-dimensional option, but v1 excludes it from the source profile, runtime capability registry, and evaluation candidates. Do not request or preserve it without a separate decision.
- The development CLI is an unstable helper, not an MCP or public product contract.
- Lab schema v2 adds source-run/failure provenance and vector SHA-256. The additive v1 migration preserves raw blob bytes, derives their SHA-256, and retains snapshot-reference-only inputs; production schema remains unchanged.
