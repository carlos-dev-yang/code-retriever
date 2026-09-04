# Bounded Multi-Locator Implementation V1

- Date: 2026-09-04
- Phase: 13
- Implementation commit: `6737494c74c1235eb32b2eeb25c5f7d7f6d89730`
- Disposition: `READY_TO_FREEZE`

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

## Not run at this checkpoint

- No scored assistant turn or blind-grade call has run.
- The actual Codex host schema probes and frozen runner preflight remain the
  next unscored checks.
- No full-project suite, paid embedding, dense/hybrid query, reindex, product
  promotion, or packaging check was run.

The derived manifest is
[`assistant-read-span-multi-locator-chi-rhf-v1.json`](../../../../testdata/retrieval/assistant-read-span-multi-locator-chi-rhf-v1.json).
It preserves the prior 30 tasks, questions, truth, corpora, schedule, model,
FTS state, grading path, and trust prompt. Its pre-commit SHA-256 is
`44dbb7577a6bc32f092a761071afeacf9c865bd13ea88e8d7362cd512d7f9935`.
