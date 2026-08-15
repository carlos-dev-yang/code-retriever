# Phase 02 Revision 4 Reconciliation Evidence

- State: done
- Owner: `/root/r4_phase02_config` (terra/high implementation agent); Codex validates once at the commit boundary
- Entry commit: `a6f1f53`
- Entry evidence: Revision 4 canonical design, execution guide, evaluation contract, Phase 00 R4 evidence, Phase 01 evidence, historical Phase 02 evidence, and the supersession policy were checked.

## Owned reconciliation

- Strict final config version 1 fields and central defaults/absolute ceilings.
- `serving_dimensions` profile and evaluation-wire vocabulary.
- Removal of configurable chunk and read-span line caps.
- Separate byte-bounded synchronous request and staged retry profiles.
- Typed fail-closed legacy-shape error with exact field mappings; no aliases.
- New semantic fingerprints without a relational production/lab schema migration.
- Mechanical call-site and fixture updates required for the new immutable typed contracts; downstream behavior remains owned by its original phase.

## Implemented reconciliation

- `RawConfig -> Resolve -> Validate -> ResolvedConfig` accepts only the final v1 fields. It defaults `index.max_source_file_bytes` to 1 MiB, `index.target_segment_bytes` to 1,024 bytes, request limits to 128/256 KiB/4/30 seconds, retry to 3 with 10/20/30-second waits, FTS/binary policy, and MCP inline bytes to 64 KiB; executable ceilings reject larger values.
- `embedding.serving_dimensions` is required. Index/vector-space/evaluation wire profiles use `target_segment_bytes` and `serving_dimensions`; only the local vector math helper retains its internal `TargetDimensions` name.
- Config decoding detects pre-R4 fields before strict decode/store use and returns `*config.LegacyConfigError` with deterministic exact mappings. It accepts no aliases. Production and lab relational schema versions remain unchanged; new profile JSON/fingerprints force the appropriate later reconciliation.
- `config.DefaultRaw(servingDimension, codec)` returns a complete valid raw v1 config without filesystem effects. Phase 13 still owns writing/initializing `.cidx/config.json` and the public CLI migration.
- Mechanical consumer/fixture changes preserve compilation. The existing index receives the renamed segment target and the existing read-span implementation no longer consumes a removed line cap. This phase does not add AST segmentation behavior, byte-bounded concurrent execution, retry waiting, query retries, or filesystem initialization.
- `PairedRunControls` and the strict run/promotion JSON schemas reject `target_dimensions` and require `serving_dimensions`; there is no official Phase 12 artifact to migrate.

## Checks run by the implementation agent

```text
gofmt -w internal/config internal/profile internal/evalcontract internal/app internal/cli internal/devapp internal/devlab internal/index internal/search internal/store
go test -count=1 ./internal/config ./internal/profile ./internal/evalcontract
go test -count=1 ./internal/app ./internal/cli ./internal/index ./internal/search ./internal/store
go build ./...
go vet ./internal/config ./internal/profile ./internal/evalcontract
jq -e . schemas/evaluation/*.json
git diff --check
```

All checks passed. Focused tests cover resolved final defaults/ceilings, the serving-dimension fingerprint fixture, full `DefaultRaw` resolution, explicit `0`/`-0`/`null` safety rejection, supplied empty/null retry-wait rejection, deterministic typed legacy mappings (including non-object `embedding.batch`), and strict evaluation-schema rejection of the old wire field.

## Main-agent commit-boundary validation

The designated one-time boundary validation passed on 2026-08-15:

```text
go test -count=1 -race ./internal/config ./internal/profile ./internal/evalcontract
go test -count=1 ./internal/app ./internal/cli ./internal/devapp ./internal/devlab ./internal/eval ./internal/index ./internal/search/... ./internal/store
go vet ./internal/config ./internal/profile ./internal/evalcontract
go build ./...
jq -e . schemas/evaluation/*.json
gofmt -l <Phase 02 and mechanically affected Go package directories>
git diff --check
git diff --quiet a6f1f53 -- internal/store/schema.go internal/lab/schema.go
rg <removed public R4 names> internal/config internal/profile internal/evalcontract schemas/evaluation
```

All commands passed. The remaining old-name matches are confined to typed legacy migration coverage, schema rejection coverage, and the local vector transform adapter's internal mathematical `TargetDimensions` field. The production and lab relational schema files are byte-identical to the entry commit.

## Checks not run / downstream handoff

- No paid provider request, corpus action, raw capture, materialization, parser/indexer behavior test beyond renamed-field compilation, or retrieval evaluation ran.
- Phase 05 owns actual AST-aware packing semantics under `target_segment_bytes`; Phase 08/10 own byte grouping, concurrency, staged waits, `Retry-After`, and cancellation; Phase 11 owns query retry policy; Phase 13 owns filesystem init and final public CLI/MCP validation.
- Phase 02 is complete and hands the immutable R4 config/profile/evaluation-wire contracts to Phases 05, 08, 10, 11, 12, and 13.
