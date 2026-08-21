# Paired Codex CLI Assistant A/B Test Plan — Version 6

- Status: `frozen_for_execution`
- Owner: `/root`
- Manifest: [`assistant-ab-chi-rhf-v6.json`](../../testdata/retrieval/assistant-ab-chi-rhf-v6.json)
- Answer schema: [`assistant-answer.schema.json`](../../schemas/evaluation/assistant-answer.schema.json)
- Blind-grade schema: [`assistant-blind-grade.schema.json`](../../schemas/evaluation/assistant-blind-grade.schema.json)
- Scope: Phase 14 diagnostic evidence only

## 1. Purpose and sole intervention

V5 preserved blind correctness and sharply reduced search/read/source activity,
but its paired model-total median ratio was 0.954 and only 6/12 pairs were
non-increasing, so the frozen efficiency gate still failed. Three treatment
calls copied the chosen locator path and exact line range but omitted the
schema-required `expected_sha256`; each failed before source delivery and was
immediately retried with the same range and the hash.

The [V5 post-result reviews](evidence/phase-14/assistant-ab-v5-external-review.md)
independently approved the same final test. V6 changes only one sentence in
the shared conditional cidx orchestration paragraph. No product, schema,
retrieval, ranking, planner, result-depth, corpus, truth, runner, reducer, or
grader change is permitted.

## 2. Frozen prompt intervention

Insert this exact sentence immediately after V5's exact-range instruction:

> When calling `read_span`, pass `path`, `start_line`, `end_line`, and
> `expected_sha256` exactly as returned by the selected locator.

The complete conditional paragraph is therefore:

> If the cidx MCP tools are available, begin with exactly one cidx `search`
> call using `max_inline_bytes=0`. Treat search only as locator discovery. Do
> not repeat an identical search. For any selected locator, first call
> `read_span` using exactly the returned `start_line` and `end_line`; do not
> widen the range before reading it. When calling `read_span`, pass `path`,
> `start_line`, `end_line`, and `expected_sha256` exactly as returned by the
> selected locator. Perform at most one additional refined search, and only
> when you can identify specific material evidence that is still missing. Do
> not repeat an identical `read_span`. Stop requesting additional source once
> every material answer claim has direct cited repository evidence.

The remainder of the V5 prompt is byte-identical. This sentence documents an
existing required tool field; it does not change the locator wire, source
return, or server behavior. The same complete prompt is supplied to both
arms, and the conditional branch is inactive when cidx is absent.

## 3. Frozen arms, panel, and controls

Both arms retain the official npm `@openai/codex` 0.148.0 native darwin-arm64
binary, `gpt-5.6-sol`, high reasoning, fresh ephemeral sessions, read-only
sandboxing, ignored user configuration and repository rules, the same answer
schema, disabled browser/apps/computer/web/multi-agent features, and identical
task prompts.

Baseline exposes no MCP server. Treatment exposes only cidx and must invoke
its FTS `search` as the first repository-discovery action. Hybrid, reindex,
shell cidx, network actions, and provider credentials remain prohibited.

V6 retains V5's locator-only structured response, caller-selected `k`
semantics with default 10, FTS planner and ranking, unchanged `read_span`, and
zero search source/text bytes. It retains the same 12 question IDs and
digests, truths, cohorts, approved pinned Go and TypeScript/TSX corpora, task
order, and arm order. All 24 scored turns are new and run pair-by-pair. No V5
observation is reused and no selective retry or replacement is allowed.

## 4. Isolation and validity

Each turn receives a fresh opaque temporary Git copy of its pinned repository.
Each treatment receives a fresh private copy of the frozen cidx state outside
the assistant working directory. Source and state identities are checked
before and after, then temporary roots are removed.

Preflight verifies corpus/state identities, FTS-only configuration, exact
four-tool discovery, status, Codex login, schemas, plan/manifest/runner/scorer
identities, and structured-only representation. Two unscored schema probes
verify output-schema behavior before scored execution.

Timeout, malformed final JSON, tool failure, forbidden action, source/state
mutation, or first-search noncompliance remains in the denominator and is
never selectively rerun. Prompt noncompliance is recorded as an experiment
outcome rather than automatically making an otherwise valid turn ungradable.

## 5. Blind correctness and measurement

After all 24 turns, deterministic arm-opaque IDs and one grading packet per
corpus are created. One tool-free grader invocation per packet receives truth,
assistant JSON, and cited excerpts, but not arm, order, tokens, cidx use, or
journey. Grades are fixed before restoring the blind key.

Reducer v3 remains unchanged. It reports orchestration, locator/evidence, and
end-to-end layers separately, including searches, all read attempts, failure
reasons, exact first locator ranges, repeated operations, source volume and
redundancy, read precision, citation utilization, blind correctness, and
official token fields. Event bytes are diagnostic, not model-visible tokens.
No weighted score is created.

## 6. Predeclared decisions

Correctness is preserved only with no complete-to-noncomplete reversal, all
frozen required groups covered, and no new unsupported or contradicted
material claim.

The hash-field mechanism succeeds only with:

- zero selected-locator `read_span` calls omitting `expected_sha256`;
- zero `INVALID_READ_SPAN_REQUEST` failures caused by that omission; and
- zero immediate same-range recovery calls whose sole change is the hash.

The official token gate is unchanged: at least eight dual-complete pairs,
paired median treatment/baseline model-total ratio at most 0.85, and at least
8/12 dual-complete ratios no greater than 1.0. Aggregate totals, uncached
input, fixed-seed bootstrap, language/cohort slices, and the three formerly
affected tasks are diagnostics rather than substitutes for that gate.

Possible conclusions stay separate:

- correctness, hash mechanism, and token gate pass: cidx marginal efficiency
  is supported on this exposed panel;
- correctness and hash mechanism pass but token gate fails: the known
  prompt-level mechanical waste is corrected, but cidx-first treatment has
  not established material model-token reduction on this panel;
- hash mechanism fails: the explicit field instruction is not a reliable
  assistant control; or
- correctness regresses: reject the intervention regardless of token counts.

## 7. Mandatory final boundary

V6 is the third and final post-V3 experiment in the authorized V4–V6 series.
The series stops after this complete result and its final ChatGPT/Grok
interpretation under every outcome. There is no V7 corrective rerun. Any
later product, retrieval, prompt, corpus, or assistant investigation requires
a separately scoped owner decision.

V6 is diagnostic evidence on exposed calibration questions. It cannot
establish `core_retrieval`, another-host/population evidence, or
`release_candidate` promotion.
