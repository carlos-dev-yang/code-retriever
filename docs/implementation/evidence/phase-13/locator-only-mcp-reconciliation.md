# Phase 13 Locator-Only MCP Reconciliation

- Date: 2026-08-22
- Scope: MCP result projection and Codex transport only
- Product retrieval change: none
- Provider/API action: none

## Accepted result contract

MCP `search` now returns one structured result representation containing only
ranked, deduplicated locators. It never returns source text. The retained
required `max_inline_bytes` input is validated for v1 wire compatibility and
is deliberately passed to the shared search service as zero, so it cannot
change result identity, order, or count.

Each result contains exactly:

```text
chunk_id
path
language
kind
qualified_symbol
start_line
end_line
indexed_sha256
match_sources
```

The adapter preserves the first ranked occurrence of a locator and merges only
its compact match-source labels. It omits source bodies, signatures, scores,
planner diagnostics, profile diagnostics, body-package state, and generation
metadata. `read_span` remains the only source-bearing cidx tool.

All successful and typed-error tool results default to structured-only MCP
content. The development-only launcher can still select text or dual output
for compatibility probes, but neither is the product default.

## Focused implementation evidence

The focused adapter fixture injects two ranked core hits for one canonical
parent. It proves that MCP:

- requests zero body bytes from the shared search service;
- keeps the first ranked chunk identity and parent line range;
- deduplicates the repeated locator and merges match-source labels;
- emits one top-level `results` member and exactly the nine locator fields;
- exposes neither body/signature nor rank/planner diagnostics; and
- emits no duplicate text copy beside `structuredContent`.

The directly affected normal and race test batches, vet, builds, Python syntax,
shell syntax, formatting, and diff checks passed:

```text
go test -count=1 ./internal/mcp
go test -count=1 -race ./internal/mcp
go test -count=1 ./internal/mcp ./internal/app ./internal/search ./internal/cli
go test -count=1 -race ./internal/mcp ./internal/app ./internal/search ./internal/cli
go vet ./internal/mcp ./internal/cli ./scripts/assistant-ab-mcp
go build -o /tmp/cidx-locator-bin ./cmd/cidx
go build -o /tmp/cidx-locator-mcp ./scripts/assistant-ab-mcp
python3 -m py_compile scripts/run-assistant-ab.py scripts/score-assistant-ab.py
bash -n scripts/verify-local-release.sh
git diff --check
```

## Real server budget-invariance probe

One isolated copy of the existing approved chi state was opened without
`VOYAGE_API_KEY`. Three sequential FTS calls used the same `NewRouter` query,
`k=10`, and `max_inline_bytes` values `0`, `12000`, and `65537`; the last value
is above that repository's configured 65536-byte source ceiling.

Every response had:

- `content=[]` and one `structuredContent` object;
- one top-level `results` key;
- 10 locators with the exact nine-field schema;
- zero source bytes; and
- the same canonical payload size (2667 bytes) and SHA-256
  `fc2ae1492edc32e2b6ea105a5f5926dc146c28b13cc3772bf1cc38be11ce612e`.

This proves the retained compatibility maximum does not control search output.
The independent `read_span` byte ceiling remains unchanged.

## Real Codex search-to-read probe

One isolated `chi-new-router` treatment turn used:

- Codex CLI `0.149.0-alpha.4`;
- model `gpt-5.6-sol` with high reasoning;
- the existing approved chi snapshot and V3 task/prompt;
- read-only source isolation and independently copied cidx state;
- FTS only, with no provider key; and
- the structured-only locator server.

The valid 25.31-second execution made no repository shell commands. Its first
repository-discovery action was cidx `search`, followed by two `read_span`
calls selected from the compact locators. It returned a source-backed answer
with no control violation. The search event had `content=[]`, one
`structured_content.results` list, and no body or diagnostic fields.

The retained local evidence digests are:

```text
events.jsonl     5827a1e61b05af5ad9e15024cf48cedf67d0d10af0484f30e1df2eaa7dcf3670
observation.json 563c6eb80101db0c86fdd2388e2ed0741b2405862c4fbe18f100fb771b0bb3d9
final.json       08b0e4c726e56b3c7cc4905b0e10c5f3bcd8d8e255b0e4b9856e4db80f2f3ad4
```

This is a conformance smoke, not an A/B efficiency result. It proves that
Codex can navigate from compact candidates to decisive source evidence. The
controlled paired V4 run remains Phase 14 work.

## Completion boundary

Phase 13 is complete for the revised public wire. Existing Phase 13 evidence
continues to cover the CLI, four-tool registry, lifecycle, concurrency,
cancellation, FTS-without-key behavior, source-bank isolation, and typed
`read_span` errors. This reconciliation replaces only its historical
source-bearing search-response evidence.

No claim is made for another MCP host, another OS/architecture, assistant token
savings, retrieval promotion, or release-candidate readiness. Those remain
Phase 14 concerns.
