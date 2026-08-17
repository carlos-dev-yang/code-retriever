# cidx v1 Evaluation and Promotion Contract

- Status: normative; solo-project relevance authority revised 2026-08-17
- Applies to: Phases 00 through 14 wherever evidence, comparison, or promotion is required
- Canonical product design: [Revision 4](../../local-code-search-mcp-v1-design-r4.md)
- Phase index: [README](README.md)
- Execution protocol: [EXECUTION-GUIDE](EXECUTION-GUIDE.md)
- Persistent phase state: [STATUS](STATUS.md)

This contract defines how cidx proves implementation correctness, retrieval quality, representation fidelity, packaging usefulness, and operational behavior. It is intentionally multi-axis. No weighted total score may replace its stage results or hard gates.

The contract adapts lessons from the sibling `knowledge-system` project after an English advisory review of its measurement contract, retrieval evaluator, promotion checker, RRF implementation, embedding experiments, and lessons learned. The transferable principles are stage-separated denominators, first-loss attribution, frozen paired runs, lane-specific evidence, and hard-gate promotion. HNSW and ANN-specific metrics are explicitly excluded.

## 1. Product and Evaluation Boundaries

1. cidx is an auxiliary local retrieval MCP used beside file readers, grep/symbol tools, compilers, and tests.
2. Product usefulness is measured as the marginal effect of adding cidx to those tools, not by forcing an assistant to use cidx and not by evaluating a cidx-only assistant as the primary product arm.
3. FTS and dense retrieval are parallel provider lanes. They must be measured separately before RRF.
4. cidx scans every eligible stored vector. Int8 differences from serving-dimension f32 are representation and codec losses, not ANN recall losses. Historical Binary/256 results are evidence-only.
5. Frozen source-backed relevance truth and serving-dimension f32 ranking are independent references:
   - frozen relevance truth measures whether useful code is retrieved under its recorded authority;
   - exhaustive serving-f32 ranking measures whether int8 preserves the chosen vector representation.
6. A codec may preserve f32 neighbors while the dense model is semantically wrong, or change f32 neighbors while retaining frozen relevance-gold results. Reports keep both facts visible.
7. Activation, successful materialization, schema validity, or readiness proves lifecycle correctness only. It is never quality evidence.
8. Latency, size, tokens, and cost are measured from the start, but they are not release gates until a budget is explicitly frozen before a confirmation run.
9. Eligible source files are capped at 1 MiB. Chunks are complete semantic AST parents, not user-configured byte slices. Production embedding segments target 1024 bytes; evaluation exercises AST-aligned 768-, 1024-, and 1536-byte segment cases without arbitrary splitting.
10. Voyage uses only the regular synchronous Embeddings endpoint: explicit 1024-dimensional f32 source output, at most 128 inputs and 256 KiB per request, concurrency 4, 30-second timeout, and at most three retries after 10/20/30 seconds, with a longer `Retry-After` winning. Batch Inference and asynchronous polling are excluded.

## 2. Evidence Path and First-Loss Model

Every required query evidence group follows this monotonic path:

```text
source available
-> parser chunk available
-> FTS-or-dense provider union found
-> dense segment-to-parent collapse survived, when applicable
-> RRF top-k survived
-> inline body survived
-> assistant used the evidence, when observed
-> task requirement was satisfied, when observed
```

FTS and dense are parallel. Record both lane outcomes. A lane miss is diagnostic; the primary retrieval loss is `PROVIDER_UNION_MISS` only when neither lane supplies a valid parent candidate.

Stable first-loss values are:

- `SOURCE_DISCOVERY`
- `PARSE_OR_CHUNK`
- lane-only `FTS_CANDIDATE_MISS`
- lane-only `DENSE_SEGMENT_MISS`
- `PROVIDER_UNION_MISS`
- `SEGMENT_PARENT_COLLAPSE`
- `RRF_FUSION`
- `BODY_PACKAGING`
- `ASSISTANT_USE`
- `ASSISTANT_RESOLUTION`
- `OPERATION_FAILURE:<stage>`
- `NOT_OBSERVED` for a downstream measurement not required by that run contract

Do not encode an unobserved optional stage as score zero. Required failures and timeouts remain in their metric denominator and receive the documented zero retrieval outcome plus a failure state.

The portable trace wire records one ordered observation for every planned stage. Every required evidence observation carries one ordered `GroupObservation` for each frozen requirement-group ID. `GroupObservation.first_loss` is the only retrieval-survival authority: a group that is absent after source discovery/parser or after the provider union starts the primary path cannot reappear and retains its original loss value. FTS and dense remain parallel lane diagnostics. The operational observation uses operation denominators and carries no retrieval groups. An `OPERATION_FAILURE:<stage>` group loss must name that observation's `failure_stage`; strict JSON Schema enforces wire shape while the core validator enforces these cross-record relationships.

## 3. Stage Scorecard

| Stage | Truth unit | Exact denominator | Required measurements |
| --- | --- | --- | --- |
| Source discovery and parser/chunker | Frozen eligible files and labeled eligible functions, methods, and types; exact source spans and bodies | File success: all eligible files. Construct recall: all labeled eligible constructs. Emission precision and fidelity: all emitted chunks in the reviewed slice | File success, construct recall, chunk precision, kind/symbol accuracy, byte/span/body fidelity, duplicate/overlap rate, unsupported-syntax classes, stale-row violations, clean-rebuild versus incremental equivalence |
| FTS candidate retrieval | Gold parent chunks and requirement groups mapped to parent chunks | Hit/MRR: all required answerable queries. Recall/coverage: all gold parents or requirement groups. Known-hard-negative rate: all verified abstainable/hard-negative queries with reviewed misleading targets | Parent Hit@1/5, Recall@k, MRR, NDCG@k, requirement coverage, exact-identifier Hit@1, duplicate rate, known-hard-negative Hit@k, returned candidate count |
| Dense segment retrieval | Gold source spans mapped to acceptable dense segments and parents | Relevance metrics: all required dense-relevant queries/groups. Codec fidelity: serving-f32 top-k and fixed-depth ranked pairs | Segment and parent-availability Hit/Recall/MRR/NDCG, serving-f32 top-k retention, missing candidates, rank displacement, pairwise inversion, ties, relevance-gold retention, vector coverage |
| Segment-to-parent collapse | Gold parents represented by pre-collapse dense segments | Survival: gold parents available before collapse. Alignment: all collapsed candidates. Parent retrieval: all query gold parents/groups | Parent Hit/Recall/MRR/NDCG, gold survival, segment-parent alignment, compression/dedup ratio, parent rank movement, relevant suppression, parent dominance |
| RRF fusion | Gold parents in the FTS/dense candidate union | Survival: gold parents available in the union. Final retrieval: all required query groups. Contribution: all candidates and gold from each lane | Fused Hit/Recall/MRR/NDCG, per-lane survival, unique-gold contribution, overlap/disagreement, rescue/harm, lane-to-fused rank movement, tie rate, deterministic order |
| Inline body packaging | Gold requirements present in fused top-k and their frozen indexed bodies | Survival: fused gold requirements. Fidelity: every serialized body. Budget loss: every packaged query | Gold/body survival, source-span/body fidelity, relevant-byte density, duplicate-body ratio, omission/truncation rate, first-gold position, omission reason, serialized bytes and estimated tokens |
| Assistant usefulness | Frozen code tasks, required facts/spans/actions, and expected test outcomes | All required tasks, including failed and timed-out runs. Utilization: all gold requirements presented to the assistant | Task success, requirement coverage, correct file/symbol, correct edit/test outcome, gold utilization, false-lead rate, time/tool calls to first useful evidence, total tool calls/tokens/cost, paired wins/losses |
| Operational correctness | Every required indexing, provider, search, and serialization operation | All scheduled/required operations, never successful operations only | Failure/timeout/retry rate, stage and total p50/p95/p99, index/vector coverage, freshness, profile integrity, storage/memory/scan scaling, paid calls/tokens/cost, privacy violations |

Assistant/task requirement coverage is a separate outcome surface. Do not append it mathematically to the retrieval survival chain because its truth unit and denominator differ.

## 4. Dataset and Label Contract

### 4.1 Query contract

Every frozen query records:

- stable query ID and exact text;
- language slice: `go | typescript | tsx | mixed`;
- one or more intent/cohort labels;
- answer mode: `SINGLE | BEST_N | EXHAUSTIVE | ABSTAINABLE`;
- expected result cardinality when known;
- required identifier, path, language, and scope constraints;
- durable file identity, content hash, qualified symbol, and source spans;
- required evidence groups and their valid alternatives;
- explicit hard-negative chunks and reasons;
- assistant-task requirements when end-to-end use is evaluated;
- review state, protocol version, relevance authority, validation limitation,
  reviewer/pass artifact digests, owner-adoption artifact digest, rationale,
  and dataset digest.

Generated chunk and segment row IDs are not durable truth. Map frozen file/hash/symbol/span truth into the active generation. A source-content change invalidates the mapping until reviewed.

### 4.2 OR alternatives and AND requirements

For query requirements `G1 ... Gn`:

- alternatives inside one `Gj` are OR: any reviewed implementation/span satisfies the requirement;
- separate groups are AND: all groups are required for complete support;
- duplicates do not add coverage;
- partial support is reported as fractional requirement coverage;
- complete requirement hit is separately binary.

If one requirement contains several mandatory spans, record its internal span coverage as well as its binary complete state.

### 4.3 Relevance grades

Use one frozen rubric:

- `0`: irrelevant, wrong, stale, or hard negative;
- `1`: useful supporting code;
- `2`: directly satisfies a required intent.

Primary strict Hit/Recall/MRR use direct grade `2` or satisfied requirement groups. NDCG uses grades `0/1/2`. A supplementary useful-support metric may use grade `>=1`, but it must not replace the strict metric.

`ABSTAINABLE` means the reviewed corpus contains no valid answer. cidx v1 has no confidence threshold or server guarantee to return an empty list, so a nonempty result list alone is not a false positive and is not a promotion failure. Record returned count and score/rank diagnostics, measure whether explicitly reviewed misleading hard-negative parents enter top-k, and measure downstream assistant false leads. If a future version adds an abstention contract, define and freeze its threshold and no-result metric separately before using it as a gate.

### 4.4 Required cohorts

At minimum include:

- exact symbol/identifier;
- path/package/module-qualified lookup;
- error string, literal, or configuration key;
- natural-language behavior;
- mixed identifier plus semantic intent;
- declaration versus implementation versus incidental mention;
- multiple implementations or exhaustive lookup;
- ambiguous/common names;
- verified hard negative and no valid answer;
- newly added, modified, renamed, deleted, and stale code;
- small and large constructs;
- Go-specific, TypeScript-specific, TSX-specific, and genuine mixed-language cases.

### 4.5 Dataset size and review floor

A promotion-capable confirmation series starts with at least:

- 90 answerable queries: at least 30 each for Go, TypeScript, and TSX;
- 18 verified `ABSTAINABLE` or hard-negative queries: at least 6 per language;
- 10 cases in every critical cohort; cohorts may overlap.

A 12–20 query set is execution smoke evidence only and cannot support a codec, dimension, fusion, or release claim.

The v1 solo-project frozen-label protocol is exactly:

```text
protocol_version     = owner-adopted-dual-ai-v1
relevance_authority  = OWNER_ADOPTED_DUAL_AI_REVIEW
review_validation    = NO_INDEPENDENT_HUMAN_REVIEW
```

It is the strongest available authority for this one-person project, but it is
not human review and must never be serialized, documented, or reported as
`HUMAN_REVIEWED`. The owner is the governance authority; the two AI systems
perform the source-backed relevance procedure.

Every frozen relation must pass all of these gates:

1. Two different AI systems or model families review independently. Each
   initial reviewer receives a separately shuffled packet and cannot see the
   other review, prior labels, arm identity, rank, score, experiment outcome,
   or owner preference.
2. Each packet binds the corpus, generation, dataset, arm set, pooling depth,
   source path/hash/symbol/span, exact reviewed source, shuffle identity, and
   complete relation set. Every relation is covered and source-attested.
3. Grade-2 and required-group disagreements are reconciled against the source;
   zero such conflict may remain. A support relation is grade 1 only when both
   reviewers agree under the declared support rubric; otherwise the frozen
   conservative value is grade 0 and the disagreement remains visible in the
   review artifact. Predeclare which metrics consume grade 1.
4. Every no-answer and hard-negative label has corpus-wide evidence plus
   explicit agreement from both reviewers. A pool miss alone is insufficient.
5. The owner adopts or rejects the reconciled digest as a whole. Owner
   adoption is not an additional relevance judgment. A relation-level owner
   override creates a new draft, records source rationale, and reopens both AI
   passes for the affected relation before any new freeze.
6. Frozen cases record both pass artifact SHA-256 values, the owner-adoption
   artifact SHA-256, the exact protocol and authority values above, and the
   permanent `NO_INDEPENDENT_HUMAN_REVIEW` limitation. Every derived report
   and promotion result propagates that limitation.

The review lifecycle is
`MACHINE_PREPARED_UNREVIEWED -> DUAL_AI_REVIEWED_UNRECONCILED ->
DUAL_AI_RECONCILED -> OWNER_ADOPTED_FROZEN`. The case wire keeps
`review.state=draft|frozen`; intermediate lifecycle values belong to the
digest-bound `label-review.json`. `review.state=frozen` is valid only at
`OWNER_ADOPTED_FROZEN`.

Pool unique top results from FTS, serving f32, active int8, and RRF for relevance review before labels freeze. Historical Binary/256 pools may remain as evidence but do not become current product arms. Pooling expands judgment coverage; it must not alter labels after confirmation results are known.

### 4.6 Calibration and confirmation

- Calibration data selects serving dimension, codec, RRF parameters, candidate limits, body budgets, and noninferiority margins.
- Confirmation data is independently frozen and cannot tune any of those values.
- Only a complete confirmation run may vote for promotion.
- A later label correction creates a new dataset version and invalidates direct delta claims against the old digest.
- If a label changes after confirmation results were exposed, provider-free
  rescoring is diagnostic/regression evidence only. It cannot restore that
  confirmation set's promotion authority. A new promotion claim requires a
  newly frozen, previously unexposed confirmation unit.

### 4.7 Paired-run compatibility

A paired delta requires identical:

- corpus manifest, worktree state, file hashes, and inclusion policy;
- query/label manifest and relevance grades;
- parser, chunker, FTS schema/tokenizer, and SQLite version;
- Voyage model/source dimension and, within one query comparison, the same ephemeral source query vector;
- reducer/version and serving dimension unless that field is the declared arm difference;
- candidate limits, collapse policy, RRF policy, body budget, and MCP schema;
- code commit and platform; latency additionally requires equivalent hardware/load;
- assistant model, prompt, tools, and budgets for end-to-end comparison.

Within one current-profile run, serving f32 and active int8 reuse the same
in-memory query f32 and the same serving-dimension document f32. Persist only
the query-vector hash. Ordinary evaluation uses 1024/int8; a frozen plan may
explicitly select supported compact 512/int8. A historical Binary or 256 reproduction
is outside the current matrix and requires the approval and isolation contract
in `RETIRED-VECTOR-PROFILES.md`. Cross-dimension deltas are paired only when
the source query vector and every other paired control match.

## 5. Metric Definitions

Let `Rq` be the unique relevant result set for query `q`, and `Lq@k` the unique top-k result list.

```text
Hit@k(q) = 1 if Rq intersects Lq@k, else 0
Hit@k = sum(Hit@k(q)) / required answerable queries

Recall@k(q) = |Rq intersect Lq@k| / |Rq|
Recall@k = macro mean over required queries with relevance labels

MRR(q) = 1 / first relevant rank, or 0 when absent
MRR = macro mean over all required queries

DCG@k(q) = sum_i=1..k ((2^relevance_i - 1) / log2(i + 1))
NDCG@k(q) = DCG@k(q) / IDCG@k(q), or 0 when no ideal gain exists
NDCG@k = macro mean over the declared denominator
```

Use unique parent IDs for parent metrics. Multiple matching segments from one parent do not earn additional gain.

For requirement groups:

```text
satisfied(Gj, L) = 1 if any reviewed OR alternative in Gj is present, else 0

RequirementCoverage(q, L) = sum_j(satisfied(Gj, L)) / number_of_groups

CompleteRequirementHit(q, L) = 1 only when every group is satisfied
```

For stage input gold `A` and stage output gold `B`:

```text
GoldSurvival(stage) = |A intersect B| / |A|
StageLoss(stage) = previous requirement coverage - current requirement coverage
```

Provider-union, collapsed-parent, fused, packaged-body, and assistant-used coverage must be monotonic. Attribute each requirement to the first transition from present to absent.

For reviewed misleading hard-negative parents `Hq`:

```text
KnownHardNegativeHit@k(q) = 1 if Hq intersects Lq@k, else 0
KnownHardNegativeHit@k = sum over verified hard-negative queries / that exact denominator
```

Do not rename this as abstention accuracy. It measures known misleading retrieval, not whether a top-k search returned anything.

## 6. Dense Representation and Codec Fidelity

For serving-dimension f32 top-k `Fk` and codec top-k `Ck`:

```text
TopKRetention = |Fk intersect Ck| / |Fk|
F32CandidatesMissing = |Fk - Ck|
Top1Mismatch = 1 when first(F) != first(C), else 0

GoldF32Retention = |Gold intersect Fk intersect Ck| / |Gold intersect Fk|
  when the denominator is nonzero
```

For shared candidates `I = Fk intersect Ck`:

```text
AbsoluteRankDisplacement(x) = |rankF(x) - rankC(x)|
MeanRankDisplacement = sum_x_in_I(AbsoluteRankDisplacement(x)) / |I|
```

Also report median, p95, maximum, and missing-candidate count. Mean displacement alone hides absent neighbors.

At a fixed comparison depth, report:

```text
PairwiseInversionRate = shared candidate pairs whose relative order differs / comparable shared pairs
ScoreTieRate = tied returned candidate pairs / comparable returned pairs
BoundaryTieRate = queries whose kth score ties a candidate outside top-k / required queries
```

Report repeated-run ranking-hash equality. Every current f32 and int8 arm also reports frozen relevance-gold metrics and its authority. Never call these fidelity measurements ANN recall.

Do not subtract codec raw scores from f32 cosine unless the codec reconstructs that exact normalized quantity. BM25, f32 cosine, int8 score, and RRF are different scales. Historical binary scores remain incomparable evidence.

## 7. RRF Contract and Diagnostics

Use a versioned formula:

```text
RRF(parent) = sum_lane(weight_lane / (k0 + rank_lane(parent)))
```

- ranks are 1-based;
- an absent lane contributes zero;
- per-lane cutoff and candidate union rules are frozen;
- tie-breaking is deterministic and lane-neutral;
- store component ranks and contributions, not only the fused score;
- `k0=60` is an acceptable initial calibration convention, not a confidence threshold, retrieval cutoff, or proven optimum.

Every RRF run reports:

- FTS/dense candidate-set Jaccard overlap@k;
- top-1 disagreement rate;
- rank correlation or rank-biased overlap on shared candidates;
- gold unique to FTS, unique to dense, and found by both;
- lane marginal recall under lane ablation;
- candidate and gold survival from each lane into fused top-k;
- mean/p95/max lane-to-fused rank movement;
- `fusion_rescue`: fused succeeds where a standalone lane fails;
- `fusion_harm`: a standalone lane succeeds but fused top-k loses gold;
- both-lane, FTS-only, and dense-only contribution counts;
- exact-score and top-k-boundary tie rates;
- repeated-run output-order determinism.

Every run must retain FTS-only, dense-only, hybrid, hybrid-without-FTS, and hybrid-without-dense arms under identical budgets. Exact-identifier cohorts protect FTS; semantic cohorts protect dense. A correct fused result does not excuse an empty, uninstrumented, or materially regressed lane.

## 8. Hard-Gate Promotion

Do not compute a weighted quality total. Every `PromotionResult` declares `scope=core_retrieval|release_candidate`, its applicable gates, and prerequisite result digests. Produce `PROMOTION_EVIDENCE_READY` only when every gate applicable to that scope passes; otherwise produce `NOT_PROMOTION_READY` with failed gates and first-loss/cohort evidence. A core-retrieval result from Phase 12 is not a release-candidate result and cannot silently imply assistant/host usefulness.

### 8.1 Zero/100% correctness gates

- all required query/arm observations exist exactly once;
- all corpus/query/profile manifests and artifact checksums validate;
- conformance fixtures have exact byte/span/body fidelity;
- no silent parser loss, stale deleted row, segment-parent misassociation, mixed generation/profile/codec, malformed payload, NaN, missing required trace field, or nondeterministic rank order;
- clean rebuild and incremental index are equivalent for the declared manifest;
- query-vector persistence violations are zero;
- f32 lab leakage into production storage is zero;
- required failures and timeouts are present in denominators;
- packaged bodies and coordinates match the indexed snapshot exactly.

Unsupported syntax may use a calibrated per-language coverage floor only after every exclusion has an explicit class; it may not be silently omitted.

### 8.2 Calibrated paired-margin gates

Freeze margins before confirmation for:

- Hit/Recall/MRR/NDCG and requirement coverage;
- exact-identifier, semantic, multi-answer, hard-negative, stale/dirty, and language cohorts;
- serving-f32 codec retention, relevance-gold retention, rank displacement, inversion, and ties;
- parent-collapse, RRF, and body-package gold survival;
- per-query regression count and maximum material-regressed cohort count;
- end-to-end task success, requirement coverage, false leads, and test results when product-usefulness evidence is required.

Establish margins by repeated frozen cidx baselines. Deterministic local retrieval should have zero repeat variance. Provider and assistant stages use paired repetitions and uncertainty reporting. Tune margins only on calibration data, freeze them, then run confirmation.

Do not copy thresholds from another corpus, embedding model, or decision. No knowledge-system threshold is a cidx default.

### 8.3 Operational observations before budgets freeze

Initially report, but do not pretend these are gates:

- stage and total p50/p95/p99 latency;
- corpus-size and segment-count scaling;
- SQLite, FTS, vector, and bytes-per-chunk size;
- peak memory and scan throughput;
- provider calls, tokens, retries, timeouts, and cost.

Before using any as a promotion gate, write a pre-result budget into the promotion contract and collect repeated baseline evidence.
No numeric performance SLA is a v1 default or implied by these observations.

## 9. Practices Prohibited

- Do not tune on confirmation data.
- Do not change dimension, codec, RRF, candidate limits, or body budget after viewing confirmation results.
- Do not publish only a mixed-language aggregate.
- Do not use provider benchmarks as cidx usefulness evidence.
- Do not compare runs with different corpus, labels, parser/tokenizer, limits, body budget, or query controls as a paired delta.
- Do not call f32-versus-codec retention ANN recall.
- Do not equate f32 rank retention with frozen source-backed relevance or frozen relevance with codec fidelity.
- Do not compare BM25, cosine, binary, int8, and RRF raw scores.
- Do not interpret cosine or RRF as probability/confidence.
- Do not treat profile activation as quality admission.
- Do not let fusion hide a broken lane.
- Do not count several segments from one parent as several relevant results.
- Do not label only one span when several implementations are valid.
- Do not accept generated no-answer labels without corpus-wide evidence and explicit agreement from both independent AI review passes.
- Do not average missing observations as zero or omit failed calls from latency/cost.
- Do not force assistant use of cidx.
- Do not treat a 12–20 query smoke set as promotion evidence.
- Do not use global mean improvement to hide a language or critical-cohort regression.

## 10. Run Artifacts and Observation Fields

Each immutable run directory contains, as applicable:

```text
corpus-manifest.json
query-contracts.jsonl
label-review.json
run-manifest.json
provider-usage.json
per-query-trace.jsonl
fts-candidates.jsonl
dense-segment-candidates.jsonl
collapsed-parent-candidates.jsonl
rrf-results.jsonl
inline-body-packages.jsonl
assistant-observations.jsonl
per-query-metrics.jsonl
aggregate-metrics.json
cohort-language-report.json
first-loss-report.json
implementation-audit.json
promotion-contract.json
promotion-result.json
report.md
artifact-checksums.json
```

A per-query trace records:

- query ID/hash, cohorts, language, answer mode, and required groups;
- corpus, label, code, parser, chunker, FTS, model, reducer, target, codec, profile, generation, and MCP-schema fingerprints;
- ephemeral query-vector hash, never vector bytes;
- per-lane candidate counts/cutoffs and candidate IDs, parents, native score/direction, rank, and relevance;
- collapse winner, parent mapping/rank, compression, and omission reason;
- RRF component ranks/contributions, fused score, tie group, and deterministic tie key;
- body bytes/tokens, source coordinates, completeness, and omission reason;
- first-loss state for every required group;
- assistant-used evidence and task outcome when observed;
- embedding, FTS, scan, collapse, fusion, body-load, serialization, and total timings;
- provider calls, tokens, attempts, retries, latency, and cost;
- error stage and terminal state.

Portable artifacts contain no absolute checkout path, source body by default, provider vector, credential, or live query text duplicated outside the user-managed dataset.

“Measurable per request” means a value exists in a response, trace, or run artifact. It is not continuous monitoring. Claim continuous monitoring only when durable collection, retention, rolling aggregation, threshold evaluation, and a visible report or notification path all exist. An opt-in local JSONL or SQLite sink is observation storage, not alerting by itself.

## 11. Phase Evidence Sequence

### Phases 00–02: contracts and instrumentation

Required evidence:

- query/label/run schemas;
- profile and codec fingerprints;
- complete first-loss and failure enums;
- deterministic artifact framing/checksums;
- calibration/confirmation split rules;
- a smoke trace containing every planned stage field.

### Phases 03–05: parser/chunker and index correctness

Required evidence:

- exact fixtures plus real-corpus reviewed slices for Go, TypeScript, and TSX;
- construct recall and emission precision;
- exact body/span fidelity;
- clean rebuild versus incremental equivalence;
- added/modified/renamed/deleted/stale and unsupported-syntax results;
- zero stale-row and generation-mixing violations.

### Phases 06–07: lexical baseline

Required evidence:

- frozen calibration and confirmation datasets;
- FTS-only per-language/cohort metrics and hard-negative behavior;
- parent survival and indexed chunk/body fidelity before packaging;
- first-loss report;
- simple baseline versus FTS comparison;
- existing-tools versus existing-tools-plus-lexical-cidx assistant comparison later in Phases 13–14 before claiming product usefulness.

This baseline is protected by all later dense and hybrid comparisons.

### Phase 08: source-f32 laboratory

Required evidence:

- exact 1024-dimensional document-f32 shape, finite-value, checksum, provenance, and raw-coverage evidence;
- cache-first restart evidence showing zero repeat API calls for durable hits;
- paid call/token/cost records for misses;
- production/lab isolation and no query-vector persistence;
- no target, codec, RRF, or quality-selection claim in this phase.

### Phases 09–10: int8 materialization

Required implementation evidence for each current config candidate:

- codec conformance and storage integrity;
- deterministic transform/codec/scorer conformance against serving-f32 fixtures;
- blob integrity, corruption rejection, and one-active-profile publication;
- repeated ranking-hash equality for controlled scorer fixtures;
- size, transform-time, memory, and zero-provider-call observations;
- no full frozen relevance-gold or cross-codec promotion claim before Phase 12.

No codec receives quality credit merely for smaller size. Full f32/int8 and frozen relevance-gold comparison belongs to Phase 12; Binary/256 are retired evidence.

### Phase 11: hybrid RRF

Required evidence:

- FTS-only, dense-only, union, hybrid, and both lane-ablation arms;
- lane contribution, survival, disagreement, rescue, harm, and rank movement;
- no broken-lane masking;
- fused package survival and deterministic tie behavior.

Phase 11 proves instrumentation and deterministic implementation behavior. Phase 12 owns promotion-capable per-language/cohort nonregression and frozen relevance-gold conclusions.

### Phase 12: confirmation and promotion evidence

Required evidence:

- frozen candidate and promotion contract before execution;
- independent confirmation dataset;
- complete per-query records for FTS, serving f32, the current active codec, union, collapse, RRF, body packaging, and lane ablations;
- same-run serving-f32 versus active-codec retention/displacement/inversion/tie metrics plus frozen relevance-gold metrics and authority;
- current int8 reports against serving-f32, with historical Binary/256 reports preserved but excluded from activation;
- complete hard-gate result;
- implementation invariant audit;
- `PROMOTION_EVIDENCE_READY` or `NOT_PROMOTION_READY`;
- `scope=core_retrieval`, with assistant/host gates explicitly outside this result rather than silently passed;
- selected config record that evaluation does not apply automatically.

### Phases 13–14: real MCP and packaging

Required evidence:

- actual tool schema, body budgets, coordinates, errors, and deterministic serialization;
- real assistant arms: existing tools, existing tools plus lexical cidx, existing tools plus hybrid cidx;
- paired marginal usefulness and false-lead evidence;
- post-activation integrity/readiness smoke checks.

Phase 14 writes a new immutable `scope=release_candidate` promotion result referencing the Phase 12 core-retrieval result and the Phase 13/14 evidence digests. It never rewrites the earlier result.

Post-activation checks prove the evaluated profile is serving; they do not replace offline evidence.

## 12. Advisory Provenance and Decision Log

This plan was reviewed in English with the referenced `@kb-metric` Codex task. The advisor summarized reusable evidence and promotion practices from its already-implemented dense/RRF evaluation work, while this document deliberately removed HNSW/ANN-specific criteria and adapted the remainder to cidx's single-profile, exhaustive-scan, auxiliary-MCP product boundary.

The 2026-08-14 advisory review used these `knowledge-system` sources as design references, without importing its corpus-specific thresholds:

- `docs/reference/rag-measurement-contract.md`
- `docs/architecture/06-evaluation-and-experimentation.md`
- `docs/architecture/05-retrieval-and-ann-hnsw.md`, only to identify and exclude ANN-specific metrics
- `internal/retrievaleval/metrics.go`
- `internal/retrievalpipeline/align.go`
- `internal/embeddingexperiment/runner.go`
- `cmd/knowledge-eval/analysis.go`
- `tools/evaluation-promotion-check/main.go`
- `docs/lessons/retrieval-system-lessons-learned.md`

Decisions:

- Use stage-separated scorecards and first-loss attribution; never a single weighted score.
- Use frozen source-backed relevance and exhaustive serving-f32 fidelity as separate references. Every usefulness artifact records its authority and the permanent absence of independent human validation.
- Preserve standalone FTS and dense evidence around RRF.
- Use paired hard gates with calibration/confirmation separation.
- Exclude HNSW construction, ANN recall, `ef_search`, graph health, and ANN tuning from cidx v1 evaluation.
- Evaluate cidx's marginal value beside existing assistant tools.
