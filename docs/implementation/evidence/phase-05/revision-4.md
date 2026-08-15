# Phase 05 Revision 4 Reconciliation Evidence

- State: done
- Owner: `/root/r4_phase05_index` (terra/high implementation agent); Codex validates once at the commit boundary
- Entry commit: `dde2893`
- Entry evidence: the canonical Revision 4 design, execution guide, evaluation contract, full Phase 05 document, Phase 02 R4 evidence, Phase 03/04 completion evidence, historical Phase 05 evidence, supersession policy, and the live index/store/chunker wiring were inspected.

## Owned reconciliation

- Rename the injected AST segmentation policy from a maximum-limit name to `TargetSegmentBytes` without reopening the proven Phase 03/04 algorithms.
- Prove the new index-profile fingerprint causes a complete free local reindex and preserves whole semantic parents in FTS while embeddings remain AST-boundary segments with oversize units intact.
- Preserve existing pre-R4 production vectors only when stored legacy profile JSON/fingerprints, source, serving dimension, reducer, normalizer, metric, codec/version, canonical input identity, lineage, timestamp, and blob integrity prove exact semantic equivalence.
- Plan compatible rekeys before publication and copy them to the new serving key in the same transaction that switches segments, profiles, generation, and active serving profile.
- Treat every unproven or changed key as pending; perform no provider, network, lab-DB, or paid action.
- Leave inactive-cache garbage collection outside this transition. Phase 10 owns only missing current-profile document embeddings after Phase 05 completes.

## Entry state

- Phase 02 already injects `ResolvedConfig.Index.TargetSegmentBytes` and emits the new index/vector fingerprints.
- Go and TypeScript/TSX chunkers already pack only complete AST statement/member boundaries and retain one oversize AST unit whole; the shared policy is renamed mechanically to `TargetSegmentBytes`.
- The existing index publish atomically relinks segments and profile metadata but does not yet copy proven-equivalent pre-R4 vector rows to the new serving key.

## Required evidence before completion

- Focused target-policy rename/build coverage and the existing Go/TS oversize/AST-boundary fixtures.
- Index-profile mismatch causes a full local rebuild with no embedding-client dependency.
- Equivalent legacy profile/vector rows count as reused in dry-run and are ready under the new key after apply.
- Dimension, reducer/normalizer/metric, source, codec/version, canonical hash, lineage/timestamp, forged profile/fingerprint, and invalid-blob mismatches never copy.
- Rekey plus segment/meta/generation switch is atomic; stale-base or injected failure leaves the old generation/profile/vector state observable.
- Checks actually run, checks not run, residual risks, and downstream handoff are recorded here before Phase 05 is marked done.

## Reconciliation implementation checkpoint

- `chunk.SegmentationPolicy.TargetSegmentBytes` now names the injected AST packing target in the shared contract, Go/TypeScript implementations, index service, and existing focused fixtures. The algorithms remain unchanged: complete AST units are packed to the target and a single oversize unit is retained whole.
- An index-profile fingerprint mismatch classifies as `ImpactLocalReindex`; Phase 05 reparses every eligible file even with an unchanged source hash. A focused R4 test uses a 32-byte target to prove one complete function parent remains one FTS chunk while its AST-boundary embedding candidates split into multiple segments.
- `IndexSnapshot` now retains only the stored canonical/source/vector profile JSON necessary to establish legacy local-reuse evidence. `config.LegacyServingProfileEquivalent` strictly decodes canonical pre-R4 JSON, reproduces its recorded fingerprints, requires the legacy `target_dimensions`/reducer/normalizer/metric/source/codec semantics to equal the requested R4 profile, and rejects any other shape or forged byte/fingerprint relationship.
- Before publication, the index service derives the exact next-generation canonical-input set, builds an immutable local rekey plan only for retained keys, and counts those rows as reused in dry-run results. The store rereads the base metadata and original old-key vector row inside the final write transaction, validates lineage/raw hash/timestamp/blob and the desired codec representation, then atomically upserts the desired-key copy with segment links, profile metadata, and the generation switch. Failed proof is pending, never a provider/lab/network action; inactive rows remain untouched.

## Checks actually run

```text
gofmt -w internal/chunk internal/config/legacy_profile.go internal/index/service.go internal/index/service_test.go internal/store/index_snapshot.go internal/store/index_publish.go internal/store/legacy_rekey.go internal/store/legacy_rekey_test.go
go test -count=1 ./internal/chunk/... ./internal/config ./internal/index ./internal/store
go test -count=1 ./internal/index ./internal/store ./internal/config ./internal/chunk/...
go test -count=1 -race ./internal/chunk/... ./internal/config ./internal/index ./internal/store
go vet ./internal/chunk/... ./internal/config ./internal/index ./internal/store
go build ./internal/chunk/... ./internal/config ./internal/index ./internal/store
git diff --check
rg -n "TargetSegmentBytes" internal/chunk internal/index
```

The focused tests pass. They cover the mechanical Go/TS policy fixtures, forced R4 full rebuild with a whole-parent FTS hit and multiple AST segments, legacy dry-run `reused=1/pending=1` followed by atomic apply, equivalent vector copy, dimension/reducer/source/codec/canonical-key/forged-JSON/invalid-blob rejection, source-row reproof inside publish, stale generation rejection, and a forced post-rekey transaction failure rollback.

## Checks not run

- No broad project test, repository-wide race run, repository-wide vet/build, FTS query package, CLI, MCP, provider, network, paid embedding, lab DB, corpus, or evaluation run has been performed by this checkpoint.
- Main commit-boundary validation and the required Terra review remain pending.

## Decisions and residual risks

- A pre-R4 vector is copied only from its active legacy serving key and only after profile JSON, fingerprint, lineage, timestamp, raw hash, source cache row, and cidx codec validation all agree. Existing desired-key rows are deterministically updated from this proven immutable record in the same transaction rather than causing an avoidable conflict.
- The legacy JSON parser intentionally accepts no current/R4 `serving_dimensions` shape, aliases, unknown fields, or non-canonical bytes. This transition is one-way and does not make old profile vocabulary runtime configuration.
- The source-row reproof protects a plan from concurrent vector-cache changes, but cannot establish any history that was absent from a migrated legacy database; absent lineage is deliberately pending.

## Terra review and main-agent commit-boundary validation

The independent Terra/high review reported no findings. It verified the mechanical target-policy rename, full local rebuild, exact future-key calculation, strict profile/vector proof, transaction-local reproof, rollback/stale behavior, dry-run accounting, Phase 05 scope, and evidence accuracy.

The designated one-time main-agent boundary validation passed on 2026-08-15:

```text
go test -count=1 -race ./internal/chunk/... ./internal/config ./internal/index ./internal/store
go vet ./internal/chunk/... ./internal/config ./internal/index ./internal/store
go build ./...
gofmt -l internal/chunk internal/config/legacy_profile.go internal/index internal/store
rg -n 'MaxSegmentBytes' internal
rg -n '"cidx/internal/(embed|embedclient|lab)"' internal/index
git diff --quiet dde2893 -- internal/store/schema.go internal/lab/schema.go
git diff --check dde2893
```

All commands passed. The two `rg` checks produced no matches, and the relational production/lab schema files are unchanged from the Phase 05 entry commit.

## Handoff and next action

- Phase 05 is complete. Phase 08 may consume the active canonical-input set and desired serving keys; Phase 10 receives only current-profile misses after Phase 08's shared request executor is reconciled.
- Phase 06 continues to receive one committed active generation with whole-parent FTS chunks. Phase 08/10 receive desired serving keys where only proven legacy rows are ready; all others remain explicit current-profile pending work with no paid action taken here.
