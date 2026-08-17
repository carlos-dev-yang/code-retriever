# Implementation Execution and Context-Recovery Guide

This guide is the durable operating procedure for the cidx v1 implementation. It must be re-read whenever work resumes after context compaction, a new conversation, or an agent handoff.

## 1. Sources of truth

Use the following order of authority:

1. The user's latest explicit instruction.
2. Repository `AGENTS.md` files that apply to the files being changed.
3. The canonical v1 design: [`../../local-code-search-mcp-v1-design-r4.md`](../../local-code-search-mcp-v1-design-r4.md).
4. The implementation index and dependency graph: [`README.md`](README.md).
5. The evaluation and promotion contract: [`EVALUATION-CONTRACT.md`](EVALUATION-CONTRACT.md), when evidence or quality claims are in scope.
6. The active phase document.
7. The persistent phase ledger: [`STATUS.md`](STATUS.md), phase evidence, and decision logs.

Conversation history may explain a decision, but it does not prove that a phase is complete. If the documents disagree, stop and reconcile them before implementing code.

## 2. Context-recovery procedure

At the start of every implementation session:

1. Open `STATUS.md` and locate every `in_progress` phase.
2. If no phase is active, select only a `planned` phase whose prerequisite rows are `done` with evidence.
3. Read this guide, the implementation index, the full active phase document, and prerequisite completion evidence.
4. Inspect the workspace and identify existing changes, generated artifacts, and file ownership.
5. Reconstruct the phase entry gate from files, not memory:
   - required types and schemas exist;
   - prerequisite fingerprints or migrations are fixed;
   - no unresolved decision changes the external contract;
   - the phase owns the files it plans to edit.
6. Record the owner, start point, and intended completion evidence in `STATUS.md` before editing code.
7. If any gate fails, leave the phase `planned` or mark it `blocked` with concrete evidence. Do not improvise a substitute architecture.

## 3. Phase lifecycle

Allowed states are `planned`, `in_progress`, `blocked`, and `done`.

- `planned`: work has not started, or prerequisites are incomplete.
- `in_progress`: one named owner is actively responsible for the phase and the entry gate is recorded.
- `blocked`: an external decision or missing prerequisite prevents meaningful progress; the blocker and required decision are recorded.
- `done`: all required artifacts exist, completion evidence is recorded, and the documented handoff is usable by downstream phases.

A phase may return from `done` to `in_progress` only when a contract change invalidates its evidence. Record the reason and all downstream phases that need revalidation.

## 4. Required phase handoff record

Before pausing, compaction, or handoff, update the phase document and `STATUS.md` with:

- the last completed checklist item;
- files created or changed;
- schema/API/profile decisions made;
- commands and checks actually run, with result summaries;
- checks not run and why;
- generated evidence and artifact paths;
- unresolved risks or blockers;
- the exact next action;
- whether downstream phase entry gates are now satisfied.

Never use “mostly done” as evidence. A successor must be able to resume without relying on private reasoning or chat history.

## 5. Contract-change procedure

When implementation evidence requires a design change:

1. Stop the active implementation path before changing a public schema, profile fingerprint, migration, paid-operation boundary, or MCP contract.
2. Identify every affected phase and artifact.
3. Update the canonical design, implementation index, relevant phase documents, and `STATUS.md` in one coherent documentation change.
4. Record invalidated evidence and required reruns.
5. Obtain user direction when the change creates a new product decision, external contract, migration risk, or paid behavior.
6. Resume code only after the new entry gate is explicit.

Implementation convenience is not authority to change the product contract.

## 6. Configuration and profile discipline

- Parse configuration once: `RawConfig -> Resolve -> Validate -> immutable ResolvedConfig`.
- Inject typed profiles into packages; packages must not reread JSON or carry copied dimension/codec constants.
- Keep code-owned protocol and algorithm identifiers in a central registry.
- Accept only the code-owned `int8` storage codec in the product. Keep its ID
  in profiles and rows, but expose no public codec selector. Historical Binary
  code and artifacts are evidence-only and must not be reachable from ordinary
  initialization, materialization, serving, search, or evaluation.
- Use `serving_dimensions` for the active vector length and `--serving-dim` for its CLI spelling. Never use this value to represent source paths, line ranges, or repository scope.
- Keep the provider source response fixed at 1024 dimensions, durably preserve document responses in the product source bank, and restrict product serving dimensions to 1024 or 512; ordinary tests use the 1024/int8 default.
- Treat the 1 MiB source-file ceiling, 1,024-byte AST-aware segment target, 64 KiB inline default, and 1 MiB inline executable ceiling as separate named contracts.
- Do not reintroduce a configurable chunk byte cap or a read-span line-count cap.
- Fingerprint resolved semantic values, not the whole config file or environment-specific paths.
- Treat row-level dimensions and codec identifiers as integrity metadata, not a second configuration source.
- Classify every setting change as one of: restart/reload only, local reindex, local rematerialization, paid re-embedding, or schema migration.
- Fail closed on unknown fields, unsupported algorithms, profile mismatch, blob-size mismatch, and incompatible dimensions.
- Never interpret Voyage provider-side quantized output as the cidx storage codec. The cidx int8 path starts from the validated 1024-dimensional float response.

## 7. Persistent storage and concurrency discipline

- `<state_root>/db/index.db` is the production authority; normal use resolves this to `<source_root>/.cidx/db/index.db`. The server must remain restartable without reconstructing state from Go heap caches.
- Keep `source_root` and `state_root` distinct in application assembly. Normal use defaults state to the source project's `.cidx`; development evaluation explicitly keeps disposable corpus sources under `.cidx/test/corpora/` and preserved state under `.cidx/test/states/` in the controlling cidx repository.
- Never persist an absolute source or state path in production or lab SQLite metadata. Runtime canonical paths are process-local safety inputs; portable commit/content/manifest/profile/input identities prove compatibility.
- Use WAL, separate reader/writer connections, short write transactions, and one atomic active-generation publish.
- Do parsing, hashing, filesystem scans, and external API waits outside write transactions.
- Pin each search to one SQLite read snapshot. FTS statistics, candidates, chunks, segments, vectors, coverage, and bodies must come from the same active generation.
- A search may use bounded temporary buffers and SQLite/OS page cache, but v1 does not maintain a second authoritative full in-memory index.
- Development raw storage is physically and dependency-wise isolated. Runtime packages must not open or import it.

## 8. Paid-operation discipline

- `index`, `reindex`, FTS search, status, and read-span operations must work without an API key or network.
- A paid document or query request requires the documented explicit approval path.
- Show planned input count, token estimate, and dated price information when available before paid apply.
- Use `VOYAGE_API_KEY` only from the process environment; never persist or log it.
- Document requests use `input_type=document`; query requests use `input_type=query`.
- Both explicitly request `output_dimension=1024`, `output_dtype=float`, and `truncation=false`.
- Validate response count, index uniqueness/range, model, exact dimension, and finite values before any publish.
- Use only the regular synchronous embeddings endpoint. Asynchronous Batch Inference is outside v1.
- Group at most 128 inputs and 256 KiB of canonical input per synchronous request, run at most four requests concurrently, and apply a 30-second request timeout.
- Make one initial attempt and at most three transient retries after 10, 20, and 30 seconds. This is staged linear backoff, not exponential backoff; a longer valid `Retry-After` wins and cancellation stops waiting.
- Treat request byte limits as bytes, not token estimates. Do not invent a `voyage-code-4` batch-token cap.

## 9. Open-source evaluation corpus protocol

The user owns corpus selection. cidx owns reproducibility and safety checks.

Tracked corpus manifests must contain:

- stable `corpus_id`;
- upstream repository URL;
- pinned commit;
- declared license/SPDX identifier and redistribution notes;
- language slices (`go`, `typescript`, `tsx`, or mixed);
- repository root subdirectory when applicable;
- include/exclude and generated/vendor-file policy;
- expected Git tree or deterministic content-manifest hash;
- dataset versions allowed to reference the corpus.

Do not commit an absolute checkout path. An ignored relative local binding such as `.cidx/test/corpora.local.json`, or an explicit development CLI argument, maps `corpus_id` to a checkout below the controlling project. Evaluation state is separately selected below `.cidx/test/states/`.

Before an official run, verify the checkout URL/commit, clean worktree, content hash, license record, expected indexed files, and absence of credentials. Do not automatically clone, update, or embed a repository. Paid capture and paid evaluation-query embedding remain separate explicit approvals.

Report Go, TypeScript, TSX, and mixed-repository slices separately where applicable. Aggregate results must include per-slice counts so one language cannot hide another language's failure. Numeric hit-rate and latency thresholds are observations, not v1 release gates.

## 10. Evaluation and promotion discipline

- Use the stage scorecard and denominators in [`EVALUATION-CONTRACT.md`](EVALUATION-CONTRACT.md); never replace them with a weighted total.
- Preserve both FTS and dense lane observations before RRF and attribute first loss along the provider-union, collapse, fusion, body, and assistant path.
- Use human relevance for usefulness and exhaustive serving-dimension f32 for int8 fidelity. Historical Binary/256 results remain evidence-only. Neither reference substitutes for the other.
- Treat required failures and timeouts as denominator members. Use `NOT_OBSERVED` only for a downstream stage that the run contract did not require.
- Freeze corpus, labels, controls, candidate policy, profile, generation, and artifact checksums. Only compatible paired runs support delta claims.
- Select parameters and margins on calibration data, freeze them before confirmation, and let only complete confirmation evidence vote for promotion.
- Require zero/100% invariant gates where specified, then calibrated paired noninferiority gates. Latency, size, and cost remain observations until a pre-result budget is frozen.
- Activation and post-activation readiness prove lifecycle integrity, not retrieval quality.
- Treat promotion results as immutable and scoped: Phase 12 `core_retrieval` cannot imply Phase 14 `release_candidate`; the latter references rather than rewrites the former.
- Exclude HNSW/ANN recall, `ef_search`, graph health, and ANN tuning from cidx v1 evaluation.

## 11. Validation and completion evidence

Validation must be proportional to the active phase and directly affected integrations.

Record:

- exact commands or programmatic checks run;
- deterministic fixture, schema, manifest, or transcript paths;
- successful and failed-path results;
- checks intentionally not run;
- remaining environment or provider risks.

Do not create test code unless the user explicitly requests it. Existing tests and phase-specific inspection tools may be run when relevant. Never claim that an API contract, platform package, or concurrency property was validated when it was only documented.

## 12. Final pre-handoff audit

Before declaring a phase complete:

1. Re-read its Context Recovery Checklist and invariants.
2. Confirm all owned files and artifacts exist.
3. Confirm no environment-specific path, credential, endpoint override, or duplicated business rule was introduced.
4. Confirm paid and free paths remain separated.
5. Confirm configuration/profile invalidation behavior is documented and implemented consistently.
6. Fill the phase completion-evidence section with actual results.
7. Update the phase decision log and `STATUS.md`.
8. Verify downstream handoff types, schemas, fixtures, and reports by their real paths.
9. When evaluation is in scope, verify that promotion artifacts contain paired per-query records, first-loss/cohort results, all applicable hard gates, and checksums.
