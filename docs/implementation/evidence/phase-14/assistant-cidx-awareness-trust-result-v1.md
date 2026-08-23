# Assistant cidx Awareness vs Trust-Priority V1 Result

- Date: 2026-08-23
- Phase: 14
- Status: `COMPLETE`
- Freeze commit: `caa57f906d9a3e6219d24c8fe71914bf82bd22ec`
- Run: `assistant-cidx-awareness-trust-chi-rhf-v1-run-001`
- Scope: paired, non-promotion assistant-use diagnostic
- Promotion inference: `NOT_APPLICABLE`
- Frozen plan: [Assistant cidx Awareness vs Trust-Priority Evaluation V1](../../ASSISTANT-CIDX-AWARENESS-TRUST-EVALUATION-V1.md)
- Freeze evidence: [execution freeze](assistant-cidx-awareness-trust-freeze-v1.md)
- Post-result review: [bounded ChatGPT and Grok review](assistant-cidx-awareness-trust-result-external-review-v1.md)

## 1. Result in one paragraph

The priority-and-bounded-trust suffix made cidx useful more consistently, but
did not make the full investigation journey simpler. `trust_priority` produced
30/30 complete answers, all 39 required groups, and no unsupported claim,
while `aware_choice` produced 28 complete, one partial, and one timed-out
ungradable answer with 37/39 groups and one unsupported claim. On the 29
efficiency-comparable pairs, `trust_priority` reduced unique source to a median
0.546 of `aware_choice` and was non-increasing on 27/29 pairs. It nevertheless
raised repository actions to a median 1.300 and model-total tokens to a median
1.189, with only 11/29 and 9/29 non-increasing pairs respectively. The result
therefore supports a quality and code-scope benefit, but not an overall
efficiency claim.

The dominant measured loss is after useful locator selection: evidence
acquisition and orchestration use too many searches and reads. The experiment
does not isolate unnecessary stopping from legitimate dependency expansion,
so it does not claim that either mechanism alone caused the loss.

## 2. Execution accounting

The frozen runner first passed its provider-free preflight and both arm schema
probes. It then executed the 30 counterbalanced pairs exactly once:

| State | `aware_choice` | `trust_priority` | Total |
| --- | ---: | ---: | ---: |
| Scheduled primary cells | 30 | 30 | 60 |
| Valid | 29 | 30 | 59 |
| Timed out | 1 | 0 | 1 |
| Selective retry | 0 | 0 | 0 |

`avail-go-multihop-real-ip / aware_choice` reached the frozen 600-second
timeout. It has no answer or official usage and remains `ungradable` in the
quality denominator. Its pair is excluded from paired action/source/token
estimates with explicit operational, grade, and usage reasons.

The scorer prepared 60 arm-blind entries and froze 60 body-free journeys. One
Codex grading call returned 20 Go grades and one returned 40 TypeScript/TSX
grades. The first shell invocation of each grader was rejected before model
execution because the empty grading directory was not a Git repository. No
grade file or model answer was produced. Adding the CLI's documented
`--skip-git-repo-check` flag fixed that invocation-only issue; each corpus then
received exactly one model grading call. No grade was edited, repaired, or
recalled.

The selected original-session explanation layer was omitted. Primary cells
used `codex exec --ephemeral`; their isolated source and state directories were
removed after capture, so reliable exact-session continuation was unavailable
without new infrastructure. No semantic-review substitute was created.

## 3. Blind answer quality

| Arm | Complete | Partial | Incorrect | Ungradable | Required groups | Unsupported | Contradicted |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `aware_choice` | 28 | 1 | 0 | 1 | 37/39 | 1 | 0 |
| `trust_priority` | 30 | 0 | 0 | 0 | 39/39 | 0 | 0 |

There were two conversions to complete and no regression:

1. `avail-go-multihop-real-ip`: the aware arm timed out; the trust arm found
   and supported both real-IP selection and application behavior.
2. `avail-go-semantic-route-context`: both found the core context commit, but
   the aware answer made an extra `Request.Pattern` claim from a cited range
   that ended before the assignment. The trust arm acquired and cited the
   necessary `routeHTTP` lines and graded complete.

The first conversion is evidence about execution robustness under this fixed
timeout, not a clean paired efficiency win. The second is direct evidence that
additional selected source can improve claim support.

## 4. Paired efficiency

All-cell totals below are descriptive workload accounting. The aware token
total includes a zero-usage timeout and is not an efficiency denominator.
Paired ratios over the 29 valid comparable pairs are the primary estimates.

| Measure | Aware total | Trust total | Trust/aware paired median | Trust non-increasing |
| --- | ---: | ---: | ---: | ---: |
| Repository actions | 180 | 241 | 1.300 | 11/29 |
| Combined unique source bytes | 356,646 | 164,195 | 0.546 | 27/29 |
| Model-total tokens | 3,586,157 | 3,924,022 | 1.189 | 9/29 |

The 28 dual-complete pairs have a model-total median ratio of 1.205. This
removes both the timeout conversion and the aware partial conversion and still
shows higher trust-arm token use.

| Language | Comparable pairs | Complete A / B | Token median | Action median | Source median |
| --- | ---: | ---: | ---: | ---: | ---: |
| Go | 9 | 8 / 10 over all 10 tasks | 1.002 | 1.100 | 0.896 |
| TypeScript | 10 | 10 / 10 | 1.266 | 1.100 | 0.452 |
| TSX | 10 | 10 / 10 | 1.205 | 1.667 | 0.348 |

The Go quality counts include the partial and timeout; its efficiency column
uses nine comparable pairs. No language slice is a standalone promotion gate.

## 5. Question-shape result

| Shape | Pairs | Quality change | Token median | Action median | Source median |
| --- | ---: | --- | ---: | ---: | ---: |
| Contract/lifecycle/state/error | 6 | 6 complete in both | 0.961 | 0.925 | 0.337 |
| Verified negative/hard negative | 3 | 3 complete in both | 0.672 | 1.000 | 0.597 |
| Exact identifier/path | 6 | 6 complete in both | 1.150 | 1.000 | 0.704 |
| Semantic single-hop | 6 | one partial -> complete | 1.434 | 1.548 | 0.587 |
| Dependency/type multi-hop | 6 | one timeout -> complete | 1.294 | 1.667 | 0.571 |
| Ambiguous/disambiguation | 3 | 3 complete in both | 2.036 | 2.200 | 0.580 |

Trust guidance was efficient on the small contract/lifecycle and negative
slices, neutral on exact actions, and costly on semantic, multi-hop, and
ambiguous tasks. These are diagnostic slices with small denominators, not a
rule for runtime query routing.

## 6. Deterministic journey

| Observation | `aware_choice` | `trust_priority` |
| --- | ---: | ---: |
| Tasks using cidx | 19/30 | 30/30 |
| cidx searches | 54 | 95 |
| successful `read_span` | 48 | 125 |
| total cidx calls | 102 | 220 |
| ordinary `rg` discovery actions | 35 | 12 |
| ordinary source reacquired after cidx | 26,070 bytes | 1,793 bytes |
| first-search complete locator hit | 11/30 | 17/30 |
| mechanically complete cidx evidence | 13/30 | 27/30 |
| selected-locator reads | 38 | 120 |
| read precision macro | 0.544 | 0.483 |
| duplicate read ranges | 2 | 12 |
| overlapping successful reads | 2 | 13 |
| gross / unique cidx read source | 82,255 / 76,402 | 197,967 / 146,218 |

The search contract recommendation was behaviorally effective:

- all 95 trust-arm searches omitted `k` and used default `k=5`;
- aware searches requested `k=10` 43 times, `k=8` six times, `k=20` three
  times, and `k=5` and `k=3` once each; and
- trust search results exposed 473 locator occurrences versus 534 in the aware
  arm despite more searches, but total structured cidx result bytes still rose
  from 242,969 to 360,716.

The trust suffix also improved the exact locator handoff. It reduced ordinary
source reacquisition by 93.1% and raised selected-locator reads from 38/48 to
120/125. There were no incomplete cidx search/read calls and no hash-copy
failure. A ready-to-use locator reference would improve ergonomics, but this
run does not identify locator copying as the current measured bottleneck.

The remaining cost moved inside cidx use:

- searches rose by 41 and reads by 77;
- gross read source rose by 115,712 bytes while precision fell;
- duplicate and overlapping reads increased; and
- lower ordinary-source volume did not offset the additional model/tool
  continuation cycles and cached-context replay.

The result therefore does not show a basic failure to find relevant code.
Nineteen trust-arm questions had their first useful locator at rank one, and
the trust arm obtained mechanically complete cidx evidence on 27/30 tasks.
It also does not prove locator sufficiency for every task: six trust tasks had
no accepted useful locator rank, including negative cases, and required
dependency/context expansion remains legitimate.

## 7. Historical predecessor comparison

The predecessor and current run are separate model samples, so this is a
descriptive reference, not a paired causal estimate.

| Directed policy | Repository actions | cidx calls | Combined unique source | Paired token median |
| --- | ---: | ---: | ---: | ---: |
| Forced-cidx predecessor | 254 | 229 | 196,476 | 1.404 |
| Current trust-priority | 241 | 220 | 164,195 | 1.189 |

The concise trust-priority policy is better than the forced policy on all four
historical descriptors, but it still fails the contemporaneous simplification
comparison against `aware_choice`.

## 8. Interpretation and next decision

This run supports these conclusions:

- concise awareness is sufficient for voluntary cidx use on 19/30 tasks;
- priority/trust makes adoption universal on this panel and improves evidence
  completeness and final quality;
- cidx can materially reduce the code surface inspected by an assistant;
- trusting exact `read_span` reduces fallback source reacquisition; and
- the current search-to-single-read flow still creates too many evidence
  acquisitions and model continuations.

It does not support these claims:

- trust-priority is generally more efficient;
- stopping behavior itself improved or is the sole cause of the token loss;
- FTS retrieval is now the dominant bottleneck or is sufficient for every
  task;
- ordinary tools should be demoted;
- batching is already proven to solve the loss; or
- the result generalizes beyond the frozen repositories, questions, model,
  prompt, FTS state, and configuration.

The smallest justified next design candidate is one bounded multi-locator
evidence request: the assistant selects a small set of existing search
locators, and one request returns separate line-addressable evidence units for
each. This targets 125 individual reads without changing FTS retrieval,
ranking, query planning, or the number of distinct facts a multi-hop question
may legitimately require.

That candidate changes an external MCP input/output contract and is not
authorized by this result alone. Before implementation, the owner must choose
the exact bound and whether it extends `read_span` or remains a separate
versioned interface experiment. If approved, freeze the same questions,
truth, corpora, model, prompt policy, FTS/index state, default `k`, isolation,
and blind grader. Compare the current single-locator trust arm with only the
new evidence-handoff arm.

Use three separate gates:

1. no answer-quality, required-group, or unsupported-claim regression;
2. fewer evidence requests and duplicate/overlapping reads without lower
   evidence coverage or uncontrolled source-byte growth; and
3. lower paired repository actions and official model-total/uncached-input
   usage, retaining every failure in its proper denominator.

Do not add a repository, retrieval retuning, dense/hybrid work, a fifth tool,
or another semantic-review system to that experiment.

## 9. Artifact seal

Ignored local artifacts remain the reproducible authority for detailed cells:

| Artifact | SHA-256 |
| --- | --- |
| Run manifest | `f5026cf9217adc2ea630415e724c9d20d09b5c45d69825186fc2ce0d703fce5a` |
| Aggregate | `191ef9bcfbbf710ac863e8e57601c75a62aace1032c446ad9e57a23dfbd46af5` |
| Paired results | `ccc2478b16f3b174f5aceee50791761cb61d8a3a928674574604f7a5e9c704ff` |
| Generated report | `c9925024448f5723da6da6ba7a6eb55faae8af5837288e7521faaccdf208dddc` |
| Frozen journey | `77f2c65cab74ddecdb38025d5e926521936e3d2d78faa3a3d7a7f537ed01e549` |
| Blind key | `02483522dc599e27063d54d9f6383c43375c840538cdcca72f5675ad62d0e268` |
| Go grades | `c17f511ab7ff4f6f1af1b2e56e59e70dab4c1f7dda3324395bd4bcd326b8b52e` |
| TypeScript/TSX grades | `fd8eb009e4f0b2c397484bc6f31eb44c20252c96e1e583d52af2c9513d6acbfe` |

Checks actually completed include the frozen focused Go/race/vet and Python
checks, provider-free preflight, both arm schema probes, 60 primary executions,
60-entry journey freeze, two arm-blind grading calls, canonical fail-closed
grade validation, and one aggregate. No broad project suite, provider call,
dense/hybrid query, product promotion, selected original-session follow-up, or
new interface implementation was run.
