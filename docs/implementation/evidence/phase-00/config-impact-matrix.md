# Phase 00 Config-Change Impact Review

The change planner compares fully resolved desired profiles with applied database profiles. It selects the smallest correct action and never treats schema versions as semantic profile versions.

| Changed semantic value | Unchanged identities | Changed identities | Required action | Paid API possible |
| --- | --- | --- | --- | --- |
| Return/candidate k, RRF, default mode, paid-query guard | all semantic profiles and vector keys | serving policy only | restart/reload | no |
| Inline/read safety policy | all semantic profiles and vector keys | serving policy only | restart/reload | no |
| FTS weights only | stored source/chunk/vector identities | lexical ranking policy | restart/reload and new evaluation run | no |
| Languages, chunker, projection, segment, symbol, FTS schema/tokenizer | source profile and compatible product source rows | index/canonical segment membership as applicable | local reindex and atomic generation publish | no for local index |
| Canonical formatter implementation with identical produced bytes | canonical input hash and product source key | canonical-text profile fingerprint | rebuild/reconcile and verify actual hashes | no for identical bytes |
| Canonical input bytes changed | none for affected input | canonical input, source and serving keys | local reindex then explicit approved document embedding or compatible source lookup | yes |
| Model/provider/source-role/dtype/truncation/adapter semantic contract | index and canonical bytes | source, vector-space, storage, serving keys | reconciliation and explicit paid document embedding | yes |
| Serving dimension 1024 ↔ 512 | index, canonical input, product source rows | vector-space, storage, serving keys | local reconciliation then rematerialize compatible source rows | no when source coverage is complete |
| Reducer/normalizer/metric | index, canonical input, product source rows | vector-space, storage, serving keys | local reconciliation then rematerialize compatible source rows | no when sources exist |
| Fixed int8 implementation/profile version | index, canonical input, source and vector-space | storage and serving keys | local reconciliation then rematerialize compatible source rows | no when sources exist |
| Database schema version | semantic identity depends on migration contents, not version number alone | schema compatibility | offline/startup migration with rollback | no |

## Ordering invariants

1. Strictly decode, default, and validate the complete candidate config.
2. Compute semantic profiles with the central registry.
3. Compare desired and applied fingerprints without writing.
4. Report the impact plan through status.
5. Let `index/reindex` own local generation and active-key reconciliation.
6. Refuse paid embed/materialize/hybrid work until required reconciliation is complete.
7. Keep FTS available throughout vector mismatch or partial vector coverage.

Changing serving dimension never causes serving/search to open the source bank or lab automatically. Explicit materialization may use compatible product source rows only after current config and active segment keys agree.

## Pre-Revision-4 transition

- Removed config fields are not aliases. A legacy shape fails with a typed field-mapping error before database open.
- Historical SQLite data remains intact. New canonical profile JSON produces new fingerprints rather than mutating historical profile identity.
- The index-profile change requires a local reindex. An existing serving vector is rekeyed only after exact legacy/new semantic equivalence and blob integrity are proved; otherwise use compatible product-source rematerialization or a separately approved document embedding.
