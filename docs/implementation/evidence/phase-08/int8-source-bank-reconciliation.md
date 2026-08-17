# Phase 08 Product Source-Bank Reconciliation

- State: accepted
- Owner: `/root`
- Entry commit: `bbf3bb6`
- Decision date: 2026-08-17
- Provider/corpus action: not run

## Result

- Added the product-owned source bank at
  `<state_root>/db/embeddings.db`. Its only vector-bearing table stores
  immutable, validated document-role 1024-f32 rows keyed by source-profile
  fingerprint and canonical-input SHA-256.
- New capture and materialization callers write run, failure, coverage,
  checksum, and artifact metadata to `<state_root>/lab/evaluation.db`. That
  database has no canonical-input byte table, raw f32 table, materialized
  variant table, or other vector blob column.
- Development callers retain a narrow compatibility API, but the implementation
  delegates every source-vector read/write to `internal/sourcebank`. Target
  variants exist only in memory during the single local publication attempt;
  evaluation SQLite stores their deterministic checksum and counts only.
- Compatible rows from the ignored historical
  `<state_root>/raw/embeddings.db` are copied into the product bank through a
  read-only connection after dimension, encoding, profile/input digests,
  checksum, finite f32 values, blob length, and SHA-256 validation. The legacy
  database is not modified or deleted. Historical run/evaluation rows remain
  preserved there and in immutable evidence rather than being rewritten as
  current evidence.
- The source bank deliberately stores no absolute path and no second repository
  identifier. The portable authority is the cryptographic source key itself:
  `(source_profile_fingerprint, canonical_input_sha256)`. This allows moving a
  complete `.cidx` state while preventing reuse for nonidentical canonical
  inputs.
- Production search, MCP, and the shipped CLI entrypoint import neither
  `internal/sourcebank` nor `internal/lab`. A missing source/evaluation DB
  therefore cannot block already materialized serving search.

## Boundary validation

After correcting one duplicate helper declaration exposed by the first compile,
the successful boundary was:

```text
go test -count=1 ./internal/sourcebank ./internal/lab ./internal/devapp
go test -count=1 -race ./internal/sourcebank ./internal/lab ./internal/devapp
go vet ./internal/sourcebank ./internal/lab ./internal/devapp
go build ./...
gofmt -l internal/sourcebank internal/lab internal/devapp/capture_test.go
rg production lab schema for vector-bearing CREATE TABLE statements
rg production search/MCP/CLI imports for internal/lab or internal/sourcebank
git diff --check
```

All successful-boundary commands passed. Existing core fixtures prove f32
round-trip, physical two-database separation, absence of vector tables in the
evaluation DB, immutable source conflict rollback, partial capture resume,
vector-free materialization metadata, and read-only legacy-row copying.

No Voyage call, API-key read, corpus operation, query embedding, or evaluation
run was performed.

## Handoff

Phase 09 receives only validated source records, source coverage/missing keys,
and current active canonical inputs. It must remove Binary/256 executable code,
derive 1024/int8 or 512/int8 exclusively as `prefix -> L2 -> int8`, and publish
the complete selected target atomically to `index.db`.
