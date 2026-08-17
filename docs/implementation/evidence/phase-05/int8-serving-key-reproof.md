# Phase 05 Int8 Serving-Key Reproof

- State: accepted
- Owner: `/root`
- Entry commit: `969bd89`
- Decision date: 2026-08-17

## Scope

This boundary re-proves the existing Phase 05 local rekey transition after the
product profile was narrowed to default 1024/int8 and optional 512/int8. It
does not open the document source bank or evaluation database and does not
perform provider, network, or embedding work.

## Result

- The production rekey implementation was already fail-closed: it requires the
  requested `ResolvedConfig` to pass integrity validation, canonical legacy
  profile JSON to reproduce every stored fingerprint, and dimension, reducer,
  normalizer, metric, codec, canonical input, lineage, timestamp, and stored
  vector integrity to match exactly.
- Because current configuration admits only 1024/int8 or 512/int8, a normal
  historical 256-dimensional profile and a normal historical Binary storage
  profile both fail equivalence and remain pending. Neither can become the
  active serving profile.
- A strictly equivalent historical int8 row may still be copied to the desired
  serving key. The immutable source row is re-read and revalidated inside the
  generation publication transaction.
- Missing or unproven rows remain current-profile pending inputs for later
  source-bank reuse or document embedding. Phase 05 does not decide or execute
  that work.

No production implementation change was required. Existing core fixtures were
mechanically moved to 1024/int8 and their existing rejection table now proves
the retired 256 and Binary cases explicitly.

## Boundary validation

The main-agent boundary was run once after the fixture reconciliation:

```text
go test -count=1 ./internal/config
go test -count=1 ./internal/store -run '^TestLegacyServingVectorRekey'
go test -count=1 ./internal/index -run '^(TestR4IndexProfileForcesFullLocalRebuildWhileFTSUsesWholeParent|TestR4LegacyVectorRekeyAppearsInDryRunAndAppliesWithFullRebuild)$'
go test -count=1 -race ./internal/store -run '^TestLegacyServingVectorRekey'
go test -count=1 -race ./internal/index -run '^(TestR4IndexProfileForcesFullLocalRebuildWhileFTSUsesWholeParent|TestR4LegacyVectorRekeyAppearsInDryRunAndAppliesWithFullRebuild)$'
go vet ./internal/config ./internal/index ./internal/store
go build ./...
gofmt -l internal/store/legacy_rekey_test.go internal/store/production_test.go internal/index/service_test.go
git diff --check
```

All commands passed. No provider, corpus, evaluation, source-bank, or lab action
was run.

## Handoff

Phase 08 may now separate immutable document-role 1024-f32 source rows into the
product source bank at `<state_root>/db/embeddings.db` and leave vector-free
evaluation metadata in `<state_root>/lab/evaluation.db`. Search, serve, and MCP
must never open either database.
