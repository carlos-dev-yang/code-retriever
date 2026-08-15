# Phase 12 Retrieval Evaluation Infrastructure Evidence

- Phase: `12-retrieval-evaluation`
- State: `blocked` — reusable corpus-independent infrastructure is implemented; official evaluation and promotion evidence require external, user-controlled inputs.
- Date: 2026-08-15

## Implemented infrastructure

- `internal/eval/retrieval.go` defines the frozen Phase 12 arm plan: FTS, target-f32, active-codec, provider union, both RRF arms, and both lane ablations. It rejects a missing, reordered, duplicate-parent, or mismatched-query arm before results can be measured.
- `RunRetrievalEvaluation` is the Phase 13 adapter seam. It accepts injected actual-arm results, retains typed failed/time-out arms as zero-valued denominator observations, and preserves caller cancellation even if an adapter returns afterward. It requires the target-f32, active-codec, provider-union, and dense-containing hybrid arms to retain the same request-local query-vector SHA-256, and explicitly rejects a query hash on FTS-only and `hybrid_without_dense` arms. Arm failure stages are variant-specific: for example, an FTS arm cannot claim a dense-segment failure. It runs the existing Phase 07 human-gold metric adapter for every arm and computes target-f32/current-codec fidelity, FTS/dense/RRF contribution, and active-hybrid body-package diagnostics once per query. It contains no provider, corpus, lab, or production-storage dependency.
- Codec fidelity retains target-f32 top-k retention, missing neighbors, top-1 mismatch, gold retention, displacement, and separate target-f32/current-codec score-tie and top-k-boundary-tie rates. Pairwise inversion counts only pairs whose native scores establish strict order in both arms, so a codec tie broken deterministically is not mislabeled as a semantic inversion.
- Fusion diagnostics retain lane overlap, lane-only/both contribution, rescue/harm, and lane-to-fused rank movement without comparing native score scales. If an arm needed by a fidelity, fusion, or body diagnostic fails, that diagnostic is explicitly unavailable with its failure stage rather than an all-zero observation. Body diagnostics require one record for every fused top-k parent, including omissions; accept only the two Phase 11 omission reasons; retain exact full-parent versus partial-range accounting, SHA-256 body-digest duplicate ratio (without source bytes), overlap-unioned relevant-byte density, and omission reasons.
- `CorePromotionEvidence` reuses `internal/evalcontract` wire schemas and validates the frozen core-retrieval promotion contract/result relationship, confirmation compatibility controls, immutable artifact checksum/completion representation, required Phase 12 evidence paths including provider usage, a human-readable report, and checksums, plus complete ready-gate coverage. It cannot treat a core result as a release-candidate result.
- The local-only adapter now writes an immutable execution artifact under `.cidx/lab/evaluations/<run-id>` after an explicit apply. It contains the documented vector-free run manifest, actual stage trace, raw FTS and pre-collapse dense-segment observations, collapsed-parent/RRF/body records, metric slices, provider usage, implementation audit, an explicit not-promotion-ready record, and an atomic checksum completion marker. `evaluation_runs` v4 stores only vector-free provenance and the artifact reference/checksum. The v3-to-v4 migration preserves historical rows as explicitly `legacy`; no raw vector or query text is migrated or added. Execution-artifact completion is intentionally distinct from `CorePromotionEvidence`: official promotion files cannot be valid without an approved frozen promotion contract and confirmation inputs.
- The adapter classifies only errors returned by the embedding client as typed operational denominator failures. Snapshot corruption, raw-bank mismatch, response validation, transform, collapse, packaging, and other local invariant failures remain structural aborts rather than being misreported as provider/FTS observations.

## Checks actually run

Codex repeated the focused boundary validation on 2026-08-15 before the
infrastructure commit. All commands below passed.

```text
gofmt -w internal/eval/retrieval.go internal/eval/eval_test.go
go test -count=1 ./internal/eval ./internal/evalcontract
go test -count=1 -race ./internal/eval ./internal/evalcontract
go vet ./internal/eval ./internal/evalcontract
go build ./internal/eval ./internal/evalcontract
gofmt -l internal/eval internal/evalcontract
go list -deps ./internal/search ./internal/store | rg 'cidx/internal/(eval|lab)'   # no matches
git diff --check
```

All passed. The focused tests include an end-to-end synthetic arm executor that supplies every frozen arm and one ephemeral query-vector hash, producing human metrics plus fidelity, fusion, and body diagnostics. They reject a mismatched query-vector digest, incomplete arm plan, missing fused-body omission, unknown omission reason, malformed/full-parent body accounting, overlapping-span density overcount, forged computed evidence, missing provider/report/checksum promotion artifacts, and a ready promotion result that does not pass every frozen gate. A tied codec reports tie/boundary diagnostics without manufacturing an inversion, and distinct parents with one body digest report duplicates. Typed failures remain in their arm metric denominator; dependent diagnostics are unavailable, and a context cancellation returned after an adapter call still cancels the run.

## Checks not run and exact blocker

- No corpus was selected, downloaded, cloned, bound, indexed, copied, captured, or embedded. No local checkout binding, approved manifest, reviewed calibration/confirmation dataset, raw-bank coverage record, official artifact, or official score exists.
- No provider/API-key/network operation occurred, and no paid query embedding was planned or applied. No query vector, source body, raw vector, credential, or promotion evidence was persisted.
- Official Phase 12 completion remains blocked until the user provides approved tracked corpus manifests, ignored local checkout bindings, reviewed calibration and confirmation datasets, and matching raw-bank/materialized-profile coverage. After preflight succeeds, a separate explicit approval is required before sending evaluation query text to Voyage AI.

## Phase 13 handoff

The corpus-independent development adapter is implemented in `internal/devlab`. Its plan command strictly validates the portable manifest, local checkout binding, reviewed dataset, indexed source slice, active profile, and complete existing raw-bank coverage without reading a provider key, creating lab state, or calling a provider. Its explicit `--apply` path adapts the real Phase 11 snapshot, collapse, RRF, and body-packaging lanes to `RunRetrievalEvaluation`; query vectors remain request-local and required failed operations remain in run denominators.

Focused adapter checks on 2026-08-15 passed for `internal/app`, `internal/eval`, `internal/lab`, `internal/devlab`, and `internal/search`, including normal and race tests, vet, a `/tmp` CLI build, the production dependency boundary, formatting, and diff validation. This is implementation evidence only: it is not an official corpus run, score, artifact set, or promotion result.

The synthetic adapter fixture creates a temporary Git repository, index, raw bank, and locally materialized serving profile. It proves a successful plan makes no provider call and creates no evaluation/artifact state; an apply performs exactly one fake query call per case, reuses its one digest across every vector-bearing arm, runs all eight real lanes, records parent body packaging, atomically publishes the required execution artifact files, and adds one vector-free DB reference. A fake client failure remains in every dependent arm denominator; a malformed provider response aborts without publication or a DB row.
