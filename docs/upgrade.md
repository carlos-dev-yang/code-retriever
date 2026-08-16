# Upgrade and compatibility

Replacing the `cidx` executable and changing a repository's `.cidx/db/index.db`
are separate operations. Stop MCP hosts before replacing a binary or running
any migration/reconciliation work; do not migrate while `serve` is accepting
requests.

Start with `cidx version --json` and `cidx status --json`. A newer production
schema, unknown configuration, unsafe runtime path, or incompatible profile fails
closed with an error. cidx never deletes or reinitializes a database to make a
new binary start. Back up the repository-local `.cidx` directory before a
planned migration or reindex if you need rollback options.

If a new binary requires local reindexing, run it deliberately after reviewing
the reported profile/schema change. A raw lab database is not a production
upgrade prerequisite and production `serve` does not open it. Downgrade
compatibility is not inferred: retain the prior binary or restore an explicit
backup when the older binary rejects a newer schema.
