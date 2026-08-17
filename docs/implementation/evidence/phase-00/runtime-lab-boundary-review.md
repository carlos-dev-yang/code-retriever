# Phase 00 Serving/Source/Lab Boundary Review

## Physical and schema boundary

| Concern | Serving production | Product source bank | Evaluation lab |
| --- | --- | --- | --- |
| Default path | `.cidx/db/index.db` | `.cidx/db/embeddings.db` | `.cidx/lab/evaluation.db` |
| Persistent authority | active index/FTS/one int8 profile | immutable document-role 1024-f32 source rows | vector-free run/evaluation metadata |
| Float vectors | forbidden | document source f32 only | forbidden |
| Query text/f32 | never persisted | never persisted | never persisted |
| Codec vectors | active cidx int8 serving rows | none | none |
| Schema/migration | serving-only factory | source-only factory | lab-only factory |
| Serving requirement | required | never opened by serve/search | never opened by serve/search |

## Dependency rules

- `serve`, `search`, `status`, `index`, serving `store`, and MCP packages do not import, open, attach, or migrate source-bank or lab files.
- Explicit document embedding/materialization may open serving plus source-bank stores; development evaluation may additionally open the lab through a separate bootstrap.
- Conversion occurs outside write transactions, rechecks repository/profile/active keys, and publishes through a short serving transaction.
- Serving, source-bank, and lab interfaces are distinct types and cannot be substituted.
- No foreign key, shared attached transaction, automatic fallback, startup synchronization, or restore promise crosses the databases.

## Lifecycle and cost boundary

1. Missing document source capture is an explicitly approved operation.
2. Each validated response is durably committed to the product source bank before local conversion, allowing safe resume.
3. Normal document embedding follows current config, reuses source hits, and publishes only fixed int8 to the serving DB.
4. Hybrid query f32 is request-memory-only and is discarded after scan.
5. Removing the lab loses only evaluation metadata. Removing the source bank does not break an already materialized target but loses provider-free future rematerialization.

## Repository and privacy boundary

- `.cidx/`, `.env`, local corpus bindings, source vectors, evaluation reports, and transcripts are Git-ignored.
- Portable corpus manifests contain provenance and hashes, never absolute checkout paths.
- No credential, provider vector, raw source body, or live query is required in portable artifacts.
- FTS-only init/index/search/status/read-span/reindex remains functional without the source bank, lab, `VOYAGE_API_KEY`, or network.

## Static review planned after packages exist

Phase 02 must provide dependency inspection proving that serving/search bootstrap has no path to `internal/sourcebank` or `internal/lab`. Phase 13 repeats the proof for the packaged `serve` graph. This Phase 00 review defines the expected result but does not claim code-level proof yet.
