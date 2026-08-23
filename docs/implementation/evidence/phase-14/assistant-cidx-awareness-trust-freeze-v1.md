# Assistant cidx Awareness vs Trust-Priority V1 Freeze

- Date: 2026-08-23
- Phase: 14
- Status: `FROZEN_FOR_EXECUTION`
- Scope: paired, non-promotion assistant-use diagnostic
- Scored turns observed at freeze: `0`

## Frozen comparison

The experiment reuses the approved 30 questions, truth, two corpora, model,
ordinary tools, FTS-only cidx state, answer schema, and counterbalanced 15/15
first-arm schedule. It runs 10 Go, 10 TypeScript, and 10 TSX questions.

- `neutral_cidx` / `aware_choice`: shared concise cidx awareness and free tool
  choice.
- `directed_cidx` / `trust_priority`: the same prompt and tools plus the exact
  frozen priority-and-bounded-trust suffix.

Both arms expose the same four MCP tools. `search` recommends omitted/default
`k=5` and explains its locator-only result; `read_span` describes exact current
source retrieval. Tool names, input/output schemas, ranking, FTS behavior, and
the source/index bindings are unchanged. Dense/hybrid search and provider
credentials are absent.

## Frozen identities

| Artifact | SHA-256 |
| --- | --- |
| Experiment manifest | `42bd09a18fa9a54c21e88c0620396f780e07dc6b305ce90a3d70d14097e8188f` |
| Runner | `b3c5350ad23710265bc7e0877f9bb83963fcd44f8e7818dcea75878c307733e0` |
| Scorer | `b48b85c8ac1c5dab2af08fb47c0f6366516208e069e53f190ce1176f7353e24e` |
| Policy trace builder | `2174d948e2282d9ffd2f4f1c6bac29736b1424dff4217d169a31b5aee9db2d6a` |
| Passive trace builder | `12ca918216a7e6767de5d33b1c41d48f9b412745d19798aa485c72e78ab72410` |
| Tool definitions | `7a8857c154da49063a4c84c3dc7ddb869a7d483629e1865bcdb9c16b6b45ac26` |
| Tool descriptions | `b7ad68fac11da0761252c2ab428ac0f3556f676a87e4c27d997cced36d99370b` |
| Tool input schemas | `c06571f982f38f1e274d79d1e1c3d46624066d56e98c8e73f441e59bd358b27d` |
| Functional tool probe | `6441f100d0fb69d558011314ae9aa34076b1fb0823d604edfbb5651379c4a266` |
| Arm B suffix | `dcff85d3b54704d342aa702287988277cacf91912cfd1688f05f66602b7c1855` |

The manifest is byte-derived from the closed predecessor manifest. Mechanical
comparison confirms identical taxonomy, question set, truth, question sources,
corpora, task order, question digests, and first-arm schedule. The new
manifest changes only experiment identity, the shared awareness prompt, Arm B
suffix, semantic labels, report wording, and the failure-safe denominator
contract required by the accepted plan.

## Implementation boundary

The implementation uses the existing runner, trace reducers, blind grader,
scorer, answer schema, and four MCP tools. Net non-documentation expansion is
exactly 500 lines including the 215-line frozen manifest; it does not cross the
plan's stop boundary. No new runner, grader schema, semantic sidecar, prior-run
audit, repository, question, product tool, or provider path was added.

The only execution corrections are:

- a private per-cell `reindex` remains observed but does not invalidate other
  isolated cells;
- operationally ungradable or incomplete-trace pairs are excluded from paired
  action/source/token efficiency estimates with explicit reasons;
- paired repository-action and unique-source-byte differences/ratios are
  recorded beside official token measures; and
- report labels use the actual comparable denominator rather than the 30
  scheduled pairs when failures or missing usage reduce it.

The bounded read-only code review returned `PASS` for these three denominator
and reporting corrections and found no direct regression.

## Checks actually run

- `go test -count=1 ./internal/mcp ./internal/evalcontract`
- `go test -count=1 -race ./internal/mcp`
- `go vet ./internal/mcp ./scripts/assistant-ab-mcp`
- clean builds of the cidx CLI and assistant MCP adapter
- Python syntax and CLI parsing for runner/scorer and both trace builders
- JSON parse, whitespace, and manifest identity checks
- provider-free synthetic checks for intact/incomplete traces, ungradable-pair
  exclusion, paired action/source metrics, undefined ratios, and exclusion
  reason counts
- read-only V3, V4, V5, and V6 scorer context loading
- provider-free preflight
  `assistant-cidx-awareness-trust-v1-preflight-007`, which matched the frozen
  manifest and all tool, source, index, prompt, execution-code, and trace
  identities

## Checks not run at freeze

- No scored primary assistant turn or blind-grade call was run.
- The two arm schema probes run immediately before the 60 scored turns and
  must both pass before the runner enters the scored loop.
- Original-session follow-up is omitted unless the existing ephemeral session
  lifecycle can be resumed reliably without new infrastructure. No semantic
  substitute is authorized.
- No full-project test suite, paid embedding, dense/hybrid query, or product
  promotion check was run because none is part of this experiment.

## Execution rule

After this freeze commit, execute the two unscored arm probes and then the 60
fresh primary turns exactly once. Preserve every failure without scored-cell
retry. Grade the new answers arm-blind against unchanged truth, aggregate once,
and interpret answer quality, code scope, actions, cidx adoption, and official
tokens as separate surfaces.
