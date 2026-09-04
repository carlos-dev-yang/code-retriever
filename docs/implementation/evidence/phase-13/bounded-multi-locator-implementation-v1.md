# Bounded Multi-Locator Implementation V1

- Date: 2026-09-04
- Phase: 13
- Implementation commit: `6737494c74c1235eb32b2eeb25c5f7d7f6d89730`
- Preflight variable-shadow correction: `3447758425bdec6dd3e708d0708a77e56b9c0f11`
- Disposition: `FROZEN_FOR_EXECUTION`

## Implemented boundary

The scalar `read_span` input, output, typed errors, and default MCP server are
unchanged. The evaluation launcher can opt into a strict v2 input containing
two to four unique locator tuples. It validates the whole request, buffers
ordered scalar-shaped evidence units, enforces the existing aggregate
`mcp.hard_max_inline_bytes`, and returns no evidence when any item fails.

The batch branch adds no retrieval, index, SQLite, embedding/provider,
configuration, session-state, automatic context-expansion, or fifth-tool
behavior. The existing runner selects `scalar-v1` for the reference arm and
`batch-v2` for the treatment arm while holding every other MCP field and the
prompt identical.

Trace accounting keeps one batch request as one MCP invocation and preserves
its individual evidence units for provenance. Batch eligibility is defined
from the scalar reference trace, before treatment adoption is known. Duplicate
and overlapping reads remain diagnostics rather than hidden corrections.

## Focused checks actually run

- `go test -count=1 ./internal/app ./internal/mcp`
- `go test -count=1 -race ./internal/app ./internal/mcp`
- `go vet ./internal/app ./internal/mcp ./scripts/assistant-ab-mcp`
- focused Go builds for the affected packages and evaluation launcher
- Python syntax compilation for the existing runner, scorer, and two trace
  builders
- exact byte-for-byte replay of all 60 historical scalar policy traces
- synthetic batch trace/accounting checks
- provider-free direct MCP probes for both contracts against the existing chi
  FTS state
- `git diff --check`

All listed checks passed. The scalar direct probe retained the previously
frozen definition, description, schema, and functional-output hashes. The v2
probe exposed the same four tool names and a distinct versioned `read_span`
schema and functional response.

## Frozen host preflight

After the corrected manifest was committed, clean binaries were built from
`b324ed53e85595ff4e169fd30370bfba68e16bf7`.

- Provider-free runner preflight:
  `assistant-read-span-multi-locator-v1-preflight-002` (`PASS`)
- Preflight run-manifest SHA-256:
  `e8b6a4988cbd8c1a2586df1811e4973654e12a5df6656fe077597e1a21486890`
- Codex host schema probes:
  `assistant-read-span-multi-locator-v1-schema-probes-001` (`2/2 PASS`)
- Schema-probe run-manifest SHA-256:
  `e3a3d4e02e5a5197ca90d4828f23de4a78d5b6847ec7ab19a8c7c6e40ea66611`

The scalar arm exposed no v2 discriminator and retained its historical tool
contract hashes. The batch-capable arm exposed `oneOf`, `input_version`, and
`locators`; its ordered two-locator functional probe passed. Both arms exposed
exactly `read_span`, `reindex`, `search`, and `status`, and neither process
received provider credentials. Actual Codex CLI sessions accepted both tool
schemas and returned the required answer envelope without invoking a scored
task.

## Not run at this checkpoint

- No scored assistant turn or blind-grade call has run.
- No primary 30-pair execution, blind grade, or aggregate has run.
- No full-project suite, paid embedding, dense/hybrid query, reindex, product
  promotion, or packaging check was run.

The derived manifest is
[`assistant-read-span-multi-locator-chi-rhf-v1.json`](../../../../testdata/retrieval/assistant-read-span-multi-locator-chi-rhf-v1.json).
It preserves the prior 30 tasks, questions, truth, corpora, schedule, model,
FTS state, grading path, and trust prompt. The first frozen preflight exposed
one runner-local variable shadow: the per-arm tool schema replaced the answer
schema path after MCP validation. No scored turn ran. The correction only
renamed that local value and the manifest was re-frozen before retrying the
unscored preflight. The corrected manifest SHA-256 is
`e75ff8ecec219ab421725aa2b4dea4859e51e44c59a7c66cd585d94b46938322`.
