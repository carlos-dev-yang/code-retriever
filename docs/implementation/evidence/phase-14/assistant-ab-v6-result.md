# Paired Assistant A/B V6 Result

- Date: 2026-08-22
- Run: `assistant-ab-v6-20260821T163500Z`
- Manifest: `assistant-ab-chi-rhf-v6`
- Scope: diagnostic assistant-use evidence only
- Series position: third and final post-V3 experiment

## Decision

V6 fixed the exact hash-field failure it was designed to fix, but it failed
both the frozen correctness-preservation rule and the paired model-token gate.

| Decision layer | Result | Evidence |
| --- | --- | --- |
| Operational validity | pass | 24/24 valid; zero timeout, control violation, source mutation, or state mutation |
| Required-hash mechanism | pass | zero omitted hashes, zero failed MCP calls, zero hash-only same-range recovery |
| Blind correctness preservation | fail | baseline 12 complete; treatment 11 complete and 1 partial |
| Paired model-token gate | fail | 11 dual-complete; median ratio 0.977; 6/11 non-increasing versus required 0.85 and 8 |
| Promotion | none | exposed diagnostic panel only |
| Series boundary | closed after external interpretation | no V7 corrective rerun |

The partial result is a complete-to-noncomplete reversal, so the V6
intervention is not correctness-safe even though every frozen required group
was covered. The treatment answer added one material claim that its cited
excerpts did not establish.

## Frozen execution integrity

The V6 manifest equals V5 for controls, question sources, corpora, and all 12
ordered task records. Its prompt equals V5 plus the one externally approved
`expected_sha256` sentence. Runner, reducer v3, answer and grade schemas,
native Codex 0.148.0, cidx/MCP binaries, tool schema, model, reasoning effort,
isolation, FTS ranking, locator wire, caller-selected `k`, and blind grading
were unchanged.

All 24 scored observations were newly executed pair-by-pair. Both unscored
schema probes passed first. No scored turn was retried or replaced. The run
finished with 24 valid observations, no timeout or control violation, and
byte-identical source trees and cidx databases before and after every
applicable turn.

One tool-free blind-grader invocation handled the eight chi entries and one
handled the sixteen React Hook Form entries. The grader saw truth, submitted
answers, and cited source excerpts, but not arm, order, token use, cidx use, or
journey.

## Blind correctness

| Arm | Complete | Partial | Incorrect | Ungradable | Required groups covered | Unsupported claims |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| baseline | 12 | 0 | 0 | 0 | 15/15 | 0 |
| cidx FTS | 11 | 1 | 0 | 0 | 15/15 | 1 |

The sole reversal is `rhf-x03-form-submit`. Its treatment answer correctly
covered the frozen submission-flow group, but asserted that
`control.handleSubmit` supplies “validated/transformed” data. The cited
excerpts showed the `Form` callback and downstream conversion/action/error
flow, but did not establish that additional validation/transformation claim.
The blind grader therefore returned `partial` with no contradicted claim.

This was not a missing target-file or failed-read event. The first cidx search
located `src/form.tsx`; the assistant then used four ordinary local source
commands and cited seven relevant spans. It did not read or cite the
`handleSubmit` implementation that would be needed for the extra claim. The
first observed correctness loss is therefore final claim/evidence discipline
after successful navigation, not proof of an FTS zero-hit or wrong-file
failure.

## Official token result

| Measure | Baseline | cidx FTS | Difference |
| --- | ---: | ---: | ---: |
| Model-total sum | 1,533,826 | 1,413,011 | -120,815 (-7.9%) |
| Input sum | 1,514,642 | 1,393,930 | -120,712 (-8.0%) |
| Uncached-input sum | 310,674 | 238,602 | -72,072 (-23.2%) |

The lower aggregate sums do not satisfy the predeclared paired gate. One pair
is excluded from the official efficiency set because treatment was partial.
Across the remaining 11 dual-complete pairs:

- model-total median treatment/baseline ratio: `0.9768402245`;
- model-total non-increasing pairs: `6/11` (required count remains 8);
- uncached-input median ratio: `0.8557382607`; and
- fixed-seed model-total bootstrap interval: `[0.7494257716, 1.4409415867]`.

Savings are concentrated in several tasks while five dual-complete tasks
increase. The paired median and count prevent those few large savings from
being mistaken for consistent marginal efficiency.

| Task | Baseline | Treatment | Model-total ratio | Uncached ratio | cidx calls |
| --- | --- | --- | ---: | ---: | ---: |
| `chi-new-router` | complete | complete | 0.918 | 1.205 | 4 |
| `rhf-x02-form-context` | complete | complete | 1.564 | 0.648 | 3 |
| `rhf-t09-control-contract` | complete | complete | 1.441 | 0.856 | 2 |
| `chi-g02-route-outcome` | complete | complete | 0.771 | 1.164 | 11 |
| `rhf-controller-component` | complete | complete | 0.749 | 0.430 | 2 |
| `chi-g09-client-ip` | complete | complete | 1.460 | 1.084 | 4 |
| `rhf-t02-controlled-field-lifecycle` | complete | complete | 0.446 | 0.388 | 3 |
| `rhf-x08-form-state-props` | complete | complete | 1.066 | 1.275 | 4 |
| `chi-g06-basic-auth` | complete | complete | 0.977 | 0.332 | 3 |
| `rhf-control-type` | complete | complete | 0.516 | 0.364 | 2 |
| `rhf-x03-form-submit` | complete | partial | 0.719* | 0.391* | 1 |
| `rhf-t10-dotted-path-types` | complete | complete | 1.244 | 1.701 | 2 |

`*` Diagnostic only; excluded from the dual-complete efficiency decision.

Language medians among dual-complete pairs are 0.948 for Go, 0.880 for
TypeScript, and 1.066 for TSX. These exposed slices are descriptive and too
small for a language policy.

## Hash-field and orchestration result

Every one of 24 `read_span` attempts supplied `expected_sha256` and completed
successfully. There were no omitted hashes, failed MCP calls,
`INVALID_READ_SPAN_REQUEST` errors, or same-range hash-only recoveries. The
three V5 hash omissions are therefore eliminated exactly as hypothesized.

| Mechanism | V4 | V5 | V6 |
| --- | ---: | ---: | ---: |
| Mechanically adherent tasks | 1/12 | 9/12 | 12/12 |
| cidx calls | 62 | 38 | 41 |
| Searches | 23 | 17 | 17 |
| Read attempts | 39 | 21 | 24 |
| Successful reads | 33 | 18 | 24 |
| Failed reads | 6 | 3 | 0 |
| Exact initial locator reads | 2/23 | 16/16 | 14/14 |
| Gross/unique read source bytes | 77,173 / 63,519 | 17,967 / 17,967 | 28,022 / 28,022 |
| Stopped after complete cidx evidence | not frozen | 4/12 | 3/12 |

The sentence fixed required-field usage but did not lower total cidx calls
below V5. Searches stayed at 17, successful reads rose by six, and source
bytes rose 55.9%. `chi-g02-route-outcome` alone made two searches and nine
distinct exact-range reads. This still satisfies the narrow V5/V6 mechanical
definition because it caps searches, forbids duplicates and widening, and
does not cap distinct selected reads.

V6 also made 14 inspection actions after reducer-detectable complete cidx
evidence versus eight in V5, and only 3/12 tasks stopped at that point. Thus
12/12 mechanical adherence means the explicit clauses were followed; it does
not mean source acquisition was minimal or consistently token-efficient.

## Locator and evidence stages

- first-search complete locator hit: `8/12`;
- any-search requirement coverage: `0.875` macro;
- search payload: 34,848 structured bytes, zero text/source bytes;
- complete cidx-read evidence: `8/12`;
- cidx-read requirement coverage: `0.708` macro;
- read precision: `0.661` macro;
- citation utilization: `0.883` macro; and
- read-source redundancy: zero.

Compared with V5, locator occurrences fell 160→125 and structured search bytes
fell 45,120→34,848, while any-search requirement coverage improved
0.750→0.875. The regression did not come from excessive search response
payload. It came from an unsupported final claim after the assistant had
already navigated to and read relevant source through ordinary tools.

## Predeclared interpretation

The required-hash mechanism passes. Correctness preservation fails, and the
paired token gate fails independently. Aggregate token savings and complete
required-group coverage cannot override either failure.

Per the frozen V6 plan and matching external stop rule, no V7 corrective rerun
is permitted. The V4–V6 sequence must close after final ChatGPT/Grok review.
Any later work must be separately scoped rather than presented as another
repair of this experiment.

## Artifact identities

| Artifact | SHA-256 |
| --- | --- |
| run manifest | `a237cb22b40997241c59bc483c3ceaa9023cfbdebe0122e352a4daf8e44d06cd` |
| frozen journey JSONL | `eb540d0432c647f984eb5237e2570e18ee0997b0c9770a28098ea0c66c8df7d2` |
| journey freeze envelope | `057342ac373e3a92450aef4fe9280083bea52d16f76e857a01ac328f8c25b5d4` |
| chi grading packet | `043f3c46f048c34b3050cf9b7f4606fe581fed2ec33a4a1f46b92bdedd75d59f` |
| RHF grading packet | `fe268a75254c5a27f2f9d99235acfd1d3673a615ff30139e0120112b8423c7d7` |
| chi blind grades | `ed97459a86efebdd6111479d3046853bd98ebe7077c38bf85b20f7117e92e325` |
| RHF blind grades | `aface2ca29e6d0d55bf70a66943911aa10cbb9b873ce99cd3b5a1aca956fe385` |
| chi grader events | `d3dc3f02e8fd4278cf8261c3cbfab7d12a654439d9429a0effc12a45ba180d6a` |
| RHF grader events | `5c4bb4736077d15921e7636e1a41a606bdbd9de70d7bcd56e463fc0347fc1aed` |
| paired results | `fab37c21add01a23ab9f19924f96832ee3835d464adc3e1be52d24cdc94824dd` |
| aggregate | `ea0d6d3d3704737e768654686651f584a7871969edd3d23e050930696448fab8` |
| generated report | `46902afed30d1175fcffe3640a418e28c1e97e34dbb6f4b3e95dd71881c4430a` |

No provider call, paid query, hybrid search, embedding, corpus/question edit,
product code change, ranking change, or public MCP contract change occurred.
