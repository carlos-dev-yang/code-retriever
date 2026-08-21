# Paired Assistant A/B V5 Freeze Checkpoint

- Date: 2026-08-22
- Status: frozen and preflight-valid; no scored V5 turn yet
- Manifest: `testdata/retrieval/assistant-ab-chi-rhf-v5.json`
- Plan: `docs/implementation/ASSISTANT-AB-TEST-PLAN-V5.md`
- Scope: diagnostic assistant use only

## Frozen comparison

The V4 and V5 JSON values for `controls`, `question_sources`, `corpora`, and
all 12 ordered task records are exactly equal. The V5 prompt is exactly the V4
prompt plus the single externally reviewed orchestration paragraph before the
answer-format paragraph. No other prompt byte changed.

The paragraph conditionally requires one initial locator search with
`max_inline_bytes=0`, prohibits a repeated search and repeated read, starts
source acquisition at exact returned locator ranges, permits at most one
specifically justified refinement, and stops source requests when material
claims have direct evidence.

The same complete prompt is supplied to both arms. Baseline has no cidx server;
treatment exposes the unchanged locator-only cidx server. Current
caller-selected `k` semantics with default 10, FTS planning/ranking,
structured result representation, `read_span`, answer schema, model, task
order, isolation, timeout, and blind grading are unchanged.

## Reducer freeze

Reducer v3 was completed before this experiment. It observes the prompt's
mechanical clauses without turning noncompliance into a harness failure. Its
read-only V4 calibration reproduced four duplicate reads and six invalid
ranges, and recorded 2/23 exact locator-range first reads. The scorer does not
infer whether a refinement was semantically justified.

## Passed preflight

Preflight run `assistant-ab-v5-preflight-20260822` and schema-probe run
`assistant-ab-v5-schema-probes-20260822` verified:

- both approved corpus commits, trees, clean source worktrees, configurations,
  and unchanged index databases;
- FTS default and disabled paid-query embedding;
- exact `status`, `search`, `read_span`, and `reindex` discovery;
- structured-only tool results and isolated status operation;
- the logged-in retained native Codex CLI 0.148.0 binary;
- plan, manifest, runner, scorer, answer schema, binaries, and tool schema; and
- valid baseline and treatment schema output with zero repository commands,
  zero MCP calls, and no control violation.

The probes are input-accounting controls only. Their token values are unscored
and are not subtracted from task usage.

## Frozen identities

| Input | SHA-256 |
| --- | --- |
| V5 plan | `91bdf26059b8cb875047ce56a663a0436c34a75e3563c09fc2250553ee4f0f4e` |
| V5 manifest | `f1f60d634fb4aeb5c43aa4b67549aed41c94cd50f7af42bf1cca6fbf568ab9f5` |
| runner | `dfe931e02e467556d479d6268398f6f353912722a3f41dc52c6f24e7596b56b3` |
| reducer/scorer v3 | `f215fe893713491dd6034e3f19d1227d61d356ca80baf58b4998112cdab586a3` |
| answer schema | `696bca2293a4aeaeca46b2eb0809e75c2a2d13e55b06b7cef4b89f8cc63e3d8b` |
| blind-grade schema | `3f9051a94d75df28cd9dbdbd32613ac23fb37edbfd17896b362cd08edf89454b` |
| native Codex CLI | `b0308517b20543012fa2171aa3d46ce455a7456c4eb2a552ab9468ba4eeb1e50` |
| cidx | `7ef69d1cc3b04007a3460625333281a437f188b279bc2688b743cb95ebf5c419` |
| MCP launcher | `d4494bc0d0a22d2be7f99e784b63340380b551588d3f75bc8fc7b3822fb0729b` |
| discovered tool schema | `663a2918088b0393d0a26204ac41981935f9a160db65032dd7e764c40eab1fa2` |
| preflight run manifest | `c1a9c3a1b6f056aef70194389f9ac94bba1bd8a0afeacdd0ebb7238b20a08da1` |
| schema-probe run manifest | `8961f980be53a4a4186d5f84652af8a909a87ba5be309d7d4990f3ce28635677` |
| preflight/schema-probe tool artifact | `703ed38327258872522ca74ee73f340bf538dc78e9315c2e43015f038689daec` |

The retained chi database remains
`4d82c7539a013e9c34f63d95be7d591d7d607b91a273410dc1a178398bfcb5bc`;
the retained React Hook Form database remains
`196dd590396a1f123304e8dc399b4f6d21fd02d4ad65e20b7dc09ab7ef78a43d`.

No provider key, Voyage request, hybrid search, embedding, corpus change,
question edit, source mutation, product code change, or scored assistant turn
occurred in this checkpoint.

## Next action

Run the two unscored schema probes and all 24 scheduled V5 task turns once
under a new run ID. Do not edit a frozen input or selectively retry an
observation. Then prepare reducer-v3 arm-blind journeys and grading packets,
obtain one blind grade packet per corpus, aggregate every task, and record the
separate correctness, mechanism, and token conclusions before any V6 choice.
