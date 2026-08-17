# Repository Implementation Instructions

These instructions apply to every change in this repository. They exist so implementation can resume safely after conversation compaction, a new task, or a handoff to another agent.

## Mandatory startup and resume protocol

Before changing implementation code:

1. Read `docs/implementation/EXECUTION-GUIDE.md` completely.
2. Read `docs/implementation/README.md` and identify the active or next eligible phase.
3. Read `docs/implementation/STATUS.md`; do not infer completion from conversation history.
4. Read `docs/implementation/EVALUATION-CONTRACT.md` before work that creates or consumes parser, retrieval, codec, fusion, packaging, assistant-use, or promotion evidence.
5. Read the active phase document completely, including its Context Recovery Checklist, prerequisites, invariants, completion evidence, handoff, and decision log.
6. Read the completion evidence of every prerequisite phase named by the active phase.
7. Inspect the current workspace before editing. Existing changes belong to the user unless proven otherwise.

If these sources disagree, stop implementation and reconcile the documentation first. A chat summary is never the source of truth for phase status or architectural decisions.

## Phase execution rules

- Work only on a phase whose prerequisites have recorded completion evidence.
- Mark a phase `in_progress` in `STATUS.md` before implementation begins and record the owner and entry evidence.
- Parallel phases are allowed only when `README.md` explicitly permits them and their file ownership does not overlap.
- Do not silently expand a phase. Record cross-phase work as a follow-up or update the dependency graph first.
- Before pausing or handing off, update `STATUS.md`, the phase completion evidence, remaining checks, blockers, and the phase decision log.
- Mark a phase `done` only when its required artifacts and evidence exist. Code presence alone is not completion.
- Record checks actually run separately from checks not run. Do not create test code unless the user explicitly authorizes it.

## Contracts that must not be reconstructed from memory

- Local AST and FTS indexing is free and never calls an embedding API.
- Paid document embedding and paid hybrid-query embedding are explicit, separately authorized operations.
- The v1 provider/model is the official Voyage AI API with `voyage-code-4`.
- Document and query requests explicitly use source dimension 1024, `output_dtype=float`, `truncation=false`, and their respective `input_type` values.
- Serving target dimensions are restricted to 1024 or 512 and are read from one validated `ResolvedConfig`. Ordinary tests, initialization, and evaluation default to 1024; 512 is an explicit compact option.
- The product-owned source bank persistently stores validated 1024-dimensional document f32 so either supported int8 target can be rematerialized locally without another provider call. Production serving storage contains only the active cidx-owned `int8` representation, and search/MCP never opens the source bank. Query f32 is never persisted. Binary and 256-dimensional code paths are absent from the product; only their historical artifacts and reports remain as evidence, and any reproduction requires new explicit user approval in a separate non-product tool.
- SQLite is the persistent authority. Search must observe one committed active generation and must not depend on a second authoritative in-memory index.
- Stable MCP exposes exactly `status`, `search`, `read_span`, and `reindex`.
- `search.max_inline_bytes` limits inline source bytes without changing rank or result identity.
- Configuration and semantic profile fingerprints are single sources of truth; do not duplicate dimensions, codecs, or business rules across packages.
- Evaluation is stage-separated and uses explicit denominators, first-loss attribution, frozen paired runs, standalone FTS/dense evidence, and hard gates. Do not create a weighted total quality score or treat activation as quality proof.
- Promotion results are scoped. Phase 12 may establish `core_retrieval`; only a later immutable Phase 14 result that references core plus assistant/host evidence may establish `release_candidate`.
- Int8 fidelity is measured against exhaustive target-dimension f32, while product usefulness is measured against human relevance and paired assistant tasks. Historical Binary/256 measurements remain evidence only. These are separate references; HNSW/ANN metrics are out of scope.

## Evaluation corpus rules

- The user selects the open-source repositories used for evaluation.
- Do not choose, clone, download, or embed a corpus without explicit user approval.
- Track reproducibility metadata, not machine-specific paths: upstream URL, pinned commit, license, language slices, root subdirectory, include/exclude policy, and expected tree/content hash.
- Keep local checkout bindings and all generated indexes, raw vectors, reports, and secrets in ignored local state.
- Do not send corpus code or evaluation queries to Voyage AI until the applicable paid action has been explicitly approved.
- Keep calibration and confirmation datasets separate. Never tune dimensions, codecs, RRF, candidate limits, body budgets, or margins on confirmation results.
