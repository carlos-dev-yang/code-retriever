# cidx Retrieval Evaluation and Embedding Execution Plan

- Status: operational plan; no paid action or promotion evidence is authorized by this document
- Date: 2026-08-17
- Governing authority: [cidx v1 Evaluation and Promotion Contract](EVALUATION-CONTRACT.md)
- Owning phases: [Phase 07 lexical evaluation](07-lexical-evaluation.md), [Phase 12 retrieval evaluation](12-retrieval-evaluation.md), and the later assistant-use portion of [Phase 14](14-packaging-and-host-integration.md)

## 1. Purpose and authority

This document defines how to prepare the approved corpora and source-backed relevance labels, when document and query embeddings may be purchased, how candidate search profiles are calibrated, and how a frozen confirmation run produces or fails to produce `core_retrieval` evidence.

It is an execution companion to the canonical evaluation contract, not a replacement for it. If this document, an implementation phase, an adviser response, or a chat summary conflicts with the canonical contract, the canonical contract wins. This document does not itself authorize a paid embedding or official evaluation; the status/evidence ledger separately records operations that were approved and completed.

The plan was independently discussed with ChatGPT and Grok. Their common recommendations—provider-free preparation first, explicit spend gates, immutable paired runs, first-loss diagnosis, and limited claims from the current two repositories—are incorporated. Their advice is not ground truth. Section 16 records the material corrections made during reconciliation.

The current product-profile authority is
[`RETIRED-VECTOR-PROFILES.md`](RETIRED-VECTOR-PROFILES.md). Dated descriptions
of Binary and 256 experiments below are preserved provenance. They do not
authorize a current Binary/256 run or make those profiles selectable.

## 2. Current starting point

The user has approved these pinned open-source corpora for the current work:

- Go: [`go-chi/chi` v5.3.1 manifest](../../testdata/retrieval/corpora/go-chi-chi-v5.3.1.json)
- TypeScript and TSX: [`react-hook-form` v7.85.0 manifest](../../testdata/retrieval/corpora/react-hook-form-v7.85.0.json)

The original datasets are deliberately narrow machine-draft lexical smoke
inputs:

- [go-chi draft](../../testdata/retrieval/lexical-go-chi-v5.3.1-draft.json)
- [react-hook-form draft](../../testdata/retrieval/lexical-react-hook-form-v7.85.0-draft.json)

The current behavior-oriented calibration drafts are:

- [12-case chi behavior draft v3](../../testdata/retrieval/behavior-go-chi-v5.3.1-draft-v3.json)
- [20-case react-hook-form behavior draft v2](../../testdata/retrieval/behavior-react-hook-form-v7.85.0-draft-v2.json)

The behavior set comprises 12 Go, 12 TypeScript, and 8 TSX cases. It is bound
to exact generation-3 content hashes, qualified symbols, and byte ranges, but
remains draft and contains no reviewed hard-negative or confirmation case. The
original 12 smoke cases prove only that the provider-free corpus, indexing,
evaluator, artifact, and replay path works. Neither set proves semantic
retrieval quality or supports promotion. The current blockers remain:

- fresh current-profile blind pools and the two independent AI source-review passes required by `owner-adopted-dual-ai-v1` are incomplete;
- the deterministic simple-search baseline is frozen and measured for v2, but
  chi G12's source-corrected v3 wording requires its simple/opened-arm pool
  refresh before those passes begin;
- the canonical `voyage-code-4` model/price, complete product document source bank, and
  historical five-profile materializations are confirmed; the exact 12 chi + 20 RHF
  exploratory series completed once under its $0.01 ceiling and its approval
  is consumed; those immutable rankings were regrouped by cohort without a
  repeat, while the new chi v3 wording is not yet a measured query;
- the two repositories do not contain a genuine cross-language Go/TypeScript/TSX behavior path;
- the confirmation-size and review floor has not been satisfied.

### 2.1 Immediate chi/RHF execution slice

The user confirmed that the immediate goal is to close the chi and react-hook-form calibration cases first. Mixed-language corpus work and promotion confirmation are later stages and do not block the current document capture or exploratory calibration loop.

For the resumed immediate slice, the ordinary `1024 / int8` profile means:

```text
segment_target_bytes = 1024
source_dimensions = 1024
serving_dimensions = 1024
storage_codec = int8
```

This is the product default and the ordinary test profile. `512/int8` remains
an explicit supported compact option. Document-role 1024 f32 is durably stored
in the product source bank but never used as a serving fallback. The compatible
bank can rematerialize 1024/int8 or 512/int8 locally without another document call. Binary and 256
artifacts already measured in this plan are retained only as historical
evidence; reproducing either requires a new explicit approval and a dedicated
development evidence path.

The immediate workflow is:

| Step | Work | User confirmation |
| --- | --- | --- |
| A | Provider-free chi/RHF corpus, inclusion, parser, chunker, parent, and 1,024-byte segment audit | Only exceptions, exclusions, or ambiguous parent boundaries need a user decision |
| B | Author a versioned working cohort set from real code behavior; use the 12 identifier smoke cases only as reference | Review question intent, direct/support parents, answer mode, and task/signal assignment |
| C | Produce a no-network document-capture plan with exact fingerprints, pending inputs/bytes, request count, and spend ceiling | Explicit document-embedding approval is mandatory |
| D | Capture document f32 once and materialize the 1024/int8 working profile | No further choice if execution still matches the approved plan |
| E | Run bounded, explicitly approved exploratory query operations; inspect first loss and revise the working cohort direction | Approve each exact apply or one explicitly bounded series |
| F | Close the chi/RHF calibration cohort, perform independent ChatGPT/Grok blind-pool judgments, reconcile, and freeze under `OWNER_ADOPTED_DUAL_AI_REVIEW` | Adopt or reject the reconciled digest as owner |
| G | Select a genuine mixed-language corpus and author independent promotion confirmation | Deferred until chi/RHF closure |

Document embedding is intentionally early relative to final label freeze because it embeds the frozen corpus document inputs, not the evaluation labels. The long interval after capture is where provisional questions, dense candidate pools, first-loss traces, cohort balance, and judgments are tested. Changing those questions or labels does not invalidate compatible product source rows.

### 2.2 Development workspace versus inspected project

The production engine and the evaluation engine are the same code path. The
separation is in application assembly and storage ownership, not in ranking or
indexing behavior:

| Context | Source root | State root | Serving DB | Product source / evaluation state |
| --- | --- | --- | --- | --- |
| Normal project use | target Git project | `<source>/.cidx` | `<state>/db/index.db` | `<state>/db/embeddings.db`; search does not open it |
| cidx development: chi | `.cidx/test/corpora/chi` | `.cidx/test/states/chi` | `<state>/db/index.db` | `<state>/db/embeddings.db`, separate lab/evaluations |
| cidx development: RHF | `.cidx/test/corpora/react-hook-form` | `.cidx/test/states/react-hook-form` | `<state>/db/index.db` | `<state>/db/embeddings.db`, separate lab/evaluations |

All explicit development paths are relative to the controlling cidx Git
project and confined below these namespaces. Corpus bindings live in ignored
`.cidx/test/corpora.local.json`. Database metadata contains no absolute source
or state path, so a checkout can be replaced or the containing project moved
without invalidating compatible vectors. Commit/content/manifest/profile and
canonical-input hashes remain the compatibility authority.

The explicit development entry points are:

```text
cidx dev workspace <init|index|status> --source-dir .cidx/test/corpora/<name> --state-dir .cidx/test/states/<name>
cidx dev embeddings <capture|materialize> --source-dir .cidx/test/corpora/<name> --state-dir .cidx/test/states/<name> [...]
cidx dev retrieval evaluate --source-dir .cidx/test/corpora/<name> --state-dir .cidx/test/states/<name> --corpus-manifest <path> --dataset <path> [...]
```

The two directory arguments must be supplied together. Normal `cidx init`,
`index`, `status`, `serve`, and MCP calls do not accept this evaluation-state
override and continue to own only the target project's `.cidx` directory.

## 3. Fixed boundaries

The following are not calibration choices:

- local discovery, Tree-sitter parsing, indexing, FTS, dataset validation, and lexical evaluation are provider-free;
- eligible source files are at most 1 MiB;
- chunks are complete semantic AST parents, not configurable byte slices;
- embedding segments are AST-aligned and preserve an indivisible oversize statement or member;
- Voyage AI `voyage-code-4` is the only v1 provider/model;
- document and query source outputs are explicitly 1,024-dimensional float vectors with `truncation=false` and the correct `input_type`;
- paid document and paid query operations require separate explicit approval;
- raw document f32 storage is development-only and never a production dependency;
- query vectors are fresh, operation-scoped, reused only among comparable arms in that run, and never persisted;
- production has one active serving profile per project;
- frozen source-backed relevance and serving-f32 codec fidelity are separate references;
- no weighted total quality score is permitted;
- required failures stay in their declared denominators;
- confirmation cannot tune labels, settings, budgets, margins, or cohort definitions;
- Phase 12 can establish only `core_retrieval`; assistant-use and `release_candidate` remain later Phase 14 concerns.

## 4. Evaluation model

### 4.1 Retrieval path and first loss

Every required answer group is traced through this ordered survival path:

```text
source discovery
-> parse and semantic-parent construction
-> FTS and dense lane observations
-> FTS-or-dense provider union
-> dense segment-to-parent collapse, when applicable
-> parent-level RRF
-> inline-body packaging
-> optional assistant use and resolution
```

FTS and dense are parallel diagnostics. A dense segment miss is not an FTS miss, and a correct fused result does not excuse a broken lane. Once a required group is lost on the primary path, it cannot reappear in the trace. Assistant-use has a different truth unit and denominator and therefore is not appended mathematically to the retrieval survival score.

### 4.2 Parent and segment responsibilities

- FTS indexes and ranks semantic parents using symbol and body fields.
- Dense retrieval ranks AST-aligned embedding segments.
- Dense segment results collapse to unique semantic parents before fusion.
- RRF combines parent ranks; native BM25, f32, binary, int8, and RRF scores are never subtracted across scales.
- Inline-body limits change only returned source packaging, never result identity or rank.

## 5. Dataset organization

### 5.1 A complete slice per language, not a Cartesian product

Each supported language needs a complete evaluation slice. “Complete” means that the slice exercises the important task, signal, answerability, structure, and failure modes; it does not mean generating every possible combination of those axes.

Use the existing `EvaluationCase` fields and namespaced cohort tags instead of adding a second schema authority. The current validator accepts unique nonempty cohort strings but does not enforce the namespace or exactly-one rules below. Until that validation is implemented, a separate provider-free coverage preflight must enforce them before an official dataset can freeze:

- exactly one existing `split`: `calibration` or `confirmation`;
- exactly one existing `language`: `go`, `typescript`, `tsx`, or `mixed`;
- exactly one existing `answer_mode`: `SINGLE`, `BEST_N`, `EXHAUSTIVE`, or `ABSTAINABLE`;
- exactly one primary `task:*` cohort tag;
- exactly one `signal:lexical_anchor`, `signal:semantic_only`, or `signal:mixed` tag;
- zero or more `diag:*` tags;
- repository identity from the dataset and corpus manifest, not a duplicated case field.

Recommended primary task tags are:

- `task:symbol_path_or_literal_lookup`
- `task:single_parent_behavior`
- `task:delegated_or_cross_parent_flow`
- `task:interface_type_or_api_contract`
- `task:lifecycle_state_error_or_configuration`
- `task:multiple_implementations_or_exhaustive`

Recommended diagnostic tags cover declaration versus implementation versus incidental mention, ambiguous names, reviewed hard negatives, stale/renamed/deleted code, small versus large constructs, wrappers versus delegates, language-specific syntax, and genuine mixed-language behavior.

Exact identifier cases remain useful as a lexical and pipeline guard, but they must not dominate calibration or confirmation. Natural-language behavior, delegation, ambiguity, multi-parent, exhaustive, and hard-negative cases provide the semantic discrimination needed for dense and hybrid evaluation.

### 5.2 Draft, calibration, confirmation, and regression

| Data class | Purpose | May select settings? | May vote for promotion? |
| --- | --- | --- | --- |
| Draft/smoke authority | Validate corpus, provisional labels, evaluator, artifacts, and obvious truth mistakes; a draft review state may still use the calibration split | No | No |
| Calibration | Choose the profile, search policy, and paired margins | Yes | No |
| Confirmation | Test one frozen policy and contract | No | Yes, if every gate passes |
| Regression | Protect an already exposed confirmation set | No new tuning on that set | No new promotion vote |

The confirmation floor is fixed by the canonical contract:

- at least 90 answerable cases, with at least 30 each for Go, TypeScript, and TSX;
- at least 18 reviewed `ABSTAINABLE` or hard-negative cases, with at least 6 per language;
- at least 10 cases in every critical cohort; cohorts may overlap;
- a frozen explicit count of genuine mixed-language cases in addition to the per-language floors; a `mixed` case does not satisfy or triple-count toward the 30 Go, 30 TypeScript, or 30 TSX minimum;
- source inspection of every frozen case by two independent AI systems under `owner-adopted-dual-ai-v1`;
- whole-digest owner adoption, with `OWNER_ADOPTED_DUAL_AI_REVIEW` and permanent `NO_INDEPENDENT_HUMAN_REVIEW` recorded and propagated;
- corpus-wide evidence and agreement from both passes for every no-answer or hard-negative label.

Do not infer a 108-case unique denominator by adding 90 and 18. An answerable query may contain a reviewed hard-negative target, so `answerable_queries`, `abstainable_queries`, `hard_negative_queries`, and `answerable_with_hard_negative_queries` have distinct counts and an explicit intersection. Freeze whether mixed negative cases are required and their count before authoring. Each case contributes once to the global query denominator even when it contributes to one language, one task, one signal, and multiple diagnostic cohort reports.

The critical-cohort registry must state exactly which `task:*`, `signal:*`, and `diag:*` values are promotion-critical, their per-language counts, and their counting rule. Cover all applicable tasks and signals per language in aggregate; do not create nonsensical combinations merely to fill a task × signal × diagnostic Cartesian cell.

The calibration count is not a promotion threshold. Freeze its planned counts and coverage matrix before any official frozen-label calibration apply. Earlier explicitly labeled exploratory applies may revise the working set, but every revision creates a new draft digest and cannot vote for selection or promotion. Never fill formal confirmation cells after seeing candidate scores.

`ABSTAINABLE` does not define a server abstention threshold in cidx v1. A nonempty result list is not by itself a false positive. Measure returned-count diagnostics, reviewed `KnownHardNegativeHit@k`, and later assistant false leads; do not report an invented abstention-accuracy metric.

### 5.3 Judgment pooling without label leakage

Before final label freeze, reviewers must see the unique top candidates from
every arm opened for the selected calibration grid. The current user-selected
initial grid is 1,024 dimensions with int8 serving, so its pool consists of
simple search, FTS, exhaustive target-dimension f32, active int8, and their
applicable RRF arms. The supported compact 512/int8 option enters the pool only if it
is explicitly opened before freeze. Retired Binary and 256 evidence never
enters a current pool without a new approval. To avoid lane
and rank bias:

- combine and deduplicate candidates by semantic parent;
- pool to at least the deepest cutoff used by any formal metric;
- shuffle them with a recorded deterministic seed;
- hide lane identity, native score, and original rank during relevance review;
- preserve path, symbol, source range, source hash, and the minimum source span needed to judge the case;
- store grades `0`, `1`, and `2` plus required-group membership and rationale;
- keep pool-generation runs explicitly non-promotional.

Pooling creates an unavoidable paid-query preparation step for dense arms. A run bound to provisional labels cannot silently become an official calibration or confirmation run after labels change. Unless a future reviewed judgment-overlay contract is implemented, rerun the paid query operation after the final dataset digest is frozen.

Fresh formal query vectors can return a parent absent from the pre-label pool. Every returned parent inside a formal scoring cutoff must have a frozen judgment. Otherwise record the unjudged parent and mark any metric requiring that grade incomplete; never silently convert an unjudged result to grade `0`.

## 6. Stage-ordered execution

### Stage 0 — Freeze the experiment contract

Entry: this plan and the canonical evaluation contract are available.

Actions:

- declare the intended claim scope;
- freeze dataset axes, metrics, denominators, first-loss values, required arms, tie rules, candidate grid, and maximum spend per operation;
- record the proposed simple baseline in Section 7 and its later acceptance gate; it must freeze before official frozen-label lexical scoring, but it is not a prerequisite for the document-capture plan;
- separate parser/chunker correctness fixes from search-policy calibration;
- freeze the critical-cohort registry, per-language counts, case-counting rule, and explicit mixed answerable/negative counts.

Exit: a versioned experiment contract exists. No provider call is allowed before this exit gate.

### Stage 1 — Verify and freeze corpora

Entry: the user-approved manifests and the ignored relative
`.cidx/test/corpora.local.json` bindings exist.

Actions:

- verify upstream URL, commit, clean-tree rule, license, root subdirectory, language slice, include/exclude policy, selected file list, and expected content hash;
- verify index/file parity and that generated, dependency, ignored, or out-of-scope files do not enter the evaluation universe;
- keep machine-specific paths and source bodies out of tracked artifacts.

Exit: immutable corpus and inclusion fingerprints are recorded. A corpus change returns to this stage.

### Stage 2 — Provider-free parser, chunker, and segment conformance

Entry: corpus fingerprints are frozen.

Actions:

- audit discovery coverage and parse status per language;
- map expected source spans to complete semantic parents;
- verify parent IDs, symbol projections, source coordinates, and incremental-versus-clean equivalence;
- verify AST-aligned segmentation and segment-to-parent mappings;
- inspect distributions and oversize AST units for 768-, 1,024-, and 1,536-byte targets without calling the provider.

Exit: source, parent, projection, and segment manifests are stable enough to author truth. Any silent parse or mapping loss is an implementation blocker, not a score to tune around.

### Stage 3 — Author draft queries and initial truth

Entry: semantic-parent manifests are stable.

Actions:

- keep the current 12 identifier cases only as authoring and pipeline reference;
- add natural-language behavior, delegation, ambiguity, exhaustive, stale/change, and hard-negative cases by language slice;
- attach acceptable parent alternatives, required groups, relevance grades, answer modes, cohorts, and supporting source spans;
- deduplicate near-identical query intents before split assignment;
- create only the versioned chi/RHF working calibration set in the immediate slice;
- defer confirmation authoring until the cohort taxonomy and chi/RHF calibration direction are closed. Confirmation must then be independently authored from separate intents/source areas and must not reword individual calibration failures.

Exit: a reviewable working calibration dataset exists. Its questions, cohort assignments, and labels remain provisional until the exploratory loop and pooled review are complete.

### Stage 4 — Freeze and run the provider-free lexical preparation baseline

Entry: provisional truth passes schema, digest, and corpus-binding validation.

Actions:

- freeze the deterministic simple-search policy;
- run simple search and production FTS with identical parent inventory and return budgets;
- attribute source, parser/chunker, and FTS first loss;
- use results to expand the blind judgment pool, not to delete difficult queries;
- use the existing smoke results as diagnostics while the working behavior cohort is authored.

Exit: provider-free lexical preparation evidence is reproducible. It does not block the compatible document-capture plan. Official Phase 07 evidence remains blocked until the simple baseline and final pooled labels, including later dense candidates, are frozen.

### Stage 5 — Freeze the document input universe and request document approval

Entry: the corpus, parser/chunker version, canonical document text, model contract, and initial segment target are stable.

Before approval, produce a provider-free plan containing:

- corpus and canonical-input fingerprints;
- segment target and exact unique document-input count/bytes;
- already-covered, pending, and incompatible counts;
- provider/model/role/source-dimension/dtype/truncation settings;
- request grouping and retry policy;
- estimated tokens/cost and a hard maximum spend;
- the ignored product source-bank path and proof that serving `index.db` is not a source-f32 destination.

The first capture should use the canonical 1,024-byte target. Changing the segment target changes segment boundaries and therefore changes the document text universe. It is not a free local rematerialization. Evaluate 768 or 1,536 with provider calls only when provider-free structure evidence or 1,024-byte dense first-loss evidence justifies a separate, explicitly approved capture under a distinct compatibility fingerprint. Segment-target calibration is therefore the first, hierarchical calibration tier; it cannot be folded into the later zero-document-call dimension/codec grid.

Exit: the user explicitly approves the exact document capture operation, or the workflow stops with no network call.

### Stage 6 — Capture document f32 and materialize candidates

Entry: document approval is recorded and preflight still matches the approved fingerprint.

Actions:

```text
cidx dev embeddings capture --apply
select one candidate in .cidx/config.json
cidx index
cidx dev embeddings materialize --activate
```

- capture only missing compatible document inputs;
- require complete finite 1,024-dimensional f32 coverage before evaluation;
- materialize the initial 1,024-dimensional int8 working profile;
- retain the ability to derive the supported compact 512/int8 option locally later;
- evaluate one active serving profile at a time;
- never copy raw f32 into production storage.

For an approved alternative segment target, first select that target, run `cidx index` to establish its canonical input set, and capture it under its own approved source-profile/input fingerprint. Exact unchanged inputs may be reused only when the implementation proves their canonical hashes compatible.

Exit: the approved product source bank is complete and either supported int8 dimension for that segment target can be materialized without another document call.

### Stage 7 — Explore cohort direction, then freeze calibration labels

Entry: the supported target set, provisional calibration queries, source coverage, and local materializations are complete.

Actions:

- run the default evaluation command first to validate and estimate without network access;
- request a bounded paid-query approval for the exact exploratory operations;
- run the 1024/int8 working profile and inspect per-query lane results, first loss, parent collapse, RRF behavior, and body survival;
- compare active int8 only with exhaustive same-dimension f32 when fidelity evidence is required; do not fuse profile ranks;
- revise query wording, task/signal assignment, answer mode, expected alternatives, and cohort balance only in a new working-dataset version;
- repeat only when another separately approved apply or an approved bounded series remains in scope;
- once the cohort direction is stable, obtain blind candidate pools across the required arms;
- do not present aggregate candidate quality to labelers;
- complete both independent AI passes, source-backed reconciliation, and whole-digest owner adoption;
- freeze the calibration dataset digest;
- rerun deterministic simple search and FTS against that final digest; the provisional Stage 4 results are not the official frozen-label baselines.

Measured checkpoint (2026-08-17): the one-time codec diagnostic is complete
for both corpora. It kept all three scorers isolated, used 32 successful query
operations, and performed no RRF or document call. Int8 tracked exhaustive f32
far more closely than binary. The later product decision removed Binary from
current code. Label-only replay uses
the immutable rankings and does not repeat Voyage. See
[`evidence/phase-07/codec-top20-diagnostic-r4.md`](evidence/phase-07/codec-top20-diagnostic-r4.md).

The separately authorized 512- and 256-dimensional follow-ups are also
complete. Each transformed the existing 1,024-f32 document bank locally and
used one fresh query operation per case. Same-run int8 remained within `.9917`
to `1.0000` mean top-20 retention across both corpora and both dimensions. The
256 controls halve vector payload again but reduce the complete SQLite files
by only `7.12%` beyond 512 and show more shallow-rank variability. Because no
query vector is persisted and all fresh query hashes differ across checkpoints,
cross-dimension changes are not causal paired deltas. Carry 512/int8 as the
ordinary product profile; the 256 result is preserved only as retired evidence;
see the [512](evidence/phase-07/codec-int8-512-diagnostic-r4.md) and
[256](evidence/phase-07/codec-int8-256-diagnostic-r4.md) evidence.

The subsequent provider-free five-profile consolidation compares every current
language, signal, and task cohort plus the actual answer placements without
FTS, RRF, or codec-rank fusion. It advances 512/int8 as the next
frozen-evaluation candidate. The later product decision retired both the
Binary and 256 arms represented in that report. See the
[five-profile comparison](evidence/phase-07/five-profile-cohort-comparison-r4.md).

Exit: the chi/RHF cohort direction and calibration truth are frozen. Exploratory and pool-generation runs remain preparation evidence and do not vote for promotion.

### Stage 8 — Run paid calibration and select one policy

Entry: calibration labels, candidate grid, selection objective, tie rules, and spend cap are frozen.

For each current profile:

```text
cidx dev retrieval evaluate --corpus-manifest <path> --dataset <path>
# inspect provider-free preflight and estimate
cidx dev retrieval evaluate --corpus-manifest <path> --dataset <path> --apply
```

Actions:

- use a separate explicit paid-query approval for the exact calibration apply or explicitly bounded apply series;
- evaluate required arms with the same ephemeral query vector inside each run;
- compare serving f32 with the active codec in the same run;
- the opened development-only codec diagnostic is a valid paired comparison because one query f32, one document-f32 bank, one snapshot, and one parent-collapse policy feed all three independently scored arms; outside that exact pipeline, treat binary-versus-int8 cross-run deltas as paired only when all recorded controls, including query hashes, match;
- begin with the confirmed 1,024-byte segment, 1,024-dimension, binary profile. Open alternative segment/dimension/codec candidates only after the working cohort is stable and first-loss evidence justifies the added comparison/capture cost;
- then choose any opened candidate plus FTS weights, candidate limits, RRF `k`, body budget, and paired noninferiority margins;
- inspect every material regression by language, task cohort, signal class, and first loss;
- select one project-wide policy or declare `NOT_PROMOTION_READY`.

Exit: one policy and all confirmation gates/margins are frozen. No confirmation case has been used to tune them.

### Stage 9 — Prepare and freeze confirmation truth

Entry: the chi/RHF cohort/calibration work is closed, the policy is frozen, the user has separately approved a genuine mixed-language corpus, and independently authored confirmation intents meet the planned coverage floor.

Actions:

- run provider-free validation and FTS pooling;
- obtain separately approved, bounded paid-query pool-generation runs for exhaustive f32 reference, the selected int8 profile, and applicable RRF candidates; do not include retired Binary/256 evidence without a new explicit authorization and a separate non-product tool;
- blind and deduplicate the pool;
- finish both label passes, including corpus-wide negative evidence;
- verify that the frozen judgment pool covers every parent through the deepest formal scoring cutoff or declare the affected metric incomplete;
- seal queries, labels, cohorts, denominators, required arms, profile, body budget, margins, promotion scope, and artifact schema.

Label reviewers may adjudicate pooled code evidence but must not see confirmation aggregate scores or change the frozen policy. Any policy change creates a new confirmation version and requires new independently authored confirmation intents that have not been scored or used for tuning. Those new intents may undergo their own blinded pre-label pooling.

Exit: the confirmation dataset and `promotion-contract.json` are immutable.

### Stage 10 — Run formal confirmation scoring once

Entry: the frozen confirmation digest has not been used to calculate aggregate quality or tune policy, all zero/100% correctness gates pass preflight, and the user separately approves the exact paid confirmation-query operation. The query intents were necessarily used earlier for blinded candidate pooling; that labeling-only use is recorded and cannot expose aggregate scores or influence the selected policy.

Actions:

- record `confirmation_pool_generation` as non-promotional preparation and `formal_confirmation_apply` as the one scored confirmation invocation;
- run every required observation exactly once under one declared current profile;
- execute the frozen simple-search and FTS observations inside the sealed formal workflow; pre-label pooling results are not their official confirmation evidence;
- keep a required failed query in the retrieval denominator with the contracted zero retrieval outcome and explicit `OPERATION_FAILURE:<stage>` state, while counting its actual attempts only in operational denominators;
- write immutable artifacts atomically and validate checksums;
- apply the frozen per-language, per-cohort, hard-negative, representation-fidelity, collapse, fusion, and body-survival gates;
- record `PROMOTION_EVIDENCE_READY`, `NOT_PROMOTION_READY`, or an invalid/incomplete run state without changing thresholds.

Exit: a complete Phase 12 `core_retrieval` result exists, or the workflow stops with explicit failed gates and first-loss evidence. The exposed confirmation set becomes regression-only.

### Stage 11 — Optional assistant and release-candidate evidence

Entry: a compatible `core_retrieval` result exists.

Assistant tasks, controls, host integration, false leads, requirement coverage, and paired product-usefulness evidence are frozen and evaluated separately under Phase 14. Retrieval success alone never implies `release_candidate` readiness.

## 7. Deterministic simple-search frozen control

On 2026-08-16 the user accepted the deliberately weak, language-neutral v1
simple-search policy below. It is implemented, fingerprinted, and measured as
an evaluation-only control:

1. Search the same frozen semantic-parent inventory used by FTS.
2. Normalize query, path, symbol, signature, and body using one versioned identifier/text normalizer.
3. Require literal normalized query-token presence; do not use BM25, learned weights, embeddings, language-conditioned rules, per-query exceptions, or corpus-tuned boosts.
4. Rank by exact qualified-symbol match, exact symbol match, path match, then count of distinct matched query tokens.
5. Break all remaining ties by normalized path, parent start byte, and stable parent identity.
6. Record returned-count behavior and exact algorithm fingerprint.

For executable precision, the accepted v1 baseline defines those terms as
follows:

- query tokens are the stable-deduplicated union returned by the existing
  `symbol.ClassifyQuery` and `identifier-split-lower-v1` normalizer;
- each parent field is tokenized with that same normalizer, and candidate
  admission means at least one query token occurs in the union of normalized
  path, symbol, qualified-symbol, signature, and exact parent-body tokens;
- exact qualified-symbol and exact symbol compare the fully normalized query
  string with the fully normalized field; path match means at least one query
  token occurs in the normalized path-token set;
- distinct matched-token count is computed once across the union of all five
  fields, not added once per field;
- the final stable identity order is normalized path, start byte, end byte,
  raw qualified symbol, indexed content hash, and raw UTF-8 path byte order;
- the policy fingerprint covers the algorithm version, normalizer ID, field
  list, `ANY` admission rule, rank tuple, and tie tuple.

Implementation remains development/evaluation-only: one generation-pinned,
read-only production-store snapshot supplies the authoritative parents and
exact stored bodies to an internal evaluator. It does not change the public
lexical searcher, MCP search, database schema, FTS weights, or production
ranking.

The clean draft-v2 chi and RHF control artifacts are accepted preparation
evidence, not official promotion evidence. Chi G12's source-corrected draft-v3
query changes its dataset digest, so its provider-free simple result and
opened-arm pool must be refreshed from clean provenance before the dual-AI
label review. The algorithm itself is not reopened by that refresh.

## 8. Required retrieval arms

Every compatible Phase 12 run records, as applicable:

1. deterministic simple-search baseline;
2. FTS-only;
3. serving-f32 dense segment candidates;
4. serving-f32 unique parents after collapse;
5. active-codec dense segment candidates and unique parents;
6. FTS-or-dense provider union;
7. FTS plus serving-f32 parent-level RRF;
8. FTS plus active-codec parent-level RRF;
9. hybrid without FTS;
10. hybrid without dense;
11. inline-body packaging and body-survival diagnostics;
12. optional assistant-use only in the later product-usefulness evaluation.

Production has one int8 current profile. Historical multi-codec diagnostics
remain immutable reports only and are not executable arms in the current
program. A correct full hybrid result cannot hide a failed FTS lane, dense
lane, collapse stage, or package stage.

## 9. First-loss diagnosis and permitted action

| First loss or failure | Diagnose | Permitted during calibration | Confirmation response |
| --- | --- | --- | --- |
| Source discovery | Manifest, inclusion rules, file/hash parity | Fix and version the corpus manifest | Invalidate the run; new corpus version |
| Parse/chunk | Grammar coverage, semantic-parent boundary, projection/span mapping | Fix implementation and regenerate truth mappings; recapture documents if canonical text changes | Invalidate; do not tune retrieval around it |
| FTS candidate miss | Tokenization, symbol/body fields, weights, candidate depth | Tune only declared FTS candidates; version tokenizer/schema changes | No tuning; report failed gate |
| Dense segment miss | Segment relevance, source-model result, serving dimension, depth | Tune dimension/depth; change segment target only with a new approved document universe | No tuning; report failed gate |
| Provider union miss | Neither lane contains a required parent | Improve an upstream lane; RRF or body budget cannot repair absence | `NOT_PROMOTION_READY` when a frozen gate fails |
| Segment-to-parent collapse | Mapping, winner selection, parent dominance or suppression | Fix implementation; this is not a quality knob | Invalidate if correctness is broken |
| RRF fusion | Lane ranks, candidate budgets, `rrf_k`, tie order | Tune only the frozen candidate grid | No tuning; report rescue/harm and failed gate |
| Body packaging | Result survived ranking but required source did not fit/resolve | Tune body budget during calibration; rank and identity must stay fixed | No tuning; report body loss |
| Query/provider operation | Timeout, response validation, retry exhaustion, cancellation | Fix operations and rerun; retain failures in denominators | Incomplete/failed as contracted; never drop the case |
| Assistant use | Tool selection, source use, false lead, task outcome | Tune only in a separately declared assistant experiment | Does not rewrite core retrieval evidence |

When exhaustive serving-dimension f32 misses frozen relevance truth, int8 quantization cannot repair the model's semantic miss. When the provider union misses, fusion cannot manufacture the answer. When confirmation fails, lower thresholds, removed cases, relabeled cohorts, changed margins, and post-hoc budgets are forbidden; the result is `NOT_PROMOTION_READY` or an invalid run.

## 10. Embedding execution ledger

### 10.1 Provider-free operations

| Operation | Provider call | Reusable output |
| --- | --- | --- |
| Corpus verification and hashing | No | Corpus manifest/fingerprint |
| Tree-sitter parse, parent chunks, projections, segments | No | Generation and canonical-input manifests |
| Index and FTS | No | Production lexical index |
| Dataset/schema/digest validation | No | Validated data contract |
| Lexical/simple/FTS evaluation | No | Immutable lexical artifacts |
| Evaluation preflight and cost estimate without `--apply` | No | Plan only |
| Serving-dimension reduction, normalization, int8 encoding | No | Candidate serving vectors |
| Dense scan, collapse, RRF, body packaging, metrics | No additional call after query vector exists | Run artifacts |

### 10.2 Paid document operation

One explicitly approved capture operation may use many synchronous HTTP requests under the fixed grouping, concurrency, timeout, and retry policy. “One operation” does not mean one literal request.

The compatible product source key includes at least:

- corpus and selected-file/content fingerprints;
- parser/chunker and canonical document-text framing;
- segment target, boundaries, and exact canonical input set;
- provider/model, `input_type=document`, source dimension, dtype, and truncation policy.

Only missing inputs under the same compatible key may resume into that bank. A complete bank can be rematerialized locally into default 1024/int8 or supported compact 512/int8 without another document call. Historical Binary/256 reproduction remains outside the product and requires a separately approved standalone evidence tool.

Keep the three unrelated size fields explicit in every plan and manifest:

```text
segment_target_bytes = 1024
source_dimensions = 1024
serving_dimensions = 1024 | 512
```

Cross-segment-target results use different canonical input universes and do not share an ordinary same-bank serving-f32 fidelity reference.

### 10.3 Paid query operations

Each applied pool-generation, calibration, or confirmation invocation is a separately approved evaluation operation unless the user explicitly approves a bounded series with exact manifests, profiles, case counts, and maximum spend. Within an invocation, every planned case contributes exactly one logical query operation; that logical operation may create an initial provider attempt plus retries. Prior document approval never implies query approval.

Within one run, one query source vector is reused across compatible f32, codec, union, collapse, RRF, and ablation arms. Only its hash is persisted. A process failure may require another paid query request because vectors are intentionally not cached.

Record `logical_query_operations`, `provider_attempts`, `retries`, `validated_responses`, and `failed_attempts` separately. Per invocation, `logical_query_operations` equals the dataset case count, `retries = provider_attempts - logical_query_operations`, and `failed_attempts = provider_attempts - validated_responses`. Every logical operation has at most four attempts under the frozen policy and at most one terminal validated response.

Provider attempts are never a retrieval denominator. A classified required provider failure leaves provider-free FTS observed and gives each required vector-dependent outcome the contracted zero plus `OPERATION_FAILURE:<stage>`. `NOT_OBSERVED` is reserved for optional unrequested stages and unavailable operational facts, not for a required retrieval outcome. Missing or duplicate observations, digest drift, unclassifiable structural payload/artifact errors, cancellation that prevents completion, or checksum corruption make the run invalid/incomplete rather than an ordinary quality failure.

`observed_total_tokens` contains only provider-reported successful-response usage, `token_observed_attempts = validated_responses`, and accounting is incomplete after any failed attempt, including retry-then-success. Failed-attempt token usage, actual chosen backoff/`Retry-After`, per-attempt status/latency lineage, and input/output token splits remain `NOT_OBSERVED`; an observed provider zero is numeric `0`, and embedding generated-response tokens are `NOT_APPLICABLE`.

The series plan must also record dataset unique-query count, planned invocation/profile count, logical operations per invocation, total logical operations across the series, maximum provider attempts/spend, and actual attempts. The same query ID in separate profile pooling, calibration, and formal confirmation is a new logical operation each time; never deduplicate operational counts across runs.

### 10.4 Invalidation and recomputation matrix

| Change | Product document source bank | Local materialization | Query evaluation |
| --- | --- | --- | --- |
| Source file, include policy, corpus commit, canonical text | New/updated compatible document inputs required | Rebuild affected candidates | Rerun |
| Parser/chunker/projection change that alters canonical input | New/updated document inputs required | Rebuild | Rerun |
| Segment target or boundaries | Distinct document-input universe; new explicit capture required except exact reusable hashes | Rebuild | Rerun |
| Provider/model/document role/source dimension/dtype/truncation | Incompatible; new capture required | Rebuild | Rerun |
| Supported serving dimension, reducer, normalizer, int8 codec | Reuse compatible source rows | Recompute locally | Rerun for the new current profile |
| FTS weights, candidate/return limits, RRF `k`, body budget | Reuse | Reuse unless the serving key itself changes | Rerun; no document call |
| Query text | Reuse | Reuse | New dataset digest and paid query operation |
| Label grade/group/cohort only | Reuse | Reuse | New dataset version; formal comparable evidence must be regenerated |
| Retrieval implementation code with identical canonical inputs | Usually reuse, but paired-run code compatibility changes | Recompute if affected | Rerun |

Label changes never invalidate compatible document source rows. Conversely, a changed segment target cannot be treated as a local-only candidate change.

## 11. Profile selection and policy stability

Adding languages or questions must not continuously move the global project policy.

- One versioned calibration set proposes a challenger policy.
- Compare incumbent and challenger with compatible controls and immutable runs.
- Require aggregate, every-language, every-critical-cohort, hard-negative, correctness, and representation-fidelity gates; an aggregate cannot hide one language.
- Treat deterministic local rank variance as a correctness defect, not statistical noise.
- Freeze margins from repeated calibration baselines before confirmation.
- After exposure, confirmation becomes regression and never re-enters calibration.
- New questions enter a new dataset version; they do not retroactively change old thresholds or labels.
- Retire a case only for source invalidation, proven label correction, duplicate removal, or explicit contract evolution—not because it is difficult.
- If one language appears to need a different search policy, first diagnose discovery, grammar, parent boundaries, segment construction, and lane loss. Do not silently introduce language-conditioned search. That would require a new product contract.

For a new language, build a complete language slice covering the primary tasks, all three retrieval-signal classes where applicable, answerable and reviewed-negative behavior, structure sizes, and language-specific syntax. Then rerun the incumbent regression suite together with the new slice before proposing a new global policy.

## 12. Current corpus sufficiency and claim scope

`go-chi` plus `react-hook-form` is sufficient for:

- real Go, TypeScript, and TSX parser/chunker inspection;
- provider-free lexical smoke and initial source-backed label preparation;
- initial repository-specific dense and hybrid calibration after approval;
- detecting obvious parent, segment, codec, fusion, and packaging defects.

It is not sufficient by itself for a broad language-generalization claim:

- Go appears only in chi, while TypeScript/TSX appears only in react-hook-form, so language and repository/domain effects are confounded;
- the repositories are indexed separately and do not create a genuine cross-language answer path;
- TS and TSX must still be reported as separate slices even though they share a repository;
- cidx v1 has no basis here for an abstention or false-answer server claim.

Under the current canonical requirement for genuine mixed-language confirmation cases, chi plus react-hook-form alone cannot produce promotion-capable confirmation evidence. The user must explicitly select and approve a repository that contains real cross-language relationships searchable in one project. Do not manufacture mixed cases by combining unrelated results from separate repositories. A second repository per language is strongly recommended for broader robustness claims, but it is an expansion of claim scope, not evidence that may be acquired without user approval.

## 13. Required immutable artifacts

Each applicable run publishes the canonical artifact set:

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

The manifest and trace set must bind:

- corpus identity, upstream commit, selected-content hash, inclusion policy, and language slices;
- dataset digest, case counts, review records, grades, groups, cohorts, answer modes, and evidence spans;
- code commit and build version;
- parser, chunker, FTS schema/tokenizer, SQLite, generation, and canonical-input fingerprints;
- provider/model/source profile, source-bank coverage/fingerprint, reducer, serving dimension, codec, and active profile;
- candidate, collapse, RRF, body, tie, and MCP-schema policy;
- query-vector hash but never query-vector bytes;
- exact per-stage candidates, native ranks/scores, parent mapping, body survival, first loss, and failure state;
- denominators and exclusion reasons for every language and cohort;
- logical query operations, actual provider attempts, retries, validated/failed counts, nullable observed provider `total_tokens`, token-observed attempts, accounting completeness, failures, latency, and spend without inventing unavailable token data or executed backoff traces;
- commands, platform, timestamps, artifact checksums, completion state, and prerequisite promotion-result digests.

Portable artifacts must not contain credentials, absolute checkout paths, raw document/query vectors, or full source bodies by default. Query text remains in the user-managed dataset.

## 14. Spend and failure stop rules

Stop before a document call when:

- the corpus, canonical input, source profile, or estimate differs from approval;
- the source bank is already complete;
- the operation exceeds the approved input or spend cap;
- source-bank or evaluation-state destinations do not match the approved project-local state root;
- the segment target is not the approved document universe.

Stop before a query call when:

- corpus, dataset, label, profile, source coverage, candidate policy, or build fingerprints differ from the approved plan;
- current vectors require reconciliation or materialization;
- the operation type—pool generation, calibration, or confirmation—was not explicitly approved;
- the confirmation policy, margins, arms, or denominators are not sealed;
- the estimated operation exceeds its approved cap.

Stop comparison or promotion when:

- either run is incomplete or compatibility fingerprints do not permit the claimed pairing;
- a required observation is missing or duplicated;
- parser, parent mapping, vector coverage, profile, generation, codec, payload, persistence, body-coordinate, or deterministic-order correctness fails;
- confirmation data influenced tuning;
- any applicable frozen gate fails.

The honest terminal outcome is then an invalid/incomplete run or `NOT_PROMOTION_READY`, not a weaker denominator or revised threshold.

## 15. Exact next actions

Corpus audit, cohort authoring, document capture, exploratory comparison, and
the isolated 1,024/512/256 codec checkpoints are complete. The immediate
sequence is:

1. Preserve the five immutable profile rankings and the provider-free
   [combined comparison](evidence/phase-07/five-profile-cohort-comparison-r4.md).
2. Complete independently randomized ChatGPT and Grok source-review passes
   over the digest-bound pools, reconcile every grade-2/group conflict, apply
   the conservative dual-agreement support rule, and record whole-digest owner
   adoption with no independent-human claim.
3. Replay the immutable rankings provider-free after label changes; do not
   call Voyage merely to rescore labels.
4. Freeze the calibration dataset digest and rerun the deterministic simple
   and FTS controls against that final digest.
5. Seal the selected candidate, margins, denominators, confirmation questions,
   and approval packet before any formal confirmation operation.

Mixed-language corpus selection and promotion confirmation remain deliberately
deferred until chi/RHF calibration closure.

The earlier hybrid exploratory query series completed after compatible raw document coverage and local candidate materializations were proved; its one-series approval is consumed. The user separately approved and consumed the 1,024-, 512-, and final 256-dimensional 32-query codec-only series under the existing USD 5 billing cap and project-local credential boundary. Any later pool/calibration/confirmation apply requires its own approval. Formal confirmation approval is requested only after the selected policy, final pooled confirmation labels, margins, denominators, arms, and promotion contract are sealed.

## 16. Adviser reconciliation

Both side-panel advisers agreed on the staged shape, explicit paid boundaries, first-loss diagnosis, immutable evidence, per-language gates, regression stability, and the limited claim supported by the current corpora. The user-designated `kb-guide` then checked the draft's measurement semantics against the live schemas, validators, and accounting wire. The following advice was corrected or tightened to match the repository contract and live implementation:

- Final relevance-label freeze is not a prerequisite for document capture. Document capture depends on a frozen corpus, parser/chunker framing, canonical document input universe, and source embedding profile. Dense candidates are needed before final pooled label freeze.
- A label correction after confirmation exposure makes provider-free rescoring diagnostic/regression evidence only. It does not restore that confirmation set's promotion authority; a new unexposed confirmation unit is required.
- A 768/1,024/1,536 segment-target change is not automatically served by one source bank key set. It changes canonical segment text and requires a distinct compatible document capture except for exact reusable inputs. The live workflow does not define a cross-profile union bank.
- Label, cohort, or query changes do not invalidate compatible document vectors. They create a new dataset/query operation and invalidate direct evaluation deltas.
- Paid query embedding is required for dense pool generation and calibration before the serving policy is selected, not only after profile freeze. Formal confirmation still requires a separate later approval.
- Genuine mixed-language behavior must come from one searchable mixed-language project. It is not created by crossing independent chi and react-hook-form result sets.
- A second repository per language is a robustness recommendation for broader claims, not permission to select or acquire another corpus without the user.
- “Confirmation once” means one formal scored apply after freeze, not one lifetime provider submission of each confirmation query; separate blinded pool-generation operations are non-promotional and recorded independently.
- The current `cohorts` validator does not enforce the proposed `task:*`/`signal:*`/`diag:*` convention, so official freeze requires an explicit provider-free coverage gate until code enforcement exists.
- Answerable, abstainable, hard-negative, and answerable-with-hard-negative denominators may overlap; no 108-unique-case claim is implied by the 90 and 18 floors.

These corrections are normative for this operational plan because they follow the canonical repository contracts.
