# cidx Phase History and Evidence Index

- Updated: 2026-09-05
- Scope: Phases 00–14
- Current product boundary: scalar four-tool MCP; Phase 12 confirmation blocked
- Authority: navigation and history index, not promotion evidence

This file is the durable entry point for another AI resuming or improving cidx.
It summarizes what each phase actually delivered and points to the detailed
phase plan and evidence. It does not replace [`../STATUS.md`](../STATUS.md),
the active phase document, immutable experiment results, or the canonical
design.

## 1. Required reading order

1. Read [`../EXECUTION-GUIDE.md`](../EXECUTION-GUIDE.md),
   [`../README.md`](../README.md), and [`../STATUS.md`](../STATUS.md).
2. Use the table below to identify the active or blocked phase.
3. Read that phase's full plan and its evidence-directory `README.md`.
4. Read the completion evidence of every prerequisite named by the phase.
5. For evaluation work, also read
   [`../EVALUATION-CONTRACT.md`](../EVALUATION-CONTRACT.md).

Older evidence is intentionally retained. When an old record conflicts with a
newer supersession record, use this order: latest explicit owner decision,
canonical Revision 4 design, `STATUS.md`, current phase plan, current evidence
index, then historical evidence. Never rewrite an old run to make it resemble
the current product.

## 2. Current phase map

| Phase | State | Durable outcome | Detailed history | Resume boundary |
| --- | --- | --- | --- | --- |
| 00 | done | Shared contracts, constants, fingerprints, and evaluation vocabulary | [plan](../00-shared-contracts-and-config.md), [evidence](phase-00/README.md) | Update first when a cross-phase contract changes |
| 01 | done | SQLite/FTS5, Tree-sitter, platform, storage, and codec feasibility | [plan](../01-runtime-storage-spike.md), [evidence](phase-01/README.md) | Reopen only for a new runtime/platform claim |
| 02 | done | Validated `ResolvedConfig`, profile identity, schemas, and store boundaries | [plan](../02-config-profiles-and-schemas.md), [evidence](phase-02/README.md) | Current product is 1024/int8, optional 512/int8 |
| 03 | done | Go Tree-sitter function/method/type chunker | [plan](../03-go-chunker.md), [evidence](phase-03/README.md) | Reopen for measured Go parsing failures |
| 04 | done | TypeScript/TSX Tree-sitter function/method/type chunker | [plan](../04-typescript-tsx-chunker.md), [evidence](phase-04/README.md) | Reopen for measured TS/TSX parsing failures |
| 05 | done | Incremental worktree indexing and atomic active-generation publication | [plan](../05-worktree-index-pipeline.md), [evidence](phase-05/README.md) | Preserve SQLite as the persistent authority |
| 06 | done | Provider-free FTS5 search with separate symbol/path/descriptive lanes | [plan](../06-fts-search.md), [evidence](phase-06/README.md) | Do not tune again on exposed calibration |
| 07 | done | Versioned lexical and relation calibration with explicit denominators | [plan](../07-lexical-evaluation.md), [evidence](phase-07/README.md) | Preserve all question/run versions; not confirmation |
| 08 | done | Durable document 1024-f32 source bank isolated from serving and lab state | [plan](../08-raw-embedding-lab.md), [evidence](phase-08/README.md) | Search/MCP must never open the source bank |
| 09 | done | Local 1024/512 int8 materialization and production vector integrity | [plan](../09-vector-materialization.md), [evidence](phase-09/README.md) | Binary/256 remain historical only |
| 10 | done | Source-bank-first Voyage document embedding and reconciliation | [plan](../10-embedding-orchestration-and-reconciliation.md), [evidence](phase-10/README.md) | Provider calls remain explicit paid operations |
| 11 | done | Request-local dense scan, FTS/dense RRF, fallback, and body packaging | [plan](../11-vector-and-hybrid-search.md), [evidence](phase-11/README.md) | Query f32 remains nonpersistent |
| 12 | blocked | Corpus-independent retrieval evaluation adapter is complete | [plan](../12-retrieval-evaluation.md), [evidence](phase-12/README.md) | Owner must freeze genuinely unexposed confirmation inputs |
| 13 | done | CLI plus exactly four MCP tools; locator-only search and scalar `read_span` | [plan](../13-cli-and-mcp.md), [evidence](phase-13/README.md) | Batch-v2 was tested, rejected, and removed |
| 14 | blocked | Local package checkpoint and assistant-use experiment history | [plan](../14-packaging-and-host-integration.md), [evidence](phase-14/README.md) | Wait for immutable Phase 12 `core_retrieval` result |

## 3. Phase-by-phase implementation history

### Phase 00 — shared contracts and configuration

- Established the canonical field catalog, named limits, model/profile IDs,
  RFC 8785 canonicalization, semantic fingerprints, and change-impact rules.
- Separated free AST/FTS work from explicitly authorized paid embedding work.
- Later supersession fixed serving to cidx-owned int8 at default 1024 or
  explicit 512 and retired executable Binary/256 paths.
- Improvement rule: change these catalogs before editing downstream copies;
  duplicated dimensions, codecs, or policy constants are defects.

### Phase 01 — runtime and storage spike

- Proved local SQLite with FTS5/WAL, one atomic active-generation transaction,
  and Tree-sitter parsing for Go, TypeScript, and TSX.
- Selected bundled grammars; parsing requires a compatible C toolchain while
  SQLite uses the pure-Go driver.
- Verified darwin/arm64 only. Other platform claims remain Phase 14 work.
- Historical codec experiments remain useful evidence but are not current
  product options.

### Phase 02 — configuration, profiles, and schemas

- Implemented strict config decoding and one `RawConfig -> Resolve -> Validate
  -> ResolvedConfig` authority.
- Added canonical profile/fingerprint hierarchy, atomic schema migrations,
  production/source-bank/lab separation, and strict evaluation schemas.
- Reconciled the current 1024/int8 default and 512/int8 compact profile while
  rejecting Binary/256 product configurations.
- Improvement rule: profile or schema changes require a cross-phase impact
  record; do not infer migration behavior from old rows.

### Phase 03 — Go chunker

- Implemented a stateless Tree-sitter Go adapter for functions, methods, and
  types with deterministic source ranges and shared projections/segments.
- Excluded exported `const`/`var` as standalone search chunks and retained
  parser diagnostics instead of silently inventing chunks.
- Improvement work must start from a reproducible parsing miss, then update the
  chunker version and downstream reindex handoff.

### Phase 04 — TypeScript and TSX chunker

- Implemented separate TypeScript and TSX Tree-sitter handling because their
  syntax and declaration forms differ from Go and from each other.
- Added function/method/type extraction, JSX-safe ranges, path-derived labels,
  and a real-corpus overload correction with a versioned reindex boundary.
- Language-specific quality must remain visible; aggregate results may not
  hide Go, TypeScript, or TSX failures.

### Phase 05 — worktree indexing pipeline

- Implemented filesystem enumeration, ignore/safety rules, content hashing,
  AST/FTS preparation outside SQLite writes, and short atomic publication.
- Search observes one committed generation. Unchanged compatible embeddings
  can be reused by identity; unproven or retired vector rows stay pending.
- There is no second authoritative in-memory index and no automatic provider
  call during index/reindex.

### Phase 06 — provider-free FTS search

- Replaced the failing all-token-AND natural-language planner with independent
  symbol, path, and descriptive lanes plus safe OR admission and deterministic
  local parent fusion.
- Kept exact required lexical anchors capable of AND matching without applying
  that restriction to ordinary natural-language questions.
- This phase is closed on exposed calibration. A future change needs new
  confirmation evidence, not another adjustment to known failures.

### Phase 07 — lexical and relation evaluation

- Versioned every question set and preserved prior runs instead of overwriting
  results. Final unchanged-v2 lexical evidence moved candidate-zero from
  32/44 to 0/44 and complete requirement hit@5 from 10/44 to 30/44.
- Measured candidate admission separately from top-five ranking and recorded
  Go/TypeScript/TSX, lexical, semantic, mixed, multi-requirement, and negative
  denominators.
- Relation/sibling packaging remains evaluation-only. The exposed chi/RHF
  material is calibration and cannot vote as Phase 12 confirmation.

### Phase 08 — document source-vector bank

- Created a durable product-owned bank for validated 1024-dimensional document
  f32 so 1024 or 512 int8 can be rematerialized locally without repaying the
  provider.
- Kept evaluation metadata physically separate and excluded persistent query
  vectors. Serving/search/MCP do not open this bank.
- The source bank is a product reuse mechanism, not a permanent multi-profile
  experiment workflow.

### Phase 09 — vector materialization

- Implemented deterministic local reduction/normalization and cidx-owned int8
  encoding for default 1024 and explicit 512 serving profiles.
- Added row integrity checks, local rematerialization, atomic publication, and
  the production vector cache used by exhaustive request-local scan.
- Removed Binary/256 executable paths. Historical measurements do not
  authorize restoring them.

### Phase 10 — embedding orchestration and reconciliation

- Implemented the official Voyage `voyage-code-4` document path with explicit
  1024 float output, document role, no truncation, response validation, and
  source-bank-first reuse.
- Preserved the separation between free index planning and paid missing-source
  capture. The synchronous executor uses central request/group/concurrency/
  timeout/retry settings; asynchronous provider Batch is outside v1.
- Provider usage is recorded only for actual provider attempts.

### Phase 11 — vector and hybrid search

- Implemented request-local query embedding, local exhaustive int8 scan,
  target-f32 diagnostic reference, FTS/dense candidate lanes, RRF, fallback,
  parent collapse, and bounded body packaging.
- Query f32 is never persisted. FTS works without provider state; hybrid query
  embedding is an explicit paid action.
- HNSW/ANN remains out of scope until measured exhaustive-scan limits justify
  a separate decision.

### Phase 12 — retrieval evaluation

- Implemented the corpus-independent stage-separated adapter, frozen arm plan,
  first-loss accounting, codec/fusion/body diagnostics, immutable artifact
  framing, and scoped `core_retrieval` promotion schema.
- Synthetic and focused integration evidence is complete; it is not an
  official corpus result or promotion result.
- Blocker: the owner must select a genuinely unexposed confirmation corpus and
  freeze its pin/license, questions, cohort floors, margins, source-bank
  coverage, and any paid query authorization. No AI may choose or download it
  on the owner's behalf.

### Phase 13 — CLI and MCP

- Implemented one CLI and stable stdio MCP with exactly `status`, `search`,
  `read_span`, and `reindex`, strict lifecycle/cancellation/error behavior,
  one repository root per process, and provider-free FTS operation.
- `search` returns compact ranked locators without source or planner internals.
  Scalar `read_span` returns one exact current range guarded by the indexed
  whole-file SHA and byte ceiling.
- A terminal 30-pair experiment tested optional 2–4 locator batch reads. It
  reduced round trips but increased duplicate/overlap rates and contained one
  valid complete-to-partial regression. Batch-v2 was rejected and scalar-v1
  restored at `ff9d4e8`.

### Phase 14 — packaging, host integration, and assistant evidence

- Recorded a local darwin/arm64 package checkpoint, runtime dependency checks,
  host-facing constraints, and the full assistant-use experiment sequence.
- Locator-only search dramatically reduced cidx response payload. Later
  awareness/trust guidance improved answer completeness and narrowed source,
  but did not establish end-to-end action/token efficiency.
- The final multi-locator candidate was owned and closed by Phase 13. Phase 14
  must not restart prompt/interface iteration from those exposed runs.
- Blocker: immutable Phase 12 core evidence must exist before cross-platform
  package/host verification and a scoped `release_candidate` result.

## 4. Cross-phase supersession timeline

- Initial codec and dimension experiments are retained as history; the current
  product contract is 1024/int8 by default and 512/int8 only when explicit.
- The 1024 document-f32 source bank became durable product state; query f32
  remained request-local and nonpersistent.
- Natural-language FTS moved from global AND admission to independent lexical
  lanes. Prior bad runs remain evidence of the corrected first loss.
- MCP search moved from source-bearing results to compact locators; scalar
  `read_span` became the sole cidx source transfer.
- Assistant studies progressed through compact output, prompt guidance,
  availability, forced use, awareness/trust, and one terminal batch-read
  contract test. No assistant efficiency or release claim was established.
- On 2026-09-05 batch-v2 was rejected, scalar-v1 restored, Phase 13 closed,
  and work returned to the Phase 12 owner gate.

## 5. Exact remaining sequence

1. Owner selects and freezes genuinely unexposed Phase 12 confirmation inputs.
2. Prepare provider-free manifests, bindings, licenses, hashes, questions,
   cohort denominators, floors, margins, and source-bank coverage checks.
3. Obtain explicit approval immediately before any paid document/query call,
   then run one immutable confirmation and decide `core_retrieval` scope.
4. If core evidence is complete, resume Phase 14 packaging/host validation and
   create a separate immutable `release_candidate` disposition.

Do not insert another experiment merely because the next phase is externally
blocked. Grader support for product-valid long spans must be reconciled before
the next scored assistant evaluation, but it must not be applied retroactively
to the closed multi-locator run.

## 6. History update rules

- Add a dated evidence file; do not replace a prior result.
- Update the phase evidence `README.md`, this index, the phase plan, and
  `STATUS.md` together when state or handoff changes.
- Record implementation changes, actual checks, checks not run, artifacts,
  decisions, rejected alternatives, remaining risk, and exact next action.
- Keep generated corpora, vectors, indexes, transcripts, and machine-specific
  paths in ignored local state; commit reproducibility metadata and digests.
- A historical success proves only its recorded scope. It does not silently
  establish current-contract compliance, promotion, or platform support.

## 7. Git trace

Every phase plan and evidence file is versioned in Git. When the reason for a
change matters, inspect the commits affecting the phase rather than inferring
it from the latest file alone:

```text
git log --oneline -- docs/implementation/<phase-plan>.md \
  docs/implementation/evidence/phase-XX
git show <commit>
```

Commit messages are navigation aids. The committed evidence content and its
recorded artifact digests define what was actually checked at that boundary.
