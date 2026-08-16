# Phase 07 Lexical Evaluation Evidence

- Phase: `07-lexical-evaluation`
- State: `blocked` — provider-free generation-3 audit, exact 32-case draft binding, clean FTS controls, and official `voyage-code-4` pricing verification are complete; document capture waits for explicit bounded approval, and official evidence still requires later human label freeze/two review passes plus a deterministic simple-search baseline policy.
- Date: 2026-08-16

Current real-data audit: [chi/RHF structural audit — Revision 4](chi-rhf-structural-audit-r4.md).

Current cohort-authoring worksheet: [chi/RHF behavior-cohort working set — Revision 4](chi-rhf-working-cohort-r4.md).

Current document-capture gate: [chi/RHF document-capture approval packet — Revision 4](chi-rhf-document-capture-approval-r4.md).

The accepted Phase 04 correction is now reflected in provider-free
generation-3 indexes: chi has 452 parents/621 segments and RHF has 322
parents/492 segments; 57 RHF production anonymous default-export functions now
have deterministic path-derived retrieval labels, the three observed overload
sets each collapse to one parent, and source-span mismatches are zero. New
source-body-free inventory hashes and the current no-network document plan are
recorded in the structural audit. The side-panel-reviewed 32-case cohort is now
bound to exact generation-3 identities and deterministic case digests. The
official live Voyage pages confirm `voyage-code-4`, its supported dimensions,
$0.12 per million-token price, and 200-million-token free allowance. Work now
waits only at the separate bounded document-capture approval gate. No provider
or API key has been used.

### 2026-08-16 behavior-binding boundary checks

The two behavior datasets contain 32 cases with 32 matching RFC 8785-framed
case digests. A read-only SQL join proved that every unique direct/support span
resolves to exactly one generation-3 production parent. The focused boundary
validation then passed:

```text
gofmt -w internal/devlab/lexical.go internal/devlab/lexical_test.go
go test -count=1 ./internal/devlab ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store
go test -count=1 -race ./internal/devlab ./internal/eval ./internal/evalcontract
go vet ./internal/devlab ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store
go build ./internal/devlab ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store
jq -e . testdata/retrieval/behavior-*-draft-v1.json
go mod tidy -diff
git diff --check
```

No repository-wide test, corpus mutation after generation 3, provider call,
API-key read, query embedding, materialization, label freeze, or promotion run
occurred.

### 2026-08-16 clean provider-free behavior baseline

The clean build at commit `2a08df7d465f72c939a3b0e85b28d8b5ca000cdf`
ran the production FTS path once against each new behavior dataset. Both runs
completed without operation failures, but natural-language behavior questions
returned no candidates: chi was 0/12 at Hit@5 and RHF was 0/20 at Hit@5, with
every first loss recorded as `FTS_CANDIDATE_MISS`. This is the lexical baseline,
not a dense or hybrid result and not a reason to tune an unfrozen simple-search
policy before the separately approved document/query embedding stages.

| Run | Cases | Mean returned | Hit@5 | `run.json` SHA-256 |
| --- | ---: | ---: | ---: | --- |
| `chi-behavior-fts-g3-1` | 12 Go | 0 | 0 | `1134ef35cdcc0cbd7c415feb7c2ab6f849e76dc15cd224b133f1554dfe1a928d` |
| `rhf-behavior-fts-g3-1` | 12 TypeScript + 8 TSX | 0 | 0 | `31ba8d90986b838fe6a95e9f514795c57b4d11bc04c5e56ca2fb8623f8acac84` |

The ignored immutable artifacts remain under each checkout's
`.cidx/lab/evaluations/runs/` directory. No provider, API key, query embedding,
or label freeze was involved.

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

## Historical pre-resume checks not run and blocker

- No corpus beyond the two explicitly authorized public checkouts was selected, downloaded, cloned, copied, or embedded. No provider, API key, paid operation, or public MCP contract change ran.
- No official frozen run, human-reviewed labels, simple-search comparison baseline, or promotion evidence exists. The recorded corpus verification and repeatability evidence is draft-smoke-only.
- A deterministic simple-search baseline is intentionally pending: Phase 07 does not yet define a corpus-independent baseline ranking policy, so this infrastructure does not invent one.
- The direct `internal/eval` source has no `embedclient`, `lab`, or provider import. A full Go dependency listing still reaches `embedclient` indirectly through the existing shared `config`/`evalcontract` package graph; that pre-existing shared-contract coupling must be split before a strict transitive dependency-boundary claim can be made.

## Next action

Have a human reviewer replace or approve the two tracked machine-draft smoke
datasets in two recorded passes, then freeze a deterministic simple-search
baseline policy before comparing it with FTS. Any future hard-negative or
no-answer label needs corpus-wide search evidence and a second review/pass.

## Authorized corpus-resume smoke

The user explicitly authorized acquisition of exactly these public checkouts;
no other corpus was selected or acquired:

- `go-chi/chi` v5.3.1 at `8b258c7bb28f97a5f2a856ff7ef962578fec9215`,
  MIT, root tree `7ccb2269b57183ac3a741f269c0da31fd03ad035`;
- `react-hook-form/react-hook-form` v7.85.0 at
  `371432c39271aab739358d19c406793771565ab3`, MIT, root tree
  `688906c5842a0d71051154343e993adb525e688f`.

Tracked portable manifests are
`testdata/retrieval/corpora/go-chi-chi-v5.3.1.json`
(`18cd5cf433ee0af47a212e6111dcd1d65f6104baa28bb528b2ec93d9afec36b9`)
and `testdata/retrieval/corpora/react-hook-form-v7.85.0.json`
(`e94f0861e6ac0c864524a23edc4bcb0ddc69a3848ef0f9c962f0b675bfde81a8`).
Their verifier-selected content hashes are respectively
`892e79de9e8c522fe3ccf6b0731a3798d0f2c67a18f1b4162685c4843245af5d`
for 78 Go files (largest 58,795 bytes) and
`717caa8346fd5a0b1a7ca69df63bf1ac8477f7c8770f1e67fa7b1fad58df132b`
for 237 `src` TypeScript/TSX files (largest 142,806 bytes).

The React Hook Form checkout is full and clean. Its checkout-local ignored
`.cidxignore` excludes the only outside-`src` TS/TSX roots at this commit:
`app`, `e2e`, `examples`, `scripts`, and `playwright.config.ts`; `.cidx/`
and `.cidxignore` are only in that checkout's `.git/info/exclude`. Each
checkout has an ignored local `.cidx/lab/corpora.local.json` self-binding.

Free local initialization/index/status records:

- chi generation 1: 78 files, 452 chunks, 621 segments, manifest
  `6bd4db89ee1a9cba70f69e125a803d147dbc0d92c95ef59b44be2dcb54302a29`;
- react-hook-form generation 1: 237 files, 275 chunks, 416 segments, manifest
  `54f6b1387ae989b1e49bdf21d3ed96189e76fb5b61b74ca282a2617c57f88b8a`.

`cidx dev retrieval evaluate --mode lexical --inventory-only` wrote ignored,
source-body-free inventory packets with SHA-256
`c6a8661e9cde7ff269d69311593411930def0f69a7c1816b9e919bfbce7cadab`
(chi) and
`d65595cfa76480278f4e71734d6a4802b8bb4129944a3384c2fccedb3e80781d`
(react-hook-form). The mode opens `app.OpenLocal` and the production store
only; it neither opens `lab.Store`, reads `VOYAGE_API_KEY`, contacts a
provider/network, nor permits `--apply`.

The tracked datasets
`testdata/retrieval/lexical-go-chi-v5.3.1-draft.json`
(`e03894820a25eecb5527049ece7d10da39da58f719c629fa3e5bc11f47ca22c4`)
and `testdata/retrieval/lexical-react-hook-form-v7.85.0-draft.json`
(`393a7562bffbe5ce3fb018b9438aaf6633c7f5c26ba4418521b74e3c3ae6df80`)
contain six cases each. Every case is explicitly `review.state=draft` with
`machine-draft/user-review-pending`; neither file is frozen, official,
promotion-capable, or usable for tuning. Every draft case digest is the
SHA-256 of RFC 8785 canonical `EvaluationCase` JSON with its `digest` field
empty; `internal/devlab.DraftCaseDigest` verifies this preparation framing
before lexical execution.

Ranking-blind review packets retain corpus/dataset digests, query text,
class, answerability/cardinality, proposed file/symbol/span/hash and
alternatives, ambiguity/rationale, accept/reject/revise/adjudicate actions,
and reviewer/timestamp/independent-source-verification fields. Their packet
digests are `d1fcc6415c6c98a73ebd95b8367fefb804830b75da8b44b45f6f5cc66e1417fc`
(chi) and `e6ab27964f0e90f52e496948723eeb518e88911e73273b23233dfa773a3f6f6a`
(react-hook-form). Both declare `dataset_status=DRAFT`,
`dataset_role=CALIBRATION_SMOKE`,
`label_authority=MACHINE_PREPARED_UNREVIEWED`,
`human_review_status=PENDING`, `run_authority=EXECUTION_ONLY`, and
`evidence_class=PIPELINE_AND_REPLAY_DIAGNOSTIC`; they also declare
`promotion_eligible=false`, `confirmation_eligible=false`,
`retrieval_arm=PROVIDER_FREE_LEXICAL_ONLY`, and `paid_provider_calls=0`.

The reviewed implementation was committed as
`28d0d6a1d93949c2151ca388a8f4b7739c7edc81`, then rebuilt with
`-trimpath -buildvcs=true` from a clean worktree. `cidx version --json` and
`go version -m` both reported that full revision and
`source_modified=false`. Reindex reused all 78 chi and 237 react-hook-form
files without changing either generation or manifest.

The clean binary published immutable ignored smoke pairs
`chi-draft-smoke-28d0d6a-{1,2}` and
`rhf-draft-smoke-28d0d6a-{1,2}` through `eval.WriteRunArtifact`; every run
records the exact implementation revision above. The first `run.json` digest
for each corpus is respectively
`284c9d07c8bd7bb40a7449a6118398db99a58976a370406b5a589daa742fa723`
and `003c13c4d0a7e9dbc46abc096e4783244f8b01b97bd4e9c7177451d17ee2dc30`.
The second-run digests are
`bdd9c8988c6e806cd0f0151bd5fa0309b5a2e58d2668c136a462ff7b0d32daee`
and `61ab6cb04c681df3d62b3d28caeed15a56fc5efa8f32b809e87b26600a2c1aac`;
their clock/run identities differ, while the framed results and summaries are
byte-identical. Earlier dirty-worktree runs are superseded execution
diagnostics and are not clean-provenance evidence.

Lexical artifact execution fails closed unless build metadata has a canonical
full lowercase hex VCS revision (40-character SHA-1 or 64-character SHA-256)
and `source_modified=false`. It also rejects any non-`draft` review state, so
this smoke command cannot stamp frozen labels as draft authority.
Inventory-only preparation remains available. A future official baseline is a
separate follow-up after human review and baseline-policy freeze.

Repeated runs produced identical diagnostic replay values using exactly
`sha256(jq -c '{results,summary}' output including LF)`:
`7b31f9cd6eac758601d99988da5c691ec462ece96b82ceb5efc0eff163242937`
(chi) and
`e869a7e0718d693958d5b34878c0604d2ace6d683cf272ee9d10ab22cabf9c3b`
(react-hook-form). The smoke denominators are 12 total: 6 Go, 4 TypeScript,
2 TSX, 0 hard-negative, and 0 confirmation. Hard-negative and confirmation
metrics are `NOT_OBSERVED`, never reported as zero. Observed metrics mean only
agreement with unreviewed draft labels under a lexical smoke configuration.
The valid conclusion is limited to provider-free lexical-path execution and
identical ranking replay for exact draft inputs; this is not a quality claim or
promotion evidence.

## Resume checks actually run

```text
go build -o /tmp/cidx-phase07 ./cmd/cidx
cidx init --serving-dim 256 --codec binary              # each authorized checkout
cidx index --root <checkout> --reason manual
cidx status --root <checkout> --json
cidx dev retrieval evaluate --mode lexical --inventory-only ...
cidx dev retrieval evaluate --mode lexical --run-id <fresh-id> ... # twice per corpus
gofmt -w internal/devlab/lexical.go internal/devlab/lexical_test.go
go test -count=1 ./internal/devlab ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store
go test -count=1 -race ./internal/devlab ./internal/eval ./internal/evalcontract
go vet ./internal/devlab ./internal/eval ./internal/evalcontract
go build -o /tmp/cidx-phase07-provenance ./cmd/cidx
git diff --check
```

All listed checks passed. No paid document/query embedding, lab database,
provider call, API-key read, promotion, simple-search comparison, or official
evaluation run occurred.

An independent Terra/high review found and rechecked three resume-boundary
defects: dirty or noncanonical executable provenance, unconditional draft
authority for non-draft inputs, and packet child-symlink escape. The accepted
tree fails closed on all three, and the final re-review reported no findings.
Codex then ran the single final commit-boundary validation:

```text
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 ./internal/devlab ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store ./internal/app ./internal/index
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 -race ./internal/devlab ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store
env -u VOYAGE_API_KEY GOPROXY=off go vet ./internal/devlab ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store
env -u VOYAGE_API_KEY GOPROXY=off go build -trimpath -buildvcs=true -o /tmp/cidx-phase07-boundary ./cmd/cidx
go mod tidy -diff
gofmt -l internal/devlab/cli.go internal/devlab/lexical.go internal/devlab/lexical_test.go
jq -e . testdata/retrieval/corpora/*.json testdata/retrieval/*draft.json
git diff --check
```

All boundary checks passed. The accepted implementation is committed before
new smoke artifacts are generated so their `code_commit` is truthful.

The clean post-commit checkpoint then ran:

```text
test -z "$(git status --porcelain)"
env -u VOYAGE_API_KEY GOPROXY=off go build -trimpath -buildvcs=true -o /tmp/cidx-phase07-clean ./cmd/cidx
cidx version --json
go version -m /tmp/cidx-phase07-clean
cidx index --root <checkout> --reason manual             # both checkouts
cidx status --root <checkout> --json                     # both checkouts
cidx dev retrieval evaluate --mode lexical --inventory-only ...
cidx dev retrieval evaluate --mode lexical --run-id <fresh-id> ... # two per corpus
jq -c '{results,summary}' <run.json> | shasum -a 256
cmp <framed-run-1> <framed-run-2>
```

The environment omitted `VOYAGE_API_KEY`; no provider or network operation
was needed. The repeat comparisons and all recorded checksum assertions
passed. The independent Terra/high reviewer then matched all four clean run
artifacts, artifact manifests, status counts, replay hashes, and ledger values
and reported no findings.

`internal/devlab/lexical_test.go` covers the draft-digest framing, lexical
mode/apply and inventory flag rejection, clean canonical code-provenance and
draft-only smoke enforcement, conventional artifact-root validation, and
descriptor-bound packet writes. The latter uses Go 1.26 `os.OpenRoot` with
root-relative directory creation, exclusive temporary creation, hard-link
publication, and reads; both `inventory` and `review` external-child-symlink
attempts are rejected without an outside write. It also covers atomic
source-body-free inventory replay/collision handling and review-packet
decoding for authority/floor fields plus `id`, `text`, `language`, and
`answer_mode`. The direct lexical source has no `lab.Open`, `VOYAGE_API_KEY`,
provider, or HTTP dependency.
