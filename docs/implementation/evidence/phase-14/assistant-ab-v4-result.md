# Paired Codex CLI Assistant A/B — Version 4 Result

- Status: `complete_diagnostic`
- Date: 2026-08-22
- Run ID: `assistant-ab-v4-20260821T151700Z`
- Scope: existing chi and React Hook Form calibration questions only
- Promotion authority: none
- External post-result review: pending before V5 selection

## 1. What changed

V4 reran all 12 task pairs under the frozen V3 panel, prompt, model, arm
order, FTS planner/ranking, caller-selected `k`, and `read_span` contract. The
sole treatment change was the accepted Phase 13 response projection:

- `search` returned one structured representation;
- every result was a compact nine-field locator;
- search returned no source, signatures, scores, or diagnostics; and
- selected source entered through `read_span` only.

Both schema probes and all 24 scored turns were new. No V3 baseline was reused
and no V4 observation was selectively retried.

## 2. Execution and correctness

All 24 scored turns were valid. Treatment used cidx first on 12/12 tasks, made
no shell-cidx, hybrid, reindex, provider, or network action, and had no control
violation. One arm-blind grader invocation per corpus packet covered all 24
opaque answers before the key was restored.

| Arm | Complete | Partial | Incorrect | Ungradable |
| --- | ---: | ---: | ---: | ---: |
| baseline | 12 | 0 | 0 | 0 |
| cidx FTS | 12 | 0 | 0 | 0 |

There was no correctness conversion or reversal. V4 therefore proves the
compact locator contract is correctness-safe on this bounded panel. It does not
prove a population accuracy gain.

## 3. End-to-end token result

| Measure | Baseline | cidx FTS | Treatment direction |
| --- | ---: | ---: | ---: |
| model-total sum | 1,531,482 | 1,522,233 | -9,249 (-0.6%) |
| input sum | 1,512,124 | 1,502,131 | -9,993 (-0.7%) |
| uncached-input sum | 309,180 | 339,379 | +30,199 (+9.8%) |
| complete answers | 12 | 12 | equal |

The paired dual-complete model-total ratio median was `1.044`, its fixed-seed
task-bootstrap interval was `[0.768, 1.231]`, and only `4/12` pairs were
non-increasing. The frozen threshold required a median no greater than `0.85`
and at least `8/12` non-increasing. V4 therefore does **not** demonstrate token
reduction even though its aggregate model-total sum happened to be 0.6% lower.

The median uncached-input ratio was `1.147`; `5/12` uncached ratios were
non-increasing. The aggregate JSON retains a legacy field named
`non_increasing_count` for this uncached count. The efficiency gate uses the
explicit `model_total_non_increasing_count=4` and is not ambiguous.

By language, model-total ratio medians were Go `1.231`, TypeScript `1.036`, and
TSX `0.768`. The cells contain four tasks each and diagnose behavior; they are
not population estimates.

## 4. Response compaction succeeded

The transport-level intervention worked exactly as designed:

| Treatment observation | V3 source-bearing | V4 locator-only |
| --- | ---: | ---: |
| structured search bytes | 390,071 | 72,172 |
| text search bytes | 402,601 | 0 |
| search source bytes | 170,423 | 0 |
| search calls | 19 | 23 |
| `read_span` calls | 29 | 39 |

V4 search output was 81.5% smaller than V3's structured search projection and
removed the equivalent text copy and all search source. V4 total cidx result
event bytes were 160,738 (72,964 search plus 87,774 evidence), versus 938,001
in the corrected V3 event accounting: an 82.9% reduction. Event-envelope bytes
remain a transport proxy and are not relabeled model-visible.

Cross-version model-token totals are descriptive only because V3 and V4 are
separate stochastic batches. V4's paired contemporaneous result remains the
efficiency authority.

## 5. Locator and evidence stages

| Stage measure | V4 result |
| --- | ---: |
| first-search complete locator hit | 7/12 |
| first-search requirement coverage, macro | 0.667 |
| any-search requirement coverage, macro | 0.833 |
| first useful locator rank, median | 1 |
| locator occurrences / task-local unique sum | 260 / 192 |
| duplicate locator exposure rate, macro | 0.144 |
| locator-to-read selection utilization, macro | 0.294 |
| locator-to-final-citation utilization, macro | 0.213 |
| navigation false-lead rate, macro | 0.258 |
| complete selected evidence hit | 12/12 |
| selected evidence requirement coverage, macro | 1.000 |
| successful / attempted `read_span` | 33 / 39 |
| read precision, macro | 0.586 |
| evidence citation utilization, macro | 0.904 |
| gross / unique read source bytes | 77,173 / 63,519 |
| redundant evidence ratio, macro | 0.148 |

The exact locator metric is intentionally strict: a candidate must overlap a
frozen accepted span. Two tasks reached complete selected evidence despite
incomplete any-search locator coverage because the returned file/neighborhood
was enough for the assistant to choose a wider decisive read. This is useful
navigation but does not rewrite the frozen exact-span metric. A later report
may add a separate file/neighborhood navigation measure without changing it.

## 6. First remaining loss

V4 made 62 cidx calls: 23 searches and 39 reads. The residual cost is no
longer bulk search serialization. It is the assistant's evidence-acquisition
behavior after receiving compact candidates.

Observed mechanical waste includes:

- six failed `read_span` calls whose requested end line exceeded the file;
- four exact duplicate successful `read_span` calls;
- one first search missing the required compatibility argument, followed by
  two identical `NewRouter` searches with different maxima;
- six successive search formulations plus five reads for
  `chi-g02-route-outcome`; and
- repeated two- or three-search refinement on four additional tasks.

At least 10/39 read attempts (six failures plus four exact repeated successes)
were avoidable without changing retrieval. The broader 0.586 read precision
shows additional selected ranges did not intersect frozen required groups,
although the 0.904 evidence-citation utilization shows most returned reads
still supported some final explanation. These are different denominators and
must not be collapsed into one quality score.

The expensive critical cells were multi-requirement (median model ratio
`1.558`) and the single known-hard-negative task (`1.900`). Mixed-signal was
`0.908`; contract-disambiguation was `0.954`. The evidence supports fixing
search/read orchestration before reducing result depth or changing retrieval.

## 7. Decision boundary before V5

Supported now:

- keep the locator-only structured contract; it removed the diagnosed payload
  problem without correctness loss;
- do not claim token benefit from V4;
- do not reduce `k` yet: exact first-search completeness is only 7/12 and later
  candidates/refinements still contribute evidence;
- do not enable hybrid/dense or spend provider tokens; and
- do not add a repository or change questions/truths.

The smallest evidence-backed V5 candidate is one versioned assistant
orchestration intervention: avoid identical search/read repeats, use returned
locator ranges before speculative expansion, permit a bounded refinement only
for a named missing material claim, and stop when every material answer claim
has direct evidence. It changes neither index, query planner, rank, result
depth, schema, nor `read_span` semantics.

This candidate is not frozen by this report. It must first receive the required
ChatGPT/Grok post-result review. V5 must then preserve the V4 panel and create a
new complete 24-turn batch. A server-side range-clamping contract change or
smaller fixed `k` remains a separate later decision and cannot be combined
with the orchestration test.

## 8. Artifact identities

Generated artifacts remain in ignored local state under the run ID.

| Artifact | SHA-256 |
| --- | --- |
| run manifest | `1a368260147a8e3bf5208b21eee3dbe59d1052a5892ed28d45c8b8f9c30645a7` |
| tool schema | `703ed38327258872522ca74ee73f340bf538dc78e9315c2e43015f038689daec` |
| journey-freeze manifest | `583e414ca31c96c112c9abdd6ad1e2925526ef0a197b1ed16563df79bfd485d7` |
| frozen journey JSONL | `799de994d0e4acc023be69a25b6890fc7e1774d58073e04d1706296e36cbe48c` |
| Go blind grades | `7f239497dfeba947ae640d43862b962b647e02babf6b693a7b81b0cccf1ee008` |
| RHF blind grades | `98c2ac118e2a64001e704a21d73f98f70fbbdfa020bb521d87ed2841985ed842` |
| paired results | `bd2334249bec15cfdebfb98f0fa66d96098446ae7ce580baa4f4bfacbdd625fa` |
| aggregate | `f202126f7e019b0f899d7f127cd6e305a95d998072eda74eab9fe961bf70c553` |
| generated local report | `902a208c5be9154479e75f6763c57f12b57e7bcf07d15d559723d7369b91a5a1` |

Frozen execution inputs include manifest
`728549057cb7da6c1a8e22c0a074652810b40f518f9582d0a0c0f7ffd7a1a9a0`,
plan `e23b1edda115b12c760ab95d54c8c0a02cd7a1d3125a1dcef4facaa2f860d716`,
runner `dfe931e02e467556d479d6268398f6f353912722a3f41dc52c6f24e7596b56b3`,
reducer `6a754955ce620ae4dbe02a617a1b207c86a80b261b3015125443b9055fe946b2`,
native Codex `b0308517b20543012fa2171aa3d46ce455a7456c4eb2a552ab9468ba4eeb1e50`,
cidx `7ef69d1cc3b04007a3460625333281a437f188b279bc2688b743cb95ebf5c419`,
and MCP launcher
`d4494bc0d0a22d2be7f99e784b63340380b551588d3f75bc8fc7b3822fb0729b`.

## 9. Checks run and not run

Run:

- exact native-Codex login, corpus/tree/state, tool-schema, and representation
  preflight;
- two unscored schema probes and 24 scheduled task turns;
- 24/24 valid-execution, source/state-isolation, and control checks;
- reducer-v2 24-row arm-blind journey freeze;
- two full-coverage blind-grade packets and ID/group reconciliation;
- paired, language, critical-cohort, locator, evidence, token, and artifact
  aggregation; and
- manifest/prompt/task equality, JSON parsing, Python syntax, and digest checks.

Not run or claimed:

- no Voyage query, paid embedding, hybrid arm, new corpus, ranking/query change,
  result-depth change, or repeated stochastic estimate;
- no another-host or another-platform test;
- no `core_retrieval` or `release_candidate` promotion; and
- no V5 implementation before post-result external review.
