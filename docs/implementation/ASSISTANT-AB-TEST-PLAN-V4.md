# Paired Codex CLI Assistant A/B Test Plan — Version 4

- Status: `frozen_for_execution`
- Owner: `/root`
- Manifest: [`assistant-ab-chi-rhf-v4.json`](../../testdata/retrieval/assistant-ab-chi-rhf-v4.json)
- Answer schema: [`assistant-answer.schema.json`](../../schemas/evaluation/assistant-answer.schema.json)
- Blind-grade schema: [`assistant-blind-grade.schema.json`](../../schemas/evaluation/assistant-blind-grade.schema.json)
- Scope: Phase 14 diagnostic evidence only

## 1. Purpose and sole intervention

Version 3 achieved 12/12 treatment compliance and preserved or improved blind
correctness, but treatment used 37.2% more total model tokens and 50.6% more
uncached input. The treatment exposed source-bearing search results, extensive
diagnostics, and duplicate text/structured result representations before later
`read_span` calls.

Version 4 preserves every V3 task, prompt, arm order, model, reasoning setting,
corpus, FTS planner/rank, caller-selected `k`, and `read_span` rule. Its sole
declared treatment change is the accepted Phase 13 response contract:

```text
search -> one structured list of compact locators, zero source bytes
assistant selection -> read_span -> selected source evidence
```

V4 estimates whether that response-only reduction preserves correctness while
reducing the end-to-end investigation journey. It does not test a new retrieval
algorithm.

## 2. Frozen arms and shared prompt

Both arms use the retained official npm `@openai/codex` 0.148.0 native
darwin-arm64 binary (SHA-256
`b0308517b20543012fa2171aa3d46ce455a7456c4eb2a552ab9468ba4eeb1e50`),
`gpt-5.6-sol`, high reasoning, fresh ephemeral sessions, read-only sandboxing,
ignored user configuration and repository rules, the same answer schema,
disabled browser/apps/computer/web/multi-agent features, and byte-identical
task prompts.

Baseline exposes no MCP server. Treatment exposes only cidx and must invoke its
FTS `search` as the first repository-discovery action. The exact V3 prompt is
preserved byte-for-byte. Hybrid, reindex, shell cidx, network actions, and
provider credentials remain prohibited.

Treatment search returns one structured representation. It contains only
ordered, deduplicated nine-field locators and no source, signatures, scores, or
planner/profile diagnostics. `read_span` is the only source-bearing cidx call.
The manifest freezes `structured`; the runner rejects a conflicting CLI option.

## 3. Frozen panel and schedule

The same 12 critical/general-v2 questions, digests, truths, hard negatives,
corpus commits, and task order are retained: Go 4, TypeScript 4, TSX 4;
lexical 3, semantic 5, mixed 4. Six pairs are baseline-first and six are
treatment-first. Each pair runs back-to-back.

All 24 scored turns are new. V3 baseline observations are not reused because
model stochasticity requires a contemporaneous paired baseline. No question,
truth, cohort, or prior result is overwritten.

## 4. Isolation and compliance

Each turn receives a fresh opaque temporary Git copy of its pinned repository.
Each treatment also receives a fresh private copy of the frozen cidx state
outside the assistant working directory. Source commit/tree/status and state
database hashes are recorded before and after; temporary roots are removed.

Before scored execution, the runner verifies both approved corpus bindings,
clean FTS-only state, exact four-tool discovery, a successful status call,
Codex login, schemas, prompt/plan/manifest hashes, and structured-only result
selection. Two unscored schema probes then verify arm output-schema behavior.

A valid treatment has cidx `search` as its first repository-discovery action,
makes no shell-cidx attempt, does not call hybrid/reindex, and leaves source and
state unchanged. Timeout, malformed final JSON, tool failure, forbidden action,
or first-search noncompliance remains an observation; no selective rerun is
allowed.

## 5. Blinded correctness and versioned artifacts

After all 24 turns, the scorer creates deterministic arm-opaque IDs and two
corpus grading packets. One grader invocation per packet receives only corpus,
question, frozen required groups/hard negatives, assistant JSON, and cited
source excerpts. It cannot see arm, order, tokens, cidx use, or journey.

Outcomes remain `complete`, `partial`, `incorrect`, or operationally
`ungradable`. Grades are fixed before the blind key is restored. Raw turns,
packets, grades, reducer outputs, pair rows, aggregate, report, and checksums
are stored under the unique V4 run ID. V1–V3 artifacts remain unchanged.

## 6. Stage-separated measurement

Reducer v2 records transport observations without claiming event-envelope
bytes are model-visible. Every task remains in the denominator.

Locator/navigation measures:

- first-search and any-search required-group coverage;
- complete first-search locator hit;
- first useful rank;
- search/refinement counts and returned locator count;
- duplicate locator exposure;
- locator-to-read selection and locator-to-citation utilization;
- navigation false leads; and
- structured, text, envelope, and source bytes separately. Search source bytes
  must be zero and text result bytes must be zero.

Evidence measures:

- `read_span` required-group coverage and complete evidence hit;
- read precision, citation utilization, and evidence false leads;
- gross, unique, and redundant source bytes; and
- the first inspection action at which complete required evidence existed.

End-to-end measures:

- blinded outcome, required-group answer coverage, false/contradicted claims,
  and paired correctness conversions;
- authoritative input, cached input, uncached input, output, reasoning subset,
  and model-total tokens;
- search, `read_span`, ordinary-read, and total inspection actions; and
- final evidence provenance.

No weighted quality score is created.

## 7. Interpretation thresholds

Correctness and efficiency remain separate. Report all pairs and also the
dual-complete subset. Efficiency is considered clearly helpful only with at
least eight dual-complete pairs, median treatment/baseline model-total ratio at
most 0.85, and at least two thirds of dual-complete ratios no greater than 1.0.
A fixed-seed paired-task bootstrap interval is diagnostic only.

Failure to meet that threshold does not mean search is nonfunctional or block
the product forever. The stage scorecards identify the first remaining loss:

- poor first-search coverage -> retrieval/query correction candidate;
- good locator coverage but excess searches -> orchestration candidate;
- good locator selection but excess or imprecise reads -> evidence-read
  guidance/bounds candidate; or
- compact journeys without token benefit -> marginal product value is not yet
  demonstrated on this panel.

## 8. Change discipline after V4

V4 changes no prompt strategy, fixed result depth, query terms, planner,
ranking, routing, embedding, or corpus. A later V5 or V6 may freeze exactly one
bounded change after V4 evidence and side-panel review. Paid hybrid/query
embedding remains out of scope without separate explicit authorization.

At most three post-V3 full experiments are authorized in this work sequence.
Each receives its own manifest, complete paired run, report, review record, and
commit. An improvement stops early when evidence shows no justified next change
or when the remaining change needs a new product decision.

## 9. Interpretation boundary

V4 is a controlled diagnostic on 12 existing calibration questions. It can
show whether compact locator delivery improves these Codex CLI investigations
and which stage needs the next correction. It cannot establish population-level
assistant benefit, `core_retrieval`, another host, or `release_candidate`.
