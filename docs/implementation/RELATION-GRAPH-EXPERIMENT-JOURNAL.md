# cidx Relation-Graph Experiment Journal and Review Dossier

- Journal status: complete record of the exposed chi/RHF calibration series
- Recorded on: 2026-08-18
- Source-tree baseline: `4af2a57` (`docs(eval): record graph-only Pareto diagnostic`)
- Phase: 07, still `in_progress`
- Product retrieval baseline: exhaustive `1024/int8` dense retrieval, with FTS
  retained as an independent lexical control
- Evaluation authority: `OWNER_ADOPTED_DUAL_AI_REVIEW`
- Permanent limitation: `NO_INDEPENDENT_HUMAN_REVIEW`
- Promotion status: calibration evidence only; not confirmation,
  `core_retrieval`, or `release_candidate` evidence

## 1. Purpose

This journal reconstructs the complete relation-graph investigation in one
reviewable sequence. It is intended for another AI reviewer, a future human
reviewer, or a maintainer recovering context without relying on chat history.

It records:

1. the retrieval gap that motivated the graph work;
2. the frozen inputs and controls;
3. each hypothesis, implementation boundary, real-corpus run, result, and
   decision;
4. the difference between formal answer completion and useful evidence;
5. the exact evidence locations and commit lineage;
6. what was retained, rejected, and deferred; and
7. the conditions under which graph work may be reopened.

This file is a synthesis, not a new source of experimental authority. If a
number here conflicts with an immutable run or its focused evidence document,
the run artifact and focused evidence document win. The evidence index is
[`evidence/phase-07/README.md`](evidence/phase-07/README.md).

## 2. Executive conclusion

The investigation established four separate facts.

1. **Exact static relation recovery works.** The sidecar recovered the pinned
   Go call and TypeScript type-reference relations at exact source ranges.
2. **A graph does not determine natural-language relevance.** The required
   G09 and X08 edges were reachable from the fixed dense seed universe, but
   simple deterministic selectors chose other valid relations.
3. **Bounding the graph frontier works as complexity control.** The real
   per-query frontier fell to at most 8 chi edges and 11 RHF edges. The global
   32-edge ceiling was never reached.
4. **The tested admission policies are not product-quality selectors.** The
   final bridge-or-Pareto arm reached formal `32/32`, but only `7/17` emitted
   bundles contained reviewed useful evidence and `10/17` were noise-only.

The current decision is therefore:

- retain the full relation-occurrence databases and traces as ignored,
  evaluation-only evidence;
- retain the bounded frontier as a provisional development control;
- do not import the sidecar, frontier, edge weights, or admission policies
  into production indexing, storage, search, MCP, FTS, RRF, or embedding
  paths;
- do not reduce the frontier further on these exposed 32 questions;
- defer any optional graph search path until a new, separately frozen
  development unit exists; and
- continue the core release path through unexposed confirmation, Phase 12,
  and Phase 14 without making graph integration a blocker.

Further graph pruning is not the current gap-closing work. Fanout explosion
was controlled. The remaining problem is deciding which structurally valid
relation is relevant to the question and when to abstain. More pruning on the
same exposed cases would increase the risk of dropping rare useful evidence
such as G09 and of encoding case-specific behavior.

## 3. What the system could and could not do before the graph work

The accepted local retrieval path was:

```text
source files
-> Tree-sitter semantic parents and AST-aware segments
-> 1024-dimensional Voyage document embeddings
-> active int8 serving vectors
-> exhaustive dense scan
-> segment-to-parent collapse
-> bounded source-body packaging
```

FTS5/BM25 remained a separate lexical lane. Two frozen RRF probes were
rejected because they regressed already complete RHF cases and worsened the
reviewed evidence mix. No additional RRF weight tuning was authorized.

The 32-case frozen baseline was:

| Lane | Complete@5 | Hit@5 | Complete@20 | Hit@20 |
| --- | ---: | ---: | ---: | ---: |
| dense `1024/int8` | 30/32 | 30/32 | 31/32 | 32/32 |
| FTS | 17/32 | 19/32 | not opened | not opened |
| equal RRF | 24/32 | 26/32 | not opened | not opened |
| FTS1:dense2 RRF | 28/32 | 29/32 | not opened | not opened |

Dense `Hit@20 = 32/32` was the key diagnosis: broad semantic localization was
already strong. The missing layer was not primarily another candidate
generator. It was the transition from a plausible candidate to exact relation
proof and a complete evidence bundle.

The gap was decomposed into:

```text
semantic localization
-> exact language-level relation resolution
-> answer-evidence completion across parents
-> bounded context packaging
```

The detailed research and online-system survey preceding implementation is in
[`RELATION-AWARE-CODE-CONTEXT-RESEARCH.md`](RELATION-AWARE-CODE-CONTEXT-RESEARCH.md).

## 4. Frozen experimental authority

### 4.1 Corpora and questions

| Corpus | Pinned source | Questions | Language slice |
| --- | --- | ---: | --- |
| go-chi/chi v5.3.1 | commit `8b258c7bb28f97a5f2a856ff7ef962578fec9215` | 12 | Go |
| react-hook-form v7.85.0 | commit `371432c39271aab739358d19c406793771565ab3` | 20 | 12 TypeScript, 8 TSX |

Tracked frozen datasets:

- [`behavior-go-chi-v5.3.1-calibration-frozen-v1.json`](../../testdata/retrieval/behavior-go-chi-v5.3.1-calibration-frozen-v1.json),
  SHA-256
  `34d95e76d57d88be57cdf23f341c10724dd42fcfe213786b8620595a0ae9c1e1`;
- [`behavior-react-hook-form-v7.85.0-calibration-frozen-v1.json`](../../testdata/retrieval/behavior-react-hook-form-v7.85.0-calibration-frozen-v1.json),
  SHA-256
  `e5c93b9e7823e155b0c31e7b2994ba1ccf96880fcad5e680bc7a46adbcbd8ecf`.

Frozen provider-free dense replays:

- chi: `90efcc02c9c4e826515ad56d5d3a96104782840503e110e3835b24880cd50bb5`;
- RHF: `5909878346400b307d1f97baea1f5ce939b0ae10ad722d4ef8799e8f57b67bd4`.

The complete label and replay checkpoint is
[`dual-ai-calibration-freeze-r4.md`](evidence/phase-07/dual-ai-calibration-freeze-r4.md).

### 4.2 Relevance review protocol

ChatGPT and Grok separately reviewed shuffled, source-complete relation pools
with rank, score, lane, prior label, experiment result, the other review, and
owner preference hidden. The owner adopted the reconciled payload digests as
a whole.

This is intentionally recorded as:

```text
protocol_version    owner-adopted-dual-ai-v1
relevance_authority OWNER_ADOPTED_DUAL_AI_REVIEW
review_validation   NO_INDEPENDENT_HUMAN_REVIEW
```

It must not be described as human-reviewed. The packets are suitable for
internal calibration and review, but the exposed set is below the
promotion-capable confirmation floor.

### 4.3 Controls shared by every graph diagnostic

- questions, labels, required groups, and hard negatives remained unchanged;
- the dense `1024/int8` top 20 was reused without a fresh query embedding;
- the primary dense top five and their bodies were protected;
- no FTS, RRF, query rewrite, relation embedding, or LLM relation generation
  participated in graph selection;
- selection was label-blind; labels were opened after selection for scoring;
- only exact in-corpus compiler-resolved facts were eligible;
- every run recorded provenance and artifact checksums;
- no document or query provider operation was performed; and
- all sidecar databases and run artifacts stayed under ignored `.cidx/test`
  state.

## 5. Representative cases and regression guards

### 5.1 G09: caller and callee are both required

The chi question requires two semantic parents:

```text
middleware.RealIP
  --CALLS-->
middleware.realIP
```

Dense placed `RealIP` at rank 8 and omitted `realIP` through rank 20. The
misleading `walkXFF` parent was rank 5 and is a reviewed grade-0 safety guard.
G09 therefore tests whether an internal seed can be connected to a graph-only
endpoint without surfacing an unrelated top-five neighbor as supporting
evidence.

Verified relation:

- relation ID:
  `aee07df0bb4d608d25e240e7cce47771b11683f4eb93292ff443eed390351c60`;
- occurrence: `middleware/realip.go:1111..1120`;
- direction/kind: `FORWARD / CALLS`;
- both endpoints: required grade 2.

### 5.2 X08: consumer and public contract are both required

The RHF question requires the public component and its props contract:

```text
module.FormState
  --TYPE_REF-->
module.FormStateProps
```

Dense placed `FormState` at rank 2 and `FormStateProps` at rank 13. The exact
type-reference edge is therefore available from a dense-localized parent, but
it competes with many other valid TypeScript type references.

Verified relation:

- relation ID:
  `4baa57b19ec63201ccb8af423b5fb56f4f8f8a154b0cebe4e7dffbfbbd36d43e`;
- occurrence: `src/formStateSubscribe.tsx:688..702`;
- direction/kind/tier:
  `FORWARD / TYPE_REF / DECLARATION_CONTRACT`;
- required endpoint body: complete, 230 bytes.

### 5.3 T09 and T10

T09 and T10 were already complete in the dense top five. They were retained
as relation-role and packaging regression guards:

- T09: cross-file/import-export type identity from `createFormControl` to
  `Control`;
- T10: same-file recursive type relations between `PathInternal` and
  `PathImpl`.

They must not be counted as new graph gains.

## 6. Experimental method

### 6.1 Relation authority

Tree-sitter located occurrences and enclosing semantic parents. Go targets
were resolved with `go/packages` and `go/types`; TypeScript/TSX targets were
resolved with a pinned TypeScript `Program` and `TypeChecker` over the exact
indexed universe.

The sidecar stored forward `CALLS`, `TYPE_REF`, and `MEMBER_OF` occurrences.
Incoming/reverse traversal was derived from the same occurrence facts rather
than stored as a second authority. Unresolved, ambiguous, out-of-corpus, and
parent-mapping failures remained visible in denominators.

### 6.2 Stored metadata evolution

The relation record evolved without relation-text embedding:

| Version | Added mechanically derived information |
| --- | --- |
| v1 | source/target parent, exact occurrence range, relation kind, resolution outcome, provenance |
| v2 | syntactic zone and role, flow role, execution mode, control context, file role, nearby normalized identifiers, occurrence ordinal, parent traits |
| v3 | `TYPE_VALUE_PARAMETER` for value-parameter type contracts in Go and TypeScript/TSX |

The metadata describes what the syntax/compiler can prove. It does not claim
business intent or answer relevance.

### 6.3 First-loss attribution

Each query was traced through:

```text
DENSE_TOP20_LOCALIZATION
-> OCCURRENCE_EXTRACTION
-> LANGUAGE_RESOLUTION
-> TARGET_PARENT_MAPPING
-> RELATION_REACHABILITY
-> RELATION_ADMISSION
-> BUNDLE_PARENT_CAP
-> RELATED_BODY_PACKAGING
-> ANSWER_EVIDENCE_COMPLETION
```

This distinction prevented a failed selector from being mistaken for a
parser, compiler, or graph-reachability failure.

### 6.4 Metric separation

No weighted total quality score was created. The series tracked separately:

- formal required-group completion;
- useful versus noise-only emitted bundles;
- grade-2, grade-1, grade-0, and unreviewed attachments;
- reviewed hard-negative and `walkXFF` attachment;
- primary top-five identity/order/score/body preservation;
- raw, deduplicated, capped, and final frontier sizes;
- body packaging and omission reasons;
- deterministic initial/repeat artifact hashes; and
- provider operations, which remained zero.

`32/32` formal completion is therefore not interchangeable with good answer
quality.

## 7. Chronological experiment log

### 7.1 E0 — frozen dense/FTS/RRF baseline

**Question.** Is the remaining loss mainly candidate discovery, or does it
occur after broad localization?

**Result.** Dense reached every case by top 20 but was complete on only 31/32
by top 20 and 30/32 by top 5. FTS and the two fixed RRF arms were weaker at
top 5; the weighted RRF arm regressed RHF T09/T10 and the reviewed grade mix.

**Decision.** Keep dense as the calibration baseline, keep FTS separate, stop
RRF weight tuning, and investigate exact relation/evidence completion.

**Evidence.** [`dual-ai-calibration-freeze-r4.md`](evidence/phase-07/dual-ai-calibration-freeze-r4.md).

### 7.2 E1 — exact compiler-resolved relation sidecar

- implementation commit: `02834052921116a6341c44d7f7fd7e51f6a87005`;
- policy: `one-hop-kind-order-type-call-member-one-bundle-v2`.

**Hypothesis.** One exact one-hop relation from the fixed dense top-20 seed
universe may supply the missing parent while preserving the primary top five.

**Graph result.**

| Corpus | Files | Parents | Occurrences | Resolved unique |
| --- | ---: | ---: | ---: | ---: |
| chi | 78 | 452 | 6,737 | 1,740 |
| RHF | 237 | 322 | 26,011 | 3,267 |

Exact G09, X08, T09, and T10 probes passed, including reverse G09 caller
lookup. Selection nevertheless remained `30/32`: both G09 and X08 were
reachable but not selected. Their first loss was `RELATION_ADMISSION`.

**Decision.** Relation extraction is viable. A fixed relation-kind/rank order
is not a relevance selector. Keep the sidecar; do not integrate it into the
product.

**Evidence.** [`relation-usage-graph-diagnostic-r4.md`](evidence/phase-07/relation-usage-graph-diagnostic-r4.md).

### 7.3 E2 — AST/compiler edge metadata and dense-first selection

- implementation commits:
  `a5efa7832148a6ddffeb600a0ba7b22089ead9a6` and
  `c197cdafa93852df2c1463d2636378caae288130`;
- dense-first policy: `query-edge-metadata-dense-first-v1`;
- conditional crossover policy:
  `query-edge-metadata-graph-first-dense-crossover-v1`.

**Hypothesis.** Callsite zone, flow, control, file role, nearby identifiers,
and parent traits may distinguish answer-bearing relations without embedding
relation text.

**Result.** Dense-first metadata selected exact G09 and improved formal
coverage to `31/32`. X08 remained an admission loss. The graph-first crossover
also ended at `31/32`, added no newly complete case, and attached the protected
`middleware.walkXFF` parent to chi G05.

**Decision.** Retain metadata for diagnostics. Reject graph-first-before-dense
under this fixed policy. The remaining gap is query-conditioned admission.

**Evidence.** [`relation-edge-metadata-diagnostic-r4.md`](evidence/phase-07/relation-edge-metadata-diagnostic-r4.md).

### 7.4 E3 — generic value-parameter contract metadata

- implementation commit: `7879ab7315bd215fab34d5756b6416158b6c382d`;
- policy: `query-edge-value-parameter-dense-first-v1`.

**Hypothesis.** X08 may represent a common script-language pattern where a
public component/function consumes an explicit props/contract type as a value
parameter. A generic AST role could identify that class without an
X08-specific rule.

**Structural result.** All six reviewed public RHF components were correctly
classified as `SIGNATURE / TYPE_VALUE_PARAMETER / DECLARATION`. The graph
contained 89 resolved chi and 513 resolved RHF value-parameter type uses.

**Retrieval result.** Coverage stayed `31/32`. X08 remained
`RELATION_ADMISSION`; the selector chose another structurally valid
value-parameter relation.

**Decision.** The metadata feature is mechanically sound, but it does not
solve relevance. The owner deferred policy judgment; no question or key was
changed after observing the result.

**Evidence.** [`relation-value-parameter-diagnostic-r4.md`](evidence/phase-07/relation-value-parameter-diagnostic-r4.md).

### 7.5 E4 — directional anchor and edge-strength comparison

- implementation commit: `dd814915902986c3fcb5a36220a35d5f8297b894`;
- arms: raw frequency, source-normalized focus, bidirectional specificity,
  and incoming-popularity control.

**Hypothesis.** Two query-matched dense anchors plus graph-wide edge
frequency/focus/fan-in statistics may identify a coherent relation. Incoming
popularity was included as a negative control.

**Result.** All four arms reached formal `32/32`, preserved top five, and
attached no declared hard negative or `walkXFF`. That headline concealed
substantial noise:

| Corpus | Arm | Useful queries | Noise-only queries |
| --- | --- | ---: | ---: |
| chi | every arm | 6 | 6 |
| RHF | raw frequency | 9 | 11 |
| RHF | source-normalized | 8 | 12 |
| RHF | specificity | 6 | 14 |
| RHF | popularity control | 10 | 10 |

Source-normalized focus and specificity selected the exact X08 relation. Raw
frequency and popularity completed X08 incidentally because the required
anchor was packaged, not because the selected relation explained the answer.

**Decision.** Reject every unconditional arm. The next problem is whether to
admit any relation bundle, not how to order every graph edge.

**Evidence.** [`relation-anchor-edge-strength-diagnostic-r4.md`](evidence/phase-07/relation-anchor-edge-strength-diagnostic-r4.md).

### 7.6 E5 — bounded frontier and bridge abstention

- implementation commit: `770ff8e0c6c151791d5599bbdf68bd730dab7e99`;
- controls: top two per `anchor x direction x structural tier` bucket,
  canonical union without backfill, hard global maximum 32.

**Hypothesis.** Control graph fanout before judging another admission rule.

**Complexity result.**

| Corpus | Raw | After self removal | Bucket-distinct | After bucket cap | Final canonical | Per-query max | Queries at 32 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| chi | 119 | 115 | 79 | 56 | 51 | 8 | 0 |
| RHF | 1,290 | 711 | 362 | 158 | 153 | 11 | 0 |

The bucket cap performed the material reduction; the global 32 ceiling was
non-binding.

**Relevance result.** Cap-only reproduced the earlier specificity selection
and all its noise. Direct-anchor bridge abstention emitted 10/32 bundles,
4/10 noise-only, preserved X08, and lost G09 because G09 connects an anchor to
a graph-only endpoint.

**Decision.** Retain the cap only as provisional development complexity
control. Reject cap-only as relevance policy and bridge-only as general
admission policy.

**Evidence.** [`relation-frontier-cap-diagnostic-r4.md`](evidence/phase-07/relation-frontier-cap-diagnostic-r4.md).

### 7.7 E6 — final graph-only Pareto admission

- implementation commit: `497c000bf0d3e9452fd8ff1ce9f570a3df144525`;
- documentation commit: `4af2a57`;
- policy: `anchor-frontier-graph-only-pareto-v1`;
- policy fingerprint:
  `2ed879c70143bdd8de1287092b877013c7392fc967b64b42175a9128604083d6`.

**Hypothesis.** From the unchanged bounded frontier, admit a direct bridge if
present; otherwise emit only when one outgoing graph-only endpoint is the
unique per-tier Pareto survivor under source focus, source target diversity,
and target incoming-source fan-in.

**Frontier partition.**

```text
204 final frontier edge views
= 11 direct bridges
+ 64 incoming exclusions
+ 38 outgoing dense-top-20 endpoint exclusions
+ 91 outgoing graph-only candidates
```

The 91 graph-only candidates produced 58 per-tier nondominated survivors.

| Outcome | Count |
| --- | ---: |
| direct bridge | 10 |
| unique graph-only winner | 7 |
| multiple winners, abstain | 13 |
| no candidate, abstain | 2 |
| emitted | 17 |
| abstained | 15 |

**Quality result.**

| Measurement | Result |
| --- | ---: |
| baseline complete | 30/32 |
| augmented formal complete | 32/32 |
| useful emitted bundles | 7/17 |
| noise-only emitted bundles | 10/17 |
| direct-bridge useful | 6/10 |
| unique-Pareto useful | 1/7 |
| declared hard-negative queries | 0 |
| `walkXFF` attachments | 0 |

G09 was recovered through the unique graph-only `RealIP -> realIP` winner.
X08 was recovered through the direct `FormState -> FormStateProps` bridge.
Initial/repeat runs were deterministic, and the primary top-five and final
frontier hashes matched the accepted frontier-cap artifacts.

**Decision.** Reject bridge-only, Pareto-only, and the combined rule for
product use. Close all policy tuning on the exposed 32 questions.

**Evidence.** [`relation-graph-only-pareto-diagnostic-r4.md`](evidence/phase-07/relation-graph-only-pareto-diagnostic-r4.md).

## 8. Consolidated measurement ledger

`Augmented complete` below means completion after protected dense primary
results plus bounded related evidence. It is not a new flat top-K ranking.

| Stage | Formal result | What improved | What failed | Product decision |
| --- | --- | --- | --- | --- |
| dense baseline | 30/32 at top 5 | all cases hit by top 20 | G09/X08 evidence groups incomplete at top 5 | retain baseline |
| exact graph v1 | 30/32 | exact G09/X08 reachability proven | fixed selector missed both | no integration |
| metadata v2 dense-first | 31/32 | exact G09 recovered | X08 admission | diagnostic only |
| metadata v2 graph-first | 31/32 | no extra gain | `walkXFF` safety failure | reject |
| value-parameter v3 | 31/32 | generic RHF contract role proven | X08 admission persists | policy deferred |
| four strength arms | 32/32 | both missing groups formally completed | 6-14 noise-only queries per corpus/arm | reject all |
| cap-only | 32/32 | frontier reduced without changing selection | 20/32 noise-only emissions | reject relevance policy |
| bridge-only | 31/32 | 10 emissions instead of 32 | loses G09; 4/10 noise-only | reject |
| bridge + unique Pareto | 32/32 | exact G09 and X08 recovered | only 7/17 useful; 10/17 noise-only | reject and close tuning |

The important experimental advance was not the final `32/32`. It was the
ability to say precisely:

- relation extraction and compiler resolution are not the G09/X08 failure;
- uncontrolled frontier size can be bounded deterministically;
- graph-first traversal can introduce unsafe context;
- edge strength can identify exact relations in representative cases but is
  not a universal relevance signal; and
- admission/abstention remains the unsolved product problem.

## 9. Why further frontier reduction is not currently justified

The user raised a valid concern that thousands of graph connections make a
specific relation difficult to interpret. The frontier-cap run directly
tested that concern.

For these two corpora, the query-local graph did not remain at thousands of
connections after normalization:

- self edges were removed;
- repeated occurrences were collapsed into canonical typed/tier edges;
- only two edges survived each anchor/direction/tier bucket;
- cross-bucket duplicates were collapsed; and
- no query exceeded 11 final edges.

This is a meaningful complexity improvement. It makes every remaining edge
inspectable and keeps evaluation bounded.

It did **not** improve relevance. Cap-only selected the same relation as the
uncapped specificity arm. The direct-bridge filter reduced noise but removed
G09. The final Pareto filter recovered G09 while still producing six
noise-only graph-only emissions.

Further reduction on the same cases would be weak evidence because:

1. the remaining 1-11 edges are already small enough to inspect;
2. the unresolved distinction is semantic relevance, not cardinality;
3. G09 demonstrates that a rare graph-only edge can be necessary;
4. choosing another cutoff after seeing G09/X08 risks case-specific tuning;
5. the global 32 ceiling was never exercised, so its general sufficiency is
   unproven; and
6. there is no unexposed confirmation set on which to measure a new cutoff.

The correct stopping point is the current bounded frontier plus abstention
evidence, not repeated pruning until the exposed labels look clean.

## 10. Full sidecar retention boundary

The complete relation facts are retained because they are useful for:

- verifying extraction and compiler-resolution denominators;
- separating reachability from admission loss;
- reproducing incoming and outgoing relation diagnostics;
- comparing future bounded policies against the same immutable fact base;
- checking exact relation spans and toolchain/source provenance; and
- designing future language or relation support without rebuilding historical
  evidence.

They are not a product dependency. The current boundary is:

```text
ignored evaluation state
  complete occurrence sidecar
  full provenance and resolution summaries
  query-local bounded frontier traces
  label-late scoring artifacts

product state
  no relation sidecar
  no graph schema
  no graph search path
  no graph MCP surface
```

A future production design must not simply copy this evaluation database into
the product. It would need a separate bounded, generation-bound adjacency
representation with independently frozen confirmation and latency/memory
budgets. Evaluation-sidecar growth must not constrain that design.

## 11. External comparison: code-review-graph

The external review inspected `tirth8205/code-review-graph` at pinned commit
[`1a010deed6c283d4aa1e7e949e78fe3a7bcdfbb3`](https://github.com/tirth8205/code-review-graph/tree/1a010deed6c283d4aa1e7e949e78fe3a7bcdfbb3).

Its primary purpose differs from this cidx diagnostic:

| Dimension | code-review-graph | cidx relation diagnostic |
| --- | --- | --- |
| primary task | review context, blast radius, architecture/navigation | natural-language candidate confirmation and answer evidence |
| graph source | broad Tree-sitter and framework heuristics | exact compiler-resolved Go/TS occurrences for the tested relation types |
| selection | traversal, communities, centrality, heuristic criticality/impact weights | fixed dense localization, bounded one-hop facts, label-blind diagnostic admission |
| retrieval | FTS5 plus optional embeddings and fusion | exhaustive `1024/int8` dense baseline; FTS kept separate in this experiment |
| evaluation emphasis | token reduction, impact/flow/review workflows | required-group completion, attachment usefulness, hard negatives, first loss |
| storage role | product graph in local SQLite | ignored evaluation-only SQLite sidecar |

Potentially useful later ideas are incremental graph maintenance, explicit
entry/root metadata, additional typed relations such as `INHERITS`,
`IMPLEMENTS`, and `TESTED_BY`, and explicit traversal budgets.

They do not close the current admission problem. Fixed edge weights,
centrality, community membership, or a graph-first search surface can promote
popular or well-connected but irrelevant code. The external project's own
README reports search-ranking and flow-detection limitations and explicitly
notes circularity in one graph-derived impact-recall measurement. Those are
useful cautions against treating graph connectivity as answer correctness.

No code or algorithm from code-review-graph was adopted. Selective graph-path
work is deferred.

## 12. Review and validation trail

### 12.1 Roles

| Reviewer/source | Role in this series | Authority limit |
| --- | --- | --- |
| owner | chose corpora, authorized provider use, adopted frozen labels, made product decisions | sole project governance authority |
| kb-guide | measurement design, denominators, cap/admission interpretation | advisory measurement review |
| ChatGPT and Grok | independent relevance packets and design-direction review | dual-AI evidence; not human review |
| Terra | scoped implementation and immutable-artifact review | code/artifact correctness, not product relevance authority |

### 12.2 Code and artifact review

Focused evidence documents record the exact boundary commands and review
findings. Across the accepted series these included:

```text
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 -race ./internal/relationdiag ./internal/devlab
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 ./...
env -u VOYAGE_API_KEY GOPROXY=off go vet ./internal/relationdiag ./internal/devlab
env -u VOYAGE_API_KEY GOPROXY=off go build ./cmd/cidx
node --check tools/relationdiag/typescript-resolver.mjs
go mod tidy -diff
gofmt -l internal/relationdiag internal/devlab/relations.go
git diff --check
```

Review corrections included exact artifact binding, TypeScript exact-universe
confinement, source/toolchain reproof, MEMBER_OF support, per-parent hard
negative accounting, correct first-loss stages, typed comparator traces,
bucket-local bridge reservation, and exact frontier/Pareto denominator checks.

## 13. Evidence and artifact map

### 13.1 Tracked evidence

| Topic | File |
| --- | --- |
| research and design context | [`RELATION-AWARE-CODE-CONTEXT-RESEARCH.md`](RELATION-AWARE-CODE-CONTEXT-RESEARCH.md) |
| frozen questions/labels/replay | [`dual-ai-calibration-freeze-r4.md`](evidence/phase-07/dual-ai-calibration-freeze-r4.md) |
| exact relation sidecar | [`relation-usage-graph-diagnostic-r4.md`](evidence/phase-07/relation-usage-graph-diagnostic-r4.md) |
| metadata and graph-first | [`relation-edge-metadata-diagnostic-r4.md`](evidence/phase-07/relation-edge-metadata-diagnostic-r4.md) |
| value-parameter follow-up | [`relation-value-parameter-diagnostic-r4.md`](evidence/phase-07/relation-value-parameter-diagnostic-r4.md) |
| edge-strength arms | [`relation-anchor-edge-strength-diagnostic-r4.md`](evidence/phase-07/relation-anchor-edge-strength-diagnostic-r4.md) |
| frontier cap and bridge ablation | [`relation-frontier-cap-diagnostic-r4.md`](evidence/phase-07/relation-frontier-cap-diagnostic-r4.md) |
| final Pareto arm | [`relation-graph-only-pareto-diagnostic-r4.md`](evidence/phase-07/relation-graph-only-pareto-diagnostic-r4.md) |
| Phase 07 evidence index | [`evidence/phase-07/README.md`](evidence/phase-07/README.md) |

### 13.2 Ignored local evidence

All machine-local references are repository-relative. No absolute checkout
path belongs in a tracked manifest.

Primary locations:

```text
.cidx/test/experiments/review-union/current/
.cidx/test/experiments/relation-anchor-edge-strength/
.cidx/test/experiments/relation-frontier-cap/
.cidx/test/experiments/relation-graph-only-pareto/
.cidx/test/states/chi-1024-int8/evaluations/
.cidx/test/states/react-hook-form-1024-int8/evaluations/
```

An immutable graph directory contains:

```text
relations.db
graph-manifest.json
resolution-summary.json
artifact-checksums.json
```

A diagnostic directory contains a policy-specific subset of:

```text
run-manifest.json
primary-top5-proof.jsonl
stage-a-reachability.jsonl
stage-b-admission-order.jsonl
stage-b-bundles.jsonl
per-query-relation-trace.jsonl
related-body-packages.jsonl
frontier-cap-diagnostic.jsonl
frontier-graph-only-pareto.jsonl
frontier-graph-only-pareto-denominators.json
aggregate-relation-metrics.json
probe-results.json
report.md
artifact-checksums.json
```

The local artifacts were present at journal creation. They are intentionally
ignored and are not portable Git history. The tracked evidence documents
preserve their relative references, entry checksums, executable commits,
policy IDs, and aggregate conclusions.

### 13.3 Commit lineage

| Commit | Purpose |
| --- | --- |
| `3b1c5fd` | freeze dual-AI calibration labels |
| `0283405` | add exact relation graph diagnostic |
| `4505044` | record exact-graph result |
| `a5efa78` | add edge metadata diagnostic |
| `c197cda` | fix empty TypeScript metadata arrays |
| `3323e45` | record metadata results |
| `7879ab7` | classify value-parameter contracts |
| `0f08c83` | record deferred value-parameter result |
| `dd81491` | compare directional edge strengths |
| `bd29870` | record edge-strength result |
| `770ff8e` | bound relation frontier |
| `f287641` | record frontier-cap result |
| `654f58c` | freeze final Pareto diagnostic |
| `497c000` | add graph-only Pareto admission |
| `4af2a57` | record final Pareto result and close exposed-set tuning |

## 14. Threats to validity

Reviewers should keep these limitations attached to every conclusion:

1. The set contains two repositories and 32 exposed calibration questions.
2. The labels were owner-adopted after dual-AI review; there was no independent
   human reviewer.
3. Every admission policy after the first diagnostic was measured on an
   already exposed set. It cannot provide confirmation or promotion evidence.
4. Go and TypeScript/TSX semantics do not establish behavior for other
   languages, dynamic dispatch, reflection, dependency injection, or runtime
   paths.
5. One-hop CALLS/TYPE_REF/MEMBER_OF facts do not model full control or data
   flow.
6. The global 32-edge ceiling was never reached, so its product adequacy is
   unmeasured.
7. Formal required-group completion does not prove that an assistant will use
   the added source correctly.
8. The evaluation sidecar deliberately optimizes provenance and diagnosis,
   not production database size, incremental update cost, or request latency.
9. No production graph path, product schema, or MCP response was exercised.
10. The final result rejects the tested policies; it does not prove that every
    possible relation-aware policy is ineffective.

## 15. What is retained, rejected, and deferred

### Retained as evidence or diagnostic machinery

- compiler-resolved occurrence facts and exact source spans;
- bidirectional lookup derived from one occurrence authority;
- syntactic and compiler-derived edge metadata;
- source focus, source diversity, target fan-in, and structural-tier traces;
- first-loss attribution;
- exact provenance and immutable artifact checksums;
- self-edge removal and repeated-occurrence aggregation; and
- the current bounded query frontier as provisional development control.

### Rejected for current product use

- relation-text embeddings;
- a graph as an equal RRF voter;
- graph-first-before-dense crossover;
- unconditional relation-kind or edge-strength ordering;
- raw occurrence frequency and incoming popularity as relevance;
- cap-only selection;
- direct-bridge-only admission;
- unique graph-only Pareto admission; and
- the combined bridge-or-Pareto policy.

### Deferred

- a selective product graph path;
- a compact production adjacency schema;
- multi-hop path or entrypoint/root projections;
- `INHERITS`, `IMPLEMENTS`, `TESTED_BY`, override, data-flow, or control-flow
  overlays;
- incremental graph publication with production generations;
- assistant-use evaluation of related evidence; and
- any new admission policy, until a separately frozen development unit and an
  unexposed confirmation set exist.

## 16. Reopening criteria and remaining work

Graph work may be reopened only when at least one of these creates a genuinely
new experiment:

- a new frozen, unexposed cohort exposes a repeatable relation/evidence loss;
- assistant-use evaluation shows that a bounded related-evidence bundle
  materially improves final answers;
- a language slice requires a relation type absent from the current fact
  model;
- a production latency/memory design can be evaluated independently from the
  complete sidecar; or
- an admission rule is specified before its outcomes are inspected and has
  confirmation data available.

The next core work is not another cutoff or edge-weight experiment. It is:

1. author and freeze a separate unexposed confirmation set under the Phase 07
   contract;
2. run the accepted core retrieval lanes without changing the closed 32-case
   calibration set;
3. complete official Phase 12 retrieval evidence;
4. evaluate assistant use and host behavior at the later phase boundary; and
5. decide release-candidate status in Phase 14 from immutable scoped evidence.

## 17. Reviewer checklist

A new reviewer should answer these questions independently:

1. Do the dataset, replay, graph, executable, policy, and artifact hashes bind
   each run to one immutable input chain?
2. Were labels unavailable until after frontier construction and admission?
3. Are G09 and X08 exact source relations proven by path, content identity,
   byte range, direction, and cardinality rather than symbol-name coincidence?
4. Does `primary-top5-proof.jsonl` show identity, order, score, and body
   preservation?
5. Do the frontier denominators reconcile raw occurrences, self removal,
   occurrence collapse, bucket caps, global deduplication, and final counts?
6. Are useful, noise-only, grade-0, unreviewed, and hard-negative attachments
   evaluated separately from formal completeness?
7. Are initial/repeat traces deterministic for the claimed stable subset?
8. Does any conclusion rely on the green safety gate as if it were a quality
   gate?
9. Is any evaluation-only sidecar imported into production code or schemas?
10. Is a proposed follow-up genuinely new, or is it another adjustment to the
    exposed G09/X08 outcomes?

The current evidence supports a strong negative product decision and a useful
diagnostic architecture decision: preserve the facts and bounded traces, but
do not confuse graph connectivity with answer relevance.
