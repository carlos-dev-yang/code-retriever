# Revision 4 Reconciliation and Evidence Supersession

- State: the current int8-only corpus-independent implementation and local darwin/arm64 package/operational checkpoint are accepted through Phase 14; Phase 07's provider-free and Voyage measurement loop is complete, and fresh label work is active under `OWNER_ADOPTED_DUAL_AI_REVIEW` with permanent `NO_INDEPENDENT_HUMAN_REVIEW`; official Phase 12 and release-candidate evidence remain gated
- Date: 2026-08-17
- Canonical authority: [`local-code-search-mcp-v1-design-r4.md`](../../../../local-code-search-mcp-v1-design-r4.md)
- Historical checkpoint: `b3a6cb1`
- Final corpus-independent design review: [current int8/source-profile implementation review](int8-source-profile-final-review.md)

Revision 4 changes the public config/profile vocabulary and fixed operational request limits. Earlier phase evidence remains an accurate record of checks performed against the earlier contract, but it is not proof of Revision 4 compliance and is not rewritten as if those checks covered the new contract.

## Compatibility decision

- Final config version 1 strictly accepts only the Revision 4 fields. Removed pre-release fields are not aliases and cannot become a second configuration authority.
- A legacy pre-R4 config fails closed with a typed error that identifies the exact mapping to `target_segment_bytes`, `serving_dimensions`, `embedding.request`, `embedding.retry`, and the line-cap-free MCP policy. Migration is explicit; `init` never overwrites an existing config.
- Existing production and lab SQLite files are preserved. The relational schema does not change merely because profile JSON field names change.
- The new index profile fingerprint forces a local reindex because the chunk cap is removed and the segment field changes meaning/name.
- Existing serving vectors may be copied/rekeyed locally only when legacy profile JSON, serving dimension, reducer, normalizer, codec, and blob integrity prove semantic equivalence. Otherwise cidx keeps FTS available and requires compatible raw rematerialization or, when no compatible raw exists, a separately approved paid document embedding.
- Evaluation wire replaces `target_dimensions` with `serving_dimensions` strictly. No official Phase 12 artifact exists, so pre-R4 smoke artifacts remain historical and are not accepted by the final schema.

## Affected phase evidence

| Phase | Historical evidence that remains useful | Invalidated Revision 4 boundary | Required new evidence |
| --- | --- | --- | --- |
| 00 | RFC 8785 method, source/canonical hashes, ownership catalogs | field catalog, vector-space/storage hashes, safety/request constants | [corrected catalogs and independently reproduced hashes](../phase-00/README.md) complete |
| 02 | strict decoding, immutable injection, DB isolation, profile hierarchy | raw/resolved names, defaults, profile JSON, evaluation wire, and later machine-path-bound SQLite metadata | [strict R4 config/wire evidence](../phase-02/revision-4.md) and [project-local source/state reconciliation](../phase-02/project-local-layout-reconciliation.md) accepted |
| 05 | live-file and atomic-generation behavior | removed chunk cap and renamed segment target in index profile | [local reindex and compatible vector-rekey evidence](../phase-05/revision-4.md) complete |
| 07 | historical lexical smoke and draft labels | fixed-profile corpus measurement, source/state separation, immutable rank evidence, and enforceable solo-project label authority | [provider-free FTS decisions, clean 32-query Voyage search, and historical AI-advisory replay evidence](../phase-07/measured-retrieval-loop-r4.md) complete; fresh current-profile blind pools and owner-adopted dual-AI freeze are active |
| 08 | product source-bank isolation, vector-free evaluation metadata, response validation, cache-first capture | token-named sequential grouping and immediate retry | [current physical split](../phase-08/int8-source-bank-reconciliation.md) and historical [shared executor evidence](../phase-08/revision-4.md) complete |
| 09 | historical transform/publication and synthetic comparisons | current int8-only runtime, v5 serving cache, and 1024/512 materialization boundary | [current int8-only materialization evidence](../phase-09/int8-only-materialization-reconciliation.md) complete |
| 10 | plan/apply approval and atomic result publication | source-bank-first provider result durability, compatible local reuse, and provider-only request accounting | [current product evidence](../phase-10/source-bank-first-document-publication.md) plus [historical request-policy evidence](../phase-10/revision-4.md) complete |
| 11 | vector scan, collapse, RRF, fallback, body packaging | query provider policy plus current int8-only executable surface | [current int8-only query/search evidence](../phase-11/int8-only-query-search-reconciliation.md) and historical [shared request-policy/fallback evidence](../phase-11/revision-4.md) complete |
| 12 | stage metrics, arm orchestration, artifact isolation | current 1024/512 int8 matrix, `serving_dimensions`, retry-attempt usage, and R4 request/profile controls | [current int8-only evaluation evidence](../phase-12/int8-only-evaluation-reconciliation.md) and historical [corpus-independent accounting/migration evidence](../phase-12/revision-4.md) accepted; official evidence remains externally gated |
| 13 | CLI/MCP adapters, four tools, stdio concurrency, status/reindex | current 1024-default/512-optional int8 init/help and provider-free source reuse | [current int8-only CLI/MCP evidence](../phase-13/int8-only-cli-mcp-reconciliation.md) and historical [provider-free init evidence](../phase-13/revision-4.md) accepted |
| 14 | historical packaging/host plan and earlier archive | current default-1024/int8, provider-free compact-512, retired-profile rejection, build/runtime reporting, local archive/verifier, notices, and Codex project setup | [current int8 package checkpoint](../phase-14/int8-profile-package-reconciliation.md) accepted from clean provenance `5f4955e1499ee8896be5c825ef0fb9b3a52abb70`; [earlier local checkpoint](../phase-14/revision-4.md) remains historical and official release-candidate evidence remains blocked |

Phases 03/04 retain their AST-boundary and oversize-segment evidence, Phase 06 retains its free FTS evidence, and Phase 09 retains its transform/codec evidence. Their connections to the new `ResolvedConfig` and fingerprints are checked at the affected owning phase boundaries rather than reopening unchanged algorithms.

## Commit sequence

1. Enter Revision 4 reconciliation and record this supersession policy.
2. Complete Phase 00 Revision 4 catalogs and hash fixtures.
3. Reconcile Phase 02 config/profile/evaluation wire.
4. Reconcile Phase 05 index profile and local transition.
5. Reconcile Phase 08 shared synchronous request execution.
6. Reconcile Phase 10 document orchestration.
7. Reconcile Phase 11 query embedding policy.
8. Revalidate Phase 12 corpus-independent adapter while leaving official promotion blocked.
9. Complete and accept Phase 13 in its own commit.
10. Implement and accept the corpus-independent Phase 14 packaging/runtime/host surface at `30748c1`.
11. Accept the local darwin/arm64 package, offline verifier, and Codex project-configuration-read checkpoint from clean provenance `a5b2baef9a18e68d6c8b5d4fb62dc2e03727edb4`; leave official evaluation and promotion evidence blocked.
12. Reconcile project-local source/state ownership, migrate path-bound production/lab metadata without provider calls, and return to the unchanged Phase 07 query-approval gate.
13. Complete the chi/RHF provider-free FTS loop, run the provenance-bound 32-query Voyage embedding search, and preserve the original blind AI-advisory label sensitivity as historical evidence.
14. Replace the unavailable human-pass gate with `owner-adopted-dual-ai-v1`, require independent source-backed ChatGPT/Grok passes plus whole-digest owner adoption, and retain `NO_INDEPENDENT_HUMAN_REVIEW` in every downstream claim.
14. Reconcile the installed package to default 1024/int8, provider-free compact 512/int8, and negative-only Binary/256 from clean provenance `5f4955e1499ee8896be5c825ef0fb9b3a52abb70`.

The external Phase 07/12 corpus gate does not block corpus-independent Phase 13 or local Phase 14 implementation. Phase 07 needs no further Voyage call while only labels change; it requires fresh digest-bound dual-AI passes and owner adoption before calibration freeze. The local darwin/arm64 package checkpoint does not establish support for other targets or hosts, nor does it establish `release_candidate` scope. Official Phase 12 `core_retrieval` plus frozen Phase 13/14 assistant and host evidence remains required before Phase 14 can establish that scope.
