# Phase 11 int8-only query/search reconciliation

- State: accepted at the Phase 11 commit boundary.
- Date: 2026-08-17.
- External actions: no credential read, corpus read, provider request, network
  operation, paid query, or metric measurement was performed.

## Accepted runtime boundary

- A hybrid query requests one ephemeral `voyage-code-4` 1024-f32 query vector
  through the already accepted synchronous executor. The vector is never
  persisted.
- The shared transform reduces that vector to the active serving dimension,
  either the default 1024 or explicit compact 512, and L2-normalizes it.
- `PrepareInt8Query` quantizes the transformed query once. The exhaustive scan
  then uses only the fixed int8 scorer against the one active production
  profile.
- Runtime search never imports or opens the product document source bank, the
  evaluation database, or development adapters.
- FTS preflight/fallback, generation and profile reproof, segment-to-parent
  collapse, deterministic RRF, coverage reporting, and indexed-body packaging
  are unchanged.

## Retired comparison removal

- Removed the obsolete `candidate_int8` evaluation variant. Current fidelity
  compares only exhaustive serving-dimension f32 with the active int8 ranking.
- Removed the unused vector-only experiment option and its public preflight
  session entry point. Current evaluation always freezes the complete
  retrieval snapshot required by its FTS, dense, and hybrid arms.
- Removed a dead vector-only hit helper.
- No Binary scorer, Binary identifier, codec-comparison command, or
  256-dimensional serving path exists in current Go or evaluation-schema
  execution surfaces. Historical Binary/256 reports remain documentation and
  immutable evidence only.

The serving-f32 arm is intentionally retained as a nonpersistent evaluation
reference. It is not a selectable production storage profile and does not add
a second runtime codec.

## Commit-boundary validation

The focused normal and race suites passed. The first static residue expression
identified one dead helper named for the retired vector-only mode; that helper
was removed. A later schema expression matched an unrelated `256 KiB` value on
the same minified JSON line, so the final schema check parsed the enum
structurally instead of repeating the already-passing test suites.

```text
go test -count=1 ./internal/config ./internal/vector ./internal/store ./internal/embedclient ./internal/embed ./internal/search ./internal/eval ./internal/devlab ./internal/app
go test -count=1 -race ./internal/vector ./internal/store ./internal/embedclient ./internal/embed ./internal/search ./internal/eval ./internal/devlab ./internal/app
go vet ./internal/config ./internal/vector ./internal/store ./internal/embedclient ./internal/embed ./internal/search ./internal/eval ./internal/devlab ./internal/app
go build ./...
gofmt -l cmd internal
go list -deps ./internal/search
jq -e '."$defs".controls.properties.serving_dimensions.enum == [1024,512]' schemas/evaluation/run-manifest.schema.json
jq -e '."$defs".controls.properties.serving_dimensions.enum == [1024,512]' schemas/evaluation/promotion-contract.schema.json
go mod tidy -diff
git diff --check
```

Static inspection found no retired codec/comparison identifier, no exact
256-dimensional serving assignment, and no `sourcebank`, `lab`, `devlab`, or
`devapp` dependency below `internal/search`.

## Handoff

Phase 12 receives the same active-int8 scan and the same nonpersistent
serving-f32 reference, collapse, RRF, fallback, and body-packaging paths. It
must not recreate a retired codec arm or a second evaluation-only search
engine.
