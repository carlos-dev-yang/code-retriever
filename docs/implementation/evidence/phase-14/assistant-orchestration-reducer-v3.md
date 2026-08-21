# Assistant Orchestration Reducer v3

- Date: 2026-08-22
- Phase: 14
- Parent decision: [`assistant-ab-v4-external-review.md`](assistant-ab-v4-external-review.md)
- Scorer SHA-256: `f215fe893713491dd6034e3f19d1227d61d356ca80baf58b4998112cdab586a3`
- Product/search change: none
- Provider/corpus action: none

## Purpose

V5 changes one model-facing orchestration paragraph. Its result therefore
needs to separate end-to-end answer correctness and tokens from mechanical
evidence that the prompt changed the search-to-read journey. Reducer v3 adds
that observation layer before V5 is frozen. It does not change prompts,
assistant execution, blind grading, retrieval, or prior artifacts.

## New arm-blind observations

For every treatment journey the reducer now records:

- whether the first search supplied `max_inline_bytes=0`;
- whether total searches stayed within the initial plus one-refinement limit;
- repeated query/mode pairs and byte-identical search argument objects;
- repeated `read_span` path/range calls;
- failed `INVALID_RANGE` reads;
- all read attempts that exactly equal an earlier returned locator range;
- the first read for each path/content hash and whether that initial read uses
  an exact earlier locator range;
- inspection actions after frozen required-group evidence first became
  complete; and
- a narrowly named mechanical-adherence flag.

The flag requires the observable clauses only: first cidx search, first
`max_inline_bytes=0`, at most two searches, no repeated query, no repeated read
range, no invalid range, and an exact locator range for each first path/hash
read. It deliberately excludes whether a refinement was semantically
justified and whether every material answer claim was already covered. Those
claims require blind-grade or report review and are not reconstructed from
event counts.

## Read-only V4 calibration

The new reducer was applied in memory to the preserved V4 event streams and
source truth. No V4 run, journey freeze, grade, or aggregate file was written.
It reproduced the already reported residuals and exposed the exact-range
baseline for V5:

| Observation | V4 value |
| --- | ---: |
| treatment tasks | 12 |
| first search used `max_inline_bytes=0` | 11/12 |
| search count at most two | 9/12 |
| repeated query/mode occurrences | 2 |
| exact duplicate read ranges | 4 |
| invalid ranges | 6 |
| first path/hash reads | 23 |
| exact locator-range first reads | 2 |
| non-exact locator-range first reads | 21 |
| mechanically adherent tasks | 1/12 |

The repeated-query measure intentionally ignores the compatibility-only
`max_inline_bytes` value: changing that value cannot change the locator result
identity. Byte-identical search argument duplicates remain available as a
separate field.

## Validation performed

- Python source compiled in memory without writing cache artifacts.
- All 12 preserved V4 treatment event streams were reduced successfully.
- Existing V4 counts for duplicate successful reads and invalid ranges were
  reproduced as 4 and 6.
- Repository diff whitespace validation passed.

No full project test suite, new assistant turn, blind regrade, or provider call
was performed. V4 artifacts remain immutable; the calibration above is a
retrospective instrumentation check only.

## Handoff

Freeze V5 against this scorer identity. The complete V5 run must create a new
journey freeze and report correctness, official tokens, and reducer v3
mechanism metrics separately. Prompt-policy noncompliance is retained as an
experimental outcome, not converted into an operationally invalid turn.
