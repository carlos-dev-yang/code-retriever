# Local installation and verification

Phase 14 currently supports only a locally built and locally verified
`darwin/arm64` archive. Other operating systems, architectures, signing, and
notarization are unverified and unsupported.

Before an archive can be redistributed, the repository owner must add a
project `LICENSE`. The build script refuses to create an archive without it;
the third-party notices do not choose a license for cidx itself.

After that owner decision, build from a clean committed checkout:

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
lab database, model, or assistant invocation. Set `CIDX_EVIDENCE_DIR` to retain
the local MCP and failure stdout/stderr transcripts instead of only printing the
result.

To use a verified binary manually, place it on `PATH` or keep its absolute
path, then run inside a Git worktree:

```sh
cidx init --serving-dim 256
cidx index --reason manual
cidx status --json
```

`init`, `index`, `status`, FTS search, and `serve` are free local operations.
They never require `VOYAGE_API_KEY`. `cidx version --json` reports the binary
provenance, target, SQLite/grammar implementation IDs, schema IDs, link policy,
and a fresh disposable FTS5/WAL/Go/TypeScript/TSX runtime probe; it intentionally
has no build timestamp.

Do not treat a successful local build as a release candidate. Assistant-use
evidence, a user-approved corpus and labels, a Phase 12 `core_retrieval`
result, and separately approved paid hybrid-query work remain required.
