# cidx v1 Persistent Implementation Status

This ledger is the authoritative resumable state for implementation work. Update it before starting a phase, before pausing, after context compaction is noticed, and before marking a phase complete.

## Current state

- Active phase: 07 — frozen chi/RHF calibration checkpoint and unexposed confirmation handoff
- Active owner: `/root`
- Completed bounded follow-up: the development-only, LLM-free relation-context
  metadata diagnostic and its conditional provider-free graph-first crossover
  are complete at clean commit `c197cdafa93852df2c1463d2636378caae288130`.
- Completed measured follow-up: X08 exposed a general missing structural label, not
  a one-off name-collision exception. All six reviewed public RHF React
  components use an explicit `*Props` value-parameter type contract, while the
  v2 sidecar records every one as generic `TYPE_LOCAL`. The reviewed v3
  implementation now classifies value-parameter type annotations mechanically,
  preserves the existing policies and questions, and adds one new
  calibration-only selector. The clean provider-free run classified the common
  six-component pattern correctly but preserved `31/32` and left X08 at
  `RELATION_ADMISSION`, with zero hard-negative/`walkXFF` attachments. The owner
  deferred the policy decision. This diagnostic cannot become confirmation or
  promotion evidence.
- Completed Phase 07 relation-graph diagnostic owner: `/root` (development-only sidecar; production integration rejected at the measured boundary)
- Phase 07 simple-control implementation owner: `/root/phase07_simple_control` (store/eval/devlab only; no corpus, provider, or production-ranking mutation)
- Last updated: 2026-08-18
- Canonical target: [`local-code-search-mcp-v1-design-r4.md`](../../local-code-search-mcp-v1-design-r4.md)
- Current blocker: the 32-case chi/RHF calibration set is frozen and replayed under `OWNER_ADOPTED_DUAL_AI_REVIEW`; Phase 07 completion and official promotion still require a separate unexposed confirmation set at the contract floors. Mixed-language, assistant, and release-candidate evidence remain later gates.
- Latest contract change: source and state roots are now distinct runtime inputs. Normal state is `<source>/.cidx` with production DB `.cidx/db/index.db`; cidx development evaluation uses disposable sources under `.cidx/test/corpora/` and preserved named states under `.cidx/test/states/`. Absolute source/state paths are removed from SQLite metadata.
- Latest evaluation change: AST/compiler edge metadata recovered chi G09 and
  raised complete related evidence from `30/32` to `31/32`, but RHF X08 remains
  `RELATION_ADMISSION`. The conditional graph-first crossover added no complete
  answer and attached protected `middleware.walkXFF` evidence to chi G05, so
  its chi arm is ineligible. Both selectors are rejected for production; the
  metadata sidecar remains development evidence.
- Completed metadata boundary: [relation-edge metadata evidence](evidence/phase-07/relation-edge-metadata-diagnostic-r4.md)
  records the clean chi/RHF v2 graphs, dense-first and graph-first runs, exact
  hashes, unchanged primary top five, zero Voyage operations, G09/X08 traces,
  the graph-first safety failure, and Terra `CLEAR`.
- Completed value-parameter measurement: [value-parameter evidence](evidence/phase-07/relation-value-parameter-diagnostic-r4.md)
  records clean commit `7879ab7315bd215fab34d5756b6416158b6c382d`, the
  six common RHF contracts, unchanged `31/32` completion, X08's remaining
  admission loss, zero provider operations, exact hashes, and the deferred
  policy boundary.
- Completed diagnostic boundary: [relation-graph evidence](evidence/phase-07/relation-usage-graph-diagnostic-r4.md) records clean commit `02834052921116a6341c44d7f7fd7e51f6a87005`, exact graph/probe/run hashes, zero provider calls, unchanged top five, and Terra `CLEAR`. It authorizes no production schema, MCP, ranking, FTS/RRF, query/label, or provider change.
- Evaluation execution plan: [`EVALUATION-EMBEDDING-EXECUTION-PLAN.md`](EVALUATION-EMBEDDING-EXECUTION-PLAN.md) records the provider-free preparation sequence, dual-AI blind pooling, separate document/query spend gates, source-bank invalidation rules, calibration/confirmation freeze points, and exact next actions; it authorizes no paid operation.
- Final corpus-independent review: [`int8-source-profile-final-review.md`](evidence/revision-4/int8-source-profile-final-review.md) maps the committed config-through-package implementation to the current product decision and records no remaining implementation finding.
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
- Next eligible phase: use the closed Phase 07 calibration checkpoint for bounded Phase 12 calibration analysis while Phase 07 prepares a separate unexposed confirmation set.
- Exact next action: leave the value-parameter selector decision deferred and
  make no further X08-specific key or question change. Return to the separate
  unexposed confirmation set required by Phase 07/12 unless the owner explicitly
  resumes this policy decision.

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
| 07 | in_progress | `/root` | Yes — full execution/index/evaluation/Phase 07 documents; Phase 06 evidence; explicit corpus/document/query authorization; tracked manifests/bindings; corrected Phase 04 and project-local Phase 02 evidence; frozen calibration replay; accepted relation sidecar plus metadata/value-parameter diagnostics | v3 classified the general 6/6 RHF value-parameter props-contract pattern, but complete evidence remains `31/32` and X08 remains admission-limited; the owner deferred the selector decision; confirmation floors remain outstanding | [Frozen calibration checkpoint](evidence/phase-07/dual-ai-calibration-freeze-r4.md), [relation metadata diagnostic](evidence/phase-07/relation-edge-metadata-diagnostic-r4.md), [value-parameter diagnostic](evidence/phase-07/relation-value-parameter-diagnostic-r4.md), [relation-context research](RELATION-AWARE-CODE-CONTEXT-RESEARCH.md), [Phase 07 evidence](evidence/phase-07/README.md), and [retired-profile contract](RETIRED-VECTOR-PROFILES.md) | Keep the selector decision deferred and return to the unexposed confirmation set unless the owner explicitly resumes it |
| 08 | done | `/root` | Yes — full Phase 08 document, historical R4 evidence, accepted source-bank decision, Phase 05 handoff, live lab/embedclient wiring, and current state layout inspected | Product `db/embeddings.db` owns immutable document 1024-f32; `lab/evaluation.db` contains metadata only; compatible legacy rows copy read-only; focused test/race/vet/build/format/import/schema boundary passed | [Current source-bank evidence](evidence/phase-08/int8-source-bank-reconciliation.md), historical [R4 evidence](evidence/phase-08/revision-4.md), and [source-bank decision](SOURCE-VECTOR-BANK-DECISION.md) | Hand compatible sources and missing keys to Phase 09/10 |
| 09 | done | `/root` | Yes — prior Phase 09 evidence, live vector/materialization code, Phase 08 source-bank boundary, five-profile evidence, and retired-profile contract inspected | Int8-only transform/materialization/search contract, production v5 cache, retired runtime removal, and one final offline boundary accepted | [Current evidence](evidence/phase-09/int8-only-materialization-reconciliation.md), historical [Phase 09 evidence](evidence/phase-09/README.md), and [retired-profile contract](RETIRED-VECTOR-PROFILES.md) | Hand direct int8 transform/scorer and source-bank rematerialization to Phase 10/11 |
| 10 | done | `/root` | Yes — accepted Phase 09 boundary, prior Phase 10 R4 evidence, active embedding path, source-bank decision, and retired-profile contract inspected | Source-bank-first provider success handling, compatible local reuse, public source/Voyage plan split, provider-only request accounting, and final offline boundary accepted | [Current evidence](evidence/phase-10/source-bank-first-document-publication.md), historical [R4 evidence](evidence/phase-10/revision-4.md), and [source-bank decision](SOURCE-VECTOR-BANK-DECISION.md) | Hand current int8 coverage/profile state to Phase 11 |
| 11 | done | `/root` | Yes — accepted Phase 09/10 boundaries, prior Phase 11 R4 evidence, live vector scan/evaluation code, five-profile evidence, and retired-profile contract inspected | Current request-local int8 scan, nonpersistent serving-f32 reference, fallback/RRF/body behavior, retired comparison removal, and focused boundary accepted | [Current evidence](evidence/phase-11/int8-only-query-search-reconciliation.md), historical [R4 evidence](evidence/phase-11/revision-4.md), and [retired-profile contract](RETIRED-VECTOR-PROFILES.md) | Hand the current evaluation arms to Phase 12 |
| 12 | blocked | `/root` | Yes — accepted current Phase 11 boundary, prior Phase 12 R4 evidence, evaluation contract, current schemas/adapters, frozen calibration labels, and retired-profile contract inspected | Corpus-independent adapter and the 32-case calibration replay are accepted; official `core_retrieval` evaluation still lacks an unexposed promotion-capable confirmation set and later assistant inputs | [Current evidence](evidence/phase-12/int8-only-evaluation-reconciliation.md), [frozen calibration](evidence/phase-07/dual-ai-calibration-freeze-r4.md), and [R4 accounting evidence](evidence/phase-12/revision-4.md) | Use calibration for bounded analysis; run official confirmation only after its independent freeze |
| 13 | done | `/root` | Yes — current Phase 02/08/11/12 boundaries, prior Phase 13 R4 evidence, live init/CLI/MCP code, source-bank decision, and retired-profile contract inspected | Default 1024/optional 512 fixed-int8 CLI, provider-free source reuse, four-tool MCP, and focused offline boundary accepted | [Current evidence](evidence/phase-13/int8-only-cli-mcp-reconciliation.md), historical [R4 evidence](evidence/phase-13/revision-4.md), and [retired-profile contract](RETIRED-VECTOR-PROFILES.md) | Hand the current CLI/MCP surface to Phase 14 |
| 14 | blocked | `/root` | Yes — accepted current Phase 13 boundary, prior Phase 14 checkpoint, current scripts/docs, source-bank decision, five-profile evidence, and retired-profile contract inspected | Clean-provenance local darwin/arm64 archive proves default 1024/int8, provider-free compact 512/int8, negative-only Binary/256, and source-bank-free four-tool serving; official promotion scope remains externally gated | [Current int8 package evidence](evidence/phase-14/int8-profile-package-reconciliation.md), historical [R4 checkpoint](evidence/phase-14/revision-4.md), [source-bank decision](SOURCE-VECTOR-BANK-DECISION.md), and [retired-profile contract](RETIRED-VECTOR-PROFILES.md) | Resume after official Phase 12 core evidence and frozen assistant/host inputs exist |

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
