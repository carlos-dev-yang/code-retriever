# Phase 00 Code-Owned Constant Catalog

These values are centralized registries or named protocol constants. They are not duplicated literals and are not user-editable configuration.

## Fixed provider and model contract

| Constant group | v1 value/contract | Planned owner |
| --- | --- | --- |
| Provider identity | `voyage-official` | `internal/config/model_registry.go` |
| Embedding endpoint | `https://api.voyageai.com/v1/embeddings` | Voyage adapter |
| Credential environment name | `VOYAGE_API_KEY` | Voyage adapter/bootstrap |
| Model | `voyage-code-4` | `ModelSpec` registry |
| Source dimensions | `1024` | `ModelSpec` registry |
| Allowed target dimensions | `256,512,1024` | `ModelSpec` registry |
| Document/query roles | `document` / `query` | source profile/adapter |
| Output dtype | provider float response validated as 1024 finite f32 values | source profile/adapter |
| Truncation | explicit `false` | source profile/adapter |
| Encoding format | omitted | source profile/adapter |

## Protocol and identity constants

| Constant group | Required contract | Decision owner |
| --- | --- | --- |
| Config schema version | named supported version and strict unknown-field behavior | Phase 02 |
| Production/lab schema versions | independent named versions and migrations | Phase 02 |
| MCP schema version and tool names | exactly `status`, `search`, `read_span`, `reindex` | Phase 13 |
| Hash domains | separate domains for input, index/canonical/source/vector-space/storage profiles, manifests, artifacts | Phase 00/02 |
| Canonical JSON algorithm | RFC 8785 JSON Canonicalization Scheme (JCS) | Phase 00 fixed; Phase 02 implements |
| Stable first-loss/failure enums | evaluation contract catalog | Phase 00/02 |
| Promotion scopes/status | `core_retrieval|release_candidate`; `PROMOTION_EVIDENCE_READY|NOT_PROMOTION_READY` | Phase 00/02 |
| Generation publish protocol | staging invisibility plus one atomic active-generation transition | Phase 01/02 |

## Algorithm registries

| Registry | Closed v1 surface | Exact implementation evidence |
| --- | --- | --- |
| Reducer | source prefix reduction from 1024 to an allowed target | Phase 01 |
| Normalizer | L2 normalization | Phase 01 |
| Metric | cosine target-space reference | Phase 01 |
| Storage codec | `binary|int8`, default `binary` | Phase 01 |
| Binary codec contract | bit mapping, packing order, padding, query preparation, scorer and version | Phase 01; not yet fixed |
| int8 codec contract | scale, rounding, clamp, norm, query preparation, scorer and version | Phase 01; not yet fixed |
| Canonical text formatter | exact byte framing/version | Phase 02 then language phases |
| Parser/chunker/symbol/FTS IDs | code-owned implementation identifiers | Phases 01–06 |
| RRF formula/tie behavior | versioned 1-based rank formula and deterministic tie policy | Phases 06/11; numeric `rrf_k` remains config |

Provider-native `output_dtype=binary|int8` is not a cidx codec and must never share a codec ID with a local encoder/scorer.

## Safety ceilings and error identifiers

The executable owns absolute ceilings for source bytes, chunk/segment bytes, result/candidate counts, inline bytes, read-span lines/bytes, batches, retries, and concurrency. Config may select only a value within those ceilings. Exact numbers are measured and fixed by their owning phase rather than guessed in Phase 00.

Stable error identifiers are code constants, not parsed message text. At minimum they cover config/schema/profile mismatch, reconciliation/materialization required, paid-query disabled, missing API key, query/document embedding failure, stale/not-found/oversized spans, invalid vector/blob, generation change, corpus approval/license/hash failure, raw coverage incomplete, and incomplete evaluation.
