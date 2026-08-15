# 02. Configuration, Profiles, and Storage Schemas

- Status: `planned` — pre-R4 implementation is historical; Revision 4 config/profile/wire reconciliation awaits Phase 00 evidence
- Prerequisite phases: `00-shared-contracts-and-config`, `01-runtime-storage-spike`
- Downstream phases: `03-go-chunker`, `04-typescript-tsx-chunker`, `05-worktree-index-pipeline`, `08-raw-embedding-lab`
- Design basis: `local-code-search-mcp-v1-design-r4.md` Sections 4.4, 6, and 9
- Evaluation authority: [EVALUATION-CONTRACT.md](EVALUATION-CONTRACT.md)

## Context Recovery Checklist

- Reopen the [implementation index](README.md), [execution guide](EXECUTION-GUIDE.md), [evaluation contract](EVALUATION-CONTRACT.md), and [status board](STATUS.md) before continuing after context compaction.
- Reopen the Phase 00 config/constant catalog and fingerprint contract, plus Phase 01 artifacts for selected SQLite/Tree-sitter bindings, connection pragmas, atomic publication, f32/binary/int8 blob validation, platform constraints, and the source-1024 local-serving-dimension decision.
- Re-check these critical invariants: only `internal/config` reads JSON; one resolved source dimension, one resolved serving dimension, and one resolved storage codec feed every consumer; desired config never partially overrides applied database profiles; production stores only the selected cidx `binary` or `int8` representation and defaults to `binary`; lab and production use different files, schemas, migrations, handles, and dependency paths; document raw f32 is dev-only and query f32 is never stored.
- Stop before schema or API implementation if Phase 01 handoff is incomplete, fingerprint semantics are ambiguous, a config value has competing authorities, a runtime package would import `lab`, or a migration cannot fail atomically before serving.
- Before pausing, update Section 11 evidence, Section 13 decisions, and [STATUS.md](STATUS.md). Preserve `in_progress` status while schema snapshots, profile fixtures, or dependency evidence remain missing.

## 1. Goal

Implement strict typed configuration and two SQLite storage boundaries so the entire project observes the same desired configuration and applied profiles.

- Read `.cidx/config.json` once and inject the validated immutable `ResolvedConfig` into every runtime service.
- Separate the index profile that invalidates source chunks and FTS from the source, vector-space, and storage profile hierarchy.
- Establish one authority for model, serving dimensions, metric, text format, and quantization codec/version.
- Distinguish user-desired config from profiles actually applied to the database.
- Fully separate production `index.db` from the initial development/evaluation f32 lab database at the file, schema, migration, and package-dependency levels.
- Make it possible to determine mechanically whether a dimension or quantization change requires local reindexing, no-cost rematerialization from compatible lab raw, or new paid document embedding.
- Fix language-neutral chunk, range, and projection interfaces first so Go and TypeScript chunkers can be implemented in parallel.

## 2. Scope and Non-Goals

### In scope

- Strict JSON decoding, default resolution, and cross-field validation
- Definitions of `ResolvedConfig`, `IndexProfile`, `CanonicalTextProfile`, `EmbeddingSourceProfile`, `VectorSpaceProfile`, `VectorStorageProfile`, `ServingVectorProfile`, and `ServingPolicy`
- Canonical JSON and SHA-256 profile fingerprints
- Canonical embedding-input hashes and the vector/cache key hierarchy
- Production database schema, migrations, and connection factory
- Separate lab database schema, migrations, and connection factory
- Minimum store API for active serving profile and derived ready/pending/failed state
- Profile-mismatch and required-reconciliation planning
- Shared `Chunker` interface and source byte/line range and projection value types and validators
- Identifier normalization contract and implementation shared by indexing and search
- Portable evaluation query/run/trace/promotion schemas, stable stage/failure enums, and deterministic artifact framing shared by later evaluators

### Out of scope

- File enumeration and AST parsing
- Actual Voyage API calls
- Raw f32 capture and materialization execution
- FTS queries and vector scans
- Public MCP schemas
- Evaluation runners, metric calculations, labels, or numeric promotion margins
- Concurrent search across multiple serving profiles, profile aliases, or automatic profile switching
- Operating the lab database as a permanent production runtime source

The raw lab is an auxiliary facility for initial codec, dimension, and retrieval-quality evaluation and for avoiding repeated charges during construction. Correct v1 runtime behavior, `serve`, `search`, and normal `embed --apply` never depend on its existence.

## 3. Prerequisites

- Phase 01 has selected the SQLite and Tree-sitter bindings, WAL pragmas, and atomic-publication approach.
- Minimum f32, binary, and int8 blob-validation/scoring rules are decided.
- The v1 reference path applies the same reducer to explicit 1024-dimensional float output for documents and queries, producing only local serving dimensions.
- The repository root remains a canonical path and `.cidx/` remains inside it, as required by r4.

## 4. Invariants

1. Only `internal/config` reads JSON config. No other package reads the file or a raw JSON map.
2. Reject unknown fields, invalid enums, out-of-range numbers, and contradictory combinations before opening or changing a database.
3. Only a fully defaulted and validated `ResolvedConfig` reaches application services.
4. Resolved `EmbeddingSourceProfile.SourceDimensions=1024` is the sole authority for API source output. Resolved `VectorSpaceProfile.ServingDimensions` is the sole authority for the production serving space. API request and response validation consume the former; reducers, quantizers, blob validators, and vector scanners receive the appropriate value explicitly and define no separate dimension constant.
5. The quantization codec and version come from the same active `VectorStorageProfile`.
6. Applied profiles in database `meta` are authoritative for currently searchable data. Search never partially applies desired config during a mismatch.
7. Exactly one serving-vector profile is active at a time. A production vector table may retain multiple cache rows, but retrieval and coverage observe only the active fingerprint.
8. Production stores only the active profile's serving `binary` or `int8` representation. It has no f32/f16 column and no codec outside this closed v1 enum.
9. The lab database has a separate file, schema version, migration, and connection type. The dependency graphs of `serve` and `search` contain no `lab` package.
10. The only raw embedding persisted in the lab is 1024-dimensional Voyage source f32 created with `input_type=document`. Search-invisible derived materializations and evaluation provenance may use other lab tables, but query f32 is never stored.
11. Canonical-input identity does not change when serving dimensions or quantization change.
12. Schema-migration versions and profile versions or fingerprints never substitute for one another.

## 5. Packages, Files, and Types to Implement

```text
internal/
  config/
    raw.go                  # JSON-input-only types and strict decoder
    resolved.go             # defaults and ResolvedConfig construction
    validate.go             # range, enum, and cross-field validation
    fingerprint.go          # canonical encoding and domain-separated hashes
    constants.go            # central schema, protocol, hash, and safety constants
    model_registry.go       # voyage-code-4 dimension, dtype, and request capability
    impact.go               # change impact and ConfigImpactPlan
  profile/
    index.go
    canonical_text.go
    embedding_source.go
    vector_space.go
    vector_storage.go
    serving.go
  chunk/
    lang.go                 # Language, ChunkKind, and Chunker interface
    model.go                # SourceChunk, ProjectionRange, and SegmentCandidate
    position.go             # shared byte/line coordinates and range validation
    projection.go           # language-neutral projection invariant checks
  symbol/
    normalize.go            # shared index/query identifier splitting and normalization
  store/
    sqlite.go               # production connection factory
    schema.go               # production schema version and migrations
    meta.go                 # applied profiles, generation, and manifest
    files.go
    chunks.go
    embeddings.go           # segment, codec-tagged vector, and failure metadata
    runs.go
  lab/
    options.go              # fixed lab path and repository identity derived from root
    store.go                # lab-only connection factory
    schema.go               # lab-only schema and migrations
  evalcontract/
    dataset.go              # versioned query, labels, requirements, review, split
    trace.go                # stage observations, denominators, first-loss and failures
    artifact.go             # run manifest and deterministic checksum framing
    promotion.go            # frozen gate contract and result wire types
schemas/
  evaluation/               # portable JSON schemas matching evalcontract types
```

Core types have these responsibilities:

- `RawConfig`: optional user-input representation; never passed into runtime packages
- `ResolvedConfig`: immutable root config after defaults and validation
- `IndexProfile`: chunker, projection, segment, symbol, and FTS rules
- `CanonicalTextProfile`: rules for assembling canonical input bytes
- `EmbeddingSourceProfile`: provider, model, resolved source dimensions, output dtype, document/query input-type mapping, truncation, and adapter version
- `VectorSpaceProfile`: source profile, serving dimensions, metric, reduction algorithm/version, and normalization
- `VectorStorageProfile`: production codec and version
- `ServingVectorProfile`: the one active composition of source, vector-space, and storage profiles
- `ServingPolicy`: reindex-independent policy such as default mode, paid-query permission, `return_k`, `candidate_k`, RRF, FTS query byte/token/token-rune limits, and MCP hard maximum
- `ProfileFingerprint`: distinct value type for a canonical-JSON SHA-256 fingerprint
- `CanonicalInputSHA256`: input identity independent of serving dimensions and quantization
- `ServingVectorKey`: serving-profile fingerprint plus canonical-input hash
- `AppliedProfiles`: index and serving profiles, active generation, manifest digest, and active serving profile actually applied to the database
- `ConfigImpactPlan`: `none | restart_only | local_reindex | local_rematerialize_if_raw | paid_embedding_required | schema_migration`, with reasons
- `ProductionStore`: handle that opens only the production schema
- `LabOptions`: store-open values such as repository identity and fixed lab path derived from root; owns no embedding semantics
- `LabStore`: handle that opens only the lab schema and implements no production interface
- `Chunker`, `ChunkRequest`, `ChunkResult`: context-aware shared language-adapter interface for parallel Phases 03 and 04; request source/path/policy are immutable inputs, results carry parser metadata and typed diagnostics
- `SourceChunk`, `ProjectionRange`, `SegmentCandidate`: shared values connecting parser output to production schema
- `LineIndex`: immutable byte-coordinate index built once per source; derives 1-based inclusive line ranges with CRLF/UTF-8-safe byte offsets
- `IdentifierNormalizer`: deterministic normalization shared by indexed symbol input and search queries
- `EvaluationCase`, `RequiredGroup`, `ExpectedAlternative`: durable reviewed truth independent of generated row IDs
- `StageTrace`, `StageObservation`, `GroupObservation`, `FirstLoss`, `FailureStage`: complete stage and denominator records; group-level first-loss is the sole retrieval-survival authority
- `EvaluationRunManifest`, `ArtifactManifest`: paired-run compatibility and immutable artifact identity
- `PromotionContract`, `PromotionResult`: frozen gates, explicit `core_retrieval|release_candidate` scope, prerequisite digests, and `PROMOTION_EVIDENCE_READY|NOT_PROMOTION_READY`, with no weighted total

`ResolvedConfig` contains no map or `any`. It is never exposed as a mutable singleton after application startup; constructors receive it explicitly.

## 6. Schema, API, and CLI Contracts

### 6.1 Production configuration structure

The final target contract fixes the safety and request-group values below; semantic ownership is fixed as follows.

```jsonc
{
  "version": 1,
  "index": {
    "languages": ["go", "typescript"],
    "max_source_file_bytes": 1048576,
    "target_segment_bytes": 1024
  },
  "embedding": {
    "model": "voyage-code-4",
    "serving_dimensions": "<one of 256, 512, 1024 selected after evaluation>",
    "reducer": "prefix_truncate_l2_v1",
    "normalizer": "l2_v1",
    "metric": "cosine",
    "storage_codec": "binary",
    "request": {
      "max_inputs": 128,
      "max_total_input_bytes": 262144,
      "max_concurrency": 4,
      "timeout_seconds": 30
    },
    "retry": {
      "max_retries": 3,
      "wait_seconds": [10, 20, 30]
    }
  },
  "search": {
    "default_mode": "fts",
    "allow_paid_query_embedding": false,
    "return_k": 5,
    "candidate_k": 20,
    "rrf_k": "<positive integer>",
    "max_query_bytes": "<positive integer below executable safety ceiling>",
    "max_query_tokens": "<positive integer below executable safety ceiling>",
    "max_query_token_runes": "<positive integer below executable safety ceiling>",
    "fts_weights": {
      "symbols": "<positive finite number>",
      "body": "<positive finite number>"
    }
  },
  "mcp": {
    "hard_max_inline_bytes": 65536
  }
}
```

The remaining placeholder marks the serving-dimension choice, not a literal JSON value. Phase 02 resolves the fixed operational defaults from one central default table. It must not silently choose `serving_dimensions`, which initial evaluation must select. `storage_codec` defaults to `binary` and accepts only `binary` or `int8`; search defaults to `fts`. `voyage-code-4` is both the v1 default and the only allowed production value for `embedding.model`. The source-file limit is 1 MiB, the segment target is 1024 bytes, the MCP hard inline default is 64 KiB, and the executable absolute inline ceiling is 1 MiB. Executable-owned chunker, formatter, provider-adapter, and codec implementation versions appear in resolved semantic profiles or code-owned policy, not as user-incrementable config fields. Request and retry values remain validated operational config and do not affect an index/vector fingerprint.

Raw config has no `output_dimensions`. For `voyage-code-4`, v1 `ModelSpec` provides `Provider=voyage-official`, `SourceDimensions=1024`, and `AllowedServingDimensions={256,512,1024}`. Document and query requests explicitly set `output_dimension=1024`, `output_dtype=float`, and `truncation=false`, and omit `encoding_format`. Documents use `input_type=document`; queries use `input_type=query`. The response validator requires exactly 1024 finite floats. V1 excludes the asynchronous Voyage Batch API: it groups synchronous requests to at most 128 inputs and 256 KiB total input bytes, runs at most four groups concurrently, and times out each request after 30 seconds. It makes an initial attempt plus at most three retries after 10, 20, and 30 seconds, using a longer `Retry-After` when supplied.

Documents and queries apply the same `VectorSpaceProfile` prefix reducer and L2 normalizer to 1024-dimensional source float, producing an allowed `serving_dimensions` value. Query f32 is discarded after codec-specific query preparation and scanning. Provider requests remain source-1024 f32; direct 512- or 256-dimensional provider requests are not a v1 path. Provider `output_dtype=int8|binary` is distinct from the cidx `int8` and `binary` codecs applied after local prefix reduction and L2 normalization, so provider-quantized output is neither requested nor used as a storage codec ID.

The official endpoint `https://api.voyageai.com/v1/embeddings` and credential environment variable `VOYAGE_API_KEY` are code-owned constants. Reject provider, endpoint, wire-field overrides, and unsupported models as unknown or unsupported config. Resolve `embedding.request` and `embedding.retry` once and inject them into every provider caller; do not replace their byte boundary with an unverified token cap or add an asynchronous Batch API path.

### 6.2 Canonical fingerprints and keys

- `canonical_input_sha256 = SHA256("cidx/input/v1" || NUL || canonical_input_utf8)`
- `index_profile_fingerprint = SHA256("cidx/index-profile/v1" || NUL || canonical_json(resolved_index_profile))`
- `canonical_text_profile_fingerprint = SHA256("cidx/canonical-text-profile/v1" || NUL || canonical_json(resolved_canonical_text_profile))`
- `embedding_source_profile_fingerprint = SHA256("cidx/embedding-source-profile/v1" || NUL || canonical_json(resolved_source_profile))`
- `vector_space_profile_fingerprint = SHA256("cidx/vector-space-profile/v1" || NUL || canonical_json(resolved_vector_space_profile))`
- `vector_storage_profile_fingerprint = SHA256("cidx/vector-storage-profile/v1" || NUL || canonical_json(resolved_storage_profile))`
- `serving_vector_profile_fingerprint = vector_storage_profile_fingerprint`
- `paid_source_key = (embedding_source_profile_fingerprint, canonical_input_sha256)`
- `serving_vector_key = (serving_vector_profile_fingerprint, canonical_input_sha256)`

Canonical JSON uses UTF-8, fixed key ordering, standard integer representations, no omitted defaults, and no unknown fields. Never delegate fingerprint determinism to map iteration order or incidental JSON-library output.

The canonical JSON for `resolved_source_profile` includes at least `provider=voyage-official`, `model=voyage-code-4`, `source_dimensions=1024`, `output_dtype=float`, `input_type_mapping={document:document,query:query}`, `truncation=false`, and the code-defined `adapter_version`. Endpoint and credential values are transport and secrets rather than source-result semantics, so they are excluded.

Each lower profile's canonical JSON includes the fingerprint of its dependency. For example, the vector-space profile includes its source-profile fingerprint, and the storage profile includes its vector-space-profile fingerprint. `canonical_input_sha256` excludes model, serving dimensions, and quantization so identical canonical bytes can have independently evaluated source results and transformation lineage.

### 6.3 Production `index.db`

- `meta`
  - schema version and canonical root
  - canonical JSON and fingerprints for applied index, canonical-text, embedding-source, vector-space, and vector-storage profiles
  - one `active_serving_profile_fingerprint`
  - active generation and manifest SHA-256
  - index/embed success and attempt times plus observed Git metadata
- `files`
  - relative path, language, indexed SHA-256, and observed mtime/size
- `chunks`
  - file id, kind, symbol, qualified symbol, and signature
  - source byte/line range and exact `source_body`
- `chunk_projections`
  - chunk id, projection kind, and ordered half-open source-relative byte ranges
- `symbols`
  - chunk id and original/normalized name
- `chunk_fts`
  - contentless FTS5 `symbols` and `body`
- `embedding_segments`
  - chunk id, segment number, and projection/display ranges
  - canonical-text-profile fingerprint, canonical-input hash, and serving-vector-profile fingerprint
- `vector_cache`
  - serving-vector key
  - codec-tagged blob, dimensions, codec/version, codec-specific metadata, and materialization provenance
- `embedding_failures`
  - paid-source key, attempts, error class, and last error/time
- `index_runs`, `index_run_files`

Do not add a `ready` enum column. A segment is ready when a `vector_cache` row for its active serving-vector key passes dimension, codec, and blob validation. If no valid vector exists and an applicable failure remains for the canonical input under the active source profile, derive `failed`; if neither exists, derive `pending`.

Store `vector_cache.dimensions` on every row only as integrity metadata that must equal the active profile dimension, not as support for arbitrary multi-profile serving.

### 6.4 Separate lab database

The default is `.cidx/lab/embeddings.db`, separate from production. `serve` startup never creates or opens it.

- `lab_meta`: lab schema version and canonical repository identity
- `lab_inputs`: document canonical-input hash and reproducible canonical bytes or a snapshot reference
- `raw_document_embeddings`: source-profile-plus-input-hash key, immutable 1024-dimensional Voyage document-role f32 blob, dimensions, checksum, API response model, and creation time
- `capture_runs`: target generation, source profile, requested/hit/miss/success/failure counts, and cost metadata
- `materialization_runs`: candidate vector-space/storage profile, raw coverage, output checksum, and evaluation-run link
- `evaluation_runs`: repository, generation, query manifest, candidate profile, and result-artifact location

The lab raw key is `(embedding_source_profile_fingerprint, canonical_input_sha256)`. A capture run references the active `EmbeddingSourceProfile` and retention provenance but creates no new vector-space authority. The lab schema contains neither query input nor query f32.

Only a future explicit offline materialization path may write from the lab into production. That optional path aids initial construction and evaluation; it is not a prerequisite for production embed.

### 6.5 Portable evaluation contracts

The `internal/evalcontract` value types and matching JSON schemas fix the data shape, not retrieval logic or pass thresholds. They encode answer modes, OR alternatives inside AND requirement groups, relevance grades, language/cohort slices, review and calibration/confirmation state, per-stage denominators, stable first-loss/failure enums, paired-run controls, artifact digests, and hard-gate results required by `EVALUATION-CONTRACT.md`.

Every schema is strict and versioned. Unknown fields fail validation. A canonical smoke trace must be serializable with all planned stages, including explicit `NOT_OBSERVED` for stages outside that run contract. Schemas store IDs, hashes, paths, ranges, and provenance; they do not require raw source bodies, provider vectors, query f32, credentials, or machine-specific checkout paths.

### 6.6 Internal APIs and CLI

Phase 02 provides:

- `config.Load(path) (ResolvedConfig, error)`
- `config.FingerprintProfiles(ResolvedConfig) (DesiredProfiles, error)`
- `config.PlanImpact(desired, applied, expectedProductionSchemaVersion) ConfigImpactPlan`; the store caller supplies the database-schema authority so it is never conflated with the config-file version
- `store.OpenProduction(root, resolvedConfig) (ProductionStore, error)`
- `lab.OpenStore(root, labOptions) (LabStore, error)`
- separate production and lab `Migrate` and `InspectSchemaVersion`
- strict `evalcontract` encode/decode/validate and canonical artifact-checksum functions

This phase does not build public CLI or MCP surfaces. Design the application API so future `cidx init --serving-dim`, `status`, `index`, and `serve` all use the same loader and store factory. Do not create `.cidx/lab/config.json`. Derive the lab path and repository identity from root, and always read model, serving dimension, reducer, normalizer, metric, and codec from the current project `ResolvedConfig`. Never inject `LabOptions` or `LabStore` into a production runtime service.

## 7. Config Usage and Change Impact

| Setting class | Example | Impact |
| --- | --- | --- |
| Index profile | chunker/projection/segment/symbol/FTS version | full local reindex |
| Canonical text format | executable-owned `CanonicalTextProfile` | recompute canonical inputs and hashes; only keys whose actual bytes/hash changed lose raw compatibility and require paid document embedding; identical hashes reuse data |
| Model/source space | `embedding.model` plus provider, 1024 source, dtype, input-type mapping, truncation, and adapter version from the model registry | existing raw and vectors are incompatible; paid document embedding required |
| Serving dimensions | `embedding.serving_dimensions` | rematerialize from compatible lab raw; without it, create a new document embedding |
| Reduction/normalization | `embedding.reducer`, `normalizer` | rematerialize from compatible lab raw; only Phase 01-approved combinations are valid |
| Quantization | `embedding.storage_codec` | rematerialize from compatible lab raw; otherwise use normal embed to produce a serving vector |
| Metric | `embedding.metric` | regenerate the serving vector/profile and verify search compatibility |
| Search policy | candidate/return/RRF/default mode | no reindex or re-embedding; restart or reload only |
| MCP hard maximum | inline source-byte cap | no reindex or re-embedding |
| Schema version | production/lab migration | maintenance migration |

The presence of raw data does not guarantee rematerialization. Compatibility requires the same embedding-source profile and canonical input, a `serving_dimensions` member of `ModelSpec.AllowedServingDimensions`, and a transform supported by the binary. Otherwise the result is `paid_embedding_required`.

Database schema version, config schema version, required MCP protocol fields, and supported absolute ceilings are central named constants and validators, not freely editable user config.

## 8. Ordered Implementation Checklist

1. Build `RawConfig` and a strict decoder that rejects unknown fields by default.
2. Define the default table in one file and construct `ResolvedConfig`.
3. Implement per-field enum/range and cross-field validation.
4. Fix a composite profile type structure in which source dimensions, serving dimensions, and quantization each have one authority path.
5. Implement the canonical JSON encoder and distinct fingerprint value types.
6. Separate responsibility for canonical-input hash, source/vector-space/storage profiles, materialization provenance, and serving-vector keys.
7. Implement desired/applied profile comparison and the change-impact planner.
8. Build the production connection factory with Phase 01-selected pragmas.
9. Implement the production schema and versioned migrations.
10. Use a closed codec enum and codec-specific validated value types so the production vector-write API accepts only cidx `binary` or `int8` rows.
11. Implement a separate lab connection factory and lab schema/migrations.
12. Verify package direction prevents a `lab` import anywhere on `serve` or `search` dependency paths.
13. Define queries that derive ready/pending/failed state and coverage from the active snapshot.
14. Define typed errors for schema and profile mismatch.
15. Implement language-neutral chunk/range/projection values and validators so Phases 03 and 04 never edit the same files.
16. Implement the common deterministic normalizer for identifier source text, case, and separators; both index and query paths use this API.
17. At phase completion, preserve human-readable snapshots of resolved config, shared chunk/symbol contracts, and both schemas as evidence.
18. Implement the portable evaluation contract types and strict JSON schemas without metric calculations or numeric margins.
19. Serialize a canonical smoke trace containing every planned stage field, stable failure/first-loss enums, paired controls, and checksum manifest.

## 9. Failure, Rollback, Concurrency, and Security

- If config validation fails, do not create a database file, migrate, or begin profile reconciliation.
- Do not run migrations while the server accepts search. A transaction failure must roll back schema version and data together.
- Fail closed when production and lab paths are confused. Record store kind and canonical-root identity in each database.
- A production database must not open under lab migrations, and a lab database must not open under production migrations.
- Read `VOYAGE_API_KEY` only from the process environment; never store it in `config.json`, the lab database, or evaluation metadata.
- Error messages contain no source body, raw vector, or credential.
- SQLite connections separate WAL reader/writer roles and do not use `BEGIN EXCLUSIVE` on normal paths.
- During a desired/applied mismatch, hybrid is marked for FTS-only fallback and makes no paid API call.
- Stale vector-cache rows do not count as ready or coverage unless their active serving profile and key match.

## 10. Validation Scenarios

Validate the following during implementation; this planning phase adds no test code.

- A config that omits a defaultable field has the same fingerprint as a config that explicitly states the resolved default.
- JSON key order and formatting do not change a fingerprint.
- Unknown fields, negative dimensions, unsupported codecs/providers, and contradictory mode/paid combinations are rejected before database access.
- Document and query API requests and response validators use the same resolved 1024 source dimensions and their correct `input_type` values.
- API requests explicitly use `output_dtype=float` and `truncation=false`, and use neither `encoding_format` nor provider-side quantized output.
- Reducer, quantizer, vector decoder, and scanner use the same resolved serving dimensions.
- Changing serving dimensions or codec leaves the index-profile fingerprint unchanged and changes only serving-profile fingerprints.
- A chunker or FTS-version change is classified as an index-profile mismatch.
- Types or constraints reject any path attempting to write a float32 blob to production.
- Production cannot be opened as a lab store, and the lab cannot be opened as a production store.
- `init` and an FTS-only runtime can be assembled without a lab database.
- A vector row for a profile other than the active profile is excluded from ready state and coverage.
- When a valid vector exists, ready state wins over an old failure; success upsert and failure deletion can occur in one transaction.
- After migration failure, the previous schema still opens or a clear migration-required error is returned.
- Evaluation schemas reject unknown fields, generated row IDs as durable truth, missing required denominators, invalid split/review states, and weighted-total result fields.
- The canonical smoke trace round-trips deterministically and represents unrequested downstream stages as `NOT_OBSERVED` rather than zero.
- Evaluation traces require one ordered observation for every stable stage, complete ordered required-group observations on required evidence stages, and immutable per-group loss after source/parser and provider-union entry. FTS/dense remain lane diagnostics; operational uses its own operation denominator and never carries retrieval groups.
- `OPERATION_FAILURE:<stage>` is a group-level first loss whose stage must match the observation `failure_stage`; JSON Schema enforces the strict wire shape while Go validation enforces cross-record group identity, monotonicity, review identity, and relevance relationships.

## 11. Completion Evidence

Completion evidence is in [Phase 02 evidence](evidence/phase-02/README.md) and was accepted by the main agent at the Phase 02 commit boundary.

- Strict immutable `ResolvedConfig`, profile hierarchy/fingerprints, impact planning, separate schemas, active-codec validation, chunk/projection contracts, normalizer, and portable evaluation types are implemented.
- The canonical-text and embedding-source Phase 00 fixtures reproduce; defaulted and explicit equivalent configs share semantic fingerprints.
- Production/lab schemas are separately versioned at 1 with atomic user-version migration checks, canonical root matching, and owner-only paths where supported. Production contains no raw f32/f16 storage or lab runtime dependency.
- Exact successful and intentionally unrun checks, including RFC-8785 finite-number/Unicode conformance, transaction-pinned active state, immutable lab rows, and real strict JSON-Schema validation, are recorded in the evidence file.

## 12. Downstream Handoff

Provide Phases 03 and 04 with:

- shared `SourceChunk`, `ProjectionRange`, and `SegmentCandidate` storage contracts;
- source byte/line coordinate rules;
- access to the index profile and per-language chunker versions; and
- projection and text-format contracts required for canonical-text hashing.

Provide Phase 05 and embedding phases with:

- production transaction and store APIs;
- the atomic-publication value object and active-profile metadata;
- canonical-input and vector-key functions;
- lab-raw compatibility rules;
- the profile-change impact planner; and
- the common identifier normalizer for FTS symbol input.

Provide Phase 06 with the same identifier normalizer and immutable lexical-search config containing `search.fts_weights`. Phase 06 adds only query classification and safe `MATCH` construction; it does not reimplement normalization.

Provide Phases 07 and 12 with the versioned `evalcontract` types/schemas, stable stage and failure enums, canonical artifact framing, and a valid all-stage smoke trace. Those phases implement labels, runners, metric calculations, and calibrated margins without redefining the wire contract.

## 13. Decision Log

| Decision | Status | Basis |
| --- | --- | --- |
| Runtime config delivery | fixed: inject strict immutable `ResolvedConfig` | Prevent per-package config interpretation and constant drift. |
| Source-dimension authority | fixed: active `EmbeddingSourceProfile.SourceDimensions=1024` | Keep the model registry and document/query API source space aligned. |
| Serving-dimension authority | fixed: active `VectorSpaceProfile.ServingDimensions` | Prevent reducer, codec, and scanner dimension mismatch. |
| Quantization authority | fixed: codec/version in the active `VectorStorageProfile` | Guarantee one production serving format. |
| Active serving-profile count | fixed: one | Keep the auxiliary MCP runtime path simple. |
| Production raw vector | excluded | Production stores only the selected cidx-owned binary or int8 representation. |
| Lab storage | fixed: separate database, schema, and package | Isolate initial evaluation data from runtime state. |
| Lab role | fixed: initial development and evaluation aid | It is not a permanent runtime feed or serving dependency. |
| Query f32 storage | excluded | The repeated-evaluation raw bank applies only to documents. |
| Evaluation wire contract | fixed before runners | Preserve stage denominators, first loss, paired controls, and hard-gate evidence across lexical, dense, hybrid, packaging, and assistant phases. |
| Canonical-input hash | fixed: independent of serving dimension and quantization | Reuse transformations of identical source raw. |
| Allowed reductions | selected: `prefix-l2-v1` with `l2-v1` | Phase 01 fixed the local source-1024 prefix plus L2 contract. Direct serving-dimension provider requests are excluded. |
| Runtime dimensions | fixed: `ModelSpec` is the source/allowed-serving authority | Provider and vector packages receive explicit source/transform specs and keep no competing runtime dimension registry. |
| Formal migrations | fixed: fail-closed atomic `user_version` migration | New databases are created transactionally; current schemas are checked; newer or unknown schemas are refused. |
