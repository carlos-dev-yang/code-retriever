# Relation-Aware Code Context Retrieval: Research and Decision Context

- Status: non-normative design research and conversation-continuity record
- Date: 2026-08-17
- Scope: the measured chi/React Hook Form retrieval gaps, relation-aware code
  context, online approaches, and the next bounded experiment
- Current product baseline: exhaustive `1024/int8` dense retrieval, with FTS as
  a separate lexical control
- Authority boundary: this document does not change the production schema,
  Phase 11 search behavior, MCP contract, frozen calibration labels, or phase
  status. A product change still requires the normal design/phase reconciliation
  procedure.

## 1. Why this document exists

The current conversation moved from codec and retrieval calibration into a
more fundamental code-understanding question:

> cidx can already locate plausible code from a natural-language question, but
> it lacks enough structural evidence to say why a particular candidate is the
> answer and which related parents must be read together.

The owner summarized the concern in practical terms: if cidx only returns
plausible neighboring code and the caller must then use `rg` to discover the
actual caller, callee, type definition, or value path, its advantage over
traditional search becomes unclear.

This note preserves:

1. the current implementation and measured baseline;
2. the exact failure examples that motivated the discussion;
3. the distinction between natural-language localization and structural proof;
4. the online systems and techniques reviewed;
5. the conclusion about Zoekt;
6. the bounded architecture and experiment currently recommended;
7. the decisions that remain deliberately unresolved.

## 2. Current cidx baseline

The current product path is intentionally small:

```text
source files
-> Tree-sitter parse
-> semantic parent chunks (function/method/type)
-> AST-aware embedding segments
-> exhaustive int8 dense scan and/or FTS5/BM25
-> segment-to-parent collapse
-> optional flat RRF in the accepted historical serving path
-> rank-invariant source-body packaging
```

Relevant implementation boundaries are:

- [`internal/chunk/model.go`](../../internal/chunk/model.go): declaration,
  parent range, signature, body, projections, and embedding segments;
- [`internal/index/service.go`](../../internal/index/service.go): persists each
  parent's own symbol and qualified symbol, but no resolved cross-parent
  occurrences;
- [`internal/store/schema.go`](../../internal/store/schema.go): chunks, FTS,
  segments, and vectors, but no call/reference/type-relation facts;
- [`internal/search/vector_scan.go`](../../internal/search/vector_scan.go):
  exhaustive active-profile vector scoring;
- [`internal/search/service.go`](../../internal/search/service.go): FTS/dense
  orchestration and result packaging;
- [`internal/search/rrf.go`](../../internal/search/rrf.go): flat reciprocal-rank
  fusion of FTS and dense parent rankings.

Tree-sitter is therefore already used for code-aware parent and segment
boundaries. It is not currently used as a complete occurrence graph, and it
cannot by itself prove the semantic target of an identifier across scopes,
imports, aliases, receiver types, overloads, or re-exports.

## 3. Frozen evidence that constrains the discussion

The current 32-query chi/RHF calibration set is frozen under
`OWNER_ADOPTED_DUAL_AI_REVIEW` with permanent
`NO_INDEPENDENT_HUMAN_REVIEW`. It is calibration evidence, not confirmation or
promotion evidence. The authoritative record is
[`dual-ai-calibration-freeze-r4.md`](evidence/phase-07/dual-ai-calibration-freeze-r4.md).

### 3.1 Aggregate results

| Lane | Complete@5 | Hit@5 | Complete@20 | Hit@20 |
| --- | ---: | ---: | ---: | ---: |
| Dense `1024/int8` | 30/32 | 30/32 | 31/32 | 32/32 |
| FTS | 17/32 | 19/32 | not opened | not opened |
| Equal RRF | 24/32 | 26/32 | not opened | not opened |
| FTS1:dense2 RRF | 28/32 | 29/32 | not opened | not opened |

The bounded RRF probes were rejected because they regressed already complete
RHF cases and worsened reviewed evidence composition. Aggregate compensation
is not allowed. The accepted calibration direction is:

- retain dense `1024/int8` as the retrieval-quality baseline;
- keep FTS as a separate lexical control;
- stop RRF weight tuning unless a genuine structural change or new frozen
  experiment justifies reopening it;
- do not interpret this result as proof that lexical retrieval can never help.

Dense `Hit@20 = 32/32` is especially important. Broad semantic localization is
already strong on this calibration set. The remaining loss is mostly exact
relation resolution, multi-parent evidence completion, and context assembly.

### 3.2 Exact representative cases

#### G09: chi `RealIP` and `realIP`

The required evidence is two separate semantic parents:

```text
middleware.RealIP
    CALLS
middleware.realIP
```

Exact frozen dense observations:

- ranks 1-7 are all grade 0;
- misleading `walkXFF` is rank 5;
- `middleware.RealIP` is rank 8 and grade 2;
- `middleware.realIP` is absent through rank 20.

The source directly calls `realIP` from `RealIP`. See the pinned upstream
source at
[`middleware/realip.go`](https://github.com/go-chi/chi/blob/8b258c7bb28f97a5f2a856ff7ef962578fec9215/middleware/realip.go#L28-L53).

This is not recoverable by expanding only the dense top five. It requires a
bounded internal seed frontier that can use rank 8 without exposing every
top-20 neighbor.

#### X08: RHF `FormState` and `FormStateProps`

Exact frozen dense observations:

- `useFormState` is rank 1, grade 1;
- `FormState` is rank 2, grade 1;
- `UseFormStateProps` is rank 3, grade 0;
- required `FormStateProps` is rank 13, grade 2.

`FormState` directly refers to `FormStateProps` in its signature. See the
pinned source at
[`src/formStateSubscribe.tsx`](https://github.com/react-hook-form/react-hook-form/blob/371432c39271aab739358d19c406793771565ab3/src/formStateSubscribe.tsx#L11-L31).

This case can be recovered from a top-five parent if a TypeScript-resolved
type-reference edge is available.

#### T09 and T10: regression and role-quality guards

T09 and T10 are already dense-complete at top five. They are not missing cases
that can "gain completeness."

- T09 protects cross-file/import-export type identity: `createFormControl`
  consumes the `Control` API through the RHF type surface.
- T10 protects same-file recursive type relations between `PathImpl` and
  `PathInternal`.

They should be used to detect relation-role or packaging regressions rather
than counted as new gains.

## 4. The actual system gap

The gap is not one undifferentiated search problem. It has four ordered parts.

### 4.1 Candidate discovery and semantic localization

Question answered:

> Which parts of the repository are plausibly related to this natural-language
> intent?

Current dense retrieval performs this well enough to find a relevant parent in
the fixed top 20 for all 32 calibration cases.

### 4.2 Relation resolution

Question answered:

> What exact declaration does this occurrence call, reference, implement, or
> belong to?

This requires language semantics. A text-name coincidence or a unique nearby
identifier is not sufficient proof.

### 4.3 Evidence completion

Question answered:

> Which related semantic parents must be read together to answer the question?

Examples include caller plus callee, consumer plus type contract, interface
plus implementation, and two mutually recursive type definitions.

### 4.4 Context packaging

Question answered:

> Which source should be placed in the bounded response, in what role, without
> displacing already useful primary evidence?

A relation graph can find a valid neighbor and still fail the product if a
naive policy floods the result with adjacency or discards the source parent
needed to understand the relation.

The concise model is:

```text
natural-language retrieval = localization
compiler-backed relations  = structural facts
evidence completion         = answer support
packaging                   = bounded delivery
```

## 5. Why usage context matters

The owner's observation that a function's value is often clearer in its uses
than in its definition is materially relevant.

A definition usually states:

```text
what this function can do
```

A callsite often supplies:

```text
when it is used
which input/receiver is supplied
which condition guards it
where its result is assigned or returned
which larger operation depends on it
```

This means a useful relation model should not reduce a call to a bare
`caller_id -> callee_id` pair. The callsite is first-class evidence:

```text
CallsiteFact
  source_parent
  source_occurrence_range
  relation_kind
  resolved_target_symbol
  resolved_target_parent
  receiver_range (optional)
  argument_ranges[]
  enclosing_control_or_assignment_role (optional)
  generation and file/content proof
  resolver/toolchain fingerprint
```

The same fact supports both directions:

- outgoing: what does this parent call or refer to?
- incoming: where and under what local context is this target used?

Static analysis can prove the identity, callsite, arguments, and surrounding
syntax. It cannot prove human or business intent. Natural-language retrieval
and the final answer layer interpret those hard facts as "why" the call exists.

## 6. Online approaches reviewed

The online review used primary project documentation and source where
possible. No single system solves natural-language relevance, semantic
resolution, execution paths, and context packaging as one operation. Mature
systems separate these responsibilities.

### 6.1 Text and symbol search: `rg` and Zoekt

[Zoekt](https://github.com/sourcegraph/zoekt) is a fast trigram-based source
text search engine. It supports substring, regexp, boolean, repository, file,
language, and symbol-oriented queries and uses code-related ranking signals.
Symbol data can be augmented with ctags.

It is well suited to:

- fast exact or partial identifier lookup;
- regexp and substring search;
- repository-scale candidate discovery;
- code-aware text ranking.

It does not establish compiler-resolved caller/callee identity, imported type
identity, value flow, or answer-complete evidence bundles. Therefore Zoekt can
be a useful future candidate generator but does not close the measured cidx
gap. Adding it now would improve the `rg` side of the system before the
relation/evidence hole is addressed.

### 6.2 Syntax-derived repository maps: Aider

[Aider's repository map](https://aider.chat/docs/repomap.html) extracts
definitions and references with Tree-sitter, builds a file dependency graph,
and uses graph ranking to choose important definitions under a token budget.
Its documentation explicitly prioritizes identifiers frequently referenced by
other parts of the repository.

Transferable idea:

- usage/reference structure is a useful context-importance signal.

Limit:

- graph importance is not query-specific relevance proof;
- syntax/name-derived edges may not have compiler-level target identity.

Graph centrality or PageRank is therefore deferred until exact relation facts
and evidence quality are proven.

### 6.3 Precise navigation: LSP, SCIP, and Stack Graphs

[Sourcegraph Code Navigation](https://sourcegraph.com/docs/code-navigation)
separates search-based navigation (text plus syntax heuristics) from precise
navigation that uses compile-time information.

[SCIP](https://github.com/scip-code/scip/blob/main/scip.proto) records source
occurrences, symbol identities and roles, relationships such as
implementation/type-definition, and enclosing source ranges. Those primitives
support definition, reference, implementation, outline, and call-hierarchy
features.

[GitHub Stack Graphs](https://github.blog/open-source/introducing-stack-graphs/)
models language name-binding rules so a reference-to-definition relation is a
valid graph path rather than a text-name match. It is valuable where a build or
full compiler invocation is unavailable.

For the current Go and TypeScript/TSX scope, native semantic authorities are
smaller and more direct than inventing new language rules:

- Go: pinned `go/packages` plus `go/types`;
- TypeScript/TSX: TypeScript `Program` plus `TypeChecker`.

SCIP remains a credible later interoperability format when more languages or
external precomputed indexes are needed. It is not required for the first
chi/RHF diagnostic.

### 6.4 Code fact and cross-reference stores: Glean and Kythe

[Meta Glean](https://engineering.fb.com/2024/12/19/developer-tools/glean-open-source-code-indexing/)
stores source-code facts such as symbol locations, calls, cross-references,
call/type hierarchies, and source/target spans. Meta uses it for code browsing,
search, documentation, find-references, call hierarchy, and cross-language
navigation.

[Kythe's callgraph model](https://kythe.io/docs/schema/callgraph.html) connects:

```text
caller semantic object
  <- childof - callsite anchor - ref/call -> callee semantic object
```

It also records complications such as declaration/definition completion and
overrides. This callsite-centered representation is the strongest direct
model for cidx's immediate need.

Transferable design:

- store the exact occurrence and semantic endpoints;
- retain the enclosing caller;
- derive incoming and outgoing navigation from one fact;
- bind facts to the program/revision in which resolution occurred.

cidx does not need Glean or Kythe's full infrastructure to adopt this data
shape.

### 6.5 Program-path analysis: CodeQL and Joern

[CodeQL path queries](https://codeql.github.com/docs/writing-codeql-queries/creating-path-queries/)
model source-to-sink data flow and expose the intermediate path steps. They
answer deeper questions such as how a value enters, is transformed, and
reaches a sensitive operation.

[Joern's Code Property Graph](https://docs.joern.io/code-property-graph/)
combines syntax, control flow, and intra-procedural data flow in a directed,
typed property graph that can be extended with further overlays.

These systems demonstrate the next depth after name/call/type resolution, but
they are not the minimum justified step for G09 and X08. Full control/data-flow
analysis should be opened only after a frozen case proves that exact one-hop
relations still fail because the answer depends on a value or control path.

### 6.6 Runtime execution context: OpenTelemetry

[OpenTelemetry traces](https://opentelemetry.io/docs/concepts/observability-primer/)
record the actual parent/child path taken by an observed request across
operations and services.

Runtime evidence can answer which path was actually exercised, but it is an
optional future source because it requires a runnable, instrumented program
and sees only executed paths. It cannot replace the default static index for
arbitrary local repositories.

## 7. Review with kb-guide, ChatGPT, and Grok

The three reviewers received the current implementation shape, frozen metrics,
exact G09/X08/T09/T10 ranks, pinned source relations, rejected RRF/Zoekt/HNSW
directions, and the constraint that labels must not influence runtime
selection.

### 7.1 Consensus

All three converged after the exact-rank correction on these points:

1. the dominant gap is relation resolution plus evidence completion, with
   context packaging as a required final stage;
2. use a fixed dense top-20 internal seed universe while protecting dense
   top five as the only primary results;
3. Tree-sitter supplies occurrence/range extraction, not semantic authority;
4. use `go/packages`/`go/types` and TypeScript `Program`/`TypeChecker` now;
5. never certify a text-name coincidence as a semantic relation;
6. emit structural context as `related_evidence`, not an equal RRF voter or
   separate user-visible ranked list;
7. keep primary dense identities, order, scores, and byte allocation unchanged;
8. use one hop, uniquely resolved in-corpus targets, and a small global cap;
9. keep FTS separate, defer SCIP/full CPG/Zoekt/HNSW and further fusion tuning.

### 7.2 Material correction to the first proposal

One initial proposal kept a rank-6-to-20 anchor hidden and exposed only the
resolved target. That cannot complete G09 because both `RealIP` and `realIP`
are required and neither is a protected top-five relevant parent.

The stable unit must be a relation bundle:

```text
RelatedEvidenceBundle
  anchor dense rank and parent
  relation fact and source occurrence
  resolved target parent
  relation/evidence roles
  generation, content, and resolver proof
  body status/omission reason
```

- If the anchor is already primary, only a missing target consumes an added
  parent slot.
- If the anchor is outside primary, both the anchor and target may be required
  additions.

### 7.3 Remaining reviewer disagreement

ChatGPT recommended advancing the structural concept when at least one of G09
or X08 becomes complete with zero regressions. kb-guide and Grok required both
before architectural adoption.

The reconciled interpretation is:

- diagnostic viability: at least one case becomes complete through a verified
  relation and the other receives explicit first-loss attribution;
- product-architecture advancement: both G09 and X08 become complete, all 30
  previously dense-complete cases remain complete, and evidence quality does
  not regress.

This avoids declaring the idea worthless after one partial diagnostic success,
while preventing production adoption from a single easy same-file case.

## 8. Recommended bounded experiment

The next experiment should be an evaluation-only semantic relation sidecar. It
must not initially change the production schema, search result order, MCP
surface, or provider usage.

### 8.1 Fixed controls

- frozen 32 queries, labels, required groups, and hard negatives;
- pinned chi/RHF corpora and current indexed generation;
- immutable dense `1024/int8` rankings;
- fixed dense top-20 seed frontier;
- immutable dense top-five primary results and bodies;
- no FTS, RRF, new embedding, query rewrite, or label-guided runtime selection;
- relations limited initially to `CALLS`, `TYPE_REF`, and `MEMBER_OF`;
- one hop only;
- uniquely resolved, in-corpus target parents only.

### 8.2 Relation authority and provenance

Use Tree-sitter to locate occurrences and enclosing semantic parents. Resolve
targets with pinned Go or TypeScript compiler semantics. Each fact records:

- corpus, revision, file identity, and content hash;
- source parent and exact occurrence range;
- relation kind and resolved target identity/range;
- index generation and semantic-parent mapping version;
- resolver implementation/toolchain/config fingerprint;
- resolution outcome: `RESOLVED_UNIQUE`, `UNRESOLVED`, `AMBIGUOUS`,
  `OUT_OF_CORPUS`, or `PARENT_MAPPING_FAILED`.

Only `RESOLVED_UNIQUE` becomes an admissible structural fact. Every other
outcome remains in diagnostic denominators.

### 8.3 Two stages inside one experiment

#### Stage A: reachability inventory

Enumerate all verified one-hop relation bundles from the fixed top-20 seeds.
Do not package or rank them yet.

This answers:

- Is G09's `RealIP -> realIP` relation represented and parent-mapped?
- Is X08's `FormState -> FormStateProps` relation represented and
  parent-mapped?
- What fraction of occurrences resolve uniquely?
- Which failures come from occurrence extraction, language resolution, or
  parent mapping?

Frozen labels are used only for offline measurement after generation. They do
not change the generated relation set.

#### Stage B: bounded label-blind admission and packaging

Protect dense top five, then select at most one relation bundle and add at most
two previously absent parent bodies. The concrete `2 x 1,024-byte` structural
slot proposal is an experiment control, not a product default.

The frozen diagnostic ordering is:

1. compiler-resolved relation kind: `TYPE_REF`, then `CALLS`, then
   `MEMBER_OF`;
2. anchor dense rank;
3. target dense rank, with absence from top 20 ordered last;
4. occurrence byte;
5. stable anchor and target identity.

The first diagnostic does not split calls into direct and value-flow roles.
That distinction is not an authoritative fact in the current resolver and
would otherwise turn a deterministic rule into an unproved heuristic.

This ordering must be frozen before replay and may not consult relevance
grades, required groups, query ID, or result outcome.

The selection policy is not assumed correct. If G09 is relation-reachable but
another bundle consumes the cap, record `RELATION_ADMISSION` as the first loss
rather than retuning the order around G09.

### 8.4 First-loss sequence

```text
DENSE_TOP20_LOCALIZATION
-> OCCURRENCE_EXTRACTION
-> LANGUAGE_RESOLUTION
-> TARGET_PARENT_MAPPING
-> RELATION_REACHABILITY
-> RELATION_ADMISSION
-> BUNDLE_PARENT_CAP
-> RELATED_BODY_PACKAGING
-> ANSWER_EVIDENCE_COMPLETION
-> ASSISTANT_USE (NOT_OBSERVED in the first provider-free experiment)
```

This sequence extends the current evaluation vocabulary for the diagnostic; it
does not become a normative wire change until the experiment justifies the
layer.

### 8.5 Measurements

Primary measurements:

- required-group and complete-evidence changes;
- G09 and X08 reachability and final bundle admission;
- byte-identical primary top-five result/body proof;
- no loss on the 30 dense-complete cases;
- explicit T09/T10 role-quality regression checks;
- added direct/support/grade-0 parent counts;
- reviewed hard-negative exposure;
- unique, unresolved, ambiguous, out-of-corpus, and parent-map denominators;
- relation facts and bundle counts per query;
- added parents, bytes, and local latency;
- complete, partial, or omitted related bodies with omission reason.

Hit@K remains secondary because the structural bundle is not a new flat ranked
lane.

### 8.6 Gates

Diagnostic viability requires:

- primary top five byte-identical;
- at least one of G09/X08 evidence-complete by verified relation;
- zero new reviewed hard-negative structural attachment;
- zero loss of an already complete required group;
- explicit first-loss for the still-incomplete case.

Product-architecture advancement requires:

- both G09 and X08 evidence-complete;
- all 30 dense-complete cases remain complete;
- T09/T10 remain complete and role-correct;
- no ambiguous relation emitted as fact;
- no evidence-quality regression;
- bounded parent, byte, and latency observations acceptable under a
  separately frozen product budget.

If Stage A cannot reach either missing target, stop this approach. If Stage A
reaches them but Stage B cannot admit them without regressions, the remaining
gap is selection/context policy rather than parsing or resolution.

## 9. Recommended data shape if the experiment succeeds

Do not begin with a general graph database. The minimum durable fact model is
an occurrence table plus semantic endpoints, from which both incoming and
outgoing relationships can be derived.

Conceptually:

```text
relation_occurrence
  relation_id
  relation_kind
  source_parent_id
  source_path/content_hash/start/end
  target_symbol_identity
  target_parent_id
  resolver_fingerprint
  generation

related_evidence_bundle (request-local derived view)
  anchor_parent_id
  relation_id
  target_parent_id
  anchor_role
  target_role
  body allocation/status
```

Likely relation directions:

- `CALLS` and derived `CALLED_BY`;
- `TYPE_REF` and derived `REFERENCED_BY`;
- `MEMBER_OF` and, only when useful, derived member lookup;
- implementation/override relations later when a measured cohort requires
  them.

Control flow, data flow, transitive graph expansion, dynamic dispatch
enumeration, centrality ranking, and runtime traces remain later overlays, not
requirements for the first relation fact store.

## 10. Approaches rejected or deliberately deferred

### Not the next gap-closing step

- Zoekt or full-body/identifier trigrams;
- another BM25 implementation;
- further FTS/RRF weight tuning;
- HNSW as an accuracy fix for an already exhaustive dense scan;
- a flat structural ranking lane or structural RRF vote;
- unbounded graph traversal;
- LLM-guessed identifiers or relations;
- per-query exceptions based on the frozen calibration outcomes.

### Credible later options

- SCIP as a portable multi-language occurrence/index interchange;
- Stack Graphs for languages without suitable native semantic tooling;
- callsite graph centrality after relation precision is established;
- CodeQL/CPG-style control or data flow after a measured path-dependent miss;
- OpenTelemetry/runtime evidence when an approved runnable project exposes it;
- Zoekt when a later corpus proves a lexical candidate-discovery or scale gap;
- assistant-use evaluation after body packaging is observable.

## 11. Decisions still open for the next conversation

The following are intentionally not fixed by this research note:

1. **Incoming usage scope.** Whether the first sidecar records only facts needed
   for the fixed outbound G09/X08 diagnostic, or records both incoming and
   outgoing views from the same callsite/type occurrence. The recommended data
   model supports both; serving exposure remains separate.
2. **Callsite context boundary.** Whether initial evidence includes only the
   containing semantic parent and exact occurrence or also argument snippets,
   the nearest guard, assignment, or return role.
3. **Bundle admission order.** The proposed label-blind order is a challenger,
   not a proven product rule. Stage A reachability must remain visible even
   when Stage B selection fails.
4. **Byte budget.** `2 x 1,024 bytes` is an experiment control. It is not tied to
   the 1,024-byte embedding-segment target or the MCP inline ceilings.
5. **Production wire shape.** `related_evidence` is the preferred stable model,
   but no MCP/schema change is authorized before the sidecar proves value.
6. **Confirmation scope.** The exposed 32-case calibration set cannot promote
   the architecture. A later unexposed confirmation set must validate any
   selected policy.
7. **Depth beyond one hop.** Transitive calls and data/control flow remain
   prohibited until a measured case demonstrates that one verified hop is
   insufficient.

## 12. Usage-oriented function graph refinement

The owner proposed a more concrete refinement after the initial relation
sidecar discussion:

> Individual functions and individual relation edges are not enough. Build a
> usage-oriented structure from higher-level functions down through the
> functions they use, then let an embedding hit enter that structure at a
> function node and move upward or downward to recover context.

This is consistent with the online review and makes the intended composition
clearer. The useful unit is not an isolated AST node or an isolated
`caller -> callee` fact. It is a short, semantically resolved usage path that
explains how a candidate participates in a larger operation.

### 12.1 Store a graph; derive a query-specific tree

The durable structure cannot literally be one global tree:

- one function may have several callers;
- several entry points may converge on one helper;
- recursive functions create cycles;
- callbacks and interface dispatch may have several possible targets;
- a repository usually has many roots rather than one application root.

The durable authority should therefore be a directed typed graph. A bounded
tree, forest, or path is a request-local projection from that graph.

```text
durable SQLite graph

HTTP handler A ----+
HTTP handler B ----+----> validate ----> normalize
background job C --+

query-local projection for a hit on validate

HTTP handler A
  -> validate
     -> normalize
```

This preserves all verified relations without loading or presenting the whole
graph. It also prevents a query-specific parent choice from becoming a second
and lossy persistent authority.

### 12.2 Function-level nodes are a sufficient initial scope

The current semantic parents already identify named functions, methods, and
types with exact source ranges. That makes function-level graph nodes a
practical initial boundary:

- Tree-sitter maps a callsite to its enclosing function/method parent;
- Go or TypeScript semantic resolution maps the callsite to the exact target
  function, method, or type parent;
- the existing chunk/body identity supplies the source evidence;
- deeper statement/variable graphs are unnecessary until a measured case
  requires data or control flow.

The initial graph does not need every possible program-analysis feature. Its
small set of composable primitives is:

1. `CALLS` / derived `CALLED_BY`;
2. `TYPE_REF` / derived `REFERENCED_BY`;
3. `MEMBER_OF`, with implementation/override added only when measured;
4. exact callsite/occurrence range and enclosing parent;
5. optional local callsite roles such as argument, guard, assignment, or
   return context.

Each primitive alone has limited answer value. Their composition with the
embedding localization supplies the useful correction:

```text
embedding: find the likely operation or usage neighborhood
graph: prove how the functions/types are connected
path selection: choose the coherent usage chain
packaging: deliver the minimal source evidence for that chain
```

### 12.3 Build it during free local indexing, not embedding

The relation graph should not depend on Voyage document capture or vector
materialization. It can be constructed during the free local index path:

```text
changed source files
-> Tree-sitter parse and semantic-parent extraction
-> occurrence/callsite extraction
-> compiler/type-checker target resolution
-> generation-bound graph delta publication

independent optional path
-> canonical segment text
-> document embedding
-> int8 serving vector publication
```

This separation matters because:

- FTS-only and unembedded projects can still use structural navigation;
- graph changes follow source/index generations rather than paid-vector state;
- changing serving dimension does not rebuild code relations;
- provider failure cannot make structural facts unavailable;
- incremental source changes need only recompute affected occurrence facts and
  their incident edges.

The initial full index may walk the complete eligible repository once to build
the first graph. Later indexes publish graph deltas under the same atomic
generation boundary as chunks and FTS.

### 12.4 SQLite is sufficient; the full graph need not be resident

The graph remains SQLite authority rather than a permanent in-memory graph.
A minimal storage shape is:

```text
relation_occurrences
  generation
  relation_kind
  source_parent_id
  source_file_id/content_hash/start/end
  target_symbol_identity
  target_parent_id
  resolver_fingerprint

indexes
  (generation, source_parent_id, relation_kind)
  (generation, target_parent_id, relation_kind)
```

The first index supports downward/outgoing traversal. The second supports
upward/incoming traversal. A request materializes only the bounded neighborhood
of the dense seeds, so memory is proportional to the selected path rather than
the repository graph.

As with the rest of search, one query must observe one committed generation.
No relation from a newer generation may be combined with an older chunk body or
vector result.

### 12.5 Embedding-guided graph traversal

The current dense result is the graph entry point, not the final structural
answer.

```text
natural-language query
-> dense top-20 semantic-parent localization
-> map each selected parent to its graph node
-> inspect bounded incoming and outgoing verified edges
-> construct candidate usage paths
-> select one small coherent path or relation bundle
-> preserve dense top-five primary results
-> attach path members as related evidence
```

Direction depends on where dense retrieval lands:

- if it finds a high-level caller, traverse downward to the implementation or
  type contract;
- if it finds a helper/definition, traverse upward to callers and entry-point
  context;
- if it finds a consumer, traverse through the occurrence to the exact
  contract definition;
- if it finds one member of a recursive or paired definition, attach the
  directly referenced partner.

This is the main expected accuracy correction. Dense similarity supplies the
query-specific semantic signal that a generic call graph lacks. The graph
supplies verified direction and context that dense similarity lacks.

### 12.6 Higher-level roots are useful metadata, not a universal truth

A top-down view benefits from classifying likely roots, for example:

- executable `main` or exported library entry;
- HTTP/router handler or middleware constructor;
- CLI command handler;
- background job or event consumer;
- React component/hook public entry;
- test entry point, when tests are in the selected corpus.

But root classification must not claim that every program has one true root.
Libraries, frameworks, callbacks, reflection, and dependency injection can
make roots numerous or externally supplied. Root kind is therefore path
metadata and a possible later selection signal, not a required semantic fact
and not a reason to discard an otherwise verified relation.

### 12.7 Path coherence can inform admission without becoming another ranker

The graph can help distinguish a coherent answer path from an adjacent but
unrelated dense candidate:

```text
coherent path
  handler/root -> RealIP -> realIP

unconnected neighbor in the same dense pool
  walkXFF
```

This does not mean every graph-connected node is relevant. Structural
connectivity is evidence of program relation, not proof of natural-language
intent. A bounded admission policy may eventually use only label-blind facts
such as:

- dense seed rank and score order;
- compiler resolution authority;
- relation kind and direction;
- path length;
- source/target occurrence role;
- whether the path reaches a classified public/entry boundary;
- stable identity and source order.

Graph centrality, learned path scoring, and relevance-label-driven rules remain
out of the first experiment. The first sidecar should expose all verified
reachability separately from the bounded path it selects, so an admission
failure cannot be mistaken for a parser or resolver failure.

### 12.8 How this extends the bounded experiment

The earlier one-hop experiment remains the smallest proof for G09 and X08.
The graph refinement adds a subsequent diagnostic rather than silently
expanding the first control:

1. prove exact one-hop relation extraction and parent mapping;
2. prove bidirectional lookup from the same stored facts;
3. derive a query-local usage path from the fixed dense seed frontier;
4. measure whether the path supplies more coherent evidence than isolated
   relation targets;
5. only then decide whether a bounded upward chain should extend beyond one
   hop.

Any multi-hop arm must freeze its maximum depth, cycle handling, root policy,
parent/byte cap, and deterministic path selection before replay. It must not be
introduced merely because the one-hop result has already been seen.

### 12.9 Updated working direction

The working direction is now more specific:

```text
semantic parent chunks
+ compiler-resolved occurrence graph
+ bidirectional caller/callee and type-reference lookup
+ embedding-guided bounded usage-path projection
+ protected related-evidence packaging
```

This is still an experimental design direction, not an authorized production
change. Its purpose is to test whether a small set of verified code-structure
primitives, composed with the already strong dense localization, materially
improves answer evidence without reintroducing broad lexical/fusion noise.

## 13. Working conclusion

The current cidx advantage over `rg` is natural-language semantic localization:
it can discover relevant code without an exact identifier. That advantage is
not yet fully realized because the user or assistant may still need `rg`, IDE
navigation, or compiler tools to prove the relationship and assemble the
answer.

The next differentiating layer is therefore not a larger text index. It is a
small, provenance-bound semantic occurrence and relation layer that turns:

```text
plausible code candidate
```

into:

```text
candidate
+ exact call/type/member relation
+ caller/callsite/target context
+ bounded answer-evidence bundle
```

Online systems support this direction but also demonstrate its boundaries:
Tree-sitter supplies syntax, language semantics supplies identity, graph facts
supply relationships, and packaging supplies usable context. None of them
alone determines natural-language relevance. The first sidecar experiment must
measure every transition rather than treating graph adjacency as proof of an
answer.

## 14. Measured diagnostic outcome

The bounded implementation and clean chi/RHF replay are complete. Compiler
resolution recovered the exact G09 call, X08/T09/T10 type relations, and the
G09 reverse caller lookup at pinned byte ranges. The graph therefore closes
the relation-extraction uncertainty.

It did not close answer identification. The fixed label-blind selector saw 88
reachable facts for G09 and 944 for X08, selected a different fact in each
case, and preserved complete-at-five at `30/32`. Both failures are attributed
to `RELATION_ADMISSION`. The measured conclusion is that a relation graph is a
useful candidate-evidence layer but not a sufficient answer selector. Current
production integration is rejected; exact artifacts, hashes, and the next
boundary are recorded in
[`evidence/phase-07/relation-usage-graph-diagnostic-r4.md`](evidence/phase-07/relation-usage-graph-diagnostic-r4.md).

## 15. Measured edge-metadata and graph-first outcome

The bounded follow-up added no relation prose and no relation embedding. It
stored mechanically derived occurrence zone, role, flow, execution/control
context, file role, nearby identifiers, parent-local ordinal, and frozen
parent traits in a v2 development sidecar.

That metadata materially helped one case. Dense-first selected the exact
`RealIP -> realIP` call for G09 and attached both complete grade-2 parents,
raising chi from `11/12` to `12/12`. RHF stayed `19/20`: the exact
`FormState -> FormStateProps` type reference was resolved and reachable, but
the selector preferred a higher-context-overlap relation between two grade-0
types. X08 therefore remains `RELATION_ADMISSION`, not extraction,
resolution, reachability, packaging, or embedding coverage.

The conditional graph-first crossover then used frozen FTS/simple-control
top-five parents as graph seeds and reranked only the best metadata tier by
the existing dense ordinal. It added no complete answer: chi stayed `12/12`
and RHF stayed `19/20`. It also attached the expressly guarded
`middleware.walkXFF` parent to G05, making the chi arm ineligible.

The evidence narrows the design direction:

- edge metadata is useful supporting evidence and should remain available in
  the development sidecar;
- graph adjacency or graph-first traversal is not a relevance decision;
- dense localization plus a fixed lexicographic metadata key still cannot
  reliably identify the answer relation;
- the remaining problem is query-conditioned evidence-group admission; and
- the exposed 32 cases must not be used to reorder the metadata key now that
  its failure is known.

Production graph integration remains rejected. A later selector must be
specified against a new calibration unit and validated on the separate
unexposed confirmation set. Exact hashes, G09/X08 traces, the graph-first
safety failure, and the zero-provider boundary are recorded in
[`evidence/phase-07/relation-edge-metadata-diagnostic-r4.md`](evidence/phase-07/relation-edge-metadata-diagnostic-r4.md).
