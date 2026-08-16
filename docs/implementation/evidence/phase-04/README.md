# Phase 04 TypeScript and TSX Chunker Evidence

- Phase: `04-typescript-tsx-chunker`
- State: `complete; Revision 4 correction and main-agent validation accepted`
- Date: 2026-08-15

## Implemented contract

- `internal/chunk/typescript` is a stateless offline adapter over the embedded Tree-sitter TypeScript (`typescript`) and TSX (`tsx`) grammars. Construction explicitly selects the grammar; source content and file extension never trigger fallback.
- It consumes the unchanged Phase 02 `chunk.Chunker`, `ChunkRequest`, range/projection, segment, diagnostic, and validation contracts. Each call copies supplied source bytes, owns its parser/tree lifecycle, and performs no filesystem, Git, database, provider, or network access.
- At the original accepted boundary it emitted source-ordered chunks for named functions and generators, direct top-level identifier-bound arrow/function values, named class methods/constructors/getters/setters, method-like direct identifier-bound class-field function values, and class/interface/type alias/enum declarations. Ordinary const/let/var values, nested callbacks, object-literal methods, computed properties, destructuring bindings, and anonymous default exports were excluded.
- Contiguous overload signatures with the same owner, name, and modifier/static category combine with their implementation. A contiguous top-level declaration-signature-only group emits one function chunk. Interface and abstract/bodyless class signatures are retained only in their parent type projection and do not create duplicate method chunks.
- A directly preceding contiguous JSDoc/comment block is included only when every intervening gap has horizontal whitespace and at most one line break. Class type projections remove exact method/constructor bodies and only the executable body node of class-field function values, retaining field parameters and arrow/function tokens.
- Function-value signatures retain their binding plus parameter/return/arrow syntax, and their actual Tree-sitter body node drives body projection and AST segmentation. Type signatures end before semantic content (`body` for class/interface/enum; `value` for a type alias), preventing executable member bodies from leaking through a full-declaration signature. Source bodies are exact slices, line coordinates are Phase 02 one-based inclusive values, projections are ordered non-overlapping source-relative ranges, and segments use only Tree-sitter statement/member/expression boundaries. Invalid UTF-8 fails closed; syntax errors exclude unsafe declarations while preserving safe declarations outside the damaged range with deterministic diagnostics.

## Checks actually run

```text
gofmt -w internal/chunk/typescript
go test -count=1 ./internal/chunk/typescript
go vet ./internal/chunk/typescript
go build ./internal/chunk/typescript
git diff --check
```

Result: passed. The focused fixture suite covers TypeScript and TSX grammar dispatch, named/generic functions, variable-bound arrow/function expressions with retained signatures and actual-body segmentation, ordinary value exclusion, class methods and method-like field functions, type kinds and body-free type signatures, overload collapse, bodyless interface/abstract-method nonduplication, JSDoc association, exact TSX JSX/UTF-8/CRLF source ranges, deterministic output, cancellation, invalid UTF-8, malformed-input recovery, and AST statement/member-boundary packing, including a long class fixture proving later type segments do not reproject earlier members.

Main-agent completion validation additionally ran:

```text
gofmt -l internal/chunk/typescript
go test -count=1 ./internal/chunk/typescript
go test -count=1 -race ./internal/chunk/typescript
go vet ./internal/chunk/typescript
go build ./internal/chunk/typescript
go list -deps ./internal/chunk/typescript
git diff --check
```

Result: passed. Dependency inspection found no production store, lab, or embedding-client dependency. Main review also verified that later type segments project only the current member window and do not re-embed preceding type members.

## Checks not run

- No Phase 02 or Phase 03 test suite was rerun.
- No repository-wide test/vet/build, corpus selection/download/indexing, filesystem traversal, Git operation, database, indexer, FTS, MCP, provider, network, or paid API validation ran.

## Residual risks

- Validation uses the current local embedded grammar package on the Phase 01 darwin/arm64 CGO environment; other packaging targets remain later-phase work.
- Parse recovery is syntax-only and does not type-check declaration legality, module resolution, or TypeScript compiler configuration.
- The shared chunk model has no persistence field for getter/setter/static callable-kind metadata. Grouping preserves modifier/static distinctions to avoid overload merging, but persistence identity remains path/range based as required.
- Class-field handling is intentionally limited to direct identifier/private-identifier fields whose value node is an arrow/function expression; only the executable body is removed from its class projection, while computed and wrapped values remain unsupported by design.

## Downstream handoff

Phase 05 can instantiate `typescript.New(chunk.TypeScript)` or `typescript.New(chunk.TSX)` and provide an injected `chunk.ChunkRequest` segmentation policy. It receives exact ordered chunks, parser metadata, diagnostics, projections, and segment candidates without a second source read or any external dependency.

## Revision 4 correction reopened on 2026-08-16

Real react-hook-form corpus evidence showed that the historical anonymous
default-export exclusion removed 57 production function bodies from the
semantic-parent universe. Fifty-one were in files with no other semantic
parent; six were in files that already exposed a type parent. The user accepted
the narrow correction: top-level
anonymous default-export function-like declarations use deterministic
path-derived values in the existing `symbol` and `qualified_symbol` retrieval
fields. No alias field, database column, FTS wire, MCP field, or evaluation
schema is added; exact source identity remains path + indexed content hash +
byte range. This boundary also repairs real-corpus overload grouping across
associated comments and recognizes any bodyless named function declaration as
an overload signature.

The current executable IDs are:

```text
typescript chunker: typescript-tsx-tree-sitter-0.23.2-jsdoc-class-fields-path-defaults-overloads-v2
index chunker version: 2
```

The single focused Phase 04 boundary validation passed:

```text
go test -count=1 ./internal/chunk/typescript ./internal/config ./internal/profile ./internal/index
go test -count=1 -race ./internal/chunk/typescript
go vet ./internal/chunk/typescript ./internal/config ./internal/profile ./internal/index
go build ./internal/chunk/typescript ./internal/config ./internal/profile ./internal/index
go list -deps ./internal/chunk/typescript
gofmt -l internal/chunk/typescript internal/config/constants.go
git diff --check
```

No store, lab, or embedding-client dependency entered the TypeScript adapter.
The provider-free generation-3 full reindex then proved the real-corpus
handoff:

- chi: 78/78 files rebuilt, 452 chunks, 621 segments, zero index errors;
- react-hook-form: 237/237 files rebuilt, 322 chunks, 492 segments, zero index
  errors;
- 57 path-labeled production functions use the accepted existing-field rule;
- files without a semantic parent fell from 179 to 128;
- `useWatch`, `insert`, and `mockZodResolver` each have exactly one function
  parent; and
- database source-body versus indexed file-span mismatches are zero in both
  corpora.

No provider, API key, network operation, document/query embedding, repository-
wide test suite, or paid action ran at this boundary.
