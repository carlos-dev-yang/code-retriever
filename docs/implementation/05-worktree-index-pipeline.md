# 05. Live Worktree Index Pipeline

- Status: `done` — Revision 4 index-profile, segment-target, and compatible local vector-rekey reconciliation passed Terra review and main-agent boundary validation
- Prerequisites: `03-go-chunker`, `04-typescript-tsx-chunker`
- Follow-up phases: `06-fts-search`, `08-raw-embedding-lab`
- Design basis: `local-code-search-mcp-v1-design-r4.md` §3, §4, §5, §9

## Context Recovery Checklist

- Reopen the [implementation index](README.md), [execution guide](EXECUTION-GUIDE.md), and [status ledger](STATUS.md) before continuing.
- Confirm the Phase 01 SQLite/FTS5 and Tree-sitter decisions, the Phase 02 schema/profile/config artifacts, the shared identifier normalizer, and the Phase 03/04 `Chunker` implementations are present and still compatible.
- Re-check these invariants after any context compaction: enumerate tracked plus untracked nonignored live files; hash, parse, and store each file from one byte slice; never call the embedding API; prepare outside the write transaction; publish one complete generation atomically.
- Stop if the runtime source root is unsafe, a required chunker/schema/profile artifact is missing, a path can escape the source root, any file cannot be prepared safely, or portable manifest/profile identity plus base-generation validation cannot prove an atomic publish. Source/state paths are not persisted in DB metadata.
- Before pausing, record executed evidence in §11, capture new architectural choices in §13, and update [STATUS.md](STATUS.md) with the exact next checklist item and unresolved stop condition.

## 2026-08-17 product-profile supersession

Binary/256 are no longer valid desired product profiles. Phase 05 preserves
AST/FTS/canonical inputs and old rows but must never make a retired storage
profile active or reusable. Only 512/1024 int8 may become current. The focused
reproof follows Phase 02 and is governed by
[`RETIRED-VECTOR-PROFILES.md`](RETIRED-VECTOR-PROFILES.md).

The focused reproof is accepted in
[`evidence/phase-05/int8-serving-key-reproof.md`](evidence/phase-05/int8-serving-key-reproof.md).
The existing implementation needed no semantic change: strict equivalence
already admits only the currently requested int8 dimension. Existing fixtures
now show that canonical historical 256 and Binary profiles remain pending,
while an exactly equivalent 1024/int8 row can still be atomically rekeyed.

## 1. Goal

Discover searchable files from the Git working tree as it exists when indexing starts, hash, parse, and chunk each file from the same bytes, then atomically publish a new local search state to SQLite.

At completion, the single application service shared by `cidx index` and the later MCP `reindex` must guarantee all of the following:

- Index tracked files and untracked, nonignored files.
- Read actual files on disk rather than HEAD blobs.
- Never call the Voyage API during free AST and FTS preparation.
- Do not block existing searches during long scan, read, and parse work.
- Publish a new `active_generation` only after every file has been prepared successfully.
- Leave the previous generation intact when a run fails or is cancelled.

## 2. Scope

### Included

- Validate the Git repository root.
- Enumerate candidate files according to Git.
- Apply built-in exclusions and `.cidxignore`.
- Plan added, changed, deleted, and reusable files.
- Read live file bytes and calculate SHA-256.
- Invoke the Phase 03 and Phase 04 language chunkers.
- Prepare chunks, projections, symbols, FTS inputs, and embedding-segment inputs.
- Reconcile index profiles.
- Calculate a deterministic manifest and run summary.
- Atomically publish through a short SQLite transaction.
- Share the same planning path between `--dry-run` and execution.
- Handle cancellation, failure, and races.

### Out of scope

- File watching or immediate indexing on save.
- A Git-commit-only snapshot.
- Embedding API calls or vector creation.
- FTS query execution or result ranking.
- Automatic Git-hook installation.
- Per-function incremental parsing inside a changed file.
- Indexing files reached through symlinks.
- A best-effort mode that publishes a generation after excluding failed files.

## 3. Prerequisites

- The SQLite/FTS5 and Tree-sitter bindings selected in Phase 01 are settled.
- The typed config, profile fingerprints, schema migrations, and reader/writer connection policy from Phase 02 are available.
- The Phase 03 Go chunker and Phase 04 TypeScript/TSX chunker implement the common `Chunker` contract.
- The shared Phase 02 identifier normalizer can produce stored symbols and FTS symbol inputs with identical rules.
- SQLite has schema for the active files, chunks, projections, symbols, segments, and FTS snapshot plus `meta.active_generation`.
- The canonical source root and desired index profile can be read from config.

If any prerequisite is missing, return to its owning phase instead of introducing a temporary substitute.

## 4. Invariants

### 4.1 Input invariants

1. The candidate set is conceptually equivalent to `git ls-files --cached --others --exclude-standard`.
2. Exclude `.git`, `.cidx`, dependency and build outputs, generated or minified files, lockfiles, `go.sum`, oversized files, symlinks, and non-regular files.
3. Canonicalize every internal path as a repository-relative path using `/` separators.
4. Reject absolute paths, `..` escapes, and canonical paths outside the root.
5. Hashing, parsing, and stored `source_body` for one file must all derive from the same byte slice read once into memory.
6. Modification time and size may support diagnostics or fast candidate selection, but SHA-256 is authoritative for change detection.

### 4.2 Publication invariants

When a search reads `G=active_generation`, all of the following belong to the same committed logical snapshot:

- Manifest and stored profile fingerprints.
- Files and indexed SHA-256 values.
- Chunks, projections, and symbols.
- FTS rows and statistics.
- Canonical-input links for embedding segments.
- Indexed `source_body` eligible for return.

A search sees either the complete pre-publish state or the complete post-publish state. It cannot observe partially prepared files, new FTS rows paired with old chunks, or partial output from a failed run.

### 4.3 Cost and responsibility invariants

- The index pipeline does not check whether an API key exists.
- `internal/index` does not import `internal/embed`.
- It does not perform paid work required by an embedding source, vector-space, or storage-profile change.
- Its embedding responsibility ends after locally creating canonical embedding inputs and profile-independent `canonical_input_sha256` values.
- Vector reuse and pending state are derived from the relationship between active inputs and stored vectors or failures; there is no stored `ready` flag.

## 5. Packages, Files, and Types to Implement

Exact filenames may follow repository conventions, but these responsibility boundaries remain fixed.

```text
internal/
  root/
    repository.go           # explicit root and Git-root validation
  ignore/
    enumerate.go            # tracked + untracked nonignored enumeration
    rules.go                # built-in exclusions and .cidxignore
  index/
    service.go              # IndexService orchestration
    request.go              # IndexRequest, IndexResult
    planner.go              # compare live paths with the stored manifest
    reader.go               # safe file read, hash, and metadata
    prepare.go              # invoke chunkers and create PreparedFile
    manifest.go             # deterministic manifest
    publish.go              # short atomic publish
    run.go                  # run lifecycle and failure diagnostics
  store/
    index_snapshot.go       # active snapshot read model
    index_publish.go        # publish-transaction repository
```

Required type responsibilities:

```text
IndexService
  Plan(ctx, IndexRequest) -> IndexPlan
  Execute(ctx, IndexRequest) -> IndexResult

IndexRequest
  Root
  DryRun
  Reason(manual|commit|mcp)
  ResolvedConfigSnapshot

IndexPlan
  ObservedGeneration
  DesiredIndexProfileFingerprint
  Added / Changed / Reused / Deleted paths
  FullRebuildRequired

PreparedFile
  RelativePath / Language
  IndexedSHA256 / ObservedMtime / ObservedSize
  Chunks / Projections / Symbols / FTSInputs / CanonicalEmbeddingInputs

PreparedGeneration
  BaseGeneration
  NextGeneration
  ManifestSHA256
  DesiredAppliedProfiles
  File deltas
  Run summary

IndexResult
  DryRun
  ActivatedGeneration(optional)
  ManifestSHA256(optional)
  Scanned / Updated / Reused / Deleted counts
  Chunk and embedding-input counts
  Failure diagnostics
```

`PreparedGeneration` is an immutable publish payload completed outside the DB transaction. The store package knows nothing about ASTs or Git; it only validates and applies this payload.

## 6. Schema, Internal APIs, and CLI Integration

### 6.1 Data read

- `meta.active_generation`
- Applied index/profile fingerprints.
- Canonical source root.
- Path, language, and indexed SHA-256 for current `files`.
- Existing chunk, projection, and input data required to reuse files.

### 6.2 Data written

- `files`
- `chunks`
- `chunk_projections`
- `symbols`
- `chunk_fts`
- `embedding_segments` or an equivalent active-input link.
- `meta.active_generation`
- `meta.active_manifest_sha256`
- Applied profile fingerprints.
- `head_observed_at_index`, `worktree_dirty_at_index`.
- Success/failure timestamps plus `index_runs` and `index_run_files`.

Prefer the v1 design that updates active tables in place. If the implementation spike instead selects generation-scoped staging, it must still prove the publication invariant in §4 and the same coherent read snapshot.

### 6.3 Application API consumers

- `cidx index` calls `IndexService.Execute` directly.
- MCP `reindex` calls the same method and does not implement a second indexer.
- `cidx index --dry-run` and `reindex(dry_run=true)` share `Plan` and preparation validation but perform no production DB write.
- `--reason commit` is run metadata only; it never changes the file source from the live working tree to HEAD.

### 6.4 Profile reconciliation

This phase applies the Phase 02 `ConfigImpactPlan` to the active snapshot.

- Index-profile mismatch: reparse every candidate file and rebuild chunks, projections, symbols, and FTS even when file hashes are unchanged.
- Canonical-text-profile mismatch: when the index profile still matches, rebuild canonical inputs and `canonical_input_sha256` from stored `source_body` and projections without rebuilding AST or FTS data.
- Embedding-source, vector-space, or storage-profile mismatch: relink active segments to the new serving key. Reuse a valid production vector for that key; otherwise let state derive as pending.
- Serving-policy-only change: do not change generation code data, FTS data, or vector keys.

The metadata update that changes desired profiles to applied profiles is part of the same final publish transaction as file, chunk, FTS, and segment-key deltas. Never publish only part of a profile mismatch. Reconciliation opens neither product source bank nor lab and calls no API. The v1 source profile is `voyage-official` / `voyage-code-4` with `SourceDimensions=1024`; serving dimensions are limited to `{1024,512}` by the central `ModelSpec`.

## 7. Config Used and Change Impact

This phase receives only the typed config resolved by Phase 02. Subpackages do not reopen config files or duplicate numeric limits and extension lists.

| Config category | Example | Change impact |
| --- | --- | --- |
| Language selection | Go, TypeScript, TSX | Index-profile change; full local rebuild |
| Chunker/projection implementation ID | Derived from the binary | Index-profile change; full local rebuild |
| Segment-boundary rules | AST boundaries with a 1024-byte target; no arbitrary split and oversized AST units remain whole; evaluation may compare only 768-, 1024-, and 1536-byte targets | Index-profile change; regenerate segments |
| Symbol/FTS input rules | Normalization/token-input version | Index-profile change; full local rebuild |
| Exclusion rules | Built-ins, `.cidxignore` | Changes manifest and candidate set |
| Maximum source-file size | `index.max_source_file_bytes=1 MiB` | Changes candidate set; reconcile on next index |
| Canonical-text rules | Formatter/profile | Recompute input/hash from stored projections; no API call |
| Embedding model/source space | `voyage-code-4`, source 1024 | Relink serving keys; missing vectors become pending; no API call |
| Search serving values | `return_k`, `candidate_k`, body-byte cap | No impact in this phase |
| Serving dimensions/codec | `{1024,512}` and fixed int8 storage profile | Does not directly trigger an AST/FTS rebuild |

Do not expose arbitrary, unimplemented chunker version numbers in user config. Fingerprints contain only the resolved combination of user-selectable rules and implementation IDs owned by the binary. Absolute safety ceilings are code constants; config can set only a stricter project policy.

## 8. Ordered Implementation Checklist

1. Canonicalize the explicit root and verify it is a Git repository containing `.cidx/config.json`.
2. Fail closed unless the DB canonical source root equals the requested root.
3. In a short read transaction, copy the config snapshot, applied profiles, active generation, and file manifest; then close it.
4. Acquire the index-writer-only `index.lock`. After acquisition, reread the active generation and refresh the plan base.
5. Enumerate tracked and untracked nonignored paths using Git's NUL-safe output.
6. Apply built-in exclusions, `.cidxignore`, language, file-type, and size policies.
7. Compare sorted live paths with the stored manifest to produce added, changed, reused, and deleted candidates.
8. Open each candidate without following symlinks and read its bytes once.
9. Make the final changed-versus-reused decision from SHA-256 of those bytes.
10. Parse new and changed files with the language chunker and validate every range, UTF-8, and source-body invariant.
11. Use the shared Phase 02 normalizer to produce original, qualified, and split symbol forms for each chunk, then construct deterministic `FTSInputs` with the projection body.
12. Preserve existing outputs for reusable files. On index-profile mismatch, reparse and rebuild symbol/FTS input even if SHA-256 is unchanged.
13. On canonical-text-profile mismatch, recalculate canonical input and hash even for reusable chunks and projections.
14. On source, vector-space, or storage-profile mismatch, plan new serving-key links and vector reuse or pending state.
15. Prepare all file deltas, canonical embedding inputs, and the sorted manifest outside the DB.
16. Check cancellation and per-file errors. Publish nothing unless every item was prepared safely.
17. For dry-run, return plan counts and errors without DB writes or successful-run metadata.
18. For execution, enter a short writer gate and verify the base generation is still active.
19. In one write transaction, apply deletions, added/changed outputs, FTS data, applied profiles, metadata, and the successful run.
20. Return `ActivatedGeneration` only after commit.
21. If failed-run recording is required, write it in a separate short transaction.
22. Release locks and temporary resources, then return the manifest/result consumed by later phases.

## 9. Failure, Rollback, Concurrency, and Security

### Failure and rollback

- A failed Git command, failed file open, root escape, unsupported file type, or parse result without safe exact ranges fails before publish.
- Tree-sitter error nodes are acceptable only when exact chunk ranges can still be constructed safely; otherwise fail the entire run.
- Let SQLite roll back a failed publish transaction; `active_generation` remains unchanged.
- When commit outcome is ambiguous, reopen the DB and verify active generation and run ID before reporting success.
- Cancellation aborts immediately before publish. Cancellation arriving after final commit does not undo a successful generation.

### Concurrency

- `index.lock` prevents only concurrent index/reindex writers.
- Do not hold a SQLite write transaction or process-wide application mutex during scan, hash, or parse.
- Search and status do not acquire `index.lock`.
- Only short final index publishes and vector/failure commits serialize on SQLite's writer lock.
- If another process changes generation during preparation, do not publish the stale plan. Replan after lock acquisition or stop safely with `BASE_GENERATION_CHANGED`.

### File races

- The bytes read for one file form a self-contained snapshot. Even if the file changes later, stored body and stored hash match each other.
- A post-index difference from the current file appears as `stale` in later status/search freshness checks.
- Fail if a symlink swap or root escape is detected while reading.

### Security

- Consume Git paths as NUL-delimited data; never interpolate filenames into a shell command string.
- Reject absolute paths and traversal.
- Exclude `.git`, `.cidx`, and secret-output paths by default.
- Do not emit source bodies in logs.
- This phase neither reads API-key environment variables nor accesses the network.

## 10. Validation Scenarios

Only core contract tests authorized for this phase are included; parser syntax coverage remains owned by the prior chunker phases.

1. A first index of a clean repository activates all tracked Go, TS, and TSX files.
2. Adding an untracked, nonignored function makes it searchable after the next index.
3. `.gitignore`, `.cidxignore`, and `.cidx` content is excluded.
4. Running with `--reason commit` after a partial commit still includes remaining uncommitted changes.
5. A file with the same hash and index profile is not reparsed.
6. A changed file hash reparses the whole file, while unchanged `canonical_input_sha256` values remain eligible for later vector reuse.
7. Changing only the canonical-text profile rebuilds canonical input keys without rebuilding AST or FTS data.
8. Changing only serving dimensions or codec applies a new serving key without rebuilding AST/FTS or calling an API.
9. File deletion removes files, chunks, and FTS rows in one transaction.
10. A parse failure in one file exposes none of the prepared outputs for other files in the active snapshot.
11. Searches immediately before and after publish each see exactly one complete old or new generation.
12. Search continues reading the existing generation during a long reindex scan and parse.
13. Dry-run reports the same change plan as execution but changes neither DB nor successful-run state.
14. If a file changes again during indexing, stored body/hash remain consistent and later freshness reports stale.
15. Symlinks, `..`, absolute paths, and root mismatch fail closed.
16. The previous generation remains searchable after cancellation, disk-full, or constraint errors.

## 11. Completion Evidence

Historical implementation evidence is recorded in [`evidence/phase-05/README.md`](evidence/phase-05/README.md). Revision 4 target-segment, forced local rebuild, and compatible vector-rekey evidence is recorded in [`evidence/phase-05/revision-4.md`](evidence/phase-05/revision-4.md). The focused integration checks cover deterministic Git enumeration, built-in and Git-ignore exclusion, live tracked/untracked preparation, unchanged/change/delete behavior, exact persisted chunks and segment inputs, unsafe source rejection, cancellation before publication, and the production-store transactional snapshot boundary. No corpus, provider, paid, FTS-search, CLI, or MCP evidence was run.

- Report of tracked plus untracked nonignored enumeration and exclusion reasons.
- Diagnostic evidence that hash, body, and chunks came from identical bytes.
- First, incremental, deletion, and profile-mismatch dry-run and execution results.
- Generation and manifest records before and after atomic publish.
- DB inspection showing the previous generation survives preparation, parse, or commit failure.
- Evidence that concurrent search never saw a mixed snapshot.
- Transaction trace showing no long write transaction during scan/parse.
- Execution log or mock-boundary evidence showing zero external API calls.

Separate scenarios actually executed from platform and load checks that remain unexecuted in the completion report.

## 12. Follow-up Handoff

### Revision 4 reconciliation checkpoint

Phase 05's active Revision 4 reconciliation injects `TargetSegmentBytes` only as an AST packing target; it does not restore a semantic-parent chunk cap. The changed index profile therefore requires a full free local rebuild. During that same transition, a pre-R4 serving vector may move to the desired serving key only when stored canonical legacy profile JSON reproduces its recorded fingerprints and proves exact source, target-dimension, reducer, normalizer, metric, codec, canonical-input, lineage, timestamp, and blob equivalence. The immutable copy plan is re-proven inside the final publish transaction and is applied atomically with segment links, profile metadata, and the generation switch. All other keys remain pending; Phase 10 does not own this historical rekey transition.

Provide Phase 06 with:

- `chunk_fts` and chunk metadata coherently linked to one active generation.
- Deterministic symbol/body FTS inputs.
- An `IndexSnapshotReader` API that reads generation, FTS candidates, and chunks in one transaction.
- Index-profile fingerprint and manifest.
- Evidence that free search can read the prior snapshot concurrently with indexing.

For Phase 08 and later, provide active `canonical_input_sha256` values and projection/segment ranges, but do not take responsibility for API calls or raw/vector creation.

## 13. Decision Log

| Decision | Rationale | Revisit when |
| --- | --- | --- |
| Read the live working tree, not HEAD | Local helper search must include uncommitted and untracked work | A separate commit-snapshot product requirement appears |
| Use one read byte slice for file hash and parse | Guarantees internal consistency for each file | An OS snapshot facility becomes an explicitly supported dependency |
| Enumerate tracked plus untracked nonignored files | Prevents new functions from disappearing before `git add` | Never for v1; core contract |
| Any unsafe file-preparation failure blocks the entire new generation | Keeping the previous complete state is safer than publishing a partial mixed state | An explicit best-effort product mode is required |
| Perform long preparation outside the DB write transaction | Prevents management work from serializing existing searches | Never for v1; core concurrency contract |
| Use only a short final publish transaction | Guarantees atomic old/new snapshots | Another storage strategy proves the same invariant |
| Build canonical inputs locally and never call the embedding API | Preserves the free-index versus paid-embedding boundary | Never for v1; core cost contract |
| Copy/rekey an inactive pre-R4 serving vector only after strict semantic and blob equivalence is proven | The segment serving-key switch, metadata update, and free local reuse must publish atomically; Phase 10 only fills current-profile misses | The relational storage model or serving-vector identity changes |
