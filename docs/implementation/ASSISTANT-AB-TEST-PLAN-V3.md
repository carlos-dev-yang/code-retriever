# Paired Codex CLI Assistant A/B Test Plan — Version 3

- Status: `frozen_for_execution`
- Owner: `/root`
- Manifest: [`assistant-ab-chi-rhf-v3.json`](../../testdata/retrieval/assistant-ab-chi-rhf-v3.json)
- Answer schema: [`assistant-answer.schema.json`](../../schemas/evaluation/assistant-answer.schema.json)
- Blind-grade schema: [`assistant-blind-grade.schema.json`](../../schemas/evaluation/assistant-blind-grade.schema.json)
- Scope: Phase 14 diagnostic evidence only

## 1. Purpose and predecessor

Version 2 preserved correctness in all three MCP-compliant treatment turns, but
nine of twelve treatment assistants interpreted “cidx search” as a shell CLI
command. The exact MCP launcher and status preflight were healthy. V2 therefore
measured an interface-instruction ambiguity and did not reach its required
eight-pair denominator.

Version 3 preserves V2 unchanged and changes only the prompt wording necessary
to identify the exposed MCP tool unambiguously. Its estimand is:

> When the same assistant must start repository discovery with the exposed cidx
> MCP FTS search, does it preserve correctness while reducing model tokens and
> the model-visible investigation journey versus ordinary local tools?

## 2. Frozen arms and exact shared prompt

Both arms use Codex CLI `gpt-5.6-sol`, high reasoning, fresh ephemeral sessions,
read-only sandboxing, ignored user configuration and repository rules, the same
answer schema, disabled browser/apps/computer/web/multi-agent features, and
byte-identical prompts.

The task runner is the official npm `@openai/codex` `0.148.0` native
darwin-arm64 binary, SHA-256
`b0308517b20543012fa2171aa3d46ce455a7456c4eb2a552ab9468ba4eeb1e50`.
It is installed only in ignored evaluation state and is used identically by both
arms. This was frozen before the first V3 scored turn because the ChatGPT app's
`0.148.0-alpha.21` bundled CLI, used by V2, began rejecting new 5.6-sol sessions
as requiring a newer CLI after V2 had completed. A model/schema probe with the
official stable binary succeeded; V2 is not used as a paired comparator for V3.

The intervention paragraph is:

```text
If the exposed MCP tool whose server is `cidx` and whose tool name is `search`
is available (`mcp__cidx__search` in tool-calling form), invoke that MCP tool as
your first repository-discovery action. Use mode `fts`. Do not run `cidx` as a
shell command and do not test its shell availability with `command -v`, `which`,
`type`, wrappers, or any equivalent. Determine availability only from the
exposed MCP tool list. Make the MCP search call before rg, grep, find, directory
listing, direct source reads, or any other repository inspection. Treat its
results as navigation rather than authority, then verify decisive evidence with
the cidx MCP `read_span` tool or ordinary source reads. If that MCP tool is not
exposed, continue with ordinary local tools.
```

Baseline exposes no MCP server. Treatment exposes only the hash-pinned cidx MCP
with `status`, `search`, `read_span`, and `reindex`; `reindex` and hybrid are
prohibited. Voyage credentials are removed and the launcher cannot create a
provider client.

## 3. Frozen tasks and schedule

The 12 Version 2 questions, question digests, corpus commits, answer truths, and
pair order remain unchanged: Go 4, TypeScript 4, TSX 4; lexical 3, semantic 5,
mixed 4. Six pairs are baseline-first and six treatment-first, balanced by
language and approximately by query family. Each pair runs back-to-back.

## 4. Execution isolation

Every turn receives a fresh opaque temporary Git copy of the pinned corpus with
no `.cidx`, result, grading, or sibling-arm data. Every treatment receives a
fresh opaque private copy of the frozen cidx state outside the assistant working
directory. The launcher binds that state to the temporary source root. Source
commit/tree/status and private database hashes are recorded before and after,
then both temporary roots are removed.

The runner freezes the Codex, cidx, launcher, plan, manifest, answer schema, tool
schema, corpus, index, environment-name, sandbox, and prompt identities. Source
mutation, hidden-artifact exposure, unexpected MCP, or batch-schedule mismatch
invalidates the batch. Timeout, malformed output, tool failure, shell-cidx
attempt, or first-search noncompliance remains an ungradable observation and is
not selectively rerun.

## 5. Intervention compliance

The runner derives the first repository-discovery action from ordered
`item.started` events. A valid treatment must have `server=cidx, tool=search` as
that first action and must make no shell-cidx availability or execution attempt.
The MCP schema and an isolated `status` call must pass before the first scored
turn. Compliance rate is reported separately from retrieval correctness.

## 6. Blinded correctness

After all 24 turns, answers receive deterministic opaque IDs. A single blind
grader sees only corpus identity, question, frozen required groups and hard
negatives, assistant JSON, and its cited source excerpts. It cannot see arm,
order, tokens, cidx use, or journey. Outcomes are `complete`, `partial`,
`incorrect`, or operationally `ungradable`; exact required-group coverage and
unsupported/contradicted claims are retained. Grades are fixed before the blind
key is restored.

## 7. Machine-frozen journey

Before grading or arm restoration, the predeclared reducer writes
`journey-frozen.jsonl` under opaque IDs and records its SHA-256 and reducer
SHA-256. It mechanically records first discovery action; shell/cidx inspection
counts; model-visible shell/cidx output bytes; visible, cited, uncited, and
cidx-returned source paths; intersections; and fixed cidx usage class. No human
or arm-aware journey classification is permitted.

## 8. Token accounting

The final cumulative `turn.completed.usage` is authoritative:

- `model_total = input_tokens + output_tokens`;
- `uncached_input = input_tokens - cached_input_tokens`;
- reasoning tokens are an output subset and are not added twice;
- missing usage is `NA`, never zero.

Raw intent-to-treat totals include MCP schema and tool costs. The unscored schema
probe pair is reported but never subtracted.

## 9. Interpretation rules

Report in order: compliance; blind correctness; required-group coverage and
false claims; cidx path use; paired model-total and uncached input; frozen
inspection actions/output bytes/paths; descriptive elapsed time.

Efficiency needs at least eight dual-complete pairs, median treatment/baseline
model-total ratio at most `0.85`, and at least two thirds of those ratios no
greater than `1.0`. Report a fixed-seed paired-task bootstrap interval as a
diagnostic that does not capture model stochasticity. No weighted quality score
or release gate is created.

## 10. Execution and retry rule

There are exactly 24 scored turns and two unscored schema probes. Scored turns
are never selectively rerun. Any correction after the first scored turn creates
a fully new manifest and 24-turn batch. Raw prompts, commands, JSONL, stderr,
answers, state checks, blind packets, grades, journey, pair data, aggregate,
report, and checksums remain in ignored local state; a tracked evidence summary
records their identities.

## 11. External review

Both side-panel reviewers examined the frozen V2 result and returned
`PROCEED_WITH_V3`. Grok approved the proposed full-batch correction. ChatGPT
required the exact model-visible MCP identifier and an explicit ban on all shell
availability checks; both are incorporated verbatim above. Neither reviewer
requested a new corpus or a release gate.

## 12. Interpretation boundary

Version 3 is a bounded paired diagnostic on these 12 frozen questions. It can
show whether cidx helped these Codex CLI investigations and whether further tool,
query, or response-budget correction is warranted. It cannot establish
`core_retrieval`, population-level assistant benefit, or `release_candidate`.
