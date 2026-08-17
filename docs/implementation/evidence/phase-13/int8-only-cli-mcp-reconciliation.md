# Phase 13 int8-only CLI and MCP reconciliation

- State: accepted at the Phase 13 commit boundary.
- Date: 2026-08-17.
- External actions: no credential read, corpus read, provider request, network
  operation, paid action, indexing action, or metric run was performed.

## Accepted public behavior

- `cidx init` defaults to 1024/int8 and accepts only
  `--serving-dim <1024|512>`. There is no codec or target-dimension flag.
- Help makes 1024 the visible default and identifies 512 as compact.
- After an explicit existing-config dimension change and `cidx index`
  reconciliation, `cidx embed --apply` uses compatible product source rows
  locally. It requires `VOYAGE_API_KEY` and constructs a provider client only
  when the plan has missing source inputs.
- Product config remains the single profile authority. Neither CLI nor MCP
  exposes Binary/256 or an arbitrary codec selector.
- MCP registers exactly `status`, `search`, `read_span`, and `reindex`. Search
  remains the Phase 11 implementation; transport adaptation does not add a
  second ranker or body packager.

The single `cidx` binary also contains the explicit public embedding command,
so whole-binary package linkage is not the source-bank isolation boundary.
The verified boundary is that MCP and search source code neither imports nor
opens product source-bank or lab state.

## Commit-boundary validation

The first focused run found only that revised help no longer contained the
existing stable phrase `init creates local config`. The wording was adjusted
without changing behavior, and the final boundary passed:

```text
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 ./internal/config ./internal/root ./internal/app ./internal/cli ./internal/mcp ./internal/index ./internal/store ./internal/search
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 -race ./internal/root ./internal/app ./internal/cli ./internal/mcp ./internal/index ./internal/store ./internal/search
env -u VOYAGE_API_KEY GOPROXY=off go vet ./cmd/cidx ./internal/config ./internal/root ./internal/app ./internal/cli ./internal/mcp ./internal/index ./internal/store ./internal/search
env -u VOYAGE_API_KEY GOPROXY=off go build ./cmd/cidx
gofmt -l cmd internal
go run ./cmd/cidx help
go mod tidy -diff
git diff --check
```

Static inspection found no retired codec/dimension flag or identifier in
non-test CLI/app/MCP code, no source-bank/lab import in MCP/search sources, and
exactly four tool definitions.

## Handoff

Phase 14 must rebuild the local package/verifier smoke from the current clean
commit and prove default 1024/int8 plus provider-free compact-512
rematerialization. Official assistant and release-candidate evidence remains a
separate gate.
