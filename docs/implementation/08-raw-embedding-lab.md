# 08. Product Document Source Bank and Evaluation Capture

- Status: `done` — the accepted executor now writes immutable 1024-f32 document rows to product source storage while evaluation SQLite remains vector-free
- Prerequisite phases: `02-config-profiles-and-schemas`, `05-worktree-index-pipeline`
- Follow-up phases: `09-vector-materialization`, `12-retrieval-evaluation`
- Design basis: `local-code-search-mcp-v1-design-r4.md` §7.1–§7.4, §9.1

## Context Recovery Checklist

- Reopen the [implementation index](README.md), [execution guide](EXECUTION-GUIDE.md), and [status ledger](STATUS.md) before continuing.
- Confirm the Phase 02 production/lab stores and profile schemas, the central `voyage-code-4` `ModelSpec`, the Phase 05 active canonical document inputs, the portable content-addressed source-key contract, and the shared embedding lock are available.
- Re-check these invariants after any context compaction: source requests explicitly use 1024-dimensional float output; the product bank stores immutable document f32 only; document and query roles remain distinct; queries are never persisted; serving/search/MCP never opens the source bank; evaluation run state is separate; provider quantized output is not a cidx codec.
- Stop before a paid call if repository/profile identity is unresolved, active canonical inputs cannot be reconstructed exactly, `VOYAGE_API_KEY` is missing for apply, synchronous request limits rely on an undocumented model-specific token cap, or response model/count/index/dimension/finite validation fails.
- Before pausing, record executed evidence in §11, capture new architectural choices in §13, and update [STATUS.md](STATUS.md) with the exact next checklist item and unresolved stop condition.

## 2026-08-17 product source-bank supersession

The owner accepted permanent product-owned 1024-f32 document source storage at
`<state_root>/db/embeddings.db`. It is populated before active int8 publication
and locally rematerializes default 1024/int8 or compact 512/int8 without another
provider call. Search and MCP never open it. Evaluation capture/run metadata
must move behind a distinct lab-only store and cannot make the source bank a
serving authority. The earlier development-only raw-bank text below is
historical where it conflicts with
[`SOURCE-VECTOR-BANK-DECISION.md`](SOURCE-VECTOR-BANK-DECISION.md).

## 1. Goal

Avoid paying repeatedly for compatible document embeddings in ordinary use and evaluation by preserving the explicitly requested 1024-dimensional float32 document vectors from Voyage AI `voyage-code-4` in a product source bank physically separate from the serving index and evaluation metadata.

This phase produces the product source bank at `<state_root>/db/embeddings.db` and a separate evaluation-only metadata store. Normal projects resolve state to `<source_root>/.cidx`; cidx's own evaluation workspace uses `.cidx/test/states/<corpus>/`. `cidx serve` and ordinary search paths neither open nor depend on the source bank.

## 2. Scope

### Included

- Collect the distinct canonical document inputs referenced by the current active index snapshot.
- Call the official Voyage AI Embeddings API with explicit `output_dimension=1024`, `output_dtype=float`, `input_type=document`, and `truncation=false`.
- Preserve validated responses as IEEE-754 float32 little-endian blobs.
- Provide plan/apply behavior that does not call the API again for an already preserved source key.
- Record collection runs, successes, terminal failures, retries, and diagnostics in the separate evaluation metadata store.
- Provide shared source-bank capture/reuse services used by public embedding and the development helper command.

### Out of scope

- An MCP tool for source-bank mutation.
- Runtime vector search or updates to `index.db.vector_cache`.
- Query-embedding persistence or a query cache.
- Preserving vectors for fixed evaluation queries.
- Operating the source DB as a runtime search feed or fallback.
- Retaining multiple active serving profiles; only one target is atomically active.
- Injecting externally created vectors, supporting arbitrary providers, or deciding a future user-facing model-pinning policy.
- Automatic raw-vector garbage collection, remote backup, or a shared cache.

## 3. Prerequisites

- Strictly validated `ResolvedConfig` and production/lab store factories from the config phase are implemented.
- The active index snapshot's `embedding_segments` provides profile-independent `canonical_input_sha256` values and enough provenance to reconstruct canonical inputs.
- `canonical_input_sha256` excludes serving dimension, reducer, normalization, metric, and quantization codec.
- The Voyage client reads `VOYAGE_API_KEY` only from the environment and separates the cost-free plan from paid apply.
- `.cidx/` is excluded from source enumeration and Git tracking.
- The central `ModelSpec` registry resolves and validates the sole initial v1 model, `voyage-code-4`, with `SourceDimensions=1024` and allowed targets `{1024, 512}`.
- Indexed source files are capped at 1 MiB. A chunk is the complete semantic AST parent, never a user-configured byte slice; embedding segments are AST-derived and never arbitrarily split.

## 4. Invariants

1. A source key is `(embedding_source_profile_fingerprint, canonical_input_sha256)`.
2. `EmbeddingSourceProfile` contains `provider=voyage-official`, `model=voyage-code-4`, resolved `SourceDimensions=1024`, `output_dtype=float`, document/query `input_type` mappings, `truncation=false`, and the provider-adapter contract version.
3. Changing serving dimension between 1024 and 512 does not change the source key.
4. Changing provider, model, resolved source dimension, dtype, input-role mapping, truncation policy, or adapter-contract version changes the source fingerprint and source key.
5. If a valid row already exists for a source key, both ordinary document embedding and development capture reuse it without another API call.
6. A source row is immutable. Never overwrite the same key silently with different vector bytes.
7. Store an API response only after validating response model, `dimensions == 1024`, response-index uniqueness and range, response count, finite values, and blob length.
8. Preserve a successful response in the source bank even if its active-segment reference disappears after the request. It is reusable product-owned document source data.
9. Do not store query text or query vectors in the source bank, including evaluation queries.
10. The `cidx serve` import graph and runtime initialization exclude both the source-bank writer and the lab package.
11. An absent or corrupt source DB does not prevent already materialized production search from operating.
12. A Voyage 1024-dimensional source vector permits only `prefix(serving) -> L2` for supported serving dimensions `{1024, 512}`.
13. Document requests use `input_type=document`; live and evaluation query requests use `input_type=query`. Provider-added role prompts are identified by the source-profile contract, not by the canonical-input hash.
14. Never treat Voyage provider `int8`, `uint8`, `binary`, or `ubinary` output as the same encoding as the cidx-owned local int8 codec.

## 5. Packages, Files, and Types to Implement

These paths identify implementation responsibilities. If an earlier phase already created a file with the same responsibility, extend that package rather than duplicating it.

| Package/file | Responsibility |
| --- | --- |
| `internal/profile/embedding_source.go` | `EmbeddingSourceProfile` fingerprint; consumes Phase 02 output |
| `internal/config/model_registry.go` | v1 source and allowed-serving-dimension capabilities by provider/model; consumes Phase 02 output |
| `internal/embedclient/client.go` | provider-independent `EmbeddingClient` interface |
| `internal/embedclient/voyage.go` | official Voyage AI request/response adapter and code-owned endpoint |
| `internal/embedclient/validate.go` | response model, count, index, dimension, and finite-value validation |
| `internal/sourcebank/{schema,store}.go` | product source-bank factory, schema, repository/source-profile identity, and strict file permissions |
| `internal/sourcebank/embeddings.go` | immutable source-row lookup/insert, f32 validation, and provenance |
| `internal/lab/{schema,store}.go` | separate evaluation metadata store; no document-vector blobs |
| `internal/lab/capture_runs.go` | development collection run/failure accounting that references source-bank keys |
| `internal/app/document_sources.go` | active-input planning and source-hit/provider-miss orchestration shared by ordinary embedding and development capture |
| `internal/app/dev_capture_embeddings.go` | use case and output model consumed by the Phase 13 development command |

Core types carry these responsibilities:

- `ModelSpec`: provides code-owned `SourceDimensions=1024` and `AllowedServingDimensions={1024,512}` for `voyage-code-4`.
- `EmbeddingSourceProfile`: creates the canonical fingerprint from provider, model, resolved source dimensions, dtype, per-role input type, truncation policy, and adapter version.
- `SourceEmbeddingKey`: contains the source-profile fingerprint and canonical-input hash.
- `SourceEmbeddingRecord`: contains the key, actual dimensions, f32 blob, vector hash, and provider provenance.
- `CaptureSourcePlan`: contains counts for active distinct inputs, source hits, paid misses, and failures, plus token and cost estimates.
- `CaptureSourceResult`: contains persisted, reused, failed, and skipped counts plus the run ID.
- `SourceBank`: an immutable product document-source port separate from both serving and evaluation storage.
- `LabStore`: a run/diagnostic metadata port that stores no document or query vector blob.
- `EmbeddingClient`: accepts a source profile, input role, and ordered inputs and returns ordered 1024-dimensional f32 results.

## 6. Schema, API, and CLI Contract

### Product source and evaluation storage locations

```text
<state_root>/
  config.json
  db/index.db
  db/embeddings.db   # immutable product document 1024-f32 source bank
  embed.lock
  lab/evaluation.db  # development/evaluation metadata only; no vectors
  evaluations/       # immutable development/evaluation artifacts only
```

The state root is project-local in v1. Normal use fixes it to the source project's `.cidx`; the development CLI may inject only a controlling-project-relative `.cidx/test/states/<corpus>` root. Do not expose a production-config path override or promote raw storage to a machine-global cache.

### Logical `<state_root>/db/embeddings.db` schema

`source_meta`

- `schema_version`
- creation time

The portable source identity is not a second repository ID. It is the existing
content-addressed key `(source_profile_fingerprint,
canonical_input_sha256)`. No absolute source/state path or duplicated project
identifier is stored.

`document_source_embeddings`

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

The source bank stores no canonical source text, query text, query vectors,
evaluation labels, run state, or serving-profile selection. Exact canonical
input bytes are reconstructed from the captured production snapshot and
verified against `canonical_input_sha256` before a provider request.

### Logical `<state_root>/lab/evaluation.db` schema owned by this phase

`capture_runs`

- run ID, captured active generation, and manifest
- source-profile fingerprint
- planned, hit, requested, persisted, and failed counts
- estimated and actual input tokens, start/end time, and status

`capture_failures`

- run ID and source key
- `terminal | retryable` classification
- error class, sanitized message, attempts, and last-attempt time

Lab rows reference source keys but do not duplicate source-vector blobs. The
canonical-input hash and source profile remain authoritative for the source
vector key.

### Voyage AI request contract

- Define request/response fields and authentication from [Voyage AI Text Embeddings](https://docs.voyageai.com/docs/embeddings) and the [Text Embedding API reference](https://docs.voyageai.com/reference/embeddings-api).
- The executable owns the official endpoint `https://api.voyageai.com/v1/embeddings`; config does not expose a custom `base_url`.
- Authenticate with `Authorization: Bearer $VOYAGE_API_KEY` and `Content-Type: application/json`.
- Use `voyage-code-4`, the default and sole initially validated v1 model, from `EmbeddingSourceProfile.Model`.
- Every raw-document request explicitly sends `input_type="document"`, `output_dimension=1024`, `output_dtype="float"`, and `truncation=false`.
- Omit `encoding_format` so the transport returns a JSON numeric array; keep that transport representation distinct from local `f32-le-v1` storage.
- Always send `output_dimension=1024`; never depend on the provider default.
- For every synchronous request group, validate response count, response model, uniqueness and range of `data[].index`, 1024 dimensions for every embedding, and finite values; restore request order from response indices.
- Fail before an API call if source/serving dimensions conflict with the `ModelSpec` capability. The serving dimension must be one of `{1024,512}`.
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
- The plan displays source-bank hits and misses, estimated provider usage, cost or `unknown`, and synchronous request-group count.
- This command is an unstable development surface. Do not add it to MCP tools or general-user installation documentation.

## 7. Config Used and Change Impact

### Production `.cidx/config.json`

- `embedding.model`
- The resolved synchronous request policy: 128 inputs, 256 KiB total input bytes, concurrency 4, 30-second timeout, and the three staged retries (10/20/30 seconds, honoring a longer `Retry-After`). These are operational request/retry settings, not Voyage Batch Inference settings.

The v1 default and sole initially validated `embedding.model` value is `voyage-code-4`. Source output dimension is not user config. The central `ModelSpec` resolves it to 1024 and fixes it in `EmbeddingSourceProfile`. Do not duplicate a configurable source-bank path or f32 encoding: `<state_root>/db/embeddings.db` and `f32-le-v1` are code-defined contracts. The preservation policy is no automatic deletion.

### Change impact

| Change | Raw API call required | Notes |
| --- | --- | --- |
| canonical input bytes | only for affected keys | produces a new `canonical_input_sha256` |
| formatter version only, with identical bytes | no | revalidate the hash only |
| provider, model, resolved source dimensions, or request-semantic contract | yes for every active input | produces a new source profile |
| serving dimensions | no | only Phase 09 local materialization |
| reducer, normalization, or metric | no | produces a new vector-space materialization |
| storage codec | not configurable | product storage is fixed to cidx-owned int8 |
| FTS, RRF, or response-byte settings | no | unrelated to raw vectors |

Validate every user-adjustable codec name once in `ResolvedConfig`. The central code-owned registry/adapter owns the official provider/model capabilities, endpoint, and synchronous request policy; config cannot replace them.

## 8. Ordered Implementation Checklist

1. Fix canonical JSON and fingerprint rules for `EmbeddingSourceProfile`.
2. Resolve `voyage-code-4` source dimension 1024 and its allowed serving-dimension set from the model-capability registry, and fail fast on invalid combinations.
3. Define float32 little-endian encode/decode and vector SHA-256 rules.
4. Implement the product source-bank schema independently from the production index and lab metadata stores; migrate preserved compatible document rows without retaining lab tables in the product file.
5. Fail closed when portable source/profile/input/manifest identity is incompatible; never compare a persisted absolute checkout path.
6. Implement a read service that gets distinct canonical inputs and exact canonical bytes from the captured active generation.
7. Use bounded source-key lookup groups to separate hits, misses, and previous lab-recorded failures.
8. Implement the cost-free plan result and JSON output.
9. Implement ordered Voyage synchronous request groups plus response-model/count/index/dimension/finite validation.
10. Durably commit every successful validated response group to the product source bank before any serving-target publication, then record evaluation accounting separately when the development command is the caller.
11. Immutably insert validated source responses even when their active reference has disappeared.
12. Distinguish terminal from retryable failures and apply bounded retries.
13. Connect cancellation and the process-wide `embed.lock` shared with public document embedding.
14. Stabilize the development capture plan/apply request/result contract for the Phase 13 CLI without registering it as MCP.
15. Verify production `serve` can build without importing or opening `internal/sourcebank` or `internal/lab`.

## 9. Failure, Rollback, Concurrency, and Security

### Failure and rollback

- Errors before an API call modify neither the source bank nor the production DB.
- If any validation fails in a grouped response, store none of that request group.
- Do not roll back earlier successful source groups when a later group fails; a subsequent run reuses them as hits.
- A process can die after the API succeeds but before local commit. No exactly-once transaction spans an external API and SQLite.
- Completion documentation must disclose the possible duplicate charge in that response-loss window.
- If the same key already contains different vector bytes, do not overwrite; stop with `RAW_VECTOR_CONFLICT`.

### Concurrency

- Within one repository, allow only one combined development source collection or public document-embedding apply operation under `embed.lock`.
- Do not hold a production SQLite write transaction during an API call.
- If the index generation changes during collection, preserve source results for captured inputs as valid content-addressed product assets.
- Collection does not hold a search handler, index read transaction, or production writer gate.

### Security

- Pass the API key only through the environment; never write it to config, run rows, or error messages.
- The `--apply` confirmation must state that canonical document inputs are sent to Voyage AI.
- Treat source vectors as sensitive source-derived data and restrict source-bank permissions at least as strongly as production-DB permissions. Apply the same restriction to lab metadata.
- Do not leave canonical source text in error bodies or request logs.
- Never add the source bank, lab DB, or evaluation artifacts to Git.

## 10. Validation Scenarios

- Run plan twice for the same snapshot and source profile; after the first apply, the second apply has zero API misses.
- Change only serving dimension between 1024 and 512 and retain the same source-hit count.
- Change the model and plan new source keys without overwriting existing rows.
- Reject an entire request group with an invalid response dimension, index, count, NaN/Inf value, or blob length.
- Interrupt after some successful request groups, resume, and do not request the persisted groups again.
- Preserve a received source row even if its active reference disappears while collection is in progress.
- Start `cidx serve` with missing or inaccessible source/lab DBs and still provide FTS and existing vector search.
- Run evaluation queries without increasing the `document_source_embeddings` row count.
- Copying or moving a source bank remains safe because only rows whose source-profile fingerprint and canonical-input SHA-256 match a requested active key are reusable; no path or separate repository-ID binding is required.
- Inspect the product source bank and prove it contains no evaluation run tables; inspect the lab DB and prove it contains no vector blobs.

Do not create validation code in this planning-document phase. During implementation, preserve evidence by using the existing harness or phase-specific validation commands that cover these scenarios.

## 11. Completion Evidence

The pre-supersession implementation evidence is recorded in [Phase 08 evidence](evidence/phase-08/README.md), including the accepted [Revision 4 reconciliation](evidence/phase-08/revision-4.md). Those focused fake-backed checks validated cache-first plan/apply, byte-bounded concurrent execution, staged retry and cancellation behavior, all-or-nothing malformed-response rejection, immutable f32 persistence, root isolation, and the historical combined-lab migration. They do not prove the new product-source/evaluation-store split. Live provider/cost/corpus evidence was not run at that boundary.

The current product split and offline boundary are accepted in
[`evidence/phase-08/int8-source-bank-reconciliation.md`](evidence/phase-08/int8-source-bank-reconciliation.md).

- Product source-bank and separate evaluation-store schema dumps and migration versions.
- Plan/apply log showing that only the first of two identical inputs incurs a paid call.
- `voyage-code-4` 1024-dimensional source response and model/count/index/finite-validation record.
- Report showing raw-row dimensions, blob length, and vector SHA-256 agreement.
- Resume record demonstrating reuse of already persisted request groups after partial failure.
- Confirmation that the production-server dependency graph excludes both `internal/sourcebank` and `internal/lab`.
- Review showing that neither API keys nor canonical source appears in logs or DB diagnostic columns.

Separate validations actually executed from validations not yet run in the completion report.

## 12. Follow-up Handoff

Provide Phase 09 with only:

- the captured manifest and active `canonical_input_sha256` set;
- the `EmbeddingSourceProfile` fingerprint;
- immutable 1024-dimensional f32 source rows; and
- source coverage and the list of missing keys.

Phase 09 does not connect the source bank to search. It uses the bank only to create the single runtime int8 representation selected by config.

## 13. Decision Log

- Preserve source f32 document embeddings as product-owned reusable input for provider-free 1024/512 int8 rematerialization.
- Keep the source bank physically separate from both the production serving DB and evaluation metadata DB.
- Do not turn the source bank into a runtime search feed, fallback, or second serving authority.
- Do not store query f32, including fixed evaluation queries, because product questions can continue to change.
- Explicitly request `output_dimension=1024` and `output_dtype=float` from `voyage-code-4` for the v1 source float response.
- Follow the Matryoshka guidance in [Voyage AI Flexible Dimensions and Quantization](https://docs.voyageai.com/docs/flexible-dimensions-and-quantization): select the supported serving prefix, then L2-normalize it.
- Keep provider-supplied quantized output separate from the cidx-owned local int8 codec; do not mix it into v1 materialization.
- The provider may offer a 2048-dimensional option, but v1 excludes it from the source profile, runtime capability registry, and evaluation candidates. Do not request or preserve it without a separate decision.
- The development CLI is an unstable helper, not an MCP or public product contract.
- The accepted migration splits compatible immutable document f32 rows into the product source bank and keeps run/failure/evaluation metadata in the lab DB; no vector blob remains in the lab DB.
- Revision 4 centralizes byte-bounded synchronous grouping, retries, cancellation, and response validation in `internal/embed`. The product source handler makes a successful group durable before another completed group is handled; the lab adapter records only development run accounting, while independent failed groups retain earlier source commits for resume.
