# Relation Evidence Handoff Specification

- Status: current; assistant A/B deferred
- Date: 2026-08-19
- Former title: Relation Evidence and Assistant Validation Handoff
- Owner direction: do not run assistant final-answer A/B. Current tests are
  required-parent selection, residual isolated noise, and mechanical
  packaging.
- Current experiment authority:
  [`RELATION-PACKAGING-NEXT.md`](RELATION-PACKAGING-NEXT.md)
- Current diagnostic:
  [overlap/selection v2](evidence/phase-07/relation-overlap-noise-diagnostic-r4.md)
- Accepted implementation boundary: `ba44fabac49d257323909ea118c66b9d8a053b9a`
- Stage E/F documentation commit: `3f5e8ae`
- Current product vector profile: 1,024-dimensional int8
- Current relation decision: `NO_POLICY_SELECTED_EVALUATION_ONLY`
- Review limitation: `NO_INDEPENDENT_HUMAN_REVIEW`

## 1. Purpose

This document gives another engineer enough context to reproduce the
accepted relation evidence, avoid the retired assistant-A/B path, and start
the packaging experiment.

It records:

1. what has been implemented and measured;
2. what Stage E/F proves and does not prove;
3. the overlap/selection replay that reclassified “noise”;
4. authoritative tracked and machine-local resources;
5. verification of existing artifacts;
6. the current next question;
7. the deferred assistant protocol, kept only as appendix.

Do not reopen the completed 32-case or 40-query units to improve a number.

## 2. Executive summary

Compiler/AST graph facts can make missing code reachable without changing
the protected dense top five. That is evidence availability, not a serving
policy and not an assistant-answer result.

Protected dense top five: 52/61 groups, 31/40 queries.

| Delivery form | Complete groups | Complete queries | Raw extras | Isolated extras after neighborhood class |
| --- | ---: | ---: | ---: | ---: |
| protected dense top five | 52/61 | 31/40 | — | — |
| closure 2 / 2,048 body bytes | 57/61 | 36/40 | 65 parents | 15 isolated parents; 5 isolated-noise queries remain |
| hint 4 / 4,096 serialized bytes | 58/61 | 37/40 | 133 parents | 13 isolated parents; 0 isolated-noise queries after file collapse |

The per-query 38/40 envelope is not a policy. Every Stage F cell still
misses `gg-g09-rename-change` and `me-x02-memo-editor`.

The nine baseline misses split as:

- six sibling symbols in files already returned by top five;
- three different files at dense ranks 14, 40, and 134.

Most Stage F “noisy parents” are overlapping neighborhood, not isolated
irrelevance. Organize that overlap in the payload. Omit isolated hops or
report them as limits.

Interpretation:

- relation evidence availability: demonstrated on this calibration;
- automatic relation-body push: not a default product path;
- search first-loss: mostly sibling packaging, then a few cross-file ranks;
- assistant hint use and final-answer improvement: not a current gate;
- safety rate: not established (frozen hard-negative denominator is zero);
- cross-repository generalization: not established;
- production relation path: not authorized.

## 3. Product and evaluation boundary

### 3.1 Runtime behavior

Runtime projects do not require relevance labels or AI review. Graph facts
are derived locally from source, Tree-sitter, and compiler/type-system
resolution.

The current product remains unchanged:

- serving codec: int8 only;
- default dimension: 1,024;
- optional dimension: 512, rematerialized locally from the 1,024-f32 source
  bank;
- stable MCP tools: `status`, `search`, `read_span`, and `reindex`;
- no fifth relation tool;
- no in-process LLM;
- no production import of `internal/relationdiag`;
- no relation graph in the production search/MCP/storage contract.

### 3.2 Evaluation-only behavior

The relation graph, bounded frontier, closure packages, hints, review
packets, policy-cell results, and packaging diagnostics are development
sidecars. They must not be copied into the product path merely because this
calibration is positive.

### 3.3 Closed data

Both of these sets are closed to tuning:

1. the historical 32-case chi/react-hook-form calibration;
2. the 40-query go-git/Zustand/Memos relation calibration.

They may be replayed for deterministic regression or for the frozen
packaging experiment in `RELATION-PACKAGING-NEXT.md`. They may not be used
to add cells, change questions, change labels, fit thresholds, choose
semantic margins, choose a hidden return k as policy, or claim independent
confirmation.

## 4. Completed implementation and evidence

### 4.1 Implemented graph and completion features

The evaluation implementation includes:

- compiler-resolved `CALLS`, `TYPE_REF`, and `MEMBER_OF` relation facts;
- exact source/target parent identities and source hashes;
- direction, structural tier, syntactic role, occurrence count, source-family
  count, target-incoming count, and parent traits;
- forward and reverse one-hop relation views;
- deterministic per-bucket top-two frontier and global cap;
- bounded contract-closure candidates with parent-count and body-byte caps;
- body-free relation hints with count and serialized-byte caps;
- immutable retrieval/graph/dataset/profile/query/provenance bindings;
- protected dense top-five equality checks;
- dual-AI source review, conflict adjudication, owner adoption, and frozen
  labels;
- deterministic Stage F per-query, cell, scope, and delivery aggregates.

No relation text was embedded. Stage E/F made zero Voyage calls and
persisted no query or document vector bytes in portable artifacts.

### 4.2 Code and documentation checkpoints

| Checkpoint | Meaning |
| --- | --- |
| `ba44fab` | accepted implementation and real CLI adjudication/adoption fix |
| `3f5e8ae` | Stage E/F result closed and documented |
| `docs/implementation/evidence/phase-07/relation-calibration-stage-ef-r4.md` | Stage E/F result report |
| `docs/implementation/evidence/phase-07/relation-overlap-noise-diagnostic-r4.md` | selection/noise replay v2 |
| `docs/implementation/RELATION-PACKAGING-NEXT.md` | current experiment authority |
| `docs/implementation/RELATION-EVIDENCE-COMPLETION-PLAN.md` | historical completion plan; assistant A/B in §13 is deferred |
| `docs/implementation/EVALUATION-CONTRACT.md` | normative evaluation and promotion rules |
| `docs/implementation/STATUS.md` | current phase state and exact next action |

### 4.3 Review and freeze facts

The fixed review universe contains:

- 40 queries;
- 616 parent attachments;
- 1,115 relation attachments;
- 1,000 prelabel query/cell emission rows;
- two complete independent AI review passes;
- 102 source-adjudicated grade/group conflicts;
- zero owner row overrides.

Important digests:

| Object | Digest |
| --- | --- |
| prelabel emissions | `48a91595e4c300b428eeb8f4443b4a938309bcbd68a1268df552d2cc80f82ba9` |
| prepared universe | `c686fe8c73f411709049369cad1f9be64671eb2c5092fae474fd2eb914fa1be0` |
| reconciliation | `19cc7b081e45bbac9e68b2b7b6e6e0f21293f64b6eeffc97a1e5ebadeab8f722` |
| frozen labels | `002a30b08e137467896df63f2e5da8bf176c965f06c6a164aee7fd4db565a19b` |

Accepted Stage F file hashes (A and B byte-identical):

| File | SHA-256 |
| --- | --- |
| `artifact-checksums.json` | `b07c0071d43a003545bc6e885306c421e70ef4c7aa4ddf7dde8304f0e72c8c73` |
| `cell-aggregates.jsonl` | `c9963ee6d65832779d4972afc8dcf3c0f733eb8aa34e70c7db91e40f5fe69cbb` |
| `delivery-aggregates.jsonl` | `3ad7e17e2fe653924df31cfb2f8d7447989dbec4f7444e6a5b2cb9c63504896f` |
| `per-query-cell.jsonl` | `240e2fe62ddf6ffc3eea3b356a0de2c102643d86a5f71df2a07585e188c0fe23` |
| `selection.json` | `7789a3cc30970b58f4501f9f2a687deea30a6a2fa96abe37810d110c8ec1dc12` |

### 4.4 Overlap and selection replay

Local v2 artifact SHA-256
`33a91723549c12486da93c07a638907537ff5065ad7375432109cbb19939656d`.

Missing groups are topology `source_parent_ids` absent from protected top
five. That matches the nine Stage F baseline-incomplete queries.

| Missing required parent | Rank | Same file as top five? |
| --- | ---: | --- |
| `me-x02-memo-editor` / `module.MemoEditor` | 7 | yes |
| `gg-g06-commit-object` / `git.Repository.CommitObject` | 9 | yes |
| `me-x05-ai-provider-contract` / `module.LocalAIProvider` | 10 | yes |
| `zu-t08-create-bound-contract` / `module.UseBoundStore` | 12 | yes |
| `me-x06-navigation-item` / `module.NavLinkItem` | 20 | yes |
| `me-g06-schedule-matchers` / `scheduler.fieldMatcher` | 22 | yes |
| `gg-g07-diff-header-contract` / `diff.File` | 14 | no |
| `gg-g08-topological-node` / `commitgraph.CommitNode` | 40 | no |
| `gg-g09-rename-change` / `object.Change` | 134 | no |

Topology-only dense top-10 recovers 3/9 of those misses. An earlier draft
that claimed 5/9 mixed frozen covering labels into the count and is
withdrawn.

Uncapped review-universe buckets: direct 72, support-near-gold 171, support
other 31, overlap-near-gold 120, overlap-near-primary 98, isolated 124.
Relation rows 1,115 collapse to 447 distinct targets.

## 5. Resource map

### 5.1 Read these first

1. `docs/implementation/EVALUATION-CONTRACT.md`
2. `docs/implementation/STATUS.md`
3. `docs/implementation/evidence/phase-07/relation-calibration-stage-ef-r4.md`
4. `docs/implementation/evidence/phase-07/relation-overlap-noise-diagnostic-r4.md`
5. `docs/implementation/RELATION-PACKAGING-NEXT.md`
6. this handoff

Historical only:

- `docs/implementation/RELATION-EVIDENCE-COMPLETION-PLAN.md`
- `docs/implementation/RELATION-GRAPH-EXPERIMENT-JOURNAL.md`
- `docs/implementation/RELATION-AWARE-CODE-CONTEXT-RESEARCH.md`
- `docs/implementation/07-lexical-evaluation.md`
- `docs/implementation/12-retrieval-evaluation.md`

### 5.2 Tracked portable inputs

```text
testdata/retrieval/corpora/go-git-go-git-v5.19.1.json
testdata/retrieval/corpora/pmndrs-zustand-v5.0.14.json
testdata/retrieval/corpora/usememos-memos-v0.30.0.json
testdata/retrieval/relation-calibration-go-git-v5.19.1-draft-v1.json
testdata/retrieval/relation-calibration-zustand-v5.0.14-draft-v1.json
testdata/retrieval/relation-calibration-memos-v0.30.0-draft-v1.json
testdata/retrieval/relation-calibration-review-series-v1.json
```

### 5.3 Machine-local ignored inputs

```text
.cidx/test/corpora/go-git/
.cidx/test/corpora/zustand/
.cidx/test/corpora/memos/

.cidx/test/states/go-git-1024-int8/
.cidx/test/states/zustand-1024-int8/
.cidx/test/states/memos-1024-int8/

.cidx/test/experiments/relation-calibration-review-v1/
.cidx/test/experiments/relation-overlap-noise-v1/
```

Accepted completion and graph directories are unchanged from Stage E/F:

```text
.cidx/test/states/go-git-1024-int8/evaluations/relation-completion-stage-b-go-git-v2/
.cidx/test/states/zustand-1024-int8/evaluations/relation-completion-stage-b-zustand-v2/
.cidx/test/states/memos-1024-int8/evaluations/relation-completion-stage-b-memos-v2/

.cidx/test/states/go-git-1024-int8/evaluations/relation-graph-stage-b-go-git-v4/
.cidx/test/states/zustand-1024-int8/evaluations/relation-graph-stage-b-zustand-v4/
.cidx/test/states/memos-1024-int8/evaluations/relation-graph-stage-b-memos-v4/
```

Accepted freeze and Stage F results:

```text
.cidx/test/experiments/relation-calibration-review-v1/prepared/prepared.json
.cidx/test/experiments/relation-calibration-review-v1/prepared/emissions-prelabels.json
.cidx/test/experiments/relation-calibration-review-v1/frozen-ba44/frozen.json
.cidx/test/experiments/relation-calibration-review-v1/frozen-ba44/adjudications.json
.cidx/test/experiments/relation-calibration-review-v1/frozen-ba44/owner-adoption.json
.cidx/test/experiments/relation-calibration-review-v1/stage-f-ba44-a/
.cidx/test/experiments/relation-calibration-review-v1/stage-f-ba44-b/
```

`stage-f-ba44-b/` must be byte-identical to `stage-f-ba44-a/`.

Accepted overlap replay:

```text
.cidx/test/experiments/relation-overlap-noise-v1/diagnostic.json
.cidx/test/experiments/relation-overlap-noise-v1/artifact-checksums.json
```

`prepared/prepared.json` binds query/cell emissions to parent and relation
attachment IDs. Do not infer an emission set from aggregate metrics.

### 5.4 Sensitive files

Do not read, copy, print, commit, or send `.cidx/credentials.env`. The
packaging experiment needs no Voyage request.

Frozen labels and Stage F aggregates are coordinator scoring inputs for the
packaging replay. They must not be compiled into the emitter.

## 6. Verification of existing evidence

Before any new experiment, verify without recomputing:

```sh
git rev-parse HEAD
git status --short

base=.cidx/test/experiments/relation-calibration-review-v1
diff -rq "$base/stage-f-ba44-a" "$base/stage-f-ba44-b"

shasum -a 256 \
  "$base/stage-f-ba44-a/artifact-checksums.json" \
  "$base/stage-f-ba44-a/cell-aggregates.jsonl" \
  "$base/stage-f-ba44-a/delivery-aggregates.jsonl" \
  "$base/stage-f-ba44-a/per-query-cell.jsonl" \
  "$base/stage-f-ba44-a/selection.json" \
  .cidx/test/experiments/relation-overlap-noise-v1/diagnostic.json

jq '{kind, selection_state, semantic_status}' "$base/stage-f-ba44-a/selection.json"
jq '{kind, inputs, method}' .cidx/test/experiments/relation-overlap-noise-v1/diagnostic.json
```

Expected Stage F counts are 1,000 / 1,025 / 3,534. Expected selection
fields:

```text
kind             policy_evaluation.v1
selection_state  NO_POLICY_SELECTED_EVALUATION_ONLY
semantic_status  NOT_OPENED_NO_FINITE_CELL_MANIFEST
```

Expected overlap diagnostic kind is `relation-overlap-noise-diagnostic.v2`
and SHA-256
`33a91723549c12486da93c07a638907537ff5065ad7375432109cbb19939656d`.

If a digest, line count, binding, corpus checkout, or source hash differs,
stop. Do not repair an accepted artifact in place.

Repeat the full normal/race/vet/build boundary only after code is changed.

## 7. The next unanswered question

The next experiment is specified in `RELATION-PACKAGING-NEXT.md`:

> Holding dense top-five identity and order fixed, does mechanical same-file
> sibling packaging raise required-parent selection without raising isolated
> noise, and does organized one-hop clustering add the remaining nearby
> cross-file parents without dumping isolated hops?

It is not intended to answer:

- whether an assistant’s final sentence improves;
- whether more graph metadata should be added;
- whether a different frontier cap or return k should be fit on this set;
- whether RRF, BM25, dimension, or codec should change;
- whether the product is promotion-ready.

## 8. Explicit prohibitions

The next implementation must not:

- start assistant A/B, Level 1, or Level 2;
- add a fifth MCP tool or put an LLM inside cidx;
- rescore or tune the 32-case or 40-query calibrations;
- choose a cell from the 38/40 envelope;
- add graph metadata merely to fix G09 or Memos X02;
- auto-push closure bodies as the default product path;
- change RRF, FTS, BM25, vector dimension, codec, or top-k as a hidden
  policy;
- call Voyage or read `.cidx/credentials.env`;
- modify production search/MCP/store/vector code;
- claim confirmation, `core_retrieval`, `release_candidate`, independent
  human review, or generalization.

## 9. Handoff checklist

Before starting packaging work:

- [ ] Read the files in §5.1.
- [ ] Confirm Git commit and dirty state.
- [ ] Reproduce Stage F hashes, line counts, and overlap v2 SHA-256.
- [ ] Confirm source roots and v2/v4 exact artifacts exist.
- [ ] Freeze the packaging experiment contract from
      `RELATION-PACKAGING-NEXT.md` before the first scored output.

Before deciding:

- [ ] Report paired completeness, sibling vs cross-file first-loss, isolated
      hops omitted, bytes, and omissions.
- [ ] Apply the predeclared packaging gates without tuning.
- [ ] Emit exactly one of `CONTINUE_SIBLING_PACKAGING`,
      `CONTINUE_ONE_HOP_CLUSTERS`, `CONTINUE_BOTH_FOR_ONE_COMBINED_TEST`,
      `STOP_DEFAULT_GRAPH_PUSH`, or `INCONCLUSIVE`.
- [ ] Preserve artifacts and update `STATUS.md`; do not edit production code.

## Appendix A — Deferred assistant A/B

The 2026-08-19 draft of this file specified a 120-task Level 1 assistant
A/B (dense-only, closure push, hint/pull) and a six-arm Level 2 protocol.
The owner rejected that as the current product test: final-answer quality
depends on host, model, and prompt, and cannot close those differences.

Keep the historical protocol in git history (`2bd74ed` and parents). Do not
execute it unless the owner later reopens assistant-use as a separate
Phase 14 host experiment. Reopening it still cannot substitute for
confirmation or `release_candidate`.

If that experiment is ever reopened, it remains evaluation-only, uses no
Voyage call, must freeze `experiment-contract.json` before the first
assistant response, and must not add a fifth MCP tool.
