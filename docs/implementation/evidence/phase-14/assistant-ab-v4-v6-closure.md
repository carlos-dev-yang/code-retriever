# Assistant A/B V4–V6 Closure and Handoff

- Date: 2026-08-22
- Phase: 14
- Status: three-experiment corrective series closed
- Scope: local auxiliary-MCP assistant-use evidence
- Next action authority: owner decision; no automatic continuation

## Executive conclusion

The three improvements corrected concrete response and tool-use defects, but
they did not establish the frozen end-to-end hypothesis that mandatory
cidx-first use consistently reduces Codex model tokens while preserving
correctness.

- V4 replaced source-heavy search responses with compact locators and
  source-only `read_span`. It removed the original response-volume defect.
- V5 added search/read orchestration guidance. It substantially reduced cidx
  calls and source delivery, but missed the token gate.
- V6 named the required locator hash. It eliminated every hash omission and
  failed read, but one treatment answer became partial and the paired token
  gate still failed.

The series therefore closes with **no assistant-efficiency claim**. It also
does not prove that cidx is useless. The measured product posture is narrower:
compact FTS locators and explicit source reads work as an auxiliary navigation
mechanism, but mandatory first use is not yet supported as a correctness-safe,
consistently token-saving host policy.

This is not a permanent release prohibition. It rejects a specific host-use
and efficiency claim on an exposed 12-task panel. Official core retrieval,
packaging, another-host evidence, and release-candidate promotion remain
separate gates.

## Starting problem

V3 preserved answer correctness but treatment consumed 37.2% more model
tokens. Search returned source and diagnostic metadata together, so the
assistant received candidate bodies before deciding what to inspect. The
first correction correctly separated two roles:

1. `search` supplies compact candidate locations; and
2. `read_span` supplies selected source evidence.

That contract remains valid. Later experiments show that reducing wire bytes
alone is not enough: assistant source-selection, stopping, caching, and final
claim discipline dominate the remaining end-to-end variance.

## Three committed improvement stages

| Stage | Sole improvement | Correctness | Paired model-total decision | Main measured result | Result commit |
| --- | --- | --- | --- | --- | --- |
| V4 | locator-only structured `search`; source-only `read_span` | 12/12 complete in both arms | median 1.044; 4/12 non-increasing — fail | cidx event payload -82.9% versus V3, but 62 calls and repeated/failed/imprecise reads remained | `ca39edf` |
| V5 | one conditional search/read orchestration paragraph; reducer v3 | 12/12 complete in both arms | median 0.954; 6/12 non-increasing — fail | calls 62→38 and gross read source 77,173→17,967 bytes; three required-hash omissions remained | `9cc1cae` |
| V6 | one sentence requiring locator `expected_sha256` | baseline 12 complete; treatment 11 complete/1 partial — fail | dual-complete median 0.977; 6/11 non-increasing — fail | zero omitted hashes/failed reads; calls rose to 41 and source to 28,022 bytes | `7490e28` |

Each stage used new complete paired observations, preserved the same approved
repositories/questions/order/model and frozen product behavior unless that
behavior was the declared intervention, and was committed before the next
stage. V4 and V5 post-result decisions and V6 closure were independently
reviewed in the existing ChatGPT and Grok side-panel conversations.

## What improved conclusively

### Response contract

The locator-only wire removed automatic source delivery without changing
ranking or result identity. V4 emitted zero search source/text bytes and
reduced cidx event payload 82.9% from V3. That implementation is not invalidated
by the later token result.

### Tool-call correctness

The V5/V6 orchestration paragraph made the first search explicit, used
`max_inline_bytes=0`, limited searches to two, eliminated identical searches,
and made initial reads use exact locator ranges. V6 then eliminated all three
V5 missing-hash failures: 24/24 reads supplied `expected_sha256` and succeeded.

### Navigation compactness

Across V4, V5, and V6 the first useful locator rank median remained 1. Median
treatment/baseline visible-source-path ratios were approximately 0.158, 0.135,
and 0.100. V6's first-search required-group coverage was 0.708 macro and
any-search coverage was 0.875, with zero source bytes in search responses.

These are meaningful locator and narrowing observations. They are not token
or answer-correctness substitutes.

## What did not improve enough

### Consistent model-token efficiency

No run passed the predeclared paired bar of median ratio at most 0.85 and at
least eight non-increasing pairs. V5 and V6 lowered aggregate treatment sums,
especially uncached input, but the savings were concentrated in a subset of
tasks and were offset by substantial regressions in others.

V6 illustrates why aggregate totals cannot replace pairing: aggregate
model-total fell 7.9%, yet the 11 dual-complete median was 0.977, only 6/11
pairs were non-increasing, and the fixed-seed interval crossed 1.

### Correctness preservation

V6 introduced one complete-to-partial reversal. `rhf-x03-form-submit` found
the correct target and covered its required group, but the final answer added
an uncited validation/transformation claim. This is a treatment outcome under
the frozen A/B contract, but it is not evidence that FTS failed to locate the
code.

### Efficient stopping

V6 reached 12/12 mechanical adherence while making 24 reads, 14 inspection
actions after reducer-detectable complete cidx evidence, and nine distinct
reads in one task. Only 3/12 tasks stopped at that evidence boundary.

The existing adherence metric is therefore a protocol-compliance measure, not
an efficiency measure. Search caps and duplicate bans do not bound distinct
source reads or discretionary post-evidence exploration.

## Correct interpretation of the partial result

The sole partial answer must remain fixed as graded. Every frozen required
group was covered, but correctness also prohibits material unsupported claims.
The assistant inspected relevant source through ordinary tools after the cidx
locator, then generalized beyond the cited evidence.

Consequently:

- do not call this an FTS zero-hit, ranking, or wrong-file failure;
- do not ignore it because the required group was covered;
- do not selectively regrade or rerun the pair; and
- do not infer from one outcome that optional cidx use causes overclaiming in
  the broader population.

It identifies a stage boundary: successful navigation and source acquisition
do not guarantee a disciplined final answer.

## Product posture after the series

Preserve the implemented compact locator/source-read contract and the required
hash handling. Do not promote the experimental mandatory cidx-first paragraph
as a generally token-efficient host default from this evidence.

The defensible posture is:

- cidx remains a lightweight local auxiliary MCP, not the only inspection
  tool;
- availability and optional locator use remain reasonable product hypotheses;
- token reduction, correctness preservation, and release readiness are not
  established by V4–V6; and
- no language-, cohort-, ranking-, or result-depth policy should be derived
  from these small exposed slices.

ChatGPT called this `CLOSE_SERIES_WITH_BOUNDED_OPTIONAL_VALUE`; Grok called it
`CLOSE_SERIES_NO_EFFICIENCY_CLAIM`. Both agreed on every gate and loss. The
combined posture is no efficiency claim plus unpromoted optional locator
value.

## Owner decision candidates, in order

These are handoff choices, not authorized implementation work.

1. **Host policy:** decide whether cidx should simply be available for
   discretionary use rather than mandatory first use. The current evidence
   favors optional positioning but has not evaluated that policy directly.
2. **Claim and stopping discipline:** if another host experiment is later
   approved, make “every material claim has observed cited evidence” and
   “stop after sufficient evidence” explicit host states. Keep caller budgets
   as maximums rather than making search return uncontrolled source.
3. **New evaluation design:** separate locator success, source acquisition,
   post-complete inspection, final claim support, correctness, and official
   tokens. Predeclare unsupported-claim and post-evidence-action decisions;
   do not collapse them into a single mechanical-adherence or weighted score.

A later experiment should be a newly scoped owner decision, not V7. It need
not start by adding a repository or changing questions, and it must not reuse
this exposed panel as independent promotion evidence.

## Claims explicitly rejected

V4–V6 do not prove:

- that mandatory cidx-first is consistently token-efficient;
- that treatment preserves correctness under the final protocol;
- that lower aggregate or uncached totals pass the paired gate;
- that compact locators alone guarantee a smaller full assistant journey;
- that 12/12 mechanical adherence means minimal source inspection;
- that FTS, dense, hybrid, or any language/cohort route is generally superior;
- that cidx is useless as a local auxiliary locator; or
- any `core_retrieval` or `release_candidate` promotion.

## Repository and evidence handoff

The feature/evaluation branch history entered `main` through merge commit
`cba66b7` without squashing. The V4–V6 result and review checkpoints remain
individual commits. Canonical evidence is:

- [V4 result](assistant-ab-v4-result.md) and
  [external review](assistant-ab-v4-external-review.md);
- [reducer v3](assistant-orchestration-reducer-v3.md),
  [V5 result](assistant-ab-v5-result.md), and
  [external review](assistant-ab-v5-external-review.md);
- [V6 freeze](assistant-ab-v6-freeze.md),
  [result](assistant-ab-v6-result.md), and
  [external review](assistant-ab-v6-external-review.md); and
- the chronological
  [FTS and assistant A/B journal](../../FTS-REMEDIATION-AND-ASSISTANT-AB-JOURNAL.md).

The closed series has no pending rerun, regrade, product code edit, provider
call, corpus change, or V7. The next step is an owner choice among the separate
directions above.
