# 04. TypeScript and TSX Chunker and Projections

- Status: `done` — implementation and main-agent completion validation accepted on 2026-08-15
- Prerequisite phase: `02-config-profiles-and-schemas`
- Downstream phases: `05-worktree-index-pipeline`, `06-fts-search`, `08-raw-embedding-lab`
- Parallel phase: `03-go-chunker`
- Design basis: `local-code-search-mcp-v1-design-r4.md` Sections 5 and 7.2

## Context Recovery Checklist

- Reopen the [implementation index](README.md), [execution guide](EXECUTION-GUIDE.md), and [status board](STATUS.md) before continuing after context compaction.
- Reopen Phase 02 artifacts for the shared chunk model, byte/line coordinates, projection validator, segment values, and index-profile versions. Confirm the offline TypeScript and TSX grammar artifacts selected in Phase 01.
- Re-check these critical invariants: `.ts` and `.tsx` grammar selection is explicit; stored bodies are exact contiguous slices; projections are ordered non-overlapping chunk-relative ranges; export status never promotes a non-callable variable; only directly identifier-bound function values receive the variable-function exception; type projections retain signatures but exclude nested callable bodies; the chunker performs no filesystem, Git, database, or Voyage access.
- Stop if TS/TSX grammar fallback would be implicit, a callable lacks a stable named identity, overload adjacency or ownership is ambiguous, a projection cannot exclude bodies with exact safe ranges, or Phase 02 shared types are being changed concurrently.
- Before pausing, update Section 11 evidence, Section 13 decisions, and [STATUS.md](STATUS.md). Record unresolved policy fixtures explicitly and keep the phase `planned` until deterministic output and projection evidence are complete.

## 1. Goal

Deterministically extract function, method, and type source chunks from one read of TypeScript or TSX source bytes. Normalize JavaScript's varied function expressions and TypeScript declarations into small v1 retrieval units without broadening exported const/var declarations themselves into retrieval units.

Each result contains:

- chunk kind, simple symbol, qualified symbol, and signature;
- original-file byte and line ranges plus exact `source_body`;
- projection ranges for FTS and canonical embedding input;
- a type projection that excludes method bodies nested in the type;
- statement/declaration boundaries and display ranges for long functions and types; and
- one domain output that hides TypeScript-versus-TSX grammar differences.

## 2. Scope and Non-Goals

### In scope

- `.ts` and `.tsx` files
- Named function and generator-function declarations
- Class and abstract-class methods and constructors
- Arrow functions and function expressions directly assigned to a module/file-scope identifier
- A fixed policy for treating arrow functions and function expressions directly assigned to a class-field identifier as method-like
- Class, interface, type-alias, and enum declarations
- Connecting function/method overload groups to implementations
- Bodyless function/method signatures in declaration files
- TypeScript declarations and function bodies containing JSX in TSX
- Excluding nested method and constructor bodies from type projections

### Out of scope

- Independent chunks or high-weight symbols for const/let/var declarations themselves
- Non-function initializers such as numbers, strings, and objects
- Independent chunks for arbitrary callbacks and immediately invoked functions
- Expanding object-literal methods as though the object were a type
- Namespace/module declarations as retrieval chunks themselves
- Import/export graphs, call graphs, type checking, and tsconfig path resolution
- Official support for JavaScript `.js` or `.jsx`
- FTS insertion, SQLite writes, or Voyage API calls

`export const handler = () => ...` is included because it is a named function value, not because it is a `const`. `export const timeout = 30` is not a chunk.

## 3. Prerequisites

- Phase 02 fixed the shared chunk model, coordinate rules, and profile-version access.
- Phase 01 packaged both TypeScript and TSX Tree-sitter grammars for offline use.
- Go and TypeScript chunkers use the same `Chunker` interface and projection validator.
- File enumeration explicitly passes whether a source is `.ts` or `.tsx`; the chunker never guesses a grammar from content.

## 4. Invariants

1. Parsing, source ranges, `source_body`, and projections all refer to the same immutable input bytes.
2. File byte ranges are zero-based and half-open; external line ranges are one-based and inclusive.
3. Stored `source_body` is an exact contiguous source slice. A projection is an ordered, non-overlapping list of ranges within it.
4. Function and method projections include the declaration and complete body.
5. A type `source_body` preserves the original contiguous declaration. Its projection excludes nested method and constructor bodies while retaining their signatures.
6. Export status never promotes a const/var into a chunk.
7. A variable-bound function becomes a named function chunk only when one function value is assigned directly to one identifier.
8. Never invent synthetic symbols for expressions without stable names, including anonymous callbacks, destructuring bindings, and computed properties.
9. When overload signatures connect to one implementation, do not create multiple source chunks that occupy duplicate result slots.
10. Segments use only complete AST-guaranteed boundaries such as statements, declarations, or JSX elements.
11. Identical source, grammar selection, and index profile produce identical ordered semantic output.
12. The chunker knows nothing about export graphs, the filesystem, SQLite, or generations.

## 5. Packages, Files, and Types to Implement

```text
internal/chunk/
  typescript/
    parser.go                   # TypeScript/TSX grammar adapter
    declarations.go             # function, method, and type extraction
    variable_functions.go       # named arrow/function-expression classification
    overloads.go                # signature groups and implementation connection
    symbol.go                   # lexical owners and qualified symbols
    signature.go
    projection.go               # type projections excluding callable bodies
    segments.go
    diagnostics.go
```

`internal/chunk/{lang,model,position,projection}.go` are Phase 02 completion artifacts. Phase 04 does not edit them and owns only the `typescript/` adapter.

In addition to types shared with the Go phase, this adapter needs:

- `GrammarVariant`: `typescript | tsx`
- `LexicalOwner`: class/interface and required lexical-nesting information
- `CallableDeclaration`: common intermediate representation for declarations, variable-bound expressions, methods, and constructors
- `OverloadGroup`: signatures and optional implementation with the same lexical owner, name, and static kind
- `TypeDeclaration`: common intermediate representation for class, interface, type alias, and enum
- `ExcludedBodyRange`: a method/constructor body removed from a type projection

Never return Tree-sitter nodes outside the package. Every result range and metadata value remains usable after tree lifetime ends.

## 6. Schema, API, and CLI Contracts

### 6.1 Chunk kinds and inclusion rules

`function`:

- function or generator declaration;
- module/file-scope `identifier = arrow_function`;
- module/file-scope `identifier = function_expression`; and
- when an overload group exists, one logical function chunk combining signatures and implementation.

`method`:

- named class/abstract-class method, getter, setter, or constructor;
- according to the fixed policy, an arrow function or function expression in a named class field; and
- bodyless interface and abstract-class method signatures always remain in the type projection, while independent method-chunk behavior is fixed together with overload/body policy.

`type`:

- class and abstract class;
- interface;
- type alias; and
- enum.

A namespace or module may contribute owner context but is not an independent `type` chunk. Exclude non-callable variable declarations regardless of export status.

### 6.2 Variable-bound functions

All conditions must hold:

- The binding name is one identifier.
- The initializer is directly an arrow function or function expression.
- The implementation does not traverse through wrapper calls, conditionals, or object properties.
- The declaration range is an exact source range containing modifiers/export and the variable declarator.

When one declaration statement contains several declarators, decide before implementation how each function declarator's exact range and shared modifiers are represented. Prefer declarator ranges so one chunk's `source_body` does not unnecessarily include unrelated declarators.

### 6.3 Symbols and qualified symbols

- File/module-scope function: include a lexical namespace when present; otherwise use a module-relative owner.
- Class method: `ClassName.Method`
- Static versus instance status should not split a useful display symbol unnecessarily, but remains in persistence identity metadata.
- Getter and setter display symbols remain the property name and differ by callable-kind metadata.
- Constructor: `ClassName.constructor`
- Type: `Namespace.TypeName` when a lexical namespace exists

Relative path and source range must always be available for persistence identity. Never use qualified symbol alone as a unique key.

Decide the v1 policy for anonymous default-export functions before implementation. If included, use a stable `default` display plus path/range identity; never insert a fake function name into source. If excluded, count it under explicitly unsupported declarations rather than emitting a diagnostic.

### 6.4 Overloads and declaration files

- Combine contiguous overload signatures and their implementation with the same owner/name into one logical chunk.
- `source_body` may use a contiguous range from the first signature through the implementation end.
- Projection includes every signature and the implementation body.
- A `.d.ts` function-signature group without an implementation may remain a searchable function chunk.
- Interface method signatures are part of the type projection by default. Creating independent method chunks as well duplicates results, so fix one policy before evaluation.
- If grouping is uncertain, never combine distant declarations by name alone.

### 6.5 Type projections and duplicate removal

A type `source_body` is the complete exact source of its class, interface, type alias, or enum. A class or abstract-class projection retains:

- declaration header, type parameters, extends, and implements;
- field/property declarations and the selected initializer policy;
- constructor, method, getter, and setter signatures; and
- the fixed decorator and modifier policy.

Remove only method and constructor body byte ranges. Remaining ranges must be source-ordered and non-overlapping. Prefer the complete declaration as the projection for interface, type alias, and enum because they contain no nested method bodies.

If a class-field arrow function becomes a separate method chunk, retain only its field signature and property name in the type projection and remove the function body to avoid duplicate indexing.

### 6.6 Long-chunk segments

- Function/method: top-level statements in the block and complete element/expression boundaries in a JSX return subtree
- Class/interface: member declarations
- Type alias: union, intersection, or object-type member boundaries
- Enum: member boundaries

Generate both segment projections and the contiguous display range covering each one. Never split by bytes in the middle of JSX text, a UTF-8 code point, a string/template literal, or a comment. Chunks remain complete semantic functions, methods, or types; segment packing follows AST statement/member boundaries toward the resolved 1024-byte target. An oversized AST unit remains whole. Evaluation may compare only 768-, 1024-, and 1536-byte segment targets.

### 6.7 CLI

Phase 04 adds no public CLI or MCP tool. Later lexical evaluation and the index pipeline invoke the same `Chunker` interface.

## 7. Config Usage and Change Impact

| Resolved value | Use | Impact of change |
| --- | --- | --- |
| `index.languages` | enable TypeScript and TSX chunkers | changes target set; local reindex |
| Executable-owned TypeScript chunker/grammar ID | TypeScript/TSX extraction and grammar dispatch | reparse every TypeScript/TSX file |
| Executable-owned projection ID | class method-body exclusion and similar rules | regenerate chunk projections, FTS, and inputs |
| `index.target_segment_bytes` | AST statement/member packing toward the 1024-byte target; semantic chunks remain whole | index-profile change; regenerate segments |
| Executable-owned canonical-text profile | downstream formatter | no effect on AST chunks; recompute canonical inputs/hashes, changing keys only when actual bytes change |

Manage function/method/type node names, export/modifier handling, and the supported-declaration set in one adapter-level group of named constants or queries. Do not duplicate range validation, segment packing, or line-conversion rules shared by Go and TypeScript.

## 8. Ordered Implementation Checklist

1. Implement a grammar adapter that selects `.ts` and `.tsx` explicitly.
2. Define callable and type intermediate forms that convert into the shared chunk model.
3. Extract named function and generator declarations.
4. Extract module/file-scope variable-bound arrow functions and function expressions.
5. Extract class methods, constructors, getters, and setters.
6. Fix v1 policies for class-field functions and anonymous default exports with fixture evidence.
7. Extract class, abstract class, interface, type alias, and enum as type chunks.
8. Group overload signatures and implementations by lexical adjacency and owner.
9. Implement bodyless callable policy for `.d.ts`.
10. Build simple and qualified symbols plus persistence-identity metadata.
11. Calculate signatures and exact source byte/line ranges.
12. Build type projections that remove only nested method bodies from type source bodies.
13. Collect statement, member, and JSX boundaries and build segment candidates and display ranges.
14. Run shared validators over ranges, projections, and result ordering.
15. Emit diagnostics that distinguish Tree-sitter error nodes from unsupported declarations.
16. Review every TypeScript/TSX fixture category and record chunker/projection versions.

## 9. Failure, Rollback, Concurrency, and Security

- The chunker writes no database. If any file result violates range invariants, exclude that file from the publication delta.
- Never auto-fallback between TypeScript and TSX grammars. If the supplied variant cannot parse safely, return an explicit error.
- Do not share a parser/tree across file processing unless the binding guarantees a safe pool.
- Propagate cancellation through parsing and long projection/segment loops.
- If syntax-error recovery crosses declaration ranges, do not return an incomplete chunk.
- Never follow source import paths, JSX URLs, or directives to open external files or the network.
- Diagnostics and logs contain no source body, API key, or complete query.
- Never execute or evaluate a computed property or dynamic name to create a string.

## 10. Validation Scenarios

Validate these fixtures and scenarios during implementation; writing this document adds no test code.

- Named, async, generator, and generic functions
- Exported and non-exported variable-bound arrow functions and function expressions
- Non-callable exported const/let/var declarations do not become chunks.
- Class instance/static/async/generator methods, constructor, getter, and setter
- Bodyless method signatures in abstract classes and interfaces
- Overload signatures and implementation combine into one logical result.
- Function-overload groups in `.d.ts`
- Class, interface, type alias, and enum with generics, extends, and implements
- Class type projection retains method signatures and removes bodies.
- Safe segment boundaries for a TSX component function with a long JSX return or body
- Nested callbacks and object-literal methods remain excluded according to policy.
- Fixed policies for anonymous default-export functions and class-field arrows
- Decorators, comments, multiline signatures, and semicolon-free syntax
- Byte and line accuracy for CRLF and multibyte UTF-8 source
- Safe declarations survive near parse errors while damaged declarations are excluded.
- Identical source/config produces deterministic ordered output.

## 11. Completion Evidence

### 2026-08-15 implementation evidence (pending main-agent validation)

- Adapter: `internal/chunk/typescript` implements the Phase 02 `chunk.Chunker` contract using the embedded Tree-sitter TypeScript and TSX grammars selected only by `typescript.New(chunk.TypeScript)` or `typescript.New(chunk.TSX)`. `ChunkerVersion` is `typescript-tsx-tree-sitter-0.23.2-jsdoc-class-fields-v1`.
- Supported chunks: named `function_declaration` and `generator_function_declaration`; top-level identifier-bound `arrow_function`, `function_expression`, and generator function values; named class methods, getters, setters, constructors, and class-field function values; and class/abstract-class/interface/type-alias/enum declarations. Ordinary variable values, nested callbacks, object-literal methods, computed names, and anonymous default-export expressions are excluded.
- Fixed policies: contiguous same-owner overload signatures combine with an implementation into one chunk; a top-level bodyless signature group remains one function chunk; interface and abstract/bodyless class signatures remain only in the containing type projection; identifier-bound class-field function values are method-like chunks and only their executable function-body ranges are excluded from the class type projection; anonymous default exports are excluded rather than assigned a synthetic name.
- Function-value chunks keep the binding, parameters, return annotation, and arrow/function token in `Signature`; their actual Tree-sitter `body` node supplies body projection and AST segmentation. Type signatures are normalized declaration headers ending at the semantic content start (`body` for class/interface/enum and `value` for type aliases), never a full declaration that would reintroduce member bodies.
- JSDoc rule: include only immediately preceding contiguous `comment` siblings whose gaps contain horizontal whitespace and at most one line break. This is part of the versioned chunker rule.
- Exactness fixtures cover TypeScript functions, generics, variable-bound arrow/function expressions, ordinary-variable exclusion, class methods/constructors/getters/setters, class-field functions, overload collapsing, interface/abstract bodyless non-duplication, class/interface/type-alias/enum chunks, TSX JSX bytes, CRLF/UTF-8 ranges, deterministic output, cancellation, invalid UTF-8, malformed-input recovery, and statement/member-boundary segmentation with later type segments excluding earlier member text.
- See [Phase 04 evidence index](evidence/phase-04/README.md) for commands, outcomes, deferred checks, and residual risks.

Before changing this phase to `done`, record actual results for:

- Supported declaration and node-kind table for TypeScript and TSX
- Included and excluded variable-bound-function examples
- Non-callable const/var exclusion results
- Policies for overloads, `.d.ts`, anonymous default exports, and class-field functions
- Example ranges showing nested bodies removed from a type projection
- Per-fixture chunk, symbol, range, projection, and segment summary
- Parse-error recovery results by TypeScript and TSX grammar
- Source-body byte equality and projection-invariant results
- Checks actually run and checks not run
- Known TypeScript/TSX syntax or grammar constraints

## 12. Downstream Handoff

Provide Phase 05 with:

- `.ts`/`.tsx` grammar dispatch and `Chunker` implementation;
- diagnostics distinguishing file-level success and failure;
- deterministic chunk ordering and index-profile version; and
- unsupported counts by declaration category.

Provide FTS and embedding phases with:

- simple and qualified symbols, including variable-bound functions;
- signature and exact `source_body`;
- type projections with duplicate method bodies removed;
- segment projections and contiguous display ranges; and
- ordered fields consumed by the canonical formatter.

## 13. Decision Log

| Decision | Status | Basis |
| --- | --- | --- |
| TypeScript source-chunk kinds | fixed: function, method, type | r4 v1 retrieval units |
| Exported const/var itself | excluded | Export status does not define a retrieval unit. |
| Named variable-bound arrow/function | included | Explicit contract for `export const handler = () => ...`. |
| Arbitrary callback or IIFE | excluded | Not a stable named unit and would cause chunk explosion. |
| Type kinds | fixed: class, interface, type alias, enum | Confirmed through v1 type fixtures. |
| Overload handling | fixed: logical signature-plus-implementation group | Avoid duplicate results while preserving signatures. |
| Independent interface-method chunk | fixed: exclude independent bodyless method chunks | Interface and abstract/bodyless class signatures remain in their parent type projection, avoiding duplicate result slots. |
| Class-field arrow | fixed: direct identifier-bound values are method-like | Emit a method chunk and exclude only its exact executable body range from the class projection. |
| Anonymous default-export function | fixed: excluded | It has no stable named identity in v1, so no synthetic `default` symbol is emitted. |
| Segment threshold and overlap | later evaluation values | Manage through config and the index profile. |
