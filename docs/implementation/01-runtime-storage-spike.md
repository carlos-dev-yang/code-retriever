# 01. Runtime and Storage Technology Spike

- Status: `done`
- Prerequisite phase: `00-shared-contracts-and-config`
- Downstream phase: `02-config-profiles-and-schemas`
- Design basis: `local-code-search-mcp-v1-design-r3.md` Sections 4.3, 7, 9.2, and 13

## Context Recovery Checklist

- Reopen the [implementation index](README.md), [execution guide](EXECUTION-GUIDE.md), [evaluation contract](EVALUATION-CONTRACT.md), and [status board](STATUS.md) before continuing after context compaction.
- Re-read the Phase 00 config/constant catalog, typed profile and hash contract, and production/lab boundary. Reopen the current candidate-platform list, spike manifest, and any SQLite, Tree-sitter, codec, or Voyage comparison artifacts already produced.
- Re-check these critical invariants: production stores no f16/f32; the lab and production databases have separate files, schemas, migrations, and connection factories; parsing and API calls occur outside write transactions; one search snapshot never mixes generations; Voyage source requests explicitly return 1024-dimensional float vectors with document/query roles and `truncation=false`.
- Stop if FTS5 cannot be guaranteed in the distributed binary, grammars require runtime downloads, atomic publish cannot preserve old-or-new snapshots, a candidate stores f32 in production, or an unverified direct-target path would replace the 1024-source reference path.
- Before pausing, update Section 11 evidence, Section 13 decisions, and [STATUS.md](STATUS.md). Record both executed and unexecuted checks; do not reopen the completed phase without a documented contract change.

## 1. Goal

Validate and fix the runtime and storage technologies on which the rest of v1 depends through small executable spikes. The result must be evidence that the following contracts work in intended deployment environments, not merely proof that a library can run locally.

- A distributed binary can include SQLite with FTS5.
- Go, TypeScript, and TSX Tree-sitter grammars can be included without runtime downloads.
- A search reader observes one complete state from either before or after index publication.
- Only short write transactions serialize; scanning, parsing, and external API waits occur outside them.
- Storage formats for float32 originals and both production codecs (`binary`, `int8`) are explicit and reproducible.
- When needed, compare whether 512- and 256-dimensional prefix-plus-L2 vectors derived from `voyage-code-4` 1024-dimensional float output can occupy the same retrieval space as Voyage direct-target output.

This phase makes no numeric commitment for hit rate, latency, or supported segment count. Measurements are observational inputs for technology choices and later defaults.

## 2. Scope and Non-Goals

### In scope

- Compare candidate Go SQLite bindings and how each includes FTS5.
- Validate WAL, separate reader/writer connections, and bounded `busy_timeout`.
- Decide between in-place active-table updates and generation-scoped staging for atomic publication.
- Validate insert, delete, update, `MATCH`, and rollback behavior for contentless FTS5.
- Validate Tree-sitter bindings and embedded grammars for Go, TypeScript, and TSX.
- Collect evidence for the CGO policy and supported OS/architecture set.
- Define encode, decode, validation, and scoring rules for little-endian float32 blobs and the cidx-owned `binary` and `int8` codecs.
- If needed, compare local reduction of Voyage 1024-dimensional source output with the API's direct 512- and 256-dimensional output.

### Out of scope

- Final database schema and migrations
- Actual Go and TypeScript chunking rules
- Retrieval ranking, RRF, and MCP transport
- Production Voyage batching and retry implementation
- Final raw-embedding lab schema and operational commands
- ANN indexes or numeric performance acceptance thresholds

Spike code must not become an alternate route around production packages. Move only the selected minimum implementation into later packages. Remove or clearly isolate rejected candidates and temporary entry points before completing the phase.

## 3. Prerequisites

- The search-generation visibility invariant in r3 Section 4.3 remains unchanged.
- The boundary that production stores only the selected cidx-owned `binary` or `int8` representation and raw float32 lives only in a separate lab database is approved. `binary` is the resolved v1 default.
- Paid checks against the official Voyage API run only after explicit user approval.
- Candidate platforms are managed as an explicit deployment-target list, not inferred solely from the local development environment.

## 4. Invariants

1. A search snapshot's manifest, FTS statistics and candidates, chunks, segments, and vector coverage all belong to the same committed logical generation.
2. After index preparation failure or publication rollback, the complete previous generation remains searchable.
3. No SQLite write transaction remains open during parsing, hashing, vector transformation, or external API calls.
4. Production `index.db` stores neither float16 nor float32 vectors.
5. Lab and production databases use separate files, schemas, migrations, and connection factories.
6. A vector blob is valid only after profile, dimensions, codec version, byte length, and finite-value checks pass.
7. Dimension-reduction and quantization fingerprints can include algorithm version and parameters, not just algorithm names.
8. A 512- or 256-dimensional vector reduced from Voyage `output_dimension=1024` is not assumed to occupy the same space as the corresponding API direct-target vector until compatibility is demonstrated.
9. Query float32 is never persisted in spike output or the lab database.

## 5. Packages, Files, and Types to Implement

Before selecting exact bindings, filenames describe roles and do not expose candidate library names through package APIs.

```text
internal/
  store/
    sqlite.go             candidate connection creation and shared pragmas
    publish.go            candidate short atomic-publication boundary
  chunk/
    parser.go             candidate minimal interface hiding language grammars
  embedclient/
    response.go           candidate official-API float32 response validation
  vector/
    transform.go          candidate shared reduction and normalization math
    codec_binary.go       candidate cidx binary encoder/validator/scorer
    codec_int8.go         candidate cidx int8 encoder/validator/scorer
  lab/
    f32codec.go           lab-only float32 blob codec candidate
```

The spike first fixes these minimum types:

- `SQLiteCapabilities`: FTS5, WAL, compile options, driver version, and platform information
- `ReadStore` and `WriteStore`: separate reader and writer connection lifetimes
- `PublishPlan`: value object carrying changes prepared outside the database into a short transaction
- `Parser`: grammar-independent interface shaped like `Parse(language, sourceBytes)`
- `F32Vector`: lab value that passed dimension and finite-value validation
- `StoredVector`: codec-tagged blob, dimensions, codec-specific metadata, and codec version
- `EmbeddingSourceSpec`: provider, model, source dimensions, dtype, document/query input-type mapping, truncation, and adapter version
- `VectorTransformSpec`: source/target dimensions and reduction, normalization, and quantization rules

Package boundaries must prevent an `F32Vector` from reaching production write APIs in `store` or `search`.

## 6. Schema, API, and CLI Contracts

### Temporary SQLite validation schema

The spike schema is not final, but it must reproduce these relationships:

- one meta row identifying the current generation;
- correspondence between source rows and contentless FTS rows;
- generation publication and FTS changes committed or rolled back in one transaction; and
- a reader seeing only the old or new snapshot as of transaction start.

Implement and compare both publication candidates:

1. In-place active-table update: apply every delta and switch meta in the final transaction.
2. Generation-scoped staging: prepare new-generation rows and then switch a pointer.

r3 prefers option 1. Option 2 remains valid if evidence shows that applying a large delta makes the write transaction unacceptably long or breaks FTS atomicity. Record the selected rationale and the rejected option's failure evidence in the decision log.

### Codec contract

- A lab f32 blob uses IEEE-754 float32, explicit byte order, and an exact `4 * dimensions` byte length.
- A production `binary` blob has exactly `ceil(dimensions/8)` bytes, validated zero padding when dimensions are not byte-aligned, and the metadata required by its fixed scorer contract.
- A production `int8` blob has exactly `dimensions` bytes plus its scale/norm metadata.
- Zero vectors, NaN, Inf, invalid lengths, and unknown codec versions fail closed.
- Phase 01 compares quantization scale and normalization approaches, but must not assign final names or versions without results.

### Voyage MRL dimension compatibility

The only v1 direct provider is `voyage-official`, and the only production model is `voyage-code-4`. Spike requests use only the code-owned endpoint `https://api.voyageai.com/v1/embeddings` and `VOYAGE_API_KEY`. `ModelSpec` records only `SourceDimensions=1024` and `AllowedTargetDimensions={256,512,1024}`. Source requests explicitly ask for 1024 rather than relying on the provider default, and omit `encoding_format`.

When a comparison is needed, use the same fixed set of document inputs for these paths:

```text
A: voyage-code-4 source float32
   -> API output_dimension=1024
   -> output_dtype=float
   -> truncation=false
   -> input_type=document for documents, input_type=query for queries
   -> select leading target dimensions
   -> L2 renormalize

B: request the same model, input_type, truncation, and input
   -> API output_dimension=512 or 256
   -> output_dtype=float
   -> validate API float32 output
   -> apply the same normalization when required
```

Record per-vector cosine, component differences, and changes in neighbor order within the same corpus. This document does not predeclare a numeric acceptance threshold.

The v1 reference path for both production documents and queries explicitly requests `ModelSpec.SourceDimensions=1024` as float32, then applies the same prefix reducer and L2 normalization. Documents use `input_type=document`; queries use `input_type=query`. This role mapping is part of the retrieval-compatible source-profile contract. Query float32 is discarded immediately after transformation.

[Voyage Flexible Dimensions and Quantization](https://docs.voyageai.com/docs/flexible-dimensions-and-quantization) describes selecting the leading dimensions of MRL vectors and normalizing them. If that official basis and observed results support compatibility, record direct API target dimensions as a candidate equivalent optimization. The spike never changes the reference path automatically. Without demonstrated compatibility, direct-target requests remain forbidden and API target variants in the raw lab are not treated as the same vector space.

Do not treat Voyage `output_dtype=int8|binary` as equivalent to either cidx storage codec, even in comparisons. Every production candidate starts from provider float at 1024 dimensions, applies local prefix reduction and L2 normalization, and then applies the selected cidx codec.

The binary spike must select one complete, versioned contract: mapping target-space values to bits, packing order, padding, zero/invalid-vector behavior, codec-specific query preparation, and similarity scoring. Do not assume Hamming distance, asymmetric scoring, or provider compatibility without recorded evidence. The int8 spike must fix scale, rounding, clamp, norm, and its matching scorer in the same way.

The target float space remains cosine. Record whether each codec scorer is an approximation or a reconstruction-based estimate, and compare ranking against exact target-f32 cosine. Do not expose codec-specific raw scores as exact cosine values.

Query output remains in memory during comparison and is never stored.

Before transformation, durably store paid source-dimension **document** f32 from this spike in an isolated lab artifact with source profile, canonical-input hash, dimensions, and checksum. Never put it in the production database. When Phase 02 supplies the formal lab schema, only rows whose metadata and checksum revalidate may be migrated or imported. Never auto-accept an unidentifiable temporary blob. Do not retain API direct-target comparison output or query output as long-term raw originals.

### CLI

This phase does not build the final public CLI. A required spike runner accepts explicit input and output paths and is not exposed as a production `cidx` command. Any paid runner first prints its dry-run plan and distinct input count, and requires a separate apply confirmation before calling the API.

## 7. Config Usage and Change Impact

Phase 01 does not finalize `.cidx/config.json`. It records candidate values as explicit spike-manifest inputs.

- SQLite binding and build flags: distribution/build decision, not part of a profile
- SQLite pragmas and `busy_timeout`: operational policy
- Grammar and parser versions: future index profile
- Source dimensions: `ModelSpec.SourceDimensions=1024`, the single authority for future `EmbeddingSourceProfile.SourceDimensions` and lab raw
- Target dimensions: one member of `ModelSpec.AllowedTargetDimensions={256,512,1024}`, the future `VectorSpaceProfile` authority for the production retrieval space
- Reduction, normalization, quantization codec, and versions: future materialization/serving profile

The official `voyage-code-4` batch-token cap is a capability this spike must verify. Before verification, do not guess a number and record it as a manifest default, registry ceiling, or production batching contract.

Do not duplicate a dimension or codec value with the same meaning across package constants. Until Phase 02 provides `ResolvedConfig`, take each value from one spike manifest and inject it into every path.

## 8. Ordered Implementation Checklist

1. Tabulate FTS5 inclusion, license, CGO requirements, and supported platforms for each SQLite binding candidate.
2. Execute `CREATE VIRTUAL TABLE ... USING fts5` and a basic query to prove FTS5 availability.
3. Validate add, update, delete, and transaction rollback for contentless FTS rows.
4. Separate WAL reader and writer connections and apply a bounded `busy_timeout`.
5. With an old reader open, publish a new generation under both publication candidates and verify that no reader sees a mixed snapshot.
6. Inject publication failure and verify that active generation and FTS roll back together.
7. Embed Go, TypeScript, and TSX grammars and parse small sources without runtime downloads.
8. Record build support and CGO toolchain requirements on every candidate OS/architecture.
9. Validate f32 blob encode/decode and invalid-blob rejection.
10. Validate both cidx codec candidates end to end: encode/decode or score-equivalence properties, blob length, zero-vector behavior, finite values, dimensions, codec-specific metadata, and matching scanner behavior.
11. If comparison is needed, use an explicitly approved run to embed fixed document inputs with `input_type=document` at source dimension 1024 and direct targets 512 and 256.
12. Durably store source-dimension document f32 in an isolated lab artifact first, then read it back and validate its checksum.
13. Only when considering a direct-target optimization, compare source prefix-plus-L2 with API target output and record the compatibility conclusion. If no comparison runs, keep direct-target requests disabled.
14. Fix the selected SQLite, Tree-sitter, publication, and codec directions in the decision log.
15. Remove rejected spike paths from the production build and leave only the minimum API Phase 02 will consume.

## 9. Failure, Rollback, Concurrency, and Security

- Exclude a binding whose FTS5 support is runtime-optional rather than silently disabling lexical search in deployment.
- If schema or publication validation fails, do not switch automatically to a fallback implementation; leave the phase incomplete.
- Serialize writers only during the publication commit. Do not adopt an application-wide mutex that blocks readers.
- Do not use `BEGIN EXCLUSIVE` on the normal path.
- Never download parser grammars at runtime.
- Do not include API keys, canonical document inputs, or raw vectors in logs or completion evidence.
- A spike database must not reuse production `.cidx/index.db`.
- Paid document source f32 goes directly to an isolated lab artifact, never through a production transaction or normal runtime store.
- Bound retries for failed API comparisons. Never retry authentication, model, or dimension errors indefinitely.

## 10. Validation Scenarios

Run these scenarios during implementation; writing this plan adds no test code.

- A build unable to create an FTS5 table fails clearly during startup.
- A pre-publication reader observes only the old manifest and old FTS; a post-publication reader observes only the new manifest and new FTS.
- An error while applying a delta leaves the active generation and FTS row count unchanged.
- Even with a long reader, a new reader obtains a consistent new snapshot after the writer commits.
- Go, TypeScript, and TSX source parses with the network blocked.
- Invalid f32, binary, or int8 blob lengths and NaN/Inf never become valid vectors.
- f32 encode/decode round-trips at the byte level, and codec metadata mismatches are rejected.
- If direct-target comparison runs, it produces a reproducible manifest comparing source-prefix-plus-L2 with API target dimensions. If it does not run, the record states that direct target remains disabled.
- Comparison query float32 appears in no database or artifact.

## 11. Completion Evidence

Implementation and main-agent commit-boundary validation evidence is recorded in [Phase 01 runtime/storage evidence](evidence/phase-01/README.md).

- Selected SQLite: `modernc.org/sqlite` v1.47.0, BSD-3-Clause; FTS5 startup probe and WAL are mandatory. Tree-sitter: `go-tree-sitter` v0.25.0 plus Go v0.25.0 and TypeScript/TSX v0.23.2 grammars, all MIT; grammar compilation requires CGO.
- Verified platform: darwin/arm64 with Apple clang. Other platforms are explicitly unverified, not unsupported by claim or silently supported.
- Contentless FTS5 create/search/insert/update/delete and injected publication rollback passed in core tests. Reader/writer stores use WAL and a 2-second bounded busy timeout; the modernc DSN configures both connection-local pragmas for every pool connection, proven across four held reader connections.
- The held reader observed only generation 1 after generation 2 published; a new reader observed only generation 2. The selected path is in-place active-table replacement and one final transaction.
- Lab f32 little-endian round trip, checksum/length/NaN rejection, binary padding/zero rejection, and int8 length/metadata/zero rejection passed. Int8 norm is recomputed from the exact persisted float32 scale and blob, and the scorer is bounded to its documented range. Binary and int8 scorer IDs and approximation semantics are fixed in the evidence.
- The source API specification is represented without a request: Voyage official, `voyage-code-4`, source 1024, dtype float, document/query roles, `truncation=false`, adapter v1. No provider reference result or raw document artifact exists because paid operation was not approved.
- The temporary isolated lab schema/factory is version 1 and document-only; it is a Phase 02 migration/import handoff. No source-f32 artifact/checksum was produced.
- Direct target API compatibility is **NOT RUN**. The source-1024 local prefix-plus-L2 path remains the only permitted reference; direct target is disabled in code.
- Exact executed and intentionally unexecuted checks, including the CGO-disabled expected failure, are recorded in the evidence file.
- Phase 02 must formalize schema migrations, profile/config injection, and database naming without changing codec contracts or enabling direct target.

Record measurements as observations, never as v1 performance guarantees.

## 12. Downstream Handoff

Provide Phase 02 with:

- selected bindings and shared connection pragmas;
- the final atomic-publication strategy;
- f32/binary/int8 blob formats, scorer pairings, and validator requirements;
- supported-platform and CGO constraints;
- a dimension-compatibility conclusion when tested, or a direct-target-disabled decision, plus the query-transform contract; and
- source-profile fields for provider, model, source dimension, dtype, input-type mapping, truncation, and adapter version, followed by reduction, normalization, and codec identifiers for downstream profiles.

Phase 02 must not finalize a schema or `ResolvedConfig` while this handoff is incomplete.

## 13. Decision Log

| Decision | Status | Basis |
| --- | --- | --- |
| SQLite binding | selected: `modernc.org/sqlite` v1.47.0 | Pure-Go SQLite with a required runtime FTS5 probe and WAL; BSD-3-Clause. |
| Tree-sitter binding and grammar packaging | selected: `go-tree-sitter` v0.25.0 plus Go/TypeScript grammars | Embedded grammar C sources compile with CGO; no runtime grammar download path. Verified only on darwin/arm64. |
| Atomic publication | selected: in-place active-table update | Held-reader and rollback tests proved old-or-new FTS/chunk/generation visibility. Generation staging was not retained as an alternate serving path. |
| Lab f32 encoding | selected: `cidx-lab-f32-le-v1` | IEEE-754 float32 little-endian, exact length and CRC-32 integrity, document-only separate lab factory/schema. |
| Production storage codecs | selected: `cidx-binary-sign-lsb-v1` and `cidx-int8-symmetric-v1` | Binary sign agreement and reconstructed int8 cosine are codec approximations, never labelled exact cosine. |
| Production document/query transform | fixed: identical reducer plus L2 over Voyage 1024-dimensional float output | Document and query input types differ but share one retrieval space and local transform. |
| Source-prefix-plus-L2 versus API target compatibility | disabled; NOT RUN | No paid comparison was approved. Direct API targets cannot replace explicit source-1024 prefix-plus-L2 without a later approved evidence run. |
| Persistent query f32 | excluded | Queries are runtime/evaluation inputs, not lab originals. |
