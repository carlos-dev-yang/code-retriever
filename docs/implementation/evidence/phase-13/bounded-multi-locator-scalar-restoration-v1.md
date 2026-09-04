# Bounded Multi-Locator Scalar Restoration V1

- Date: 2026-09-05
- Phase: 13
- Restoration commit: `ff9d4e8`
- Status: `COMPLETE`
- Product contract after restoration: scalar `read_span` v1

## Restored boundary

The terminal experiment rejected the evaluation-only batch-v2 branch. The 12
implementation and evaluation-launcher paths changed for that branch were
restored byte-for-byte to the accepted scalar baseline at `e6c21af`. The
historical plan, frozen manifest, result, and ignored run artifacts were not
removed or rewritten.

The stable MCP surface is again exactly `status`, `search`, `read_span`, and
`reindex`. `read_span` accepts one `path`, `start_line`, `end_line`, and
`expected_sha256`; it returns one complete, byte-bounded source span after
path, range, current-file hash, and source-size validation. There is no
`input_version`, locator array, partial batch success, fifth tool, retrieval
change, or persistent server session state.

## Checks actually run

- Exact comparison of all 12 restored paths against `e6c21af`: pass.
- `go test -count=1 ./internal/app ./internal/mcp`: pass.
- `go test -count=1 -race ./internal/app ./internal/mcp`: pass.
- `go vet ./internal/app ./internal/mcp ./scripts/assistant-ab-mcp`: pass.
- Go package and assistant MCP launcher builds: pass.
- Python compilation of the four affected existing runner/trace/scorer files:
  pass without writing bytecode.
- Reconstruction of all 60 historical scalar policy traces from their frozen
  events/final records and comparison with preserved JSON: pass.
- `git diff --check`: pass.

The terra/high implementation reviewer performed the scoped restoration and
an initial validation pass. The main agent independently inspected the MCP
registry, scalar decoder/service path, and removed runner branches, repeated
the focused test/race/vet/build/Python checks, and repeated the 60-trace replay
before committing.

## Checks intentionally not run

No full-project or cross-platform suite was run because the change only
reverses the bounded experiment branch to an already accepted baseline. No
Voyage request, dense/hybrid query, source-bank mutation, corpus change,
reindex, packaging claim, or release-candidate promotion was performed.

## Remaining boundary

Before another scored assistant evaluation, the grader must prospectively
accept every product-valid byte-bounded source range or deterministically
present long ranges without breaking citation identity. The closed run is not
regraded. Product work now returns to the owner-gated Phase 12 confirmation;
Phase 14 release-candidate work remains blocked on its immutable core result.
