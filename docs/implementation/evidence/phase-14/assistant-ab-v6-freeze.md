# Paired Assistant A/B V6 Freeze Checkpoint

- Date: 2026-08-22
- Status: frozen and preflight-valid; no scored V6 turn yet
- Manifest: `testdata/retrieval/assistant-ab-chi-rhf-v6.json`
- Plan: `docs/implementation/ASSISTANT-AB-TEST-PLAN-V6.md`
- Scope: diagnostic assistant use only

## Frozen comparison

The V5 and V6 JSON values for `controls`, `question_sources`, `corpora`, and
all 12 ordered task records are exactly equal. The V6 prompt is exactly the V5
prompt plus this one sentence after the exact-range instruction:

> When calling `read_span`, pass `path`, `start_line`, `end_line`, and
> `expected_sha256` exactly as returned by the selected locator.

No other prompt byte changed. The sentence names a field already required by
the frozen tool schema. It does not modify the product, MCP wire, FTS planner
or ranking, caller-selected `k` semantics with default 10, locator identity,
or source-return behavior.

The same complete prompt is supplied to both arms. Baseline has no cidx
server; treatment exposes the unchanged locator-only cidx server. Model,
reasoning effort, task and arm order, source/state isolation, timeout, answer
schema, reducer v3, official token accounting, and blind grading are
unchanged. Every scored V6 observation must be new.

## Passed preflight

Preflight run `assistant-ab-v6-preflight-20260822` and schema-probe run
`assistant-ab-v6-schema-probes-20260822` verified:

- both approved corpus commits, trees, clean source worktrees,
  configurations, and unchanged index databases;
- FTS default and disabled paid-query embedding;
- exact `status`, `search`, `read_span`, and `reindex` discovery;
- structured-only tool results and isolated status operation;
- the logged-in retained native Codex CLI 0.148.0 binary;
- plan, manifest, runner, scorer, answer schema, binaries, and tool schema;
  and
- valid baseline and treatment schema output with zero repository commands,
  zero MCP calls, no timeout, and no control violation.

The probes are input-accounting controls only. Their token values are
unscored and are not subtracted from task usage.

## Frozen identities

| Input | SHA-256 |
| --- | --- |
| V6 plan | `d1dd7d97d4e5400ed44c8147cabae2ed4b89364a530a6918f2750b4415b72a09` |
| V6 manifest | `6294dcef9ec65fb6e3dd2b23b5756d9b3edd595dfc655f0ca1de1a974c716cca` |
| runner | `dfe931e02e467556d479d6268398f6f353912722a3f41dc52c6f24e7596b56b3` |
| reducer/scorer v3 | `f215fe893713491dd6034e3f19d1227d61d356ca80baf58b4998112cdab586a3` |
| answer schema | `696bca2293a4aeaeca46b2eb0809e75c2a2d13e55b06b7cef4b89f8cc63e3d8b` |
| blind-grade schema | `3f9051a94d75df28cd9dbdbd32613ac23fb37edbfd17896b362cd08edf89454b` |
| native Codex CLI | `b0308517b20543012fa2171aa3d46ce455a7456c4eb2a552ab9468ba4eeb1e50` |
| cidx | `7ef69d1cc3b04007a3460625333281a437f188b279bc2688b743cb95ebf5c419` |
| MCP launcher | `d4494bc0d0a22d2be7f99e784b63340380b551588d3f75bc8fc7b3822fb0729b` |
| discovered tool schema | `663a2918088b0393d0a26204ac41981935f9a160db65032dd7e764c40eab1fa2` |
| preflight run manifest | `911b2c662276d38afd6d484f837e96f82dd589432b6abec917ae9e9c22f3cb4d` |
| schema-probe run manifest | `a8faec27ae299bf47481f9171675f5aedf22aa1f632240ec070d3e90d2f9320d` |
| preflight/schema-probe tool artifact | `703ed38327258872522ca74ee73f340bf538dc78e9315c2e43015f038689daec` |

The retained chi database remains
`4d82c7539a013e9c34f63d95be7d591d7d607b91a273410dc1a178398bfcb5bc`;
the retained React Hook Form database remains
`196dd590396a1f123304e8dc399b4f6d21fd02d4ad65e20b7dc09ab7ef78a43d`.

No provider key, Voyage request, hybrid search, embedding, corpus change,
question edit, source mutation, product code change, or scored assistant turn
occurred in this checkpoint.

## Final-run boundary

Run all 24 scheduled V6 turns exactly once under one new run ID. Do not edit a
frozen input, reuse a V5 observation, or selectively retry a turn. Prepare one
arm-blind grade packet per corpus, obtain one tool-free grade invocation per
packet, and aggregate all pairs with reducer v3.

The hash mechanism, blind correctness, and token gate are separate decisions.
After the complete result, send the same fixed result packet to ChatGPT and
Grok and close V4–V6 under every outcome. No V7 follows from this sequence.
