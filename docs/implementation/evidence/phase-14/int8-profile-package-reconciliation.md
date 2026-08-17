# Phase 14 Current Int8 Package Reconciliation

- Status: accepted local darwin/arm64 package and operational checkpoint;
  Phase 14 remains `blocked` for official evaluation, assistant-use, and
  `release_candidate` evidence
- Date: 2026-08-17
- Clean source commit: `5f4955e1499ee8896be5c825ef0fb9b3a52abb70`
- Product contract: default 1024/int8, explicit compact 512/int8, durable
  document source-1024 f32, and no Binary/256 executable product path
- Historical predecessor: [Revision 4 local checkpoint](revision-4.md)

## Boundary exercised

The current package verifier exercised one clean-provenance darwin/arm64
archive from checksum validation through the installed binary. It created a
synthetic Go/TypeScript/TSX repository, initialized the default profile, and
proved that the generated configuration is 1024/int8. Free init/index did not
create the document source bank.

The verifier then inserted three deterministic, integrity-valid source-1024
f32 fixture rows for the repository's real source profile and canonical input
hashes. This is a local orchestration fixture, not provider or retrieval-quality
evidence. From those same rows, the installed binary materialized complete
1024/int8 coverage, changed the project configuration to 512, reconciled the
index, and materialized complete 512/int8 coverage without a provider request.

| Installed-binary observation | 1024/int8 | 512/int8 |
| --- | ---: | ---: |
| source inputs | 3 | 3 |
| Voyage inputs | 0 | 0 |
| requested provider inputs | 0 | 0 |
| succeeded serving vectors | 3 | 3 |
| failed / discarded | 0 / 0 | 0 / 0 |
| actual provider tokens | 0 | 0 |
| ready coverage | 3 / 3 | 3 / 3 |

The final production rows were independently checked as 512-dimensional
`cidx-int8-symmetric-v1`. A relocated copy containing only project config and
production `index.db` served status and the four MCP tools without the source
bank or development state. This proves that the source bank supports document
publication/rematerialization and is not a serving dependency.

## Retired profile proof

- `cidx init --serving-dim 256` failed before creating `.cidx`.
- `cidx init --codec binary` failed before creating `.cidx`.
- The package has no positive Binary or 256 fixture.
- Historical Binary/256 comparisons remain documents only; this checkpoint
  neither imports nor reactivates them.

## Package and transcript identity

All paths below are ignored local evidence and are not committed product state.

| Artifact | Local path | SHA-256 |
| --- | --- | --- |
| archive | `dist/phase14-5f4955e/cidx_dev-5f4955e1499e_darwin_arm64.tar.gz` | `ab870df6d22a62419babb6d5695b3677a7a96e393e48dad03741ddce71ed8724` |
| checksum manifest | `dist/phase14-5f4955e/checksums.txt` | `f794f7d63c99e3b6ffe0cc072887049a33751af37602b52028b8ee1af87863d5` |
| package log | `dist/phase14-5f4955e/package.log` | `024a1922e0be0492f8399d7a9a4d48051a811e89e237ff72a2aca8eb5281bce8` |
| verifier log | `dist/phase14-5f4955e/verification.log` | `af7f4f08a2122406c07e1aa2dcb94efbe5f0554279fe44e063cca73f3f3082b8` |
| direct MCP transcript | `.cidx/test/evidence/phase14-5f4955e/mcp.jsonl` | `5511ca12c74cdac3cc0259e1c7d53d489813727ab0e8a476ddc30d19e7513de7` |
| Codex config/read transcript | `.cidx/test/evidence/phase14-5f4955e/codex-app-server.jsonl` | `338c4d90ec83f552c6db80d60705cee9a7056025c240d8543672d23d8175741b` |

The embedded version report records `source_modified=false`, darwin/arm64,
CGO enabled, `modernc.org/sqlite` v1.47.0, FTS5 and WAL available, registered
Go/TypeScript/TSX grammars, and production schema range 1 through 5. The
archive verifier also accepted deliberate-corruption rejection, neutral archive
ownership/modes/timestamps, portable diagnostic paths, exact license inputs,
runtime/manifest agreement, newer-schema rejection, source-root relocation,
stdout protocol purity, and isolated project-scoped Codex configuration read.

## Checks actually run

- `scripts/package-local.sh` from a clean independent local clone at the exact
  source commit; no network action
- `scripts/verify-local-release.sh` with retained transcript directory
- archive checksum success plus deliberate corrupt-archive rejection
- installed-binary default 1024/int8 and compact 512/int8 source rematerialization
- installed-binary Binary/256 negative CLI checks
- installed-binary Go/TypeScript/TSX FTS and exactly-four-tool MCP smoke
- installed-binary source-bank-free relocated status/MCP smoke
- isolated Codex app-server project configuration read; no model, assistant,
  provider, corpus, or paid invocation

A linked Git worktree was not used as the provenance source because its Git
indirection did not expose complete Go VCS metadata and the package script
correctly rejected `source_modified=unknown`. The accepted archive instead came
from the clean independent local clone above.

## Remaining gates

This checkpoint supports only local darwin/arm64. It does not establish another
platform or host, signing/notarization, official Phase 12 `core_retrieval`,
paired assistant usefulness, or `scope=release_candidate`. Phase 14 therefore
remains blocked rather than done. The next product work is the Phase 07
owner-adopted dual-AI label freeze and immutable replay, with permanent
`NO_INDEPENDENT_HUMAN_REVIEW`, followed by official Phase 12 and assistant/host
promotion evidence when their other inputs are available.
