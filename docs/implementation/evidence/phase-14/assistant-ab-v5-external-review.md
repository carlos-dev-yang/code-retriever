# Assistant A/B V5 Post-result External Review

- Date: 2026-08-22
- Phase: 14
- Scope: advisory decision for the final permitted paired experiment
- Product/search change in this checkpoint: none
- Provider/corpus action: none

## Fixed evidence presented

The same English V5 packet was sent to the existing ChatGPT and Grok
side-panel conversations. It included the complete paired correctness and
token result, V4-to-V5 mechanism changes, language/cohort caution, and the
three exact failed `read_span` calls. The reviewers were constrained to one of
`PROCEED_V6_HASH_FIELD`, `STOP_AFTER_V5`, or `CHANGE_V6_INTERVENTION` and were
explicitly prohibited from proposing a new corpus, question edit, paid query,
hybrid/dense, result-depth change, ranking/retrieval change, server retry, or
language-specific route.

## Matching disposition

ChatGPT: `PROCEED_V6_HASH_FIELD`

Grok: `PROCEED_V6_HASH_FIELD`

Both reviews found a direct prompt/schema mismatch:

- the frozen `read_span` schema requires `expected_sha256`;
- three V5 calls copied locator path and line range but omitted only that
  field;
- all failed before source delivery with `INVALID_READ_SPAN_REQUEST`;
- all immediately succeeded after the same range was called with the locator
  hash; and
- the V5 prompt named the path/range behavior but omitted the hash field.

This makes the addition more specific than a retrieval or ranking hypothesis.
Neither review claimed that the three recoveries explain the whole remaining
token gap or guarantee the efficiency gate. A complete new pair remains
necessary; editing three turns or comparing new treatment turns with the V5
baseline would break the paired design.

## Accepted sole V6 change

Insert this exact sentence immediately after V5's existing exact-range
instruction:

> When calling `read_span`, pass `path`, `start_line`, `end_line`, and
> `expected_sha256` exactly as returned by the selected locator.

The remainder of the V5 prompt stays byte-identical. This is guidance for an
existing required field, not a tool schema or source-return change.

## Frozen guardrails

V6 must retain V5's:

- approved repositories, pinned revisions, local states, 12 tasks, truths,
  cohorts, task order, and arm order;
- native Codex version, model, reasoning effort, isolation, timeout, answer
  schema, and disabled features;
- FTS planning/ranking, locator-only structured response, current
  caller-selected `k` semantics with default 10, and source-only `read_span`;
- runner, reducer v3 definitions, official token accounting, and blind grader;
- complete contemporaneous 24-turn execution with no selective replacement;
  and
- the existing correctness and paired model-token criteria.

Prompt noncompliance remains an outcome rather than an automatic ungradable
turn. No other prompt sentence changes.

## V6 decision evidence

Correctness is preserved only with no complete-to-noncomplete reversal, all
frozen required groups covered, and no new unsupported or contradicted
material claim.

The hash mechanism succeeds only with:

- zero selected-locator `read_span` calls omitting `expected_sha256`;
- zero `INVALID_READ_SPAN_REQUEST` failures caused by that omission; and
- zero immediate same-range recovery calls whose sole change is adding the
  hash.

Report searches, all read attempts, failures by reason, exact first locator
ranges, repeated operations, source volume/redundancy, precision, and citation
utilization without turning them into additional intervention changes.

The official token decision remains paired model-total median ratio at most
0.85 and at least 8/12 non-increasing pairs. Aggregate, uncached, bootstrap,
and the three formerly affected tasks remain separately reported.

## Mandatory stop rule

V6 ends this V4–V6 three-experiment sequence under every outcome. If the hash
mechanism succeeds but the token gate fails, record that the known locator and
prompt-level mechanical waste was corrected while cidx-first treatment still
did not establish material model-token reduction on this frozen panel. Do not
create V7 or another corrective rerun from this sequence. Any later product or
retrieval investigation requires a separately scoped owner decision.

## Decision

The external reviews and local first-loss audit agree. Freeze and execute V6
with the one sentence above as its only change, then stop the experiment series
after its complete result and final external interpretation.
