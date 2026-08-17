# Phase 07 Measured Retrieval Loop — Revision 4

- State: `in_progress`
- Date: 2026-08-17
- Authority: calibration pool-building only
- Promotion eligible: no
- Labels at this checkpoint: draft; the then-required human passes were not performed
- Fixed serving profile: Voyage `voyage-code-4`, source 1,024, serving 1,024, binary

Current authority supersession: this is historical Binary-era measurement
evidence. Current product evaluation uses default 1024/int8, and fresh frozen
labels use `owner-adopted-dual-ai-v1` with
`OWNER_ADOPTED_DUAL_AI_REVIEW` / `NO_INDEPENDENT_HUMAN_REVIEW`.

## Decision rule

A higher Hit@5, MRR, or NDCG is not sufficient. A candidate advances only
when reviewed requirement groups, complete-query coverage, known hard
negatives, evidence neighborhoods, and later-stage first loss do not regress.
The 32 cases stay in their language and cohort denominators; gains in one
slice never cancel a correctness loss in another.

## Provider-free FTS experiments

Every arm ran twice against the same chi draft-v3 and RHF draft-v2 snapshots.
Each repeat was byte-identical. Tokenization, candidate depth 20, return depth
5, exact-symbol handling, tie order, and all non-FTS lanes were held fixed.

### Production AND versus safe OR 5:1

The production all-token AND expression returned no candidates for all 32
natural-language cases. The evaluation-only safe-token OR expression produced:

| Corpus | Cases | Hit@5 | requirement coverage@5 | complete@5 | MRR@5 | provisional NDCG@5 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| chi | 12 | 0.5833 | 0.5417 | 0.5000 | 0.4583 | 0.4970 |
| RHF | 20 | 0.6000 | 0.5750 | 0.5500 | 0.5083 | 0.5085 |

All OR queries reached the candidate cap, so these values are provisional
pool-building observations rather than evidence that unrestricted OR is a
production policy.

### OR versus minimum-two-token admission

Minimum-two-token admission reduced pre-cap eligible pools, but all 32 queries
still saturated candidate depth. Required-group status was 0 improved, 32
unchanged, 0 regressed. Only chi G08 and RHF T04 changed top-five identities;
both evidence neighborhoods became less useful. The candidate was rejected.

### OR 5:1 versus OR 5:5

Eligible identities and counts were identical query by query, isolating the
field-weight change. Chi recorded 0 improved, 11 unchanged, 1 regressed; G07
lost `selectEncoder`. RHF recorded 3 improved, 15 unchanged, 2 regressed; T04
and X06 lost required implementation parents. Aggregate RHF coverage concealed
that exchange while MRR and NDCG fell. The candidate was rejected.

## Accepted comparator and completed Voyage embedding search

FTS micro-tuning stopped. Safe OR 5:1 remained only an explicit development
comparator and production AND remained unchanged. All 32 queries then ran
freshly and coherently through the shared Voyage query embedding, exhaustive
target-f32, active binary, parent collapse, RRF, ablation, and body-packaging
code. The run reused the fingerprinted document bank, made zero document
provider calls, persisted no query vector, and reused each request-local query
vector only across the arms for that same query.

The clean binary at commit `70bbf1c3b67aa79eaaff4fba495ddbc4e805b6df`
reported `source_modified=false` and executable SHA-256
`52c30efb91d6e6960b25d6c4627c9cecc0ca10d0bc6fc24097f16b0fb79db6bd`.
The coherent series completed 32/32 requests with zero retry or failure, 646
provider-reported tokens, and USD `0.00007752` accounted cost. Chi run
`retrieval-c2d3137b9ae27e5cd502ebd4` has artifact SHA-256
`d20d2083a4f261f9b4384d6d31f9546fe3a0938000ddb09fc5a7a2b2212f4776`;
RHF run `retrieval-81718a0f3b5124aa680c46dc` has artifact SHA-256
`b742a2bd29807af6e93ad8b67b76d2ccce23f1f95397f3ed0e8764ba64896e84`.
All planned retrieval and body stages are `OBSERVED`; assistant use remains
`NOT_OBSERVED`.

| Corpus / arm | FTS OR | target f32 | binary | OR + f32 RRF | OR + binary RRF |
| --- | ---: | ---: | ---: | ---: | ---: |
| chi complete-required@5 | 0.5000 | 0.9167 | 0.9167 | 0.7500 | 0.7500 |
| RHF complete-required@5 | 0.5500 | 0.9500 | 0.8500 | 0.8000 | 0.8000 |

Fusion therefore does not advance. On chi it rescued no binary miss and lost
`routeHTTP` and `ThrottleWithOpts`. On RHF it rescued T01 and X01 but lost T07
`getFieldValue`, T08 `createFormControl`/`Control`, and T09 `Control`.
Aggregate RHF concealed a TypeScript decline from `0.9167` to `0.7500` and a
TSX rise from `0.7500` to `0.8750`. Body packaging introduced no required-group
loss. These are draft, pool-building observations and not serving-policy or
promotion evidence.

## Blind advisory review boundary

The refreshed semantic-parent union is immutable, deduplicated, includes all
truth parents, hides arm/rank/score/prior labels, and uses independent order for
the two passes. Chi contains 191 relations over 12 cases. ChatGPT and Grok each
returned 191/191 source-verified AI judgments with zero missing, duplicate, or
invalid grade/group rows. Reconciliation produced 163 initial exact agreements,
22 exact reconsideration agreements, one final exact agreement, and five
source-only AI adjudications. The final advisory set contains 126 grade-0, 51
grade-1, and 14 grade-2 relations; all 13 existing draft grade-2 parents remain
grade 2. The one additional proposed grade-2 parent is not imported into the
draft required-group topology.

The chi advisory-label SHA-256 is
`031bc3de8bcfbe1df85ac676d7e2bdb66d3fd7037e57373e5fe2bbb74682e1cc`;
the reconciliation-manifest SHA-256 is
`7d7c9c8569e7a9b5f9a37f1732c9de3a6b3bff585dff86306f3b0bb2079cc2f9`;
the provider-free advisory-replay SHA-256 is
`1743d6561bac2508ce3f6280a00293bb1d939e0189b6039251b115e4652c8722`.
The replay reports advisory direct relevance separately from completeness and
first loss under the unchanged draft required-group topology. It cannot select
FTS, codec, fusion, or serving policy.

User delegation authorizes the two AI reviewers to execute the review over
sanitized public-corpus data. It is not a post-result digest-bound human
adoption. The evidence therefore remains
`AI_ADVISORY_CALIBRATION_REPLAY`, `DRAFT`, and
`BLOCKED_HUMAN_REVIEW_PENDING`; it has no calibration-selection,
confirmation, promotion, or release authority.

RHF contains 281 relations over 20 cases. Both AI passes covered 281/281
relations with zero missing, duplicate, or structurally invalid rows. Their
110 initial differences consist of 101 grade-0/grade-1 support-boundary
differences and nine direct-relevance or required-group differences. The nine
direct differences were source-adjudicated under the same full-group rule;
the resulting 22 direct parents are exactly the 22 existing draft grade-2
parents, with no downgrade or newly imported direct parent. The direct-map
SHA-256 is
`5168706e841e333ab42e7d66b5ecaf3d0c19a4415cf4b5f34eceda53ec783fe0`.

The 101 subjective support differences deliberately remain unresolved. Two
complete support-label endpoint maps are retained with SHA-256 values
`e2ef13d05a69cd6e784a058d3894644a1025702135e96a2f613bbd54f7aa8aae`
and
`5cd574beac5dbfbfb41c721babc5033879954521315251d19b8ce31e76f937d3`.
RHF NDCG is therefore a label-sensitivity range, not a confidence interval;
no midpoint and no single reconciled full-label digest exists.

| RHF arm | direct Hit@5 | direct recall@5 | direct MRR@5 | support-sensitive NDCG@5 |
| --- | ---: | ---: | ---: | ---: |
| FTS OR | 0.6000 | 0.5750 | 0.5083 | [0.5208, 0.6019] |
| target f32 | 0.9500 | 0.9500 | 0.6308 | [0.7388, 0.8569] |
| binary | 0.8500 | 0.8500 | 0.6292 | [0.6945, 0.8190] |
| OR + f32 RRF | 0.8000 | 0.8000 | 0.5975 | [0.6638, 0.7394] |
| OR + binary RRF | 0.8000 | 0.8000 | 0.5683 | [0.6469, 0.7252] |

The RHF reconciliation-manifest SHA-256 is
`bb3c514a5238436a0ae844bcc81b313165f5e54dc6b453a181d2356fdd6ae4b4`;
the provider-free replay SHA-256 is
`36ad731e8762e31781819143f1e2413dce28f82762547ad0c63ccddc039806c2`.
Per-query, language, cohort, and global endpoint values are present in that
ignored replay artifact. One attempted Grok reconciliation of all 110
differences substituted 45 unrequested relation identities and is explicitly
invalid for full-support use; its exact nine direct relations were present,
but the accepted direct adjudication is separately digest-bound. No malformed
or substituted relation enters either support endpoint.

## Provenance and authorization boundary

The new development-only policy is a structured, full-field fingerprint. It
records the OR operator, query builder and normalizer versions, SQLite FTS
tokenizer/schema, fields, 5:1 resolved weights, depths, exact-symbol policy,
tie policy, and `production_policy_unchanged=true`. Production CLI, MCP, and
`Search` accept no such option.

Before a provider request, the evaluator requires:

- clean canonical VCS provenance and an executable SHA-256;
- exact corpus, generation, index manifest, profile, materialization, dataset,
  and ordered query-text identities;
- an ordered document-bank fingerprint derived from immutable raw-vector
  identities without serializing vectors;
- exactly 32 series operations, zero document operations, zero reused query
  vectors/rankings, and no query-vector persistence;
- a non-secret authorization reference, dated pricing identity, USD-per-million
  rate, conservative maximum cost, and a cap that contains that maximum.

The experiment manifest schema additionally records all profile fingerprints,
source/serving dimensions and codec, retry-policy fingerprint, actual provider
usage and accounted cost, retrieval/collapse/RRF/body policy identities,
per-query vector hashes, and planned/observed stage states. Any generation or
profile drift invalidates the run. Credentials never enter plans or artifacts.

## Review inputs

The primary metric-contract advisor recommended this coherent all-query run
instead of replaying 31 historical dense ranks with one new query. Two
independent final-direction reviews agreed to stop FTS tuning, keep OR 5:1
provisional, and proceed while tracking saturation, correctness exchanges,
hard negatives, requirement fragmentation, and lexical-to-body first loss.

## Focused implementation checks

Before the clean-provenance execution commit, the following passed:

```text
go test -count=1 ./internal/search ./internal/devlab
go test -race -count=1 ./internal/search ./internal/devlab
go vet ./internal/search ./internal/devlab
go build ./internal/search ./internal/devlab
git diff --check
```

The focused production/evaluation test proves that the public and ordinary
evaluation paths retain AND while only the fingerprinted development policy
uses safely quoted OR tokens. Mutated policy fields, incomplete authority,
and over-cap plans reject before execution. No provider, credential, corpus
mutation, document embedding, query embedding, or artifact publication was
performed at this checkpoint.

## Exact next action

Do not rerun Voyage while only labels change. Preserve the rejected OR-fusion
decision, keep production AND and current dense behavior unchanged, regenerate
the current digest-bound chi/RHF pools, and hand independently shuffled copies
to ChatGPT and Grok. Every relation must be covered and source-inspected;
reconciliation and whole-digest owner adoption must follow the current
contract before a calibration baseline is frozen or another retrieval-policy
comparison is authorized.
