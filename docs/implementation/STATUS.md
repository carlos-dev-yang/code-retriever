# cidx v1 Persistent Implementation Status

This ledger is the authoritative resumable state for implementation work. Update it before starting a phase, before pausing, after context compaction is noticed, and before marking a phase complete.

## Current state

- Active phase: none — Phase 00 Revision 4 reconciliation complete; Phase 02 is next
- Active owner: none
- Last updated: 2026-08-15
- Canonical target: [`local-code-search-mcp-v1-design-r4.md`](../../local-code-search-mcp-v1-design-r4.md)
- Current blocker: none for Revision 4 implementation reconciliation. Phase 07/12 official evidence separately requires user-selected corpus manifests, local bindings, reviewed labels, compatible raw coverage, and separate paid-query approval.
- Latest contract change: Revision 4 fixes the 1 MiB source ceiling, AST-aware 1,024-byte segment target, synchronous request grouping, three 10/20/30-second transient retries, 64 KiB inline default, byte-bounded line-cap-free `read_span`, `serving_dimensions` terminology, and FTS/binary defaults.
- Latest evaluation change: segment candidates are 768/1,024/1,536 bytes; usefulness, codec fidelity, stage loss, and operations stay separate with no weighted total or numeric v1 SLA.
- Next eligible phase: `02-config-profiles-and-schemas`.
- Exact next action: assign Phase 02 implementation to a terra/high agent, record its entry gate, and implement strict Revision 4 config/profile/evaluation-wire contracts.

Existing phase completion rows and implementation are historical work produced against earlier design revisions. They must not be read as proof that the current code satisfies Revision 4; the implementation remains a prototype until it is explicitly reconciled and revalidated against the final target contract.

## Phase ledger

| Phase | Status | Owner | Prerequisite evidence checked | Last completed checkpoint | Evidence / blocker | Next action |
| --- | --- | --- | --- | --- | --- | --- |
| 00 | done | Codex | Yes — r4 design, execution guide, evaluation contract, implementation index, historical Phase 00 evidence, and clean workspace checked | Revision 4 catalogs, compatibility policy, cross-phase review, and independently recomputed vector/storage hash fixtures completed | [Phase 00 evidence index](evidence/phase-00/README.md) and [Revision 4 supersession](evidence/revision-4/README.md) | Enter Phase 02 |
| 01 | done | terra/high implementation agent; Codex validation | Yes — [Phase 00 evidence](evidence/phase-00/README.md) | Executable spikes and [Phase 01 evidence](evidence/phase-01/README.md) validated | Core, race, vet, build, runner, dependency-boundary, format, and module checks passed | Enter Phase 02 |
| 02 | planned | — | Historical [Phase 00](evidence/phase-00/README.md), [Phase 01](evidence/phase-01/README.md), and [Phase 02](evidence/phase-02/README.md) evidence inspected; new Phase 00 R4 evidence pending | Pre-R4 implementation is historical; raw/resolved config, profiles, defaults, and evaluation wire still use removed names | [Revision 4 supersession index](evidence/revision-4/README.md) | After Phase 00, implement strict R4 config/profile/wire and legacy-shape handling |
| 03 | done | terra/high implementation agent; Codex validation | Yes — [Phase 02 evidence](evidence/phase-02/README.md) and `internal/chunk` shared contracts | Go Tree-sitter adapter, exact chunk/projection/segment fixtures, decision log, and [Phase 03 evidence](evidence/phase-03/README.md) accepted | Main focused test, race, vet, build, format, and diff checks passed | Enter Phase 04 |
| 04 | done | terra/high implementation agent; Codex validation | Yes — [Phase 02 evidence](evidence/phase-02/README.md), [Phase 03 evidence](evidence/phase-03/README.md), and `internal/chunk` shared contracts inspected | TypeScript/TSX adapters, exact callable/type projections, overload and segmentation fixtures, Phase 04 decisions, and [Phase 04 evidence](evidence/phase-04/README.md) accepted | Main focused test, race, vet, build, dependency-boundary, format, and diff checks passed | Enter Phase 05 |
| 05 | planned | — | Historical [Phase 03](evidence/phase-03/README.md), [Phase 04](evidence/phase-04/README.md), and [Phase 05](evidence/phase-05/README.md) evidence remains valid for AST/index behavior; reconciled Phase 02 pending | Pre-R4 index wiring still injects a removed chunk cap and old segment field | [Revision 4 supersession index](evidence/revision-4/README.md) | Reconcile index profile/segment injection and local reindex behavior after Phase 02 |
| 06 | done | terra/high implementation agent; Codex validation | Yes — [Phase 05 evidence](evidence/phase-05/README.md) and the store/config/symbol handoff inspected | Safe query construction, central resolved query policy/fingerprint, generation-pinned FTS/BM25 materialization with full pre-limit ordering, and [Phase 06 evidence](evidence/phase-06/README.md) accepted | Main focused race, vet, build, format, dependency-boundary, and diff checks passed | Enter Phase 07 or the unpaid implementation portion of Phase 08 |
| 07 | blocked | Codex (infrastructure completion) | Yes — [Phase 06 evidence](evidence/phase-06/README.md) and existing portable manifest/binding, dataset, metric, lexical adapter, and artifact worktree changes inspected | Production truth-inventory preflight, full lexical stage traces, reproducible-run enforcement, artifact authority, and safe whole-segment glob infrastructure implemented | [Phase 07 evidence](evidence/phase-07/README.md); official baseline requires user-selected tracked manifests and ignored local bindings | Commit this blocked infrastructure, then implement corpus-independent Phase 12; official Phase 07 evidence awaits manifests, bindings, and labels |
| 08 | planned | — | Historical [Phase 08 evidence](evidence/phase-08/README.md) inspected; reconciled Phase 02/05 pending | Raw capture remains isolated, but grouping/retry execution uses the pre-R4 token-named sequential policy | [Revision 4 supersession index](evidence/revision-4/README.md) | Adopt the shared byte-bounded synchronous request executor |
| 09 | done | terra/high implementation agent; Codex validation | Yes — [Phase 01](evidence/phase-01/README.md), [Phase 02](evidence/phase-02/README.md), [Phase 05](evidence/phase-05/README.md), and [Phase 08](evidence/phase-08/README.md) evidence reviewed | Shared transform/codecs, lab v3 staging evidence, production v2 lineage, narrow active-key planning, and complete-set atomic publication accepted | Main focused race, vet, build, migration, dependency-boundary, format, module, diff, and synthetic fidelity checks passed; no provider/corpus/paid action | Enter Phase 10 |
| 10 | planned | — | Historical [Phase 10 evidence](evidence/phase-10/README.md) inspected; reconciled Phase 05/08 pending | Plan/apply and publication behavior remain useful, but document groups retry immediately and serially under the old policy | [Revision 4 supersession index](evidence/revision-4/README.md) | Apply bounded concurrency, 10/20/30 waits, `Retry-After`, and cancellation through the shared executor |
| 11 | planned | — | Historical [Phase 06](evidence/phase-06/README.md), [Phase 09](evidence/phase-09/README.md), and [Phase 11](evidence/phase-11/README.md) core evidence inspected; reconciled Phase 10 pending | Vector/RRF/body behavior remains valid; query embedding lacks the shared Revision 4 retry/concurrency policy | [Revision 4 supersession index](evidence/revision-4/README.md) | Reconcile query embedding policy and rerun the affected fallback/body boundary |
| 12 | blocked | — | Historical corpus-independent [Phase 12 evidence](evidence/phase-12/README.md) exists; reconciled Phase 08/11 pending; official Phase 07 corpus evidence remains externally blocked | Core orchestration exists, but evaluation wire/profile terms require R4 revalidation; no official promotion evidence exists | [Revision 4 supersession index](evidence/revision-4/README.md); official evidence still needs approved corpus/labels/raw coverage/paid query | Reconcile the local adapter after Phase 11; keep official promotion blocked |
| 13 | planned | — | `b3a6cb1` contains the mixed pre-R4 CLI/MCP checkpoint and [historical Phase 13 evidence](evidence/phase-13/README.md); reconciled Phase 05/10/11 and Phase 12 core pending | Most adapters exist, but `init`, config names/defaults, request policy, and line-cap-free `read_span` are not R4 complete | [Revision 4 supersession index](evidence/revision-4/README.md) | Complete Phase 13 after the affected core phases; official Phase 12 promotion is not its entry gate |
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
