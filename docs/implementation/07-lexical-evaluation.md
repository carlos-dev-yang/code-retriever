# 07. Lexical Chunking and Search Evaluation

- Status: `blocked` — generation-3 chi/RHF audit, 32-case draft cohort binding, clean FTS controls, and official `voyage-code-4` price verification are complete; document capture waits for explicit bounded user approval, while official frozen evidence remains gated on later pooled label review and the simple-search policy.
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

## 3. Prerequisites

- Phase 03 and Phase 04 chunkers return exact source ranges and retrieval projections.
- Phase 05 can create a deterministic generation and manifest from a clean worktree.
- Phase 06 returns FTS-only ordinal ranks with diagnostics.
- The Phase 02 index-profile fingerprint and resolved serving config can be recorded.
- The user has selected each open-source evaluation corpus and supplied its tracked corpus manifest.
- Each `corpus_id` has either an ignored local binding in `.cidx/lab/corpora.local.json` or an explicit CLI checkout-path input.
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
  lab/
    corpora.local.json        # ignored corpus_id-to-checkout-path bindings
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

1. an ignored `.cidx/lab/corpora.local.json` mapping maintained by the user; or
2. an explicit development-CLI checkout-path input for that run.

The resolver canonicalizes the supplied path, confirms it is an existing local checkout, and verifies the commit, clean-tree state, expected hash, root subdirectory, filters, language slice, and license metadata before indexing. It records the portable corpus identity and verified content hash in the run artifact, but does not copy the environment-specific absolute path into tracked artifacts.

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

Every case records a `language_slice` of `go`, `typescript`, `tsx`, or `mixed`. A `mixed` case must genuinely allow or expect relevant results across language boundaries; it is not a substitute for an unknown label. Reports always show Go, TypeScript, and TSX separately when those slices are present. An aggregate may be included only with per-slice case counts and denominators so one language cannot hide another language's failure.

Represent valid alternatives as OR entries within a required group and separate mandatory requirements as AND groups. Use frozen relevance grades `0=wrong/stale/hard-negative`, `1=useful support`, and `2=direct requirement satisfaction`. Strict Hit/Recall/MRR use grade 2 or satisfied groups; NDCG uses all grades. Do not narrow answers after seeing results.

Create separate calibration and confirmation splits. A promotion-capable confirmation series starts with at least 90 answerable queries (30 each for Go, TypeScript, and TSX), 18 verified abstainable/hard-negative queries (6 each), and 10 cases in every critical cohort; cohorts may overlap. A 12–20 query set is smoke evidence only.

Every frozen query receives human review. Record two independent approvals when possible. Solo development records two separated review passes and the single-reviewer limitation. No-answer and hard-negative labels require corpus-wide search evidence and a second independent review/pass. Pool unique top results from simple search, FTS, and later f32/binary/int8/RRF arms before final label freeze.

Metric calculations record denominators, negative-query handling, and failure states. Preflight-invalid data makes the run incomplete. A required query execution failure remains in the run denominator with zero retrieval metrics and an explicit operation-failure state; an optional unrequested downstream stage is `NOT_OBSERVED`.

Because v1 has no abstention threshold, an `ABSTAINABLE` query is not failed merely because top-k metadata is nonempty. Report returned count and rank/score diagnostics, count only reviewed misleading parents in `KnownHardNegativeHit@k`, and leave assistant false-lead measurement to the paired product-usefulness stage.

### 6.3 Development execution boundary

This phase implements the shared dataset/corpus loaders and free `LexicalEvaluationRunner`, but it does not own final development-CLI flag semantics. During implementation it may run through a Go package runner or temporary development harness. Phase 12 combines retrieval variants with the paid-query plan, and Phase 13 connects `cidx dev retrieval evaluate`.

There is no `--apply` in this phase. Each execution creates a new immutable run artifact with fingerprints and never overwrites an existing result. Adoption of a run as a comparison baseline is recorded in artifacts and decisions; v1 does not add a separate CLI contract for updating an approval pointer.

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
- Phase 07 neither reads an API key nor accesses the network.
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

Completion reports must not use the metric value itself as a success declaration. They should demonstrate that the evaluation is reproducible and that failures are traceable.

## 12. Follow-up Handoff

Provide Phases 08–12 with:

- the user-selected, manifest-verified corpus IDs and query/answer dataset;
- lexical baseline rank source data;
- query categories and representative lexical failures;
- the evaluator contract used to compare f32, binary, and int8 at the same serving dimension; and
- the fixed `candidate_k`, `return_k`, and RRF input conditions used by hybrid comparison.
- frozen calibration/confirmation digests and the lexical baseline that later lanes must protect.

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
| Do not use a generative-model judge. | Avoid cost, nondeterminism, and mixing answer definition into scoring. | It is separately designed as supplementary evaluation. |
| Defer the simple-search baseline policy. | The phase requires a deterministic baseline but does not define its ranking policy; infrastructure must not invent corpus-tuned behavior. | A reviewed corpus and baseline policy are supplied. |
| Use human-reviewed path/kind/symbol targets as ground truth. | These map directly to the code-search unit. | Multi-hop task evaluation is added separately. |
| Preserve failure taxonomy alongside metrics. | It identifies which implementation layer should change next. | Never. |
| Call production services directly. | This prevents divergence between evaluation and actual behavior. | Never. |
| Expose evaluation only through development surfaces. | Keep the public user and MCP surfaces small and stable. | Productization is explicitly requested. |
| Use the shared stage scorecard and first-loss model. | A final Hit@5 cannot prove parser correctness or explain where evidence disappeared. | Never. |
| Freeze calibration separately from confirmation. | Parameter tuning and promotion evidence cannot come from the same queries. | A new versioned evaluation policy is approved. |
| Keep required failures in denominators. | Dropping failed observations inflates retrieval and operational metrics. | Never. |
| Acquire only the two explicitly user-authorized public checkouts for this resume. | The user named commits, versions, licenses, and allowed local acquisition; this is a narrowly recorded exception to the default no-download rule. | Any corpus identity, commit, or authorization changes. |
| Resume Phase 07 with real-data structural audit and behavior-cohort authoring before paid work. | The user prioritized measured corpus behavior over additional test scaffolding and fixed the first working profile to 1,024-byte segments, 1,024 serving dimensions, and binary storage. | The structural audit finds a canonical-input defect or the exact document-capture plan reaches its explicit approval gate. |
