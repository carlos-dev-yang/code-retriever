# Relation Evidence and Assistant Validation Handoff Specification

- Status: handoff-ready; assistant validation has not been executed
- Date: 2026-08-19
- Repository documentation commit before this handoff: `3f5e8ae`
- Accepted implementation boundary: `ba44fabac49d257323909ea118c66b9d8a053b9a`
- Current product vector profile: 1,024-dimensional int8
- Current relation decision: `NO_POLICY_SELECTED_EVALUATION_ONLY`
- Review limitation: `NO_INDEPENDENT_HUMAN_REVIEW`

## 1. Purpose

This document gives another AI or engineer enough context to review the work,
reproduce the accepted evidence, and decide whether the relation graph merits
one additional assistant-use experiment.

It records:

1. what has already been implemented and measured;
2. what the completed experiment proves and does not prove;
3. the authoritative tracked and machine-local resources;
4. the exact next question to test;
5. a short go/no-go assistant A/B protocol;
6. the larger protocol required only if promotion-grade evidence is pursued;
7. artifact, blindness, scoring, safety, and stop criteria.

The immediate purpose is a design decision, not metric optimization. Do not
reopen the completed 32-case or 40-query calibration units to improve a number.

## 2. Executive summary

The completed relation work demonstrates that compiler/AST-derived graph
facts can make missing code evidence reachable without changing the protected
dense top five. It does not demonstrate that an assistant will use that
evidence or produce a better final answer.

The 40-query relation calibration has 61 required groups. The protected dense
top five completes 52 groups and 31 queries. Two representative, already
measured delivery forms produce the following diagnostic results:

| Delivery form | Complete groups | Complete queries | Main cost |
| --- | ---: | ---: | --- |
| protected dense top five | 52/61 | 31/40 | baseline |
| closure, count 2 / 2,048 body bytes | 57/61 | 36/40 | 65 parents, 32 noisy parent attachments |
| body-free hints, count 4 / 4,096 serialized bytes | 58/61 | 37/40 | 133 parents, 40 noisy parent hints |

The per-query best result across all 25 cells reaches 38/40, but that is a
post-result upper envelope and is not a valid policy. Every cell still misses:

- `gg-g09-rename-change`
- `me-x02-memo-editor`

The correct interpretation is:

- relation evidence availability: demonstrated on this calibration;
- automatic relation-body push precision: not established;
- assistant hint use and final-answer improvement: not observed;
- safety rate: not established because the final hard-negative denominator is
  zero;
- cross-repository generalization: not established;
- production relation path: not authorized.

## 3. Product and evaluation boundary

### 3.1 Runtime behavior

Runtime projects do not require relevance labels or AI review. The graph is
derived locally from source, Tree-sitter structure, and compiler/type-system
resolution. The labels in this handoff are a one-time evaluation reference.

The current product remains unchanged:

- serving codec: int8 only;
- default dimension: 1,024;
- optional dimension: 512, rematerialized locally from the 1,024-f32 source
  bank;
- stable MCP tools: `status`, `search`, `read_span`, and `reindex`;
- no fifth relation tool;
- no production import of `internal/relationdiag`;
- no relation graph in the production search/MCP/storage contract.

### 3.2 Evaluation-only behavior

The relation graph, bounded frontier, closure packages, hints, review packets,
and policy-cell results are development/evaluation sidecars. They may be read
by a development-only assistant harness. They must not be copied into the
product path merely because this calibration is positive.

### 3.3 Closed data

Both of these sets are closed to tuning:

1. the historical 32-case chi/react-hook-form calibration;
2. the 40-query go-git/Zustand/Memos relation calibration.

They may be replayed for deterministic regression or used for a bounded
assistant design A/B. They may not be used to add cells, change questions,
change labels, fit thresholds, choose semantic margins, or claim independent
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

No relation text was embedded. Stage E/F made zero Voyage calls and persisted
no query or document vector bytes in portable artifacts.

### 4.2 Code and documentation checkpoints

| Checkpoint | Meaning |
| --- | --- |
| `ba44fab` | accepted implementation and real CLI adjudication/adoption fix |
| `3f5e8ae` | Stage E/F result closed and documented |
| `docs/implementation/evidence/phase-07/relation-calibration-stage-ef-r4.md` | authoritative result report |
| `docs/implementation/RELATION-EVIDENCE-COMPLETION-PLAN.md` | original full relation/assistant plan |
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

The accepted Stage F run was repeated byte-for-byte. Its portable file hashes
are:

| File | SHA-256 |
| --- | --- |
| `artifact-checksums.json` | `b07c0071d43a003545bc6e885306c421e70ef4c7aa4ddf7dde8304f0e72c8c73` |
| `cell-aggregates.jsonl` | `c9963ee6d65832779d4972afc8dcf3c0f733eb8aa34e70c7db91e40f5fe69cbb` |
| `delivery-aggregates.jsonl` | `3ad7e17e2fe653924df31cfb2f8d7447989dbec4f7444e6a5b2cb9c63504896f` |
| `per-query-cell.jsonl` | `240e2fe62ddf6ffc3eea3b356a0de2c102643d86a5f71df2a07585e188c0fe23` |
| `selection.json` | `7789a3cc30970b58f4501f9f2a687deea30a6a2fa96abe37810d110c8ec1dc12` |

## 5. Resource map

### 5.1 Read these first

The next reviewer should read these files in order:

1. `docs/implementation/EVALUATION-CONTRACT.md`
2. `docs/implementation/RELATION-EVIDENCE-COMPLETION-PLAN.md`
3. `docs/implementation/evidence/phase-07/relation-calibration-stage-ef-r4.md`
4. `docs/implementation/STATUS.md`
5. this handoff specification

For historical reasoning, use:

- `docs/implementation/RELATION-GRAPH-EXPERIMENT-JOURNAL.md`
- `docs/implementation/RELATION-AWARE-CODE-CONTEXT-RESEARCH.md`
- `docs/implementation/07-lexical-evaluation.md`
- `docs/implementation/12-retrieval-evaluation.md`

### 5.2 Tracked portable inputs

These files are safe reproducibility metadata and belong in Git:

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

These paths contain source, SQLite state, raw review packets, and generated
evidence. They are intentionally ignored and must not be committed:

```text
.cidx/test/corpora/go-git/
.cidx/test/corpora/zustand/
.cidx/test/corpora/memos/

.cidx/test/states/go-git-1024-int8/
.cidx/test/states/zustand-1024-int8/
.cidx/test/states/memos-1024-int8/

.cidx/test/experiments/relation-calibration-review-v1/
```

Accepted completion inputs:

```text
.cidx/test/states/go-git-1024-int8/evaluations/relation-completion-stage-b-go-git-v2/
.cidx/test/states/zustand-1024-int8/evaluations/relation-completion-stage-b-zustand-v2/
.cidx/test/states/memos-1024-int8/evaluations/relation-completion-stage-b-memos-v2/
```

Accepted exact graph inputs are the physical v4 directories, not later
logically equivalent copies:

```text
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

.cidx/test/experiments/relation-calibration-review-v1/stage-f-ba44-a/selection.json
.cidx/test/experiments/relation-calibration-review-v1/stage-f-ba44-a/per-query-cell.jsonl
.cidx/test/experiments/relation-calibration-review-v1/stage-f-ba44-a/cell-aggregates.jsonl
.cidx/test/experiments/relation-calibration-review-v1/stage-f-ba44-a/delivery-aggregates.jsonl
.cidx/test/experiments/relation-calibration-review-v1/stage-f-ba44-a/artifact-checksums.json
```

`stage-f-ba44-b/` must be byte-identical to `stage-f-ba44-a/`.

`prepared/prepared.json` is the coordinator's exact attachment-identity map:
its `emissions` rows bind each query/cell to parent and relation attachment
IDs, while `candidates` and `relation_attachments` bind those IDs to source
evidence. `prepared/emissions-prelabels.json` is the immutable query/control
and byte-accounting input. Build assistant arm payloads from these accepted
objects and the checksum-bound Stage A completion artifacts; do not infer an
emission set from aggregate metrics.

### 5.4 Sensitive and non-input files

Do not read, copy, print, commit, or send `.cidx/credentials.env`. The next
short assistant A/B requires no Voyage request and no Voyage credential.

Do not give the assistant runner any of these answer-bearing resources:

- `frozen-ba44/frozen.json`;
- `pass-1.json` or `pass-2.json`;
- adjudication packets or results;
- Stage F cell/delivery aggregates;
- the result sections of the Stage E/F evidence report.

Those resources belong only to the coordinator and blinded judge after all
assistant outputs have been sealed.

## 6. Verification of existing evidence

Before any new experiment, verify the current state without recomputing or
changing it:

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
  "$base/stage-f-ba44-a/selection.json"

jq . "$base/stage-f-ba44-a/selection.json"
wc -l \
  "$base/stage-f-ba44-a/per-query-cell.jsonl" \
  "$base/stage-f-ba44-a/cell-aggregates.jsonl" \
  "$base/stage-f-ba44-a/delivery-aggregates.jsonl"
```

Expected counts are 1,000, 1,025, and 3,534. Expected selection fields are:

```text
kind             policy_evaluation.v1
selection_state  NO_POLICY_SELECTED_EVALUATION_ONLY
semantic_status  NOT_OPENED_NO_FINITE_CELL_MANIFEST
```

If a digest, line count, binding, corpus checkout, or source hash differs,
stop. Do not repair an accepted artifact in place.

The accepted implementation boundary already passed full normal/race tests,
vet, build, module verification, formatting, dependency checks, four-tool MCP
checks, and runtime FTS5/WAL/language probes. A new AI does not need to repeat
that full boundary merely to review this design. Repeat it only after code is
changed.

## 7. The next unanswered question

The next experiment must answer exactly this:

> When the primary retrieval result is held fixed, does relation evidence help
> an assistant produce a more complete and correct answer, and is body-free
> hint/pull behavior better than automatic closure-body push?

It is not intended to answer:

- whether more relation metadata should be added;
- whether a different frontier cap would score better;
- whether RRF, BM25, HNSW, dimension, or codec should change;
- whether semantic thresholds should be fit;
- whether the old questions or labels should be revised;
- whether the product is promotion-ready.

## 8. Two validation levels

### 8.1 Level 1: short design go/no-go

Use this level if the owner only needs to decide whether relation product work
is worth continuing. It uses three arms and the existing 40 queries, for 120
independent assistant tasks.

| Arm | Input difference | Purpose |
| --- | --- | --- |
| A — baseline | the exact protected primary results only | current reference |
| B — closure | A plus the precomputed count-2 / 2,048-body-byte closure payload | test automatic body push |
| C — hint/pull | A plus the precomputed count-4 / 4,096-byte body-free hints; existing `read_span` remains the actuator | test agent-selected expansion |

These two cells are calibration behavior probes, not selected product policy.
They were chosen as compact representative delivery forms whose complete
denominators and costs are already recorded. Their use in Level 1 cannot be
reported as confirmation.

Do not add the closure-plus-hints arm unless Level 1 shows that both B and C
independently add final-answer value. Do not add a fourth arm merely to search
for a positive result.

### 8.2 Level 2: canonical full assistant evidence

This level is required only if the project seeks full Phase 14 assistant
evidence. It follows `RELATION-EVIDENCE-COMPLETION-PLAN.md` and includes all six
arms:

1. existing assistant tools only;
2. existing tools plus lexical cidx;
3. existing tools plus unchanged hybrid cidx;
4. existing tools plus bounded closure;
5. existing tools plus body-free hints and `read_span`;
6. existing tools plus closure and hints.

Level 1 cannot substitute for Level 2 in a `release_candidate` claim.

## 9. Level 1 execution contract

### 9.1 Freeze before execution

Write an `experiment-contract.json` before the first assistant response. It
must record:

- schema and experiment version;
- Git commit and dirty state;
- every input artifact path and SHA-256;
- exact query IDs and order;
- exact arm definitions and payload digests;
- assistant provider, model ID/version, host, and account class;
- system prompt and user-prompt template digests;
- tool names and schema digests;
- context window, input/output token caps, timeout, and retry policy;
- temperature, seed, and reasoning setting when the provider exposes them;
- repetition count;
- task-order randomization seed;
- spend and time ceiling;
- judge model families and blindness rules;
- the decision gates in Section 12.

Do not start if any field is left as “decide later.” Operational values that
depend on the selected assistant model must be filled in before results.

### 9.2 Isolation and blindness

- Run each query/arm in a fresh conversation with no memory of another arm.
- Randomize arm order per query and hide human-readable arm labels from the
  runner.
- The runner sees the question, fixed primary results, its arm payload, and
  the same tool instructions. It never sees labels, grades, aggregates, other
  arm outputs, or prior reviews.
- The coordinator may see arm identity but must not grade the final answer.
- The blind judge sees the question, source-backed expected requirements,
  final answer, citations, and tool transcript, but not the arm identity.
- Seal all 120 outputs before opening group grades or aggregate results.
- Do not tune prompts, caps, timeouts, or tool instructions after any output is
  inspected.

### 9.3 Fixed retrieval and graph payload

The primary result identity and order must be identical in A, B, and C.
Relation evidence may only add context or affordances; it may not rerank or
remove primary results.

Arm B uses only complete source bodies already admitted by the fixed closure
cell. Enforce both count and body-byte caps.

Arm C presents only body-free hint fields already frozen in the hint artifact:

- relation kind, direction, tier, and syntactic role;
- source occurrence identity;
- target path, hash, line/range, symbol, and qualified symbol;
- occurrence counts or declared compact structural metadata;
- deterministic ordinal and omission status.

Do not expose raw vector scores as confidence, relevance labels, grades,
hard-negative labels, source bodies, or answer-group IDs. The assistant may
open an exact target only through the existing `read_span` behavior.

### 9.4 Tools

Every arm receives the same tool surface. Do not force cidx or force a relation
expansion. `reindex` must not be used during the experiment because it mutates
the frozen state. Record attempted and completed calls separately.

No new `expand_relations`, `find_callers`, or `find_callees` tool is permitted.

### 9.5 One observation row per task

Record at least:

```text
query_id
opaque_arm_id
task_order
model/provider/version
prompt_digest
tool_schema_digest
primary_result_digest
relation_payload_digest
started_at / completed_at / latency_ms
final_answer
claimed files, symbols, ranges, and requirements
all tool calls and outcomes
hints displayed, inspected, followed, or ignored
read_span calls and returned source bytes
correct-edge and wrong-edge expansions
hard-negative or misleading-edge following
input/output/cached/reasoning tokens when available
provider-reported cost and local estimated cost
failure, timeout, retry, or cancellation state
```

Do not omit failed or timed-out tasks from denominators.

## 10. Grading contract

### 10.1 Mechanical checks first

Before AI judgment, verify:

- cited file/hash/range exists in the frozen corpus;
- cited body matches the indexed source hash;
- every `read_span` range is in bounds;
- the transcript contains no undeclared tool or hidden source access;
- primary result identity/order is equal across arms;
- relation payload respects its count and byte cap;
- all 120 task rows exist exactly once;
- no output was produced after the contract changed.

### 10.2 Blind final-answer review

Use the frozen required groups as the source-backed answer reference. Judge
each final answer on:

- complete satisfaction of every required group;
- correct file/symbol/range claims;
- unsupported or false claims;
- whether a relation expansion supplied a genuinely necessary fact;
- whether an irrelevant relation caused a false lead;
- whether the assistant correctly abstained when the available evidence was
  insufficient.

Use two independent AI reviewer families for ambiguous answer judgments. Both
must inspect source evidence, not search scores. Reconcile disagreements before
unblinding arm identities. Preserve the permanent
`NO_INDEPENDENT_HUMAN_REVIEW` limitation.

### 10.3 Required reports

Report raw paired outcomes; do not create a weighted quality score.

At minimum report:

- complete tasks and complete required groups per arm;
- paired A-loss-to-B/C-win and A-win-to-B/C-loss transitions;
- correct and wrong relation expansions;
- hint inspection and hint-follow rates;
- hard-negative/misleading-edge follows;
- time and tool calls to first useful evidence;
- `read_span` calls and source bytes;
- total context/tool bytes;
- total tool calls, latency, failures, tokens, and cost;
- results by corpus, language, relation-challenge, and naturalistic cohorts;
- assistant first-loss attribution.

## 11. Recommended artifact layout

Keep the next experiment in ignored local state:

```text
.cidx/test/experiments/relation-assistant-ab-v1/
  experiment-contract.json
  task-order.json
  arm-inputs/
  raw-outputs/
  assistant-observations.jsonl
  mechanical-validation.json
  blind-judge-packet.json
  judge-pass-1.json
  judge-pass-2.json
  reconciliation.json
  per-query-results.jsonl
  cohort-language-report.json
  resource-report.json
  decision.json
  report.md
  artifact-checksums.json
```

Portable reports must contain no credentials, absolute paths, source bodies,
query/document vectors, or private provider response metadata.

## 12. Level 1 decision gates

These are practical design-continuation gates, not promotion gates.

### 12.1 Hint/pull continuation

Return `CONTINUE_HINT_PULL` only if all are true:

1. C converts at least three of the six currently evidence-reachable baseline
   misses into complete final answers;
2. C causes zero graph-induced regression among baseline-complete tasks;
3. C causes zero verified hard-negative or misleading-edge final-answer error;
4. correct relation expansions outnumber wrong expansions;
5. the improvement is not produced by undeclared source access;
6. all failures, bytes, tokens, latency, and cost remain reported.

The “three of six” threshold is a calibration go/no-go rule: it requires the
assistant to realize at least half of the evidence availability already
demonstrated by the hint arm. It is not a generalization or release margin.

### 12.2 Closure continuation

Return `CONTINUE_CLOSURE` only if all are true:

1. B converts at least three of the five closure-reachable baseline misses into
   complete final answers;
2. B causes zero graph-induced regression among baseline-complete tasks;
3. B causes zero hard-negative or misleading-body final-answer error;
4. its context cost and false-lead behavior are not worse than C without a
   compensating final-answer gain.

### 12.3 Stop or inconclusive

Return `STOP_PRODUCT_GRAPH` when neither B nor C passes its continuation gate.
Preserve the evaluation sidecar and all evidence, but do not integrate the
graph into production search or MCP.

Return `INCONCLUSIVE` rather than tuning when:

- runner failures or model instability invalidate paired comparison;
- the experiment contract changed after output exposure;
- primary results differ across arms;
- source or artifact bindings fail;
- judge blindness is broken;
- result differences depend on an undeclared tool or external context.

An inconclusive result requires a newly frozen repeat contract. It does not
authorize changing questions, labels, graph metadata, or caps on the current
run.

## 13. What happens after Level 1

### 13.1 If hint/pull passes

Freeze one exact hint/pull candidate contract. Keep the four-tool MCP surface;
use a dev-only adapter until a separate product wire design is approved. Then
prepare a distinct unexposed confirmation unit.

### 13.2 If closure passes and hint/pull does not

Freeze only the mechanically defined low-entropy closure class and its count
and body-byte caps. Do not generalize closure to callers, callees, tests, or
usage examples without new evidence.

### 13.3 If both pass

Run the combined arm once under a separately frozen contract. Adopt a combined
path only when it improves final tasks over both isolated arms, not merely
because it exposes more evidence.

### 13.4 If neither passes

Stop relation product work. Keep the complete graph only as evaluation and
future research evidence. The existing int8 retrieval and four-tool MCP remain
the product.

## 14. Later unexposed confirmation

Do not begin confirmation before an exact policy or explicit no-policy
contract is frozen.

The normative floors are:

- at least 90 answerable queries, including at least 30 each for Go,
  TypeScript, and TSX;
- at least 18 verified `ABSTAINABLE` or hard-negative queries, including at
  least six per language;
- at least ten cases in every critical cohort;
- genuine mixed-language cases.

These denominators may overlap; they do not mechanically require 108 unique
questions. Do not pad the set with narrow edge cases. Repository selection,
clone/download, new document embedding, pool-query embedding, formal query
execution, assistant host/model/spend, and production-contract changes each
remain separately controlled actions.

Confirmation labels use the same owner-adopted dual-AI protocol and permanent
`NO_INDEPENDENT_HUMAN_REVIEW` limitation. Confirmation may evaluate the frozen
contract once and may not tune it.

## 15. Explicit prohibitions

The next AI must not:

- rescore or tune the 32-case or 40-query calibration;
- choose a cell from the 38/40 per-query upper envelope;
- add graph metadata merely to fix G09 or Memos X02;
- change questions or labels after seeing assistant outputs;
- fit raw int8 similarity ratios or new semantic thresholds;
- change RRF, FTS, BM25, vector dimension, codec, or top-k in this experiment;
- call Voyage or re-embed documents/queries for Level 1;
- read `.cidx/credentials.env`;
- send labels, grades, or Stage F results to the assistant runner;
- add a fifth MCP tool;
- modify production search/MCP/store/vector code;
- claim confirmation, `core_retrieval`, `release_candidate`, independent human
  review, or generalization from Level 1.

## 16. Handoff checklist

Before starting:

- [ ] Read the five authority/resources in Section 5.1.
- [ ] Confirm Git commit and clean/dirty state.
- [ ] Reproduce all accepted Stage F hashes and line counts.
- [ ] Confirm source roots and v2/v4 exact artifacts exist.
- [ ] Choose Level 1 or Level 2 explicitly.
- [ ] Select and record assistant model, host, account, and spend ceiling.
- [ ] Freeze prompts, tools, budgets, randomization, repetitions, and judges.
- [ ] Generate arm payloads without reading labels.
- [ ] Check identical primary result identity/order across arms.

Before scoring:

- [ ] Seal every assistant output and transcript.
- [ ] Mechanically validate all paths, hashes, ranges, caps, and denominators.
- [ ] Build a blinded judge packet with no arm identity.
- [ ] Complete and reconcile independent answer judgments.

Before deciding:

- [ ] Unblind only after reconciliation.
- [ ] Report paired wins and losses, not only aggregate accuracy.
- [ ] Report wrong-edge and hard-negative behavior.
- [ ] Report bytes, tokens, latency, failures, and cost.
- [ ] Apply the predeclared decision gates without tuning.
- [ ] Emit exactly one of `CONTINUE_HINT_PULL`, `CONTINUE_CLOSURE`,
  `CONTINUE_BOTH_FOR_ONE_COMBINED_TEST`, `STOP_PRODUCT_GRAPH`, or
  `INCONCLUSIVE`.
- [ ] Preserve artifacts and update `STATUS.md`; do not edit production code.

## 17. Current recommended scope

For a fast decision, run Level 1 only. It requires 120 assistant tasks and no
new embedding or corpus operation. If Level 1 does not produce a clear final
answer gain, stop. Do not spend time on further graph policy micro-tuning.

Proceed to Level 2 and unexposed confirmation only after Level 1 demonstrates
that an assistant can convert the already-proved evidence availability into
correct final answers with acceptable false-lead and resource behavior.
