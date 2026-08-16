# Provider-free lexical smoke inputs

These portable manifests name the two public repositories explicitly selected
and authorized by the user. They contain no checkout path or source body.

Local checkout bindings live only in the controlling cidx project's ignored
`.cidx/test/corpora.local.json`. Binding values are project-relative paths
below `.cidx/test/corpora/`; they are never persisted in a database or tracked
manifest. The verifier checks origin, pinned commit, clean status, root tree,
the manifest's selected-content SHA-256, root license evidence, language
slices, and production-index file parity before it opens the lexical runner.

Development/evaluation state is separate from those disposable checkouts and
lives below `.cidx/test/states/<corpus>/`. The same application, index, search,
and SQLite implementations used by normal projects receive the source root and
state root explicitly; only the workspace resolver and artifact location differ.

`go-chi-chi-v5.3.1` selects all 78 Go files (`<= 58,795` bytes each).
`react-hook-form-v7.85.0` selects the 237 TypeScript/TSX files below `src`
(`<= 142,806` bytes each); its full local checkout has an ignored
`.cidxignore` that excludes the exact outside-`src` TS/TSX roots `app`,
`e2e`, `examples`, `scripts`, and `playwright.config.ts` so the production
index has identical source coverage.

The `lexical-*-draft.json` files are the original 12-case exact-identifier
smoke reference. The `behavior-*-draft-v1.json` files bind the current
behavior-oriented calibration working set to the corrected generation-3
inventories: 12 Go cases for chi and 12 TypeScript plus 8 TSX cases for RHF.
Every behavior case records a deterministic digest, exact file-content hash,
qualified symbol, byte range, provisional direct/support grade, one `task:*`
cohort, and one `signal:*` cohort.

All four datasets are deliberately `review.state=draft`. The behavior set is
source-checked and side-panel-advised, but that does not count as either formal
human label pass. These files are not frozen, official, promotion-capable, or
suitable for confirmation tuning. They also do not authorize a paid document
or query embedding operation.
