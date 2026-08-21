# Paired Codex CLI Assistant A/B Test Plan — Version 5

- Status: `frozen_for_execution`
- Owner: `/root`
- Manifest: [`assistant-ab-chi-rhf-v5.json`](../../testdata/retrieval/assistant-ab-chi-rhf-v5.json)
- Answer schema: [`assistant-answer.schema.json`](../../schemas/evaluation/assistant-answer.schema.json)
- Blind-grade schema: [`assistant-blind-grade.schema.json`](../../schemas/evaluation/assistant-blind-grade.schema.json)
- Scope: Phase 14 diagnostic evidence only

## 1. Purpose and sole intervention

Version 4 proved that locator-only search preserves answer correctness and
removes source/diagnostic response excess, but it did not establish token
benefit. Both arms were 12/12 complete; the paired model-total median ratio was
1.044 and only 4/12 pairs were non-increasing. Treatment still made 23
searches and 39 reads, including six invalid ranges and four duplicate reads.

The [post-result reviews](evidence/phase-14/assistant-ab-v4-external-review.md)
independently selected the same next test. V5 changes only one conditional,
assistant-facing orchestration paragraph. No product, tool schema, retrieval,
ranking, query planner, result depth policy, corpus, truth, or grader changes.

## 2. Frozen prompt intervention

The exact paragraph inserted into the otherwise byte-identical V4 prompt is:

> If the cidx MCP tools are available, begin with exactly one cidx `search`
> call using `max_inline_bytes=0`. Treat search only as locator discovery. Do
> not repeat an identical search. For any selected locator, first call
> `read_span` using exactly the returned `start_line` and `end_line`; do not
> widen the range before reading it. Perform at most one additional refined
> search, and only when you can identify specific material evidence that is
> still missing. Do not repeat an identical `read_span`. Stop requesting
> additional source once every material answer claim has direct cited
> repository evidence.

This is one search-to-evidence journey policy. It does not change server-side
range behavior or block the assistant from ordinary local tools. The same
complete prompt is supplied to both arms; the conditional cidx branch is
inactive when the server is absent.

## 3. Frozen arms and controls

Both arms use the retained official npm `@openai/codex` 0.148.0 native
darwin-arm64 binary, `gpt-5.6-sol`, high reasoning, fresh ephemeral sessions,
read-only sandboxing, ignored user configuration and repository rules, the
same answer schema, disabled browser/apps/computer/web/multi-agent features,
and byte-identical task prompts.

Baseline exposes no MCP server. Treatment exposes only cidx and must invoke
its FTS `search` as the first repository-discovery action. Hybrid, reindex,
shell cidx, network actions, and provider credentials remain prohibited.

Treatment keeps the V4 locator-only structured response, current
caller-selected `k` semantics with default 10, the same FTS planner and
ranking, and unchanged `read_span`. Search source/text bytes remain zero.

## 4. Frozen panel and schedule

V5 retains V4's 12 question IDs, question digests, truths, critical/general
cohorts, approved Go and TypeScript/TSX corpus revisions, task order, and arm
order: Go 4, TypeScript 4, TSX 4; lexical 3, semantic 5, mixed 4; six
baseline-first and six treatment-first pairs.

All 24 scored turns are new and each pair runs back-to-back. No V4 baseline or
treatment observation is reused. No selective retry or replacement is
allowed.

## 5. Isolation, validity, and prompt noncompliance

Each turn receives a fresh opaque temporary Git copy of its pinned repository.
Each treatment receives a fresh private copy of the frozen cidx state outside
the assistant working directory. Source and state identities are checked
before and after, then temporary roots are removed.

Preflight verifies corpus/state identities, FTS-only configuration, exact
four-tool discovery, status, Codex login, schemas, plan/manifest/runner/scorer
identities, and structured-only representation. Two unscored schema probes
verify output-schema behavior before any scored turn.

The existing operational validity rules remain: timeout, malformed final JSON,
tool failure, forbidden action, source/state mutation, or first-search
noncompliance stays in the denominator and is never selectively rerun.

The new prompt clauses are the intervention under test. Repeated searches,
repeated or widened reads, too many refinements, or continuing after evidence
are recorded as outcomes of the prompt; they do not automatically make an
otherwise operationally valid turn ungradable.

## 6. Blind correctness

After all 24 turns, deterministic arm-opaque IDs and one grading packet per
corpus are created. One grader invocation per packet receives question truth,
assistant JSON, and cited source excerpts but not arm, order, tokens, cidx use,
or journey. Outcomes remain `complete`, `partial`, `incorrect`, or
operationally `ungradable`. Grades are fixed before restoring the blind key.

V1–V4 manifests, runs, journeys, grades, and reports remain unchanged.

## 7. Stage-separated measurement

Reducer v3 is frozen before execution and records three separate layers.

Orchestration mechanism:

- first-search `max_inline_bytes=0` adherence;
- initial plus at most one refined search;
- repeated query/mode and duplicate argument counts;
- duplicate read ranges and `INVALID_RANGE` attempts;
- exact locator-range use for the first read of each path/content hash;
- inspection actions after frozen required-group evidence becomes complete;
  and
- the narrowly mechanical adherence count. Semantic justification for a
  refinement is reviewed separately and is never inferred from event counts.

Locator and evidence quality retain V4's explicit denominators: first/any
search required-group coverage, useful rank, locator utilization and false
leads, read precision and citation utilization, complete evidence coverage,
gross/unique source bytes, and redundancy.

End-to-end measures remain blind outcome, required-group coverage, false
claims, authoritative official token fields, repository-inspection actions,
and final evidence provenance. Event-output bytes remain diagnostics and are
not called model-visible tokens. No weighted score is created.

## 8. Predeclared interpretation

Correctness preservation requires no complete-to-noncomplete reversal, no new
material false claim, and no loss of required-group evidence coverage.

The existing efficiency bar remains unchanged and separate: at least eight
dual-complete pairs, median treatment/baseline model-total ratio at most 0.85,
and at least two thirds of dual-complete ratios no greater than 1.0. The
fixed-seed bootstrap remains diagnostic only.

Mechanism improvement is reported, not blended into that bar. The intended
direction is 12/12 first-search argument compliance, 12/12 at most two
searches, zero repeated query/mode, zero duplicate read range, zero invalid
range, and a higher exact-initial-locator read rate than V4's 2/23. Evidence
coverage must not fall. These measures show whether the prompt controlled the
observed behavior; they do not by themselves establish token benefit.

Possible conclusions are kept distinct:

- correctness and token gate pass: cidx marginal efficiency is supported on
  this panel;
- correctness and mechanism improve but token gate fails: the prompt changed
  the journey, but token benefit remains unestablished and the next bottleneck
  must be isolated;
- correctness holds but mechanism does not improve: reject the prompt as an
  unreliable control; or
- correctness regresses: reject the intervention regardless of token counts.

## 9. Third-experiment boundary

V4 is the first and V5 is the second of at most three post-V3 full experiments
authorized for this sequence. V6 is not preselected. It may be frozen only
after V5's complete result and side-panel review identify one remaining loss
that does not require a new corpus, paid provider call, public contract change,
or other owner decision.

V5 remains a controlled diagnostic on exposed calibration questions. It is
not `core_retrieval`, another host, population evidence, or
`release_candidate` promotion.
