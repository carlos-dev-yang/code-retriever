# cidx v1 Persistent Implementation Status

This ledger is the authoritative resumable state for implementation work. Update it before starting a phase, before pausing, after context compaction is noticed, and before marking a phase complete.

## Current state

- Active phase: 11 — int8-only query scan and current evaluation surface
- Active owner: `/root`
- Phase 07 simple-control implementation owner: `/root/phase07_simple_control` (store/eval/devlab only; no corpus, provider, or production-ranking mutation)
- Last updated: 2026-08-17
- Canonical target: [`local-code-search-mcp-v1-design-r4.md`](../../local-code-search-mcp-v1-design-r4.md)
- Current blocker: Phase 07 label freeze is paused while the accepted product-owned 1024-f32 source bank, default 1024/int8 target, optional provider-free 512/int8 rematerialization, and complete Binary/256 code removal are reconciled through config, source storage, materialization, embedding, search, CLI, evaluation, and package smoke. Formal confirmation, promotion, and assistant work remain separately gated.
- Latest contract change: source and state roots are now distinct runtime inputs. Normal state is `<source>/.cidx` with production DB `.cidx/db/index.db`; cidx development evaluation uses disposable sources under `.cidx/test/corpora/` and preserved named states under `.cidx/test/states/`. Absolute source/state paths are removed from SQLite metadata.
- Latest evaluation change: after the five-profile comparison, the owner selected 1024/int8 as the ordinary/default target and 512/int8 as an explicit compact target rematerialized from preserved 1024-f32 document source rows. Binary/256 implementations are removed and only their historical documents remain.
- Evaluation execution plan: [`EVALUATION-EMBEDDING-EXECUTION-PLAN.md`](EVALUATION-EMBEDDING-EXECUTION-PLAN.md) records the provider-free preparation sequence, human-label pooling, separate document/query spend gates, raw-bank invalidation rules, calibration/confirmation freeze points, and exact next actions; it authorizes no paid operation.
- Immediate working decision: close chi/RHF calibration after runtime reconciliation with `segment_target_bytes=1024`, `source_dimensions=1024`, `serving_dimensions=1024`, and fixed `storage_codec=int8`; preserve document source f32 durably and use 512/int8 only as an explicit compact arm. Query/reference f32 remains non-serving and nonpersistent.
- Accepted cohort-design rule: representative real code-search intents take priority over filling a numeric quota. Keep an edge case only when it separates a material parser, semantic-parent, type/wrapper, codec, or retrieval failure; do not manufacture narrowly worded microcases merely to increase cohort counts. Coverage floors remain requirements for later confirmation, not targets for artificial padding.
- Measured cohort decision: keep chi G07 and RHF T01/X01/X08, add no new question, and narrow only chi G12. The recorded failures begin at top-five ranking, not corpus discovery, parsing, chunking, segmentation, raw coverage, or materialization. Full tables and the rejected score-only X08 removal advisory are recorded in [`cohort-score-review-r4.md`](evidence/phase-07/cohort-score-review-r4.md).
- Completed measured iteration: the immutable 1024-f32, 1024-binary, 1024-int8, 512-int8, and 256-int8 parent rankings are consolidated in the [five-profile cohort and answer report](evidence/phase-07/five-profile-cohort-comparison-r4.md). All five arms remain independent; the report uses no FTS or RRF and includes language/task/signal cohorts, actual answer placements, useful-source review, f32 fidelity, and complete storage measurements.
- Accepted user decision: RHF production top-level anonymous default-export
  function-like declarations receive versioned deterministic retrieval labels
  in the existing fields: `symbol=<filename stem>` and
  `qualified_symbol=module.<repository-relative path without extension>`.
  Path + indexed content hash + byte range remain source identity. No alias
  field or DB/FTS/MCP/evaluation-schema expansion is authorized. The existing
  overload grouping defect (`useWatch`, `insert`, `mockZodResolver`) is repaired
  in the same chunker-version/reindex boundary. The accepted implementation
  found 57 such production functions: 51 in previously parentless files and six
  in files with an existing type parent.
- Next eligible phase: complete Phase 11, then proceed through 12, 13, and 14 before resuming Phase 07.
- Exact next action: revalidate request-local int8 query preparation, current-profile scan, fallback/RRF/body behavior, and remove remaining retired comparison surfaces from executable evaluation code.

Existing phase completion rows and implementation are historical work produced against earlier design revisions. They must not be read as proof that the current code satisfies Revision 4; the implementation remains a prototype until it is explicitly reconciled and revalidated against the final target contract.

## Phase ledger

| Phase | Status | Owner | Prerequisite evidence checked | Last completed checkpoint | Evidence / blocker | Next action |
| --- | --- | --- | --- | --- | --- | --- |
| 00 | done | `/root` | Yes — r4 design, execution guide, evaluation contract, implementation index, historical Phase 00 evidence, five-profile comparison, and explicit user decision checked | Revision 4 catalogs plus the int8-only 512/1024 product boundary and approval-gated Binary/256 evidence boundary recorded | [Phase 00 evidence index](evidence/phase-00/README.md), [int8 profile decision](evidence/phase-00/int8-profile-retirement-r4.md), and [retired-profile contract](RETIRED-VECTOR-PROFILES.md) | Complete Phase 02 reconciliation |
| 01 | done | terra/high implementation agent; Codex validation | Yes — [Phase 00 evidence](evidence/phase-00/README.md) | Executable spikes and [Phase 01 evidence](evidence/phase-01/README.md) validated | Core, race, vet, build, runner, dependency-boundary, format, and module checks passed | Enter Phase 02 |
| 02 | done | `/root` | Yes — canonical R4, source-bank and retired-profile contracts, Phase 00/01 evidence, live config/profile/evaluation-wire code, schemas, and downstream references inspected | Default 1024/optional 512, fixed int8, source-bank impact identity, strict Binary/256 rejection, focused race/test/vet/build/schema/format/diff boundary accepted; physical source/lab split remains Phase 08 | [Current int8/source evidence](evidence/phase-02/int8-source-profile-reconciliation.md), historical [R4 config/wire evidence](evidence/phase-02/revision-4.md), and [layout evidence](evidence/phase-02/project-local-layout-reconciliation.md) | Enter Phase 05 serving-key reconciliation |
| 03 | done | terra/high implementation agent; Codex validation | Yes — [Phase 02 evidence](evidence/phase-02/README.md) and `internal/chunk` shared contracts | Go Tree-sitter adapter, exact chunk/projection/segment fixtures, decision log, and [Phase 03 evidence](evidence/phase-03/README.md) accepted | Main focused test, race, vet, build, format, and diff checks passed | Enter Phase 04 |
| 04 | done | `/root`; main-agent validation | Yes — original Phase 04 document/evidence, Phase 02/03 contracts, real chi/RHF structural audit, accepted user decision, and current workspace inspected | Revision 4 path-derived existing-field labels, overload correction, version bump, focused boundary validation, and full provider-free generation-3 handoff accepted | [Phase 04 evidence](evidence/phase-04/README.md) | Resume Phase 07 against corrected inventory |
| 05 | done | `/root` | Yes — current int8/source contract, Phase 02 reconciliation, full Phase 05 document, historical evidence, strict legacy equivalence code, publication reproof, and existing core fixtures inspected | Exact int8-equivalent legacy rows can be atomically rekeyed; canonical retired 256/Binary profiles and every unproven row remain pending; focused test/race/vet/build/format boundary passed with no source-bank/lab/provider action | [Current int8 reproof](evidence/phase-05/int8-serving-key-reproof.md) and historical [R4 evidence](evidence/phase-05/revision-4.md) | Hand active canonical inputs and current-profile pending keys to Phase 08/10 |
| 06 | done | terra/high implementation agent; Codex validation | Yes — [Phase 05 evidence](evidence/phase-05/README.md) and the store/config/symbol handoff inspected | Safe query construction, central resolved query policy/fingerprint, generation-pinned FTS/BM25 materialization with full pre-limit ordering, and [Phase 06 evidence](evidence/phase-06/README.md) accepted | Main focused race, vet, build, format, dependency-boundary, and diff checks passed | Enter Phase 07 or the unpaid implementation portion of Phase 08 |
| 07 | blocked | `/root` | Yes — full execution/index/evaluation/Phase 07 documents; Phase 06 evidence; explicit corpus/document/query authorization; tracked manifests/bindings; corrected Phase 04 and project-local Phase 02 evidence; measured artifacts and source-backed query decisions | Immutable five-profile evidence remains preserved; current evaluation is paused until 1024/int8 is the reconciled ordinary profile and Binary/256 code is absent | [Phase 07 evidence](evidence/phase-07/README.md), [five-profile comparison](evidence/phase-07/five-profile-cohort-comparison-r4.md), and [retired-profile contract](RETIRED-VECTOR-PROFILES.md) | Resume human review and label replay only after downstream profile reconciliation |
| 08 | done | `/root` | Yes — full Phase 08 document, historical R4 evidence, accepted source-bank decision, Phase 05 handoff, live lab/embedclient wiring, and current state layout inspected | Product `db/embeddings.db` owns immutable document 1024-f32; `lab/evaluation.db` contains metadata only; compatible legacy rows copy read-only; focused test/race/vet/build/format/import/schema boundary passed | [Current source-bank evidence](evidence/phase-08/int8-source-bank-reconciliation.md), historical [R4 evidence](evidence/phase-08/revision-4.md), and [source-bank decision](SOURCE-VECTOR-BANK-DECISION.md) | Hand compatible sources and missing keys to Phase 09/10 |
| 09 | done | `/root` | Yes — prior Phase 09 evidence, live vector/materialization code, Phase 08 source-bank boundary, five-profile evidence, and retired-profile contract inspected | Int8-only transform/materialization/search contract, production v5 cache, retired runtime removal, and one final offline boundary accepted | [Current evidence](evidence/phase-09/int8-only-materialization-reconciliation.md), historical [Phase 09 evidence](evidence/phase-09/README.md), and [retired-profile contract](RETIRED-VECTOR-PROFILES.md) | Hand direct int8 transform/scorer and source-bank rematerialization to Phase 10/11 |
| 10 | done | `/root` | Yes — accepted Phase 09 boundary, prior Phase 10 R4 evidence, active embedding path, source-bank decision, and retired-profile contract inspected | Source-bank-first provider success handling, compatible local reuse, public source/Voyage plan split, provider-only request accounting, and final offline boundary accepted | [Current evidence](evidence/phase-10/source-bank-first-document-publication.md), historical [R4 evidence](evidence/phase-10/revision-4.md), and [source-bank decision](SOURCE-VECTOR-BANK-DECISION.md) | Hand current int8 coverage/profile state to Phase 11 |
| 11 | in_progress | `/root` | Yes — accepted Phase 09/10 boundaries, prior Phase 11 R4 evidence, live vector scan/evaluation code, five-profile evidence, and retired-profile contract inspected | Earlier query adapter and ranking work remains historical; current runtime/evaluation surface must be int8-only with no retired comparison command | [Phase 11 R4 evidence](evidence/phase-11/revision-4.md), [Phase 10 handoff](evidence/phase-10/source-bank-first-document-publication.md), and [retired-profile contract](RETIRED-VECTOR-PROFILES.md) | Revalidate current int8 scan/fallback/RRF/body boundary |
| 12 | blocked | `/root/r4_phase12_executor` (terra/high); `/root/r4_phase12_review` (terra/high); Codex validation | Yes — full Phase 12/evaluation/execution/canonical contracts, [Phase 07](evidence/phase-07/README.md), [Phase 08 R4](evidence/phase-08/revision-4.md), [Phase 09](evidence/phase-09/README.md), [Phase 11 R4](evidence/phase-11/revision-4.md), historical [Phase 12](evidence/phase-12/README.md), `kb-guide` accounting advice, live adapter/artifact/schema, and clean workspace checked at `0258872` | Corpus-independent two-layer accounting, conservative token observability, deterministic usage wire, lab v5 preservation, independent review/remediation, and one-time main boundary validation accepted | [Phase 12 R4 accepted core evidence](evidence/phase-12/revision-4.md); official evidence still needs approved corpus/labels/raw coverage/paid query | Proceed to Phase 13; resume official evaluation only after the external inputs and separate paid approval exist |
| 13 | blocked | `/root` | Yes — prior Phase 13 R4 evidence, live init/CLI code, source-bank decision, five-profile evidence, and retired-profile contract inspected | Earlier init atomicity and four-tool proof remain accepted; public init must default to 1024/int8, allow compact 512, expose provider-free rematerialization, and remove codec selection | [Phase 13 R4 evidence](evidence/phase-13/revision-4.md), [source-bank decision](SOURCE-VECTOR-BANK-DECISION.md), and [retired-profile contract](RETIRED-VECTOR-PROFILES.md) | Reconcile after Phase 11 |
| 14 | blocked | `/root` | Yes — prior Phase 14 checkpoint, current scripts/docs, source-bank decision, five-profile evidence, and retired-profile contract inspected | The earlier local archive remains historical evidence; package smoke must be rebuilt against default 1024/int8 and provider-free 1024→512 rematerialization before it is current product proof | [Phase 14 R4 local checkpoint evidence](evidence/phase-14/revision-4.md), [source-bank decision](SOURCE-VECTOR-BANK-DECISION.md), and [retired-profile contract](RETIRED-VECTOR-PROFILES.md); release-candidate gates remain unchanged | Reconcile verifier/package defaults after Phase 13, then resume Phase 07/12 gates |

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
