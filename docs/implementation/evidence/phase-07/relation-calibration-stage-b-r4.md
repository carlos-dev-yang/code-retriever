# Relation Calibration Stage B — Revision 4

- Date: 2026-08-19
- State: in progress; corpus selection and portable provenance frozen
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

## Next boundary

Initialize and index three isolated 1024/int8 evaluation workspaces. Verify
indexed-file parity and parser/semantic-parent inventories, then build the
generation-bound relation sidecars. Any mixed-repository resolver gap must be
fixed only in the evaluation path and revalidated before cohort authoring.
