# 09. Runtime Vector Materialization

- Status: `planned`
- Prerequisite phases: `01-runtime-storage-spike`, `02-config-profiles-and-schemas`, `05-worktree-index-pipeline`, `08-raw-embedding-lab`
- Follow-up phases: `10-embedding-orchestration-and-reconciliation`, `11-vector-and-hybrid-search`, `12-retrieval-evaluation`
- Design basis: `local-code-search-mcp-v1-design-r3.md` §6, §7.4, §9

## Context Recovery Checklist

- Reopen the [implementation index](README.md), [execution guide](EXECUTION-GUIDE.md), and [status ledger](STATUS.md) before continuing.
- Confirm the Phase 01 atomic SQLite publication decision, Phase 02 profile/config schemas, Phase 05 active manifest and serving keys, Phase 08 repository-bound raw bank, and current single serving-profile reconciliation are available.
- Re-check these invariants after any context compaction: the source is always 1024-dimensional document f32; targets are only `{256,512,1024}`; the transform is `prefix(target) -> L2 -> selected cidx codec`; v1 codecs are only `binary` and `int8`, with `binary` as default; production persists no f32 and reads one active serving profile; candidate preparation is search-invisible and final publication is atomic; no API call occurs in materialization.
- Stop if raw coverage is incomplete, source/vector/storage fingerprints do not reconcile with current config, a source checksum/dimension/finite check fails, codec or metric compatibility is unsupported, or generation/manifest/serving keys change before publish.
- Before pausing, record executed evidence in §11, capture new architectural choices in §13, and update [STATUS.md](STATUS.md) with the exact next checklist item and unresolved stop condition.

## 1. Goal

Deterministically transform the Phase 08 `voyage-code-4` 1024-dimensional float32 document bank into the one vector space and cidx-owned `binary` or `int8` storage representation selected by current production config, then atomically publish the current serving set to production `index.db.vector_cache`. The resolved default codec is `binary`.

The reducer, normalizer, and codec implemented here are not lab-only converters. Later ordinary document embedding and hybrid queries use the same packages and `VectorSpaceProfile`.

## 2. Scope

### Included

- A common source-f32 to target-dimensions to normalization pipeline followed by a selected `binary` or `int8` codec.
- Hierarchical fingerprints for `EmbeddingSourceProfile`, `VectorSpaceProfile`, and `VectorStorageProfile`.
- Coverage validation between the current active manifest and raw bank.
- Search-invisible candidate rows prepared in the lab and atomic publication into the already reconciled single active production profile.
- Development-only `cidx dev embeddings materialize` plan/activate behavior.
- Provenance linking raw f32 baselines to production binary/int8 representations.

### Out of scope

- Voyage AI API calls.
- Creating or storing query embeddings.
- Rebuilding FTS or AST data, or replacing serving-key reconciliation.
- Persistent multi-profile serving, profile catalogs, or traffic splitting.
- A runtime fallback in which the server reads the raw DB.
- External vector import, a model-pinning service, or a remote vector database.
- ANN/HNSW construction.
- Computing evaluation metrics themselves.

## 3. Prerequisites

- The Phase 08 raw DB opens and validates against the current repository identity and source profile.
- The config loader provides these layers as immutable resolved values:
  - `EmbeddingSourceProfile`
  - `VectorSpaceProfile`
  - `VectorStorageProfile`
- The active index snapshot exposes distinct `canonical_input_sha256` values and segment references.
- `cidx index` has already reconciled active-segment serving keys to current config.
- The production store reads search-visible `vector_cache`, `active_serving_profile_fingerprint`, and generation from the same snapshot. The active fingerprint must already match current config.
- WAL, bounded-writer policy, and active-generation publication invariants are implemented.

## 4. Invariants

1. `VectorSpaceProfile` contains the source-profile fingerprint, target dimensions, reducer, normalization, and metric.
2. `VectorStorageProfile` contains the vector-space fingerprint and codec.
3. `VectorSpaceProfile.TargetDimensions` is the single authority for target dimension. Transform, query, blob validator, scan loop, and coverage use the same value.
4. Before work begins, validate that target dimensions are one of the active `ModelSpec.AllowedTargetDimensions` values `{256, 512, 1024}` and do not exceed source 1024. Do not permit an arbitrary value merely because `1 <= target <= source`.
5. The v1 reducer selects the first `target dimensions` values from the source vector using the Voyage Matryoshka prefix method.
6. L2-normalize the prefix vector before either binary or int8 quantization.
7. Validate supported metric/normalization combinations during config resolution.
8. `VectorStorageProfile.Codec` is the single authority for codec identity and accepts only `binary` or `int8`. A versioned binary contract fixes bit mapping, packing, padding, query preparation, and scoring; a versioned int8 contract fixes scale, rounding, clamp, norm, query preparation, and scoring.
9. Production search and coverage read codec-valid rows for exactly one `active_serving_profile_fingerprint` at a time.
10. Inactive candidates and old cache rows may remain physically present, but never participate in the search join and are eligible for separate garbage collection.
11. Search sees either the complete old serving set or the complete new serving set, never mixed dimensions or codecs.
12. Never copy raw f32 bytes into production `index.db`.
13. Rematerializing another target dimension or codec from the same raw bank does not call the Voyage AI API.
14. The v1 materialization source is always one 1024-dimensional f32 representation.
15. The serving representation is produced from `SpaceVector` by the selected cidx-owned codec. Do not assume either codec is the same representation as Voyage `output_dtype=int8|binary`.
16. `VectorSpaceProfile.Metric=cosine` describes the target-f32 reference space. Each codec owns a versioned scorer that may approximate that ordering; raw codec scores are not relabeled as exact cosine.

## 5. Packages, Files, and Types to Implement

| Package/file | Responsibility |
| --- | --- |
| `internal/profile/vector_space.go` | `VectorSpaceProfile` fingerprint; consumes Phase 02 output |
| `internal/profile/vector_storage.go` | `VectorStorageProfile` and `ServingVectorProfile`; consumes Phase 02 output |
| `internal/vector/transform.go` | source-f32 validation plus reducer and normalizer orchestration |
| `internal/vector/reducer_prefix.go` | versioned Voyage Matryoshka prefix reduction |
| `internal/vector/normalize_l2.go` | L2 normalization with zero/finite validation |
| `internal/vector/codec.go` | storage-codec interface and registry |
| `internal/vector/codec_binary.go` | cidx binary encoder, bit packing, validation, query preparation, and scorer |
| `internal/vector/codec_int8.go` | cidx int8 encoder, metadata validation, query preparation, and scorer |
| `internal/vector/distance.go` | codec-aware scoring contracts; no generic decoder may bypass the active codec |
| `internal/store/vector_build.go` | current-profile atomic publish, abort, and garbage collection |
| `internal/lab/materializations.go` | active raw rows, derived variants, and run provenance |
| `internal/lab/materializer.go` | feeds raw rows into the common transform and production publish |
| `internal/app/dev_materialize.go` | plan/activate use case consumed by the Phase 13 development command |

Core types:

- `VectorSpaceProfile`: source fingerprint, dimensions, reducer, normalization, metric, and canonical fingerprint.
- `VectorStorageProfile`: vector-space fingerprint, codec identity, and canonical fingerprint.
- `SourceVector`: ephemeral f32 slice validated for source profile and dimensions.
- `SpaceVector`: ephemeral f32 slice satisfying target dimension and normalization.
- `StoredVector`: codec-tagged blob, dimensions, codec-specific metadata, and provenance.
- `VectorTransformer`: side-effect-free service that creates a `SpaceVector` from a `SourceVector`.
- `VectorCodec`: creates a `StoredVector` from a `SpaceVector` and provides the search-scoring contract.
- `MaterializationPlan`: captured generation, required raw keys, missing raw keys, build count, and expected bytes.
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
- codec-tagged blob and codec-specific metadata; binary rows use the Phase 01 bit layout, while int8 rows use the Phase 01 scale/norm layout
- source raw-vector SHA-256 provenance
- materialization time
- primary key `(serving_vector_profile_fingerprint, canonical_input_sha256)`

Every searchable row's serving profile must equal `meta.active_serving_profile_fingerprint`. The physical dimension and codec columns validate corruption; they do not override config. Old cache rows with another fingerprint may remain but are excluded from search. Prepare new candidates in lab staging, not in the production table.

Lab `materialization_runs`

- build ID, captured generation, and manifest
- source, space, and storage profile fingerprints
- planned, staged, missing, and rejected counts
- `planned | building | ready | published | aborted | failed`
- start/end time and sanitized error

Lab `materialized_variants`

- materialization fingerprint and `canonical_input_sha256`
- transformed binary or int8 bytes and validation metadata
- raw-vector SHA-256 provenance
- available only to evaluation/materialization; production search cannot access it

Multiple candidate variants in the lab do not imply persistent multi-profile serving. Production exposes one active search-visible profile.

### Common transformation API

```text
Resolve profiles
  -> Validate source dimensions and finite f32 values
  -> PrefixReduce(target dimensions)
  -> L2Normalize
  -> Encode(selected `binary` or `int8` codec)
  -> Validate stored vector metadata
```

The binary codec version specifies exact value-to-bit mapping, bit order, padding, zero/invalid-vector behavior, query preparation, and similarity scoring. The int8 codec version specifies exact scale, rounding, clamp, zero-vector behavior, reconstructed-norm calculation, query preparation, and scoring. Do not reimplement these rules in multiple packages.

### Atomic publish

1. Prepare every transformation in lab `materialized_variants`, outside the production write transaction.
2. Immediately before publish, recheck that active generation, manifest, active-segment serving keys, and `active_serving_profile_fingerprint` still match current config. If not, do not publish.
3. In one production write transaction, upsert the complete set of validated candidate rows for current serving keys plus embedding-run metadata. The materializer neither changes config nor switches the active fingerprint to another candidate.
4. Before commit, search sees pre-publish current-profile coverage; after commit, search sees the complete new row set. No partial candidate batch becomes visible.
5. After commit, old cache rows for other profiles may be garbage-collected in a separate maintenance transaction.
6. On rollback, the pre-publish current-profile vector set and coverage remain intact, while lab candidates remain search-invisible.

If the full-row final transaction is too large under the initial SQLite-spike bounds, introduce generation-scoped production staging. Even then, do not use a pointer as an arbitrary candidate-selection mechanism; first prove that one search cannot combine pre- and post-publish state.

### Development CLI

```text
cidx dev embeddings materialize
cidx dev embeddings materialize --activate
```

- The default execution displays config, raw coverage, expected rows/bytes, current and desired fingerprints, and required actions.
- `--activate` calls no API. It builds the one serving set selected by current config and publishes it atomically to production.
- If active-segment serving keys or the active serving fingerprint do not yet match current config, fail before transformation with `PROFILE_RECONCILIATION_REQUIRED` and require `cidx index`.
- If even one raw row is missing for an evaluation snapshot, do not publish by default; return `RAW_COVERAGE_INCOMPLETE`.
- Do not provide profile-name listing, a promote command, or persistent profile selection. Compare candidates sequentially by changing config explicitly one choice at a time.
- This command is an unstable development surface, not part of the general MCP surface.

## 7. Config Used and Change Impact

### Flat production config and resolved profile

- `embedding.model`
- `embedding.target_dimensions`
- `embedding.reducer`
- `embedding.normalizer`
- `embedding.metric`
- `embedding.storage_codec`

`ResolvedConfig` resolves `embedding.model=voyage-code-4` through the central `ModelSpec` to obtain code-owned `SourceDimensions=1024`. It assembles flat fields into `EmbeddingSourceProfile -> VectorSpaceProfile -> VectorStorageProfile`, yielding one `ServingVectorProfile`. Source output dimension is not a config field. No package reads JSON independently or defines separate source/target dimension constants.

| Config change | Raw collection | Local materialization | FTS/index |
| --- | --- | --- | --- |
| source profile | new raw rows required | required after raw collection | serving-key local reconciliation required |
| target dimensions | not required | complete serving set required | serving-key local reconciliation required |
| reducer, normalization, or metric | not required | complete serving set required | serving-key local reconciliation required |
| codec | not required | complete serving set required | serving-key local reconciliation required |
| RRF, candidates, or max inline bytes | not required | not required | not required |

Development materialization uses the one resolved profile from `.cidx/config.json` at execution time. A separate lab config or arbitrary CLI flag cannot override dimensions or codec outside that profile.

## 8. Ordered Implementation Checklist

1. Fix the canonical nested fingerprints of the three profiles.
2. Implement validators for source 1024, model-allowed target dimensions, and reducer/normalization/metric/codec compatibility.
3. Make the f32 source decoder validate dimensions, byte length, checksum, and NaN/Inf values.
4. Implement the prefix reducer as an independent pure operation.
5. Specify and implement the L2 normalizer's zero-vector policy.
6. Fix the exact arithmetic and blob contracts for the `VectorCodec` interface, the default binary codec, and the alternative int8 codec.
7. Validate that every transformed result has the configured target dimension.
8. Implement stored-vector blob-length and codec-metadata validation.
9. Implement a one-to-one coverage plan between the active snapshot and raw bank.
10. Implement lab materialization runs/variants and atomic publication of the production current profile.
11. Batch raw decode and transformation outside the production write transaction.
12. Implement build resume or abort policy without touching the previous active serving set.
13. Revalidate generation, manifest, and desired config fingerprints before publish.
14. Implement single-transaction current-profile vector publication and rollback.
15. Stabilize the development materialize plan/activate request and structured-result contract for the Phase 13 CLI.
16. Verify post-publish garbage collection of stale candidates and runtime old rows cannot delete active rows.
17. Preserve dependency direction so ordinary document-embedding and query packages can import the common transformer.

## 9. Failure, Rollback, Concurrency, and Security

### Failure and rollback

- Fail before staging for invalid config, missing raw data, checksum mismatch, zero or invalid norm, or unsupported codec.
- If transformation of one raw row fails, do not mark the evaluation full build ready.
- Build failure or cancellation does not change the active serving fingerprint or existing vector rows.
- A failed publish transaction leaves every pre-publish current-profile vector row intact.
- Failure to clean staging after commit does not damage serving correctness; retry during later maintenance.
- Rollback never modifies or deletes raw-DB rows.

### Concurrency

- Serialize public document embedding and development materialize activation with the per-repository `embed.lock`.
- During a build, search continues to use the committed effective state before publish. If hybrid is unready because profile reconciliation is pending, preserve FTS fallback.
- Publication serializes only the SQLite writer and does not acquire an application-wide search mutex.
- If the index generation changes during a build, reject publish and plan again against the new snapshot.
- A WAL reader sees either state before the publish commit or state after it.

### Security

- Store neither source-f32 nor target-f32 blobs in the production DB.
- Do not print vector elements or canonical source in transformation errors.
- The development materializer is the only process that opens both raw and production DBs; validate both paths and repository identities.
- Keep temporary buffers within the process lifetime and do not dump them automatically into evaluation artifacts.

## 10. Validation Scenarios

- Transform the same raw row with the same profile repeatedly and obtain byte-identical binary or int8 blobs and metadata.
- Validate binary blob length as `ceil(target_dimensions/8)`, padding rules, deterministic bit packing, and scorer compatibility for every supported target dimension.
- Validate int8 blob length, scale/norm metadata, deterministic encoding, and scorer compatibility for every supported target dimension.
- Reject target dimensions outside `{256,512,1024}` or greater than source 1024 before any DB write.
- Materialize two different target-dimension configs sequentially and ensure every row and scan dimension matches the current config in each run.
- Change the codec and retain raw hits while changing only storage fingerprint and serving bytes.
- While building, search sees only the pre-publish effective state; after publish, a new search sees the complete new current-profile row set. No request mixes the two.
- Change index generation immediately before publish and retain the pre-publish serving set without writing.
- Corrupt one raw row and prevent activation of a partial serving set.
- Force publish-transaction failure and verify vector rows plus active fingerprint still match the pre-publish state.
- Inspect the production DB and find no f32 blob.
- Start `cidx serve` without reading lab materialized variants or the raw DB.

## 11. Completion Evidence

- Profile-fingerprint and dimension/codec validation report.
- Deterministic transformation checksum for identical input.
- binary and int8 score/ranking error summaries against the target-f32 baseline.
- Blob-length, row-count, and coverage results for at least two target-dimension candidates.
- Snapshot record showing no pre/post state mixing during concurrent search and publish.
- Existing-vector-set checksum retained after a failed publish.
- Schema-query result proving that only one active serving profile's codec-valid vectors participate in search joins and coverage.
- Confirmation that Voyage client call count is zero during materialization.

## 12. Follow-up Handoff

Provide Phase 10 with:

- a validated `ServingVectorProfile`;
- a side-effect-free `VectorTransformer`;
- a versioned `VectorCodec`;
- a store service that atomically replaces the active serving set; and
- raw-coverage and materialization-result models.

Provide Phase 11 search only with `VectorSpaceProfile`, `VectorStorageProfile`, and the codec-aware scorer. Do not hand off the raw lab store or development materializer.

## 13. Decision Log

- Production search reads one profile selected by config: target dimension, reducer, normalization, metric, and one storage codec. V1 defaults to `binary` and permits `int8` as the only alternative.
- The raw bank can produce multiple candidates, but they are materialized and compared sequentially, one at a time.
- Do not build permanent multi-profile serving or a promote/catalog workflow.
- The raw DB is an initial-evaluation helper, not a runtime dependency.
- Dimension and codec belong to the central config's fingerprinted profiles; row metadata is validation-only.
- Documents and queries share the reducer and normalizer.
- Queries transform into target f32 space, then use the active codec's in-memory query preparation and scorer; query representations are not persisted.
- Materialization has one path: fixed source 1024, then local prefix reduction and L2 normalization.
- Both cidx-owned codecs and Voyage provider-quantized output are separate profile and encoding contracts.
