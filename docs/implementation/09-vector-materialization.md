# 09. Runtime Vector Materialization

- Status: `done` — int8-only 1024-default/512-optional materialization,
  production schema v5, and current-code retirement boundary accepted
- Prerequisite phases: `01-runtime-storage-spike`, `02-config-profiles-and-schemas`, `05-worktree-index-pipeline`, `08-raw-embedding-lab`
- Follow-up phases: `10-embedding-orchestration-and-reconciliation`, `11-vector-and-hybrid-search`, `12-retrieval-evaluation`
- Design basis: `local-code-search-mcp-v1-design-r4.md` §6, §7.4, §9

## Context Recovery Checklist

- Reopen the [implementation index](README.md), [execution guide](EXECUTION-GUIDE.md), and [status ledger](STATUS.md) before continuing.
- Confirm the Phase 01 atomic SQLite publication decision, Phase 02 profile/config schemas, Phase 05 active manifest and serving keys, Phase 08 product source bank, and current single serving-profile reconciliation are available.
- Re-check these invariants after any context compaction: the product source bank holds 1024-dimensional document f32; serving dimensions are only `{1024,512}`; the transform is `prefix(serving_dimensions) -> L2 -> int8`; `index.db` persists no f32 and reads one active serving profile; candidate preparation is search-invisible and final publication is atomic; no API call occurs in materialization.
- Stop if source coverage is incomplete, source/vector/storage fingerprints do not reconcile with current config, a source checksum/dimension/finite check fails, int8 or metric compatibility is unsupported, or generation/manifest/serving keys change before publish.
- Before pausing, record executed evidence in §11, capture new architectural choices in §13, and update [STATUS.md](STATUS.md) with the exact next checklist item and unresolved stop condition.

## 2026-08-17 product-profile supersession

Production materialization is now `prefix(1024|512) -> L2 -> int8` only, with
1024 used by ordinary fixtures. Binary codec code and every 256 path are removed
from the product; only immutable historical reports and artifacts remain. The
earlier dual-codec body below is historical where it conflicts with
[`RETIRED-VECTOR-PROFILES.md`](RETIRED-VECTOR-PROFILES.md).

## 1. Goal

Deterministically transform the Phase 08 product-owned `voyage-code-4` 1024-dimensional float32 document bank into the configured 1024- or 512-dimensional vector space and the fixed cidx-owned int8 representation, then atomically publish the current serving set to production `index.db.vector_cache` without a provider call.

The reducer, normalizer, and codec implemented here are not lab-only converters. Later ordinary document embedding and hybrid queries use the same packages and `VectorSpaceProfile`.

## 2. Scope

### Included

- A common source-f32 to serving-dimensions to normalization pipeline followed by the fixed int8 codec.
- Hierarchical fingerprints for `EmbeddingSourceProfile`, `VectorSpaceProfile`, and `VectorStorageProfile`.
- Coverage validation between the current active manifest and product source bank.
- Search-invisible candidate preparation outside the production write transaction and atomic publication into the already reconciled single active production profile.
- Development-only `cidx dev embeddings materialize` plan/activate behavior.
- Provenance linking source f32 rows to production int8 representations.

### Out of scope

- Voyage AI API calls.
- Creating or storing query embeddings.
- Rebuilding FTS or AST data, or replacing serving-key reconciliation.
- Persistent multi-profile serving, profile catalogs, or traffic splitting.
- A runtime fallback in which the server reads the source bank.
- External vector import, a model-pinning service, or a remote vector database.
- ANN/HNSW construction.
- Computing evaluation metrics themselves.

## 3. Prerequisites

- The Phase 08 source bank opens and validates against the current repository identity and source profile.
- The config loader provides these layers as immutable resolved values:
  - `EmbeddingSourceProfile`
  - `VectorSpaceProfile`
  - `VectorStorageProfile`
- The active index snapshot exposes distinct `canonical_input_sha256` values and segment references.
- `cidx index` has already reconciled active-segment serving keys to current config.
- The production store reads search-visible `vector_cache`, `active_serving_profile_fingerprint`, and generation from the same snapshot. The active fingerprint must already match current config.
- WAL, bounded-writer policy, and active-generation publication invariants are implemented.

## 4. Invariants

1. `VectorSpaceProfile` contains the source-profile fingerprint, serving dimensions, reducer, normalization, and metric.
2. `VectorStorageProfile` contains the vector-space fingerprint and codec.
3. `VectorSpaceProfile.ServingDimensions` is the single authority for serving dimension. Transform, query, blob validator, scan loop, and coverage use the same value.
4. Before work begins, validate that serving dimensions are one of the active `ModelSpec.AllowedServingDimensions` values `{1024, 512}` and do not exceed source 1024. Do not permit an arbitrary value merely because `1 <= serving <= source`.
5. The v1 reducer selects the first `serving dimensions` values from the source vector using the Voyage Matryoshka prefix method.
6. L2-normalize the prefix vector before int8 quantization.
7. Validate supported metric/normalization combinations during config resolution.
8. `VectorStorageProfile.Codec` is the single authority for the fixed int8 identity. Its versioned contract fixes scale, rounding, clamp, norm, query preparation, and scoring.
9. Production search and coverage read codec-valid rows for exactly one `active_serving_profile_fingerprint` at a time.
10. Inactive candidates and old cache rows may remain physically present, but never participate in the search join and are eligible for separate garbage collection.
11. Search sees either the complete old serving set or the complete new serving set, never mixed dimensions or codecs.
12. Never copy raw f32 bytes into production `index.db`.
13. Rematerializing 1024 or 512 from the same compatible source bank does not call the Voyage AI API.
14. The v1 materialization source is always one 1024-dimensional f32 representation.
15. The serving representation is produced from `SpaceVector` by the cidx-owned int8 codec. Do not treat it as Voyage provider-quantized output.
16. `VectorSpaceProfile.Metric=cosine` describes the serving-f32 reference space. Each codec owns a versioned scorer that may approximate that ordering; raw codec scores are not relabeled as exact cosine.

## 5. Packages, Files, and Types to Implement

| Package/file | Responsibility |
| --- | --- |
| `internal/profile/vector_space.go` | `VectorSpaceProfile` fingerprint; consumes Phase 02 output |
| `internal/profile/vector_storage.go` | `VectorStorageProfile` and `ServingVectorProfile`; consumes Phase 02 output |
| `internal/vector/transform.go` | source-f32 validation plus reducer and normalizer orchestration |
| `internal/vector/reducer_prefix.go` | versioned Voyage Matryoshka prefix reduction |
| `internal/vector/normalize_l2.go` | L2 normalization with zero/finite validation |
| `internal/vector/codec.go` | fixed int8 storage/scoring contract; no codec registry or selector |
| `internal/vector/codec_int8.go` | cidx int8 encoder, metadata validation, query preparation, and scorer |
| `internal/vector/distance.go` | codec-aware scoring contracts; no generic decoder may bypass the active codec |
| `internal/store/vector_build.go` | current-profile atomic publish, abort, and garbage collection |
| `internal/sourcebank/embeddings.go` | immutable validated 1024-f32 document-source reads |
| `internal/app/materialize.go` | shared source-read, transform, plan, and production-publish use case |
| `internal/app/dev_materialize.go` | evaluation-root adapter consumed by the Phase 13 development command; uses the same materializer |

Core types:

- `VectorSpaceProfile`: source fingerprint, dimensions, reducer, normalization, metric, and canonical fingerprint.
- `VectorStorageProfile`: vector-space fingerprint, codec identity, and canonical fingerprint.
- `SourceVector`: ephemeral f32 slice validated for source profile and dimensions.
- `SpaceVector`: ephemeral f32 slice satisfying serving dimension and normalization.
- `StoredVector`: codec-tagged blob, dimensions, codec-specific metadata, and provenance.
- `VectorTransformer`: side-effect-free service that creates a `SpaceVector` from a `SourceVector`.
- `Int8Codec`: creates a `StoredVector` from a `SpaceVector` and provides the sole production search-scoring contract.
- `MaterializationPlan`: captured generation, required source keys, missing source keys, build count, and expected bytes.
- `VectorBuild`: build ID, captured manifest, profile fingerprints, staged row counts, and status.

`VectorTransformer` and `VectorCodec` do not reread config internally. Construct them with already validated immutable profiles.

## 6. Schema, API, and CLI Contract

### Production serving schema

`vector_cache`

- `serving_vector_profile_fingerprint`
- `canonical_input_sha256`
- `space_profile_fingerprint`
- `source_profile_fingerprint`
- dimensions and codec
- int8 blob plus scale/norm metadata using the Phase 01 int8 layout
- source raw-vector SHA-256 provenance
- materialization time
- primary key `(serving_vector_profile_fingerprint, canonical_input_sha256)`

Every searchable row's serving profile must equal `meta.active_serving_profile_fingerprint`. The physical dimension and codec columns validate corruption; they do not override config. Old cache rows with another fingerprint may remain but are excluded from search. Prepare new candidates in lab staging, not in the production table.

Materialization run accounting may be written to the separate lab metadata store
when the caller is a development command, but product materialization neither
requires that store nor writes serving candidates into it. Transformed candidates
remain bounded process state until the atomic production publication boundary.
Production exposes one active search-visible profile.

### Common transformation API

```text
Resolve profiles
  -> Validate source dimensions and finite f32 values
  -> PrefixReduce(serving dimensions)
  -> L2Normalize
  -> Encode with the fixed cidx `int8` codec
  -> Validate stored vector metadata
```

The int8 codec version specifies exact scale, rounding, clamp, zero-vector behavior, reconstructed-norm calculation, query preparation, and scoring. Do not reimplement these rules in multiple packages.

### Atomic publish

1. Read and validate source rows, then prepare every transformation in bounded process memory outside the production write transaction.
2. Immediately before publish, recheck that active generation, manifest, active-segment serving keys, and `active_serving_profile_fingerprint` still match current config. If not, do not publish.
3. In one production write transaction, upsert the complete set of validated candidate rows for current serving keys plus embedding-run metadata. The materializer neither changes config nor switches the active fingerprint to another candidate.
4. Before commit, search sees pre-publish current-profile coverage; after commit, search sees the complete new row set. No partial candidate subset becomes visible.
5. After commit, old cache rows for other profiles may be garbage-collected in a separate maintenance transaction.
6. On rollback, the pre-publish current-profile vector set and coverage remain intact; no candidate becomes search-visible.

If the full-row final transaction is too large under the initial SQLite-spike bounds, introduce generation-scoped production staging. Even then, do not use a pointer as an arbitrary candidate-selection mechanism; first prove that one search cannot combine pre- and post-publish state.

### Development CLI

```text
cidx dev embeddings materialize
cidx dev embeddings materialize --activate
```

- The default execution displays config, source coverage, expected rows/bytes, current and desired fingerprints, and required actions.
- `--activate` calls no API. It builds the one serving set selected by current config and publishes it atomically to production.
- If active-segment serving keys or the active serving fingerprint do not yet match current config, fail before transformation with `PROFILE_RECONCILIATION_REQUIRED` and require `cidx index`.
- If even one source row is missing for an evaluation snapshot, do not publish by default; return `SOURCE_COVERAGE_INCOMPLETE`.
- Do not provide profile-name listing, a promote command, or persistent profile selection. Switch only between explicit 1024 and 512 config values, then plan/apply local rematerialization.
- This command is an unstable development surface, not part of the general MCP surface.

## 7. Config Used and Change Impact

### Flat production config and resolved profile

- `embedding.model`
- `embedding.serving_dimensions`
- `embedding.reducer`
- `embedding.normalizer`
- `embedding.metric`
- `embedding.storage_codec`

`ResolvedConfig` resolves `embedding.model=voyage-code-4` through the central `ModelSpec` to obtain code-owned `SourceDimensions=1024`. It assembles flat fields into `EmbeddingSourceProfile -> VectorSpaceProfile -> VectorStorageProfile`, yielding one `ServingVectorProfile`. Source output dimension is not a config field. No package reads JSON independently or defines separate source/serving dimension constants.

| Config change | Raw collection | Local materialization | FTS/index |
| --- | --- | --- | --- |
| source profile | new source rows required | required after source collection | serving-key local reconciliation required |
| serving dimensions | not required | complete serving set required | serving-key local reconciliation required |
| reducer, normalization, or metric | not required | complete serving set required | serving-key local reconciliation required |
| RRF, candidates, or max inline bytes | not required | not required | not required |

Development materialization uses the one resolved profile from `.cidx/config.json` at execution time. A separate lab config or arbitrary CLI flag cannot override dimensions or codec outside that profile.

## 8. Ordered Implementation Checklist

1. Fix the canonical nested fingerprints of the three profiles.
2. Implement validators for source 1024, model-allowed serving dimensions, and reducer/normalization/metric/codec compatibility.
3. Make the f32 source decoder validate dimensions, byte length, checksum, and NaN/Inf values.
4. Implement the prefix reducer as an independent pure operation.
5. Specify and implement the L2 normalizer's zero-vector policy.
6. Fix the exact arithmetic and blob contract for the int8 codec.
7. Validate that every transformed result has the configured serving dimension.
8. Implement stored-vector blob-length and codec-metadata validation.
9. Implement a one-to-one coverage plan between the active snapshot and source bank.
10. Implement bounded candidate preparation and atomic publication of the production current profile without a lab-store dependency.
11. Decode and transform source rows in bounded local groups outside the production write transaction.
12. Implement build resume or abort policy without touching the previous active serving set.
13. Revalidate generation, manifest, and desired config fingerprints before publish.
14. Implement single-transaction current-profile vector publication and rollback.
15. Stabilize the development materialize plan/activate request and structured-result contract for the Phase 13 CLI.
16. Verify post-publish garbage collection of stale candidates and runtime old rows cannot delete active rows.
17. Preserve dependency direction so ordinary document-embedding and query packages can import the common transformer.

## 9. Failure, Rollback, Concurrency, and Security

### Failure and rollback

- Fail before staging for invalid config, missing source data, checksum mismatch, zero or invalid norm, or incompatible int8 metadata.
- If transformation of one source row fails, do not mark the full build ready.
- Build failure or cancellation does not change the active serving fingerprint or existing vector rows.
- A failed publish transaction leaves every pre-publish current-profile vector row intact.
- Failure to clean staging after commit does not damage serving correctness; retry during later maintenance.
- Rollback never modifies or deletes source-bank rows.

### Concurrency

- Serialize public document embedding and development materialize activation with the per-repository `embed.lock`.
- During a build, search continues to use the committed effective state before publish. If hybrid is unready because profile reconciliation is pending, preserve FTS fallback.
- Publication serializes only the SQLite writer and does not acquire an application-wide search mutex.
- If the index generation changes during a build, reject publish and plan again against the new snapshot.
- A WAL reader sees either state before the publish commit or state after it.

### Security

- Store neither source-f32 nor serving-f32 blobs in the production DB.
- Do not print vector elements or canonical source in transformation errors.
- Only an explicit materialization operation opens both source and production DBs; validate both paths and portable repository identities.
- Keep temporary buffers within the process lifetime and do not dump them automatically into evaluation artifacts.

## 10. Validation Scenarios

- Transform the same source row with the same profile repeatedly and obtain byte-identical int8 blobs and metadata.
- Validate int8 blob length, scale/norm metadata, deterministic encoding, and scorer compatibility for every supported serving dimension.
- Reject serving dimensions outside `{1024,512}` or greater than source 1024 before any DB write.
- Materialize two different serving-dimension configs sequentially and ensure every row and scan dimension matches the current config in each run.
- While building, search sees only the pre-publish effective state; after publish, a new search sees the complete new current-profile row set. No request mixes the two.
- Change index generation immediately before publish and retain the pre-publish serving set without writing.
- Corrupt one source row and prevent activation of a partial serving set.
- Force publish-transaction failure and verify vector rows plus active fingerprint still match the pre-publish state.
- Inspect the serving `index.db` and find no f32 blob; inspect the separate source bank and find only validated document-role 1024-f32 rows.
- Start `cidx serve` without reading the product source bank or lab state.

## 11. Completion Evidence

Current product boundary (2026-08-17):

- [`evidence/phase-09/int8-only-materialization-reconciliation.md`](evidence/phase-09/int8-only-materialization-reconciliation.md)
  records the accepted int8-only implementation, production v5 migration,
  direct int8 scan, removed runtime selections, and one final offline
  test/race/vet/build/static boundary.
- No corpus, credential, provider, or network action occurred.

Historical pre-supersession implementation handoff record (2026-08-15):

- Implemented the then-current offline source-f32 -> prefix -> L2 -> selected binary/int8 path, lab-only candidate staging, and a full-set production publication boundary. Binary and lab-staged product materialization are no longer current proof.
- Added additive lab v2-to-v3 and production v1-to-v2 migrations. Existing raw/capture rows and existing production vector rows are preserved; production legacy rows without source/space/raw-SHA lineage are intentionally not ready until rebuilt.
- The historical materialization planner used a narrow one-snapshot active-key API, verified lab/production root identity, read raw rows in bounded groups, and republished lab-staged variants. The current boundary must reprove those guarantees using the product source bank and no lab dependency.
- Focused implementation checks passed: normal and race tests, vet, and builds for `./internal/config ./internal/vector ./internal/lab ./internal/store ./internal/app`; production dependency inspection found no `internal/lab` import.
- Main-agent commit-boundary validation passed for config integrity, vector transforms/codecs, lab migrations and staging, production migration/publication, the application materializer, and the directly affected index integration.
- Not run: a provider request, API-key read, corpus access, paid work, CLI/MCP wiring, a live server, broad load testing, or retrieval-quality evaluation.

- Profile-fingerprint and dimension/codec validation report.
- Deterministic transformation checksum for identical input.
- int8 score/ranking error summaries against the serving-f32 baseline.
- Int8 blob-length, row-count, and coverage results for both 1024 and 512 serving dimensions.
- Snapshot record showing no pre/post state mixing during concurrent search and publish.
- Existing-vector-set checksum retained after a failed publish.
- Schema-query result proving that only one active serving profile's codec-valid vectors participate in search joins and coverage.
- Confirmation that Voyage client call count is zero during materialization.

## 12. Follow-up Handoff

Provide Phase 10 with:

- a validated `ServingVectorProfile`;
- a side-effect-free `VectorTransformer`;
- a versioned fixed `Int8Codec`;
- a store service that atomically replaces the active serving set; and
- raw-coverage and materialization-result models.

Provide Phase 11 search only with `VectorSpaceProfile`, `VectorStorageProfile`, and the int8 scorer. Do not hand off the product source bank, evaluation store, or materializer to search.

## 13. Decision Log

- Production search reads one profile selected by config: serving dimension, reducer, normalization, metric, and fixed int8 storage codec.
- The source bank can produce 1024/int8 or 512/int8, but only one is materialized and active at a time.
- Do not build permanent multi-profile serving or a promote/catalog workflow.
- The source bank is durable product state used only by explicit document embedding/rematerialization operations, never a runtime search dependency.
- Dimension and codec belong to the central config's fingerprinted profiles; row metadata is validation-only.
- Documents and queries share the reducer and normalizer.
- Queries transform into serving f32 space, then use the active codec's in-memory query preparation and scorer; query representations are not persisted.
- Materialization has one path: fixed source 1024, then local prefix reduction and L2 normalization.
- The cidx-owned int8 codec and Voyage provider-quantized output are separate profile and encoding contracts.
- Production provenance is additive: v2 records source/space profile, raw SHA-256, and materialization time. Legacy rows remain physically preserved but fail the new readiness guard rather than being silently trusted or deleted.
