# cidx v1 Persistent Implementation Status

This ledger is the authoritative resumable state for implementation work. Update it before starting a phase, before pausing, after context compaction is noticed, and before marking a phase complete.

## Current state

- Active phase: none
- Active owner: none
- Last updated: 2026-08-15
- Current blocker: none
- Latest contract change: production storage codec now defaults to cidx-owned `binary`; `int8` is the only alternative, and exact encoder/scorer contracts remain a Phase 01 evidence item
- Latest evaluation change: `EVALUATION-CONTRACT.md` owns stage denominators, query labels, calibration/confirmation separation, codec-fidelity metrics, RRF diagnostics, hard gates, run artifacts, shared Phase 11 body-packaging evidence, and paired Phase 14 assistant-use evidence
- Next eligible phase: Phase 01
- Exact next action: enter Phase 01 with its completion evidence and assign implementation to a terra/high agent

## Phase ledger

| Phase | Status | Owner | Prerequisite evidence checked | Last completed checkpoint | Evidence / blocker | Next action |
| --- | --- | --- | --- | --- | --- | --- |
| 00 | done | Codex | n/a; no prerequisite phases | RFC 8785 profiles and all Phase 00 catalogs/reviews completed | [Phase 00 evidence index](evidence/phase-00/README.md) | Enter Phase 01 |
| 01 | planned | — | No | None | Requires Phase 00 | Wait |
| 02 | planned | — | No | None | Requires 00, 01 | Wait |
| 03 | planned | — | No | None | Requires 02 | Wait |
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
