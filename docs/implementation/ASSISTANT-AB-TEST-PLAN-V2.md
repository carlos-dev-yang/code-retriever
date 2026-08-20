# Paired Codex CLI Assistant A/B Test Plan — Version 2

- Status: `completed_with_intervention_noncompliance`
- Owner: `/root`
- Manifest: [`assistant-ab-chi-rhf-v2.json`](../../testdata/retrieval/assistant-ab-chi-rhf-v2.json)
- Answer schema: [`assistant-answer.schema.json`](../../schemas/evaluation/assistant-answer.schema.json)
- Blind-grade schema: [`assistant-blind-grade.schema.json`](../../schemas/evaluation/assistant-blind-grade.schema.json)
- Scope: Phase 14 diagnostic evidence only

## 1. Why Version 2 exists

Version 1 completed all 24 assistant turns and blind grading, but the treatment
assistant called cidx zero times. It therefore measured optional-tool adoption,
not retrieval value. Baseline was complete on 12/12 questions; treatment was
complete on 11/12 and partial on one, with no cidx-derived evidence. Those
artifacts remain immutable as a pilot and are not silently replaced.

The final side-panel review also required per-execution index isolation,
auditable tool exposure, corrected task strata, corrected token accounting, and
more exact source-exposure evidence. Version 2 addresses those requirements and
tests a different, explicit estimand:

> When the same prompt tells the assistant to use cidx first if it is available,
> does the cidx-enabled assistant preserve answer correctness while reducing
> the source investigation and model-token journey?

## 2. Fixed arms and shared prompt

Both arms use Codex CLI `gpt-5.6-sol`, high reasoning, a fresh ephemeral
session, a read-only sandbox, ignored user configuration and repository rules,
the same output schema, disabled browser/apps/computer/web/multi-agent features,
and byte-identical task prompts.

The shared prompt says:

```text
If a local code-search tool named cidx is available, your first
repository-discovery action must be one cidx search call. Make that call before
any rg, grep, find, directory listing, direct source read, or other repository
inspection. Treat its results as navigation rather than authority, then verify
decisive evidence with cidx read_span or ordinary source reads. If cidx is
unavailable, continue with ordinary local tools.
```

The baseline has no MCP server. Treatment has only the hash-pinned cidx MCP and
must make the initial FTS search; Voyage credentials are removed, hybrid search
is prohibited, and no paid call can occur. After the initial search, the
assistant remains free to reject weak results and use ordinary tools. This is a
tool-aware required-first-search comparison, not a claim about automatic
discovery or forced acceptance of a weak cidx result.

## 3. Frozen task panel and order

The same versioned 12 questions are retained so the result is comparable to the
pilot without editing prior question or result artifacts.

- Languages: Go 4, TypeScript 4, TSX 4.
- Query families: lexical 3, semantic 5, mixed 4.
- Modifiers: multi-requirement 3, contract disambiguation 4, hard negative 1.

Each pair runs back-to-back. The six baseline-first tasks are
`chi-new-router`, `chi-g06-basic-auth`,
`rhf-t02-controlled-field-lifecycle`, `rhf-t10-dotted-path-types`,
`rhf-controller-component`, and `rhf-x08-form-state-props`. The other six are
treatment-first. This balances language and places two lexical, two semantic,
and two mixed questions in baseline-first as closely as the odd 3/5/4 totals
permit. The exact sequence is in the manifest.

## 4. Isolation and mutation controls

Every assistant execution receives a fresh opaque temporary Git worktree copy
containing only the pinned source snapshot and no `.cidx`, grading bundle, run
artifact, or sibling-arm output. The temporary path does not encode arm identity.

Every treatment execution also receives a separate opaque state directory,
copied from the frozen FTS index. A dedicated evaluation MCP launcher opens that
state with `OpenWorkspaceLocal`; its source root is the treatment's temporary
source copy and its state root is outside the assistant working directory. The
launcher never constructs a Voyage client. Neither a mistaken `reindex` nor a
database write can affect another execution or the frozen source index.

The harness records source commit/tree/status before and after, private state
database hashes before and after, executable hashes, full cidx tool schema,
effective arguments, environment-variable-name allowlist, and transcript
digests. A successful source mutation, model-generated network access, prompt or
tool mismatch, hidden-artifact exposure, transcript loss, or schedule violation
invalidates the batch. Runtime timeout, malformed output, tool failure, no-use,
blocked write, or an attempted private-state reindex remains an observed
ungradable outcome and is never selectively rerun.

## 5. Tool approval and exposure

Non-interactive Codex requires an approval decision for unannotated MCP tools.
Only the exact local launcher is set to
`default_tools_approval_mode="approve"`; no user or automatic reviewer runs and
no reviewer tokens enter treatment. The launcher exposes exactly `status`,
`search`, `read_span`, and `reindex`, and the complete names, descriptions, and
parameter schemas are hashed before execution. The prompt prohibits `reindex`
and hybrid search. Baseline receives no MCP definition.

## 6. Correctness and blind grading

The Version 1 grading rubric remains fixed: `complete`, `partial`, `incorrect`,
and `ungradable`, with exact required-group coverage and separate unsupported and
contradicted claims. A complete answer covers every frozen required group and
contains no material false claim. Operational failures alone are ungradable.

After all turns, answers receive deterministic opaque IDs. A separate Codex
grader sees only repository identity, question, frozen truth, the assistant JSON,
and the exact cited source excerpts. It does not see arm, order, tokens, cidx use,
or tool journey. Grades are locked before arm restoration and journey analysis.

## 7. Token accounting

The final cumulative `turn.completed.usage` object is authoritative:

- `input_total = input_tokens`;
- `cached_input = cached_input_tokens`;
- `uncached_input = input_total - cached_input`;
- `output_total = output_tokens`;
- `reasoning_output` is a diagnostic subset of output, not added again;
- `model_total = input_total + output_total`.

Missing usage is `NA`, never zero. Raw treatment totals include all real tool
schema and MCP costs. A no-tool null pair is descriptive only and is not
subtracted from the primary result.

## 8. Search-journey accounting

The journey reducer is predeclared code, not a human assessment. Immediately
after the 24 turns, and before the blind key is used to restore arm identity, it
reads each ordered JSONL transcript and final answer and writes an immutable
`journey-frozen.jsonl` keyed only by opaque blind ID. Its SHA-256 and reducer
SHA-256 are recorded before grading.

The reducer deterministically records:

- the first repository-discovery action and whether it was a cidx search;
- completed shell repository-inspection, cidx search, and cidx `read_span`
  action counts;
- UTF-8 bytes of model-visible repository-inspection output, split into shell
  and cidx output;
- deduplicated source paths visible in those outputs;
- final cited source paths and the mechanical set difference of visible but
  uncited paths;
- cidx-returned paths, their intersection with final cited paths, and the
  mechanical usage class `no_use`, `no_cited_path`, `navigation`, or
  `read_span_cited`.

The report does not use subjective labels such as “false lead,” “accepted
evidence,” or “first point all requirements were known.” No journey field is
manually edited after arm restoration. Source exposure means repository output
actually delivered to the model, not files scanned internally. Tool counts and
retrieval hits are not correctness; correctness remains the separate blind
grade.

## 9. Diagnostic interpretation

No weighted score or release gate is created. Report, in order:

1. complete/partial/incorrect/ungradable and required-group coverage;
2. complete-to-noncomplete reversals and false claims;
3. cidx adoption and evidence provenance;
4. paired model-total and uncached-input tokens;
5. paired model-visible source exposure and inspection actions;
6. elapsed time as descriptive evidence only.

At least eight dual-complete pairs are required for an efficiency label.
Directional token evidence requires a median paired model-total ratio at most
`0.85` and at least two thirds of pairs non-increasing. Report the fixed-seed
paired-task bootstrap interval, but it does not capture model-run stochasticity.
Accuracy is helpful only with at least two baseline non-complete to treatment
complete conversions, no reverse conversion, and no new operational failure.
Required-first-search value requires at least one treatment-complete answer with
a cidx-returned path cited in the final answer and an improved predeclared
machine-derived journey measure without correctness loss.

## 10. Execution and artifact rule

There are exactly 24 scored assistant executions and two unscored null probes.
There are no selective retries. Any correction after a scored turn starts
creates a new manifest and a complete new batch; the old batch remains preserved.
Raw JSONL, stderr, final JSON, prompts, commands, pre/post state, blind packets,
grades, paired results, aggregate, report, and checksums are retained in ignored
local state. The tracked report records their digests and the exact manifest,
runner, schemas, Codex CLI, launcher, product binary, corpus, and index identities.

This diagnostic can support a decision to continue, correct tool adoption/query
behavior, or stop. It cannot establish `core_retrieval` or `release_candidate`.

## 11. External pre-execution review

- Grok: `APPROVED_FOR_EXECUTION` on the isolated Version 2 plan and again on the
  final amendment.
- ChatGPT: initially `REVISE_THEN_PROCEED`; required an unambiguous first cidx
  search and arm-blind or deterministic freezing of journey measures.
- Resolution: both requirements are incorporated above and enforced by the
  runner/reducer. ChatGPT then returned `APPROVED_FOR_EXECUTION`. A treatment
  turn that does not start repository discovery with cidx is retained as an
  operationally ungradable observation and is not rerun.

## 12. Frozen execution result

- Run: `assistant-ab-v2-20260820T140000Z`.
- Baseline: 12/12 blindly graded complete.
- Treatment: 3/12 invoked MCP cidx first and all three were complete; the other
  nine attempted shell commands such as `cidx search` or `command -v cidx` and
  are operationally ungradable.
- Compliant model-total ratios: `0.844`, `0.745`, `1.418`; median `0.844`, but
  the required denominator of eight dual-complete pairs was not met.
- The result identifies prompt/tool-interface ambiguity, not retrieval failure,
  and supports no efficiency conclusion.
- The task model remained `gpt-5.6-sol` high. After all answers and journeys
  were frozen, the current CLI began rejecting new 5.6-sol sessions as requiring
  a newer CLI. Before viewing any grade, the single blind grader model was fixed
  to `gpt-5.5` high for both corpora; two failed 5.6-sol grader starts produced
  no grades.
- Both external reviewers recommended a complete Version 3 batch with the exact
  MCP tool identity made explicit. V2 artifacts remain immutable.
