# cidx v1 Persistent Implementation Status

This ledger is the authoritative resumable state for implementation work. Update it before starting a phase, before pausing, after context compaction is noticed, and before marking a phase complete.

## Current state

- Active phase: none — Phase 09 accepted at its commit boundary
- Active owner: Codex
- Last updated: 2026-08-15
- Current blocker: Phase 07 needs user-selected corpus manifests; any Phase 08 paid capture still requires explicit approval
- Latest contract change: production storage codec now defaults to cidx-owned `binary`; `int8` is the only alternative, and exact encoder/scorer contracts remain a Phase 01 evidence item
- Latest evaluation change: strict Phase 02 traces use per-group first-loss observations with an explicit provider-union stage, typed operation denominators, frozen review-pass identities, ABSTAINABLE/no-answer truth, and graded durable relevance judgments
- Next eligible phase: Phase 10 embedding orchestration; Phase 07 remains gated on user-selected corpus manifests
- Exact next action: enter Phase 10 without making a provider request, reading an API key, or accessing a corpus

## Phase ledger

| Phase | Status | Owner | Prerequisite evidence checked | Last completed checkpoint | Evidence / blocker | Next action |
| --- | --- | --- | --- | --- | --- | --- |
| 00 | done | Codex | n/a; no prerequisite phases | RFC 8785 profiles and all Phase 00 catalogs/reviews completed | [Phase 00 evidence index](evidence/phase-00/README.md) | Enter Phase 01 |
| 01 | done | terra/high implementation agent; Codex validation | Yes — [Phase 00 evidence](evidence/phase-00/README.md) | Executable spikes and [Phase 01 evidence](evidence/phase-01/README.md) validated | Core, race, vet, build, runner, dependency-boundary, format, and module checks passed | Enter Phase 02 |
| 02 | done | terra/high implementation agents; Codex validation | Yes — [Phase 00](evidence/phase-00/README.md) and [Phase 01](evidence/phase-01/README.md) evidence | Formal migrations, active-vector state, secure store factories, shared chunk handoff, strict evaluation wire/schemas, and [Phase 02 evidence](evidence/phase-02/README.md) accepted | Main commit-boundary core, race, vet, build, dependency, format, module, schema, and diff checks passed | Enter Phase 03 |
| 03 | done | terra/high implementation agent; Codex validation | Yes — [Phase 02 evidence](evidence/phase-02/README.md) and `internal/chunk` shared contracts | Go Tree-sitter adapter, exact chunk/projection/segment fixtures, decision log, and [Phase 03 evidence](evidence/phase-03/README.md) accepted | Main focused test, race, vet, build, format, and diff checks passed | Enter Phase 04 |
| 04 | done | terra/high implementation agent; Codex validation | Yes — [Phase 02 evidence](evidence/phase-02/README.md), [Phase 03 evidence](evidence/phase-03/README.md), and `internal/chunk` shared contracts inspected | TypeScript/TSX adapters, exact callable/type projections, overload and segmentation fixtures, Phase 04 decisions, and [Phase 04 evidence](evidence/phase-04/README.md) accepted | Main focused test, race, vet, build, dependency-boundary, format, and diff checks passed | Enter Phase 05 |
| 05 | done | terra/high implementation agent; Codex validation | Yes — [Phase 02](evidence/phase-02/README.md), [Phase 03](evidence/phase-03/README.md), and [Phase 04](evidence/phase-04/README.md) evidence plus shared store/chunk contracts inspected | Live worktree preparation, exact profile reconciliation, atomic delta publication, and [Phase 05 evidence](evidence/phase-05/README.md) accepted | Main boundary core/race, vet, build, format, dependency-boundary, and diff checks passed after correcting the canonical final-LF fixture | Enter Phase 06 |
| 06 | done | terra/high implementation agent; Codex validation | Yes — [Phase 05 evidence](evidence/phase-05/README.md) and the store/config/symbol handoff inspected | Safe query construction, central resolved query policy/fingerprint, generation-pinned FTS/BM25 materialization with full pre-limit ordering, and [Phase 06 evidence](evidence/phase-06/README.md) accepted | Main focused race, vet, build, format, dependency-boundary, and diff checks passed | Enter Phase 07 or the unpaid implementation portion of Phase 08 |
| 07 | planned | — | No | None | Requires 06 and user-selected corpus manifests | Wait |
| 08 | done | terra/high implementation agent; Codex validation | Yes — [Phase 02](evidence/phase-02/README.md) and [Phase 05](evidence/phase-05/README.md) evidence plus profile/store handoff inspected | Voyage adapter, cache-first raw capture, lab schema v2 migration, resumable failure handling, shared embed lock, and [Phase 08 evidence](evidence/phase-08/README.md) accepted | Main focused race, vet, build, format, dependency-boundary, migration, retry, rollback, and diff checks passed; live provider evidence remains NOT RUN | Enter Phase 09 |
| 09 | done | terra/high implementation agent; Codex validation | Yes — [Phase 01](evidence/phase-01/README.md), [Phase 02](evidence/phase-02/README.md), [Phase 05](evidence/phase-05/README.md), and [Phase 08](evidence/phase-08/README.md) evidence reviewed | Shared transform/codecs, lab v3 staging evidence, production v2 lineage, narrow active-key planning, and complete-set atomic publication accepted | Main focused race, vet, build, migration, dependency-boundary, format, module, diff, and synthetic fidelity checks passed; no provider/corpus/paid action | Enter Phase 10 |
| 10 | planned | — | No | None | Requires 05, 08, 09 | Wait |
| 11 | planned | — | No | None | Requires 06, 09, 10 | Wait |
| 12 | planned | — | No | None | Requires 07, 08, 09, 11, approved corpus bindings, and paid-query approval | Wait |
| 13 | planned | — | No | None | Requires 05, 06, 10, 11, 12 | Wait |
| 14 | planned | — | No | None | Requires 13 | Wait |

## Resume note template

Copy this block into the active phase's working note when work starts or resumes:

```text
Phase:
Owner:
Entry evidence checked:
Last completed checklist item:
Files changed:
Checks run and results:
Checks not run:
Decisions made:
Remaining risks/blockers:
Exact next action:
Downstream handoff readiness:
```

## Status update rules

- Keep one row per phase; do not encode progress only in chat.
- Use repository-relative evidence paths and stable artifact identifiers.
- Never record secrets, absolute environment-specific corpus paths, or raw source text here.
- If a contract change invalidates a completed phase, move it back to `in_progress`, explain why, and list every downstream phase that requires revalidation.
- `done` requires the completion evidence named in the phase document. A checked task list without evidence is insufficient.
