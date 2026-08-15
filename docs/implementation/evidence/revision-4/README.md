# Revision 4 Reconciliation and Evidence Supersession

- State: Phase 00, Phase 02, and Phase 05 reconciliation complete; later affected phases pending
- Date: 2026-08-15
- Canonical authority: [`local-code-search-mcp-v1-design-r4.md`](../../../../local-code-search-mcp-v1-design-r4.md)
- Historical checkpoint: `b3a6cb1`

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
| 02 | strict decoding, immutable injection, DB isolation, profile hierarchy | raw/resolved names, defaults, profile JSON, evaluation wire | strict R4 config, legacy typed error, fingerprints and schema validation |
| 05 | live-file and atomic-generation behavior | removed chunk cap and renamed segment target in index profile | [local reindex and compatible vector-rekey evidence](../phase-05/revision-4.md) complete |
| 08 | raw-bank isolation, response validation, cache-first capture | token-named sequential grouping and immediate retry | shared byte-bounded synchronous executor evidence |
| 10 | plan/apply approval and atomic result publication | request concurrency, staged waits, `Retry-After`, cancellation | deterministic bounded document orchestration evidence |
| 11 | vector scan, collapse, RRF, fallback, body packaging | query provider policy | shared request-policy/fallback evidence |
| 12 | stage metrics, arm orchestration, artifact isolation | `serving_dimensions` wire and R4 request/profile controls | corpus-independent adapter revalidation; official evidence remains externally gated |
| 13 | CLI/MCP adapters, four tools, stdio concurrency, status/reindex | `init`, config names/defaults, request policy, read-span line cap | final CLI/MCP evidence and phase-scoped acceptance commit |

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

The external Phase 07/12 corpus gate does not block corpus-independent Phase 13 completion. Official Phase 12 `core_retrieval` plus Phase 13/14 assistant and host evidence remains required before Phase 14 can establish `release_candidate` scope.
