# Paired Codex CLI Assistant A/B — Version 3 Result

- Status: `complete_diagnostic`
- Date: 2026-08-20
- Run ID: `assistant-ab-v3-20260820T143000Z`
- Scope: existing chi and React Hook Form calibration questions only
- Promotion authority: none

## What was tested

Twelve frozen questions were each run once in two fresh Codex CLI sessions:

- baseline: ordinary read-only local tools, no MCP;
- treatment: the same prompt and tools plus the exact cidx MCP, with
  `server=cidx, tool=search` required as the first repository-discovery action;
- task model: `gpt-5.6-sol`, high reasoning;
- treatment retrieval: local FTS only, no Voyage credential or paid query;
- corpora: pinned chi and React Hook Form snapshots already selected by the
  owner; no repository was added;
- correctness: separately blind graded after arm-blind journey freezing.

Version 1 is preserved as a zero-adoption optional-tool pilot. Version 2 is
preserved as a 3/12-compliance prompt/interface diagnostic. Version 3 names the
exact MCP tool identity and achieved 12/12 compliance; neither earlier run was
overwritten or selectively retried.

## Frozen controls

| Item | Value |
| --- | --- |
| Codex CLI | official npm `0.148.0`, SHA-256 `b0308517b20543012fa2171aa3d46ce455a7456c4eb2a552ab9468ba4eeb1e50` |
| cidx binary | `e06977885044e155d5e84f7c0f7bee3d2b7eabe9e34629aabb9cb815b26a0f89` |
| MCP launcher | `31cdaeace8125b1fa48103fbf4c6efe2c0d96ad3dfa186f54d5688b6b98f3adf` |
| MCP schema | `ef5da708b935e22790a0aa5e12994d1db11b58c8f7e2d5dd45fd33560ce708a0` |
| Manifest | `3b031fe6e548faae93d8e8d43db8c0814cf114787dc3c8cb6853999edf86e069` |
| Plan at execution | `e408ba6b79dbc9c0ccb4938d48de8244c59bc9e23f139c2ee2d920257ff78113` |
| Runner | `9560c24282aa28fa2aed7ace900100375883df4380cca923e8c3e04deabf8082` |
| Answer schema | `696bca2293a4aeaeca46b2eb0809e75c2a2d13e55b06b7cef4b89f8cc63e3d8b` |
| chi commit/tree | `8b258c7bb28f97a5f2a856ff7ef962578fec9215` / `7ccb2269b57183ac3a741f269c0da31fd03ad035` |
| RHF commit/tree | `371432c39271aab739358d19c406793771565ab3` / `688906c5842a0d71051154343e993adb525e688f` |

The two arms differed only by cidx MCP exposure. The schema probes differed by
two model tokens, so static MCP schema overhead is not the observed efficiency
driver.

## Correctness

| Arm | Complete | Partial | Incorrect | Ungradable |
| --- | ---: | ---: | ---: | ---: |
| Baseline | 11 | 1 | 0 | 0 |
| cidx FTS | 12 | 0 | 0 | 0 |

All treatment answers cited source paths returned by cidx, and there was no
complete-to-noncomplete reversal. `chi-g06-basic-auth` changed from baseline
partial to treatment complete: the baseline made an unsupported no-response-body
claim, while treatment retrieved and cited the full helper. The predeclared
two-conversion “accuracy helpful” label was not met, so this is one bounded
positive observation rather than a general accuracy claim.

## Tokens and journey

| Measure | Baseline | cidx FTS | Direction |
| --- | ---: | ---: | --- |
| Total model tokens | 1,223,579 | 1,678,341 | cidx `+37.2%` |
| Total uncached input | 241,613 | 363,843 | cidx `+50.6%` |
| Complete answers | 11 | 12 | cidx `+1` |
| Compliant tasks | 12 | 12 | equal |

Among 11 dual-complete pairs, the treatment/baseline model-total median was
`1.378`; only `3/11` pairs were non-increasing and the fixed-seed paired-task
bootstrap interval was `[0.832, 2.140]`. The frozen efficiency requirement was a
median at most `0.85` and at least `8/11` non-increasing, so it failed clearly.

Model-visible repository output bytes increased in every treatment task.
Treatment made 2–8 cidx calls; searches commonly requested `k=10` and
`max_inline_bytes=12,000` or `20,000`. Serialized search results were commonly
about 34–75 KB before follow-up `read_span` calls. The transcript contains both
text and structured forms of MCP results, but this evidence does not by itself
prove both forms entered the model context; official token usage and the
machine-frozen visible-output measure remain the authority.

## Critical-cohort diagnostic

| Critical cohort | Tasks | Baseline complete | cidx complete | Median model ratio | Non-increasing | Median inspection delta | Median visible-output ratio |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| lexical anchor | 3 | 3 | 3 | 0.832 | 2/3 | +2 | 5.631 |
| semantic only | 5 | 5 | 5 | 1.435 | 0/5 | +3 | 5.341 |
| mixed signal | 4 | 3 | 4 | 1.378 | 1/3 dual-complete | +2 | 19.645 |
| contract disambiguation | 4 | 4 | 4 | 1.816 | 1/4 | +3 | 16.922 |
| multi requirement | 3 | 3 | 3 | 1.435 | 0/3 | +3 | 6.604 |
| known hard negative | 1 | 1 | 1 | 2.255 | 0/1 | +4 | 19.645 |

The bounded panel suggests lexical-anchor questions are the best current fit for
FTS-first. Semantic, contract, multi-requirement, and hard-negative tasks preserve
correctness but incur much larger journeys. Cohort cells are small and overlapping;
they diagnose where to correct, not population performance.

## External review disposition

Before V3, side-panel ChatGPT and Grok both returned `PROCEED_WITH_V3` after V2
exposed shell-versus-MCP ambiguity. After V3, both returned
`CORRECT_BEFORE_NEXT_AB` and agreed that an unchanged repeat is not justified.
The review conversations are preserved in the owner's signed-in sessions:
[ChatGPT review](https://chatgpt.com/c/6a86c914-2d18-83e8-9b86-d29de2922b2a)
and [Grok review](https://grok.com/c/d1f4ad8f-65de-49fb-a3c2-fbd026d4bd84?rid=91cc4af2-a9cc-414f-95cd-f0c76e44706f).
Their ranked correction order was:

1. reduce search response volume and investigate/remediate redundant text versus
   structured representations; keep search compact and reserve source for
   targeted `read_span`;
2. tighten tool guidance and later measure candidates such as `k=3–5`, a much
   smaller inline budget, one initial search, and a bounded number of refinements;
3. only after response volume is controlled, route lexical questions to FTS and
   evaluate an explicitly approved hybrid/dense path for semantic or multi-step
   questions.

An unchanged repeat is rejected. Response-volume correction must precede any
new A/B; applying the second, tool-guidance correction in the same controlled
revision is recommended, but was not a unanimous reviewer hard gate. Exact new
defaults are experiment candidates, not product constants accepted by this
result.

## Artifact digests

Generated artifacts remain in ignored local state below run ID
`assistant-ab-v3-20260820T143000Z`.

| Artifact | SHA-256 |
| --- | --- |
| run manifest | `a371decfdfecc5c8f7fbeff4efc2a2ec6f053377507f0dfd3c43f225806ba64c` |
| tool schema | `f92deceaae6de250d738b73970e6a2f255e26a8a4ea5d265e458326ef668cb96` |
| journey-freeze manifest | `a08af973998a0a012bd1763409ab475e567551ca9285994e22bd47b95d3ae2ea` |
| frozen journey JSONL | `870f0a7178aa37330a01e4d04011268f3ef12aa18b67e61ff37990ebfaac8308` |
| Go blind grades | `8f77485406472b3f291beeb0d6e79f9b9cab86403f8bba3357c9eccc7787efaf` |
| RHF blind grades | `35bf103609d8a40bb45ed6e28e6844be4b29a6862f5b66adabdac7540f5c796f` |
| paired results | `6160c1b64f044d0a4836a9b3da1ef3352b9c7a11ee11c1841ae05324af1060be` |
| aggregate | `69d9e2e6522def0638dff75e793e0d371195ee0c9142a14bb0f753a4477c2323` |
| local report | `d7c649c0e89637ecb9f10b79a0efbed78de512a6ffbc42520b87a75abedb2717` |

These are the final artifacts after critical-cohort aggregation was added. Raw
answers, blind grades, and the arm-blind frozen journey were not rewritten.

## Checks run and not run

Run:

- both external pre-execution reviews and both post-result reviews;
- isolated MCP four-tool and clean-status preflight;
- 24 V2 and 24 V3 scored Codex CLI task turns plus schema probes;
- full-coverage machine journey freeze and two-corpus blind grading;
- manifest/schema parsing, runner/scorer syntax checks, and final aggregation;
- focused Go build of the no-provider A/B MCP launcher;
- `git diff --check`.

Not run or not claimed:

- no Voyage query, paid embedding, hybrid A/B, new corpus, HNSW, or promotion;
- no repeat estimate of model stochasticity;
- no product response-size or routing implementation change;
- no broad project test suite.
