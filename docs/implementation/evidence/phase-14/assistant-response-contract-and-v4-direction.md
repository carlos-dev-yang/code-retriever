# Assistant Locator/Evidence Contract Review and V4 Direction

- Date: 2026-08-21
- Status: reviewed design direction; no product implementation or V4 run yet
- Scope: Phase 14 diagnostic assistant integration only
- Corpora: existing owner-approved chi v5.3.1 and React Hook Form v7.85.0
- Provider activity: none
- Promotion authority: none
- Predecessor: [Assistant A/B Version 3 result](assistant-ab-v3-result.md)
- External review: [ChatGPT](https://chatgpt.com/c/6a86c914-2d18-83e8-9b86-d29de2922b2a) and [Grok](https://grok.com/c/d1f4ad8f-65de-49fb-a3c2-fbd026d4bd84?rid=2123c116-d76c-4d21-a7b3-870c1285c94c)

This report separates candidate navigation from source-evidence acquisition,
defines distinct measurements for both stages, and fixes the smallest justified
direction before another assistant experiment. It preserves all V1-V3 question,
run, grade, and transcript artifacts. It does not freeze the final V4 manifest
or authorize a public MCP schema edit by itself.

## 1. Decision

The next diagnostic must treat the two MCP operations as different stages:

```text
search
  -> compact, deduplicated candidate locators
  -> no source body
  -> no serving/evaluation diagnostics

assistant selects locators
  -> read_span
  -> exact selected source plus provenance

assistant resolves task
  -> final source-backed answer
```

The initial compact experiment keeps FTS planning, ranking, the current
caller-selected `k` semantics, prompt, question panel, and `read_span` behavior
unchanged. Most V3 calls selected `k=10`, but some selected 5, 6, or 12; V4
must not force one value merely to simplify the report. Intentionally reducing
`k`, changing routing, or enabling hybrid/dense in the same experiment would
confound response-contract reduction with retrieval behavior.

This distinction is important:

- retrieval may consider many internal candidates;
- `search` may return enough cheap locators to preserve recall; and
- only the small subset explicitly selected by the assistant should cause
  source text to enter the context.

The defect observed in V3 is primarily excessive external expansion of
candidates, not proof that the internal candidate pool or top ranks are wrong.

## 2. Corrected V3 accounting

### 2.1 Stable conclusions

The official V3 facts remain:

- baseline was 11 complete and 1 partial;
- cidx FTS was 12 complete with no reverse regression;
- total model tokens increased from 1,223,579 to 1,678,341 (`+37.2%`);
- uncached input increased from 241,613 to 363,843 (`+50.6%`);
- treatment issued 19 `search` and 29 `read_span` calls;
- 176 search hits were serialized, 154 with inline bodies;
- search bodies contained 170,423 bytes of source, followed by 46,009 bytes
  of source from `read_span`; and
- only 24 of 85 task-local returned path occurrences were finally cited.

The transcript contains 390,071 bytes of structured search JSON and 402,601
bytes of the equivalent text representation. Their simultaneous presence in
events does not prove that the host supplied both to the model. Official token
usage remains the end-to-end authority.

### 2.2 Frozen journey reducer erratum

The V3 journey reducer's shell-command detector does not reliably match the
first repository command when Codex records it as, for example,
`/bin/zsh -lc "rg ..."`. The quote before `rg` is not one of the detector's
accepted command boundaries. It therefore undercounted baseline shell output.

A read-only audit of the raw events gives these transcript-output proxies:

| Proxy | Baseline | cidx treatment |
| --- | ---: | ---: |
| Raw shell `aggregated_output` | 697,490 B | 36,147 B |
| Serialized cidx result envelope | — | about 952,986 B |
| Combined raw/event proxy | about 697 KB | about 989 KB |

Under this corrected proxy, treatment is larger in 11 of 12 tasks, not 12 of
12. `rhf-t09-control-contract` is the exception because the baseline's first
broad `rg` emitted about 351 KB. The frozen journey rows and their digests
remain immutable, but their baseline visible-byte values and derived
visible-output ratios must not be used as valid evidence. The correctness and
official token results are unaffected.

The next reducer must record all command results without guessing repository
inspection from a quoted shell string, or must normalize the shell wrapper
before classification.

### 2.3 Compact projection

Reprojecting the 19 frozen searches to the proposed fields below produces
49,711 bytes of compact structured JSON. A slightly wider locator projection
from the owner-supplied review produced 55,866 bytes. The exact value depends
on whether generation and additional match details are retained, but both are
about 13-14% of the current 390,071-byte structured search result.

With unchanged structured `read_span` responses, the projected structured
search-plus-read payload is about 102-108 KB, roughly 23-25% of the current
442,589 bytes. This is a serialization projection, not a model-token forecast.

## 3. Model-facing contract

### 3.1 `search`: locator only

The compact result should preserve only what the assistant needs to choose and
safely read a candidate:

```json
{
  "results": [
    {
      "chunk_id": 486,
      "path": "src/types/form.ts",
      "language": "typescript",
      "kind": "type",
      "qualified_symbol": "module.Control",
      "start_line": 870,
      "end_line": 917,
      "indexed_sha256": "...",
      "match_sources": ["symbol", "descriptive_fts"]
    }
  ]
}
```

Array order is rank, so a separate rank field is unnecessary. Match sources
must be short, stable enums rather than planner prose or scores.

V4 retains `indexed_sha256` because the current stateless `read_span` contract
requires `expected_sha256`. Replacing it with an opaque result ID would require
a new resolution/state contract and would expand the experiment. That option
is deferred.

The following are not model-facing search fields:

- source body, body ranges, comments, or previews;
- full or projected signatures;
- BM25, vector, or RRF scores and every component rank;
- symbol/path match tiers beyond one compact match-source summary;
- query shape, selected/dropped terms, planner explanations, and query
  normalization;
- candidate-pool, lane, coverage, deduplication, or omission diagnostics;
- manifest, source, vector-space, and vector-storage fingerprints;
- serialization, timing, or profiling diagnostics; and
- repeated constant source-state/content-source labels.

Those values remain available in evaluation traces and diagnostic logs. They
must not be discarded merely because they are removed from the assistant wire.

Canonical results are deduplicated by indexed content identity, path, parent
range, and qualified symbol before return. Several matching segments or lanes
for one parent remain one locator with merged match sources.

### 3.2 `read_span`: evidence only

`read_span` remains the only cidx operation that returns source text. Its
current path, start/end line, expected hash, returned body, and provenance are
enough for the compact experiment. It should not repeat search diagnostics.

Search and `read_span` must therefore have zero source-body overlap by design.
Repeated or overlapping `read_span` calls remain measurable and are not hidden.

### 3.3 One semantic representation

The server currently emits the same result as both text JSON and structured
content. Before implementation, a host-conformance probe must determine which
single representation Codex and other claimed hosts accept and expose. Use
structured-only when supported; otherwise use one compact text representation.
Do not infer that removing one event representation will halve model tokens.

### 3.4 `max_inline_bytes`

For the first compact diagnostic, preserve the current input schema, prompt,
and caller-selected `max_inline_bytes` values. The versioned response projector
omits body fields regardless, while logs retain the requested/effective values.
This avoids combining payload compaction with an input-schema or guidance
change. If locator-only search is later adopted as the product contract, a
required inline-body argument becomes misleading and should be removed or
deprecated through the normal public-contract change procedure.

## 4. Stage-separated measurements

Let:

- `Gq` be the frozen required evidence groups for task `q`;
- `Lq,s@k` be the unique canonical locators returned by search call `s`;
- `Uq` be the union of locators across all search calls for `q`;
- `Rq` be the unique locators selected for a successful `read_span`;
- `Eq` be the unique source ranges actually returned by `read_span`;
- `Cq` be the final cited source ranges; and
- `accepted(g)` be every frozen OR alternative that satisfies group `g`.

Failures and timeouts remain in task denominators. A per-call denominator is
explicitly limited to scheduled or successful calls as stated; missing data is
never silently converted to zero.

### 4.1 Locator/navigation quality

These metrics determine whether `search` gave the assistant useful places to
read. They do not grant credit merely because a locator was returned and do
not depend on final-answer correctness.

```text
FirstSearchRequirementCoverage@k(q)
  = groups with an accepted locator in Lq,1@k / |Gq|

CompleteFirstSearchLocatorHit@k(q)
  = 1 only when FirstSearchRequirementCoverage@k(q) = 1

AnySearchRequirementCoverage(q)
  = groups with an accepted locator in Uq / |Gq|

FirstUsefulLocatorRank(q)
  = first rank in Lq,1 containing any accepted alternative,
    or 0/missing under the declared all-task MRR-style aggregate

DuplicateLocatorExposureRate
  = repeated canonical locator occurrences after the first / all locator occurrences

LocatorSelectionUtilization(q)
  = |Uq intersect Rq| / |Uq|

LocatorCitationUtilization(q)
  = |Uq intersect Cq| / |Uq|

NavigationFalseLeadRate(q)
  = inspected locators that are neither accepted alternatives nor cited / |Rq|
```

Report first-search and any-search coverage separately. Union-only coverage can
hide an inefficient sequence of repeated refinements.

Payload accounting is also stage-specific:

- structured search bytes;
- text search bytes;
- complete event-envelope bytes;
- source bytes in search, which must be zero in the compact arm;
- bytes per unique locator;
- search call and refinement counts; and
- confirmed host/model-visible bytes only when the host provides that fact.

Transcript envelope bytes must not be relabeled as model-visible bytes when
host ingestion is unknown.

### 4.2 `read_span`/evidence quality

These metrics determine whether selected reads supplied the source needed to
answer the task.

```text
EvidenceRequirementCoverage(q)
  = groups with an accepted alternative intersecting Eq / |Gq|

CompleteEvidenceHit(q)
  = 1 only when EvidenceRequirementCoverage(q) = 1

ReadSpanPrecision(q)
  = successful read spans intersecting an accepted alternative / successful read spans

EvidenceCitationUtilization(q)
  = unique read spans intersecting Cq / |Eq|

EvidenceFalseLeadRate(q)
  = read spans that are neither accepted alternatives nor cited / |Eq|

RedundantEvidenceRatio(q)
  = (gross delivered source bytes - unique delivered source bytes)
    / gross delivered source bytes
```

Also report gross and unique evidence bytes, successful/failed call counts,
and the ordered inspection action at which all required evidence first became
available. That last field is mechanical only when every accepted alternative
is already frozen and span-mapped.

### 4.3 End-to-end assistant usefulness

Keep these outcomes separate rather than calculating a weighted total:

- blind complete/partial/incorrect/ungradable outcome;
- required-group answer coverage and material unsupported/contradicted claims;
- complete-to-noncomplete and noncomplete-to-complete conversions;
- official input, cached input, uncached input, output, and model-total tokens;
- search, `read_span`, ordinary-read, and total inspection actions;
- locator payload bytes and evidence source bytes as separate observations;
- locator-to-read and locator-to-citation utilization;
- evidence-to-citation utilization; and
- final citation provenance: locator plus `read_span`, ordinary read, or other.

The 12-question panel remains diagnostic smoke evidence and cannot establish
`core_retrieval` or `release_candidate` promotion.

## 5. Information-reduction order

1. Correct the reducer and record raw shell, structured, text, envelope, and
   source bytes independently.
2. Make search locator-only and remove source bodies, signatures, and
   assistant-irrelevant diagnostics.
3. Deduplicate canonical parents before serialization and merge lane reasons.
4. Emit one semantically equivalent result representation after host
   conformance is proven.
5. Preserve the current caller-selected `k` semantics and do not introduce a
   smaller cap/default. Report the common compact-`k=10` calls separately.
6. Only after the compact contract is measured, compare explicitly fixed
   `k=10`, `k=5`, and `k=3` as a separate, frozen experiment.
7. Consider lexical-versus-hybrid routing only if semantic/multi-requirement
   journeys remain expensive after response compaction.

Reducing `k` first is rejected because it mixes payload reduction with recall
risk. Returning ten small locators may be cheaper and safer than returning
three large source-bearing records.

## 6. Smallest controlled V4 diagnostic

Before execution:

1. fix and version the journey reducer;
2. preserve the V1-V3 artifacts and record the V3 visible-byte erratum;
3. freeze and hash a `locator-v1` result projection;
4. prove one-representation host compatibility with unscored schema/tool
   probes; and
5. write the exact V4 manifest before the first scored turn.

The scored diagnostic then preserves:

- the existing versioned 12-question panel and truth;
- Go 4, TypeScript 4, and TSX 4 composition;
- task order and paired baseline/treatment schedule;
- Codex CLI model and reasoning effort;
- prompt and mandatory first MCP search intervention;
- fresh isolated source and state per turn;
- FTS planner, index, candidate generation, ranking, and caller-selected `k`
  semantics;
- `read_span` behavior and evidence policy;
- blind correctness grading and pre-arm-restoration journey freeze; and
- the no-selective-retry rule.

The only declared treatment change from V3 is the compact search response
contract and its compatible single representation. Run all 24 baseline and
treatment turns again; a baseline rerun is required because assistant
stochasticity prevents treating the older baseline as contemporaneous.

V4 reports the three stage scorecards above. It does not simultaneously change
tool-call limits, prompt strategy, ranking, query planning, routing, embedding,
or result depth. Any later orchestration experiment receives its own version.

## 7. External review disposition

The owner-supplied review and both side-panel reviews agreed that response
contract overlap is the smallest justified correction.

Both independent side-panel reviewers returned
`APPROVE_LOCATOR_EVIDENCE_SPLIT` and agreed on:

- locator-only search and source-only `read_span`;
- evaluation/log-only ranking and planner diagnostics;
- deduplication and one semantic result representation;
- no intentional lower-`k` change before the compact response is measured;
- the stage-separated metric families above;
- a corrected reducer before V4; and
- no ranking, routing, hybrid, dense, or new-corpus change in V4.

ChatGPT suggested that an opaque result ID could replace the indexed hash. This
report retains the hash for V4 because it preserves the current stateless
`read_span` safety contract. That is the only material design choice not copied
directly from both reviews.

## 8. Explicit non-inferences and deferred decisions

V3 does not establish that:

- both MCP result representations entered the model context;
- duplicate serialization caused a specific fraction of the token increase;
- FTS ranking is the primary bottleneck;
- a lower `k` is safe or more efficient after compaction;
- hybrid or dense retrieval is required;
- cidx is inherently inefficient;
- search bodies are necessary for correctness; or
- the 12-question panel supplies release evidence.

Deferred until after compact V4 evidence:

- `k=5` or `k=3` product/default changes;
- removal of `max_inline_bytes` from the public search input;
- opaque result-ID or server-side result-handle semantics;
- prompt or tool-call-count restrictions;
- lexical-versus-semantic routing;
- paid hybrid/dense execution; and
- any promotion threshold or release claim.

## 9. Implementation impact and handoff

No implementation changed during this review. Before code work, the public
contract procedure must reconcile the canonical design, Phase 13/14 tool
contracts, MCP output wiring, host compatibility, evaluation schemas, reducer,
and V4 plan in one coherent change. The code change must not alter rank or
result identity when removing bodies and diagnostics.

Checks performed for this report:

- re-read the Phase 14 and evaluation contracts;
- parsed all 12 frozen V3 baseline and treatment event streams;
- recomputed shell, search, `read_span`, body, metadata, and representation
  byte totals;
- inspected actual V3 commands, queries, hits, citations, and tool calls;
- projected the compact locator payload from the frozen search results; and
- obtained and retained matching ChatGPT and Grok reviews.

Checks not performed:

- no MCP or reducer implementation;
- no host one-representation compatibility probe;
- no assistant V4 execution;
- no Voyage request or paid operation;
- no new corpus; and
- no promotion or release evaluation.
