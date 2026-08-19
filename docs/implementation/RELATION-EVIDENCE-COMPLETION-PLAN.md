# Relation Evidence Completion: Semantic Admission, Contract Closure, and Assistant Pull

- Status: historical; Stage A–F completed; assistant-pull A/B deferred
- Current experiment authority:
  [RELATION-PACKAGING-NEXT.md](RELATION-PACKAGING-NEXT.md)
- Current handoff:
  [RELATION-ASSISTANT-VALIDATION-HANDOFF.md](RELATION-ASSISTANT-VALIDATION-HANDOFF.md)
- Date: 2026-08-18
- Owning phases: Phase 07 calibration and relation evidence, Phase 12 core
  retrieval evaluation, and later Phase 14 assistant-use evidence
- Historical record:
  [Relation-Graph Experiment Journal](RELATION-GRAPH-EXPERIMENT-JOURNAL.md)
- Product baseline: exhaustive `1024/int8` retrieval, separate FTS control,
  exactly four MCP tools
- Current authorization: documentation and corpus-independent, provider-free
  evaluator implementation only
- Not authorized by this document: selecting or downloading a new corpus,
  document embedding, query embedding, assistant/API execution, production
  graph storage, production search changes, or MCP schema changes

## 1. Purpose

The completed chi/RHF graph series proved exact relation recovery and bounded
frontier construction, but it did not produce a sufficiently precise server
admission policy. The exposed 32-case calibration set is closed. This plan
defines the next experiment without using those cases to select another
threshold, edge weight, closure rule, or hint budget.

The next series tests three separate questions:

1. Can the request's existing dense semantic scores identify a relevant
   endpoint inside a compiler-resolved bounded relation frontier?
2. Can a small mechanically defined contract closure improve interpretive
   completeness without turning into unconditional graph expansion?
3. Does metadata-only relation disclosure plus the existing `read_span` tool
   outperform automatic body push in an assistant workflow?

These are different questions and produce different evidence. Server-side
retrieval quality belongs to Phase 07/12. Assistant tool use belongs to the
later Phase 14 evidence boundary and runs independently of whether server push
looks precise.

## 2. Fixed conclusions from the closed series

The following decisions are inputs, not hypotheses to reopen:

- exact compiler-resolved `CALLS`, `TYPE_REF`, and `MEMBER_OF` facts are useful
  evaluation evidence;
- the complete occurrence database remains an ignored evaluation sidecar;
- dense top five remains protected primary evidence;
- graph-first-before-dense is rejected under the measured policy;
- raw frequency, incoming popularity, and unconditional specificity are not
  relevance authority;
- the top-two-per-anchor/direction/tier frontier is a provisional development
  complexity control, not a product threshold;
- relation-text embedding remains rejected;
- no HNSW/ANN layer is required for the exhaustive local int8 scan;
- no further admission or pruning policy may be tuned on the exposed 32
  cases; and
- the old 32 may be replayed only after a new policy decision as historical
  regression evidence. They cannot select or modify that policy.

The accepted negative result remains: graph reachability is solved for the
representative misses; query-conditioned relevance and evidence admission are
not.

## 3. Corrected diagnosis

The old policies were not completely query-independent. They used lexical and
ordinal query signals:

- normalized query-to-symbol overlap;
- nearby occurrence identifiers;
- `props`/`contract` intent;
- dense top-20 rank; and
- query-token coverage for anchor selection.

They did not use the continuous dense similarity of graph endpoints. The
unmeasured signal is therefore specific:

```text
same request-local query representation
+ same active int8 document vectors
+ graph-filtered endpoint parent scores
```

The graph supplies a small, compiler-proven candidate universe. Dense scoring
supplies natural-language relevance inside that universe. This is candidate
filtering plus semantic gating, not RRF lane fusion and not relation-text
embedding.

## 4. Existing data path to reuse

No production scorer change is needed to run the evaluation.

The existing Phase 12 retrieval adapter already performs one exhaustive
active-int8 scan for each approved query and writes every segment observation
to `dense-segment-candidates.jsonl`. It also writes collapsed parent rankings,
the query-vector digest, run manifest, provider usage, and artifact checksums.

The new provider-free relation pass consumes those immutable outputs:

```text
one approved query embedding operation
-> request-local query f32
-> one prepared int8 query
-> all active segment scores
-> dense-segment-candidates.jsonl
-> provider/query vector destroyed

provider-free relation completion pass
-> verify retrieval artifact and graph checksums
-> collapse every segment with the existing parent identity/tie contract
-> derive global parent score/rank/distribution
-> join bounded relation endpoint parents
-> emit semantic, closure, and hint evidence
```

This design makes no repeated Voyage call for another graph arm. It persists
native scores and the existing query-vector SHA-256, never query-vector bytes.
It must use the serving active-int8 arm, not the target-f32 reference, as the
ordinary product experiment.

## 5. Experiment isolation

### 5.1 Evaluation-only boundary

The first implementation is confined to:

- `internal/relationdiag` for score-artifact validation, parent collapse,
  relation joins, closure/hint derivation, label-late analysis, and immutable
  artifact publication;
- `internal/devlab/relations.go` for a development-only command;
- narrowly required generalization of Phase 12 experiment planning and
  artifact validation in `internal/devlab`, removing assumptions that every
  approved series has exactly 32 queries or requires the historical
  `safe-token-or-v1` FTS comparator; and
- tracked plans, manifests, datasets, review records, and evidence after the
  user approves new corpora.

The implementation must not modify or become a dependency of:

- production index/store schemas;
- production vector scoring or search ranking;
- FTS/RRF behavior;
- stable MCP schemas or the four-tool registry;
- source-vector bank ownership; or
- document/query embedding request semantics.

### 5.2 Stable MCP boundary

The stable tool list remains exactly:

```text
status
search
read_span
reindex
```

No `expand_relations`, `find_callers`, `find_callees`, or fifth tool is added.
The initial assistant experiment uses a dev-only adapter that presents
body-free hints and invokes the existing `read_span` contract. Any later
product `search` response change requires an explicit design and phase
reconciliation after assistant evidence exists.

## 6. Semantic endpoint evidence

### 6.1 Parent scoring

Vectors are segment-level. A semantic parent score is not one endpoint dot
product. For every query the evaluator must:

1. load the complete active-int8 segment-score row;
2. map each segment to the bound current semantic parent;
3. select the same best segment using the production score and stable tie
   order;
4. order every parent, not only the old top 20;
5. retain equal-score/boundary ambiguity explicitly; and
6. join graph endpoints by portable path/hash/kind/qualified-symbol/range
   identity, not generated database row ID.

### 6.2 Recorded semantic features

For each query and graph endpoint record:

- global collapsed-parent rank and parent count;
- rank percentile with exact numerator and denominator;
- native active-int8 score, labelled as a codec-native similarity rather than
  probability or confidence;
- anchor score and endpoint-minus-anchor gap;
- best and runner-up graph candidate scores and ambiguity gap;
- query-distribution summary required for normalization;
- normalized gap values only under a versioned formula;
- relation kind, direction, structural tier, and canonical relation identity;
- whether the endpoint is already in primary top five, dense top 20, or absent
  from both; and
- deterministic semantic ordinal.

Raw cosine multiplication or ratio thresholds are prohibited. Cosine may be
negative or near zero and is not a calibrated confidence. The calibration
unit may select only from this finite rule family:

- global parent rank or percentile ceiling;
- endpoint-to-anchor normalized gap;
- graph top-one/top-two normalized ambiguity gap; and
- default abstention when the relevant denominator is degenerate or a frozen
  margin is not met.

The exact normalization formula, margins, and tie behavior must be selected on
new calibration and frozen before confirmation.

### 6.3 Semantic arm

The semantic arm starts from the existing final bounded frontier. It does not
reopen the graph builder or current cap policy.

```text
protected primary results
-> final bounded one-hop frontier
-> endpoint parent semantic features
-> frozen semantic gate
-> zero or one related-evidence bundle
-> existing count/body packaging controls
```

The first implementation records all features before a gate is selected. A
single approved calibration query capture can therefore support provider-free
policy analysis without another query operation.

## 7. Contract closure evidence

### 7.1 Principle

Closure candidates are divided by a case-independent principle:

```text
interpretive completion
  declarations/types needed to understand the retrieved signature

behavioral expansion
  callers, callees, helpers, uses, tests, and control context beyond the
  declaration
```

Only the first class is eligible for the contract-closure experiment.
Executable dependencies remain query-conditioned semantic candidates or
assistant-pull hints.

### 7.2 Candidate inventory

A closure candidate must be:

- one hop and outgoing from a primary parent;
- `RESOLVED_UNIQUE` and in corpus;
- `TYPE_REF` in a mechanically declared signature/contract role;
- attached to a production parent;
- mapped to a complete current semantic-parent body;
- absent from the protected primary result set;
- parent-deduplicated;
- cycle-free for this request; and
- independent of corpus, query ID, symbol suffix, React, or G09/X08-specific
  exceptions.

The calibration inventory records value-parameter, return, generic-constraint,
and heritage/signature roles separately. A role does not become an automatic
closure merely because it is present in the inventory.

### 7.3 Dual budget

Byte budget alone is insufficient because many small types can still flood a
response. Count alone is insufficient because one large target can consume
the entire context. Every closure policy therefore freezes both:

- maximum distinct closure parents per primary result and per request; and
- maximum aggregate complete-body bytes.

There is no transitive closure, recursive type walk, or backfill. When a count,
byte, body-completeness, or cycle condition fails, record the omission reason.

Calibration may select caps only from a finite predeclared grid. Confirmation
uses one frozen pair. The caps are evaluation controls until separate product
evidence exists.

## 8. Relation hint evidence

### 8.1 Hint is not body push

Hint disclosure moves the failure cost from source-body contamination to a
smaller attention and metadata budget. It does not eliminate selection cost.
The current observed maximum of 11 hints is corpus-specific, while the
development hard ceiling is 32 and was never exercised by chi/RHF.

The hint experiment therefore enforces both:

- maximum distinct hints; and
- maximum serialized hint bytes.

### 8.2 Hint fields

Each hint contains only:

- stable relation identity;
- kind, direction, and structural tier;
- source occurrence path/hash/range;
- target symbol and qualified symbol;
- target path/hash/range;
- already-primary and dense-depth flags;
- relation occurrence count or compact structural metadata where declared;
- deterministic semantic ordinal; and
- omission/truncation status.

It contains no source body, raw query vector, document vector, relation prose,
raw similarity advertised as confidence, relevance label, hard-negative label,
or absolute path.

### 8.3 FTS-only behavior

The first experiment supplies semantically ordered hints only when the same
request has a valid active-int8 query representation. FTS-only has no query
vector and therefore returns no semantic relation-hint arm in this series.
Do not silently substitute structural popularity or the rejected graph-first
ordering.

### 8.4 Actuation

The assistant receives body-free hints through the dev-only harness. It may
choose an exact target and call the existing `read_span` with the bound
path/hash/range. The server does not force relation expansion.

## 9. Datasets and cohort structure

### 9.1 Closed historical data

The frozen chi/RHF 32-case set may not:

- reveal top-20-outside endpoint scores for policy design;
- select semantic margins;
- select closure roles or caps;
- select hint count/byte budgets;
- select assistant prompts or tool policy; or
- vote for confirmation or promotion.

After the new policy and confirmation result are sealed, it may be replayed as
historical regression evidence without changing that policy.

### 9.2 New calibration unit

The user selects every repository, pinned commit, license, language slice, and
local acquisition authorization. Before any score is opened, the calibration
plan freezes:

- repository and content identities;
- query count and cohort matrix;
- relation-challenge and naturalistic-prevalence slices;
- question texts and durable source truth;
- pool depths and candidate union;
- feature family, closure-role inventory, count/byte grids, and hint schema;
- provider operation count and spend cap; and
- review/adoption protocol.

The relation-challenge slice deliberately includes representative patterns:

- caller/callee and public-to-private helper;
- component or function to parameter/return contract;
- interface to implementation or implementation to interface;
- function to test or example;
- configuration/literal to consumer;
- recursive/paired declarations;
- multiple possible callers/implementations;
- generic hub and popular but irrelevant targets; and
- hard negatives with valid nearby graph relations.

The naturalistic-prevalence slice estimates how often these gaps occur in
ordinary code-search questions. It must not be replaced by an all-edge-case
challenge benchmark. Report both denominators separately; do not combine them
into a weighted quality total.

### 9.3 Independent confirmation

After calibration selects one policy, freeze a separate unexposed confirmation
unit. It must meet the canonical floors:

- at least 90 answerable queries, with at least 30 each for Go, TypeScript,
  and TSX;
- at least 18 verified `ABSTAINABLE` or hard-negative queries, with at least 6
  per language;
- at least 10 cases in every critical cohort; and
- genuine mixed-language cases.

These are denominators, not an unconditional claim that at least 108 unique
questions are required. A verified hard-negative case may also be answerable,
and cohorts overlap. Do not pad the set with narrow microcases merely to raise
the unique count. Freeze the actual overlap matrix and unique count before
pool generation.

A second repository per language is recommended for broader generalization,
but no repository may be selected or acquired without user approval.

## 10. Label and score separation

Every new dataset uses the existing solo-project authority:

```text
protocol_version    owner-adopted-dual-ai-v1
relevance_authority OWNER_ADOPTED_DUAL_AI_REVIEW
review_validation   NO_INDEPENDENT_HUMAN_REVIEW
```

Candidate pooling may use the complete union of declared retrieval and
relation arms. The two AI reviewers receive source-complete, separately
shuffled packets with rank, score, arm, prior label, experiment outcome, the
other pass, and owner preference hidden. Reconciliation and owner adoption
freeze the label payload before policy scoring.

Calibration labels may select policies and margins. Confirmation labels may
not. Confirmation pool generation is recorded as non-promotional preparation;
formal scoring happens once after policy, margins, arms, denominators, and
artifact schemas are sealed.

## 11. Calibration execution sequence

### Stage A — freeze infrastructure contract

- finalize the scorer-artifact input contract;
- finalize semantic feature formulas available for calibration;
- finalize closure-role inventory and finite dual-cap grid;
- finalize hint fields and finite count/byte grid;
- finalize artifact names, checksums, first-loss states, and stop conditions;
- implement and review corpus-independent provider-free plumbing.

Exit: implementation is committed and validated without corpus, provider, or
production-path mutation.

### Stage B — user-approved corpus preparation

- obtain explicit repository/commit/license approval;
- create tracked portable manifests and ignored local bindings;
- verify clean checkout and content hashes;
- run free init/index/parser/chunker inventory;
- build the existing evaluation-only relation sidecar;
- inventory relation resolution and frontier distributions.

Exit: clean provider-free corpus and graph evidence exists. No document or
query provider call has occurred.

### Stage C — document source and serving coverage

If the approved corpus lacks complete document source-bank coverage:

- produce an exact document-capture plan;
- request separate user approval;
- execute only within the approved source/model/token/cost boundary;
- preserve validated 1024-f32 document source rows; and
- materialize the ordinary 1024/int8 serving state locally.

Exit: complete source-bank and active-int8 coverage are bound to the corpus
generation. Document authorization does not authorize query operations.

### Stage D — calibration pool capture

- freeze exact dataset/query count and series plan;
- request explicit calibration pool-query authorization;
- execute one logical query operation per case;
- reuse its query representation across every declared retrieval arm;
- write all active-int8 segment scores and existing retrieval artifacts; and
- perform no relation selection in the provider request path.

Exit: immutable score artifacts exist; query vectors do not.

### Stage E — blind pool and label freeze

- join retrieval candidates, complete bounded relation frontier, closure
  inventory, and hint candidates;
- create source-complete blinded review packets;
- complete both AI passes, reconciliation, and whole-digest owner adoption;
- freeze the calibration label payload.

Exit: every candidate used by calibration has source-backed authority.

### Stage F — provider-free policy selection

- derive semantic endpoint features from the immutable full score artifact;
- compare only the predeclared semantic feature family;
- compare closure role/cap cells and hint budget cells;
- report usefulness, noise, hard negatives, bytes, and abstention separately;
- select one policy and paired margin set; and
- freeze policy fingerprint, caps, formulas, arms, and confirmation gates.

Exit: calibration is exposed and closed. No further query provider operation
is required for policy fitting.

## 12. Confirmation execution sequence

1. User selects and approves distinct confirmation repositories and commits.
2. Author independently from different intents and source areas; do not reword
   calibration failures.
3. Freeze the coverage/overlap matrix, pool plan, and provider approval packet.
4. Run only blinded pool-generation operations needed for labels.
5. Complete dual-AI review, reconciliation, and whole-digest adoption without
   showing aggregate policy results.
6. Seal policy, margins, arms, denominators, promotion contract, and artifact
   schemas.
7. Request a distinct formal confirmation-query authorization.
8. Execute formal confirmation once.
9. Produce `PROMOTION_EVIDENCE_READY` or `NOT_PROMOTION_READY` for the scoped
   Phase 12 core result; never tune after exposure.

While relation output remains a development sidecar rather than an implemented
product path, its confirmation result is design evidence. It cannot claim that
production `core_retrieval` already serves relation evidence.

## 13. Independent assistant-use A/B

Assistant infrastructure may be prepared in parallel after the hint artifact
schema freezes. Actual A/B execution is independent of server-push precision
and uses fixed assistant model/version, prompt, existing tools, task order,
context/tool budgets, corpus snapshot, expected outcomes, repetitions, and
spend control.

Required arms are:

1. existing assistant tools only;
2. existing tools plus lexical cidx;
3. existing tools plus unchanged hybrid cidx;
4. existing tools plus bounded closure evidence;
5. existing tools plus body-free relation hints and `read_span`; and
6. existing tools plus closure and hints.

Do not force a cidx or relation-hint call. Record:

- final task and required-group success;
- exact relation hints inspected;
- correct and wrong edge expansions;
- hard-negative and `walkXFF` hint following;
- `read_span` calls and source bytes;
- total context/tool bytes;
- tool calls, latency, failures, retries, tokens, and cost; and
- assistant first loss.

The result chooses among server push, agent pull, a combination, or no product
graph path. It does not rewrite the Phase 12 core result.

## 14. Immutable artifacts

The provider-free relation completion run writes, at minimum:

```text
run-manifest.json
input-artifact-binding.json
semantic-parent-scores.jsonl
relation-endpoint-features.jsonl
contract-closure-candidates.jsonl
relation-hints.jsonl
semantic-admission-results.jsonl
closure-package-results.jsonl
per-query-relation-trace.jsonl
aggregate-relation-metrics.json
cohort-language-report.json
first-loss-report.json
report.md
artifact-checksums.json
```

The run manifest binds:

- corpus, dataset, label, code, graph, retrieval-artifact, config/profile,
  generation/manifest, and policy fingerprints;
- serving dimension and int8 codec;
- query-vector hashes, never vectors;
- score-collapse and normalization policy;
- frontier, closure, hint, body, and byte caps;
- evidence class and promotion eligibility;
- provider operations for the relation pass, which must be zero; and
- `NO_INDEPENDENT_HUMAN_REVIEW`.

## 15. Measurements and first loss

Do not create a weighted total. Report separately:

- dense baseline and protected-primary equality;
- relation reachability;
- semantic gate admission and abstention;
- closure candidate, admitted, omitted, and body-complete counts;
- hint count/bytes and truncation;
- required-group coverage and complete evidence;
- useful, noise-only, grade-2, grade-1, grade-0, and unreviewed attachments;
- known hard-negative and `walkXFF` exposure;
- graph endpoint global-rank/percentile distribution;
- top-one/top-two ambiguity distribution;
- per-language and per-cohort denominators;
- query/provider usage from the bound retrieval run; and
- local relation-pass latency and memory as observations, not initial gates.

Diagnostic-local first-loss additions may include:

```text
SEMANTIC_SCORE_BINDING
SEMANTIC_GATE
SEMANTIC_AMBIGUITY_ABSTENTION
CONTRACT_CLOSURE_ELIGIBILITY
CONTRACT_CLOSURE_PARENT_CAP
CONTRACT_CLOSURE_BYTE_CAP
RELATION_HINT_BUDGET
ASSISTANT_HINT_SELECTION
ASSISTANT_READ_SPAN
```

They do not change the normative public evaluation wire unless a later design
adopts the relation path.

## 16. Implementation gates

Before any real corpus run, one consolidated implementation boundary must
prove:

- exact checksum verification of graph and retrieval input artifact sets;
- corpus/dataset/generation/profile/query-hash compatibility;
- every segment score maps to one current semantic parent;
- parent collapse and stable ties match the existing search evaluation path;
- query vectors and document vectors are absent from relation artifacts;
- relation evaluation makes zero provider operations;
- labels do not influence candidate generation, score features, closure
  eligibility, or hints;
- primary top five remains byte-identical;
- closure parent and byte caps are both enforced;
- hint count and serialized-byte caps are both enforced;
- deterministic repeat artifact hashes;
- failures and omissions remain in denominators;
- production search/MCP/vector packages do not import `relationdiag`;
- exactly four MCP tool schemas remain unchanged; and
- no root-relative escape, symlink traversal, source-body leakage, credential,
  or absolute path enters a portable artifact.

Write only core tests needed to prove these contracts. Run one normal/race,
vet, build, format, dependency-boundary, module, and diff validation at the
implementation commit boundary rather than repeatedly validating each edit.

## 17. Stop conditions requiring user input

Stop before each of these actions until the user supplies the exact required
authority:

- selecting new repository identities, commits, licenses, or language slices;
- cloning, downloading, or updating a corpus;
- choosing a genuine mixed-language corpus;
- sending new corpus source to Voyage for document embedding;
- running calibration pool query embeddings;
- running confirmation pool query embeddings;
- running the formal confirmation query series;
- selecting assistant model, host, API, task set, repetition count, and spend;
  and
- changing the production search/MCP/storage contract.

The project-local credential file and the user's account billing cap are
operational safeguards, not authorization for a new corpus or operation.

## 18. Decision rules

After calibration and confirmation:

- adopt semantic server admission only if it improves evidence without
  protected-slice, hard-negative, noise, or body-budget regression;
- adopt contract closure only for mechanically defined role classes whose
  separately reported usefulness and false-attachment evidence passes the
  frozen gate;
- prefer agent pull when hints plus `read_span` improve final tasks with lower
  context cost and acceptable wrong-expansion behavior;
- prefer server push when assistant hint use is unreliable but closure or
  semantic admission is precise under frozen evidence;
- combine closure and hints only if the paired assistant arm improves the
  final task rather than merely retrieval proxies; and
- retain no product graph path when confirmation or assistant gates fail.

No positive result silently edits production. Product adoption requires a
separate generation-bound storage/search/wire plan and fresh compatible
evidence. No fifth MCP tool is planned.
