# 10. Embedding Orchestration and Profile Reconciliation

- Status: `planned` — pre-R4 orchestration is historical; byte grouping, bounded concurrency, and staged retry require reconciliation
- Prerequisites: `05-worktree-index-pipeline`, `08-raw-embedding-lab`, `09-vector-materialization`
- Followed by: `11-vector-and-hybrid-search`, `13-cli-and-mcp`
- Design source: `local-code-search-mcp-v1-design-r4.md` sections 4.4, 6, and 7

## Context Recovery Checklist

Read the [implementation index](README.md), [execution guide](EXECUTION-GUIDE.md), and [project status](STATUS.md) before resuming.

- Confirm that Phase 05 publishes the active index and canonical document inputs atomically, Phase 08 provides the isolated raw lab, and Phase 09 provides the shared transformer, codec, and materializer.
- Re-check that indexing and MCP `reindex` are free and never call Voyage AI; only an explicit paid apply may embed documents.
- Re-check the source contract: `voyage-code-4`, explicit 1024-dimensional float output, `input_type=document`, `truncation=false`, then shared `prefix(serving_dimensions) -> L2 -> selected cidx codec` for serving dimensions 256, 512, or 1024. V1 codecs are `binary` and `int8`; `binary` is the default.
- Re-check that one serving profile is active, raw f32 is not a production dependency, query f32 is never persisted, and an external API wait holds no SQLite transaction or application-wide search mutex.
- Stop if source/profile identity, the raw-lab boundary, paid-call authorization, or publication atomicity is unresolved. Do not improvise provider plugins, multi-profile serving, or vector import.
- Before pausing, update this phase's evidence and decision log, then update [STATUS.md](STATUS.md) with completed checks, open risks, and the exact next action.

## 1. Objective

Connect free local indexing, paid document embedding, local vector transformation, and publication of the current serving-profile vectors through one application workflow.

Initial development and evaluation use `embeddings capture -> embeddings materialize --activate`. Normal usage later uses public `cidx embed --apply`: it receives a 1024-dimensional f32 response from `voyage-code-4`, passes it through the shared in-memory transform and active `binary` or `int8` codec, and directly prepares the production serving vector without requiring raw preservation.

## 2. Scope and Non-goals

### In scope

- Derive active-input and active-serving-vector state.
- Separate free planning from paid apply.
- Group synchronous requests, call the API, transform, and publish for normal `cidx embed`.
- Define the boundary between development capture/materialization and public embedding.
- Reconcile source, vector-space, and storage-profile mismatches.
- Handle late responses, partial success, retries, and failures.
- Ensure index, embed, and search never invoke one another automatically.

### Out of scope

- API calls during indexing or automatic document embedding during search.
- Treating the raw lab as a mandatory cache for normal embedding.
- Query embedding caches or persisted raw queries.
- External vector import, arbitrary providers, or a long-term model-pinning product.
- Permanent multi-profile serving, schedulers, daemons, or background embedding queues.
- An MCP tool that starts paid document embedding.

## 3. Prerequisites

- The active index snapshot and canonical document inputs publish atomically.
- Phase 08 raw collection and the Phase 09 shared transformer/materializer exist.
- `ResolvedConfig` resolves source, vector-space, and storage profiles exactly once.
- The production store reads generation, manifest, serving fingerprints, coverage, and failures from one consistent snapshot.
- The Voyage client and canonical document-input builder are separate interfaces.

## 4. Invariants

1. `cidx index` and MCP `reindex` never call Voyage AI.
2. Normal `cidx embed` neither calls the API nor writes vectors without `--apply`.
3. One repository `embed.lock` prevents duplicate paid document commands across public embed and development capture.
4. Active input identity is `canonical_input_sha256`, independent of serving dimensions and codec.
5. A segment is ready only when its `ServingVectorKey` has a valid vector row.
6. Validity covers profile fingerprint, dimensions, codec, blob length, scale/norm, and finite metadata.
7. Every source f32 response uses the shared `VectorTransformer`; document and query paths do not duplicate reduction.
8. Normal embedding explicitly requests `output_dimension=1024`, `output_dtype=float`, `input_type=document`, and `truncation=false`, then applies `prefix(serving_dimensions) -> L2 -> selected cidx binary/int8 codec`. Raw f32 may be discarded after a successful runtime commit.
9. Development capture first persists validated f32 in the lab DB and never writes runtime vectors.
10. Development materialization never calls the API and publishes only the current config's single serving profile.
11. FTS remains available during reconciliation. Hybrid falls back without a query API call when no matching serving profile exists.
12. No external API wait holds a SQLite write transaction or application-wide search mutex.
13. A late response enters runtime staging only after rechecking the current generation and desired fingerprints.
14. Raw-lab presence never changes public planning, readiness, or runtime search semantics.
15. Validate response model, count, unique in-range indexes, 1024 dimensions, and finite values before transformation.
16. Provider-native quantization is not a shortcut for either cidx codec; public embedding receives float32 and uses the active Phase 09 codec.

## 5. Implementation Packages, Files, and Types

| Package/file | Responsibility |
| --- | --- |
| `internal/embed/plan.go` | Compute a paid plan from active inputs and vector/failure state |
| `internal/embed/request_groups.go` | Deterministic synchronous request groups within the shared input-count and byte ceilings |
| `internal/embed/runner.go` | Public document-embedding orchestration |
| `internal/embed/failure.go` | Retryable and terminal failure classification |
| `internal/app/embed.go` | CLI use case, confirmation boundary, progress, and result |
| `internal/app/reconcile.go` | Desired/active profile differences and required actions |
| `internal/store/embedding_state.go` | Snapshot join of active inputs, valid vectors, and failures |
| `internal/store/vector_publish.go` | Current-profile incremental upsert and failure removal |
| `internal/app/status.go` | Separate free index state from paid vector state |

Core types:

- `EmbeddingPlan`: captured generation/manifest, desired profile, active distinct keys, reused vectors, paid inputs, failures, and cost estimate.
- `EmbeddingRun`: immutable config snapshot, plan ID, progress, and cancellation state.
- `EmbeddingState`: derived `ready | pending | failed` view and reason.
- `ProfileReconciliation`: desired/active source-space-storage differences and local-only or paid action.
- `DocumentEmbeddingPipeline`: canonical input -> source API -> shared transform -> storage vector.
- `VectorPublishCandidate`: key, fingerprints, stored vector, and source-response provenance.

## 6. Schema, API, and CLI

### Production state schema

`embedding_failures` stores the source-profile fingerprint, `canonical_input_sha256`, `terminal | retryable` class, attempts, last sanitized error, and last-attempt time. When a valid serving vector exists, effective state is ready; a failure cannot override readiness.

`embedding_runs` stores run ID, captured generation/manifest, desired source-space-storage fingerprints, `planned | running | partially_succeeded | succeeded | failed | cancelled` state, active/reused/requested/persisted/failed counts, estimated and actual token/cost usage, and start/finish times. `requested_count` is provider input attempts and includes retries; succeeded, failed, and discarded counts are distinct approved-input outcomes.

Phase 09 owns a full current-profile publish to `vector_cache`. Normal embedding incrementally fills missing keys for that same profile.

### Derived state

For each active canonical input, derive state in this order:

1. If the active segment's serving vector key has a valid vector, return `ready`.
2. Otherwise, if an applicable terminal failure exists for the current paid-source key, return `failed`.
3. Otherwise return `pending`.

A development raw-bank hit is not a production readiness state. Report `locally_materializable` only in a development materialization plan.

### Public CLI

```text
cidx embed
cidx embed --apply
cidx embed --apply --retry-failed
```

- Default execution reports active distinct inputs, reused vectors, paid-request count, token/cost estimate, and reconciliation reason.
- Only `--apply` calls the document API.
- Normal apply does not open the lab DB and immediately transforms the f32 response.
- When filling the same serving profile, retain valid vectors and upsert only missing keys.
- If desired config differs from the active segment key, fail with `PROFILE_RECONCILIATION_REQUIRED` before the API call and require `cidx index` to publish local key reconciliation.
- After reconciliation, active fingerprints and segment keys already identify the new config. Apply writes missing vectors in short transactions. Failed keys may remain absent, but every row used by search has the same active profile.

### Development-only CLI

```text
cidx dev embeddings capture [--apply] [--retry-failed]
cidx dev embeddings materialize [--activate]
```

Use this only for initial quality evaluation and codec choice. Public `cidx embed --apply` must prepare hybrid vectors without it. Do not add a permanent public `--save-raw` option.

### Reconciliation matrix

| Desired change | Index required | API required | Local raw materialization | Hybrid state |
| --- | --- | --- | --- | --- |
| Canonical input format | Recompute local input | New key requires embedding | Possible in dev with matching raw | FTS fallback until ready |
| Provider/source model/source dimensions/request semantics | Reconcile serving key | Required normally | Only with matching source raw bank | FTS fallback until ready |
| Serving dimensions/reducer/normalizer/metric | Reconcile serving key | Normal path may need another source response | Free from initial lab bank | FTS fallback until ready |
| Storage codec | Reconcile serving key | Needed if source f32 is unavailable | Free from initial lab bank | FTS fallback until ready |
| FTS/RRF/response policy | Policy only | No | No | Existing vectors remain valid |

Raw preservation is not permanent. A later space or codec change in normal operation may require another paid call. Do not hide that cost or assume a lab DB.

## 7. Configuration and Change Impact

Production inputs are `embedding.model`, `embedding.serving_dimensions`, `embedding.reducer`, `embedding.normalizer`, `embedding.metric`, and `embedding.storage_codec`, plus the code-defined canonical-input/text-format version. Public embed and server construction must not accept the lab store.

`voyage-code-4` is the v1 default and initially the only validated model. Source dimensions are not configurable: `ModelSpec` resolves `SourceDimensions=1024` and `AllowedServingDimensions={256,512,1024}`. The official endpoint and request semantics belong to the code-owned Voyage adapter.

Use only the regular synchronous Embeddings endpoint, never Voyage Batch Inference or asynchronous polling. The code-owned request policy is at most 128 inputs and 256 KiB total input bytes per request, at most four concurrent requests, and a 30-second timeout. Retry an initial attempt at most three times after 10, 20, and 30 seconds, with a longer provider `Retry-After` taking precedence.

Freeze config at run start. If desired fingerprints change before publication, reject publication with `CONFIG_CHANGED_DURING_RUN`; never switch only later batches.

Keep these sources of truth separate:

- User/evaluation tuning: resolved config.
- API/model capabilities: versioned registry.
- Schema/codec algorithm versions: code constants included in fingerprints.
- Stored dimensions/codec: row integrity metadata.

## 8. Ordered Implementation Checklist

1. Connect profile-independent canonical-input keys to production queries.
2. Implement derived state across active input, valid vector, and failure rows.
3. Implement reconciliation and the requirement to run `cidx index` first.
4. Implement API-free public planning and cost estimation.
5. Wire the shared `embed.lock` across paid document commands.
6. Build deterministic synchronous request groups from the captured snapshot.
7. Build document requests with `input_type=document`, `output_dimension=1024`, `output_dtype=float`, `truncation=false`; omit `encoding_format`; share response validation.
8. Pass validated f32 to the Phase 09 transformer and codec outside the API boundary.
9. Prevent an incremental current-profile publish from switching active fingerprints.
10. Revalidate active references and config fingerprints immediately before each publish.
11. Atomically write success and delete the applicable failure.
12. Implement bounded retry, terminal/retryable classes, and `--retry-failed`.
13. Preserve partial successes and reflect exact state and coverage.
14. Freeze public plan/apply request, result, and cancellation contracts for Phase 13.
15. Report free index state, serving coverage, and paid pending separately.
16. Verify that development raw/materialize is absent from the public dependency path.
17. Hand off pre-query fallback on profile mismatch to Phase 11.

## 9. Failure, Rollback, Concurrency, and Security

### Failure and rollback

- Planning failure changes no DB state and calls no API.
- A grouped API response failing validation writes no vector from that request group.
- Commit successful same-profile request groups durably in short transactions; do not roll them back because a later group fails.
- Keep FTS or partial-coverage hybrid available while a reconciled profile is incomplete.
- A failed generation/profile recheck prevents a late vector from joining the active set.
- A late development-capture response may enter the raw bank. A late inactive public f32 may be discarded but never enters runtime storage.
- A crash before public f32-derived output commits may cause the next run to pay for another request; this follows from not making raw retention a product contract.

### Concurrency

- `embed.lock` serializes public document embed, development capture, and development materialization activation.
- Query embedding does not take the document lock.
- API calls and transforms hold no SQLite write transaction.
- The short vector writer gate is not a search mutex.
- Concurrent index changes require reference revalidation at commit.
- Concurrent search observes one committed snapshot before or after a vector commit.

### Security

- Read credentials only from `VOYAGE_API_KEY`; never persist them in plans or run rows.
- Before apply, disclose that canonical document input leaves the machine.
- Sanitized errors exclude bodies, vectors, and authorization headers.
- Ephemeral public f32 never enters debug logs, crash artifacts, or production DB.
- Only an explicit development command opens the raw lab.

## 10. Validation Scenarios

- Without `VOYAGE_API_KEY`, index, FTS search, and embed planning work; only apply fails.
- A ready same-profile key is not requested again.
- A serving-dimension or codec mismatch falls back before query embedding.
- Public embed creates the selected profile without a lab DB and does not increase lab raw rows.
- Development capture/materialize and public direct paths produce byte-identical serving vectors for the same input/profile.
- API waits do not block status or FTS search through an application lock.
- If an active segment disappears before the response, its vector does not join the active set.
- Success publication and applicable failure removal appear atomically.
- After partial API failure, ready/pending/failed counts and coverage agree within one snapshot.
- A mid-run config change never publishes a mixed profile.

## 11. Completion Evidence

See the resumable [Phase 10 evidence index](evidence/phase-10/README.md) for
the current executed-check record.

Implementation handoff record (2026-08-15; accepted at the main-agent boundary):

- Added the production v2-to-v3 migration. It preflights the exact v2 schema, atomically preserves meta/vector/index data and active-profile pointers, converts preserved v2 failures to terminal historical failures, and adds classified unresolved failures plus `embedding_runs`.
- Added a public no-lab plan/apply application boundary. Planning takes no client or credential; apply requires an explicit approval flag and an injected client. Canonical inputs are reconstructed from the active stored source/projections and verified against their canonical SHA-256 before requests.
- Added a pinned active snapshot with derived ready/pending/failed states; it joins active segment keys, valid current-profile vectors, and latest terminal/retryable failures. Incremental writes revalidate config, generation, manifest, profiles, and active key in a short transaction, then either publish/clear failure or discard the stale key.
- Added direct source-f32 transformation through the shared transformer and selected codec, with an ephemeral little-endian f32 SHA-256 provenance value. Production schema/storage contains no f32 vector representation and the public path has no lab dependency.
- Added focused fake-provider checks for free planning, explicit approval, exact document/1024/float/non-truncating request fields, serving-blob parity with the shared transformer/codec, ready exclusion, retry planning, schema/run records, and the existing store migration/atomic publication checks.
- Checks run: `go test -count=1 ./internal/app ./internal/embed ./internal/lab ./internal/store`; `go test -count=1 -race ./internal/app ./internal/embed ./internal/store`; `go vet ./internal/app ./internal/embed ./internal/lab ./internal/store`; `go build ./internal/app ./internal/embed ./internal/lab ./internal/store`; direct dependency inspection of `./internal/embed ./internal/store` for `internal/lab`; `git diff --check`. All passed; the fake-provider test proves FTS completes while its document request is blocked. No real client, credentials, network, corpus, or paid operation was used.
- Main-agent boundary checks passed: focused race tests for `internal/app`, `internal/embed`, `internal/lab`, and `internal/store`; focused vet/build; formatting; production dependency-boundary inspection; and diff validation.
- Checks not run: live Voyage/provider behavior, real API-key handling, CLI/MCP wiring, query/hybrid integration, corpus/evaluation work, broad project validation, and load testing.

- API-call count difference between free plan and paid apply.
- Identical serving-blob checksums from development two-step and public direct paths.
- Results for every reconciliation-matrix mismatch.
- Partial-success, failure, and retry reports.
- Concurrent FTS completion while a paid request waits.
- Store proof that a late response did not enter the active vector set.
- Inspection proving no f32, key, or source body remains in production DB or logs.
- End-to-end public hybrid preparation without a raw lab.

## 12. Handoff

Phase 11 receives the active `ServingVectorProfile` and fingerprint, valid codec-tagged rows and coverage, the shared transformer for a 1024-dimensional query f32 vector, the matching codec-specific query preparation/scorer, and stable fallback reasons for profile mismatch, missing vectors, missing credentials, or disabled paid queries.

Phase 12 uses the development raw bank and materializer separately. It must not bypass public orchestration to create multiple runtime-active profiles.

## 13. Decision Log

- Public `cidx embed [--apply]` is the normal hybrid-enablement path.
- The initial two-step raw lab saves evaluation cost; it is not a public prerequisite.
- Normal embedding uses Voyage 1024 f32 -> prefix(serving_dimensions)/L2 -> active cidx codec (`binary` by default or `int8`).
- Normal embedding does not require persistent raw f32, and query f32 is never stored.
- Only the serving profile selected by config is active.
- Profile changes leave FTS intact and hybrid safely falls back until ready.
- General-user model locking and external vector injection remain deliberately undesigned.
- Public plan/apply receives credentials only through an injected client; it never reads the environment. The Phase 13 adapter remains responsible for obtaining an explicitly approved credential/client without exposing it in a plan, result, run row, or log.
- Historical v2 failures migrate as terminal because v2 had no retryability field; later retryable attempts append while unresolved and supersede that effective state. A successful current-vector publication deliberately deletes its applicable failure rows atomically.
