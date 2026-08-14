# cidx

cidx is a planned lightweight local code-search MCP server. It combines free local AST/FTS indexing with optional, explicitly authorized Voyage AI embeddings and keeps SQLite as the persistent authority.

Phase 00 shared contracts are complete. Implementation code has not started yet; consult the status ledger for the next eligible phase.

## Product boundary

- Local indexing and FTS search never require an API key or network access.
- Paid document embedding and paid hybrid-query embedding are separate explicit operations.
- The stable MCP surface is limited to `status`, `search`, `read_span`, and `reindex`.
- Production serves one configured vector profile and stores only its cidx-owned `binary` or `int8` representation.
- The development-only document-f32 lab is isolated from runtime serving.

## Start here

1. Read [repository instructions](AGENTS.md).
2. Read the [implementation execution guide](docs/implementation/EXECUTION-GUIDE.md).
3. Check the [persistent phase status](docs/implementation/STATUS.md).
4. Open the active phase from the [implementation plan index](docs/implementation/README.md).

The canonical product design is [Revision 3](local-code-search-mcp-v1-design-r3.md), and evaluation behavior is governed by the [evaluation and promotion contract](docs/implementation/EVALUATION-CONTRACT.md).

Do not add a Go module path, license, remote, paid API call, or evaluation corpus by inference. Those actions require their documented phase decision or explicit user direction.
