# cidx

cidx is a lightweight local code-search MCP server under active v1 implementation. It combines free local AST/FTS indexing with optional, explicitly authorized Voyage AI embeddings and keeps SQLite as the persistent authority.

The Revision 4 core through Phase 13 and the Phase 14 corpus-independent implementation checkpoint are accepted. The local darwin/arm64 package and operational checkpoint were accepted from clean provenance `a5b2baef9a18e68d6c8b5d4fb62dc2e03727edb4`; Phase 14 remains blocked only for its official evaluation and release-candidate scope. Consult the status ledger for the exact boundary.

The current Phase 07 chi/RHF checkpoint includes the clean 32-query Voyage
embedding-search run and dual-AI advisory replays. It rejects OR fusion as a
serving candidate and leaves production AND unchanged. The advisory labels are
not human label authority; formal calibration still waits on the required
separated human source review, after which the immutable ranks can be replayed
without another Voyage call.

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
The owner selected Apache-2.0 and its unmodified canonical text is in
[`LICENSE`](LICENSE). The accepted checkpoint covers only the locally verified
darwin/arm64 archive and project-scoped Codex configuration read; other
platforms and hosts, code signing, and notarization remain unsupported or
unverified. It does not establish `release_candidate` status.

## Start here

1. Read [repository instructions](AGENTS.md).
2. Read the [implementation execution guide](docs/implementation/EXECUTION-GUIDE.md).
3. Check the [persistent phase status](docs/implementation/STATUS.md).
4. Open the active phase from the [implementation plan index](docs/implementation/README.md).

The canonical product design is [Revision 4](local-code-search-mcp-v1-design-r4.md), and evaluation behavior is governed by the [evaluation and promotion contract](docs/implementation/EVALUATION-CONTRACT.md).

Do not add a Go module path, license, remote, paid API call, or evaluation corpus by inference. Those actions require their documented phase decision or explicit user direction.
