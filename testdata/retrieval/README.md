# Provider-free lexical smoke inputs

## Question-set versioning

Question sets and run results are append-only. Keep every earlier file, create
a new `version` when query text, membership, ground truth, required groups, or
cohort assignments change, and record all source versions in `supersedes` plus
a concise `change_summary`. Every version binds a tracked taxonomy version and
SHA-256. Every execution uses a new run ID whose manifest records the exact
question-set ID, version, canonical SHA-256, taxonomy version, and taxonomy
SHA-256.

The current taxonomy is
`cohort-taxonomy-critical-general-v1.json`. The current existing-repository
revisions are `question-set-go-chi-v5.3.1-critical-general-v2.json` and
`question-set-react-hook-form-v7.85.0-critical-general-v2.json`. They combine
the preserved behavior and exact-identifier source versions into new draft
question sets; they do not overwrite or reinterpret prior run artifacts.
`question-set-run-registry-v1.json` records which immutable run used each
question-set/taxonomy version and the exact artifact digest.

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
smoke reference. The current behavior-oriented calibration working set is
`behavior-go-chi-v5.3.1-draft-v3.json` plus
`behavior-react-hook-form-v7.85.0-draft-v2.json`, bound to the corrected
generation-3 inventories: 12 Go cases for chi and 12 TypeScript plus 8 TSX
cases for RHF. Version 2 records the accepted source-backed corrections: chi
G09 adds `walkXFF` as a grade-0 hard negative, while RHF T10 makes `PathImpl`
and `PathInternal` grade-2 direct requirements and `Path` grade-1 support.
Chi version 3 changes only G12's query wording and direct rationale so it no
longer implies deterministic iteration order across distinct Go map keys.
Earlier versions remain historical exploratory-run inputs.
Every behavior case records a deterministic digest, exact file-content hash,
qualified symbol, byte range, provisional direct/support grade, one `task:*`
cohort, and one `signal:*` cohort.

The two current frozen calibration datasets are
`behavior-go-chi-v5.3.1-calibration-frozen-v1.json` and
`behavior-react-hook-form-v7.85.0-calibration-frozen-v1.json`. They contain 12
Go, 12 TypeScript, and 8 TSX cases, respectively, and are bound to
`owner-adopted-dual-ai-v1`, `OWNER_ADOPTED_DUAL_AI_REVIEW`, and
`NO_INDEPENDENT_HUMAN_REVIEW`. Their whole-digest adoption record is
`reviews/owner-adoption-chi-rhf-calibration-v1.json`. These 32 cases are frozen
calibration evidence only: they are not human-reviewed, confirmation, or
promotion-capable and must not be edited after the exposed replay.

`relation-packaging-experiment-contract-v1.json` is the frozen packaging
experiment grid. `relation-sibling-packaging-adopted-v1.json` is the
evaluation-only adopted cell (sibling count 4 / 4096 bytes). Neither file
authorizes a production MCP change, a graph product path, or confirmation.

The seven earlier datasets remain deliberately `review.state=draft` as
historical smoke and authoring inputs. Neither the draft nor frozen files
authorize a document or query embedding operation.
