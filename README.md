# cidx

cidx is a lightweight local code-search MCP server under active v1 implementation. It combines free local AST/FTS indexing with optional, explicitly authorized Voyage AI embeddings and keeps SQLite as the persistent authority.

The Revision 4 core through Phase 13 and the Phase 14 corpus-independent implementation checkpoint are accepted. Phase 14 artifact generation is blocked on an owner-selected project license; official corpus and release-candidate evidence remains separately gated. Consult the status ledger for the exact checkpoint.

## Product boundary

- Local indexing and FTS search never require an API key or network access.
- Paid document embedding and paid hybrid-query embedding are separate explicit operations.
- The stable MCP surface is limited to `status`, `search`, `read_span`, and `reindex`.
- Production serves one configured vector profile and stores only its cidx-owned `binary` or `int8` representation.
- The development-only document-f32 lab is isolated from runtime serving.

## Local Phase 14 surface

The local darwin/arm64 package builder and offline verifier are documented in
[install](docs/install.md). Codex project-scoped MCP setup, optional hook
composition, and fail-closed upgrade guidance live in
[hosts](docs/hosts.md), [hooks](docs/hooks.md), and [upgrade](docs/upgrade.md).
No archive is currently claimed: generation and public redistribution remain
blocked until the repository owner chooses and adds a project license.

## Start here

1. Read [repository instructions](AGENTS.md).
2. Read the [implementation execution guide](docs/implementation/EXECUTION-GUIDE.md).
3. Check the [persistent phase status](docs/implementation/STATUS.md).
4. Open the active phase from the [implementation plan index](docs/implementation/README.md).

The canonical product design is [Revision 4](local-code-search-mcp-v1-design-r4.md), and evaluation behavior is governed by the [evaluation and promotion contract](docs/implementation/EVALUATION-CONTRACT.md).

Do not add a Go module path, license, remote, paid API call, or evaluation corpus by inference. Those actions require their documented phase decision or explicit user direction.
