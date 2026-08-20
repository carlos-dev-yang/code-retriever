# Critical/General Question-Set v2 Diagnostic

- Date: 2026-08-20
- Code commit: `252c168b1385357fe106b1589bdd3e5a8428d98a`
- Repositories: existing approved chi v5.3.1 and React Hook Form v7.85.0 checkouts only
- Provider operations: zero
- Evidence class: provider-free calibration diagnostic; not promotion evidence

> **2026-08-20 interpretation update:** the 32 semantic/mixed zero-candidate
> outcomes diagnose the planner used by these retained runs: global
> all-token-`AND`. They do
> not prove that FTS cannot contribute to natural-language retrieval. The
> adopted remediation and revised scorecard are in
> [Natural-Language FTS Query-Planner Review — Revision 4](natural-language-fts-query-planner-review-r4.md).
> All question and run results below remain unchanged.

> **2026-08-20 rerun outcome:** planner v2 was executed as new immutable runs
> against these unchanged questions. CompleteRequirementHit@5 improved from
> `10/44` to `30/44`, all 32 prior semantic/mixed candidate-zero outcomes were
> eliminated, and no historical FTS pass regressed. At candidate depth 20,
> `38/44` queries contain every required group; eight of those are displaced
> beyond the returned top five. Exact lineage and stage diagnostics are in
> [the planner v2 rerun report](natural-language-lexical-rerun-v2.md). The
> historical tables below remain the retained baseline rather than being
> rewritten.

## 1. Version and provenance contract

The earlier behavior and exact-identifier question files remain unchanged.
Each corpus now has a new v2 question set that records its predecessors,
change summary, `critical-general-v1` taxonomy identity, and canonical digest.
Every new FTS/simple run records the explicit question-set ID, version,
canonical digest, taxonomy version, and taxonomy digest in `manifest.question_set`.

The machine-readable mapping from versions to run IDs and artifact digests is
[`question-set-run-registry-v1.json`](../../../../testdata/retrieval/question-set-run-registry-v1.json).

| Corpus | v2 cases | Canonical question-set SHA-256 |
| --- | ---: | --- |
| chi | 18 | `b0818e5b52f036ecb9bf1b71e88e9f057b54ac6c55280da6079ee8650a89fee4` |
| React Hook Form | 26 | `e43aba2a6f5453757d271f87a121835aec83c02b7429c7bf864c81a1a9ad1d1b` |

The new sets contain the prior 32 behavior questions plus the prior 12
exact-identifier questions. Query text and source truth were copied from the
preserved versions. The new draft revision changes set membership and cohort
classification; it does not rewrite earlier results.

## 2. Taxonomy

Every answerable question has one critical signal cohort:

- `critical:lexical_anchor`;
- `critical:semantic_only`;
- `critical:mixed_signal`.

Applicable risk cohorts are `critical:multi_requirement`,
`critical:contract_disambiguation`, and `critical:known_hard_negative`.
Each question also has one `general:task:*` cohort and optional
`general:diag:*` cohorts. Language remains a separate slice.

Critical means a later release candidate cannot hide a failed cohort behind a
global average. This diagnostic does not freeze numeric release thresholds.
Operational integrity remains a separate 100% gate.

## 3. Execution

The retained states initially held an older index-profile fingerprint. A
provider-free local reindex moved both states from generation 1 to generation
2 while preserving their source manifests:

| Corpus | Files | Chunks | Segments | Generation | Manifest |
| --- | ---: | ---: | ---: | ---: | --- |
| chi | 78 | 452 | 621 | 2 | `6bd4db89…302a29` |
| React Hook Form | 237 | 322 | 492 | 2 | `54f6b138…f88b8a` |

Four new immutable runs then completed with `VOYAGE_API_KEY` absent:

- `chi-critical-general-v2-fts-2`;
- `chi-critical-general-v2-simple-2`;
- `rhf-critical-general-v2-fts-2`;
- `rhf-critical-general-v2-simple-2`.

All 88 query-arm operations completed without an execution failure.

## 4. Results

Primary value below is `CompleteRequirementHit@5`.

| Critical cohort | Cases | FTS | Simple control |
| --- | ---: | ---: | ---: |
| lexical anchor | 12 | **10/12 (83.3%)** | **10/12 (83.3%)** |
| semantic only | 24 | **0/24 (0%)** | **13/24 (54.2%)** |
| mixed signal | 8 | **0/8 (0%)** | **5/8 (62.5%)** |
| multi requirement | 4 | **0/4 (0%)** | **2/4 (50.0%)** |
| contract disambiguation | 5 | **2/5 (40.0%)** | **1/5 (20.0%)** |

The one reviewed hard-negative case had `KnownHardNegativeHit@5=0` in both
arms. Its answer requirement was not found; safety and answer retrieval are
reported separately.

| Arm | Overall | Go | TypeScript | TSX |
| --- | ---: | ---: | ---: | ---: |
| FTS | **10/44 (22.7%)** | 6/18 (33.3%) | 2/16 (12.5%) | 2/10 (20.0%) |
| Simple control | **28/44 (63.6%)** | 9/18 (50.0%) | 11/16 (68.8%) | 8/10 (80.0%) |

General task cohorts remain report-only:

| General task cohort | Cases | FTS | Simple control |
| --- | ---: | ---: | ---: |
| delegated or cross-parent flow | 12 | 0/12 (0%) | 7/12 (58.3%) |
| interface/type/API contract | 5 | 2/5 (40.0%) | 2/5 (40.0%) |
| lifecycle/state/error/configuration | 8 | 0/8 (0%) | 5/8 (62.5%) |
| single-parent behavior | 19 | 8/19 (42.1%) | 14/19 (73.7%) |

## 5. Interpretation

The taxonomy is useful. The global FTS result alone looks uniformly weak, but
the critical split shows that the current planner works for many direct code
anchors and fails to admit every semantic and mixed question. Because those 32
queries returned zero candidates, this run cannot measure attainable BM25
ranking quality for them. The simple control recovers many semantic and mixed
questions, yet remains weak on contract disambiguation and only partially
covers multi-requirement work. These are materially different failure modes
that a single aggregate would hide.

This is not a search-quality improvement claim: no retrieval algorithm or
ranking policy changed. The improvement is in evaluation coverage and
diagnostic precision. The result requires candidate admission and ranking to
be measured separately. It supports keeping FTS as an independent lexical lane
beside symbol, path, and dense candidates, not using it as a mandatory dense
prefilter or expecting the current all-token query to answer agent prose.

The question set is still a draft calibration diagnostic. Critical cohort
sizes are uneven, the hard-negative denominator is one, and no new current-v2
dense/hybrid run was made because that would require explicit paid query
embedding approval. Consequently this evidence cannot choose release
thresholds or prove the MCP's assistant usefulness.

## 6. Handoff

Phase 06 natural-language lexical remediation and the owner-directed
provider-free rerun are complete. The final run binds planner version 2 and its
serving-policy identity and records symbol, path, descriptive FTS, local-union
admission, and final rank. Paired assistant A/B over the same repositories is
the next separate evaluation. It must create new immutable artifacts rather
than modify these questions or any prior result.
