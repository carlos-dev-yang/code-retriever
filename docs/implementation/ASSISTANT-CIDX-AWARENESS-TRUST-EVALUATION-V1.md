# Assistant cidx Awareness vs Trust-Priority Evaluation V1

- Status: `accepted_for_implementation`
- Date: 2026-08-23
- Phase: 14, packaging and host integration
- Evidence class: paired, non-promotion assistant-use diagnostic
- Predecessor: [Forced cidx Prompt Diagnostic V1 Result](evidence/phase-14/assistant-forced-cidx-prompt-result-v1.md)
- Product interaction design: [Assistant Search-to-Evidence Flow](ASSISTANT-SEARCH-EVIDENCE-FLOW-DESIGN.md)
- Evaluation authority: [Evaluation and Promotion Contract](EVALUATION-CONTRACT.md)
- Reset boundary: [Follow-up Overbuild Incident](evidence/phase-14/assistant-cidx-followup-overbuild-incident.md)
- External review: [bounded ChatGPT and Grok review](evidence/phase-14/assistant-cidx-awareness-trust-plan-review-v1.md)

## 1. Owner question

The previous forced-use diagnostic showed that cidx could locate the required
code and reduce unique source volume, but forcing it for every repository
search multiplied model/tool turns and increased model tokens. This experiment
asks a smaller and more realistic question:

> When the same assistant is told that cidx exists, does concise awareness
> guidance let it choose cidx where useful, and does an additional instruction
> to prefer and trust exact cidx evidence reduce the total code-investigation
> journey without reducing final-answer quality?

The experiment does not assign cidx a permanently primary or secondary product
role. The consumer remains free to decide that role. It tests whether the MCP
can narrow code exploration accurately enough, with enough evidence, to reduce
an AI agent's total investigation work.

## 2. Fixed predecessor facts

The predecessor consisted of 30 paired questions and 60 primary Codex CLI
turns. Its action counts are totals over 30 questions, not counts for one
question and not all cidx calls:

| Arm | Repository actions | Mean per question | cidx calls | Mean cidx calls per question |
| --- | ---: | ---: | ---: | ---: |
| no cidx awareness | 104 | 3.47 | 0 | 0.00 |
| forced cidx | 254 | 8.47 | 229 | 7.63 |

The 229 cidx calls were 110 searches and 119 successful reads. The remaining
25 directed-arm repository actions were ordinary known-file reads. The forced
arm preserved the aggregate blind answer result (`29 complete / 1 partial`,
`39/39` required groups), reduced combined unique source from 532,030 to
196,476 bytes, and reduced unsupported material claims from four to one. It
also raised model-total tokens by 52.7%, with a paired median ratio of 1.404.

Cached input rose by 71.2% while uncached input rose by only 1.8%. This is
consistent with repeated model/tool turns replaying accumulated context, but
it does not prove that identical code was repeatedly placed in the prompt.
Repeated turns, repeated or overlapping source, unique source bytes, cached
input, and uncached input must therefore remain separate observations.

The ambiguous `FormState` question is the clearest diagnostic example: cidx
reduced unique source from 17,188 to 1,938 bytes, but the assistant performed
10 searches and six reads and used 4.332 times the model-total tokens. The next
experiment must determine whether that continued exploration came from query
reformulation, insufficient adjacent or dependency evidence, inconvenient
handoff, reassurance, or another concrete reason. It must not assume that
smaller source is automatically sufficient evidence.

The predecessor result is closed. Its questions, answers, grades, traces, and
reports are not repaired, reclassified, or rerun as a prerequisite.

## 3. Scope and non-goals

This experiment changes only model-visible awareness and trust guidance over
the same exposed tools. It reuses:

- the same approved Go, TypeScript, and TSX repositories and source commits;
- the same frozen 30 questions, truth, required groups, and question shapes;
- the same model, reasoning effort, answer contract, and ordinary tools;
- the same local provider-free FTS index and four cidx tools;
- the existing assistant A/B runner, isolation, passive trace, policy
  classifier, blind-grade path, and scorer; and
- the same balanced 15/15 first-arm schedule pattern.

It does not add or change:

- a repository, question truth, retrieval ranking, FTS planner, dense or
  hybrid search, embedding provider, or paid provider operation;
- the public four-tool set, MCP input/output schemas, source storage, or
  production defaults;
- a predecessor re-audit, per-action semantic reviewer, semantic packet or
  sidecar pipeline, mandatory 60-session self-report, or a second 60-context
  review pass;
- a new evaluation runner, large scorer rewrite, or new schema family; or
- a latency, hit-rate, promotion, `core_retrieval`, or `release_candidate`
  gate.

The expected implementation is a small generalization of existing evaluation
files. If execution requires a new standalone runner, a new schema family, or
more than 500 net non-documentation lines, stop and obtain an owner decision
before expanding the experiment.

## 4. Paired arms

Both arms expose byte-identical ordinary repository tools, all four cidx tools,
the same concise cidx descriptions, and the same FTS state. Neither arm hides
`rg`, forces cidx on every question, or penalizes cidx non-use.

Both arms receive this shared awareness paragraph:

> `cidx.search` and `cidx.read_span` are available for repository code
> discovery. `cidx.search` locates ranked functions, methods, and types from
> natural-language, identifier, or path queries; `cidx.read_span` returns the
> current source for selected locators. Use cidx when it helps narrow the code
> to inspect, and use ordinary repository tools when they better fit the task.

### Arm A: `aware_choice`

Arm A receives only the shared awareness paragraph. It can choose cidx,
ordinary tools, both, or neither. This is the natural nudge arm.

### Arm B: `trust_priority`

Arm B receives the same paragraph plus exactly this suffix:

> When locating repository code, prefer `cidx.search` over `rg` or equivalent
> text search. When `cidx.read_span` succeeds, trust its exact path, range, and
> indexed hash instead of reacquiring the same or overlapping source merely
> for reassurance. Start with the recommended default candidate count. Refine
> only for a missing answer requirement or a concrete ambiguity, dependency,
> context gap, or exhaustive check. Ordinary tools remain available when they
> are a better fit.

Arm B is a priority-and-bounded-trust instruction, not a cidx-only policy. Its
effect is interpreted as one combined intervention; this 30-question run does
not separately identify the causal effect of each sentence.

To reuse the existing validated runner without renaming its protocol, the
manifest technical IDs remain `neutral_cidx` for `aware_choice` and
`directed_cidx` for `trust_priority`. The new run identity and report always
show both the technical ID and the semantic label.

## 5. Shared concise MCP descriptions

The following model-visible descriptions are byte-identical in both arms and
are frozen in the experiment manifest. They describe the interface without
embedding a long orchestration policy.

### `search`

> Locate ranked functions, methods, and types from a natural-language,
> identifier, or path query. Returns locators without source text; inspect
> selected locators with `read_span`. Omit `k` to use the recommended default
> of 5; valid values are 1 through 20. Top-k results do not prove
> repository-wide absence.

### `read_span`

> Return the complete current source for one selected locator's exact path,
> range, and indexed hash.

Trust, stopping, and adjacent/dependency guidance appears only in Arm B's
suffix. `status` and `reindex` keep their existing capability descriptions.
Codex receives these descriptions from the live MCP `tools/list` response, so
the implementation changes only the two description strings in
`internal/mcp/schema.go` and uses the same built binary in both arms. Tool
names, input schemas, outputs, ranking, and server behavior remain unchanged.

## 6. Evaluation unit and schedule

The primary unit is one `question_id + arm + final answer` cell. “Thirty tasks
used cidx” is not an answer-quality result.

- run 30 paired questions and 60 fresh primary turns;
- preserve the existing 10 Go, 10 TypeScript, and 10 TSX distribution and the
  existing exact/path, semantic, multi-hop, disambiguation, negative, and
  contract/lifecycle shapes;
- run `aware_choice` first for 15 questions and `trust_priority` first for 15;
- start each primary turn in a fresh, isolated session and preserve its session
  identity for possible selected post-run questioning;
- assign a new experiment version and run identity; and
- keep timeouts, failures, cidx non-use, ordinary-only use, and mixed use in
  their scheduled denominators.

Reuse the runner's existing per-cell isolation. Every primary cell starts with
its own source snapshot, cidx SQLite copy, state root, and MCP process, all
derived from the same frozen initial source/index binding. Record initial and
post-turn source and database identities. A model-issued `reindex` may change
only that cell's private database and remains observed behavior; it never
becomes another cell's starting state. Stop for an initial identity mismatch,
source mutation, missing provenance, or cross-cell leakage, not merely because
the model changed its own private generation. Store prompts, answers, traces,
grades, and session artifacts outside the model-visible and indexed source
snapshot.

Every new final answer is freshly graded, arm-blind, against that question's
unchanged frozen truth and required groups. Existing predecessor grades remain
fixed historical results. The new run never overwrites them.

Every scheduled cell receives an outcome. A timeout, non-zero process exit,
missing or invalid final output, or mechanically ungradable answer is
`ungradable`, is not retried, and receives zero coverage for positive required
groups. A question with no positive required group remains `N/A` rather than a
fabricated coverage failure. Claim-support rates use gradable answers with
material claims and always show that denominator.

## 7. Three separate measurement layers

### 7.1 Final-answer and claim quality

For every question/arm cell, record:

- complete, partial, incorrect, or ungradable answer outcome;
- covered required groups;
- supported, unsupported, and contradicted material claims;
- correct file, symbol, and contract resolution; and
- paired conversion or regression.

This layer decides whether the assistant actually answered the question. Tool
adoption and exact-range compliance cannot substitute for it.

### 7.2 Deterministic journey trace

Use the existing passive/policy trace to record, without a model reviewer:

- ordered cidx and ordinary repository actions;
- first useful locator and its rank;
- requested `k`, including default, larger, and invalid values;
- exact locator reads;
- same-file, same-hash contained or overlapping reads derived from a locator;
- ordinary reads of unrelated source;
- unique source bytes, overlapping or reacquired bytes, and repository-output
  proxy bytes;
- the earliest action where all source cited by the final answer was
  mechanically available, when observable;
- searches and reads after that mechanical evidence frontier;
- independently available locator reads that could have been grouped; and
- actions that introduced source used by the final answer versus actions that
  did not introduce newly cited source.

The mechanical evidence frontier proves only that source later cited by the
answer was available. It does not prove that the assistant understood it at
that moment or that every later action was unnecessary. Exact locator-range
conformance remains a handoff diagnostic, while exact, derived-overlap, and
unrelated reads are reported separately. None is a product-success proxy.

### 7.3 Selected original-agent explanation

First seal all 60 primary answers, traces, grades, and primary usage. Only then
select at most 12 sessions for a single tool-free follow-up in the same
original Codex CLI session. Selection is purposive diagnostic sampling, not a
quality denominator. A session is eligible when its sealed trace shows at
least one concrete issue:

- more than one search after the first useful locator;
- ordinary source reacquisition after a successful cidx read;
- an exact duplicate or overlapping range;
- a partial, incorrect, ungradable, or unsupported-claim outcome;
- unusually high actions or paired token ratio; or
- a required adjacent/dependency evidence gap visible in the final grade.

Keep coverage across question shapes and include both a cidx-use and a cidx-
non-use case when available. Ask only the relevant subset of these questions:

- Why did you choose or not choose cidx?
- Why was each search after the first useful locator necessary?
- At what point did you believe you had enough evidence to answer?
- Did `read_span` omit adjacent or dependency context needed for a final
  claim?
- Why did you inspect the same or overlapping source with another tool?
- What made an ordinary tool more trustworthy or convenient?
- What smallest response or interface change would have ended exploration
  sooner?
- Which parts of this explanation are observed facts and which are inference?

The follow-up cannot call tools, revise the answer, change its grade, or enter
primary token accounting. Its response is untrusted diagnostic evidence and
must be checked against the sealed objective trace. If exact-session resume is
not reliable with a small runner change, omit this layer and report the
blocker; do not build a separate semantic-review system.

## 8. Target journey and context sufficiency

The harness observes rather than enforces this target journey:

```text
one compact search
-> select a small plausible locator set
-> read enough source to judge the candidates and support the answer
-> perform one concrete refinement, dependency/context expansion,
   disambiguation, or exhaustive check only when needed
-> answer
```

The goal is not to minimize bytes at all costs. A parent function may locate
the answer but still omit behavior delegated to another function or type. A
legitimate adjacent or dependency read must remain available and is counted as
evidence expansion, not waste. Re-reading the same range for reassurance is a
different event and is measured separately.

Context sufficiency is diagnosed using three observations together:

1. whether the final answer is supported by the source already returned;
2. whether a later ordinary or cidx read added a concrete cited dependency or
   adjacent-context fact; and
3. for selected problem traces, what the original assistant reports was
   missing or inconvenient.

This permits source delivery to grow when real evidence is missing and shrink
when later actions only repeat already available evidence.

## 9. Metrics and denominators

Report measures separately; do not create a weighted total score.

### Answer quality

- outcomes and required-group coverage over all 60 scheduled cells;
- material-claim support over gradable answers with material claims;
- paired conversions and regressions; and
- all-question, language, and question-shape slices.

### Tool choice and retrieval journey

- cidx use, ordinary-only use, mixed use, and no repository-tool use by arm and
  question shape;
- first discovery family;
- cidx searches, cidx reads, ordinary searches, and ordinary reads per
  question, including median, p75, maximum, and total;
- first useful rank, required positive-locator coverage, and negative-question
  handling;
- requested `k` distribution;
- mechanical evidence frontier and later-action counts;
- exact locator, derived-overlap, dependency/context, duplicate, and unrelated
  read observations; and
- selected diagnostic reasons, reported only over valid selected follow-ups.

### Scope and model cost

- unique source bytes, overlapping/reacquired bytes, and repository-output
  proxy bytes;
- official input, cached input, uncached input, output, and model-total tokens;
- paired token ratios and non-increasing pairs; and
- a descriptive, non-causal comparison of source volume, action/turn count,
  cached-context growth, and final quality.

Reading more code is expected to increase repository output, but model-total
tokens also depend on how many model/tool turns replay the accumulated
conversation. Source volume and token use must therefore be reported together
without treating either as a substitute for the other.

All 60 scheduled cells remain in answer-quality, execution-state, and tool-
choice denominators. All-cell action, source, and token totals are descriptive.
Paired action and source-efficiency estimates require two completed cells with
intact traces; paired token estimates additionally require valid official
usage in both cells. Incomplete trace prefixes are reported as censored
diagnostics, not as completed short journeys. Every excluded pair and reason
remains visible, and no arm is called more efficient when differential failure
or missing usage could explain the difference.

## 10. Interpretation

The run answers two comparisons:

1. `aware_choice` versus `trust_priority`: whether the short priority/trust
   suffix changes selection, stopping, evidence sufficiency, and final quality;
2. each new arm versus the predecessor as a descriptive historical reference:
   whether 8.47 repository actions per question and the 1.404 paired token
   ratio move in the intended direction on the same panel.

Interpret results as follows:

| Observation | Interpretation |
| --- | --- |
| Trust guidance preserves answer support and reduces actions/tokens | concise host guidance improves the cidx journey |
| It narrows source but still multiplies turns/cached input | prompt guidance is insufficient; evaluate handoff or bounded multi-read |
| It needs dependency/context reads that awareness omits | do not reduce evidence indiscriminately; improve selected context delivery |
| It reacquires already available source | improve locator handoff, source convenience, or deduplication guidance |
| Both arms rarely use cidx | the tool is not discoverable/useful enough for this panel under concise awareness |
| Trust guidance increases unsupported claims | the trust wording or returned evidence is unsafe and must not be adopted |
| Exact/path/negative questions choose ordinary tools | valid specialization, not automatic cidx failure |

This diagnostic may justify a next interface experiment, but it cannot itself
promote the product or establish performance outside the frozen panel.

## 11. Deferred interface candidates

Keep the four public tools unchanged for this run. Observe whether the result
supports either later candidate:

1. a ready-to-use locator reference that avoids manually copying path, range,
   and hash from `search` into `read_span`;
2. one lightweight external evidence request that internally performs bounded
   selection/read branching or reads several chosen locators while returning
   separate line-addressable evidence units.

Do not implement either before this prompt experiment. Distinct locators must
not be globally capped: legitimate multi-hop questions can require several
parents. The target is fewer duplicate decisions and round trips, not fewer
necessary facts.

## 12. Minimal execution sequence

1. Obtain one bounded pre-run review of this exact plan from ChatGPT and Grok.
2. Adopt only contradictions or missing definitions that would invalidate the
   simple A/B; record broader suggestions as deferred.
3. Inspect and minimally generalize the existing manifest, runner, trace, blind
   grader, and scorer for the two arm labels over their existing technical IDs,
   exact prompt text, concise tool descriptions, and optional resumable session
   identity.
4. Run focused legacy compatibility and syntax/contract checks; do not rebuild
   or semantically re-audit the predecessor.
5. Freeze the questions, truth, schedule, common initial source/index identity,
   per-cell isolation contract, model, prompt, tool descriptions, execution-
   code identity, and expected model-call count in a clean pre-run commit.
6. Run small unscored arm probes, then execute 30 pairs and 60 primary turns
   exactly once. Preserve failures instead of repairing or silently retrying
   scored cells.
7. Build new arm-blind grading packets for the 60 new answers and grade each
   corpus once using the existing path.
8. Aggregate final quality, deterministic trace, scope, action, and official
   usage measures once.
9. Select at most 12 problem traces and run the sealed, tool-free original-
   session diagnostic; keep its cost and interpretation separate.
10. Write the result and obtain one bounded post-result interpretation review
    from the same two side-panel chats.

Expected model operations are 60 primary turns, the existing two corpus-level
blind-grade calls, and no more than 12 selected follow-ups. Preflight probes are
unscored and reported separately. No semantic action-review calls are planned.

## 13. Stop rules

Stop rather than widening scope when:

- initial source/index, question, truth, tool, prompt except the frozen suffix,
  model, or schedule identity differs between paired cells, or private state
  leaks across cells;
- the same implementation or execution approach fails twice for the same
  cause;
- grading would require manual repair of a model answer;
- session follow-up could modify primary artifacts or requires a separate
  semantic-review system;
- a new runner, schema family, product interface, ranking change, corpus, or
  provider action appears necessary; or
- advisory review requests work outside the fixed experiment question.

## 14. Bounded external-review contract

Send this plan and the predecessor facts in Section 2 to ChatGPT and Grok. Ask
each reviewer for `PROCEED` or `BLOCKED` and no more than three exact minimal
corrections. Review is limited to:

- fidelity to the owner's two arms and 17 recorded decisions;
- whether final-answer grading, mechanical trace facts, and selected
  original-agent explanations are correctly separated;
- whether the shared descriptions and sole Arm B suffix preserve the intended
  comparison; and
- any missing denominator or confound that would invalidate the 30-pair run.

The review may not add a predecessor re-audit, semantic sidecar, mandatory
self-report, dedicated runner/scorer, new repository, dense/HNSW work, product
redesign, or recursively expanded review. Such advice is recorded as out of
scope rather than implemented.

## 15. Completion evidence before a scored run

Require only:

- this plan with the two bounded review dispositions;
- a small implementation diff or an explicit owner decision if the size
  boundary is exceeded;
- focused checks actually run and checks not run;
- a clean freeze commit containing exact prompts, descriptions, schedule, and
  identities; and
- proof that no predecessor artifact or product behavior was changed.

## 16. Owner-decision coverage

This table is a context-recovery index for the 17 owner annotations that
defined this plan. It introduces no additional work.

| # | Fixed decision | Plan authority |
| ---: | --- | --- |
| 1 | `104 -> 254` is 30-question total repository actions; forced cidx averaged 7.63 cidx calls and 8.47 repository actions per question | Section 2 |
| 2 | Judge each new question/arm final answer against frozen truth; exact locator range is diagnostic rather than product success | Sections 6, 7.1, 7.2 |
| 3 | Both arms are told cidx exists; Arm A receives awareness and free choice | Section 4 |
| 4 | A located parent can still require adjacent or dependency evidence; measure this separately from search success | Sections 7.3, 8 |
| 5 | Source/search scope and model-token use are separate measures | Sections 2, 9 |
| 6 | Cached-input growth is associated with repeated turns but is not proof of identical-source repetition | Sections 2, 9 |
| 7 | Observe a shorter search-to-evidence journey without sacrificing answer support | Section 8 |
| 8 | Shared MCP guidance recommends omitted/default `k=5` and states the maximum 20 | Section 5 |
| 9 | Ask selected completed CLI sessions why they chose, repeated, or distrusted cidx | Section 7.3 |
| 10 | Treat the `FormState` case as a context-sufficiency and stopping diagnosis, not a locator failure | Sections 2, 8 |
| 11 | Arm A is natural awareness; Arm B adds preference and bounded trust without forcing cidx-only use | Section 4 |
| 12 | Preserve objective traces to locate the actual loss; do not rebuild the predecessor audit | Sections 3, 7.2 |
| 13 | Separate the mechanical evidence frontier from semantic necessity; reduce duplicates while retaining needed expansion | Sections 7.2, 8 |
| 14 | Keep host/tool wording concise: shared interface facts, with trust/stopping only in Arm B | Sections 4, 5 |
| 15 | Keep four tools for this run; defer ready locator handoff or bounded internal branching/multi-read | Section 11 |
| 16 | Reuse the same 30 questions in a new version; both arms have awareness and only Arm B has priority/trust | Sections 4, 6 |
| 17 | Join tool choice, scope, actions, official tokens, claims, and selected post-run explanations by question | Sections 7, 9 |
