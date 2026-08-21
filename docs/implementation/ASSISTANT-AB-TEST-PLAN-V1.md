# Paired Codex CLI Assistant A/B Test Plan — Version 1

- Status: `frozen_for_execution`
- Owner: `/root`
- Scope: Phase 14 diagnostic evidence over the existing chi and React Hook Form calibration repositories
- Frozen task manifest: [`assistant-ab-chi-rhf-v1.json`](../../testdata/retrieval/assistant-ab-chi-rhf-v1.json)
- Frozen answer schema: [`assistant-answer.schema.json`](../../schemas/evaluation/assistant-answer.schema.json)
- Evaluation authority: [`EVALUATION-CONTRACT.md`](EVALUATION-CONTRACT.md)
- Result scope: diagnostic assistant-use evidence only; never `core_retrieval` or `release_candidate` promotion evidence

## 1. Question this experiment answers

Does optional access to the current four-tool, FTS-default cidx MCP help a Codex CLI assistant reach a source-backed answer with at least the same correctness while reading less irrelevant source and consuming materially fewer tokens than the same assistant using its normal local shell and file tools alone?

The experiment does not ask whether cidx can replace the assistant's existing tools. It measures the marginal value of adding cidx to those tools. A cidx-arm run may choose not to call cidx; that is retained as a valid no-use observation.

The primary estimand is the intent-to-treat effect of optional cidx
availability. All observed model, MCP-schema and tool overhead remains part of
the treatment and is not subtracted from primary token accounting.

## 2. Fixed comparison

| Control | Arm A — baseline | Arm B — cidx available |
| --- | --- | --- |
| Assistant | Codex CLI | Same Codex CLI |
| Model | `gpt-5.6-sol` | `gpt-5.6-sol` |
| Reasoning effort | `high` | `high` |
| Prompt | Exact shared template plus the same task question | Byte-identical to Arm A for the paired task |
| Working tree | Same pinned, clean repository snapshot | Same pinned, clean repository snapshot |
| Built-in tools | Read-only local shell and file inspection | Same |
| User config | Ignored | Ignored |
| Persistent conversation | None; `codex exec --ephemeral` | None; `codex exec --ephemeral` |
| Output | Same strict JSON schema | Same strict JSON schema |
| Model tool network | Browser, apps, computer use and standalone web search disabled; prompt prohibits network commands | Same |
| Extra capability | None | Current `cidx serve --root <corpus>` MCP only |
| MCP approval | N/A | Exact hash-pinned local cidx server is pre-approved; no interactive or automatic reviewer |
| cidx mode | N/A | FTS default; no Voyage query embedding and no paid operation |

The exact Codex CLI version, resolved model name if reported, cidx commit and binary SHA-256, corpus commit, configuration fingerprints, command line, timeout, and environment-variable names are written to the run manifest. Secret values are never recorded.

## 3. Shared prompt

Each pair receives this exact template, with only `{{TASK_ID}}` and `{{QUESTION}}` substituted identically in both arms:

```text
You are investigating the local source repository in the current working directory.

Task ID: {{TASK_ID}}
Question: {{QUESTION}}

Find the source-backed answer. Use any tools available to you, but do not assume that a particular tool is required. Do not modify files, run network requests, or rely on undocumented behavior. Stop once you have enough direct evidence; avoid broad repository tours.

Return only JSON matching the supplied output schema. Explain the mechanism concisely. Support every material claim with repository-relative file and symbol evidence. If the repository does not establish a claim, state the uncertainty instead of guessing.
```

The prompt never names cidx. This prevents forced tool use and keeps the product comparison honest.

## 4. Task selection and versioning

The unit contains 12 questions: four Go, four TypeScript, and four TSX. It covers the three critical retrieval families and the diagnostic modifiers most likely to expose false confidence.

| Coverage axis | Selected count | Why it is included |
| --- | ---: | --- |
| Lexical anchor | 3 | Checks exact identifier and declaration/implementation lookup where FTS should be strongest |
| Semantic-only | 6 | Checks whether cidx narrows natural-language investigation without exact query terms |
| Mixed signal | 3 | Checks combined identifier and behavior language |
| Multi-requirement | 3 | Prevents a plausible single-file answer from hiding missing evidence |
| Contract disambiguation | 4 | Separates declarations/contracts from nearby implementations |
| Known hard negative | 1 | Measures whether either arm follows a plausible but wrong branch |
| Go / TypeScript / TSX | 4 / 4 / 4 | Prevents one language adapter from determining the overall result |

The tasks reference the preserved critical/general v2 question files by path, SHA-256 and case digest. Gold paths, symbols, judgments and required groups are loaded only by the scorer; they are not copied into the assistant prompt or working repository.

This is a new assistant-test manifest version. It does not alter or overwrite any question-set version or earlier retrieval result. Any later task, prompt, model, rubric or threshold change creates a new manifest version and every result records the exact manifest digest it used.

## 5. Execution order and independence

There are 24 fresh Codex CLI executions: 12 tasks times two arms. Each task runs once per arm. The task order is frozen in the manifest, the first arm alternates six/six across the sequence to reduce systematic warm-order bias, and the paired arms execute back-to-back before moving to the next task.

One execution per cell is accepted only because this is a directional
diagnostic over frozen tasks. Model stochasticity remains a limitation; this
unit cannot estimate a population effect or support a release claim.

Every execution:

1. verifies the pinned commit and clean source tree;
2. uses a fresh ephemeral Codex session;
3. ignores user-level Codex configuration and MCP servers, and disables browser,
   app, computer-use, standalone-web-search and multi-agent features;
4. uses a read-only sandbox and the same wall-clock timeout;
5. writes raw JSONL events, stderr, final JSON, exit status and elapsed time to an immutable run directory;
6. leaves failures, malformed outputs and timeouts in the denominator;
7. does not reuse another task's assistant context.

The cidx arm receives only an experiment-local MCP definition. The baseline receives no MCP definition. Codex's per-server `default_tools_approval_mode` is fixed to `approve` for that exact locally built, SHA-256-recorded cidx binary because non-interactive Codex otherwise rejects unannotated MCP calls before dispatch. This is not an automatic reviewer and introduces no reviewer-model tokens. The shared prompt and control contract still prohibit `reindex`; any `reindex` call invalidates that execution. The same current FTS index is copied into ignored corpus-local state before the run and its fingerprint is recorded. No reindex, embedding, hybrid query, source mutation or provider request is permitted during the assistant runs.

## 6. Answer and evidence scoring

Scoring is stage-separated; no weighted total is created.

### 6.1 Deterministic evidence coverage

For each hidden required group, the scorer checks whether at least one cited evidence item matches either an accepted repository-relative path and qualified symbol or an accepted path and source span when no stable qualified symbol is supplied. Results are:

- `covered`: a cited path/symbol satisfies the group;
- `missing`: no cited evidence satisfies it;
- `invalid_evidence`: a cited path or symbol does not exist at the pinned snapshot.

Report required-group coverage as a numerator and denominator per task, arm, language and critical cohort.

### 6.2 Source-backed answer correctness

Before journey or token analysis, the final explanation is graded blind against
the exact cited source and frozen question truth. The grader receives only the
question, repository identity and assistant JSON; arm, journey, tokens, cidx use
and execution order remain hidden until every score is locked.

The outcomes are:

- `complete`: every required group is covered, the question is answered, and there is no material false claim;
- `partial`: some correct source-backed mechanism is given, but a required group or requested branch is missing;
- `incorrect`: the main conclusion contradicts source or follows a hard negative;
- `ungradable`: missing/malformed output or operational failure.

A task is an assistant success only when it is `complete`. `Partial` requires a
directionally correct answer and at least one covered required group, with a
material gap or unsupported claim remaining. `Incorrect` includes a wrong
mechanism, fabricated source identity, or an answer that does not answer the
question. Operational failures are always `ungradable`, remain in every
denominator, and are never converted to `incorrect` during post-processing.
Review records the specific missing group, unsupported claim or contradicted
claim rather than changing gold truth.

Frozen grading examples:

- `complete`: a two-group provider/consumer question explains both mechanisms,
  cites accepted evidence for both groups, and makes no material unsupported
  claim;
- `partial`: the provider mechanism and evidence are correct but the required
  consumer path is omitted, or the conclusion is directionally correct while
  one required branch is unsupported;
- `incorrect`: the answer relies on a frozen hard-negative branch as the main
  mechanism or states a behavior contradicted by the cited source;
- `ungradable`: the process times out, the final JSON is absent or malformed,
  or another operational failure prevents an answer from being judged.

### 6.3 Evidence precision and false leads

Report, separately:

- cited evidence items supported by a positive frozen judgment;
- cited hard-negative or grade-zero items;
- nonexistent citations;
- unique source files inspected before the final answer;
- whether the known hard-negative branch was opened, cited, or used in the conclusion.

A grade-zero or hard-negative source used as positive support for a material
claim prevents `complete`. Merely identifying it as an explicit contrast does
not. Unsupported and source-contradicted claims are counted separately.

## 7. Search journey observations

Raw Codex JSONL is the authority. The reducer records, per execution:

- shell/file command executions and outcomes;
- cidx MCP tool calls, arguments, result IDs and outcomes;
- unique source files read through either path;
- ordered evidence-discovery events;
- first event that exposes an accepted required-group path/symbol;
- irrelevant and hard-negative branches inspected;
- final cited evidence and whether it came from cidx, ordinary tools, or both;
- no-use in the cidx arm.

Tool counts are descriptive, not correctness substitutes. cidx returning a gold item is not a success unless the assistant uses sufficient evidence in a correct final answer.

## 8. Token and time accounting

The runner consumes the official `codex exec --json` JSONL stream. For every completed turn it retains:

- `input_tokens`;
- `cached_input_tokens`;
- derived `uncached_input_tokens = input_tokens - cached_input_tokens`;
- `output_tokens`;
- `reasoning_output_tokens`;
- total reported tokens;
- elapsed wall time.

Before the 24 task executions, one baseline/treatment null pair instructs the
assistant to return fixed JSON without calling a tool. Its input-token
difference estimates the static treatment tool-schema tax. This probe is not a
task outcome. Primary token results remain the raw intent-to-treat totals;
schema-adjusted values, if derived, are labeled descriptive and never replace
the real product cost.

Primary efficiency comparisons are paired per task:

1. total input-token difference and ratio;
2. uncached input-token difference and ratio;
3. output and reasoning-token differences;
4. unique source files inspected;
5. source-inspection command/tool count;
6. elapsed time as descriptive evidence only.

The report gives raw paired values, medians, sums, win/tie/loss counts and a bootstrap interval for the median paired ratio. It does not claim latency or token guarantees from 12 tasks.

## 9. Predeclared interpretation

The run is useful even if cidx does not win. Interpret the evidence in this order:

1. correctness and required-group coverage;
2. hard-negative and material-error behavior;
3. evidence journey and cidx utilization;
4. token usage;
5. elapsed time.

Provisional diagnostic signals, frozen before execution:

- `correctness_safe`: cidx has no fewer complete tasks and no more incorrect tasks than baseline;
- `accuracy_helpful`: cidx converts at least two baseline non-complete tasks to complete while converting none in the opposite direction;
- `token_reduction_directional`: with at least four dual-complete pairs, the median uncached-input ratio is at most `0.90`, the median paired difference is below zero, and at least half the pairs are non-increasing;
- `token_reduction_supported`: the directional conditions hold and the bootstrap interval's upper bound is below `1.0`;
- `exploration_reduction_observed`: among dual-complete pairs, median unique source files or source-inspection actions falls by at least `20%` without lower required-group coverage;
- `optional_tool_value_observed`: at least one complete cidx-arm task both uses cidx evidence and avoids a baseline false lead or materially shortens the evidence journey.

The earlier `0.85` raw-total ratio remains a reported stretch observation, not
a gate. These are diagnostic labels, not release gates or a weighted product
score. Any label with fewer than four relevant pairs is
`INSUFFICIENT_DENOMINATOR` rather than pass/fail.

## 10. Stop and invalidation rules

Stop before assistant execution if:

- either corpus commit/tree hash is wrong or dirty;
- the question-set or task-manifest digest differs;
- Codex authentication or requested model is unavailable;
- Arm A has any MCP server or Arm B has anything other than cidx;
- cidx does not expose exactly `status`, `search`, `read_span`, and `reindex`;
- the cidx status is stale, wrong-root, or not FTS-default;
- the prompt, output schema, sandbox, effort, timeout or ordering cannot be held equal;
- either arm can modify source or use external network;
- the treatment cannot pin MCP approval to the one recorded cidx binary without
  enabling any other MCP server or approval reviewer.

The stop check freezes the complete cidx tool exposure—names, descriptions and
parameter schemas—and records its digest before the first task. It also
requires the null schema-overhead probe to complete before task execution.

Invalidate only the affected pair if the process is killed externally, the JSONL stream is corrupt, or controls differ. Operational failures produced under valid controls stay in the denominator.

## 11. Artifact layout

Tracked inputs:

```text
docs/implementation/ASSISTANT-AB-TEST-PLAN-V1.md
testdata/retrieval/assistant-ab-chi-rhf-v1.json
schemas/evaluation/assistant-answer.schema.json
```

Ignored local outputs:

```text
.cidx/test/assistant-ab/<run-id>/
  run-manifest.json
  controls.json
  tool-schema.json
  <task-id>/<arm>/events.jsonl
  <task-id>/<arm>/stderr.txt
  <task-id>/<arm>/final.json
  <task-id>/<arm>/observation.json
  blinded-grading.jsonl
  paired-results.jsonl
  aggregate.json
  report.md
  checksums.json
```

The final tracked evidence report may contain metrics, hashes and concise excerpts needed to support conclusions. Raw source/tool transcripts remain ignored local sensitive artifacts unless separately approved.

## 12. Review and amendment rule

Before any Codex CLI assistant run, this draft is sent unchanged to one side-panel ChatGPT conversation and one side-panel Grok conversation. Both reviewers receive the product context, manifest summary, scoring rules and explicit questions about confounds, fairness, token accounting, grading and stop rules.

Accepted corrections are applied here and recorded below. A material change after the first assistant execution requires a new plan/manifest version; the current run is never silently reinterpreted.

### External review record

Both reviewers received the same 6,105-character self-contained protocol on
2026-08-20 before any Codex CLI assistant run.

| Reviewer | Conversation | Verdict | Accepted must-fix points |
| --- | --- | --- | --- |
| ChatGPT | `https://chatgpt.com/c/6a86c914-2d18-83e8-9b86-d29de2922b2a` | `REVISE_THEN_PROCEED` | Declare raw intent-to-treat tokens primary; freeze full tool schema; blind final-answer grading before journey; keep failures ungradable; run pairs back-to-back; accept one run only as diagnostic |
| Grok | `https://grok.com/c/d1f4ad8f-65de-49fb-a3c2-fbd026d4bd84` | `REVISE_THEN_PROCEED` | Measure tool-schema overhead with a null pair; blind arm identity; document one-run variance; make hard-negative reliance explicit; require denominator floors and full per-pair artifacts |

Disposition:

- Accepted: all common controls, blind grading, explicit ungradable semantics,
  full tool-schema freeze, back-to-back pairing, null probe, denominator floor,
  and hard-negative/source-span clarifications.
- Reconciled: ChatGPT advised never subtracting schema overhead; Grok advised
  reporting schema-adjusted tokens. Raw intent-to-treat tokens are primary
  because schema cost is real, while the null probe and any adjusted value are
  secondary diagnostics.
- Adjusted: the original 15% label becomes a 10% uncached-input directional
  label plus a stricter bootstrap-supported label; the 15% result remains a
  visible stretch observation.
- Deferred: repeat trials and broader repositories are useful for later
  confirmation but are not blockers for this owner-authorized diagnostic.

Execution preflight note: the first null probe was rejected before model
generation because the provider's strict structured-output dialect requires
every object property to appear in `required`, with optional values expressed
as nullable types. `start_line` and `end_line` were therefore made required and
nullable, and the null prompt was updated accordingly. No task answer, usage,
or treatment observation was produced by the rejected probe.

## 13. Completion evidence

This unit is complete when:

- both external reviews and dispositions are recorded;
- all valid 24 executions or their retained operational failures exist;
- answer/evidence grading has explicit denominators and task-level reasons;
- token and journey reducers reconcile to raw JSONL;
- the report states whether to proceed, correct cidx, or expand the assistant sample;
- no retrieval setting, existing question truth, corpus source or prior result was changed.
