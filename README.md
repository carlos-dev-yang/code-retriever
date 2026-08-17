# cidx

cidx is a lightweight local code-search MCP server under active v1 implementation. It combines free local AST/FTS indexing with optional, explicitly authorized Voyage AI embeddings and keeps SQLite as the persistent authority.

The Revision 4 corpus-independent implementation is accepted through the current Phase 14 local package boundary. Default 1024/int8, provider-free compact 512/int8, retired Binary/256 rejection, and source-bank-free serving were verified in a darwin/arm64 archive from clean provenance `5f4955e1499ee8896be5c825ef0fb9b3a52abb70`. Phase 14 remains blocked only for official evaluation, assistant-use, and release-candidate scope. Consult the status ledger for the exact boundary.

The current Phase 07 chi/RHF checkpoint freezes 32 calibration questions under
the owner-adopted, independent ChatGPT/Grok source-review protocol. A
provider-free replay retains `1024/int8` dense as the calibration baseline,
keeps FTS separate, and rejects the tested `1:1` and `FTS1:dense2` RRF arms.
The labels are explicitly `NO_INDEPENDENT_HUMAN_REVIEW`; a separate unexposed
confirmation set is still required for promotion.

## Product boundary

- Local indexing and FTS search never require an API key or network access.
- Paid document embedding and paid hybrid-query embedding are separate explicit operations.
- The stable MCP surface is limited to `status`, `search`, `read_span`, and `reindex`.
- Production defaults to 1024-dimensional cidx-owned `int8`, can rematerialize an explicit 512/int8 target from its preserved 1024-f32 document source bank without another provider call, and contains no Binary/256 executable path. Their historical reports remain document evidence only.
- The product document source-1024 f32 bank is isolated from runtime serving;
  evaluation state is separate and contains no serving vector authority.

## Local Phase 14 surface

The local darwin/arm64 package builder and offline verifier are documented in
[install](docs/install.md). Codex project-scoped MCP setup, optional hook
composition, and fail-closed upgrade guidance live in
[hosts](docs/hosts.md), [hooks](docs/hooks.md), and [upgrade](docs/upgrade.md).
The owner selected Apache-2.0 and its unmodified canonical text is in
[`LICENSE`](LICENSE). The current checkpoint covers only the locally verified
darwin/arm64 archive, both supported int8 dimensions, source-bank-free
four-tool serving, and a project-scoped Codex configuration read; other
platforms and hosts, code signing, and notarization remain unsupported or
unverified. It does not establish `release_candidate` status.

## Start here

1. Read [repository instructions](AGENTS.md).
2. Read the [implementation execution guide](docs/implementation/EXECUTION-GUIDE.md).
3. Check the [persistent phase status](docs/implementation/STATUS.md).
4. Open the active phase from the [implementation plan index](docs/implementation/README.md).

The canonical product design is [Revision 4](local-code-search-mcp-v1-design-r4.md), and evaluation behavior is governed by the [evaluation and promotion contract](docs/implementation/EVALUATION-CONTRACT.md).

Do not add a Go module path, license, remote, paid API call, or evaluation corpus by inference. Those actions require their documented phase decision or explicit user direction.
