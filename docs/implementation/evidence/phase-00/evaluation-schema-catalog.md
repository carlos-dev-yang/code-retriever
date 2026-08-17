# Phase 00 Evaluation Schema and Enum Catalog

Phase 02 owns strict versioned wire types and schemas. Phases 07 and 12 own truth mapping, metric calculations, and runners. Phase 14 adds assistant/host evidence without rewriting earlier immutable results.

## Required schema families

| Schema | Required identity and controls |
| --- | --- |
| Query contract | query ID/text, language/cohorts, answer mode, required groups, OR alternatives, relevance grades, hard negatives, durable file/hash/symbol/spans, review passes, calibration/confirmation split, digest |
| Corpus manifest | corpus ID, upstream URL, pinned commit, SPDX/license evidence, root subdirectory, languages, include/exclude policy, clean-tree rule, tree/content hash |
| Run manifest | corpus/query/code/config/profile/generation/MCP fingerprints, arm definitions, candidate/collapse/RRF/body policy, platform and assistant controls, start/end state |
| Per-query stage trace | lane candidates/ranks/native scores, collapse mapping, RRF components, package ranges/bytes, assistant presented/used evidence, operation outcomes, exact denominators |
| Artifact manifest | run-relative paths, media/schema type, byte size, SHA-256, completion marker; no absolute local path |
| Promotion contract | scope, frozen applicable gates/margins/cohorts/denominators, calibration evidence digests, confirmation inputs |
| Promotion result | immutable scope/status, prerequisite digests, passed/failed gates, first-loss/cohort evidence, incomplete/not-observed reasons |

## Stable enums

### Language, answer, relevance, and split

- Language: `go|typescript|tsx|mixed`
- Answer mode: `SINGLE|BEST_N|EXHAUSTIVE|ABSTAINABLE`
- Relevance: `0=irrelevant/wrong/stale/hard-negative`, `1=useful support`, `2=direct requirement`
- Dataset split: `calibration|confirmation`
- Promotion scope: `core_retrieval|release_candidate`
- Promotion status: `PROMOTION_EVIDENCE_READY|NOT_PROMOTION_READY`

### First loss

- `SOURCE_DISCOVERY`
- `PARSE_OR_CHUNK`
- diagnostic lane states `FTS_CANDIDATE_MISS|DENSE_SEGMENT_MISS`
- `PROVIDER_UNION_MISS`
- `SEGMENT_PARENT_COLLAPSE`
- `RRF_FUSION`
- `BODY_PACKAGING`
- `ASSISTANT_USE`
- `ASSISTANT_RESOLUTION`
- `OPERATION_FAILURE:<stage>`
- `NOT_OBSERVED`

The primary retrieval first loss is `PROVIDER_UNION_MISS` only when neither provider lane contains valid gold. `NOT_OBSERVED` is not numeric zero. Required failures and timeouts remain in denominators.

## Metric ownership

- Human relevance owns Hit/Recall/MRR/NDCG, requirement coverage, complete hit, known-hard-negative hits, and assistant usefulness.
- Exhaustive serving-dimension f32 owns current int8 representation fidelity: retention, missing candidates, displacement, inversion, and ties. Historical Binary metrics remain immutable reports only.
- FTS, dense, union, collapse, RRF, body, and assistant stages keep separate denominators and first-loss attribution.
- BM25, cosine, int8, and RRF raw scores are never compared directly or interpreted as confidence.
- No weighted total exists.

## Hard-gate framing

- Zero/100% invariants cover completeness, byte/span/body fidelity, generation/profile/codec isolation, deterministic rank, rebuild equivalence, query-vector nonpersistence, and lab/runtime isolation.
- Numeric noninferiority margins are learned only from calibration baselines and frozen before confirmation.
- Operational latency/size/cost is observational until a budget is frozen before results.
- Phase 12 emits immutable `scope=core_retrieval`; Phase 14 emits a new `scope=release_candidate` result referencing the core digest.

## Explicit exclusions

- No HNSW/ANN recall, graph-health, or `ef_search` schema.
- No generated judge as truth authority.
- No persistent query-vector field.
- No weighted total or automatic profile activation field.
- No abstention accuracy until v1 defines a confidence threshold; use reviewed known-hard-negative hits and assistant false leads instead.
