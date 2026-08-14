# Phase 03 Go Chunker Evidence

- Phase: `03-go-chunker`
- State: `complete; main-agent validation accepted`
- Date: 2026-08-15

## Implemented contract

- `internal/chunk/golang` is a stateless offline Go Tree-sitter adapter that implements the Phase 02 `chunk.Chunker` interface. Each invocation owns its parser/tree and reads only the supplied source bytes.
- It emits deterministic source-order chunks only for named functions, methods, and named `type_spec`/`type_alias` declarations. Const/var declarations and anonymous function literals are excluded.
- Function/method chunks include complete declaration and body projections. Type chunks retain their exact type source; grouped declarations emit one isolated exact source span per member. Because a group member's contiguous source span cannot include the wrapper `type` token, its display/search `Signature` is intentionally the stable synthetic `type Name ...` header.
- Directly attached contiguous Go doc-comment blocks are included in the source body and projection. The code-owned chunker ID is `go-tree-sitter-0.25.0-doc-comments-v1`.
- Pointer and generic receiver syntax normalizes to the first receiver base type identifier (`*Receiver[T] -> Receiver`). Invalid receivers are not invented.
- Chunk ranges use original bytes and Phase 02 one-based-inclusive `LineIndex` lines. Source-body equality, projection ordering, segment display containment, and result ordering are validated before returning.
- Long functions split only at top-level Tree-sitter statement boundaries; struct/interface members are likewise available as type boundaries. The injected `SegmentationPolicy` owns the byte cap and boundary-policy identity. A single oversize AST statement/member remains one validated oversize candidate rather than being byte-split.
- Invalid UTF-8 fails closed. Tree-sitter syntax errors are diagnosed deterministically; safe declarations outside an error may be returned, while unsafe declarations and overlapping errors are non-indexable.

## Checks actually run

```text
gofmt -w internal/chunk/golang
go test -count=1 ./internal/chunk/golang
go vet ./internal/chunk/golang
go build ./internal/chunk/golang
git diff --check
```

Result: passed. The focused fixture suite verifies functions, generic pointer receivers, grouped struct/interface/alias declarations and synthetic grouped-type signatures, doc-comment inclusion, const/var/anonymous exclusions, exact CRLF/Unicode source ranges, deterministic output, cancellation, malformed-input recovery, statement-boundary packing, and validated oversize type-member candidates.

Main-agent completion validation additionally ran:

```text
gofmt -l internal/chunk/golang
go test -count=1 ./internal/chunk/golang
go test -count=1 -race ./internal/chunk/golang
go vet ./internal/chunk/golang
go build ./internal/chunk/golang
git diff --check
```

Result: passed. The main review also checked that the adapter has no filesystem, database, Git, or provider dependency and that grouped-type source spans remain byte-exact while signatures preserve the `type` header.

## Checks not run

- No prior Phase 02 test suite was rerun.
- No repository-wide test, vet, build, race, corpus, database, filesystem traversal, Git operation, provider, paid API, indexer, FTS, or MCP validation ran.

## Residual risks

- The grammar and fixtures are validated locally on the Phase 01 darwin/arm64 CGO environment only; other packaging targets remain a later-phase concern.
- Parse recovery is intentionally syntax-only: it does not type-check Go build tags, imports, or receiver legality beyond a safe Tree-sitter receiver base extraction.
- Grouped type members deliberately cannot project the outer `type` token without violating source-relative exact-range rules; downstream canonical text must consume the explicit signature together with the member projection.

## Downstream handoff

Phase 05 can instantiate `golang.New()` and provide the existing `chunk.ChunkRequest` with its injected segmentation policy. It receives ordered chunks, parser metadata, recoverable/non-indexable diagnostics, exact bodies/ranges, projections, and segment candidates without any filesystem, database, or provider dependency.
