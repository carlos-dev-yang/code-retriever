# Local installation and verification

Phase 14 currently supports only a locally built and locally verified
`darwin/arm64` archive. Other operating systems, architectures, signing, and
notarization are unverified and unsupported.

The repository owner selected Apache-2.0 on 2026-08-15; the unmodified
canonical license text is at the repository root in [`LICENSE`](../LICENSE).
No copyright holder or project `NOTICE` was inferred. The third-party notices
remain separate from cidx's own license. The build script still refuses an
archive without that root `LICENSE`.

After the license change is committed, build from that clean committed checkout:

```sh
scripts/package-local.sh
scripts/verify-local-release.sh dist/cidx_<version>_darwin_arm64.tar.gz
```

The verifier requires macOS `sandbox-exec`, `git`, `sqlite3`, `codex`, and the
standard archive/checksum tools. `sqlite3` is used only to corrupt a disposable
fixture into a newer-schema negative case; cidx itself uses bundled modernc
SQLite and never depends on that command. The verifier exits as unverified if
`sandbox-exec` is not available. It blocks network access, verifies the archive
checksum and a corruption rejection, checks executable permission and Mach-O
linkage evidence, then uses a synthetic Git repository whose path contains
spaces and non-ASCII characters. It does not use an API key, provider, corpus,
lab database, model, or assistant invocation. It initializes the default
1024/int8 profile, injects deterministic synthetic document source vectors
only as local verifier fixtures, proves provider-free publication, switches to
512/int8, and proves the same source rows are reused with zero provider
requests. It also runs MCP from a relocated copy without the source-bank file
and proves serving does not create one. Set `CIDX_EVIDENCE_DIR` to retain the
local MCP and failure stdout/stderr transcripts instead of only printing the
result. If verification fails, it copies every transcript produced before the
failure and still exits nonzero; a copied transcript is not a successful
verification claim.

To use a verified binary manually, place it on `PATH` or keep its absolute
path, then run inside a Git worktree:

```sh
cidx init
cidx index --reason manual
cidx status --json
```

This creates the default 1024/int8 profile. To use the compact profile in an
existing repository, explicitly change `embedding.serving_dimensions` in
`.cidx/config.json` from `1024` to `512`, then run:

```sh
cidx index --reason manual
cidx embed --dry-run
cidx embed --apply
```

When `.cidx/db/embeddings.db` already contains all compatible document
source-1024 rows, the plan reports only local source inputs and `--apply`
performs zero Voyage requests. If source rows are missing, the plan reports
those Voyage inputs separately and `--apply` requires the explicitly supplied
`VOYAGE_API_KEY`.

`init`, `index`, `status`, FTS search, and `serve` are free local operations.
They never require `VOYAGE_API_KEY`. `cidx version --json` reports the binary
provenance, target, SQLite/grammar implementation IDs, schema IDs, link policy,
and a fresh disposable FTS5/WAL/Go/TypeScript/TSX runtime probe; it intentionally
has no build timestamp.

Do not treat a successful local build as a release candidate. Assistant-use
evidence, a user-approved corpus and labels, a Phase 12 `core_retrieval`
result, and separately approved paid hybrid-query work remain required.
