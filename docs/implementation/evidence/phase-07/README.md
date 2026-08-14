# Phase 07 Lexical Evaluation Infrastructure Evidence

- Phase: `07-lexical-evaluation`
- State: `blocked` — reusable offline infrastructure is implemented; official evidence requires user-selected, tracked corpus manifests and user-provided ignored local bindings.
- Date: 2026-08-15

## Implemented infrastructure

- `internal/eval` owns strict portable corpus-manifest and evaluation-dataset JSON loading, RFC 8785 canonical fingerprints, and validation of corpus provenance, complete commits, SPDX-form license declarations, language slices, repository-relative roots, selection patterns, and deterministic selected-content hashes.
- Local bindings are intentionally separate from portable manifests. The conventional binding loader accepts only the ignored `.cidx/lab/corpora.local.json`; an explicit checkout path takes precedence. Checkout verification is local and read-only: canonical worktree root, origin, exact commit, clean status, root license record, symlink-free selected files, declared language slices, and selected-content hash.
- Metrics reuse `internal/evalcontract.EvaluationCase` truth values. They implement OR alternatives within a requirement group, AND across groups, graded relevance/NDCG without duplicate gain, Hit/Recall/MRR, requirement coverage, complete requirement hit, known-hard-negative observations, returned counts, explicit answerable/required/hard-negative denominators, operation-failure retention, first-loss counts, and Go/TypeScript/TSX/mixed summaries.
- The lexical runner is an adapter over the production lexical search interface; it has no alternate FTS implementation. Its injected production truth inventory uses one narrow read-only store transaction to pin metadata and enumerate authoritative source chunks. It rejects stale/missing required alternatives, relevance judgments, and hard negatives before metrics exist. Successful searches must match that pin; a drift yields `NON_REPRODUCIBLE_RUN`. Required search failures stay in the denominator as typed FTS-candidate failures without serializing raw errors.
- Every ranked case carries a valid shared `StageTrace`: source/parser truth is present after preflight, FTS presence follows results, operation failures terminate at `fts_candidate`, later unrequested stages are explicit `NOT_OBSERVED`, and abstainable cases retain empty groups.
- Immutable artifacts publish `run.json` and `summary.md` through a new temporary directory plus atomic rename, then write `artifact-manifest.json` as the completion marker. They reject existing run IDs, forged summaries, missing/duplicate query results, invalid traces, invalid portable hits/ranks, unsafe portable data, and inconsistent corpus/dataset/generation pins.
- Manifest include/exclude rules use a validated whole-segment `**` matcher, including zero-segment matching for patterns such as `**/*.go`; excludes always take precedence.

## Checks actually run

Codex repeated the focused boundary validation on 2026-08-15 before the
infrastructure commit. All commands below passed.

```text
gofmt -w internal/eval
go test -count=1 ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store
go test -count=1 -race ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store
go vet ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store
go build ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store
gofmt -l internal/eval internal/evalcontract internal/search/lexical internal/store
git diff --check
```

The focused tests cover portable/duplicate-field manifest rejection, local Git binding verification including wrong-content-hash and dirty-worktree rejection, whole-segment glob selection/exclusion safety (including nested `**` and empty excludes), OR/AND multi-span requirements, duplicate-parent max-grade NDCG, wrong indexed-hash rejection, exact hard-negative denominators across answerable and abstainable cases, full language/cohort summaries, retained operation failures and valid abstainable traces, production truth snapshots, inventory preflight, generation drift, caller cancellation, and atomic immutable artifact publication including forged-summary, missing metric, and rank rejection.

## Checks not run and blocker

- No corpus was selected, downloaded, cloned, bound as an official corpus, indexed, copied, or embedded. No provider, API key, paid operation, network request, CLI, MCP, or public contract change ran.
- No official run, corpus verification record, lexical baseline, labels, comparison baseline, or repeatability evidence exists because the user has not selected and supplied the corpus manifests and local bindings.
- A deterministic simple-search baseline is intentionally pending: Phase 07 does not yet define a corpus-independent baseline ranking policy, so this infrastructure does not invent one.
- The direct `internal/eval` source has no `embedclient`, `lab`, or provider import. A full Go dependency listing still reaches `embedclient` indirectly through the existing shared `config`/`evalcontract` package graph; that pre-existing shared-contract coupling must be split before a strict transitive dependency-boundary claim can be made.

## Next action

After the user selects the open-source corpora and provides reviewed tracked manifests plus ignored local bindings, verify each existing checkout without modification, create the reviewed dataset, and run the free lexical baseline.
