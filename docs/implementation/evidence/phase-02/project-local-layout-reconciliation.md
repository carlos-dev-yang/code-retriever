# Phase 02 Project-Local Source/State Reconciliation

- State: accepted historical layout evidence; source/lab file ownership superseded on 2026-08-17
- Date: 2026-08-16
- Owner: `/root`
- Canonical authority: [`local-code-search-mcp-v1-design-r4.md`](../../../../local-code-search-mcp-v1-design-r4.md)
- Trigger: user-directed separation of normal project operation from cidx development/evaluation operation

The relative source/state-root and portable-identity proof remains accepted.
Its combined `<state_root>/raw/embeddings.db` layout is historical: current
implementation must migrate compatible document f32 to
`<state_root>/db/embeddings.db` and keep vector-free evaluation metadata at
`<state_root>/lab/evaluation.db`. The paths and hashes below are preserved as
the exact pre-split migration input, not a current storage contract.

## Historical accepted storage contract

Normal operation resolves one target Git worktree as `source_root` and fixes
`state_root=<source_root>/.cidx`. Its production authority is
`<state_root>/db/index.db`.

cidx development/evaluation resolves two explicit controlling-project-relative
inputs:

```text
source_root=.cidx/test/corpora/<name>
state_root=.cidx/test/states/<name>
```

The same application, index, search, status, production-store, and vector
implementations consume both layouts. Only workspace resolution and state
ownership differ. Development raw vectors use `<state_root>/raw/embeddings.db`
and artifacts use `<state_root>/evaluations/`. Normal CLI/MCP does not accept an
evaluation state override.

Production schema v4 and lab schema v6 remove `canonical_root`. Absolute
source/state paths are runtime-only safety inputs and are not compatibility
identity. Git commit/content, active manifest, profiles, serving key, and
canonical-input hashes remain the portable authority.

## Implementation boundary

- `internal/workspace` validates paired relative source/state roots and confines
  them below `.cidx/test/corpora/<name>` and `.cidx/test/states/<name>` without
  symlink components.
- `internal/app` assembles ordinary and explicit workspaces through the same
  services and carries both roots in memory.
- `internal/store` uses `<state_root>/db/index.db`, migrates a closed legacy
  `.cidx/index.db`, and atomically migrates production v3 to v4 without losing
  generations, FTS, chunks, segments, vectors, or run history.
- `internal/lab` uses `<state_root>/raw/embeddings.db` and migrates v5 to v6
  without losing canonical inputs, raw document vectors, capture runs,
  materializations, variants, or evaluation provenance.
- Index and embedding locks are state-owned. Source enumeration, live-body
  reads, Git status, and corpus verification remain source-owned.
- The ignored local binding is `.cidx/test/corpora.local.json`; its values must
  be relative and below `.cidx/test/corpora/`.
- The unstable `cidx dev workspace`, `embeddings`, and `retrieval evaluate`
  commands accept `--source-dir` and `--state-dir` together. Public commands
  retain the single-project `.cidx` contract.

No ranking, parser, chunker, segmenter, codec, embedding-provider, MCP schema,
or evaluation metric algorithm changed. Existing tests were updated only for
the path/schema contract; no new test code was added.

## Preserved real-corpus state

The already approved chi and react-hook-form local assets were moved from the
legacy clone-owned layout into ignored named state. No corpus was fetched or
updated and no embedding provider was invoked.

| Corpus | Production identity after migration | Indexed inventory | Active vectors | Raw bank |
| --- | --- | --- | --- | --- |
| chi | generation 3; manifest `6bd4db89ee1a9cba70f69e125a803d147dbc0d92c95ef59b44be2dcb54302a29` | 78 files; 452 chunks; 621 segments | 619 ready; 0 pending; 0 failed; coverage 621/621 | 619 inputs; 619 raw vectors; 1 capture; 1 materialization; 619 variants |
| react-hook-form | generation 3; manifest `54f6b1387ae989b1e49bdf21d3ed96189e76fb5b61b74ca282a2617c57f88b8a` | 237 files; 322 chunks; 492 segments | 492 ready; 0 pending; 0 failed; coverage 492/492 | 492 inputs; 492 raw vectors; 1 capture; 1 materialization; 492 variants |

All four migrated SQLite files passed `PRAGMA integrity_check`. After the
schema migrations, a closed-database `VACUUM` removed the obsolete absolute
checkout strings from free pages as well as the logical schema. Post-compaction
SHA-256 values are:

| Relative database | SHA-256 |
| --- | --- |
| `.cidx/test/states/chi/db/index.db` | `92c11610c9cec709afa3c1671900ff39fb8c24a9c89fd00635d29d06d56997aa` |
| `.cidx/test/states/chi/raw/embeddings.db` | `84d1b0aa2430a91ec301fb65a84f27be45ad67d0ffddcf1729aaf7fd8a339b85` |
| `.cidx/test/states/react-hook-form/db/index.db` | `49a1a9026cc89be4a47c5f7c1b3fc28437cc1b95580a20c83cfc9434eb20c2ef` |
| `.cidx/test/states/react-hook-form/raw/embeddings.db` | `53f260ff816b2901b72c849427f35297a3bc6cec184c6329b983f8c5b328a1f5` |

Byte-string inspection after compaction found neither the former machine path
nor `canonical_root` in those databases.

## Real entry-point proof

Provider-free `cidx dev workspace status` reopened both named workspaces and
reported the identities/counts above. Provider-free retrieval planning through
the same explicit source/state roots then reproduced:

| Corpus | Query count | Estimated query tokens | Raw document inputs | API call |
| --- | ---: | ---: | ---: | --- |
| chi | 12 | 1,307 | 619 | none |
| react-hook-form | 20 | 2,391 | 492 | none |

A separate disposable normal project was initialized and indexed at
`.cidx/db/index.db`, copied intact to a different filesystem path, and reopened
successfully. The moved project retained the same applied profiles, three
files/chunks/segments, and 0/3 vector coverage; integrity was `ok`, schema was
v4, and `meta` contained no `canonical_root` column. This proves relocation is
not accepted merely for the evaluation harness: the normal project entry point
also has no persisted machine-path binding.

## Checks

Completed before the final boundary:

- focused `go test -count=1` for app, store, lab, devlab, eval, and index;
- real chi/RHF workspace status, raw materialization plan, and retrieval plan;
- real disposable normal-project initialization, index, relocation, and reopen;
- SQLite schema, counts, integrity, compaction, and stale-path scans;
- `bash -n scripts/verify-local-release.sh`;
- `git diff --check`.

Final boundary checks passed:

- `go test -count=1 ./...`;
- `go test -race -count=1` for app, store, lab, devlab, eval, and index;
- `go vet ./...`;
- `go build ./...`;
- `go mod tidy -diff`;
- repository-wide `gofmt -l` cleanliness;
- package/verifier script syntax;
- production app/search dependency scan excluding `internal/lab`;
- final `git diff --check`.

The compacted chi/RHF databases were reopened once more after the boundary and
reported the same generation, manifest, inventory, ready counts, zero stale or
index errors, and complete coverage. No provider, API key, paid query, corpus
network, or assistant operation occurred.
