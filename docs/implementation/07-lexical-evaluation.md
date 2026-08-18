# 07. Lexical Chunking and Search Evaluation

- Status: `in_progress` — the chi/RHF 32-case calibration set is frozen under
  `owner-adopted-dual-ai-v1`, its provider-free replay is accepted, and a
  separate unexposed promotion-capable confirmation set remains outstanding
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

## 2026-08-17 product-profile supersession

The five-profile report is decision evidence, not the current test matrix.
Current ordinary tests use 1024/int8. Supported compact 512/int8 may appear
only in a frozen declared arm. Document source f32 is durable product state;
query/reference f32 remains non-serving. Binary and 256 reports and local
states are preserved historical evidence, but current code has no route that
selects them. See
[`RETIRED-VECTOR-PROFILES.md`](RETIRED-VECTOR-PROFILES.md).

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
- Calibration and confirmation splits plus a dataset contract reusable by later source-f32, int8 vector, RRF, body-package, and assistant evaluation.

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

The completed five-profile codec diagnostic is immutable historical evidence.
It compared target-f32, Binary, and int8 arms at dimensions that included 256,
but it no longer authorizes an executable diagnostic path. Current Phase 07
resumes with one default 1024/int8 product run; an explicit frozen 512/int8 arm
may reuse the same product source bank. Both may use an in-memory serving-f32
reference, and no current run can derive, activate, or score Binary/256.

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
7. Frozen ground truth uses source-chunk path, kind, and qualified symbol under the recorded `OWNER_ADOPTED_DUAL_AI_REVIEW` authority; it is never described as human-reviewed.
8. Numeric metrics are observations. This document does not create release pass/fail thresholds.
9. No-hit and wrong-hit outcomes retain causal categories rather than being hidden in one aggregate number.
10. Production must not receive per-query exceptions or large symbol boosts merely to improve evaluation results.
11. Vector comparisons reuse this lexical dataset and ground-truth definition, or record the reason and a new dataset version.
12. Result artifacts omit full source bodies by default and use path, symbol, range, and hash for reproduction.
13. Final token counts vary by tokenizer and host; they are supplementary observations rather than authority for byte limits or retrieval quality.
14. The evaluator follows `EVALUATION-CONTRACT.md`: stage denominators remain separate, required failures stay in denominators, calibration cannot vote for promotion, and confirmation cannot tune settings.
15. Frozen source-backed relevance truth is the product reference. Later codec fidelity uses serving-dimension f32 as a second independent reference; Phase 07 must not pre-label f32 results as truth.

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

The resolver canonicalizes the supplied path, confirms it is an existing local checkout, and verifies the commit, clean-tree state, expected hash, root subdirectory, filters, language slice, and license metadata before indexing. It records the portable corpus identity and verified content hash in the run artifact, but does not copy the runtime absolute path into tracked artifacts or SQLite. A separate `.cidx/test/states/<corpus>/` root owns config, production/source/lab DBs, and artifacts while exercising the same production index/search services.

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

Every frozen query follows `owner-adopted-dual-ai-v1`: two different AI
systems independently inspect shuffled, rank/score/arm/prior-label-hidden,
source-complete packets; reconciliation leaves no grade-2 or required-group
conflict; support receives grade 1 only on dual agreement; and the owner adopts
or rejects the reconciled digest without relation-level regrading. Frozen
records use `OWNER_ADOPTED_DUAL_AI_REVIEW` plus permanent
`NO_INDEPENDENT_HUMAN_REVIEW`, never `HUMAN_REVIEWED`. No-answer and
hard-negative labels require corpus-wide search evidence and explicit agreement
from both passes. Pool unique top results from simple search, FTS, current
serving-f32/int8/RRF arms, existing truth parents, and relevant historical
diagnostic provenance before final label freeze.

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
independent AI judgments. Both initial passes remain blind to machine labels,
arm, score, rank, outcomes, owner preference, and the other pass. The record
enumerates every covered relation and includes packet/pass digests, reviewer
system identity, timestamps, source-inspection attestation, decision,
rationale, required-group assignment, and `source_verified`. Reconciliation
must resolve grade-2/group conflicts against source; grade 1 requires dual
agreement. An uncovered relation is `UNREVIEWED`, never implicit grade 0.

The final pool is the semantic-parent-deduplicated union, at the frozen pooling
depth, of every opened arm plus existing truth parents. Before label freeze,
require zero unreviewed relations in both passes, zero unresolved disagreements,
zero duplicate/conflicting judgments, zero missing source verifications, and
zero invalid grade-2 group assignments. Bind that proof to corpus, generation,
dataset, arm set, scoring depth, simple-policy fingerprint, shuffle seed, and
final pool digest. A previously unseen parent added by a later arm reopens both
passes for that relation. The current pool uses 1024/int8 plus its declared
f32/FTS/control arms. Supported compact 512/int8 must be explicitly opened before it
is pooled; retired Binary/256 evidence cannot be silently included.

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
deduplicate the top five parents from FTS, serving f32, active int8, RRF, and
simple control plus every draft-v2 truth parent. They contain 133 chi and 175
RHF query-parent relations, with all labels and lane/rank/score information
hidden. Those historical packets predate the current authority and profile
boundary. Fresh digest-bound packets and both independent AI passes remain
incomplete.

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
f32, active int8, parent collapse, RRF, ablation, and body packaging. All
32 responses validated with zero retry/failure, 646 provider-reported tokens,
USD `0.00007752` accounted cost, zero document provider operations, and no
query-vector persistence. The run showed real fusion rescues and regressions;
OR fusion is rejected as a serving candidate and production AND is unchanged.

The historical refreshed blind unions contain 191 chi and 281 RHF relations.
Two AI reviewers independently covered every relation with rank, score, arm,
prior label, and experiment result hidden. These remain advisory because they
predate the current profile/pool and `owner-adopted-dual-ai-v1` adoption wire;
they cannot be relabeled retroactively as the required fresh passes. Chi has a
complete reconciled advisory label map. RHF direct relevance is fully reconciled and
matches all 22 existing draft direct parents, while 101 grade-0/grade-1 support
differences remain deliberately separate. RHF NDCG is reported as the two
support-map endpoints and their min/max label-sensitivity range, not a
confidence interval or midpoint. Completeness and first loss continue to use
the unchanged draft required-group topology. The exact run, packet, pass,
reconciliation, and replay hashes are recorded in
[`measured-retrieval-loop-r4.md`](evidence/phase-07/measured-retrieval-loop-r4.md).

The fresh current-profile chi/RHF calibration packets, two independent AI
source passes, deterministic reconciliation, whole-digest owner adoption, and
provider-free replay are complete. The accepted checkpoint is recorded in
[`dual-ai-calibration-freeze-r4.md`](evidence/phase-07/dual-ai-calibration-freeze-r4.md).
It closes these 32 exposed calibration questions but does not satisfy the
promotion-capable confirmation floor. Formal Phase 07 completion still needs a
separate unexposed confirmation set reviewed under the same authority. If only
calibration labels change, rescore immutable rankings provider-free; do not
call Voyage again. If confirmation labels change after results were exposed,
that rescore is diagnostic/regression only and a new unexposed confirmation
unit is required for promotion.

On 2026-08-18 the clean relation-metadata v2 sidecars and two predeclared
provider-free policies completed at commit
`c197cdafa93852df2c1463d2636378caae288130`. Metadata dense-first preserved
the frozen dense top five, recovered chi G09 through the exact
`RealIP -> realIP` call, and raised complete related evidence from `30/32` to
`31/32`. RHF X08 remained `RELATION_ADMISSION` even though the exact
`FormState -> FormStateProps` relation was resolved and reachable. The
conditional graph-first crossover added no complete answer and made its chi
arm ineligible by attaching the protected `middleware.walkXFF` parent to G05.
All six immutable graph/run artifacts passed checksum, provenance, probe,
zero-provider, and independent Terra review. The accepted calibration-only
evidence is
[`relation-edge-metadata-diagnostic-r4.md`](evidence/phase-07/relation-edge-metadata-diagnostic-r4.md).
It authorizes neither selector for production and does not change the separate
unexposed-confirmation requirement.

The subsequent v3 value-parameter diagnostic confirmed that the underlying
RHF component-to-props structure is common: all six reviewed public components
were classified as `SIGNATURE / TYPE_VALUE_PARAMETER / DECLARATION`. The new
calibration selector preserved the accepted `31/32` complete result and zero
hard-negative attachments but did not close X08, which remains
`RELATION_ADMISSION`. The owner deferred the policy decision, so no further
ordering or question change was made. Exact evidence is
[`relation-value-parameter-diagnostic-r4.md`](evidence/phase-07/relation-value-parameter-diagnostic-r4.md).

The owner then reopened one bounded decision with four predeclared
anchor-first directional strength definitions over the same immutable v3
graphs and frozen dense top 20. Clean commit
`dd814915902986c3fcb5a36220a35d5f8297b894` completed all eight chi/RHF arms
and all eight deterministic repeats with zero provider operation. Every arm
preserved the primary top five and raised formal completeness to `32/32`, with
zero hard-negative or `walkXFF` attachment. That aggregate is not accepted as
quality proof: between 6 and 14 queries per corpus/arm received only grade-0 or
unreviewed attachments. Source-normalized focus and bidirectional specificity
selected the exact X08 `FormState -> FormStateProps` relation; raw frequency
and incoming popularity completed X08 only because the required rank-13 parent
was already selected as an anchor. No arm is accepted for production. Exact
evidence is
[`relation-anchor-edge-strength-diagnostic-r4.md`](evidence/phase-07/relation-anchor-edge-strength-diagnostic-r4.md).

The owner then authorized one bounded frontier test rather than another edge
ordering. Clean commit `770ff8e0c6c151791d5599bbdf68bd730dab7e99`
applied a provisional top two per selected-anchor/direction/structural-tier
bucket, canonical union without backfill, and a hard 32-edge ceiling. Across
all 32 exposed calibration queries the real maximum was only 8 edges for chi
and 11 for RHF; no query reached 32. The control materially compressed
occurrences but selected exactly the same relations as the prior unconditional
specificity arm, so its noise did not improve. Direct-anchor bridge abstention
reduced emission from 32 to 10 queries and noise-only emission from 20 to 4,
but it lost chi G09 because `RealIP -> realIP` ends at a graph-only parent rather
than the other selected dense anchor. It retained exact RHF X08. Therefore the
bucket cap remains a provisional development complexity control, while both
cap-only and bridge-only relevance policies are rejected. Exact evidence is
[`relation-frontier-cap-diagnostic-r4.md`](evidence/phase-07/relation-frontier-cap-diagnostic-r4.md).

The final predeclared exposed diagnostic then admitted either a direct anchor
bridge or a unique graph-only Pareto winner from the unchanged bounded
frontier. Clean commit `497c000bf0d3e9452fd8ff1ce9f570a3df144525`
completed chi and RHF initial/repeat runs with zero provider operation. It
recovered exact G09 and X08 evidence and reached `32/32` formal completeness,
while preserving the accepted dense top five, frontier digests, and safety
gates. The admission evidence was not precise enough for product use: 10 of 17
emitted bundles were noise-only, and the unique-Pareto branch was useful in
only 1 of 7 emissions. Direct bridge, unique Pareto, and their combined rule
are rejected as serving policies. The complete relation sidecar remains
evaluation-only; product storage/search has no dependency on it. Tuning on
these exposed 32 cases is now closed. Exact evidence is
[`relation-graph-only-pareto-diagnostic-r4.md`](evidence/phase-07/relation-graph-only-pareto-diagnostic-r4.md).

The owner subsequently accepted a new-unit-only sequence for semantic graph
admission, bounded contract closure, and metadata-only assistant hints. The
sequence is frozen in
[`RELATION-EVIDENCE-COMPLETION-PLAN.md`](RELATION-EVIDENCE-COMPLETION-PLAN.md).
It does not reopen the exposed 32. The corpus-independent implementation may
reuse the complete active-int8 segment scores already written by the Phase 12
retrieval artifact and must make zero provider calls of its own. New corpus
selection, acquisition, document capture, calibration queries, confirmation
queries, and assistant execution remain separate explicit user-approval
gates.

The corpus-independent Stage A implementation is accepted at clean commit
`c863c049128470a190639f5e74b28a4b16a7f0f7`. It consumes checksum-bound graph
and retrieval artifacts plus an ID/text-only dataset projection, proves the
active-int8 canonical-input universe and distinct collapsed parents, and emits
semantic endpoint, contract-closure, and body-free hint inventories without a
provider call or vector persistence. Terra's final review is `CLEAR`; the main
one-time offline boundary passed. No corpus or score was run. See
[`relation-evidence-completion-stage-a-r4.md`](evidence/phase-07/relation-evidence-completion-stage-a-r4.md).

Completion reports must not use the metric value itself as a success declaration. They should demonstrate that the evaluation is reproducible and that failures are traceable.

## 12. Follow-up Handoff

Provide Phases 08–12 with:

- the user-selected, manifest-verified corpus IDs and query/answer dataset;
- lexical baseline rank source data;
- query categories and representative lexical failures;
- the evaluator contract used to compare exhaustive f32 with active int8 at the same serving dimension; historical Binary/256 reports remain document evidence only; and
- the fixed `candidate_k`, `return_k`, and RRF input conditions used by hybrid comparison.
- frozen calibration/confirmation digests and the lexical baseline that later lanes must protect.
- historical AI-advisory direct maps and support-label sensitivity endpoints as
  preparation only; only a fresh owner-adopted dual-AI digest is frozen truth.

Embedding phases must not silently modify dataset answers. If a correction is required, increment the dataset version and record the annotation reason. They also must not infer corpus download or embedding authorization from the existence of a manifest or local binding.

Phase 12 extends the shared Phase 07 `internal/eval` dataset, ground-truth, metric, and report types. It must not create a second incompatible dataset schema or duplicate those types without an adapter.

## 13. Decision Log

| Decision | Reason | Revisit when |
| --- | --- | --- |
| Use 1024/int8 for ordinary evaluation, allow explicit compact 512/int8, preserve 1024-f32 document source rows, and remove Binary/256 executable arms. | The user selected the highest-fidelity measured int8 target as the default, required provider-free local dimension changes, and chose documents rather than hidden code paths as the Binary/256 evidence boundary. | A new explicit product-contract decision authorizes another profile or source-retention policy. |
| Build a lexical baseline before retrieval comparisons. | This separates vector improvement from chunking defects; Phase 08 implementation may proceed in parallel. | Never for the Phase 12 prerequisite contract. |
| The user selects every open-source corpus. | Corpus suitability, licensing, checkout ownership, and download authorization belong to the user. | Product scope explicitly introduces managed corpora. |
| Separate portable tracked manifests from ignored local bindings. | Reproducibility needs pinned provenance while machine paths must not enter version control. | A dedicated portable workspace resolver is designed. |
| Never select, download, or embed a corpus in Phase 07 without explicit user authorization. | This phase is a free, local lexical evaluator and a manifest alone is not authorization for external actions or paid work. | A separate user-authorized acquisition or embedding workflow is designed. |
| Observe hit@k/MRR without a numeric gate. | The corpus and product usage pattern are not yet sufficiently settled. | Representative corpora and product requirements accumulate. |
| Use `OWNER_ADOPTED_DUAL_AI_REVIEW` as the solo-project frozen relevance authority. | The sole owner cannot produce an independent human pass. ChatGPT and Grok independently accepted a strict source-backed, rank/score/arm-hidden two-system protocol with whole-digest owner adoption and permanent `NO_INDEPENDENT_HUMAN_REVIEW`. | An independent human review program becomes available or the authority contract is explicitly versioned again. |
| Use the accepted deterministic simple-search policy as an evaluation-only control. | On 2026-08-16 the user accepted a corpus-independent `ANY` normalized-token admission rule over the authoritative semantic-parent snapshot, followed by exact qualified/symbol, path, matched-token, and stable identity ordering. It adds no alias, BM25, embedding, boost, public wire, or production-ranking change. | A later evaluation-contract revision explicitly replaces the control. |
| Stop FTS micro-tuning and carry safe OR 5:1 only as a provenance-bound development comparator. | Repeated AND/OR, minimum-two-token, and 5:1/5:5 experiments showed that aggregate gains can conceal lost required Go/TSX parents. One coherent 32-query Voyage run now provides the missing f32, binary, fusion, and body evidence while production remains AND. | The blinded two-pass pool review is complete and a measured structural change is justified without cross-slice correctness regression. |
| Reject OR fusion as a serving candidate under the current draft evidence. | The coherent Voyage run showed lower complete-required@5 than pure dense for both corpora, with no chi rescue and concrete chi/RHF required-parent regressions despite isolated RHF rescues. Aggregate RHF also concealed a TypeScript decline and TSX rise. | Authority-compliant frozen labels plus a new structural fusion design show no required-group regression in every protected slice. |
| Report the historical RHF support relevance as a two-endpoint NDCG sensitivity range. | The old blind AI passes agree on direct truth but differ on 101 subjective grade-0/grade-1 relations. They predate the current authority, pool, and profile boundary, so a forced midpoint would manufacture precision. | The fresh dual-AI review freezes the current support map; grade 1 requires dual agreement. |
| Do not repeat Voyage when only labels change. | The corpus, query texts, profiles, document bank, query vectors' resulting immutable ranks, and retrieval policies are already bound and complete; rescoring new labels is provider-free. | Any bound corpus, question, embedding, or retrieval-policy identity changes. |
| Open one paired f32/binary/int8 top-20 diagnostic before human freeze. | The user explicitly requested an int8 comparison and wider candidate review. One Voyage query vector per case is reused across all three isolated dense arms; candidate int8 documents are locally derived once from the existing raw bank, production remains binary, and no RRF enters the codec comparison. | The corpus, questions, serving dimension, transform, document bank, or codec implementation changes. |
| Measure one isolated 512-dimension int8 follow-up. | The user considers 1,024-dimensional int8 too large for the intended local footprint. Reuse the approved 1,024-f32 raw document bank and shared prefix-L2 transform in separate development state; request fresh query embeddings once because query vectors are not persisted. Compare 512-int8 only to same-run 512-f32 for codec fidelity, and report 512 versus 1,024 as separate checkpoints. | The bound corpus, questions, source model/dimension, transform, raw bank, or codec implementation changes. |
| Carry 512/int8 as the compact candidate without changing the working baseline. | The completed follow-up halved int8 vector payload, reduced the two complete SQLite files by `26.48%`, retained every direct answer by top 20, and preserved `1.0000` chi / `.9950` RHF same-run f32 top-20 membership. The 32 draft questions are not sufficient for production promotion. | Authority-compliant frozen review or confirmation evidence contradicts the result, or the bound model/transform/codec/profile changes. |
| Measure one final isolated 256-dimension int8 follow-up. | The user wants the smaller supported serving dimension measured before choosing between compact candidates. Reuse the approved 1,024-f32 raw document bank and shared prefix-L2 transform in separate development state; request each of the same 32 query embeddings once because query vectors are not persisted. Compare 256-int8 only to same-run 256-f32 for codec fidelity and report cross-dimension results as separate checkpoints. | The bound corpus, questions, source model/dimension, transform, raw bank, or codec implementation changes. |
| Keep 512/int8 preferred and retain 256/int8 as a memory-constrained alternative. | The completed 256 run preserved `.9917` chi / `.9950` RHF same-run f32 top-20 membership and every direct answer by top 20, but saved only another `7.12%` of the two complete SQLite files beyond 512 and showed less stable shallow placement, most clearly RHF T12. Fresh query hashes differ across checkpoints, so this is an observed practical choice rather than a causal paired dimension claim. | A frozen same-source-query comparison or confirmation evidence reverses the storage/quality tradeoff. |
| Advance 512/int8 as the preferred historical evaluation candidate after five-profile consolidation. | Provider-free replay of all 32 questions found 31/32 direct and complete answers by top 5, useful source code for 32/32 by top 5, `.9969` same-run f32 top-20 retention, and a complete DB only `13.53%` larger than the 1,024-binary controls. The later owner decision made 1024/int8 the default and retained 512/int8 only as the compact option. | Authority-compliant frozen labels, a causal same-source-query dimension grid, or confirmation evidence reverses the result. |
| Retain the paired Binary/int8 run as historical decision evidence. | The clean paired run showed int8 top-20 retention above `.992` on both corpora with zero top-1 mismatch against exhaustive f32, while Binary retained materially less of the f32 neighborhood. Binary is now removed from the executable product. | Authority-compliant frozen review or a new confirmation corpus contradicts the calibration result, or the bound codec/transform/profile changes. |
| Prefer representative cohort intents and reject quota-padding edge cases. | The user wants questions that expose material failure modes, not detail added only to reach a count. Difficult cases remain valuable when they isolate a real parser, parent-collapse, type/wrapper, codec, or retrieval distinction. | New evidence shows a missing material failure mode that cannot be covered by a representative question. |
| Use measured cohort failures before revising questions; keep G07/T01/X01/X08 and narrow only G12. | Repeated advisory grading was slower and less decisive than the existing real rankings. The four misses each retain a distinct source-backed diagnostic boundary, while G12 alone contained wording broader than the Go source contract. | A new measured run or source change invalidates one of those distinct boundaries. |
| Accept the T10 and G09 source-backed label revisions. | `PathImpl` and `PathInternal` directly implement T10 while public `Path` is useful support; `walkXFF` is a reviewed misleading implementation for the deprecated G09 contract. | Pinned source identity or the question intent changes. |
| Use authority-compliant source-reviewed path/kind/symbol targets as ground truth. | These map directly to the code-search unit while keeping the absence of independent human review explicit. | Multi-hop task evaluation or a new review authority is added separately. |
| Freeze the 32 chi/RHF cases as calibration, retain 1024/int8 dense as the retrieval-quality baseline, keep FTS separate, and reject the tested 1:1 and FTS1:dense2 RRF arms. | Complete dual-AI source review and provider-free replay showed dense Complete@5 `30/32`; both fixed RRF formulations regressed required RHF results, and the weighted probe also worsened the reviewed top-5 label mix. The measurement guide, ChatGPT, and Grok agreed to stop weight tuning at this boundary. | A structural candidate-generation/chunking change or a newly frozen unexposed dataset justifies a new arm; this decision does not claim lexical evidence can never help. |
| Evaluate compiler-resolved relations first as an immutable development sidecar with protected dense top five and at most two related-evidence parents. | Dense already localizes every calibration query by top 20, while G09 and X08 expose missing call/type evidence completion. kb-guide review found production schema, MCP, and serving integration premature until resolver accuracy, deterministic admission, packaging, and actual evidence value are measured. | The fixed chi/RHF diagnostic proves or rejects the relation architecture; any production integration still requires a separate plan and unexposed confirmation. |
| Reject the fixed relation selector for production while retaining the compiler-resolved sidecar as development evidence. | Clean provider-free replay recovered every pinned G09/X08/T09/T10 relation exactly, but selected unrelated one-hop facts for G09 and X08 and preserved complete-at-five at `30/32`. The first loss is relation admission, not extraction or parent mapping. | A separately specified query-conditioned evidence-group selector is measured on a new evaluation unit without retuning the exposed 32 cases. |
| Run one predeclared relation-context metadata diagnostic and only then a conditional graph-first crossover. | The owner rejected relation-text embedding and authorized AST/compiler-derived edge context only. The first arm preserves dense-first retrieval and freezes its metadata key before results. If admission remains unresolved, the second arm uses the already frozen FTS/simple-control parents as graph seeds and the already frozen dense-1024/int8 ordinal as the post-graph reranker. Both are new immutable calibration-diagnostic units over unchanged questions/labels, make zero Voyage calls, use no RRF, and cannot become confirmation or production evidence. | A new unexposed confirmation unit supports a production admission decision, or a graph-only parent outside the frozen dense depth requires a separately approved fresh query-embedding design. |
| Retain relation metadata as development evidence, reject both measured selectors for production, and reject graph-first crossover under the fixed policy. | Metadata dense-first recovered G09 and reached `31/32`, proving that AST/compiler occurrence context can improve evidence assembly. X08 remained an admission loss. Graph-first added no completeness and attached protected `walkXFF` evidence to G05, failing its explicit gate. The result separates useful stored structure from an insufficient relevance decision. | A newly versioned calibration unit specifies a different query-conditioned admission design before exposure, and the separate unexposed confirmation set then validates it without a protected-slice or safety regression. |
| Treat X08 as a representative value-parameter contract case, not a one-off exception. | The exact `FormState`/`FormStateProps` collision is unique, but every one of the six reviewed public RHF React components declares its `*Props` contract as a value-parameter type annotation. The v2 graph collapses all six into `TYPE_LOCAL`. The next bounded unit therefore adds one mechanically derived role distinct from generic type parameters and evaluates a new policy across all 32 cases; it does not edit X08, add a component-name exception, call Voyage, or retry graph-first traversal. | Retain the role only if all six common contracts are classified correctly and complete-answer plus protected-hard-negative evidence does not regress. Production still requires a separate unexposed confirmation unit. |
| Defer the value-parameter selector policy decision after measurement. | The v3 graph classified the common six-component pattern correctly, while all-32 replay remained `31/32`, changed only T09/X08 selections, attached no declared hard negative, and left X08 at `RELATION_ADMISSION`. The owner does not need to decide this policy now. | Resume only on explicit owner direction; do not tune another key or replace X08 as an implicit consequence of this run. |
| Resume the policy question with one predeclared anchor-first directional edge-strength series. | The owner explicitly requested a real test before policy adoption and emphasized graph grades and strengths. Reuse the immutable v3 facts and frozen dense top 20, select two anchors with the existing query/symbol normalizer, enumerate the same outgoing and incoming one-hop candidates, and compare four isolated lexicographic definitions: raw frequency, source-normalized focus, bidirectional target specificity, and incoming popularity. Structural tiers are mechanical graph facts, not relevance grades; all counts come from the complete graph within identical relation-kind/tier strata. | Decide only after all 32 cases, hard negatives, changed attachments, first losses, body packaging, deterministic repeats, and zero-provider evidence are reviewed. Any later production proposal still requires an unexposed confirmation unit. |
| Reject all four anchor/edge-strength arms as serving policies and retain their metadata as calibration evidence. | All arms formally completed `32/32`, but unconditional bundle selection produced 6–14 noise-only queries per corpus/arm. Source normalization and target specificity uniquely selected X08's exact relation; raw frequency and popularity only packaged the correct anchor while selecting another edge. The result supports anchor-first localization plus source/target strength as ingredients, not an always-on ranker. | A new versioned unit freezes an admission/abstention rule before exposure and the separate unexposed confirmation set validates it without primary, hard-negative, body-budget, or evidence-quality regression. |
| Retain the measured top-two bucket frontier only as a provisional development complexity control. | It reduced chi to at most 8 and RHF to at most 11 final edges per query, preserved primary evidence, and produced deterministic repeats. The global 32-edge ceiling, forced bridge reservation, and displacement paths were not exercised by the real corpora, and bounding alone reproduced the prior selector and noise. | A separately frozen evaluation unit measures another corpus/fan-out distribution without tuning this exposed set. |
| Reject direct-anchor bridge-only abstention as the general relation admission rule. | It reduced emitted noise but lost reachable G09 evidence, because a selected anchor can point to a useful graph-only endpoint. Four of ten emitted bundles were still noise-only. Exact X08 survived, so bridge presence remains a useful feature rather than a complete decision. | Completed by the final Pareto diagnostic in the next row; no further exposed-set tuning is permitted. |
| Reject direct-bridge, unique graph-only Pareto, and their combined admission rule for product use; close all policy tuning on the exposed 32 cases. | The final arm recovered G09 and X08 and reached `32/32` without primary or hard-negative regression, but only `7/17` emitted bundles were useful. The new Pareto branch supplied the missing G09 evidence yet was useful in only `1/7` emissions. The measured graph features remain useful diagnostics, not a sufficient relevance decision. | A separately frozen new development unit specifies another admission design before exposure, followed by the existing unexposed confirmation requirement. |
| Freeze the next relation series around existing semantic scores, interpretive contract closure, and body-free assistant hints. | The previous selectors used lexical and ordinal query features but never the continuous dense score of graph endpoints. Existing retrieval artifacts already contain every active-int8 segment score from one request, so a provider-free relation pass can derive global parent rank/percentile and ambiguity without persisting vectors or repeating queries. Contract closure and assistant pull answer separate questions and keep dual count/byte budgets. | After corpus-independent evaluator review, the user approves new calibration repositories. A distinct confirmation unit validates the frozen policy, and assistant A/B independently decides push versus pull before any product wire change. |
| Accept the corpus-independent relation-completion Stage A boundary without claiming a result. | Clean commit `c863c04` binds producer/consumer evidence, label-free candidate construction, active-int8 completeness, closure/hint budget grids, and final reproof. It performs no corpus or provider operation and leaves product search/MCP unchanged. | The owner selects every new calibration repository; only then may the new unit be prepared and separately approved for document/query operations. |
| Do not edit the exposed 32-case calibration set to improve its scores. | The questions, labels, and immutable ranks have now influenced a retrieval-policy decision. Further tuning on them would invalidate their role as an unbiased confirmation unit. | Never for this dataset version; create a new versioned calibration or an unexposed confirmation unit. |
| Preserve failure taxonomy alongside metrics. | It identifies which implementation layer should change next. | Never. |
| Call production services directly. | This prevents divergence between evaluation and actual behavior. | Never. |
| Expose evaluation only through development surfaces. | Keep the public user and MCP surfaces small and stable. | Productization is explicitly requested. |
| Use the shared stage scorecard and first-loss model. | A final Hit@5 cannot prove parser correctness or explain where evidence disappeared. | Never. |
| Freeze calibration separately from confirmation. | Parameter tuning and promotion evidence cannot come from the same queries. | A new versioned evaluation policy is approved. |
| Keep required failures in denominators. | Dropping failed observations inflates retrieval and operational metrics. | Never. |
| Acquire only the two explicitly user-authorized public checkouts for this resume. | The user named commits, versions, licenses, and allowed local acquisition; this is a narrowly recorded exception to the default no-download rule. | Any corpus identity, commit, or authorization changes. |
| Resume Phase 07 with real-data structural audit and behavior-cohort authoring before paid work. | The user prioritized measured corpus behavior over additional test scaffolding and fixed the first working profile to 1,024-byte segments, 1,024 serving dimensions, and binary storage. | The structural audit finds a canonical-input defect or the exact document-capture plan reaches its explicit approval gate. |
