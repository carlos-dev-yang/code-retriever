# 03. Go Chunker and Projections

- Status: `done` — implementation and main-agent completion validation accepted on 2026-08-15
- Prerequisite phase: `02-config-profiles-and-schemas`
- Downstream phases: `05-worktree-index-pipeline`, `06-fts-search`, `08-raw-embedding-lab`
- Parallel phase: `04-typescript-tsx-chunker`
- Design basis: `local-code-search-mcp-v1-design-r3.md` Sections 5 and 7.2

## Context Recovery Checklist

- Reopen the [implementation index](README.md), [execution guide](EXECUTION-GUIDE.md), and [status board](STATUS.md) before continuing after context compaction.
- Reopen Phase 02 artifacts for `Chunker`, `SourceChunk`, `ProjectionRange`, `SegmentCandidate`, byte/line coordinates, projection validation, index-profile access, and canonical-text formatting. Confirm the offline Go grammar artifact selected in Phase 01.
- Re-check these critical invariants: all ranges derive from one immutable source byte slice; `source_body` is an exact contiguous slice; projections are ordered, non-overlapping, chunk-relative half-open ranges; v1 emits only named functions, methods, and types; segmentation uses AST boundaries rather than arbitrary byte cuts; the chunker performs no database, filesystem, Git, or Voyage access.
- Stop if a declaration cannot produce safe exact ranges, receiver normalization is ambiguous, malformed input would change bytes through replacement, Phase 02 shared types are moving concurrently, or parser packaging requires a runtime download.
- Before pausing, update Section 11 evidence, Section 13 decisions, and [STATUS.md](STATUS.md). Record fixture gaps and keep the phase `planned` until deterministic output and range/projection evidence are complete.

## 1. Goal

Deterministically extract function, method, and type source chunks from one read of Go source bytes, while separating stored source from search and embedding projections.

Downstream phases must be able to consume each result without rereading the file or reinterpreting the Go AST:

- chunk kind, simple symbol, qualified symbol, and signature;
- original-file byte and line ranges plus exact `source_body`;
- ordered projection ranges for FTS and canonical embedding input;
- AST statement/declaration boundaries suitable for splitting long source chunks;
- each segment's projection and contiguous source display range; and
- distinction between recoverable parse errors and fatal errors that reject the file.

## 2. Scope and Non-Goals

### In scope

- Parse package declarations and named declarations from `.go` source.
- Top-level function declarations
- Method declarations with receivers
- Struct, interface, named-type, and alias-type declarations
- Individual type-spec handling within grouped `type (...)` declarations
- Qualified symbols incorporating package and method receiver
- Source-relative half-open projection ranges
- AST-boundary candidates for long functions, methods, and types
- Deterministic output order and diagnostics

### Out of scope

- Independent chunks or high-weight symbols for const/var declarations
- Independent chunks for anonymous function literals
- Call graphs, import resolution, or cross-package type resolution
- Build-tag-specific semantic type checking
- `go/packages` or compiler-type-checker symbol resolution
- FTS insertion, SQLite writes, or Voyage API calls
- Generated-file detection and ignore policy

Go chunking receives only file content. It knows nothing about repository traversal, current filesystem state, or Git metadata.

## 3. Prerequisites

- Phase 02 shared chunk-storage types and coordinate rules are fixed.
- Phase 01 proved offline packaging for the Go Tree-sitter grammar and binding.
- The index profile exposes Go chunker, projection, and segment versions.
- If segment size and overlap are not yet finalized, inject the named policy from `ResolvedConfig`; never copy numeric values into the chunker.

## 4. Invariants

1. Input is one immutable source byte slice. Parsing, hashing, and range slicing all refer to those bytes.
2. Byte coordinates are original-file, zero-based, half-open `[start, end)`. Line coordinates in external contracts are one-based and inclusive.
3. `source_body` is byte-for-byte identical to a contiguous input slice.
4. Projection ranges are zero-based, half-open, relative to `source_body`, source-ordered, and non-overlapping.
5. Function and method projections include the complete declaration and body.
6. Type projections include the declaration, fields, embedded types, and interface method signatures.
7. Go method bodies are not nested inside type declarations, so the chunker never invents body duplication between method and type chunks.
8. Go const/var declarations never become independent v1 chunks, even when they contain a function expression. Do not extend the explicit TypeScript variable-bound-function exception to Go.
9. Segment boundaries use only AST statement or declaration boundaries, never arbitrary byte counts.
10. The same source and index profile produce the same semantic chunk sequence and ordering before persistence IDs are assigned.
11. The chunker creates neither SQLite row IDs nor generations.
12. A declaration that cannot yield safe exact ranges is never emitted as a successful chunk.

## 5. Packages, Files, and Types to Implement

```text
internal/chunk/
  golang/
    parser.go               # Tree-sitter Go adapter
    declarations.go         # function, method, and type extraction
    symbol.go               # receiver, package, and qualified symbols
    signature.go            # signature extraction without function body
    projection.go           # projections by Go declaration kind
    segments.go             # statement/declaration boundary collection
    diagnostics.go          # parse and range error classification
```

`internal/chunk/{lang,model,position,projection}.go` are Phase 02 completion artifacts. Phase 03 does not edit them and owns only the `golang/` adapter.

Recommended shared type shapes:

- `Chunker.Chunk(context.Context, ChunkRequest) (ChunkResult, error)`
- `ChunkRequest`: relative path, language, immutable source bytes, and resolved index profile
- `ChunkResult`: ordered chunks, diagnostics, and parser metadata
- `SourceChunk`: kind, symbol, qualified symbol, signature, source range/body, projection, and segment candidates
- `SourceRange`: file byte start/end plus one-based line start/end
- `ProjectionRange`: chunk-relative byte start/end plus purpose
- `SegmentCandidate`: projection ranges, contiguous display range, and AST-boundary metadata
- `Diagnostic`: severity, code, source range, and safe-to-index flag

Do not make `SourceChunk` the same struct as the database model. The indexing application layer converts validated domain output into persistence rows.

## 6. Schema, API, and CLI Contracts

### 6.1 Chunk kinds

- `function`: named receiver-less `function_declaration`
- `method`: `method_declaration` with a receiver
- `type`: each named `type_spec`, including structs, interfaces, defined types, and aliases

`const_declaration`, `var_declaration`, `short_var_declaration`, and anonymous `func_literal` are not source chunks.

### 6.2 Symbols and qualified symbols

- Function simple symbol: declared function name
- Function qualified symbol: at least `packageName.Function`
- Method simple symbol: method name
- Method qualified symbol: at least `packageName.ReceiverBase.Method`
- Type simple symbol: type-spec name
- Type qualified symbol: at least `packageName.Type`

Normalize a receiver to its base named type by removing pointer markers, generic instantiation, and parentheses. If normalization is impossible, do not invent a string; exclude the declaration and emit a diagnostic.

Go permits multiple `init` functions in one package. Preserve `init` as the display symbol, but include path and source range in persistence identity. A qualified symbol alone is never a unique key.

### 6.3 Signatures

- Function and method signatures derive from the exact source slice before the body begins.
- Type signatures include a stable header containing the type name, type parameters, and alias or underlying-type declaration.
- Whitespace may be normalized for search display, but source-range calculation and canonical bodies use original bytes.

Decide associated doc-comment inclusion at implementation start and reflect it in the Go chunker version. If included, no other token may occur between comment and declaration, and the `source_body` start extends to the comment start. A projection must never reference comment bytes outside its source range.

### 6.4 Type projections

A Go type declaration stores its entire original source range as `source_body`. Its projection includes:

- the `type` keyword, type name, type parameters, and alias marker;
- struct fields and tags;
- embedded types; and
- interface method signatures and type elements.

Interface methods have no bodies, so their signatures remain in the type projection. A top-level method declaration is a separate method chunk and is not inside the type source range.

### 6.5 Long-chunk segments

- Functions and methods: statement boundaries in the top-level block
- Structs, interfaces, and types: field, method-spec, and type-element declaration boundaries
- A segment contains one or more complete boundary units.
- When a segment projection is discontinuous, compute the smallest contiguous `display_range` covering it.
- Receive threshold and overlap policy from the resolved index profile.
- Choose one shared embedding contract for chunks below threshold: either always generate one segment or use the chunk projection directly without a segment. Do not vary by language.

### 6.6 CLI

Phase 03 adds no public CLI. It exposes a package API so later evaluation tools can invoke the same `Chunker` interface. Debug output must not become a production command that bypasses stdout protocol or the database.

## 7. Config Usage and Change Impact

| Resolved value | Use | Impact of change |
| --- | --- | --- |
| `index.languages` | enable Go chunker | changes target set; local reindex |
| Executable-owned Go chunker ID | all Go extraction rules | reparse every Go file |
| Executable-owned projection ID | type/function projections | regenerate affected chunks, FTS, and inputs |
| `index.max_chunk_bytes`, `max_segment_input_bytes` | segment-boundary packing | index-profile change; regenerate segments |
| Executable-owned canonical-text profile | formatter after chunking | no effect on AST chunk ranges; recompute canonical inputs/hashes, changing keys only when bytes change |

Do not scatter Tree-sitter node-kind magic strings across files. Own them in one set of named adapter constants or one query definition. Use the shared implementation for coordinate conversion and range validation.

## 8. Ordered Implementation Checklist

1. Confirm the shared `Chunker`, source-range, projection, and diagnostic types.
2. Encapsulate immutable source bytes and Tree-sitter tree lifetime in the parser adapter.
3. Read the package clause and implement the error policy for missing or unsafe package names.
4. Extract top-level function declarations.
5. Implement method extraction and receiver-base normalization.
6. Extract each type spec from single and grouped type declarations.
7. Build function, method, and type signatures from original source ranges.
8. Validate byte and one-based line ranges and create exact `source_body`.
9. Build FTS/embedding projection ranges by type.
10. Collect function-block statement and type-declaration boundaries.
11. Use resolved segment policy to produce segment candidates and display ranges.
12. Fix output order by source order plus a stable secondary key.
13. Classify Tree-sitter error nodes and retain only safe declarations.
14. Review outputs for every Go fixture category and update chunker/projection-version decisions.

## 9. Failure, Rollback, Concurrency, and Security

- The chunker writes no database and has no internal rollback. Fully validate a file result before adding it to an index publication delta.
- Convert potentially panicking grammar or binding calls into typed errors that expose neither file path nor source body.
- Propagate cancellation into the parser and long segment-processing loops.
- Do not assume one parser or tree is goroutine-safe. Parallel files use independent parser state.
- Follow the Phase 02 malformed-UTF-8 policy and fail closed; never change byte ranges through replacement characters.
- If a range exceeds source bounds or projections overlap, reject the complete file result from publication.
- Diagnostics and logs contain no complete source body.
- Never use a file path or import string to perform additional filesystem access.

## 10. Validation Scenarios

Validate these fixtures and scenarios during implementation; writing this document adds no test code.

- Ordinary, variadic, and generic functions
- Value-receiver, pointer-receiver, and generic-receiver methods
- Struct, interface, defined type, alias, and grouped type declarations
- Interface embedded types and method signatures
- Multiple `init` functions become separate chunks without symbol collision.
- Const/var declarations and anonymous function literals do not become independent chunks.
- Byte and line ranges remain exact for CRLF source and multibyte UTF-8 identifiers or comments.
- The intended difference between type `source_body` and projection is preserved, and ranges remain source-relative.
- Long blocks segment only at statement boundaries, and display ranges are exact source slices.
- Define whether safe chunks survive when a syntax error lies outside them.
- A damaged range inside a declaration emits a diagnostic that blocks file publication.
- Repeating the same source/config produces the same ordered semantic output.

## 11. Completion Evidence

Before changing this phase to `done`, record actual results for:

- Supported Go declaration and node-kind table
- Excluded declarations and reasons
- Doc-comment inclusion policy and chunker version
- Receiver-normalization examples and failure policy
- Per-fixture chunk, symbol, range, projection, and segment summary
- Source-body byte equality and projection-invariant results
- Parse-error recovery results
- Checks actually run and checks not run
- Known Go syntax or grammar constraints

Hit rate and numeric segment thresholds are not completion conditions for this phase. Pass results forward as baseline inputs to lexical and retrieval evaluation.

### 2026-08-15 implementation handoff

- Adapter: `internal/chunk/golang` implements the Phase 02 `Chunker` contract with the offline embedded Go Tree-sitter grammar (`0.25.0`). `ChunkerVersion` is `go-tree-sitter-0.25.0-doc-comments-v1`.
- Supported node kinds: top-level `function_declaration`, `method_declaration`, and each `type_spec` or `type_alias` inside `type_declaration`. Functions produce `function` chunks, receiver-bearing declarations produce `method`, and type specs/aliases produce `type`.
- Exclusions: `const_declaration`, `var_declaration`, and anonymous function literals are never traversed as chunk roots.
- Doc comments: a directly preceding contiguous Go comment block is included when every comment-to-comment/declaration gap contains only horizontal whitespace and at most one line break. The chunk `source_body` and first projection therefore retain that semantic context. A blank line or intervening token prevents association.
- Grouped types: each group member owns its exact individual type-spec/alias span (and any directly attached member doc comment), so grouped chunks do not contain neighboring specs. A grouped member cannot include the group-level `type` token because projection ranges must remain source-relative and byte-exact; its display/search signature uses the stable synthetic `type Name ...` header.
- Receiver normalization: the first receiver type identifier in source order is the deterministic base identity. This maps `R`, `*R`, and `*R[T]` to `R`; an absent or erroneous receiver excludes the method and emits `GO_INVALID_RECEIVER`.
- Source/projection/segment checks: every emitted source body is copied from one original byte slice; byte ranges are zero-based half-open; line ranges use the Phase 02 CRLF/UTF-8-safe `LineIndex`; projections are validated source-relative/non-overlapping; segments either retain the complete projection or pack Tree-sitter statement/field/member units without byte cuts. A single AST unit exceeding the injected byte cap remains one valid oversize segment rather than being split.
- Parse policy: invalid UTF-8 fails closed. Unsafe declarations are excluded with a non-indexable diagnostic. Syntax errors outside an otherwise safe declaration produce a recoverable `GO_PARSE_ERROR` diagnostic with `safe_to_index=true`; a diagnostic overlapping an emitted chunk is non-indexable.
- Focused fixtures cover ordinary/generic functions, generic pointer receivers, structs/interfaces/aliases and grouped types, doc comments, const/var/anonymous exclusions, CRLF plus Unicode coordinates, deterministic results, cancellation, malformed-input recovery, and statement-boundary segmentation.

## 12. Downstream Handoff

Provide Phase 05 with:

- the `Chunker` API and deterministic output order;
- diagnostics distinguishing file-level success and failure;
- chunk, source, projection, and segment range contracts; and
- Go chunker/index-profile versions.

Provide FTS and embedding phases with:

- simple and qualified symbols plus signatures;
- exact `source_body`;
- FTS/embedding projection ranges;
- segment projections and display ranges; and
- ordered fields consumed by the canonical formatter.

## 13. Decision Log

| Decision | Status | Basis |
| --- | --- | --- |
| Go source-chunk kinds | fixed: function, method, type | r3 v1 retrieval units |
| Const/var chunks | excluded | Not a v1 target, regardless of export status. |
| Anonymous function literal | excluded | Not a stable named retrieval unit. |
| Grouped type handling | fixed: one chunk per type spec | Preserve each named type as an independent retrieval unit. |
| Method receiver representation | fixed: qualified symbol uses base named type | Avoid splitting retrieval keys by pointer or generic notation. |
| `init` identity | fixed: includes path/range; symbol alone is not unique | Go permits multiple `init` functions per package. |
| Doc-comment inclusion | fixed: directly attached contiguous comment blocks are included; `go-tree-sitter-0.25.0-doc-comments-v1` | Preserve useful semantic context while retaining exact source-relative ranges. |
| Segment threshold and overlap | later evaluation values | Keep them in config/profile without a numeric release gate. |
