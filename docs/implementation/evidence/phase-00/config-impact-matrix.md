# Phase 00 Config-Change Impact Review

The change planner compares fully resolved desired profiles with applied database profiles. It selects the smallest correct action and never treats schema versions as semantic profile versions.

| Changed semantic value | Unchanged identities | Changed identities | Required action | Paid API possible |
| --- | --- | --- | --- | --- |
| Return/candidate k, RRF, default mode, paid-query guard | all semantic profiles and vector keys | serving policy only | restart/reload | no |
| Inline/read safety policy | all semantic profiles and vector keys | serving policy only | restart/reload | no |
| FTS weights only | stored source/chunk/vector identities | lexical ranking policy | restart/reload and new evaluation run | no |
| Languages, chunker, projection, segment, symbol, FTS schema/tokenizer | source profile and compatible paid raw | index/canonical segment membership as applicable | local reindex and atomic generation publish | no for local index |
| Canonical formatter implementation with identical produced bytes | canonical input hash and paid raw key | canonical-text profile fingerprint | rebuild/reconcile and verify actual hashes | no for identical bytes |
| Canonical input bytes changed | none for affected input | canonical input, raw/source and serving keys | local reindex then explicit paid document embedding or compatible raw lookup | yes |
| Model/provider/source-role/dtype/truncation/adapter semantic contract | index and canonical bytes | source, vector-space, storage, serving keys | reconciliation and explicit paid document embedding | yes |
| Serving dimension | index, canonical input, paid source raw | vector-space, storage, serving keys | local reconciliation then rematerialize compatible raw or embed | conditional |
| Reducer/normalizer/metric | index, canonical input, paid source raw | vector-space, storage, serving keys | local reconciliation then rematerialize compatible raw | no when raw exists |
| `binary` to `int8` or reverse | index, canonical input, source and vector-space | storage and serving keys | local reconciliation then rematerialize compatible raw or normal embed | conditional |
| Database schema version | semantic identity depends on migration contents, not version number alone | schema compatibility | offline/startup migration with rollback | no |

## Ordering invariants

1. Strictly decode, default, and validate the complete candidate config.
2. Compute semantic profiles with the central registry.
3. Compare desired and applied fingerprints without writing.
4. Report the impact plan through status.
5. Let `index/reindex` own local generation and active-key reconciliation.
6. Refuse paid embed/materialize/hybrid work until required reconciliation is complete.
7. Keep FTS available throughout vector mismatch or partial vector coverage.

Changing serving dimension or codec never causes the runtime to open the lab automatically. The explicit development materializer may use compatible raw only after current config and active segment keys agree.

## Pre-Revision-4 transition

- Removed config fields are not aliases. A legacy shape fails with a typed field-mapping error before database open.
- Existing SQLite schemas remain intact. New canonical profile JSON produces new fingerprints rather than mutating historical profile identity.
- The index-profile change requires a local reindex. An existing serving vector is rekeyed only after exact legacy/new semantic equivalence and blob integrity are proved; otherwise use compatible raw rematerialization or a separately approved paid embedding.
