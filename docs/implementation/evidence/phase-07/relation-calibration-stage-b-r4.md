# Relation Calibration Stage B — Revision 4

- Date: 2026-08-19
- State: in progress; provider-free corpus, graph, and draft calibration unit frozen
- Owner: `/root`
- Authority: [`RELATION-EVIDENCE-COMPLETION-PLAN.md`](../../RELATION-EVIDENCE-COMPLETION-PLAN.md)
- Promotion status: preparation only; no calibration, confirmation, or promotion result yet

## Selected calibration repositories

The owner authorized the previously proposed three repositories for the new,
unexposed calibration unit. Stable upstream tags were resolved to immutable
commits and cloned only below ignored `.cidx/test/corpora/` state.

| Corpus | Tag / commit | License | Selected source slice | Git tree | Selected-content SHA-256 |
| --- | --- | --- | --- | --- | --- |
| `go-git-go-git-v5.19.1` | `v5.19.1` / `3c3be601aa6c0fd0d536c0d1e4f898b4c60e65fe` | Apache-2.0 | 474 Go files | `5a1a3e1ea1b25aa788b16e731f3421c61f9d02cc` | `6aad05e8967f3753fc3e931b29886423deac9bb3b70db5ac59be8f3c4e39b0a8` |
| `pmndrs-zustand-v5.0.14` | `v5.0.14` / `bfb2a9e7ce52608d54d8a077fb87ac9d12e73c58` | MIT | 21 TypeScript + 13 TSX files | `ef6d48f7e24cc3cb56c82b862bcf23d813168fd4` | `c75def47f3fe0281ebac2e59f3310464ceac376c824006406f107dff0d935e6f` |
| `usememos-memos-v0.30.0` | `v0.30.0` / `2036c1ffc1b0a1e1fa6a473738c2a5ef520df67f` | MIT | 316 Go + 198 TypeScript + 235 TSX files | `76556530f2cf9f4f159e05dd79e9553ec461d42e` | `00973cde42c69cb631210b3503a6e33618c95d3c92e96850fad0aaf6839eaa3a` |

Memos excludes generated `proto/gen/**` and `web/src/types/proto/**` outputs.
Its portable manifest binds the repository-relative `web/tsconfig.json` used
by the TypeScript resolver.
No source file in the selected slices exceeds the fixed 1 MiB source ceiling.
All three checkouts were clean and matched their recorded commit and tree.

## Frozen boundary before execution

- Product defaults remain source dimension 1024, serving dimension 1024, and
  serving codec int8.
- The prior chi/RHF 32-case set remains closed historical calibration and is
  not used to select any new question, margin, closure rule, or hint budget.
- Provider-free discovery, indexing, parser coverage, graph construction, and
  inventory are run before any document or query embedding operation.
- Relation-challenge and naturalistic-prevalence cases are authored separately.
- Document capture and calibration queries remain distinct, plan-bound Voyage
  operations. Their exact input counts and cost ceilings must be recorded
  before each apply boundary.
- Local checkout bindings, databases, graph sidecars, vectors, and generated
  artifacts remain ignored. Only portable manifests, datasets, and evidence
  are tracked.

## Checks completed

- all three manifest JSON documents parse successfully;
- upstream URLs, exact tags/commits, licenses, trees, selected file counts,
  source ceilings, and selected-content hashes were independently verified;
- checkout-local state is clean; and
- no Voyage document/query call or assistant run occurred.

## Parser checkpoint

The first provider-free index pass found one structural corpus gap before any
question or score was inspected. Go-git and Memos indexed successfully, while
Zustand stopped on valid semicolonless consecutive generic call signatures in
four central public API files. Excluding those files would remove the contracts
the calibration is meant to exercise.

The bounded correction is TypeScript chunker v3 / global index-chunker v3. It
uses a same-length parser-only shadow for an erroring type alias and accepts the
shadow only when all parse errors disappear. Persisted source bytes and ranges
remain original, and each substituted separator emits a safe diagnostic. A
core fixture covers top-level and nested call signatures; malformed syntax
retains the existing fail-closed behavior. Focused normal/race tests, vet,
build, formatting, and diff checks passed. Clean-corpus reindex and relation
sidecar proof remain the next boundary.

The first mixed-language graph attempt also exposed a legacy root-only
`tsconfig.json` assumption. Corpus manifest v1 now supports an optional,
strictly relative `typescript_config`; it is valid only for a TypeScript/TSX
slice and participates in the portable manifest fingerprint. Existing corpora
continue to default to root `tsconfig.json`. Memos binds
`web/tsconfig.json`; no absolute path or machine-specific override is stored.

Reissuing the Memos inventory under the new corpus-manifest fingerprint found
that the immutable inventory filename bound only corpus ID, index generation,
and index manifest. It now also binds the full corpus-manifest fingerprint, so
two portable selection policies can safely coexist over one unchanged index
generation. Existing immutable packets are preserved; no artifact is deleted
or overwritten.

The first complete mixed graph also showed 788 file-resolution rows for 749
indexed Memos files. The 39-row difference exactly matched generated Go files
excluded by the manifest: `go/packages` correctly loaded them for compiler
resolution, but the diagnostic persisted their file states and classified
references to them as parent-mapping failures. The resolver now retains the
compiler's wider read scope while constraining persisted file states and
target membership to the committed index snapshot. Excluded or dependency
targets are `OUT_OF_CORPUS`; they cannot enter the graph universe.

## Provider-free index and relation boundary

The three isolated workspaces now pass indexed-file parity with no stale,
unindexed, or parse-error row:

| Corpus | Generation | Files | Parents/chunks | Segments | Distinct pending inputs | Index manifest |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| go-git | 2 | 474 | 5,057 | 5,659 | 5,655 | `40b2d4e15932c0ad194ecc28185eb541af6f05a1b63daf06e33623207cc0ba3c` |
| Zustand | 1 | 34 | 124 | 127 | 127 | `37f179e4f8f9f9f8604e5dce70178379d486dd37f95d101a7705d8eee37671eb` |
| Memos | 3 | 749 | 3,489 | 4,877 | 4,877 | `fc9b373a6fbb8d1de9486994907d56dc29d4b90f6a3b986471f9adad36141523` |

Clean v4 relation sidecars were repeated as v5 with identical logical graph
hashes:

| Corpus | Parents | Occurrences | Logical graph SHA-256 |
| --- | ---: | ---: | --- |
| go-git | 5,057 | 54,020 | `9b30a0495b4d13eefb241179358b9465fb1d60758bffb21692a116415a5db4da` |
| Zustand | 124 | 5,586 | `f4a354214a955694daf30c0b39f96e80ceeb8c226dd27a24b0b5641906293657` |
| Memos | 3,489 | 48,896 | `8b4209236fa0309355d15207517780257251a09b411dc82091b39546fcc3eacb` |

The Go resolver may read excluded generated files for compiler type checking,
but v4/v5 persist only indexed files and classify an excluded target as
`OUT_OF_CORPUS`. Artifact checksum manifests are complete. No provider call
was made while producing these indexes and graphs.

## Frozen draft calibration unit

The new calibration unit contains 40 source-bound questions. It intentionally
uses a 20/20 Go versus TypeScript/TSX split and a 20/20 naturalistic versus
relation-challenge split instead of padding the set with narrow edge cases.
The relation-challenge half samples caller/callee, declaration contract,
component contract, parser helper, recursive helper, iterator contract, and
algorithm/data-contract patterns. Two go-git questions carry reviewed,
confusable grade-0 hard negatives; the remaining cases do not invent weak
negatives merely to fill a quota.

| Dataset | Cases | Naturalistic | Relation challenge | Language split | File SHA-256 | Canonical dataset fingerprint |
| --- | ---: | ---: | ---: | --- | --- | --- |
| `relation-calibration-go-git-v5.19.1-draft-v1.json` | 12 | 6 | 6 | 12 Go | `b1054ee51ec7007ecd2397e8da8e17676938cd1d640dcd7864d4fd346c6b6fed` | `b5ddc298a3a7c8b5a816ce439208667fa566c69e175777d9205a6bf983ef80ba` |
| `relation-calibration-zustand-v5.0.14-draft-v1.json` | 12 | 6 | 6 | 12 TypeScript | `9c5922122b4d85027316392aed773ab2c84de21235c7822001e9c5e5b54eb614` | `9ddff18f0a0b5d7665e60b322e25f7b11ad41f0201ac4d3c650886974aba4db7` |
| `relation-calibration-memos-v0.30.0-draft-v1.json` | 16 | 8 | 8 | 8 Go + 8 TypeScript/TSX | `e5d254ca6008f92bc720f74db08e4358b47125fc1dc778a2a3db4efe0b148d24` | `4d3a546c9c7fd891c80f10f5be3c2e9a105f3458275d345a93b38529fa47fbd7` |

Every source span resolves uniquely in the v4 graph, every case passes the
strict portable dataset validator, and every draft digest reproduces from its
RFC 8785 framing. Question authoring used source structure only; no dense score
or result from these cases was inspected. Labels remain machine-prepared draft
authority until the later blind pooled review.

One optional lexical smoke reached final artifact construction and exposed a
pre-existing lexical-manifest incompatibility (`SourceDimensions=1` versus the
current 1024-only evaluation wire). Dataset, truth mapping, and digest checks
had already passed. The relation calibration does not consume that lexical
artifact; production FTS will be captured in the Phase 12 retrieval artifact.

## Next boundary

### Document source capture

The draft unit was frozen at clean commit
`d59a36ef5b1f4f79adf80b0853c8d8ef70caf5ce`. A clean Go 1.26.4 executable
reported that commit with `source_modified=false` before any provider call.

The public Voyage documentation retrieved on 2026-08-19 did not yet list
`voyage-code-4`; the owner had independently confirmed its table entry as
USD 0.12 per million input tokens and a 200-million-token allowance. The
experiment therefore records the price identity as
`owner-confirmed-voyage-code-4-2026-08-19-usd0.12-per-million`, without changing
the canonical v1 model. The project-local credential file was mode 0600,
git-ignored, and contained only the expected key variable; its value was never
printed or copied into an artifact.

| Corpus | Distinct inputs | Dry upper-bound tokens | Request groups | Actual tokens | Persisted | Failed |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| go-git | 5,655 | 3,492,377 | 45 | 1,050,252 | 5,655 | 0 |
| Zustand | 127 | 63,919 | 1 | 16,273 | 127 | 0 |
| Memos | 4,877 | 3,875,172 | 39 | 970,012 | 4,877 | 0 |
| **Total** | **10,659** | **7,431,468** | **85** | **2,036,537** | **10,659** | **0** |

At the frozen rate, the conservative initial-attempt ceiling was USD 0.8918
and the initial-plus-three-retries ceiling was USD 3.5671, both within the
owner's USD 5 billing ceiling. Actual observed input tokens correspond to at
most USD 0.2444 before any provider free allowance.

All three source banks pass SQLite integrity checks and contain exactly one
validated 1024-f32 row for every distinct canonical input. Local
materialization published exactly 10,659 `cidx-int8-symmetric-v1` serving rows,
occupying 10,914,816 int8 payload bytes.
Workspace status reports complete segment coverage (`5659/5659`, `127/127`,
and `4877/4877`), zero pending/failed inputs, and unchanged index generations
and manifests. A repeated capture plan has zero paid misses for all corpora.

## Next boundary

Run provider-free retrieval plans for the 12/12/16 draft datasets under one
`RELATION_CALIBRATION_POOL_BUILDING` series of exactly 40 logical query
operations. Freeze the same price identity and USD 5 series cap, then apply
each query once. The resulting full active-int8 score artifacts feed relation
completion; no query vector is persisted and no historical chi/RHF query is
reopened.
