# Phase 00 Cross-Phase Terminology Review

- State: `superseded and reconciled` — 2026-08-17 product source-bank/int8 decision
- Scope: implementation plans 01 through 14 plus Revision 4 and the evaluation contract

## Terms that every phase must consume unchanged

- `ResolvedConfig`
- `IndexProfile`
- `CanonicalTextProfile`
- `EmbeddingSourceProfile`
- `VectorSpaceProfile`
- `VectorStorageProfile`
- `ServingVectorProfile`
- `CanonicalInputSHA256`
- `ServingVectorKey`
- `ConfigImpactPlan`
- product document source 1024 and serving dimensions 1024/512
- fixed storage codec `int8`, default target 1024
- serving authority `.cidx/db/index.db`
- product document source bank `.cidx/db/embeddings.db`, never opened by search
- separate evaluation metadata `.cidx/lab/evaluation.db`, with no vector blobs
- stable MCP tools `status|search|read_span|reindex`
- stage-separated evaluation, dual dense references, scoped promotion, and no HNSW/ANN metrics

## Review queries to run after the hash decision

- Find duplicate or conflicting config field names, source dimensions, serving-dimension sets, codec enums, or defaults.
- Find OpenAI/Azure/custom endpoint remnants.
- Find runtime imports or prose dependencies on the lab.
- Find query-f32 persistence, production f32/f16, or provider-quantized codec assumptions.
- Find alternate first-loss/promotion enums, weighted totals, or unscoped promotion language.
- Find Phase 11/13 body-packager ownership conflicts.
- Find phase status, prerequisite, and artifact-link inconsistencies.

The final review records exact commands and results. It does not mark Phase 00 complete while the profile/hash fixture remains blocked.

## 2026-08-15 review record

Checks executed from the repository root:

```text
rg provider/OpenAI/Azure/base_url terms across r3 and implementation docs
rg source/serving dimension literals and output_dimension variants
rg production/query f32/f16 persistence language
rg binary/int8/default codec contracts across owning phases
rg body-packager ownership across Phases 11–13
rg weighted-total and scoped-promotion language
verify every Phase 00–14 document has numbered sections 1 through 13
verify Markdown fence parity, local link targets, trailing whitespace, and conflict markers
git check-ignore -v .env .cidx/index.db
```

Results:

- Phase 00 status is synchronized across the phase document, implementation index, and status ledger.
- `.env` and `.cidx/` resolve to explicit ignore rules.
- No conflicting provider remains. OpenAI/custom-endpoint mentions are historical comparisons or explicit rejections, not active config paths.
- Source 1024 and serving dimensions `{256,512,1024}` are consistent. The only 2048 mention explicitly excludes it from v1.
- Production f32/f16 and query-f32 persistence are consistently forbidden; document f32 references belong to the isolated lab or transient API path.
- `binary|int8` is the closed cidx codec set and `binary` is consistently the default.
- Phase 11 is the sole owner of `internal/search/inline_body.go`; Phase 13 only adapts/serializes it.
- Weighted totals are consistently rejected and promotion scopes are explicit.
- All Phase 00–14 section sequences, inspected local Markdown links, fences, whitespace, and conflict-marker checks passed.

Expected historical non-conflicts retained:

- r3 references OpenAI only as an unsupported provider or Voyage benchmark comparator.
- Phase 01 may compare provider direct 512/256 output with local 1024-prefix reduction, but production continues to request source 1024.
- Phase 08 names provider 2048 only to state that v1 excludes it.

## 2026-08-17 supersession review record

The prior review results above are preserved as historical evidence. The owner
subsequently fixed the product to default 1024/int8 with explicit compact
512/int8, accepted a durable product document source bank, and removed all
Binary/256 executable paths. The current terminology audit therefore requires
`AllowedServingDimensions={1024,512}`, fixed `storage_codec=int8`, source
storage at `.cidx/db/embeddings.db`, vector-free evaluation metadata at
`.cidx/lab/evaluation.db`, and no source-bank/lab dependency in search or MCP.

## Revision 4 reconciliation record

The 2026-08-15 Revision 4 pass additionally checked the canonical design, implementation index, execution guide, evaluation contract, phase headers, config/profile catalogs, and current Go/schema identifiers. It confirmed the final terms `target_segment_bytes`, `serving_dimensions`, `embedding.request`, `embedding.retry`, and line-cap-free `read_span`; separated synchronous grouping from asynchronous Batch Inference; and recorded every remaining pre-R4 implementation identifier in the supersession index. The renamed vector-space and dependent storage fixtures were recomputed with two independent SHA-256 tools.

After RFC 8785 approval, the five fixture digests were independently reproduced and the structural/link/terminology checks were rerun. No new conflict was found.
