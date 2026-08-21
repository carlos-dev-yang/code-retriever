# Assistant Journey Reducer Version 2

- Date: 2026-08-21
- State: implementation checkpoint; Phase 13 remains `in_progress`
- Scope: assistant event accounting and locator/evidence stage measurements
- Product search behavior changed: no
- Provider, network, or paid operation: none
- Corpus mutation: none

## Why the reducer changed

Version 1 classified repository commands directly from the outer command
string. Codex records many commands through a quoted shell wrapper such as
`/bin/zsh -lc "rg ..."`; the quote before the inner command was not an accepted
boundary. Baseline shell output was therefore undercounted.

Version 2 unwraps bounded `sh`/`bash`/`zsh`/`dash`/`ksh` `-c` invocations before
classification. It also records all shell event output independently from the
repository-inspection subset. MCP accounting now keeps structured payload,
text payload, complete result envelope, and returned source bytes as separate
observations. None of these values is named model-visible without host proof.

## Stage measurements added

The reducer maps the already-frozen source spans to line ranges and reports:

- first-search and any-search requirement coverage;
- first useful locator rank;
- duplicate locator exposure;
- locator-to-read and locator-to-citation utilization;
- navigation false leads;
- `read_span` requirement coverage and precision;
- evidence-to-citation utilization and false leads;
- gross, unique, and redundant evidence bytes; and
- the ordered inspection action where complete evidence first became
  available, when mechanically derivable.

Required groups keep their existing AND semantics. Alternatives inside a
group remain OR, while every mandatory span inside one alternative must be
covered. The reducer does not change questions, grades, retrieval, or the
frozen V3 records.

## Read-only V3 replay

The complete V3 run was copied to disposable state without its grading-derived
files. Version 2 regenerated the 24 journey rows; the existing blind grades
were then copied into that disposable state only to exercise aggregation.
The original V3 directory was neither edited nor overwritten.

Reproduced end-to-end outcomes and official tokens were byte-for-value equal
to the published V3 result. The corrected event accounting was:

| Arm | Shell event bytes | Repository shell bytes | cidx result-envelope bytes | Repository event-output proxy |
| --- | ---: | ---: | ---: | ---: |
| baseline | 697,490 | 697,490 | 0 | 697,490 |
| cidx FTS | 36,147 | 36,147 | 938,001 | 974,148 |

The event-output proxy is diagnostic serialization evidence, not confirmed
model input.

The new stage results over the 12 treatment tasks were:

| Measurement | Result |
| --- | ---: |
| First-search complete locator hit | 9/12 |
| First-search requirement coverage, macro | 0.792 |
| Any-search requirement coverage, macro | 0.917 |
| Locator occurrences / task-local unique sum | 176 / 148 |
| Locator selection utilization, macro | 0.264 |
| Locator citation utilization, macro | 0.235 |
| Search structured / text / source bytes | 390,071 / 402,601 / 170,423 |
| Search calls carrying both representations | 19/19 |
| Complete cidx evidence hit | 12/12 |
| Evidence requirement coverage, macro | 1.000 |
| `read_span` precision, macro | 0.551 |
| Evidence citation utilization, macro | 1.000 |
| Gross / unique read source bytes | 46,009 / 45,547 |
| Read calls carrying both representations | 29/29 |

This confirms the stage split is informative: search supplied all required
locations eventually for 11/12 tasks, selected reads supplied complete cidx
evidence for 12/12, but only about one quarter of unique locators were selected
and about 23.5% were finally cited. The observation supports response
compaction; it does not prove a smaller `k` is safe.

## Validation

Checks run:

- Python syntax compilation for both assistant scripts;
- direct wrapper-normalization assertion for a quoted `zsh -lc "rg ..."`
  command;
- CLI help execution for both scripts;
- complete 24-record disposable V3 `prepare` and `aggregate` replay;
- preservation of 11 complete + 1 partial baseline, 12 complete treatment,
  and the published official token totals;
- journey schema-version, record-count, aggregate, and report inspection; and
- repository whitespace validation.

Replayed journey SHA-256:
`3d989cfcceca97a81239b5c1c546227b72372224b596e90a2466649515f1a28b`.

Implementation SHA-256 values at this checkpoint:

- runner: `193251b862b16c940f65309100263af246507f8ddfaaf4d5e3ef30e353d560cd`;
- reducer: `6a754955ce620ae4dbe02a617a1b207c86a80b261b3015125443b9055fe946b2`.

Checks not run:

- no product MCP response change;
- no Codex representation-conformance probe;
- no new assistant task turn;
- no Voyage request; and
- no promotion or release evaluation.

## Handoff

Next, probe structured-only and text-only tool results through Codex CLI, then
implement the locator-only result projection using the compatible single
representation. A scored assistant batch remains blocked until that Phase 13
wire boundary is validated and committed.
