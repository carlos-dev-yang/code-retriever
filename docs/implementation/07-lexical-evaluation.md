# 07. Lexical Chunking and Search Evaluation

- Status: `in_progress` — isolated 1,024-, 512-, and 256-dimensional f32/binary/int8 diagnostics are complete over the existing 12 chi + 20 RHF questions. The working `1024/binary` baseline remains unchanged; 512/int8 is the preferred compact candidate, 256/int8 is the memory-constrained alternative, and formal label freeze, calibration, confirmation, and promotion remain separately gated.
- Prerequisite phase: `06-fts-search`
- Follow-up phase: `12-retrieval-evaluation`
- Design basis: `local-code-search-mcp-v1-design-r4.md` §13, §14
- Evaluation authority: [EVALUATION-CONTRACT.md](EVALUATION-CONTRACT.md)
- Operational execution companion: [EVALUATION-EMBEDDING-EXECUTION-PLAN.md](EVALUATION-EMBEDDING-EXECUTION-PLAN.md)

## Context Recovery Checklist

- Reopen the [implementation index](README.md), [execution guide](EXECUTION-GUIDE.md), [evaluation contract](EVALUATION-CONTRACT.md), and [status ledger](STATUS.md) before continuing.
- Confirm the Phase 03/04 chunk inventory, Phase 05 deterministic generation and manifest, Phase 06 lexical runner and diagnostics, the reviewed evaluation-dataset schema, and user-selected corpus manifests plus their local bindings are available.
- Re-check these invariants after any context compaction: the user selects every open-source corpus; the tracked manifest pins identity, provenance, license, language slice, filters, and expected content; an ignored local binding or explicit CLI input alone supplies the checkout path; this phase never selects, downloads, or embeds a corpus absent exact user authorization; all compared runs record the same reproducibility inputs and report Go, TypeScript, TSX, and mixed denominators separately.
- Stop if a required corpus manifest or local binding is missing, the checkout is dirty or differs from its pinned commit/hash, license or redistribution scope is unclear, an expected target is absent, or generation/config changes would make the run non-reproducible.
- Before pausing, record executed evidence in §11, capture new architectural choices in §13, and update [STATUS.md](STATUS.md) with the exact next checklist item and unresolved stop condition.

## 1. Goal

Before comparing vector and hybrid results, record a reproducible baseline for how Go, TypeScript, and TSX chunking plus FTS-only search behave on real code-navigation questions.

This phase does not copy a passing score or SLA from another project. It builds the parser/chunker and FTS portions of the shared stage scorecard, freezes calibration/confirmation labels, records first loss, and establishes repeated cidx baselines from which later paired noninferiority margins can be chosen before confirmation.

## 2. Scope

### Included

- Tracked manifests for user-selected open-source evaluation corpora pinned to exact commits and expected content hashes.
- Ignored machine-local bindings, or explicit CLI inputs, that map corpus IDs to existing user checkout paths.
- Representative Go, TypeScript, and TSX queries with expected function, method, and type targets.
- Comparison of FTS-only search with simple identifier/text baselines.
- Observation of Hit@k, Recall@k, MRR, NDCG@k, requirement coverage, reviewed known-hard-negative Hit@k, returned-count behavior, stage survival, and first-loss categories.
- Review of chunk ranges, source bodies, and projection duplication.
- Reproducible run manifests containing config, profile, index manifest, and corpus provenance.
- Machine-readable results and human-readable summaries.
- Calibration and confirmation splits plus a dataset contract reusable by later raw-f32, vector, binary/int8, RRF, body-package, and assistant evaluation.

### Out of scope

- Selecting a corpus on the user's behalf.
- Downloading, cloning, updating, or embedding any corpus.
- Committing an environment-specific absolute checkout path.
- Declaring a minimum hit@k or MRR as a v1 gate.
- Declaring a maximum p50/p95 latency SLA.
- Production telemetry collection.
- Voyage API calls and paid embedding evaluation.
- Per-query hard-coded boosts tuned to evaluation outcomes.
- Creating a large public benchmark.
- Automated scoring by a generative-model judge.
- Treating an MCP host's final context-token use as a product guarantee.

The active Phase 07 diagnostic exception is a development-only, non-promotion
codec comparison over the already authorized chi/RHF raw document bank and
query set. It leaves the working `1024/binary` baseline unchanged. The original
checkpoint compares 1,024-dimensional target-f32/binary/int8; the authorized
follow-ups use separate development state, locally reduce the same 1,024-f32
document bank to 512 or 256 dimensions, and compare target-f32/binary/int8 at
the selected dimension. Each run reuses one ephemeral Voyage query f32 across
its three arms and captures isolated top-20 parent rankings without FTS or RRF.
Cross-dimension results are comparisons of separate checkpoints, not same-query
paired deltas, and no diagnostic opens another production serving profile.

## 3. Prerequisites

- Phase 03 and Phase 04 chunkers return exact source ranges and retrieval projections.
- Phase 05 can create a deterministic generation and manifest from a clean worktree.
- Phase 06 returns FTS-only ordinal ranks with diagnostics.
- The Phase 02 index-profile fingerprint and resolved serving config can be recorded.
- The user has selected each open-source evaluation corpus and supplied its tracked corpus manifest.
- Each `corpus_id` has either an ignored relative local binding in `.cidx/test/corpora.local.json` or an explicit CLI checkout-path input.
- Each bound local checkout can be verified against its manifest's pinned commit, clean-tree requirement, expected tree/content hash, license, language slices, root subdirectory, and include/exclude filters.

## 4. Invariants

1. Every lexical run being compared records the same corpus ID and pinned commit, query set, index profile, `candidate_k`, `return_k`, BM25 weights, and tie-break settings.
2. The tracked corpus manifest contains no environment-specific absolute local path.
3. A local binding maps only `corpus_id` to the user's checkout path and remains ignored; an explicit CLI path is runtime input and is not written into the tracked manifest.
4. Corpus identity is verified using the pinned commit, clean-tree state, and expected tree/content hash before an official baseline run.
5. This phase neither chooses nor downloads a corpus absent exact user authorization, and it never calls an embedding API.
6. A dirty worktree is never ignored. An official baseline fails when dirty; a diagnostic run must be marked explicitly as nonbaseline.
7. A human reviewer defines ground truth using source-chunk path, kind, and qualified symbol.
8. Numeric metrics are observations. This document does not create release pass/fail thresholds.
9. No-hit and wrong-hit outcomes retain causal categories rather than being hidden in one aggregate number.
10. Production must not receive per-query exceptions or large symbol boosts merely to improve evaluation results.
11. Vector comparisons reuse this lexical dataset and ground-truth definition, or record the reason and a new dataset version.
12. Result artifacts omit full source bodies by default and use path, symbol, range, and hash for reproduction.
13. Final token counts vary by tokenizer and host; they are supplementary observations rather than authority for byte limits or retrieval quality.
14. The evaluator follows `EVALUATION-CONTRACT.md`: stage denominators remain separate, required failures stay in denominators, calibration cannot vote for promotion, and confirmation cannot tune settings.
15. Human relevance truth is the product reference. Later codec fidelity uses serving-dimension f32 as a second independent reference; Phase 07 must not pre-label f32 results as truth.

## 5. Packages, Files, and Types to Implement

Keep the evaluator in development-only code outside production search packages.

```text
internal/
  eval/
    corpus_manifest.go        # tracked corpus provenance and content contract
    corpus_binding.go         # ignored local binding or explicit CLI path resolution
    ground_truth.go           # shared path/symbol relevance judgment
    metrics.go                # shared Hit/Recall/MRR/NDCG, requirement and survival metrics
    report.go                 # shared immutable run artifact
    lexical_runner.go         # clean local-corpus index plus free lexical queries
    lexical_diagnostics.go    # lexical error classification and inventory
testdata/
  retrieval/
    corpora/                  # tracked manifests only; no corpus source or local paths
    lexical-v1.jsonl          # reviewed query/answer dataset
    README.md                 # binding, corpus preparation, and annotation rules
.cidx/
  test/
    corpora.local.json        # ignored relative corpus_id-to-checkout bindings
    corpora/<corpus>/         # disposable approved checkout
    states/<corpus>/          # preserved config, DBs, and evaluation artifacts
```

Filenames may follow Phase 02 repository conventions, but development evaluation code must not enter the production MCP package.

Phase 07 imports Phase 02 `internal/evalcontract` query/run/trace/promotion types and validators. It owns lexical truth mapping, calculations, runners, and reports, not a second dataset or trace wire schema.

Recommended types:

```text
CorpusManifest
  CorpusID
  UpstreamURL
  PinnedCommit
  LicenseSPDX
  LanguageSlices[]
  ExpectedTreeOrContentHash
  RootSubdir
  Include[]
  Exclude[]

CorpusLocalBinding
  CorpusID
  CheckoutPath

EvaluationDataset
  Version
  CorpusID
  Cases[]

EvaluationCase
  ID
  Text
  LanguageSlice(go|typescript|tsx|mixed)
  Cohorts[]
  AnswerMode(SINGLE|BEST_N|EXHAUSTIVE|ABSTAINABLE)
  RequiredGroups[]
  HardNegatives[]
  RelevanceGrades
  ReviewPasses[] / DatasetSplit(calibration|confirmation)

RequiredGroup
  GroupID
  Alternatives[]              # OR within a group; AND across groups

ExpectedAlternative
  RelativePath / ContentHash
  Kind(function|method|type)
  QualifiedSymbol / SourceSpans[]
  RelevanceGrade(0|1|2)

EvaluationRunManifest
  DatasetVersion / CorpusID / CorpusCommit / CorpusContentHash / CleanWorktree
  BinaryVersion
  SchemaVersion
  IndexProfileFingerprint
  ActiveGeneration / ManifestSHA256
  Resolved lexical config
  StartedAt / FinishedAt

CaseEvaluation
  QueryID
  RankedHits
  FirstRelevantRank(optional)
  RequirementCoverage / CompleteRequirementHit
  StageSurvival / FirstLoss
  OutcomeCategory / FailureStage
  Diagnostics

EvaluationSummary
  Observed Hit@1 / Hit@5 / Recall@k / MRR / NDCG@k
  RequirementCoverage / CompleteRequirementHit / FirstLossCounts
  PerLanguageSlice summaries and case counts
  Metric denominators and excluded-case reasons
  No-hit and error counts
  Outcome category counts
  Qualitative review queue
```

## 6. Dataset, Corpus, API, and CLI Contract

### 6.1 Tracked corpus manifest and local binding

Use one reviewed tracked manifest per user-selected open-source corpus. It records only portable identity and selection rules:

- stable `corpus_id`;
- `upstream_url`;
- exact `pinned_commit`;
- license identifier or expression in SPDX form, plus any necessary notice reference;
- selected Go, TypeScript, and/or TSX language slices;
- expected clean-tree or selected-content hash;
- optional repository-relative `root_subdir`;
- deterministic include and exclude patterns.

Do not place a checkout path in this tracked manifest. Resolve `corpus_id` at execution time from one of two sources:

1. an ignored `.cidx/test/corpora.local.json` mapping maintained by the user, whose values stay below `.cidx/test/corpora/`; or
2. an explicit development-CLI checkout-path input for that run.

The resolver canonicalizes the supplied path, confirms it is an existing local checkout, and verifies the commit, clean-tree state, expected hash, root subdirectory, filters, language slice, and license metadata before indexing. It records the portable corpus identity and verified content hash in the run artifact, but does not copy the runtime absolute path into tracked artifacts or SQLite. A separate `.cidx/test/states/<corpus>/` root owns config, production/raw DBs, and artifacts while exercising the same production index/search services.

This phase consumes only checkouts the user selected and provided or explicitly authorized it to acquire by exact repository and commit. It does not recommend, choose, update, or embed any corpus.

### 6.2 Evaluation dataset

Use JSONL or an equivalent line-oriented schema. Every query follows the full label contract in `EVALUATION-CONTRACT.md` and has a unique stable ID. The set balances these categories:

- exact function, method, and type names;
- identifiers with different case or separators;
- partial qualified symbols;
- natural-language descriptions of implementation intent;
- concepts such as error handling, validation, and persistence that appear in multiple files;
- queries that must distinguish a type definition from method implementations;
- representative Go and TypeScript/TSX patterns; and
- negative queries reviewed as having no relevant result;
- path/package/module-qualified, literal/config-key, mixed identifier-semantic, multi-implementation, ambiguous-name, and declaration-versus-implementation cases; and
- added, modified, renamed, deleted, stale, small, and large construct cases.

These categories are coverage dimensions, not a quota that justifies artificial
microcases. Author representative code-search intents first. Retain a difficult
or edge case only when it distinguishes a material parser/chunker,
semantic-parent, language construct, codec, or retrieval failure that a normal
question would not expose. Do not fill cohort counts with increasingly narrow
wording that adds no new diagnostic information. Later confirmation floors are
minimum coverage requirements; independently authored realistic questions may
satisfy several floors when the cohorts genuinely overlap.

Every case records a `language_slice` of `go`, `typescript`, `tsx`, or `mixed`. A `mixed` case must genuinely allow or expect relevant results across language boundaries; it is not a substitute for an unknown label. Reports always show Go, TypeScript, and TSX separately when those slices are present. An aggregate may be included only with per-slice case counts and denominators so one language cannot hide another language's failure.

Represent valid alternatives as OR entries within a required group and separate mandatory requirements as AND groups. Use frozen relevance grades `0=wrong/stale/hard-negative`, `1=useful support`, and `2=direct requirement satisfaction`. Strict Hit/Recall/MRR use grade 2 or satisfied groups; NDCG uses all grades. Do not narrow answers after seeing results.

Create separate calibration and confirmation splits. A promotion-capable confirmation series starts with at least 90 answerable queries (30 each for Go, TypeScript, and TSX), 18 verified abstainable/hard-negative queries (6 each), and 10 cases in every critical cohort; cohorts may overlap. A 12–20 query set is smoke evidence only.

Every frozen query receives human review. Record two independent approvals when possible. Solo development records two separated review passes and the single-reviewer limitation. No-answer and hard-negative labels require corpus-wide search evidence and a second independent review/pass. Pool unique top results from simple search, FTS, and later f32/binary/int8/RRF arms before final label freeze.

Before retaining a difficult calibration question, record its real
`source_basis`, nearest existing query IDs, the distinct parser/chunker,
semantic-parent, language-construct, codec, or retrieval boundary it exercises,
the concrete distinguishing observation, and the diagnostic coverage lost if
the case is removed. Mark it `KEEP`, `RESERVE`, or `REJECT` with reviewer/pass
identity. Difficulty or a poor score alone is not sufficient to keep a case.

One case contributes once to the global query denominator, once to its language
slice, once to its single `task:*` cohort, once to its single `signal:*` cohort,
and to every genuinely applicable `diag:*` projection. Overlapping cohort
denominators are reported separately and are never summed to reconstruct the
global denominator.

Every pooled `(query_id, semantic_parent_id)` relation must receive two final
effective human judgments. Pass 1 remains machine-label-blind. Pass 2 or final
reconciliation may use a hash-bound batch approval plus exceptions instead of
individual clicks, but the record must enumerate every covered relation and
include packet digest, reviewer, timestamps, source-inspection attestation,
default decision, exceptions, and machine-overlay digest/provenance. The
materialized result still stores grade, rationale, required-group assignment,
and `source_verified` per relation. Compare complete pass maps and adjudicate
every disagreement; an uncovered relation is `UNREVIEWED`, never implicit
grade 0.

The final pool is the semantic-parent-deduplicated union, at the frozen pooling
depth, of every opened arm plus existing truth parents. Before label freeze,
require zero unreviewed relations in both passes, zero unresolved disagreements,
zero duplicate/conflicting judgments, zero missing source verifications, and
zero invalid grade-2 group assignments. Bind that proof to corpus, generation,
dataset, arm set, scoring depth, simple-policy fingerprint, shuffle seed, and
final pool digest. A previously unseen parent added by a later arm reopens both
passes for that relation. Int8 may remain explicitly unopened/out of scope for
the current 1,024/binary grid; it must not be silently described as pooled.

Metric calculations record denominators, negative-query handling, and failure states. Preflight-invalid data makes the run incomplete. A required query execution failure remains in the run denominator with zero retrieval metrics and an explicit operation-failure state; an optional unrequested downstream stage is `NOT_OBSERVED`.

Because v1 has no abstention threshold, an `ABSTAINABLE` query is not failed merely because top-k metadata is nonempty. Report returned count and rank/score diagnostics, count only reviewed misleading parents in `KnownHardNegativeHit@k`, and leave assistant false-lead measurement to the paired product-usefulness stage.

### 6.3 Development execution boundary

This phase implements the shared dataset/corpus loaders and free `LexicalEvaluationRunner`, but it does not own final development-CLI flag semantics. During implementation it may run through a Go package runner or temporary development harness. Phase 12 combines retrieval variants with the paid-query plan, and Phase 13 connects `cidx dev retrieval evaluate`.

The provider-free lexical and simple-control arms have no `--apply`. During
pre-freeze pool building, the already accepted Phase 12/13 development runner
may execute a separately authorized Voyage query embedding search. That path
must be explicit, non-promotable, and fully provenance-bound; it may not alter
production FTS, expose an experimental policy through public CLI/MCP, embed a
document, persist a query vector, or reuse historical query rankings. Each
execution creates a new immutable run artifact and never overwrites an earlier
result.

Any later CLI path argument is only a local binding for a manifest's existing `corpus_id`; it must not create an untracked corpus choice or bypass manifest verification.

### 6.4 Internal API and outputs

`EvaluationRunner` calls the production `IndexService` and `LexicalSearcher` directly. It must not reimplement chunking or FTS behavior for evaluation.

Each run creates at least two outputs:

- immutable JSON: run manifest, per-query ranks, metric source data, explicit denominators, and exclusion reasons;
- human-readable summary: representative successes, failure categories, and possible next adjustments.

Write every execution below a new run ID. Never overwrite or reuse an existing baseline artifact merely because the filename would match.

## 7. Config Used and Change Impact

The evaluator invents no defaults; it records the same resolved config snapshot used by production.

| Setting | Evaluation treatment |
| --- | --- |
| corpus manifest and local binding | verify portable manifest identity against the local checkout; never persist the machine path in tracked artifacts |
| tokenizer/symbol version | requires a new index generation; dataset version may remain unchanged |
| `candidate_k`, `return_k` | record exact values in the run manifest |
| BM25 field weights/tie-break | record in the run manifest and compare changes as separate variants |
| file-size and ignore rules | report differences between the corpus manifest and index manifest |
| segment target | record the selected 1024-byte target; only 768-, 1024-, and 1536-byte candidates may be compared in a separate frozen evaluation profile, always at AST boundaries |
| max inline bytes | does not affect rank evaluation; do not mix returned-body volume into lexical score |
| embedding source/vector/codec | does not affect the Phase 07 lexical run |

Select experimental variants through an explicit evaluation profile. Do not mutate global config during a run or pass hidden numeric values into search functions.

## 8. Ordered Implementation Checklist

1. Receive the user-selected open-source corpus manifests; do not select or download a corpus absent exact authorization.
2. Validate every tracked manifest's `corpus_id`, upstream URL, pinned commit, SPDX license, language slices, expected tree/content hash, root subdirectory, and include/exclude rules.
3. Resolve each `corpus_id` through the ignored local binding file or an explicit CLI input without writing the local path into tracked artifacts.
4. Verify that the bound checkout is clean and matches the pinned commit, expected content, filters, and declared language slice.
5. Run the Phase 05 index on that verified local checkout and pin its manifest.
6. Extract chunk inventory and manually review function, method, and type boundaries, duplication, and omissions.
7. Write query cohorts, OR/AND requirement annotation, relevance-grade, dual-review, and calibration/confirmation instructions.
8. Create the promotion-capable dataset floor or explicitly mark a smaller dataset as smoke-only; compare every expected alternative with an actual indexed chunk.
9. Implement corpus-manifest, local-binding, and dataset schema validators.
10. Implement the production `LexicalSearcher` adapter and a simple baseline adapter.
11. Calculate per-query ranks, requirement coverage, complete hit, stage survival, and first loss.
12. Aggregate Hit@1/5, Recall@k, MRR, NDCG@k, requirement coverage, known-hard-negative Hit@k, returned counts, and first loss by Go, TypeScript, TSX, and mixed slice, then optionally aggregate globally with explicit denominators.
13. Classify failures using the stable first-loss model, preserving FTS lane miss separately from source/chunk failures.
14. Report duplicate results and evidence of type/method duplicate indexing separately.
15. Record binary, schema, profile, config, generation, corpus ID, pinned commit, and verified content hash in the run manifest.
16. Generate JSON and human-readable reports.
17. Verify that comparisons before and after config changes display all condition differences explicitly.
18. Store artifacts immutably by run ID/fingerprint without overwriting an existing run.
19. Stabilize evaluator interfaces around the Phase 02 `evalcontract` types so Phases 08–12 consume the same dataset, case, requirement, trace, and artifact contracts.

## 9. Failure, Rollback, Concurrency, and Security

### Failure and rollback

- Fail before baseline creation for a missing manifest or binding, checkout/commit/hash mismatch, dirty official run, unclear license metadata, dataset-schema error, or absent expected target.
- A preflight-invalid run produces no comparable metrics. After a valid run starts, never drop a required query execution error: retain it in the denominator with zero retrieval outcomes and an explicit failure stage.
- If config, profile, generation, or verified corpus content changes during a run, do not approve it as a baseline; mark it `NON_REPRODUCIBLE_RUN`.
- Complete result files at a temporary path below the new run directory, then publish with atomic rename. Do not modify earlier runs.
- Report-generation failure must not change the production index.

### Concurrency

- An official baseline targets one fixed generation and may run queries sequentially or with bounded parallelism.
- Even with parallel execution, sort per-query results by query ID for a deterministic report.
- If reindex changes the active generation during evaluation, either keep the originally pinned generation snapshot or fail; never combine generations.
- The evaluator does not hold the writer lock for a long period.

### Security and data handling

- Validate that dataset and output paths cannot traverse outside the explicit evaluation root.
- Canonicalize the locally supplied checkout path, but do not place it in tracked manifests or portable results.
- Do not copy corpus source into this repository or include full source bodies in reports by default.
- Do not upload private-repository corpora externally.
- The Phase 07 lexical/simple paths neither read an API key nor access the
  network. A separately authorized pool-building retrieval invocation reads
  `VOYAGE_API_KEY` only after its provider-free plan, provenance, and cost gates
  pass; credentials never enter plans, logs, or artifacts.
- Queries may contain secrets, so logs retain only the dataset ID and the minimum necessary normalized diagnostics.
- Preserve upstream license notices and do not claim redistribution rights beyond the recorded license.

## 10. Validation Scenarios

1. Repeat a run with the same corpus, config, and dataset and obtain identical query ranks and summary.
2. Change the corpus commit, verified content hash, or index profile and observe the difference in the run manifest.
3. Detect a missing binding, wrong commit, dirty checkout, expected-hash mismatch, invalid filter/root subdirectory, and missing license metadata before indexing.
4. Confirm no tracked corpus manifest or portable artifact contains the environment-specific absolute checkout path.
5. Catch a typo or deleted expected target through schema and target validation.
6. For OR alternatives and AND requirement groups, calculate strict hit, requirement coverage, complete hit, and graded NDCG correctly without duplicate gain.
7. Distinguish an abstainable query, retrieval miss, required execution failure, and optional `NOT_OBSERVED` stage with explicit denominators.
8. Report a missing chunk and a displaced rank under different categories.
9. Surface duplicated type and method bodies in the inventory report.
10. Record complete settings for candidate/weight comparison runs.
11. Prevent dirty worktrees and mid-run generation changes from becoming official baselines.
12. Ensure a new execution never modifies an earlier immutable baseline artifact.
13. Complete lexical evaluation without an API key or network access.
14. Do not add an automatic SLA judgment when numeric metrics are low or change.
15. Confirm this phase did not choose, update, copy, or embed a corpus, and that any acquisition had exact user authorization.
16. Confirm an aggregate report cannot omit or merge away the Go, TypeScript, TSX, and mixed slice counts.
17. Confirm calibration results cannot update confirmation labels or margins and confirmation output cannot silently tune FTS weights or candidate limits.
18. Confirm exact repeat runs produce identical per-query ranking hashes and first-loss records.

## 11. Completion Evidence

- Reviewed tracked corpus-manifest schema and example containing portable provenance only.
- Ignored local-binding or explicit-input resolution record with no local path in tracked artifacts.
- Verification record for pinned commit, clean tree, expected content hash, root/filter selection, language slice, and license metadata.
- Versioned query contract with answer modes, OR/AND requirements, relevance grades, hard negatives, cohorts, review records, and calibration/confirmation split.
- Run manifest containing corpus commit, content hash, index manifest, and profile.
- Per-query stage trace and raw-rank JSON with requirement coverage and first loss.
- Observed Hit/Recall/MRR/NDCG, requirement coverage, known-hard-negative Hit@k, returned-count behavior, and first-loss summaries with explicit denominators for each language slice, plus any aggregate.
- Missing/duplicate review record from the chunk inventory.
- Comparison report for simple search versus FTS-only search.
- Repeatability record under identical conditions.
- Evidence that dirty/config-mismatch/generation-change runs cannot be approved as a baseline.
- Evidence that a new run does not overwrite an existing baseline artifact.
- Evidence that no corpus was selected, updated, copied, or embedded by this phase, and that any acquisition had exact user authorization.
- Dataset-size/review record distinguishing smoke, calibration, and promotion-capable confirmation evidence.

On 2026-08-16 the provider-free simple control ran from clean commit
`d343e12c36c2d17e40c00fe2fab445299f151715` with `VOYAGE_API_KEY` absent.
Its immutable draft-v2 artifacts record chi Hit@5 `5/12` and RHF Hit@5
`14/20`; these are diagnostic observations, not gates. The final blind pools
deduplicate the top five parents from FTS, serving f32, active binary, RRF, and
simple control plus every draft-v2 truth parent. They contain 133 chi and 175
RHF query-parent relations, with all labels and lane/rank/score information
hidden. Two separated human passes remain incomplete.

The later measured cohort review reuses the immutable Voyage embedding-search
rankings and clean simple artifacts without another provider request. Under
accepted draft-v2
truth, RHF T10 is complete because `PathImpl` and `PathInternal` are recorded
at ranks 2 and 3. The remaining binary failures are chi G07 and RHF T01/X01/X08;
all stay because they expose distinct multi-parent, orchestrator, wrapper, and
thin-type boundaries. No new question is added. Chi G12 alone moves to
`behavior-go-chi-v5.3.1-draft-v3.json` with wording that does not imply stable
iteration order across Go map keys. See
[`cohort-score-review-r4.md`](evidence/phase-07/cohort-score-review-r4.md).

On 2026-08-17 the provenance-bound comparator at clean commit
`70bbf1c3b67aa79eaaff4fba495ddbc4e805b6df` ran all 12 chi and 20 RHF
queries freshly through Voyage query embedding plus FTS, exhaustive target
f32, active binary, parent collapse, RRF, ablation, and body packaging. All
32 responses validated with zero retry/failure, 646 provider-reported tokens,
USD `0.00007752` accounted cost, zero document provider operations, and no
query-vector persistence. The run showed real fusion rescues and regressions;
OR fusion is rejected as a serving candidate and production AND is unchanged.

The refreshed blind unions contain 191 chi and 281 RHF relations. Two AI
reviewers independently covered every relation with rank, score, arm, prior
label, and experiment result hidden. These are supplementary advisory reviews,
not the two effective human judgments required above. Chi has a complete
reconciled advisory label map. RHF direct relevance is fully reconciled and
matches all 22 existing draft direct parents, while 101 grade-0/grade-1 support
differences remain deliberately separate. RHF NDCG is reported as the two
support-map endpoints and their min/max label-sensitivity range, not a
confidence interval or midpoint. Completeness and first loss continue to use
the unchanged draft required-group topology. The exact run, packet, pass,
reconciliation, and replay hashes are recorded in
[`measured-retrieval-loop-r4.md`](evidence/phase-07/measured-retrieval-loop-r4.md).

Formal Phase 07 completion is still blocked on the separated human source
passes. A human may adopt a digest-bound batch plus explicit exceptions, but
must cover every relation, attest source inspection, and leave zero unresolved
human disagreement. If only labels change, rescore the immutable rankings
provider-free; do not call Voyage again.

Completion reports must not use the metric value itself as a success declaration. They should demonstrate that the evaluation is reproducible and that failures are traceable.

## 12. Follow-up Handoff

Provide Phases 08–12 with:

- the user-selected, manifest-verified corpus IDs and query/answer dataset;
- lexical baseline rank source data;
- query categories and representative lexical failures;
- the evaluator contract used to compare f32, binary, and int8 at the same serving dimension; and
- the fixed `candidate_k`, `return_k`, and RRF input conditions used by hybrid comparison.
- frozen calibration/confirmation digests and the lexical baseline that later lanes must protect.
- the AI-advisory direct map and support-label sensitivity endpoints as review
  preparation only, never as frozen human truth or policy-selection evidence.

Embedding phases must not silently modify dataset answers. If a correction is required, increment the dataset version and record the annotation reason. They also must not infer corpus download or embedding authorization from the existence of a manifest or local binding.

Phase 12 extends the shared Phase 07 `internal/eval` dataset, ground-truth, metric, and report types. It must not create a second incompatible dataset schema or duplicate those types without an adapter.

## 13. Decision Log

| Decision | Reason | Revisit when |
| --- | --- | --- |
| Build a lexical baseline before retrieval comparisons. | This separates vector improvement from chunking defects; Phase 08 implementation may proceed in parallel. | Never for the Phase 12 prerequisite contract. |
| The user selects every open-source corpus. | Corpus suitability, licensing, checkout ownership, and download authorization belong to the user. | Product scope explicitly introduces managed corpora. |
| Separate portable tracked manifests from ignored local bindings. | Reproducibility needs pinned provenance while machine paths must not enter version control. | A dedicated portable workspace resolver is designed. |
| Never select, download, or embed a corpus in Phase 07 without explicit user authorization. | This phase is a free, local lexical evaluator and a manifest alone is not authorization for external actions or paid work. | A separate user-authorized acquisition or embedding workflow is designed. |
| Observe hit@k/MRR without a numeric gate. | The corpus and product usage pattern are not yet sufficiently settled. | Representative corpora and product requirements accumulate. |
| Use generative-model judgments only as supplementary blind-review preparation. | The user explicitly delegated sanitized public-corpus review to ChatGPT and Grok, but the evaluation contract still requires effective human judgments. AI outputs may expose ambiguity and produce sensitivity endpoints; they cannot freeze labels, select policy, or satisfy promotion gates. | The human-review authority contract is explicitly versioned and changed. |
| Use the accepted deterministic simple-search policy as an evaluation-only control. | On 2026-08-16 the user accepted a corpus-independent `ANY` normalized-token admission rule over the authoritative semantic-parent snapshot, followed by exact qualified/symbol, path, matched-token, and stable identity ordering. It adds no alias, BM25, embedding, boost, public wire, or production-ranking change. | A later evaluation-contract revision explicitly replaces the control. |
| Stop FTS micro-tuning and carry safe OR 5:1 only as a provenance-bound development comparator. | Repeated AND/OR, minimum-two-token, and 5:1/5:5 experiments showed that aggregate gains can conceal lost required Go/TSX parents. One coherent 32-query Voyage run now provides the missing f32, binary, fusion, and body evidence while production remains AND. | The blinded two-pass pool review is complete and a measured structural change is justified without cross-slice correctness regression. |
| Reject OR fusion as a serving candidate under the current draft evidence. | The coherent Voyage run showed lower complete-required@5 than pure dense for both corpora, with no chi rescue and concrete chi/RHF required-parent regressions despite isolated RHF rescues. Aggregate RHF also concealed a TypeScript decline and TSX rise. | Human-frozen labels plus a new structural fusion design show no required-group regression in every protected slice. |
| Report RHF support relevance as a two-endpoint NDCG sensitivity range. | The two complete blind AI passes agree on direct truth after reconciliation but differ on 101 subjective grade-0/grade-1 relations. A single forced label map or midpoint would manufacture precision; Hit/Recall/MRR use the reconciled direct map and completeness/first loss use unchanged draft groups. | Compliant human review resolves every support relation. |
| Do not repeat Voyage when only labels change. | The corpus, query texts, profiles, document bank, query vectors' resulting immutable ranks, and retrieval policies are already bound and complete; rescoring new labels is provider-free. | Any bound corpus, question, embedding, or retrieval-policy identity changes. |
| Open one paired f32/binary/int8 top-20 diagnostic before human freeze. | The user explicitly requested an int8 comparison and wider candidate review. One Voyage query vector per case is reused across all three isolated dense arms; candidate int8 documents are locally derived once from the existing raw bank, production remains binary, and no RRF enters the codec comparison. | The corpus, questions, serving dimension, transform, document bank, or codec implementation changes. |
| Measure one isolated 512-dimension int8 follow-up. | The user considers 1,024-dimensional int8 too large for the intended local footprint. Reuse the approved 1,024-f32 raw document bank and shared prefix-L2 transform in separate development state; request fresh query embeddings once because query vectors are not persisted. Compare 512-int8 only to same-run 512-f32 for codec fidelity, and report 512 versus 1,024 as separate checkpoints. | The bound corpus, questions, source model/dimension, transform, raw bank, or codec implementation changes. |
| Carry 512/int8 as the compact candidate without changing the working baseline. | The completed follow-up halved int8 vector payload, reduced the two complete SQLite files by `26.48%`, retained every direct answer by top 20, and preserved `1.0000` chi / `.9950` RHF same-run f32 top-20 membership. The 32 draft questions are not sufficient for production promotion. | Separated human review or confirmation evidence contradicts the result, or the bound model/transform/codec/profile changes. |
| Measure one final isolated 256-dimension int8 follow-up. | The user wants the smaller supported serving dimension measured before choosing between compact candidates. Reuse the approved 1,024-f32 raw document bank and shared prefix-L2 transform in separate development state; request each of the same 32 query embeddings once because query vectors are not persisted. Compare 256-int8 only to same-run 256-f32 for codec fidelity and report cross-dimension results as separate checkpoints. | The bound corpus, questions, source model/dimension, transform, raw bank, or codec implementation changes. |
| Keep 512/int8 preferred and retain 256/int8 as a memory-constrained alternative. | The completed 256 run preserved `.9917` chi / `.9950` RHF same-run f32 top-20 membership and every direct answer by top 20, but saved only another `7.12%` of the two complete SQLite files beyond 512 and showed less stable shallow placement, most clearly RHF T12. Fresh query hashes differ across checkpoints, so this is an observed practical choice rather than a causal paired dimension claim. | A frozen same-source-query comparison or confirmation evidence reverses the storage/quality tradeoff. |
| Retain int8 as the measured candidate without changing production binary. | The clean paired run showed int8 top-20 retention above `.992` on both corpora with zero top-1 mismatch against exhaustive f32. Binary retained `.7042` on chi and `.7575` on RHF and lost useful source-reviewed neighborhoods even when a raw hit metric improved. The codecs remain isolated and are never fused with one another. | Separated human review or a new confirmation corpus contradicts the calibration result, or the bound codec/transform/profile changes. |
| Prefer representative cohort intents and reject quota-padding edge cases. | The user wants questions that expose material failure modes, not detail added only to reach a count. Difficult cases remain valuable when they isolate a real parser, parent-collapse, type/wrapper, codec, or retrieval distinction. | New evidence shows a missing material failure mode that cannot be covered by a representative question. |
| Use measured cohort failures before revising questions; keep G07/T01/X01/X08 and narrow only G12. | Repeated advisory grading was slower and less decisive than the existing real rankings. The four misses each retain a distinct source-backed diagnostic boundary, while G12 alone contained wording broader than the Go source contract. | A new measured run or source change invalidates one of those distinct boundaries. |
| Accept the T10 and G09 source-backed label revisions. | `PathImpl` and `PathInternal` directly implement T10 while public `Path` is useful support; `walkXFF` is a reviewed misleading implementation for the deprecated G09 contract. | Pinned source identity or the question intent changes. |
| Use human-reviewed path/kind/symbol targets as ground truth. | These map directly to the code-search unit. | Multi-hop task evaluation is added separately. |
| Preserve failure taxonomy alongside metrics. | It identifies which implementation layer should change next. | Never. |
| Call production services directly. | This prevents divergence between evaluation and actual behavior. | Never. |
| Expose evaluation only through development surfaces. | Keep the public user and MCP surfaces small and stable. | Productization is explicitly requested. |
| Use the shared stage scorecard and first-loss model. | A final Hit@5 cannot prove parser correctness or explain where evidence disappeared. | Never. |
| Freeze calibration separately from confirmation. | Parameter tuning and promotion evidence cannot come from the same queries. | A new versioned evaluation policy is approved. |
| Keep required failures in denominators. | Dropping failed observations inflates retrieval and operational metrics. | Never. |
| Acquire only the two explicitly user-authorized public checkouts for this resume. | The user named commits, versions, licenses, and allowed local acquisition; this is a narrowly recorded exception to the default no-download rule. | Any corpus identity, commit, or authorization changes. |
| Resume Phase 07 with real-data structural audit and behavior-cohort authoring before paid work. | The user prioritized measured corpus behavior over additional test scaffolding and fixed the first working profile to 1,024-byte segments, 1,024 serving dimensions, and binary storage. | The structural audit finds a canonical-input defect or the exact document-capture plan reaches its explicit approval gate. |
