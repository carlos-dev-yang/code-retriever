# Natural-Language FTS Query-Planner Review — Revision 4

- Date: 2026-08-20
- Scope: existing chi v5.3.1 and React Hook Form v7.85.0 evidence only
- Evidence class: architecture, implementation remediation, and evaluation handoff; no promotion claim
- Provider operations: zero
- External advisory review: ChatGPT and Grok received the same English technical packet

## 1. Outcome

The measured FTS failure is primarily a **candidate-admission defect** in the
current query planner. It is not evidence that SQLite FTS5, lexical retrieval,
or the cidx MCP is generally useless.

The current planner normalizes every query token and joins all tokens with
`AND`. On the versioned 44-question calibration set, this admitted and completed
10 of 12 lexical-anchor questions, but all 24 semantic-only questions and all 8
mixed-signal questions returned zero FTS candidates. BM25 cannot rank a parent
that the boolean query excluded.

The adopted remediation target is:

1. keep `AND` only for explicit, high-confidence constraints that the caller
   clearly requires in the same result;
2. infer an internal query shape from the request rather than using evaluation
   cohort labels at runtime;
3. split local lexical retrieval into independent symbol, path, and descriptive
   FTS candidate lanes;
4. use an OR-based descriptive FTS lane with measured term-coverage diagnostics
   instead of global all-token `AND`;
5. keep dense retrieval independent in hybrid mode and fuse ordinal ranks only
   after each lane has produced candidates; and
6. evaluate candidate admission separately from ranking and final top-k quality.

The first revision changed the implementation target and evaluation plan. The
2026-08-20 implementation checkpoint below changes production lexical planning
and diagnostics without changing the four-tool MCP surface, frozen question
text, ground truth, cohort assignment, or any prior run result.

### 1.1 Implementation checkpoint

The target is now implemented as lexical planner version 2:

- safe normalized descriptive terms are joined by `OR`, never by implicit
  global `AND`;
- symbol, path, and descriptive FTS produce independent candidate lists;
- canonical parents are deduplicated and fused by ordinal RRF;
- hybrid dense retrieval remains independent and is fused only after local
  lexical fusion;
- serving-policy fingerprints include the planner version; and
- evaluation and MCP results expose plan, lane, coverage, and candidate-zero
  diagnostics.

Focused normal/race tests, vet, CLI build, formatting, and diff validation
passed. No real-corpus rerun or provider action is part of this implementation
checkpoint; those remain Phase 07 work.

## 2. Evidence behind the diagnosis

The immutable critical/general v2 diagnostic contains 44 questions:

| Query family | Cases | Current FTS CompleteRequirementHit@5 | Current FTS candidate-zero cases |
| --- | ---: | ---: | ---: |
| lexical anchor | 12 | 10/12 | 0 |
| semantic only | 24 | 0/24 | 24 |
| mixed signal | 8 | 0/8 | 8 |
| total | 44 | 10/44 | 32 |

The evaluation-only any-token control completed 28/44. That control is not a
production policy, but it proves that many indexed parents contain usable
lexical evidence that the all-token `AND` query never admits.

The two lexical-anchor misses are also not zero-candidate failures. For
`rhf-use-field-array`, the correct parent was rank 15; for `rhf-use-form`, it
was outside the current top 20. Those are ranking or candidate-depth problems,
not the same failure as the 32 natural-language zero-candidate cases.

Consequently the existing aggregate `10/44` combines at least two different
problems:

- **admission loss:** no candidate survives the boolean query; and
- **ranking/depth loss:** a relevant candidate exists but is displaced or lies
  beyond `candidate_k`/`return_k`.

Those failures must no longer be interpreted through one FTS score.

## 3. Current implementation and concrete defects

### 3.1 One boolean shape for every query

`internal/search/lexical/query.go` appends every identifier and text token to
one list and emits quoted tokens joined by `AND`. There is no deduplication,
stop-word or document-frequency selection, minimum-match policy, staged
relaxation, or query-shape decision.

`internal/symbol/query_tokens.go` distinguishes identifier-shaped fragments
from ordinary text, but both classes are recombined into the same `AND`
expression. This is token classification, not query planning.

### 3.2 Exact symbol is not a candidate lane

The current exact-symbol candidate is populated only for one identifier-style
input fragment with no text fragments. The SQL equality test then checks the
qualified symbol and the final sorter uses it only after BM25 score comparison.
An exact unqualified symbol such as `useFieldArray` therefore cannot reliably
admit or outrank `module.useFieldArray` as an independent lookup.

Exact qualified symbol, exact short symbol, normalized qualified symbol, and
normalized short symbol need their own candidate-generation order. They are
not BM25 decorations.

### 3.3 Path is not searchable

The FTS table has only `symbols` and `body`. File path participates only in a
stable tie-break after candidates exist. Path-led questions therefore lack a
path candidate source even though path/package/module intent is common in code
navigation.

### 3.4 Symbol normalization is asymmetric

The index stores original and normalized symbol and qualified-symbol forms in
the FTS symbol field. The body field contains the raw signature and raw AST
projections. Identifier forms appearing only in a body are not accompanied by
the same camelCase/snake_case decomposition used for query tokens.

This asymmetry is not the root cause of the 32 all-token failures, but it can
cause additional lexical misses after the boolean planner is fixed.

### 3.5 Candidate and ranking evidence are collapsed

The current report emphasizes final CompleteRequirementHit@5. It does not make
the boolean plan, selected terms, lane candidate counts, gold survival at
`candidate_k`, and conditional rank quality first-class outputs. That makes an
admission defect look like a weak BM25 ranker.

## 4. Target retrieval architecture

Use **lexical retrieval** as an umbrella over three free local lanes. SQLite FTS
is one of those lanes, not the name for every lexical operation.

| Lane | Candidate source | Primary purpose | Ranking contract |
| --- | --- | --- | --- |
| symbol | authoritative `symbols` rows | exact and normalized function/method/type lookup | exact qualified, exact short, normalized qualified, normalized short, then stable keys |
| path | authoritative indexed file paths | file/package/module-led lookup and filtering | exact path, basename/segment match, normalized path terms, then stable keys |
| descriptive FTS | FTS5 `symbols`/`body` projections | lexical evidence from natural-language and mixed descriptions | OR-based admission, BM25 ordinal, measured coverage diagnostics, stable ties |
| dense | exhaustive current-profile int8 segment scan | semantic similarity in explicitly authorized hybrid search | existing cosine-derived ordinal after parent collapse |

The lane sequence is:

```text
raw query
  -> safe query-shape planner
  -> explicit anchors + descriptive terms
  -> symbol lane -------+
  -> path lane ---------+--> canonical parent union --> ordinal fusion --> top k
  -> descriptive FTS ---+
  -> dense lane --------+    (hybrid only; never gated by FTS)
```

The dense lane must not consume only FTS candidates. Doing so would cap semantic
recall at lexical admission and reproduce the current defect in hybrid mode.
FTS and dense remain parallel providers, as required by the evaluation
contract.

Within FTS itself, broad candidate admission is analogous to a first retrieval
stage, but it is not ANN and it is not an ANN replacement. cidx continues to
perform an exhaustive local dense scan; HNSW/ANN remains out of scope.

## 5. Query-shape planner

Runtime behavior must not read evaluation labels such as `semantic_only` or
`mixed_signal`. It derives a plan from syntax and token evidence.

### 5.1 Internal shapes

| Shape | Evidence | Plan |
| --- | --- | --- |
| anchor | an explicitly delimited or high-confidence symbol/path/literal with little descriptive text | symbol/path lookup plus a narrow lexical query |
| descriptive | ordinary natural-language terms with no reliable code anchor | descriptive OR admission; no invented MUST term |
| mixed | one or more reliable anchors plus descriptive terms | independent anchor lane and descriptive FTS lane; union before fusion |
| empty | no safe searchable term after normalization | explicit input error; never scan all rows |

High-confidence anchors include explicitly quoted/backticked code strings,
qualified identifiers, camelCase/PascalCase/snake_case identifiers, path-like
tokens, file names, and literal/config-key forms. A capital letter or digit by
itself is not sufficient evidence that every decomposed token is mandatory.

### 5.2 `AND` policy

`AND` is permitted only when the caller explicitly communicates that multiple
anchors are required in the same result, or when one exact phrase is represented
as a required unit. It is not the default connective for inferred descriptive
terms.

The first implementation should not add an expert FTS syntax or a new MCP tool.
The planner may use explicit quoting/backticks and code-shaped tokens while
recording its reasoning in diagnostics. A future structured `must` field would
be a separate MCP wire decision and is not authorized by this report.

### 5.3 Descriptive FTS policy

The initial remediation arm is deliberately simple:

1. normalize safely;
2. deduplicate terms;
3. retain informative terms while identifying common terms for diagnostics;
4. admit a row when any selected descriptive term matches;
5. rank with the existing BM25 field weights and deterministic keys; and
6. record distinct matched-term coverage for every candidate.

Do not immediately freeze a stop-word list, minimum-should-match percentage,
coverage bonus, prefix rule, fuzzy edit distance, or stemming policy. Compare
those only as separately declared calibration arms after the OR baseline shows
its candidate set and ranking behavior. This avoids replacing one unmeasured
global rule with another.

Staged relaxation or document-frequency-guided term dropping is a valid later
arm if plain OR is too noisy. It must be bounded and observable; it must not
enumerate arbitrary token subsets.

## 6. Role of FTS in cidx

FTS has three first-class roles:

1. **free lexical retrieval** when no embedding key or paid query is available;
2. **independent lexical evidence** in hybrid retrieval; and
3. **offline fallback** if dense preflight or query embedding fails.

FTS is not:

- a mandatory prefilter for dense retrieval;
- a replacement for semantic embeddings;
- a universal answerer for every natural-language request;
- an exact-symbol or path index by itself; or
- proof of MCP usefulness or uselessness.

The MCP can still be useful when FTS alone misses a semantic query, provided
the independent dense lane or another local anchor lane finds the required
parents and the final assistant can use the bounded source response. That
product question belongs to later paired assistant A/B, not to this FTS-only
diagnostic.

## 7. Complementary retrieval conditions

### Implemented before the next FTS rerun

- query-shape diagnostics;
- removal of global inferred-term `AND`;
- independent exact/normalized symbol candidate generation;
- independent path candidate generation;
- descriptive OR candidate admission;
- matched-term and lane-contribution diagnostics; and
- deterministic parent union without duplicate chunks.

### Must preserve

- safe internal MATCH construction; raw user input never becomes FTS grammar;
- one committed generation per search;
- contentless FTS/chunk integrity checks;
- free FTS-only operation with no provider dependency;
- exhaustive dense scan independent of lexical candidates;
- rank-only fusion; BM25 and cosine raw scores are never compared directly;
- fixed candidate and result limits from resolved config; and
- exactly four stable MCP tools.

### Defer until the basic planner is measured

- prefix indexes, fuzzy matching, stemming, synonyms, NEAR, and custom
  tokenizers;
- fixed minimum-should-match or term-coverage weights;
- lane-specific RRF weights;
- query-conditioned graph or sibling expansion; and
- adaptive automatic paid dense calls.

Graph/sibling evidence may later organize or complete results reached from a
high-confidence seed. It cannot repair a zero-candidate lexical planner and is
not part of this remediation.

## 8. Revised evaluation scorecard

Keep final relevance metrics, but add explicit admission and lane diagnostics.
No weighted total quality score is created.

### 8.1 Query-plan and candidate-admission stage

For every query and every local lane, record:

- inferred query shape;
- explicit anchors and descriptive selected/dropped terms;
- internal boolean shape without raw secret-bearing query text;
- candidate count before parent union;
- `CandidateZero` and `CandidateZeroRate`;
- `GoldParentCandidateRecall@20`;
- `RequiredGroupCandidateCoverage@20`;
- `ExactSymbolHit@1/@20` for applicable anchor questions;
- `PathHit@1/@20` for applicable path questions; and
- unique-gold contribution by symbol, path, descriptive FTS, and dense lanes.

`candidate_k=20` is the current observed depth, not a permanent quality
threshold. Any changed depth is a separate run condition.

### 8.2 Ranking stage

Report both unconditional and admission-conditional outcomes:

- Hit@1/5, Recall@5, MRR, and NDCG@5;
- CompleteRequirementHit@5 and requirement coverage;
- rank of the first relevant parent;
- relevant parents present at 20 but absent at 5; and
- duplicate-parent and stable-tie diagnostics.

If no gold parent entered the first 20 candidates, classify the loss as
admission. If a gold parent entered the first 20 but failed the returned top 5,
classify it as ranking/depth displacement. Do not call both failures “FTS miss”.

### 8.3 Hybrid stage

Preserve separate symbol/path/descriptive-FTS/dense candidate lists and report:

- each lane's candidate and gold contribution;
- union survival;
- overlap and disagreement;
- RRF rescue and harm;
- parent-collapse effects for dense segments; and
- hybrid-without-each-lane ablations.

Changing FTS admission changes the RRF input and therefore requires a focused
Phase 11 integration revalidation before hybrid evidence is comparable.

### 8.4 Failure taxonomy

Use distinct diagnostic causes:

```text
QUERY_PLAN_EMPTY
REQUIRED_ANCHOR_NOT_INDEXED
REQUIRED_ANCHOR_FILTERED_ALL
SYMBOL_CANDIDATE_MISS
PATH_CANDIDATE_MISS
DESCRIPTIVE_FTS_ZERO_CANDIDATES
GOLD_ABSENT_AT_CANDIDATE_K
GOLD_DISPLACED_BEYOND_RETURN_K
DENSE_CANDIDATE_MISS
PARENT_COLLAPSE_LOSS
RRF_FUSION_LOSS
OPERATION_FAILURE:<stage>
```

These refine diagnostics under the existing stage-separated first-loss model;
they do not allow a lost requirement group to reappear later in the trace.

## 9. Question-set and run versioning

The existing v2 question text, truth, cohorts, and four run artifacts remain
fixed. The new planner can be evaluated against the same v2 question set
because the evaluation input did not change. It must create a new immutable run
ID and bind the new query-planner/lexical-profile fingerprint.

If question text, ground truth, cohort assignments, or requirement groups are
edited, create question-set v3 with `supersedes` and a change summary. Preserve
v2 and every v2 run. Do not overwrite a run and do not use the phrase
“re-scoring prohibition”; the rule is simply:

- old question versions and their results stay recorded;
- changed questions receive a new version; and
- every run identifies exactly which question and planner versions it used.

## 10. Side-panel advisory review

ChatGPT and Grok independently received the same English packet containing the
44-question result, current FTS query/index/rank rules, dense/RRF behavior, and
the proposed lane split. Their responses are advisory; owner direction and
repository evidence remain authoritative.

Both reviewers agreed on these points:

- the primary defect is candidate admission, not BM25 scoring;
- global all-token `AND` is inappropriate for natural-language requests;
- `AND` belongs only to explicit high-confidence anchors/constraints;
- exact symbol and path retrieval should be candidate lanes, not BM25
  tie-breaks;
- semantic/mixed requests need independent descriptive and anchor treatment;
- FTS and dense should remain parallel, then fuse by rank;
- evaluation must separate candidate recall/zero rate from top-k ranking; and
- noisy OR, common code words, duplicate lane results, and mixed-query anchor
  domination must be measured explicitly.

ChatGPT most strongly emphasized independent symbol/path/FTS/dense lanes and
candidate-generation metrics. Grok additionally suggested adaptive dense
triggers and illustrative cross-lane weighted scores. Those two suggestions
are not adopted now: paid dense use remains explicit, and BM25/cosine raw scores
remain incomparable. Any lane-weight change requires a separately frozen RRF
arm.

## 11. External implementation guidance

The adopted direction is consistent with official search-system behavior:

- [SQLite FTS5](https://sqlite.org/fts5.html) supports explicit `AND`, `OR`,
  `NOT`, prefixes, NEAR, and column filters. BM25 orders rows only after MATCH
  admits them; FTS5 does not decide the application's natural-language boolean
  policy.
- [Elasticsearch match query](https://www.elastic.co/docs/reference/query-languages/query-dsl/query-dsl-match-query)
  analyzes text and defaults to OR, with optional `minimum_should_match`.
- [Meilisearch matching strategy](https://www.meilisearch.com/docs/capabilities/full_text_search/how_to/use_matching_strategy)
  distinguishes strict all-term matching from staged term dropping and a
  frequency strategy intended for natural-language queries.
- [PostgreSQL text-search controls](https://www.postgresql.org/docs/current/textsearch-controls.html)
  show that an AND-oriented plain-text query is paired with language analysis,
  normalization, and stop-word removal. cidx's current raw normalized-token AND
  does not have that richer analyzer.
- [Sourcegraph code-search syntax](https://sourcegraph.com/docs/code-search/queries)
  exposes explicit symbol/path/filter syntax, while
  [Sourcegraph query assist](https://sourcegraph.com/docs/code-search/query-assist)
  translates natural language into a structured search query instead of
  treating every word as an implicit exact requirement.

These systems do not imply that one universal policy is correct for cidx. They
do show that strict all-term matching is a deliberate precision mode, not a
safe default for unanalyzed agent prose.

## 12. Ordered next work

1. Commit the completed Phase 06 planner and diagnostics so evaluation build
   provenance records `vcs.modified=false`.
2. Rerun the same chi/RHF v2 questions as new provider-free immutable runs.
3. Compare old AND, new symbol/path/descriptive lanes, and their local union at
   the admission and ranking stages. The old run is a historical baseline, not
   overwritten output.
4. Revalidate the Phase 11 RRF integration with the real run evidence; make no
   paid query call without separate approval.
5. Update the critical/general report using the new run IDs and the revised
   stage scorecard.
6. Prepare assistant A/B only after the new lexical behavior and limitations
   are explicit. Assistant A/B remains the later test of marginal MCP
   usefulness, source-reading reduction, answer correctness, and false leads.

No new repository is required for this remediation. The existing approved chi
and React Hook Form repositories are the calibration inputs requested by the
owner.
