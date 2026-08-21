# Paired Codex CLI Assistant A/B Version 5 Result

- Date: 2026-08-22
- Run: `assistant-ab-v5-20260821T155000Z`
- Manifest: `assistant-ab-chi-rhf-v5`
- Scope: diagnostic assistant use only; not promotion evidence
- Provider/embedding action: none
- New corpus or question: none

## Decision summary

V5 preserved correctness and materially improved the intended search/read
mechanism, but it did not pass the predeclared paired model-token gate.

| Dimension | Result | Decision |
| --- | ---: | --- |
| valid scored turns | 24/24 | pass |
| blind complete, baseline | 12/12 | preserved |
| blind complete, cidx | 12/12 | preserved |
| complete-to-noncomplete reversals | 0 | pass |
| covered blind required groups | 30/30 | pass |
| unsupported / contradicted claims | 0 / 0 | pass |
| model-total sum, treatment vs baseline | -6.9% | favorable descriptive result |
| uncached-input sum | -34.4% | favorable descriptive result |
| paired model-total median ratio | 0.954 | fails required <=0.85 |
| model-total non-increasing pairs | 6/12 | fails required 8/12 |
| fixed-seed median bootstrap 95% | [0.728, 1.329] | diagnostic; crosses 1 |

The V5 prompt is effective as a mechanism intervention: it sharply reduced
tool activity and source delivery while preserving answers. It is not yet
evidence that cidx reliably reduces total model tokens across this panel.

## Execution and grading integrity

The frozen runner completed both schema probes and every scheduled back-to-back
pair exactly once. All 26 observations, including the two unscored probes,
were valid with no timeout, source/state mutation, forbidden tool, or control
violation. No scored task was retried or replaced.

Reducer v3 froze all 24 journeys before arm restoration. One blind grader
model call covered the eight Go answers and one covered the sixteen React Hook
Form answers. Neither grader called a tool.

The first attempted Go grader command stopped before a model turn because its
fresh empty directory was not a Git trust root. It produced a zero-byte event
file and no grade. That record is preserved. The command was corrected only by
adding the Codex CLI `--skip-git-repo-check` runner option; packet, schema,
model, reasoning, and output target were unchanged. The accepted path-fix call
is the sole Go model grading observation.

## Official tokens

| Arm | Model total | Input | Cached input | Uncached input | Output | Reasoning subset |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| baseline | 1,357,198 | 1,339,952 | 971,520 | 368,432 | 17,246 | 6,097 |
| cidx FTS | 1,263,161 | 1,245,039 | 1,003,520 | 241,519 | 18,122 | 8,355 |

Treatment reduced total input by 7.1% and uncached input by 34.4%, but cached
input increased 3.3%. The paired model-total median remains the decision
authority; aggregate sums do not override its failed 0.85/8-of-12 gate.
Uncached input was non-increasing in 10/12 pairs with median ratio 0.628.

| Task | Language | Model-total ratio | Uncached ratio | cidx calls |
| --- | --- | ---: | ---: | ---: |
| chi-new-router | Go | 1.411 | 0.451 | 4 |
| rhf-x02-form-context | TSX | 0.770 | 0.637 | 4 |
| rhf-t09-control-contract | TypeScript | 0.488 | 0.619 | 2 |
| chi-g02-route-outcome | Go | 1.133 | 1.027 | 1 |
| rhf-controller-component | TSX | 0.758 | 0.417 | 4 |
| chi-g09-client-ip | Go | 1.686 | 2.325 | 3 |
| rhf-t02-controlled-field-lifecycle | TypeScript | 0.698 | 0.434 | 1 |
| rhf-x08-form-state-props | TSX | 1.247 | 0.771 | 4 |
| chi-g06-basic-auth | Go | 1.769 | 0.591 | 5 |
| rhf-control-type | TypeScript | 0.571 | 0.788 | 3 |
| rhf-x03-form-submit | TSX | 1.147 | 0.510 | 3 |
| rhf-t10-dotted-path-types | TypeScript | 0.774 | 0.638 | 4 |

## Orchestration mechanism

The V4-to-V5 comparison is cross-run and descriptive. V5's contemporaneous
paired result above remains authoritative for efficiency.

| Observation | V4 | V5 | Change |
| --- | ---: | ---: | ---: |
| search calls | 23 | 17 | -26.1% |
| `read_span` attempts | 39 | 21 | -46.2% |
| total cidx search/read calls | 62 | 38 | -38.7% |
| invalid range failures | 6 | 0 | eliminated |
| repeated search query/mode | 2 | 0 | eliminated |
| first search `max_inline_bytes=0` | 11/12 | 12/12 | complete |
| tasks with at most two searches | 9/12 | 12/12 | complete |
| exact locator-range first reads | 2/23 | 16/16 | complete |
| mechanically adherent tasks | 1/12 | 9/12 | improved |

V5 still has three failed `read_span` attempts. They are not out-of-range
reads. In `chi-g06-basic-auth`, `rhf-control-type`, and
`rhf-x03-form-submit`, the assistant passed exact locator path/start/end but
omitted the required `expected_sha256`. Each call failed with
`INVALID_READ_SPAN_REQUEST`, then the same range was immediately retried with
the locator hash and succeeded. This is why reducer v3 reports three duplicate
range attempts while source redundancy is zero.

The V5 prompt explicitly named path lines but did not name the fourth required
locator field. The three recoveries are the first remaining mechanically
isolated loss.

## Locator and evidence stages

| Observation | V4 | V5 |
| --- | ---: | ---: |
| first-search complete locator hit | 7/12 | 7/12 |
| any-search complete locator hit | 10/12 | 8/12 |
| complete cidx evidence hit | 12/12 | 8/12 |
| locator occurrences | 260 | 160 |
| task-local unique locators | 192 | 132 |
| search structured bytes | 72,172 | 45,120 |
| search source/text bytes | 0 / 0 | 0 / 0 |
| successful reads | 33 | 18 |
| read precision, macro | 0.586 | 0.717 |
| evidence citation utilization, macro | 0.904 | 0.967 |
| gross / unique read source bytes | 77,173 / 63,519 | 17,967 / 17,967 |
| source redundancy | 14.8% | 0% |

The lower any-search and cidx-evidence coverage are not blind-answer failures.
The assistant used ordinary source tools where the compact locator journey did
not cover every frozen group, and all final answers remained complete. This is
consistent with cidx's role as an auxiliary locator rather than the only code
reader. It does not justify reducing result depth or changing retrieval.

## Cohort and language diagnostics

- TypeScript: median model-total ratio 0.635, 4/4 non-increasing.
- TSX: median 0.958, 2/4 non-increasing.
- Go: median 1.549, 0/4 non-increasing.
- Lexical-anchor: median 0.758; semantic-only: 0.774;
  multi-requirement: 0.774; contract-disambiguation: 0.909.
- Mixed-signal: median 1.467; the single known-hard-negative task: 1.686.

These overlapping cells are small and exposed. They identify heterogeneity but
do not authorize language routing, question rewriting, or a retrieval change.
In particular, V5 corrected the earlier semantic/multi-step mechanism cost on
this run while Go remained expensive; a single run cannot distinguish stable
language behavior from task/model variance.

## First loss and bounded V6 candidate

The locator/evidence contract and the V5 orchestration paragraph remain
accepted. The exact next candidate is one addition to that same paragraph:

> When calling `read_span`, pass `path`, `start_line`, `end_line`, and
> `expected_sha256` exactly as returned by the selected locator.

This targets all three remaining failed/retry calls without changing the MCP
schema, retrieval, ranking, result depth, range semantics, corpus, questions,
or grader. Two of the three affected tasks are model-total regressions, so the
hypothesis is capable of changing the current 6/12 count, but no
counterfactual token saving is claimed.

Do not freeze V6 until the fixed V5 evidence and this causal attribution are
reviewed by both side-panel AIs. Do not lower `k`, add hybrid/dense, add a new
repository, change tool code, or rewrite questions at this boundary.

## Checks performed and absent

Performed:

- exact frozen-input preflight and two schema probes;
- 24/24 isolated scored turns with post-turn source/state checks;
- arm-blind reducer-v3 journey freeze;
- one actual tool-free grader model call per corpus packet;
- complete grade/key/required-group validation and aggregation;
- direct event audit of every failed cidx read; and
- artifact checksums and repository diff whitespace validation.

Not performed or claimed:

- no provider key, Voyage request, paid embedding, or hybrid search;
- no new or changed corpus, question, truth, or cohort version;
- no selective rerun, repeated stochastic estimate, or V6 turn;
- no broad project test suite, because product code did not change; and
- no `core_retrieval` or `release_candidate` promotion.

## Artifact identities

All run artifacts remain in ignored local evaluation state under the run ID.

| Artifact | SHA-256 |
| --- | --- |
| V5 plan | `91bdf26059b8cb875047ce56a663a0436c34a75e3563c09fc2250553ee4f0f4e` |
| V5 manifest | `f1f60d634fb4aeb5c43aa4b67549aed41c94cd50f7af42bf1cca6fbf568ab9f5` |
| run manifest | `9d0fe15bd74180453e3f041bd19d618929198375e705de060d6c2c14c8c79366` |
| frozen journey JSONL | `30ed320ab06184038d36c459086a5e388c0cbe7bb8917d1e07cac7063a73b48e` |
| journey freeze envelope | `0e7d9e477a9f0371954af27c755ff34b1d0bdea5d3c8697ffc260a475ae708e1` |
| Go blind grades | `10795cdd347daf6f710d586465e853c51bae4e07c5091e6baa1c2f4e1cebbb15` |
| React Hook Form blind grades | `aba6a9b233379eac8e1a315341be0b33c76e378c4d018002af96fe6d899e46ad` |
| paired results | `a0ee3f3e4fab7d63956a0adc838ed5cbfe420f815d8fb21a506c0f53a4fb482c` |
| aggregate | `c130d0294d7465ceb168ba59f64ff8bded1522af4babf86a069aba1b8e7cf506` |
| generated report | `fa281cab03109b10ac0e5f5b1911152afcbdad757a62ffd7a1ed56235d2776d2` |
| zero-byte pre-model Go grader attempt | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| accepted Go grader events | `92b8a42f01952c8f4187889c612910e353e931a0858f8b8b2788a34b3154b6ec` |
| accepted RHF grader events | `9213098d9b78023b638d5c9a4d861bde50122a611575dbf93ac9e01880c75f24` |
| runner | `dfe931e02e467556d479d6268398f6f353912722a3f41dc52c6f24e7596b56b3` |
| reducer/scorer v3 | `f215fe893713491dd6034e3f19d1227d61d356ca80baf58b4998112cdab586a3` |

## Result

`CORRECTNESS_PRESERVED`, `ORCHESTRATION_MECHANISM_IMPROVED`, and
`TOKEN_REDUCTION_GATE_NOT_MET` are all true. The run is complete diagnostic
evidence, not promotion evidence. The next action is external review of the
single `expected_sha256` prompt addition before any third experiment.
