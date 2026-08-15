# cidx v1 Persistent Implementation Status

This ledger is the authoritative resumable state for implementation work. Update it before starting a phase, before pausing, after context compaction is noticed, and before marking a phase complete.

## Current state

- Active phase: none — implementation paused by explicit user direction
- Active owner: none
- Last updated: 2026-08-15
- Canonical target: [`local-code-search-mcp-v1-design-r4.md`](../../local-code-search-mcp-v1-design-r4.md)
- Current blocker: implementation may resume only after explicit user direction and a Revision 4 reconciliation pass. Phase 07/12 official evidence additionally requires user-selected corpus manifests, local bindings, reviewed labels, compatible raw coverage, and separate paid-query approval.
- Latest contract change: Revision 4 fixes the 1 MiB source ceiling, AST-aware 1,024-byte segment target, synchronous request grouping, three 10/20/30-second transient retries, 64 KiB inline default, byte-bounded line-cap-free `read_span`, `serving_dimensions` terminology, and FTS/binary defaults.
- Latest evaluation change: segment candidates are 768/1,024/1,536 bytes; usefulness, codec fidelity, stage loss, and operations stay separate with no weighted total or numeric v1 SLA.
- Next eligible phase: none while implementation is paused.
- Exact next action after an explicit resume: reconcile central config/schema/profile names and defaults against Revision 4, identify affected migrations and evidence, then restart at the smallest invalidated phase boundary.

Existing phase completion rows and implementation are historical work produced against earlier design revisions. They must not be read as proof that the current code satisfies Revision 4; the implementation remains a prototype until it is explicitly reconciled and revalidated against the final target contract.

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
| 07 | blocked | Codex (infrastructure completion) | Yes — [Phase 06 evidence](evidence/phase-06/README.md) and existing portable manifest/binding, dataset, metric, lexical adapter, and artifact worktree changes inspected | Production truth-inventory preflight, full lexical stage traces, reproducible-run enforcement, artifact authority, and safe whole-segment glob infrastructure implemented | [Phase 07 evidence](evidence/phase-07/README.md); official baseline requires user-selected tracked manifests and ignored local bindings | Commit this blocked infrastructure, then implement corpus-independent Phase 12; official Phase 07 evidence awaits manifests, bindings, and labels |
| 08 | done | terra/high implementation agent; Codex validation | Yes — [Phase 02](evidence/phase-02/README.md) and [Phase 05](evidence/phase-05/README.md) evidence plus profile/store handoff inspected | Voyage adapter, cache-first raw capture, lab schema v2 migration, resumable failure handling, shared embed lock, and [Phase 08 evidence](evidence/phase-08/README.md) accepted | Main focused race, vet, build, format, dependency-boundary, migration, retry, rollback, and diff checks passed; live provider evidence remains NOT RUN | Enter Phase 09 |
| 09 | done | terra/high implementation agent; Codex validation | Yes — [Phase 01](evidence/phase-01/README.md), [Phase 02](evidence/phase-02/README.md), [Phase 05](evidence/phase-05/README.md), and [Phase 08](evidence/phase-08/README.md) evidence reviewed | Shared transform/codecs, lab v3 staging evidence, production v2 lineage, narrow active-key planning, and complete-set atomic publication accepted | Main focused race, vet, build, migration, dependency-boundary, format, module, diff, and synthetic fidelity checks passed; no provider/corpus/paid action | Enter Phase 10 |
| 10 | done | terra/high implementation agent; Codex validation | Yes — [Phase 05](evidence/phase-05/README.md), [Phase 08](evidence/phase-08/README.md), and [Phase 09](evidence/phase-09/README.md) evidence reviewed | Public opaque plan/apply, v2-to-v3 migration, derived active state, retry/failure accounting, guarded incremental publication, and [Phase 10 evidence](evidence/phase-10/README.md) accepted | Main focused race, vet, build, format, dependency-boundary, stale-plan, partial-batch, late-response, migration, and diff checks passed; no credentials, network, corpus, or paid action | Enter Phase 11 |
| 11 | done | terra/high implementation agent; Codex validation | Yes — [Phase 06](evidence/phase-06/README.md), [Phase 09](evidence/phase-09/README.md), and [Phase 10](evidence/phase-10/README.md) evidence and their documented store/profile handoffs reviewed | Query transform, valid-vector preflight, codec scan, segment collapse, deterministic RRF, lexical fallback isolation, rank-invariant body packaging, and [Phase 11 evidence](evidence/phase-11/README.md) accepted | Main focused race, vet, build, format, dependency-boundary, fallback, snapshot, body-budget, and diff checks passed; no real provider, credential, network, corpus, lab runtime, or paid action used. Deduplicated parent bodies remain a Phase 12 load-measurement risk, not a latency claim. | Enter corpus-independent Phase 12 implementation; Phase 07 evidence remains gated |
| 12 | blocked | Codex (historical Phase 12 core implementation) | Yes — [Phase 07](evidence/phase-07/README.md), [Phase 08](evidence/phase-08/README.md), [Phase 09](evidence/phase-09/README.md), and [Phase 11](evidence/phase-11/README.md) evidence reviewed; Phase 07 official corpus evidence remains externally blocked | Reusable arm orchestration, shared metrics, serving-f32/codec fidelity, lane/body diagnostics, and core-promotion evidence validation implemented against the earlier contract | [Phase 12 evidence](evidence/phase-12/README.md); official evidence requires user-selected manifests/bindings, reviewed labels, compatible raw coverage, and paid-query approval | Reconcile Revision 4 before running an official corpus |
| 13 | blocked | none | Phase 05/06/10/11 historical evidence exists; Revision 4 reconciliation and explicit implementation resume are required | Uncommitted prototype work is not accepted as Revision 4 completion evidence | Implementation paused; Phase 12 promotion remains externally gated | On explicit resume, reconcile Revision 4 config/MCP contracts before continuing |
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
