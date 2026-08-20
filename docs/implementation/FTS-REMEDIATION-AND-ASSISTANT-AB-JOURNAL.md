# FTS Remediation and Assistant A/B Work Journal

- Work period: 2026-08-20 through 2026-08-21
- Status: completed calibration remediation and completed assistant diagnostic
- Repositories: the existing owner-approved chi v5.3.1 and React Hook Form
  v7.85.0 snapshots only
- Provider operations: zero for the FTS reruns and assistant A/B
- Promotion authority: none; this journal establishes neither
  `core_retrieval` nor `release_candidate`
- Authoritative status: [`STATUS.md`](STATUS.md)
- Final lexical evidence:
  [`natural-language-lexical-rerun-v2.md`](evidence/phase-07/natural-language-lexical-rerun-v2.md)
- Final assistant evidence:
  [`assistant-ab-v3-result.md`](evidence/phase-14/assistant-ab-v3-result.md)

This document is the durable chronological handoff for the natural-language
FTS correction and the paired Codex CLI assistant experiment that followed it.
It does not replace the immutable question, run, grade, or artifact records.

## 1. Why this work started

The versioned `critical-general-v2` question set combined the retained
behavior questions and exact-identifier questions into 44 cases:

- 12 lexical-anchor cases;
- 24 semantic-only cases; and
- 8 mixed identifier-plus-description cases.

The historical FTS planner completed `10/44`, but the important observation was
not the aggregate score. All 32 semantic and mixed questions produced **zero
FTS candidates**. The planner split the query into normalized tokens, quoted
every token, and joined every token with `AND`. One nonmatching descriptive
word therefore excluded a parent before BM25 could rank it.

The evaluation-only any-token control completed `28/44`. It was not adopted as
a production algorithm, but it proved that useful indexed lexical evidence
existed and that candidate admission, rather than SQLite FTS5 itself, was the
dominant initial defect.

This established two rules that remain binding:

1. candidate admission and ranking must be measured separately; and
2. an FTS-only calibration result must not be presented as proof that the MCP
   is useful or useless to an assistant.

## 2. Question and result versioning

The question sets were versioned before retrieval changes:

| Commit | Work |
| --- | --- |
| `252c168` | created the critical/general v2 question sets, taxonomy, run binding, and reproducibility records |
| `e816a96` | recorded the immutable historical FTS/simple-control diagnostic |

Earlier question versions and run results were retained. Corrections created
new question-set versions or new run IDs rather than overwriting an old result.
Every run is bound to the question-set digest, taxonomy digest, code commit,
corpus commit/tree, and artifact digest through
[`question-set-run-registry-v1.json`](../../testdata/retrieval/question-set-run-registry-v1.json).

The phrase “do not rewrite a result” does not prohibit improving questions in a
new version. It means that an old question version and its grades stay
addressable, while a changed question or truth becomes a new version and new
run.

## 3. Adopted retrieval direction

The all-token planner was replaced by independent local candidate lanes:

```text
raw query
  -> query-shape inference
  -> symbol candidates --------+
  -> path candidates ----------+--> canonical-parent union
  -> descriptive FTS candidates+--> deterministic ordinal fusion --> top k
  -> dense candidates ----------+    (hybrid only and independently authorized)
```

The operative rules are:

- runtime planning infers anchor, descriptive, mixed, or empty shape from the
  query; it never reads evaluation cohort labels;
- exact and normalized qualified/short symbols are candidate sources, not
  merely BM25 tie-breakers;
- explicit paths and path-like expressions use an independent path lane;
- safe descriptive terms use OR admission and retain matched-term coverage
  diagnostics;
- `AND` is reserved for explicitly supplied high-confidence constraints that
  must occur in the same result, not inferred natural-language prose;
- local lanes are deduplicated by canonical parent and fused ordinally;
- dense retrieval remains independent from FTS candidates, so hybrid recall is
  never capped by lexical admission; and
- BM25 and vector scores are not compared as if they shared a numeric scale.

This keeps FTS useful as free local retrieval, one provider in hybrid search,
and offline fallback. It does not make FTS a semantic embedding replacement or
a mandatory dense prefilter.

## 4. Sequential implementation and verification log

### 4.1 Planner v2 and independent lanes

Commit `1e8fad9` implemented:

- safe OR-based descriptive MATCH construction;
- symbol, path, and descriptive FTS candidate lanes;
- deterministic local parent fusion;
- query-plan, lane, term-coverage, and candidate-zero diagnostics;
- serving-policy fingerprinting for the planner version; and
- propagation through FTS, hybrid, evaluation, and MCP response paths without
  changing the stable four-tool MCP surface.

Raw user text never becomes executable FTS grammar. Dense search remains a
parallel provider in hybrid mode.

### 4.2 First real-artifact integrity failures

The first real-corpus preflight failed closed rather than publishing an invalid
artifact:

- `0531540` corrected invalid lexical artifact-field reporting;
- `dcdfd78` propagated the authoritative indexed source SHA into lexical
  candidates.

The first successful planner-v2 run then completed `29/44`. It revealed a
previously passing `Router interface` regression: PascalCase was not being
treated as a possible code anchor.

### 4.3 PascalCase overcorrection

Commit `d352015` classified every leading-capital code-shaped word as an
anchor. It restored the missing case and reached `30/44`, but also explained
ordinary sentence-initial words such as `How` as syntactic anchors. The score
improved while the planner explanation became too broad, so that version was
preserved as an intermediate result rather than accepted.

### 4.4 Final index-resolved weak anchors

Commit `2e1a270` made a leading-capital single word a weak anchor candidate. It
becomes a symbol anchor only when the pinned index snapshot contains the
corresponding symbol. `Router` therefore remains a useful anchor, while
ordinary prose remains descriptive when the repository provides no matching
symbol evidence.

The final immutable run reproduced `30/44` with the intended query shapes and
zero execution failure. Commit `7a15ac2` recorded the complete run lineage and
final interpretation.

## 5. FTS result after correction

Primary lexical metric: `CompleteRequirementHit@5`.

| Arm | Overall | Go | TypeScript | TSX |
| --- | ---: | ---: | ---: | ---: |
| historical all-token-AND | 10/44 | 6/18 | 2/16 | 2/10 |
| historical any-token control | 28/44 | 9/18 | 11/16 | 8/10 |
| final planner v2 | **30/44** | **12/18** | **9/16** | **9/10** |

The final planner retained all ten historical FTS passes and recovered twenty
previous failures. Candidate-zero changed from `32/44` to `0/44`.

| Final stage | Result |
| --- | ---: |
| nonzero local candidate set | 44/44 |
| all required groups present by candidate depth 20 | 38/44 |
| all required groups in returned top five | 30/44 |
| complete at depth 20 but displaced from top five | 8/44 |
| incomplete already at candidate depth 20 | 6/44 |

The initial admission defect is corrected. The remaining fourteen top-five
misses are split into six candidate-generation/provider-depth losses and eight
ranking/depth displacements. They must not be reported as one generic FTS
failure.

Known evaluation gaps remain:

- mixed-signal, multi-requirement, and contract-disambiguation cells are weak
  and small;
- the v2 set contains no real path-shaped query, so the production path lane
  has synthetic validation but no real-corpus usefulness result;
- the hard-negative denominator is one; and
- no paid dense/hybrid v2 comparison was authorized.

## 6. Assistant A/B chronology

The FTS score did not answer whether an AI assistant benefits from cidx. Three
versioned paired protocols therefore used the same 12 selected critical-cohort
questions from the existing repositories.

### 6.1 Version 1: optional-tool adoption failure

Version 1 exposed cidx optionally. No treatment assistant used it. The run is
retained as an adoption/interface pilot, not a retrieval-quality result.

### 6.2 Version 2: ambiguous tool identity

Version 2 required a first cidx search, but only `3/12` treatment tasks used the
MCP. Nine assistants interpreted “cidx search” as a shell command or performed
a shell availability check. Those observations remain operationally ungradable
and were not selectively replaced. Both side-panel reviewers approved a full
new version with the exact MCP identity and explicit shell-command prohibition.

### 6.3 Version 3: compliant paired diagnostic

Version 3 froze:

- official Codex CLI `0.148.0`;
- `gpt-5.6-sol` with high reasoning for task execution;
- 12 paired questions, balanced arm order, and fresh opaque source/state for
  every turn;
- a byte-identical shared prompt;
- baseline with no MCP and treatment with only the exact cidx MCP;
- treatment FTS-only, with no Voyage credential or paid query;
- blind correctness grading after an arm-blind machine journey freeze; and
- no selective retry.

The execution plan remains byte-identical to its run-manifest SHA-256
`e408ba6b79dbc9c0ccb4938d48de8244c59bc9e23f139c2ee2d920257ff78113`.
The run ID is `assistant-ab-v3-20260820T143000Z`.

| Measure | Baseline | cidx FTS |
| --- | ---: | ---: |
| complete answers | 11 | 12 |
| partial answers | 1 | 0 |
| total model tokens | 1,223,579 | 1,678,341 |
| uncached input tokens | 241,613 | 363,843 |
| compliant treatment tasks | — | 12/12 |
| cidx calls | — | 48 |

There was no complete-to-noncomplete reversal. One question,
`chi-g06-basic-auth`, changed from partial to complete because the cidx journey
retrieved and cited the full helper instead of making an unsupported
no-response-body claim.

The current integration did not meet the frozen efficiency rule:

- cidx total model tokens were `37.2%` higher;
- uncached input was `50.6%` higher;
- the median treatment/baseline model-total ratio among 11 dual-complete pairs
  was `1.378`; and
- only `3/11` dual-complete pairs were non-increasing.

Every treatment exposed more repository-output bytes. Searches commonly used
`k=10`, `max_inline_bytes=12,000` or `20,000`, and serialized roughly 34–75 KB
before later `read_span` calls. Schema-only probes differed by two model tokens,
so static MCP schema overhead does not explain the increase.

The transcript records both textual and structured MCP result forms. That is a
serialization lead to investigate, not proof that both forms entered model
context. Official token usage and the frozen model-visible output measure are
the current authority.

## 7. What the assistant result means

The supported conclusion is deliberately narrow:

> Correctness was preserved and one bounded question improved, but the current
> forced FTS-first assistant integration is not token-efficient.

The forced first call was a diagnostic intervention used to obtain treatment
compliance and isolate the current integration. It is not the intended
release-candidate assistant policy; Phase 14's final marginal-usefulness test
still requires ordinary tool choice rather than forcing cidx.

It does **not** establish that:

- cidx is generally inaccurate or inefficient;
- FTS planner v2 should be reverted;
- dense or hybrid is necessarily cheaper;
- one correctness conversion generalizes beyond these questions;
- the MCP is ready or permanently blocked from release; or
- the calibration repositories can supply promotion evidence.

The critical-cohort breakdown is diagnostic. Lexical-anchor tasks were the
best current FTS-first fit: their dual-complete median model ratio was `0.832`
and `2/3` were non-increasing. Semantic-only, contract-disambiguation,
multi-requirement, and the one hard-negative task required larger journeys.
The cells overlap and are small, so they guide the next correction without
becoming runtime labels or release thresholds.

Both external reviewers returned `CORRECT_BEFORE_NEXT_AB`. They rejected an
unchanged rerun and ranked response-volume correction first, tool guidance
second, and lexical-versus-semantic routing later.

## 8. Next-work direction

### Stage A — measure the response contract before changing retrieval

This is the exact next action.

1. Trace what `internal/mcp` serializes for `search` and what the Codex CLI
   records as model-visible tool output.
2. Measure text, structured content, metadata, inline source, and escaping
   overhead separately on the frozen V3 treatment calls.
3. Determine whether equivalent text/structured representations are actually
   both supplied to the model; do not infer this from the transcript shape.
4. Produce a proposed compact response contract and before/after byte estimate.

No ranking, query planner, corpus, or paid provider changes belong to Stage A.

### Stage B — smallest response/orchestration correction

Subject to the external-contract review required below:

- preserve result IDs, rank, paths, parent ranges, freshness, and source hashes;
- keep search metadata compact and retrieve decisive source through targeted
  `read_span`;
- avoid semantically duplicate response representations when the host contract
  permits it;
- evaluate smaller `k` and inline-body budgets as frozen experiment values,
  not silently accepted product defaults;
- guide the assistant toward one broad search followed by targeted reads and a
  bounded number of refinements; and
- preserve the four stable MCP tools and rank invariance under body budgets.

Candidate values such as `k=3–5` or a smaller inline budget came from the
reviewers. They are experiment candidates, not current configuration decisions.

Changing the public MCP result schema is an external-contract decision. Before
editing it, update the canonical design, Phase 13/14 contracts, wire schemas,
and compatibility plan, then obtain owner confirmation. A payload-only change
that preserves the existing schema still requires a focused host compatibility
review.

### Stage C — Version 4 paired assistant experiment

After the response correction is frozen:

- preserve Version 1–3 and create a new V4 manifest and run ID;
- reuse the same 12 questions, corpora, model, reasoning effort, arm order,
  isolation, blind grader contract, and journey reducer unless a separately
  recorded reason requires a new version;
- change only the declared response/tool-guidance intervention;
- run the complete 24-turn batch, never selective replacements;
- compare correctness, model-total tokens, uncached input, inspection actions,
  model-visible output bytes, calls, cited paths, and false claims; and
- keep the result diagnostic rather than promotion evidence.

An unchanged V3 repeat is not justified. Repeated stochastic estimates may be
designed later, but they do not precede correction of the measured structural
payload problem.

### Stage D — routing only after volume is controlled

If semantic and multi-step tasks remain expensive after Stage C, design a
separate routing experiment:

- infer query structure at runtime; never route by evaluation cohort name;
- retain FTS-first for clear lexical anchors;
- compare explicitly authorized hybrid/dense behavior for descriptive or
  multi-requirement queries;
- keep dense independent from FTS candidate admission; and
- request explicit paid-query approval before any Voyage execution.

Do not acquire a new repository for these stages. A new owner-selected corpus
is a later confirmation decision, only if the existing diagnostic cannot answer
the next bounded question.

## 9. Resume and stop checklist

Before any implementation resumes:

1. Read `EXECUTION-GUIDE.md`, `README.md`, `STATUS.md`,
   `EVALUATION-CONTRACT.md`, this journal, Phase 13, Phase 14, and the V3 result.
2. Verify the V3 plan SHA-256 and preserve all V1–V3 manifests and results.
3. Confirm the worktree and exact branch; existing changes belong to the owner.
4. Start with Stage A response accounting, not another A/B or a ranking change.
5. Do not edit the critical/general v2 question truth to improve results. A
   changed question becomes a new version and run.
6. Do not use Voyage, hybrid, a new repository, or a new MCP tool without the
   corresponding explicit authorization.
7. Stop before a public MCP schema change, config-default change, database
   migration, paid action, or release/promotion claim and obtain the required
   decision.
8. Record checks actually run and checks omitted; do not convert a diagnostic
   observation into a release gate.

## 10. Evidence map

| Topic | Record |
| --- | --- |
| historical cohort baseline | [`critical-general-question-set-v2.md`](evidence/phase-07/critical-general-question-set-v2.md) |
| adopted lexical architecture and external review | [`natural-language-fts-query-planner-review-r4.md`](evidence/phase-07/natural-language-fts-query-planner-review-r4.md) |
| sequential code/run lineage | [`natural-language-lexical-rerun-v2.md`](evidence/phase-07/natural-language-lexical-rerun-v2.md) |
| V1 optional-adoption plan | [`ASSISTANT-AB-TEST-PLAN-V1.md`](ASSISTANT-AB-TEST-PLAN-V1.md) |
| V2 interface-compliance result/plan | [`ASSISTANT-AB-TEST-PLAN-V2.md`](ASSISTANT-AB-TEST-PLAN-V2.md) |
| V3 frozen execution plan | [`ASSISTANT-AB-TEST-PLAN-V3.md`](ASSISTANT-AB-TEST-PLAN-V3.md) |
| V3 final result and artifact digests | [`assistant-ab-v3-result.md`](evidence/phase-14/assistant-ab-v3-result.md) |
| operational phase authority | [`STATUS.md`](STATUS.md) |

## 11. Validation already performed and deliberately absent

FTS correction evidence records focused normal/race tests, vet, the CLI build,
real-corpus provider-free reruns, artifact checksums, and clean-VCS binary
identity. The V3 assistant evidence records 24 scored task turns, two schema
probes, fresh per-turn isolation, blind grading, machine-frozen journeys,
manifest/schema validation, and two external pre/post reviews.

Not performed or claimed:

- no Voyage request, paid embedding, or hybrid assistant arm;
- no new corpus;
- no broad project test suite for this documentation/A/B handoff;
- no repeated estimate of model stochasticity;
- no product response-compaction implementation; and
- no `core_retrieval` or `release_candidate` promotion.
