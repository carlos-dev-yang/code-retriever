# Phase 02 Configuration, Profiles, and Schema Evidence

- Phase: `02-config-profiles-and-schemas`
- State: `historical completion; reopened by the 2026-08-17 int8/source-bank contract`
- Date: 2026-08-15

The checks below remain accepted for their original boundary. Current Phase 02
completion additionally requires default 1024/optional 512, fixed int8 with no
Binary/256 path, and separate serving, product-source, and vector-free lab
stores. See the [source-bank decision](../../SOURCE-VECTOR-BANK-DECISION.md)
and [retired-profile contract](../../RETIRED-VECTOR-PROFILES.md).

## Implemented contracts

- `internal/config` is the only config JSON reader. It rejects invalid UTF-8, duplicate keys, unknown fields, trailing values, unsupported model/codec/transform settings, and invalid limits before store opening.
- `RawConfig -> Resolve -> Validate -> ResolvedConfig` provides immutable typed values. `embedding.target_dimensions` is required; binary is the default storage codec.
- The model registry derives `voyage-code-4`, provider, source 1024, roles, dtype, truncation, and allowed target dimensions from Phase 01 code-owned values.
- Index, canonical-text, source, vector-space, and storage profiles use sorted canonical JSON and domain-separated SHA-256. Phase 00 canonical-text and source-profile fixtures reproduce exactly.
- `ConfigImpactPlan` classifies schema, local reindex, canonical-text reconciliation/re-hash, paid-source, local-rematerialization, and restart-only policy requirements without performing any action. Canonical-text change does not itself authorize paid work: only changed hashes found by reconciliation may become downstream paid misses.
- `internal/chunk` owns language-neutral chunk/range/projection/segment values and the context-aware `Chunker` interface. Requests inject a versioned segmentation policy, results carry typed parser diagnostics/metadata, and an immutable `LineIndex` provides O(log n) byte-to-1-based-inclusive-line conversion (CRLF and UTF-8 safe). `symbol.IdentifierNormalizer` is shared for index/query identifier splitting.
- Production and lab use distinct connection factories, paths, atomic `PRAGMA user_version` migrations, canonical symlink-resolved root identity, and owner-only `.cidx`/database permissions where supported. Production vector storage has only cidx codec blob metadata; lab raw storage is document-only and fixed to dimension 1024.
- `internal/evalcontract` defines portable query/run/trace/promotion values, stable enums, typed per-stage truth-unit denominators, paired controls, deterministic artifact framing, and matching recursively strict/versioned JSON schema artifacts. Evaluation cases carry durable constraints, ABSTAINABLE zero-group/no-answer truth, graded 0/1/2 relevance judgments, reviewed hard-negative reasons, review state/passes/reviewer identity/rationale, solo-review limitation, and optional assistant-task requirements without generated row IDs. Group-level trace observations are the sole first-loss authority; the explicit provider-union stage begins the primary path, optional assistant stages remain `NOT_OBSERVED`, and operational observations carry only operation denominators.

## Selected Phase 01 values

The active reducer is `prefix-l2-v1` with `l2-v1`, `cosine`, `cidx-binary-sign-lsb-v1`, and `cidx-int8-symmetric-v1`. Phase 00's `prefix-v1` vector-space digest remains only a canonicalization fixture, not an accepted active reducer.

## Checks actually run

```text
gofmt -w internal/config internal/embedclient internal/evalcontract internal/lab internal/profile internal/store internal/vector internal/chunk
go test -count=1 ./internal/config ./internal/profile ./internal/store ./internal/lab ./internal/evalcontract ./internal/chunk ./internal/symbol ./internal/vector ./internal/embedclient
go vet ./internal/config ./internal/profile ./internal/store ./internal/lab ./internal/evalcontract ./internal/chunk ./internal/symbol ./internal/vector ./internal/embedclient
go build ./internal/config ./internal/profile ./internal/store ./internal/lab ./internal/evalcontract ./internal/chunk ./internal/symbol ./internal/vector ./internal/embedclient
go list -deps ./internal/store  # no cidx/internal/lab result
go mod tidy -diff
git diff --check
```

Results: strict config tests reproduce Phase 00 canonical-text/source fingerprints and reject duplicate/unknown fields and lone UTF-16 surrogates while accepting valid pairs; RFC-8785 used-value checks cover HTML-sensitive strings, UTF-16 key order, explicit zero, finite fractional FTS weights, recursive profile UTF-8 prevalidation, and ECMAScript-style number rendering. Omitted search defaults resolve successfully while explicit zero and hybrid-without-permission reject; all impact classes and precedence are covered. Production/lab tests prove atomic new/current/newer/unknown migration behavior, symlink-root equivalence, symlinked state/database rejection, owner permissions where supported, cidx-only vector CHECK constraints, required metadata, and no f32/f16 columns. Active-state reads pin the active profile and rows to one read transaction; writes recheck the active profile in their own mutation transaction. Lab f32 rows are immutable with transaction-safe concurrent idempotent identical writes only. A real Go JSON-Schema validator validates strict nested case/trace/promotion fixtures and rejects generated row IDs, path traversal, stage-order/denominator/unobserved violations, invalid split/review, and weighted-total or inconsistent promotion fields; Go validation additionally enforces group identity, first-loss immutability, failure-stage matching, and grade/cross-record truth. All listed test, vet, build, dependency, module, and diff checks passed.

Final focused correction pass: `go test -count=1 ./internal/chunk ./internal/evalcontract ./internal/lab ./internal/store` passed after the group-level operation-failure and line-index changes. The final strictness review additionally makes case digests/cohorts/constraints canonical and unique, denies duplicate artifact paths, and keeps config-file and production-schema version authorities separate by injecting the expected production schema version into impact planning. No paid/provider, corpus, parser/indexer/search/MCP, raw capture, materialization, or retrieval-evaluation action ran.

Commit-boundary strictness follow-up: `go test -count=1 ./internal/evalcontract` and `git diff --check` passed. Applied-profile handoff now includes active generation, manifest digest, and active serving profile; it is read from production metadata only and adds no publication behavior.

## Main-agent commit-boundary validation

The main agent reviewed the final implementation directly and ran the following Phase 02 boundary checks after the implementation-agent handoff:

```text
git diff --check
go test -count=1 ./internal/config ./internal/profile ./internal/store ./internal/lab ./internal/evalcontract ./internal/chunk ./internal/symbol ./internal/vector ./internal/embedclient
go test -count=1 -race ./internal/store ./internal/lab ./internal/config ./internal/evalcontract ./internal/chunk
go vet ./internal/config ./internal/profile ./internal/store ./internal/lab ./internal/evalcontract ./internal/chunk ./internal/symbol ./internal/vector ./internal/embedclient
go build ./internal/config ./internal/profile ./internal/store ./internal/lab ./internal/evalcontract ./internal/chunk ./internal/symbol ./internal/vector ./internal/embedclient
gofmt -l internal/config internal/profile internal/store internal/lab internal/evalcontract internal/chunk internal/symbol internal/vector internal/embedclient
go list -deps ./internal/store
go mod tidy -diff
jq -e . schemas/evaluation/*.json
```

After the final strictness fixes, the main agent reran only the directly affected packages:

```text
go test -count=1 ./internal/config ./internal/store ./internal/evalcontract ./internal/vector
go test -count=1 -race ./internal/store ./internal/evalcontract ./internal/config
go vet ./internal/config ./internal/store ./internal/evalcontract ./internal/vector
go build ./internal/config ./internal/store ./internal/evalcontract ./internal/vector
```

All checks passed. Formatting produced no file list, the production store dependency graph contained no `cidx/internal/lab`, every evaluation schema parsed, module metadata had no pending tidy diff, and `git diff --check` remained clean.

## Checks not run and remaining risk

- No `serve` application exists yet, so only the production-store dependency path—not a future serve graph—can be checked for lab isolation.
- No paid API, corpus, parser/indexer/search/MCP, raw capture, materialization, or retrieval evaluation ran.
- No persisted Phase 01 spike database migration is supported; spikes are isolated test artifacts, not a released schema. Formal production/lab migrations fail closed rather than guessing an unknown historic schema.

## Downstream handoff

Phases 03/04 consume shared chunk/profile contracts. Phases 05–11 consume resolved config, profiles, store boundaries, active-vector validation, impact planning, and normalization. Phases 07/12 consume evalcontract values/schema/artifact framing without redefining them.
