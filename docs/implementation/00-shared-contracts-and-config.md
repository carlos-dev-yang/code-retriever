# 00. Shared Contracts and Configuration

- Status: done
- Prerequisite phases: none
- Downstream phases: all phases 01 through 14
- Baseline design: [r3](../../local-code-search-mcp-v1-design-r3.md)

## Context Recovery Checklist

- Reopen the [implementation index](README.md), [execution guide](EXECUTION-GUIDE.md), [evaluation contract](EVALUATION-CONTRACT.md), and [status board](STATUS.md) before continuing after context compaction.
- Re-read the r3 cost, snapshot, retrieval, and response contracts. Phase 00 has no prerequisite implementation artifact, but its config catalog, constant catalog, and profile/hash fixtures become prerequisites for every later phase.
- Re-check these critical invariants: FTS paths never call Voyage; the only v1 provider/model pair is `voyage-official`/`voyage-code-4`; document and query source requests explicitly produce 1024-dimensional float vectors with their distinct input roles; production stores only one active cidx-owned `binary` or `int8` representation and defaults to `binary`; only dev lab capture may persist document-role source f32; runtime never imports or opens the lab.
- Stop if the provider/model contract, source or allowed target dimensions, canonical framing, profile ownership, or production/lab boundary is unresolved or conflicts with r3. Do not let a downstream phase invent a local interpretation.
- Before pausing, update Section 11 evidence, Section 13 decisions, and [STATUS.md](STATUS.md). Keep the phase `planned` while any required evidence remains a placeholder.

## 1. Goal and Resulting Capabilities

Make every downstream implementation use the same product boundary, profile/hash definitions, configuration authority, and production/lab separation. After this phase, package-level design must not need to reinterpret dimension, codec, generation, or cost boundaries.

Completion must make it possible to:

- determine which typed profile owns each config field;
- determine whether a config change requires `restart`, `local reindex`, `local rematerialize`, or `paid embed`;
- design keys and lineage for production vectors and initial-evaluation raw vectors without conflating them;
- review, at the static dependency level, whether a runtime package depends on the lab database; and
- verify that phase-level completion evidence does not violate shared invariants.

## 2. Scope and Non-Goals

### 2.1 In scope

- Product, cost, snapshot, freshness, and MCP-surface invariants
- The boundary between config values and code-owned constants
- Immutable `ResolvedConfig` and typed profile concepts
- Canonical-input, source, vector-space, and storage fingerprints and keys
- Roles of the production database and the initial-evaluation lab database
- Config-change impact and reconciliation classification
- Shared error handling and fail-closed principles
- Versioned evaluation query, stage-trace, failure, artifact, and promotion-contract ownership

### 2.2 Out of scope

- Actual Go packages and migrations: Phase 02
- Exact SQLite and Tree-sitter binding selection: Phase 01
- Initial target dimension selection: after Phase 12 evaluation
- Exact `binary` and `int8` codec algorithms and versioned blob/scorer contracts: Phase 01; v1 config exposes only these two IDs and defaults to `binary`
- Preselected universal hit-rate, latency, or noninferiority thresholds; calibration freezes project-specific margins before confirmation
- A future fixed-model policy for general users or an external-vector injection policy
- A product for long-term raw-embedding retention or synchronization

## 3. Prerequisites and Inputs

- The r3 cost, snapshot, retrieval, and response contracts
- The official Voyage MRL dimension-reduction contract for `voyage-code-4`
- v1 language scope: Go, TypeScript, and TSX
- Stable v1 MCP tools: `status`, `search`, `read_span`, and `reindex`

Voyage documentation describes MRL, or Matryoshka Representation Learning, output dimensions. The v1 reference path requests source f32 with `output_dimension=1024`, `output_dtype=float`, and `truncation=false`, selects an allowed leading target prefix locally, and L2-normalizes it. It does not assume that the API result for `output_dimension=512` or `output_dimension=256` is identical to the local transform; Phase 01 may compare them when needed.

## 4. Fixed Invariants

### 4.1 Product and cost

1. `cidx` is a local auxiliary search tool. It does not replace an IDE, LSP, `rg`, or a file reader.
2. `index`, MCP `reindex`, and FTS search never call the Voyage API.
3. Normal document embedding and hybrid query embedding call the API only after explicit authorization.
4. Initial raw capture is a paid operation only under a separate `cidx dev ... --apply` command.
5. Search never refreshes a stale index or implicitly runs paid document embedding.

### 4.2 Data and retrieval

1. Indexing reads live working-tree bytes at execution time.
2. A search read transaction observes exactly one committed active generation.
3. Production search observes exactly one active serving-vector profile.
4. `ready` is derived only when a valid vector exists for the active serving key.
5. FTS-only search remains functional when vectors are absent or only partially available.
6. Indexed source bodies and live-file freshness are distinct concepts.
7. `max_inline_bytes` limits only the amount of raw source body returned. It must not change ranking or the identity, order, or count of up-to-`k` results.

### 4.3 Runtime and lab

1. Production `.cidx/index.db` contains only the active profile's cidx-owned quantized vectors. V1 permits `binary` and `int8`, defaults to `binary`, and never mixes them in one active profile.
2. The only raw embedding that initial-evaluation `.cidx/lab/embeddings.db` may persist is a 1024-dimensional Voyage f32 vector created with `input_type=document`. Search-invisible derived variants and run provenance may use separate lab tables.
3. Query f32 is never persisted, for either fixed evaluation queries or live queries.
4. Production `serve`, `search`, `status`, `index`, and normal `embed` never open or attach the lab database.
5. The lab is neither a continuous runtime vector source nor a fallback.
6. After initial evaluation, new documents may follow the normal embed path, be converted into the current serving profile, and discard raw f32.
7. A missing or incomplete lab database must not break production functionality or correctness.

## 5. Packages, Files, and Types to Implement

This phase creates no code, but fixes the shared type names and ownership that Phase 02 must provide.

```text
internal/config/
  raw.go                  RawConfig and file input shape
  resolved.go             immutable ResolvedConfig
  validate.go             strict field and cross-field validation
  fingerprint.go          canonical JSON and domain-separated hashes
  constants.go            central protocol, schema, and safety constants
  model_registry.go       Voyage model and source/target dimension capabilities
  impact.go               config-change impact planning

internal/profile/
  index.go                IndexProfile
  canonical_text.go       CanonicalTextProfile
  embedding_source.go     EmbeddingSourceProfile
  vector_space.go         VectorSpaceProfile
  vector_storage.go       VectorStorageProfile
  serving.go              ServingVectorProfile assembly
```

The minimum typed values are:

- `ResolvedConfig`
- `ResolvedLimits`
- `IndexProfile`
- `CanonicalTextProfile`
- `EmbeddingSourceProfile`
- `VectorSpaceProfile`
- `VectorStorageProfile`
- `ServingVectorProfile`
- `ProfileFingerprints`
- `ConfigImpactPlan`
- `ModelSpec`
- `ReducerID`, `NormalizerID`, `MetricID`, and `StorageCodecID`
- `EvaluationCase`, `RequiredGroup`, `ExpectedAlternative`, `EvaluationRunManifest`, `StageTrace`, `FirstLoss`, `PromotionContract`, and `PromotionResult`

Phase 02 may consolidate package names, but it must preserve this ownership:

- Config reads files, resolves values, and validates them.
- Profiles produce semantic fingerprints.
- The vector transformer consumes resolved profiles and never reads config files.
- Dimension and codec values in database rows are validation copies, not configuration authority.
- Evaluation contract types and stable enums are defined before runners; Phase 07/12 consume them and may not create incompatible local copies.

## 6. Configuration, Profile, and Hash Contracts

### 6.1 Config shape

The following schema illustrates field shapes. It is not a config that fixes a specific target dimension or codec.

```jsonc
{
  "version": 1,
  "index": {
    "languages": ["go", "typescript"],
    "max_source_file_bytes": "<positive integer>",
    "max_chunk_bytes": "<positive integer>",
    "max_segment_input_bytes": "<positive integer>"
  },
  "embedding": {
    "model": "voyage-code-4",
    "target_dimensions": "<one of 256, 512, 1024 selected after evaluation>",
    "reducer": "<supported reducer id>",
    "normalizer": "<supported normalizer id>",
    "metric": "cosine",
    "storage_codec": "binary",
    "batch": {
      "max_inputs": "<positive integer>",
      "max_input_tokens": "<positive integer>",
      "max_retries": "<non-negative integer>",
      "request_timeout_ms": "<positive integer>"
    }
  },
  "search": {
    "default_mode": "fts",
    "allow_paid_query_embedding": false,
    "return_k": "<positive integer>",
    "candidate_k": "<integer >= return_k>",
    "rrf_k": "<positive integer>",
    "fts_weights": {
      "symbols": "<positive finite number>",
      "body": "<positive finite number>"
    }
  },
  "mcp": {
    "hard_max_inline_bytes": "<positive integer below executable safety ceiling>",
    "max_read_span_lines": "<positive integer>"
  }
}
```

Real JSON must not contain the string placeholders above. `init` must receive a target dimension or let the initial-evaluation workflow write it. `storage_codec` resolves to `binary` when omitted and accepts only `binary` or `int8`. Hybrid serving must not start before a target dimension is selected.

### 6.2 Resolution and validation

```text
bytes
-> strict JSON decode
-> verify supported schema version
-> apply code defaults
-> resolve provider, model, source dimension, and allowed targets from the model registry
-> cross-field validation
-> assemble and fingerprint semantic profiles
-> immutable ResolvedConfig
```

Required validation:

- Reject unknown fields.
- Raw config does not accept `provider`, `endpoint`, `output_dimensions`, `output_dimension`, `output_dtype`, `input_type`, or `truncation`.
- The only v1 direct provider and production model are `voyage-official` and `voyage-code-4`.
- The endpoint `https://api.voyageai.com/v1/embeddings` and credential name `VOYAGE_API_KEY` are code-owned constants and are absent from config and fingerprints.
- `ModelSpec` owns `SourceDimensions=1024` and `AllowedTargetDimensions={256,512,1024}` in one place.
- Document and query source requests both explicitly use `output_dimension=1024`.
- Requests explicitly use `output_dtype=float` and `truncation=false`, omit `encoding_format`, and apply the registry's document/query input-type mapping.
- `target_dimensions` must be a member of `ModelSpec.AllowedTargetDimensions`.
- The only v1 metric is cosine.
- The reducer, normalizer, and codec combination must be implemented. V1 accepts only `storage_codec=binary|int8`, with `binary` as the resolved default.
- Validate every byte, token, batch, and retry value. Do not guess a `voyage-code-4` batch-token cap and write it into a default or ceiling before the official capability is verified.
- Require `candidate_k >= return_k`.
- A config hard maximum must not exceed the executable's code-owned absolute ceiling.
- Reject provider or endpoint override fields that v1 does not expose as unknown fields.
- Config contains no credentials or API keys.

### 6.3 Immutable injection

Build `ResolvedConfig` once at process start or an explicit reload. Give each module only the small typed view it needs through its constructor.

| Consumer | Injected values |
| --- | --- |
| File enumerator | languages and source-size/ignore limits |
| Chunker | per-language chunk, projection, and segment limits |
| Canonical formatter | `CanonicalTextProfile` |
| Voyage client validator | `EmbeddingSourceProfile` and batch/retry limits |
| Document/query transformer | the same `VectorSpaceProfile` |
| codec encoder/scanner | the same `VectorStorageProfile`; `binary` or `int8` selects a matched encoder, validator, query preparation, and scorer |
| Store validator | all applied fingerprints and blob constraints |
| Search | search policy, active serving profile, and inline limits |
| MCP adapter | request limits and schema constants |

Mutable global config, package-local dimension literals, and codecs read arbitrarily from database rows are forbidden as sources of calculation authority.

### 6.4 Hashes and keys

Every hash uses canonical encoding or explicit framing to avoid ambiguous lengths, and uses a separate domain.

```text
canonical_input_sha256 =
  SHA256("cidx/input/v1" || NUL || canonical_input_utf8)

canonical_text_profile_fingerprint =
  SHA256("cidx/canonical-text-profile/v1" || NUL || canonical_json(resolved profile))

embedding_source_profile_fingerprint =
  SHA256("cidx/embedding-source-profile/v1" || NUL || canonical_json({
    provider: "voyage-official",
    model: "voyage-code-4",
    source_dimensions: 1024,
    output_dtype: "float",
    input_type_mapping: {
      document: "document",
      query: "query"
    },
    truncation: false,
    adapter_version
  }))

vector_space_profile_fingerprint =
  SHA256("cidx/vector-space-profile/v1" || NUL || canonical_json({
    source_profile_fingerprint,
    target_dimensions,
    reducer_id,
    normalizer_id,
    metric
  }))

vector_storage_profile_fingerprint =
  SHA256("cidx/vector-storage-profile/v1" || NUL || canonical_json({
    vector_space_profile_fingerprint,
    storage_codec_id
  }))
```

The endpoint and `VOYAGE_API_KEY` are transport and secret data, not source-vector semantics, so they are excluded from the source fingerprint. The input-type mapping and adapter version can change vector semantics for the same canonical input and therefore must be included.

In v1, `serving_vector_profile_fingerprint` is an explicit alias of `vector_storage_profile_fingerprint`; it is not hashed again.

Keys:

- Paid raw/source result: `(embedding_source_profile_fingerprint, canonical_input_sha256)`
- Production serving vector: `(serving_vector_profile_fingerprint, canonical_input_sha256)`
- Paid attempt/failure: `(embedding_source_profile_fingerprint, canonical_input_sha256)`
- Active segment reference: `(serving_vector_profile_fingerprint, canonical_input_sha256)`

If a canonical formatter version changes but the produced bytes remain identical, `canonical_input_sha256` remains identical and the paid source result may be reused. The applied canonical-text fingerprint still differs, so Phase 05 reconciliation must rebuild the bytes and hash from stored chunks and projections and verify them.

### 6.5 Vector transform order

The v1 reference contract uses this order:

```text
Voyage source f32 (`output_dimension=1024`, `output_dtype=float`)
-> validate finite values and length
-> select the first target_dimensions components
-> L2 normalize
-> encode with the selected cidx codec (`binary` by default, or `int8`)
-> validate row and blob
```

Documents and queries use `input_type=document` and `input_type=query`, respectively, while sharing the same `VectorSpaceProfile` and transformer function. Query vectors are never persisted; after target-space transformation they remain only in the codec-specific in-memory representation required by the scanner. An API `output_dimension=target` optimization is forbidden until Phase 01 evidence justifies a separate contract change. Voyage provider-side `output_dtype=int8|binary` is distinct from the cidx codecs applied after local L2 normalization. Provider-quantized output is neither requested nor treated as a production codec ID.

The `binary` codec is a complete encoder/validator/query-preparation/scorer contract, not merely a smaller blob label. Phase 01 must fix its bit-value rule, bit order, padding validation, similarity calculation, and versioned codec ID together. The `int8` codec likewise fixes scale, rounding, clamp, norm, and scorer behavior. A production row is valid only under the exact active codec contract.

`metric=cosine` names the target float vector space. A quantized codec may only approximate float cosine ordering. It must not label a Hamming, asymmetric, or reconstructed score as exact cosine; its profile ID and evaluation report must identify the codec-specific scorer used.

## 7. Change Impact and Reconciliation

### 7.1 Decision order

1. Validate config bytes into a new `ResolvedConfig`.
2. Compare it with the database's applied fingerprints.
3. Distinguish schema migration from semantic profile changes.
4. Produce the smallest correct `ConfigImpactPlan`.
5. `status` reports the plan without writing.
6. Only `index/reindex` publishes local snapshot and active-segment-key reconciliation.
7. Paid embed and hybrid search must not silently mix profiles before reconciliation.

### 7.2 Impact classes

| Class | Examples | Action |
| --- | --- | --- |
| `restart_only` | return/candidate k, RRF, inline cap | keep DB vectors and FTS |
| `local_reindex` | chunker, projection, FTS tokenizer | publish a new local generation |
| `local_rematerialize_if_raw` | target dimension, reducer, normalizer, codec | use the dev materializer when compatible lab raw exists; runtime never reads the lab automatically |
| `paid_embedding_required` | model, canonical bytes, lab miss | explicit normal embed or dev capture |
| `schema_migration` | DB schema version | offline or startup migration |

After a target-dimension change, `index` only reconnects active segments to the new serving key; it does not convert data automatically from the lab database. During initial evaluation, the user must run the dev materializer or explicitly run normal embed. Before the dev materializer may publish to production, its candidate profile must equal the current project config and active segment key. It never edits config automatically. FTS remains available throughout.

## 8. Implementation Checklist

- [ ] Map r3 and this document's terms to a draft Go type naming scheme.
- [ ] Assign ownership of the config-field catalog and code-constant catalog to packages.
- [ ] Record canonical JSON rules and hash framing as a decision.
- [ ] Define the model registry's source and update policy.
- [ ] Define semantic fields and fixture formats for every profile fingerprint.
- [ ] Map every config change to the impact classification table.
- [ ] Add the production/lab package import prohibition to linting or structural review.
- [ ] Document runtime/lab database boundaries and deletion effects.
- [ ] Prevent an unresolved target dimension or codec from becoming a silent code default.
- [ ] Cross-check that Phases 01 through 14 use these terms and keys.
- [ ] Freeze versioned evaluation query/run/trace/promotion schemas, stage/failure enums, and artifact checksum framing from `EVALUATION-CONTRACT.md` without choosing numeric margins.

## 9. Failure, Rollback, Concurrency, and Security

- Config parse, validation, or fingerprint failures terminate before any database open, migration, or write.
- Never silently fall back from an unsupported profile or ignore an unknown field.
- On a config/stored-profile mismatch, hybrid search does not call the API and falls back to FTS.
- Read the API key only from `VOYAGE_API_KEY`; never store it in config, databases, manifests, or logs.
- Paid-command and host documentation must disclose that document code and query text are sent to Voyage.
- Treat the lab database as a project-local sensitive artifact and exclude it from Git.
- Deleting the lab database must not affect production. Conversely, do not promise to restore a deleted index database from the lab.
- A search running during config reload uses exactly the immutable config and database snapshot pinned when it started.

## 10. Validation Scenarios

This phase creates no test code. It fixes the scenarios Phase 02 must validate.

1. Changing the target dimension in one place changes the resolved value used by the document transformer, query transformer, blob validator, and scanner together.
2. Changing the codec does not change the canonical-input or source-profile hash, but does change storage and serving fingerprints.
3. Changing the model cascades through source, vector-space, and storage fingerprints.
4. Identical canonical input bytes reuse the source raw key even if the formatter implementation version differs.
5. A target outside the allowed set, an unknown codec, provider or endpoint override fields, and an API key in config are rejected before startup.
6. The production package dependency graph contains no lab package.
7. FTS and existing production-vector search work without a lab database.
8. Reusing one evaluation run's query embedding in memory for that config's f32, selected-codec, and hybrid comparison creates no query row in either database. A different candidate profile runs in a separate evaluation run.
9. A vector row whose dimension or codec differs from the active serving profile is not considered `ready`.
10. A search-policy change creates neither a reindex nor pending embedding work.

## 11. Completion Evidence

Phase entry record, 2026-08-15:

- Prerequisite evidence: none required for Phase 00.
- Authorities read: repository instructions, r3 design, implementation index, execution guide, evaluation contract, status ledger, and this phase document.
- Workspace state: Git initialized on `main`; no implementation code, Go module, remote, license, or commit exists.
- Repository bootstrap: `.env` and `.cidx/` are ignored; text/binary attributes, editor defaults, and a root navigation README are present.
- Intended completion evidence: the seven catalog/review entries below, with cross-phase terminology checks.
- Checks not run: no code, tests, builds, provider calls, migrations, or platform checks exist at this phase entry.

Before changing the phase to `done`, every item below must be complete:

- Final config-field catalog: [draft complete](evidence/phase-00/config-field-catalog.md)
- Final code-constant catalog: [draft complete; later-phase values explicitly deferred](evidence/phase-00/code-constant-catalog.md)
- Canonical profile/hash fixture document: [RFC 8785 and five reproduced digests complete](evidence/phase-00/profile-hash-fixtures.md)
- Config-change impact matrix review: [draft complete](evidence/phase-00/config-impact-matrix.md)
- Production/lab dependency-boundary review: [draft complete](evidence/phase-00/runtime-lab-boundary-review.md)
- Evaluation query/run/trace/promotion schema and stable-enum review: [draft complete](evidence/phase-00/evaluation-schema-catalog.md)
- Cross-phase terminology review: [complete](evidence/phase-00/cross-phase-review.md)
- Checks actually run and results: Git initialization/status/ignore checks, RFC 8785 digest recomputation, and documentation structure/link/terminology checks are recorded in the [evidence index](evidence/phase-00/README.md) and [cross-phase review](evidence/phase-00/cross-phase-review.md).
- Checks not run: no implementation code, tests, builds, provider calls, SQLite migrations, Tree-sitter bindings, codec algorithms, or platform checks exist yet.
- Remaining decisions assigned to later phases: initial target dimension (Phase 12), exact binary/int8 algorithm contracts and SQLite/Tree-sitter binding selection (Phase 01).

The status remains `planned` while any evidence entry is a placeholder.

## 12. Downstream Handoff

Provide Phase 01 with:

- default/source/target dimension terms and registry values;
- transform order and the conditional API direct-dimension comparison requirement;
- the requirement that production and lab SQLite databases are separate files; and
- the requirement to decide the exact codec ID and blob layout in the spike.

Provide Phase 02 with:

- config-field, type, fingerprint, and impact contracts;
- production and lab schema keys; and
- strict validation and immutable-injection requirements; and
- ownership of portable evaluation contract types/schemas and deterministic artifact framing.

Provide Phases 03 through 14 with:

- only the invariants and typed profiles in this document; and
- a prohibition on phase-local literals or independent profile interpretation.

## 13. Decision and Change Log

| Date | Decision | Rationale |
| --- | --- | --- |
| 2026-08-14 | Request explicit 1024-dimensional float output for Voyage documents and queries. | Keep one v1 source space and compare 512- and 256-dimensional candidates from the same raw vectors. |
| 2026-08-14 | Persist 1024-dimensional source f32 only in the initial document-evaluation lab. | Reduce repeated API cost without turning runtime into a permanent multi-profile system. |
| 2026-08-14 | Never persist raw query vectors. | Queries may change and a persistent query cache is not a product goal. |
| 2026-08-14 | Runtime config activates one target dimension and codec. | Every retrieval component must observe the same representation. |
| 2026-08-14 | Defer future model distribution and external-vector policy. | Those decisions exceed the initial local MCP quality-validation scope. |
| 2026-08-14 | Default production storage to cidx-owned `binary` and retain `int8` as the only alternative v1 codec. | Keep one compact active serving representation while preserving an explicit comparison and fallback option. |
| 2026-08-14 | Define stage-separated evaluation schemas before retrieval runners. | Preserve denominators, first-loss attribution, paired controls, and hard-gate evidence across phases without a weighted total. |
| 2026-08-15 | Initialize the repository without inventing module, license, remote, or provider credentials. | Git and local-ignore hygiene are prerequisites; unresolved external identifiers remain explicit decisions. |
| 2026-08-15 | Use RFC 8785 JCS for semantic-profile canonical JSON. | Give profile fingerprints a published cross-language byte contract rather than implementation-dependent map serialization. |
