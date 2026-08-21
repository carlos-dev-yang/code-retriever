# Paired Assistant A/B V4 Freeze Checkpoint

- Date: 2026-08-22
- Status: frozen and preflight-valid; no scored V4 turn yet
- Manifest: `testdata/retrieval/assistant-ab-chi-rhf-v4.json`
- Plan: `docs/implementation/ASSISTANT-AB-TEST-PLAN-V4.md`
- Scope: diagnostic assistant use only

## Frozen comparison

The V3 and V4 JSON values for `question_sources`, `corpora`,
`prompt_template`, and all 12 ordered task records are exactly equal. V4
retains the same native Codex CLI 0.148.0 binary, model, reasoning effort,
read-only isolation, arm order, mandatory first FTS search, caller-selected
`k`, and no-selective-retry rule.

The sole treatment change is `locator_only_structured_search_result`:

- `search` returns one structured representation;
- each result is a nine-field locator;
- search returns zero source bytes and no signatures/scores/diagnostics; and
- selected source enters through `read_span` only.

The runner compares its result-representation CLI option with the frozen
manifest. An intentional `dual` invocation exited 2 before creating a run and
reported `got dual, want structured`. It also resolves the selected Codex
binary before entering per-turn temporary working directories.

One attempted full-run directory reached preflight and then stopped at the
first unscored baseline schema probe because the runner had forwarded the
caller's relative Codex path into the temporary working directory. It created
no schema observation and ran zero scored tasks. The path was resolved at the
runner boundary before any V4 observation existed; the manifest, prompt,
schedule, model, and treatment remained unchanged. A new run ID is required
for the complete execution.

## Passed preflight

Preflight run `assistant-ab-v4-preflight-native-20260822` verified:

- both existing approved corpus commits, trees, clean worktrees, configs, and
  index databases;
- FTS default and disabled paid-query embedding;
- exact `status`, `search`, `read_span`, and `reindex` discovery;
- isolated status operation and server shutdown;
- the existing logged-in native Codex CLI 0.148.0 binary;
- the answer schema, plan, manifest, runner, cidx, launcher, and tool schema;
  and
- structured-only result selection.

The first npm wrapper-based preflight was not accepted as the frozen control
because its recorded launch-file hash described `codex.js`, not the exact V3
native executable. No scored turn was run under that discovery. The accepted
preflight and all future V4 turns use the native binary directly.

## Frozen identities

| Input | SHA-256 |
| --- | --- |
| V4 plan | `e23b1edda115b12c760ab95d54c8c0a02cd7a1d3125a1dcef4facaa2f860d716` |
| V4 manifest | `728549057cb7da6c1a8e22c0a074652810b40f518f9582d0a0c0f7ffd7a1a9a0` |
| runner | `dfe931e02e467556d479d6268398f6f353912722a3f41dc52c6f24e7596b56b3` |
| native Codex CLI | `b0308517b20543012fa2171aa3d46ce455a7456c4eb2a552ab9468ba4eeb1e50` |
| cidx | `7ef69d1cc3b04007a3460625333281a437f188b279bc2688b743cb95ebf5c419` |
| MCP launcher | `d4494bc0d0a22d2be7f99e784b63340380b551588d3f75bc8fc7b3822fb0729b` |
| discovered tool schema | `663a2918088b0393d0a26204ac41981935f9a160db65032dd7e764c40eab1fa2` |
| accepted path-fix/schema-probe run manifest | `a203fbcf48613b3a24be8cdcb202d225e04ee664e9a6dc64ae8ee0b2f4282262` |
| accepted preflight tool-schema artifact | `703ed38327258872522ca74ee73f340bf538dc78e9315c2e43015f038689daec` |

The retained chi state database is
`4d82c7539a013e9c34f63d95be7d591d7d607b91a273410dc1a178398bfcb5bc`;
the retained React Hook Form state database is
`196dd590396a1f123304e8dc399b4f6d21fd02d4ad65e20b7dc09ab7ef78a43d`.
No provider key, Voyage call, hybrid search, corpus change, or source mutation
occurred.

The post-fix check used the same relative CLI argument that had failed before.
The runner resolved it to the frozen native executable, then both baseline and
treatment schema probes completed with valid schema output, zero repository
commands, zero MCP calls, and no control violation. The treatment receives MCP
schema context in this probe but the fixed probe prompt intentionally invokes
no tool. Probe token counts are unscored and are not subtracted from task
usage.

## Next action

Run exactly two schema probes and all 24 scheduled task turns under a new V4
run ID. Do not change any frozen input or selectively retry an observation.
Then prepare arm-blind grading packets, obtain one grade per corpus packet,
freeze reducer-v2 journey rows before arm restoration, aggregate all tasks, and
record the stage-separated result.
