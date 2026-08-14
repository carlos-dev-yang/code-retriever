# cidx v1 Persistent Implementation Status

This ledger is the authoritative resumable state for implementation work. Update it before starting a phase, before pausing, after context compaction is noticed, and before marking a phase complete.

## Current state

- Active phase: none; Phase 03 is complete and ready to commit
- Active owner: none
- Last updated: 2026-08-15
- Current blocker: none
- Latest contract change: production storage codec now defaults to cidx-owned `binary`; `int8` is the only alternative, and exact encoder/scorer contracts remain a Phase 01 evidence item
- Latest evaluation change: strict Phase 02 traces use per-group first-loss observations with an explicit provider-union stage, typed operation denominators, frozen review-pass identities, ABSTAINABLE/no-answer truth, and graded durable relevance judgments
- Next eligible phase: Phase 04 — TypeScript and TSX chunker
- Exact next action: commit the accepted Phase 03 boundary, then enter Phase 04 without rerunning prior-phase validation

## Phase ledger

| Phase | Status | Owner | Prerequisite evidence checked | Last completed checkpoint | Evidence / blocker | Next action |
| --- | --- | --- | --- | --- | --- | --- |
| 00 | done | Codex | n/a; no prerequisite phases | RFC 8785 profiles and all Phase 00 catalogs/reviews completed | [Phase 00 evidence index](evidence/phase-00/README.md) | Enter Phase 01 |
| 01 | done | terra/high implementation agent; Codex validation | Yes — [Phase 00 evidence](evidence/phase-00/README.md) | Executable spikes and [Phase 01 evidence](evidence/phase-01/README.md) validated | Core, race, vet, build, runner, dependency-boundary, format, and module checks passed | Enter Phase 02 |
| 02 | done | terra/high implementation agents; Codex validation | Yes — [Phase 00](evidence/phase-00/README.md) and [Phase 01](evidence/phase-01/README.md) evidence | Formal migrations, active-vector state, secure store factories, shared chunk handoff, strict evaluation wire/schemas, and [Phase 02 evidence](evidence/phase-02/README.md) accepted | Main commit-boundary core, race, vet, build, dependency, format, module, schema, and diff checks passed | Enter Phase 03 |
| 03 | done | terra/high implementation agent; Codex validation | Yes — [Phase 02 evidence](evidence/phase-02/README.md) and `internal/chunk` shared contracts | Go Tree-sitter adapter, exact chunk/projection/segment fixtures, decision log, and [Phase 03 evidence](evidence/phase-03/README.md) accepted | Main focused test, race, vet, build, format, and diff checks passed | Enter Phase 04 |
| 04 | planned | — | No | None | Requires 02 | Wait |
| 05 | planned | — | No | None | Requires 03, 04 | Wait |
| 06 | planned | — | No | None | Requires 05 | Wait |
| 07 | planned | — | No | None | Requires 06 and user-selected corpus manifests | Wait |
| 08 | planned | — | No | None | Requires 02, 05 and explicit paid-capture approval | Wait |
| 09 | planned | — | No | None | Requires 01, 02, 05, 08 | Wait |
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
