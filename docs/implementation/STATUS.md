# cidx v1 Persistent Implementation Status

This ledger is the authoritative resumable state for implementation work. Update it before starting a phase, before pausing, after context compaction is noticed, and before marking a phase complete.

## Current state

- Active phase: 14 — Phase 13 is complete with structured-only locator search,
  source-only `read_span`, reducer v2, real Codex conformance, and
  byte-identical locator output across compatibility-budget values. Paired V4
  is complete: both arms blindly graded 12/12 complete; locator search emitted
  zero source/text and reduced cidx event payload 82.9% versus V3, but paired
  model-token median was 1.044 and only 4/12 tasks were non-increasing. The
  first remaining loss is repeated/refined searches, six failed reads, four
  exact duplicate reads, and imprecise evidence acquisition. ChatGPT and Grok
  independently returned `PROCEED_V5_ORCHESTRATION`; reducer v3 measures its
  observable clauses and reproduced V4's residuals without modifying V4. The
  V5 is complete: 24/24 turns were valid and both arms blindly graded 12/12
  complete. Search/read/source activity and uncached input fell, but the paired
  model-total median is 0.954 and only 6/12 pairs are non-increasing, so the
  token gate fails. Three `read_span` calls omitted the required locator hash
  and were retried. ChatGPT and Grok both returned
  `PROCEED_V6_HASH_FIELD`. V6 then completed 24/24 valid turns. Hash omissions
  fell to zero, but treatment blindly graded 11 complete and one partial
  against baseline 12 complete. The 11 dual-complete pairs have model-total
  median 0.977 and 6 non-increasing, so correctness preservation and the token
  gate both fail. ChatGPT and Grok agree on the gates, final-claim loss, narrow
  adherence metric, and no-V7 stop. Close with no efficiency claim; bounded
  optional locator value remains an unpromoted product hypothesis. The
  closure/handoff is complete. The owner has now selected a separate
  host-decided search-to-evidence design for a future 30-question/60-turn,
  FTS-only availability study. ChatGPT and Grok both returned `FINAL_ACCEPT`;
  documentation is reconciled. The owner-authorized neutral tool-description
  plus passive harness trace/reducer Step 2 boundary is now implemented and
  validated and committed at `143b578`. The owner has now authorized the next
  host-decided availability experiment. Step 3 is complete: the source-first
  30-question version, truth sidecar, neutral prompt, 15/15 counterbalanced
  schedule, dual-AI review/adoption, exact model-visible tool contract, and
  provider-free preflight/schema probes, default `k=5`, 64 KiB read limit, and
  exact per-corpus config/index byte identities are frozen at commit `b334aa7`.
  Step 4 is now `in_progress`. The frozen 60 turns completed exactly once:
  60/60 are valid and the freely available cidx tools were selected in 0/30
  treatment turns. Blind packets and journeys are frozen, but blind grading is
  paused after two pre-model structured-output schema rejections
  (`uniqueItems`, then `allOf`). No grade or aggregate exists. New test code,
  public MCP input/output expansion, and paid provider work remain unauthorized.
  ChatGPT and Grok have now reviewed the fixed result. Both classify 0/30 as a
  neutral-exposure adoption/interface result rather than retrieval failure and
  accept the output-only v2 grading adapter. The selected later experiment is
  a fresh two-arm informed comparison with one byte-identical capability-
  awareness paragraph in both arms, unchanged questions, and unchanged tool
  descriptions. The owner has now explicitly superseded that immediate follow-
  up with a bounded non-promotion prompt diagnostic: both arms expose identical
  cidx tools and FTS state, while only the directed arm is told to use cidx
  rather than `rg` or equivalent ordinary repository search. Prompt
  noncompliance remains in the denominator. The versioned blind-grade schema
  for post-turn claim accounting remains canonical. The output-only supported-
  keyword adapter and its isolated no-tool compatibility probe are now frozen.
  The neutral run's single aggregate attempt rejected a noncanonical
  required-group status, so its quality result is closed as `NOT_OBSERVED`; no
  grade is repaired or recalled. A stricter supported-keyword adapter now
  constrains both the v2 root envelope and canonical enum values, and an
  isolated no-tool Codex endpoint probe passed. The policy-v2 body-free trace,
  runner, and scorer are implemented at commits `c930587`, `c7bb19d`, and
  `1f4b5b7`; they preserve legacy V3–V6 replay, enforce equal cidx exposure and
  a sole directed suffix, bind both trace builders, retain noncompliance in the
  denominator, and require the frozen 30/15/15 schedule. Independent freeze
  review returned `ACCEPT`, and the executable 30/15/15 manifest was sealed.
  The frozen preflight and both schema probes passed, all 60 scored turns
  completed exactly once, and blind grading plus the single aggregate are
  complete. Both arms grade 29 complete and one partial with all 39 required
  groups; directed cidx reduces combined unique source 532,030 to 196,476 and
  unsupported claims four to one, but raises repository actions 104 to 254 and
  paired model-total ratio median to 1.404. Answerable tasks reach complete
  locator/read evidence 27/27; the measured loss is repeated search/read turns
  and source reacquisition after useful locators. No public interface change,
  optional-use follow-up, or promotion inference is authorized by this result.
  The owner rejected and hard-reset the subsequent overbuilt awareness/trust
  implementation before any new scored turn. The required plan was then
  restored without that infrastructure and passed bounded ChatGPT/Grok review.
  The minimal successor was frozen at
  `caa57f906d9a3e6219d24c8fe71914bf82bd22ec` and its one-time 60-turn run is
  complete. Trust-priority produced 30/30 complete answers versus aware-choice
  28 complete, one partial, and one timed-out ungradable answer. On 29 valid
  comparable pairs it cut unique source to median 0.546, but raised repository
  actions to median 1.300 and model-total usage to median 1.189. The measured
  next boundary is evidence acquisition after useful locator selection, not a
  retrieval retune. ChatGPT and Grok accepted this interpretation with the
  correction that the run does not isolate stopping from legitimate dependency
  expansion. No public interface successor is authorized yet.
- Active owner: `/root`
- 2026-08-23 Phase 14 execution entry: `/root` re-read the execution guide,
  implementation index, status ledger, evaluation contract, full Phase 14,
  accepted awareness/trust plan and review, Phase 13 locator completion, Step
  2 implementation evidence, and the closed forced-prompt result. The clean
  entry commit is `f47563a`. The existing 30-question runner, policy/passive
  trace, blind grader, scorer, approved corpus bindings, and FTS-only state are
  reusable without a new runner or schema family. Authorized edits are limited
  to a new frozen manifest, the two reviewed MCP description strings, this
  checkpoint/freeze documentation, and only a small failure-denominator fix if
  focused probes prove it necessary. No successor scored turn may run before a
  clean freeze commit and passing provider-free preflight plus both arm schema
  probes.
- 2026-08-23 Phase 14 awareness/trust freeze: the exact 30-pair manifest,
  shared awareness prompt, sole trust-priority suffix, two concise MCP
  descriptions, existing runner/trace/grade path, and failure-safe efficiency
  denominators are frozen in
  [`assistant-cidx-awareness-trust-freeze-v1.md`](evidence/phase-14/assistant-cidx-awareness-trust-freeze-v1.md).
  Focused Go, race, vet, Python, synthetic denominator, legacy V3-V6 context,
  and provider-free preflight checks pass. The implementation stays at the
  500-line non-documentation boundary including the manifest.
- 2026-08-23 Phase 14 awareness/trust result: both arm probes passed and all 60
  primary cells ran exactly once. One aware-choice cell timed out and remains
  ungradable without retry; blind grading and one canonical aggregate are
  complete. Trust-priority improved complete answers from 28/30 to 30/30 and
  required-group coverage from 37/39 to 39/39, while paired unique source fell
  to median 0.546. It also increased cidx calls from 102 to 220, paired actions
  to median 1.300, and paired model-total usage to median 1.189. Exact results
  and artifact seals are in
  [`assistant-cidx-awareness-trust-result-v1.md`](evidence/phase-14/assistant-cidx-awareness-trust-result-v1.md);
  the adopted interpretation and unapproved bounded multi-locator candidate are
  in the [external review](evidence/phase-14/assistant-cidx-awareness-trust-result-external-review-v1.md).
  Semantic sidecars, provider work, retrieval changes, and public MCP changes
  remain prohibited without a new owner decision.
- Completed bounded work: the final provider-free graph-only Pareto admission
  diagnostic is complete at clean commit
  `497c000bf0d3e9452fd8ff1ce9f570a3df144525`. It reuses the
  evaluation-only complete relation sidecar and the accepted bounded frontier;
  product storage, search, MCP, embeddings, questions, and labels are
  unchanged. All policy tuning on the exposed 32 cases is now closed.
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
- Completed Phase 07 anchor/edge-strength diagnostic owner: `/root` (four predeclared provider-free calibration arms; no product/search/provider change)
- Completed bounded Phase 07 frontier-cap diagnostic owner: `/root`. Clean
  commit `770ff8e0c6c151791d5599bbdf68bd730dab7e99` and eight provider-free
  artifacts prove that per-bucket top two reduces the observed frontier to at
  most 8 chi and 11 RHF edges; no query reaches the global 32-edge ceiling.
  Cap-only reproduces the prior specificity selector and noise. Bridge-only
  reduces emission but loses G09 and retains four noise-only bundles. The cap
  remains a provisional development control; neither arm is a product policy.
- Completed final Phase 07 Pareto-admission diagnostic owner: `/root`. Four
  deterministic provider-free runs reproduced combined chi+RHF formal
  completeness `32/32` in both the initial and repeat pair, but emitted only 7
  useful bundles versus 10 noise-only bundles. The unique
  graph-only branch was useful in `1/7`; bridge, Pareto, and their combined
  rule are rejected for product use. The full sidecar remains evaluation-only.
- Phase 07 simple-control implementation owner: `/root/phase07_simple_control` (store/eval/devlab only; no corpus, provider, or production-ranking mutation)
- Last updated: 2026-08-23
- Owner review index: [`OWNER-REVIEW-INDEX.md`](OWNER-REVIEW-INDEX.md) — single
  entry for packaging freeze, live results, adopted contract, and remaining
  gated work
- Packaging experiment entry: live closed-unit replay returned
  `CONTINUE_SIBLING_PACKAGING` at contract digest
  `cb726ace5f81d980260a8111520d5b2f00f9318f128682f3ddc6cc8ff7a54c28`.
  Sibling decision cell recovers 5/6 named same-file misses (32/40 topology-complete
  queries) with unchanged primary top five and zero labeled isolated extras.
  One-hop recovers both nearby misses but also completes `gg-g09` in every grid
  cell, so the one-hop gate fails and Arm D stays unauthorized. Do not retune
  this closed unit. Assistant A/B was deferred at that checkpoint and later
  completed as Version 3 after lexical remediation.
- Latest owner direction: do not add a repository and do not constrain cidx to
  a secondary-only product role. Preserve V4–V6 as closed history. The next
  planning direction lets the host decide whether cidx is first, main,
  occasional, or unused; it separates compact candidate locators from complete
  selected evidence and claim-required dependency expansion. The accepted
  design uses the existing approved corpora, the existing nine-field search
  wire and `read_span`, a passive host-side trace, and 30 frozen pairs/60 turns
  in FTS-only/no-credential mode. See
  [`ASSISTANT-SEARCH-EVIDENCE-FLOW-DESIGN.md`](ASSISTANT-SEARCH-EVIDENCE-FLOW-DESIGN.md)
  and the matching
  [`FINAL_ACCEPT` review](evidence/phase-14/assistant-search-evidence-flow-external-review.md).
  The owner has additionally authorized one separate prompt-policy diagnostic
  where both arms expose cidx and the directed arm must use it instead of `rg`
  or equivalent ordinary repository search. This is non-promotion mechanism
  evidence; it preserves violations and does not replace the optional-use
  product comparison. See
  [`ASSISTANT-FORCED-CIDX-PROMPT-EVALUATION-V1.md`](ASSISTANT-FORCED-CIDX-PROMPT-EVALUATION-V1.md).
  That diagnostic is now complete. Its immutable
  [result](evidence/phase-14/assistant-forced-cidx-prompt-result-v1.md) preserves
  aggregate quality and substantially narrows inspected code, but rejects the
  current blanket policy as a token-efficiency claim because model/tool turns
  increase materially. The next change is an owner decision about the
  search-to-evidence interface, not an inferred secondary-only product role.
  The attempted successor was removed by hard reset. Resume only from the
  [restored successor plan](ASSISTANT-CIDX-AWARENESS-TRUST-EVALUATION-V1.md) and
  the [incident record](evidence/phase-14/assistant-cidx-followup-overbuild-incident.md);
  the plan authorizes only its minimal implementation and focused preflight
  until a clean scored-run freeze exists.
- Canonical target: [`local-code-search-mcp-v1-design-r4.md`](../../local-code-search-mcp-v1-design-r4.md)
- Completed critical/general checkpoint: every prior question-set file and run
  remains preserved; `critical-general-v1`, chi/RHF question-set v2, explicit
  run provenance, and four new provider-free FTS/simple runs are recorded in
  [the v2 report](evidence/phase-07/critical-general-question-set-v2.md).
  FTS completes 10/12 lexical-anchor questions but 0/24 semantic-only and 0/8
  mixed-signal questions; simple control completes 28/44 overall. This proves
  the taxonomy separates material lane behavior, not that ranking improved.
  All 32 semantic/mixed FTS cases returned zero candidates, so the immediate
  cause is the global all-token-AND admission plan rather than measured BM25
  rank quality. Current remediation:
  [natural-language FTS query-planner review](evidence/phase-07/natural-language-fts-query-planner-review-r4.md).
- Phase 06 remediation completed: planner v2 uses safe descriptive OR admission,
  independent symbol/path/descriptive lanes, deterministic local parent RRF,
  shared hybrid input, and MCP/evaluation diagnostics. Focused normal/race,
  vet, build, format, and diff checks passed. Phase 07 then completed three
  preserved real-corpus run pairs. The final clean pair at `2e1a270` reduced
  candidate-zero from `32/44` to `0/44`, improved CompleteRequirementHit@5
  from `10/44` to `30/44`, and lost no historical pass. At candidate depth 20,
  38/44 queries contain every required group; eight are then displaced beyond
  top five and six remain candidate-depth losses. Exact evidence:
  [planner v2 rerun](evidence/phase-07/natural-language-lexical-rerun-v2.md).
- Latest contract change: source and state roots are now distinct runtime inputs. Normal state is `<source>/.cidx` with production DB `.cidx/db/index.db`; cidx development evaluation uses disposable sources under `.cidx/test/corpora/` and preserved named states under `.cidx/test/states/`. Absolute source/state paths are removed from SQLite metadata.
- Latest evaluation change: the 40-query relation Stage E/F unit completed two
  blinded reviews, source-only adjudication, whole-digest adoption, and two
  byte-identical 25-cell evaluations. Baseline completeness is 52/61 groups
  and 31/40 queries. Closure count 2/2,048 bytes reaches 57/61 and 36/40;
  hint count 4/4,096 bytes reaches 58/61 and 37/40, with substantial noise in
  both. The descriptive upper envelope is 38/40 and is not a policy.
  `NO_POLICY_SELECTED_EVALUATION_ONLY` is final for this closed unit.
- Latest selection/noise diagnostic: v2 replay (SHA-256
  `33a91723549c12486da93c07a638907537ff5065ad7375432109cbb19939656d`)
  shows six of nine baseline misses are sibling symbols in files already
  returned by dense top five; three are cross-file at ranks 14, 40, and 134.
  Topology-only dense top-10 recovers 3/9, not the withdrawn 5/9 draft.
  Hint-cell isolated parents are 13/133 and file collapse leaves zero
  noise-only queries. Assistant A/B is deferred. Exact report:
  [overlap/selection diagnostic](evidence/phase-07/relation-overlap-noise-diagnostic-r4.md).
- Latest bounded-frontier result: chi compresses `119 -> 115 -> 79 -> 56 ->
  51` and RHF `1290 -> 711 -> 362 -> 158 -> 153` across self removal,
  occurrence collapse, bucket truncation, and canonical dedupe. Cap-only is
  complete `32/32` but emits 20 noise-only queries. Bridge-only emits `10/32`,
  keeps X08, loses G09, and leaves 4 noise-only emissions. Primary top five,
  hard-negative safety, deterministic repeats, and zero-provider gates pass.
- Latest admission result: the unchanged 204-edge combined frontier partitions
  into 11 bridge views, 64 incoming exclusions, 38 dense-endpoint exclusions,
  and 91 graph-only candidates. Outcomes are 10 direct bridges, 7 unique
  Pareto winners, 13 multiple-winner abstentions, and 2 no-candidate
  abstentions. Exact G09 and X08 evidence is recovered, but 10/17 emitted
  bundles are noise-only. The rule is not a serving candidate.
- Completed relation-series authority: Stage A–F remain historical evidence
  under [the relation evidence completion plan](RELATION-EVIDENCE-COMPLETION-PLAN.md).
  Current follow-up is governed by
  [the packaging plan](RELATION-PACKAGING-NEXT.md) and the corrected
  [handoff](RELATION-ASSISTANT-VALIDATION-HANDOFF.md). The exposed 32 cases
  and the 40-query unit remain closed. No new corpus, provider operation,
  production graph path, search wire, fifth MCP tool, or assistant A/B is
  authorized.
- Completed relation Stage A boundary: clean implementation commit
  `c863c049128470a190639f5e74b28a4b16a7f0f7` adds the provider-free
  completion consumer and plan-bound Phase 12 producer class. It binds graph,
  active-int8 score coverage, distinct collapsed parents, raw/canonical
  dataset identities, exact query texts, profiles, provenance, and final live
  reproof without decoding labels or persisting vectors. Terra's final review
  is `CLEAR`; the main offline full/race/vet/build/module/format/dependency/
  diff boundary passed. Exact evidence is [relation Stage A
  evidence](evidence/phase-07/relation-evidence-completion-stage-a-r4.md).
- Completed Stage E/F review boundary: immutable label-blind emissions contain
  1,000 query/cell rows; the source-complete universe contains 616 parent and
  1,115 relation attachments over 40 queries. ChatGPT and Grok independently
  covered the universe. A third Terra source-only pass resolved 102
  grade/group conflicts, and the owner adopted the reconciled digest with zero
  row override under `NO_INDEPENDENT_HUMAN_REVIEW`. Clean commit `ba44fab`
  fixes the adjudication-aware owner-adoption path and passed independent Terra
  review plus the full main boundary. Stage F's two byte-identical runs emit
  1,000 per-query cells, 1,025 cell aggregates, and 3,534 delivery aggregates.
  Exact evidence is [the Stage E/F report](evidence/phase-07/relation-calibration-stage-ef-r4.md).
  No policy, product graph path, confirmation, or assistant-use claim is made.
- Completed offline readiness checkpoint: the full normal/race/vet/build/
  module/format/schema/script/dependency boundary passed, 64 retained artifact
  manifests covering 727 files match their recorded bytes and SHA-256 values,
  and the frozen chi/RHF Pareto replay reproduced all ten accepted key hashes
  with zero provider operations. A clean package run exposed an outdated
  third-party module allowlist; commit `31d37fe` reconciles the exact linked
  `x/tools` dependency set, after which package and installed-release
  verification passed. The host's Go 1.26.3 shim is recorded separately from
  the accepted already-cached Go 1.26.4 offline toolchain. Exact evidence is
  [the offline readiness report](evidence/phase-07/relation-evidence-completion-offline-readiness-r4.md).
- Completed Stage B corpus boundary: the owner selected `go-git/go-git`
  `v5.19.1`, `pmndrs/zustand` `v5.0.14`, and `usememos/memos`
  `v0.30.0`. Their exact commits, licenses, Git trees, selected language
  slices, generated-file exclusions, and selected-content hashes are frozen
  in [the Stage B evidence](evidence/phase-07/relation-calibration-stage-b-r4.md).
  Ignored checkouts are clean. Isolated 1024/int8 indexing, indexed-file
  parity, parser inventory, relation-sidecar construction, and the later
  retrieval/completion artifacts are complete.
- Completed Stage B parser correction: the first provider-free pass indexed
  go-git and Memos but exposed valid semicolonless consecutive generic call
  signatures in four Zustand public API files that the embedded grammar did
  not accept. TypeScript chunker v3 performs a same-length, error-free-only
  shadow parse and preserves original bytes/ranges; global index chunker v3
  forces reindex. Focused parser/index/app/devlab normal and race checks, vet,
  build, format, and diff passed. No question, label, semantic score, or
  provider operation was involved.
- Completed mixed-language resolver correction: the Memos graph pass exposed the
  root-only `tsconfig.json` assumption. The portable corpus manifest now owns
  an optional, strictly relative `typescript_config`; Memos binds
  `web/tsconfig.json`, while existing corpora retain the root default. The
  field participates in the manifest fingerprint and is unavailable without
  a TypeScript/TSX slice. No machine path or production configuration changed.
- Completed immutable-inventory correction: inventory addresses now bind the
  full corpus-manifest fingerprint in addition to corpus, generation, and
  index manifest. This lets distinct portable selection policies coexist over
  one unchanged index generation without deleting or overwriting retained
  evidence.
- Completed Go resolver-universe correction: Memos proved that `go/packages`
  loaded 39 generated Go files excluded from the committed index. Compiler
  loading may remain wider for type resolution, but persisted file states and
  target membership are now restricted to the exact index snapshot; excluded
  and dependency targets classify as `OUT_OF_CORPUS`, not parent-mapping
  failures.
- Completed Stage B document and retrieval boundary: clean commit `d59a36e` froze 40 new draft
  cases before score exposure. Voyage document capture persisted all 10,659
  distinct 1024-f32 source inputs with zero failure and 2,036,537 observed
  input tokens, then local materialization published complete 1024/int8
  coverage. The source-bank and production SQLite integrity checks pass and
  repeated capture plans have zero paid misses. The later bounded query series,
  retrieval artifacts, and relation-completion artifacts are immutable inputs
  to the completed Stage E/F review; Stage E/F itself made zero provider calls.
- Completed final admission boundary: [graph-only Pareto evidence](evidence/phase-07/relation-graph-only-pareto-diagnostic-r4.md)
  records the frozen rule, clean implementation and executable, plan/run
  hashes, exact denominators, deterministic repeats, G09/X08 bodies, zero
  provider/hard-negative/`walkXFF`, metric-guide rejection, and Terra `CLEAR`.
- Completed frontier boundary: [frontier-cap evidence](evidence/phase-07/relation-frontier-cap-diagnostic-r4.md)
  records the exact implementation, clean executable, eight artifacts,
  complexity and relevance denominators, G09/X08 traces, `kb-guide` review,
  and Terra `CLEAR` artifact audit.
- Completed strength boundary: [anchor/edge-strength evidence](evidence/phase-07/relation-anchor-edge-strength-diagnostic-r4.md)
  records clean commit `dd814915902986c3fcb5a36220a35d5f8297b894`, four
  isolated definitions, eight initial runs and eight deterministic repeats,
  exact X08/G09 traces, unchanged primary top five, zero provider operations,
  verified checksums, and Terra `CLEAR`.
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
- Completed bounded work: Version 1 preserved zero-adoption optional exposure;
  Version 2 preserved shell-versus-MCP instruction noncompliance; Version 3 ran
  all 12 baseline/treatment pairs with 12/12 MCP compliance, blind correctness,
  and arm-blind machine journey evidence. Treatment was 12/12 complete versus
  baseline 11 complete + 1 partial, but used 37.2% more model tokens and 50.6%
  more uncached input. Both reviewers require correction before another A/B.
- Completed bounded work: `/root` prepared a separate strict passive
  blind-grade output adapter revision. Entry evidence checked: the canonical
  v1/v2 grade schema, frozen v2 output adapter, root-envelope recovery
  protocol, unchanged passive-grade semantic validator, and future
  packet/prompt generator. The new adapter constrains the root grade envelope
  to version 2 and uses canonical enum sets for outcomes, group status, and
  material-claim classification. It passed offline JSON, supported-keyword,
  frozen-structure parity, canonical-enum alignment, Python compilation,
  future-prompt wording, an isolated no-tool endpoint compatibility probe, and
  diff checks. The frozen adapter and canonical
  schema remain unchanged; no model call, grading, aggregation, runner, trace,
  arm, or prompt-policy change occurred. Evidence:
  [strict adapter v3](evidence/phase-14/assistant-blind-grade-strict-output-adapter-v3.md).
- Exact next action: keep the Availability V1 grading closed with quality
  `NOT_OBSERVED`; do not call its graders again. The strict-adapter endpoint
  probe passed. Finish and freeze the manifest-defined same-tools neutral-
  versus-directed prompt
  runner/accounting before any new scored turn.
  Do not add metadata, another tool, a repository, new questions, test code,
  public MCP schema fields, or provider traffic.

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
| 06 | done | `/root` | Yes — Phase 05/06 evidence, current lexical/index/store code, v2 run diagnostics, evaluation contract, and dual side-panel review inspected | Planner v2, safe descriptive OR, independent symbol/path/descriptive lanes, deterministic local parent RRF, serving-policy fingerprint, and shared hybrid/MCP/evaluation diagnostics passed focused normal/race/vet/build/format checks | [Current and historical Phase 06 evidence](evidence/phase-06/README.md) and [planner review](evidence/phase-07/natural-language-fts-query-planner-review-r4.md) | Hand unchanged v2 inputs and lane diagnostics to Phase 07 |
| 07 | done | `/root` | Yes — execution/index/evaluation/Phase 07 documents; completed revised Phase 06 evidence; existing chi/RHF manifests, checkouts, states, question versions, and retained run artifacts | Planner v2 completed three preserved run pairs; final clean pair has `0/44` candidate-zero, `38/44` complete admission at depth 20, `30/44` complete top-five, and no historical-pass regression | [Critical/general v2 report](evidence/phase-07/critical-general-question-set-v2.md), [Phase 07 evidence](evidence/phase-07/README.md), [planner review](evidence/phase-07/natural-language-fts-query-planner-review-r4.md), and [final rerun](evidence/phase-07/natural-language-lexical-rerun-v2.md) | Preserve this exposed calibration; assistant response accounting is complete and implementation proceeds separately in Phase 14 |
| 08 | done | `/root` | Yes — full Phase 08 document, historical R4 evidence, accepted source-bank decision, Phase 05 handoff, live lab/embedclient wiring, and current state layout inspected | Product `db/embeddings.db` owns immutable document 1024-f32; `lab/evaluation.db` contains metadata only; compatible legacy rows copy read-only; focused test/race/vet/build/format/import/schema boundary passed | [Current source-bank evidence](evidence/phase-08/int8-source-bank-reconciliation.md), historical [R4 evidence](evidence/phase-08/revision-4.md), and [source-bank decision](SOURCE-VECTOR-BANK-DECISION.md) | Hand compatible sources and missing keys to Phase 09/10 |
| 09 | done | `/root` | Yes — prior Phase 09 evidence, live vector/materialization code, Phase 08 source-bank boundary, five-profile evidence, and retired-profile contract inspected | Int8-only transform/materialization/search contract, production v5 cache, retired runtime removal, and one final offline boundary accepted | [Current evidence](evidence/phase-09/int8-only-materialization-reconciliation.md), historical [Phase 09 evidence](evidence/phase-09/README.md), and [retired-profile contract](RETIRED-VECTOR-PROFILES.md) | Hand direct int8 transform/scorer and source-bank rematerialization to Phase 10/11 |
| 10 | done | `/root` | Yes — accepted Phase 09 boundary, prior Phase 10 R4 evidence, active embedding path, source-bank decision, and retired-profile contract inspected | Source-bank-first provider success handling, compatible local reuse, public source/Voyage plan split, provider-only request accounting, and final offline boundary accepted | [Current evidence](evidence/phase-10/source-bank-first-document-publication.md), historical [R4 evidence](evidence/phase-10/revision-4.md), and [source-bank decision](SOURCE-VECTOR-BANK-DECISION.md) | Hand current int8 coverage/profile state to Phase 11 |
| 11 | done | `/root` | Yes — accepted Phase 09/10 boundaries, prior Phase 11 R4 evidence, live vector scan/evaluation code, five-profile evidence, and retired-profile contract inspected | Current request-local int8 scan, nonpersistent serving-f32 reference, fallback/RRF/body behavior, retired comparison removal, and focused boundary accepted | [Current evidence](evidence/phase-11/int8-only-query-search-reconciliation.md), historical [R4 evidence](evidence/phase-11/revision-4.md), and [retired-profile contract](RETIRED-VECTOR-PROFILES.md) | Hand the current evaluation arms to Phase 12 |
| 12 | blocked | `/root` | Yes — accepted current Phase 11 boundary, prior Phase 12 R4 evidence, evaluation contract, current schemas/adapters, relation completion authority, retired-profile contract, closed Stage E/F, and adopted packaging contract inspected | Corpus-independent adapter accepted; packaging/no-policy authority is now frozen evaluation-only sibling 4/4096; official `core_retrieval` still lacks independent confirmation | [Remaining-work handoff](evidence/revision-4/remaining-work-review-handoff-r4.md), [adopted sibling contract](../../testdata/retrieval/relation-sibling-packaging-adopted-v1.json), [current evidence](evidence/phase-12/int8-only-evaluation-reconciliation.md), and [R4 accounting evidence](evidence/phase-12/revision-4.md) | After owner-selected unexposed confirmation, seal margins and run official core evaluation; assistant V3 remains separate non-promotion evidence |
| 13 | done | `/root` | Yes — execution guide, index, status, evaluation contract, full Phase 13/14 documents, current Phase 13 evidence, live MCP handlers/schema, frozen V3 artifacts, and accepted response-contract review inspected | Structured-only is the product default; search emits only nine-field deduplicated locators, requests zero source bytes, and preserves caller-selected ranking. Focused normal/race/static checks, 0/12000/65537 byte-identical server responses, and a real Codex search-to-read journey passed | [Locator-only completion](evidence/phase-13/locator-only-mcp-reconciliation.md), [reducer v2](evidence/phase-13/assistant-reducer-v2.md), [Codex representation probe](evidence/phase-13/codex-result-representation-probe.md), and existing [int8-only evidence](evidence/phase-13/int8-only-cli-mcp-reconciliation.md) | Preserve the nine-field locator and source-only `read_span`; reopen only for the separately authorized neutral-description validation |
| 14 | in_progress | `/root` | Yes — Step 3 clean commit `b334aa7`, awareness/trust freeze `caa57f9`, execution guide, implementation index, status, evaluation contract, full Phase 14 and accepted search-to-evidence design/review, Phase 01/02/13 prerequisite evidence, current MCP schema, runner, reducer, and preserved V3–V6 artifacts inspected | The same-tools forced prompt diagnostic and the minimal awareness/trust successor are complete. In the successor, trust-priority grades 30/30 complete versus aware-choice 28 complete, one partial, and one timeout; paired unique source median is 0.546, but paired action and model-total medians are 1.300 and 1.189. Post-result reviewers agree that evidence-acquisition/orchestration after useful locator selection is the measured boundary and that stopping versus legitimate dependency expansion is not isolated | [Awareness/trust result](evidence/phase-14/assistant-cidx-awareness-trust-result-v1.md), [post-result review](evidence/phase-14/assistant-cidx-awareness-trust-result-external-review-v1.md), [execution freeze](evidence/phase-14/assistant-cidx-awareness-trust-freeze-v1.md), [accepted plan](ASSISTANT-CIDX-AWARENESS-TRUST-EVALUATION-V1.md), [forced prompt result](evidence/phase-14/assistant-forced-cidx-prompt-result-v1.md), and [overbuild incident](evidence/phase-14/assistant-cidx-followup-overbuild-incident.md) | Await an explicit owner decision on the bounded multi-locator evidence-request contract; no retrieval/provider/public-interface change is authorized |

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
