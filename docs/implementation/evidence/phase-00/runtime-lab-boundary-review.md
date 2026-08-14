# Phase 00 Production/Lab Boundary Review

## Physical and schema boundary

| Concern | Production | Development lab |
| --- | --- | --- |
| Default path | `.cidx/index.db` | `.cidx/lab/embeddings.db` |
| Persistent authority | active index/FTS/selected serving vector profile | initial evaluation document-f32 preservation only |
| Float vectors | forbidden | 1024-dimensional document-role f32 allowed |
| Query text/f32 | never persisted | never persisted |
| Codec vectors | active cidx `binary` or `int8` serving rows | search-invisible derived evaluation/materialization artifacts may exist |
| Schema/migration | production-only version and factory | independent lab-only version and factory |
| Runtime requirement | required | optional; deletion cannot break runtime |

## Dependency rules

- `serve`, `search`, `status`, `index`, normal `embed`, production `store`, and MCP packages do not import, open, attach, or migrate the lab.
- Development capture/materialize/evaluate paths may import the lab through a separate development bootstrap.
- Only the development materializer may open both stores. It performs conversion outside write transactions, rechecks repository/profile/active keys, and publishes through a short production transaction.
- Production interfaces and lab interfaces are distinct types and cannot be substituted.
- No foreign key, shared attached transaction, automatic fallback, startup synchronization, or restore promise crosses the databases.

## Lifecycle and cost boundary

1. Initial raw capture is an explicit paid development operation.
2. The paid document response is durably committed to the lab before optional local conversion, allowing safe resume.
3. Normal document embedding later follows current config, writes only the serving codec, and discards f32.
4. Hybrid query f32 is request-memory-only and is discarded after scan.
5. Removing the lab loses only the no-recharge evaluation/rematerialization aid; production remains valid.

## Repository and privacy boundary

- `.cidx/`, `.env`, local corpus bindings, raw vectors, evaluation reports, and transcripts are Git-ignored.
- Portable corpus manifests contain provenance and hashes, never absolute checkout paths.
- No credential, provider vector, raw source body, or live query is required in portable artifacts.
- FTS-only init/index/search/status/read-span/reindex remains functional without the lab, `VOYAGE_API_KEY`, or network.

## Static review planned after packages exist

Phase 02 must provide a dependency inspection proving that production bootstrap has no path to `internal/lab`. Phase 13 repeats the proof for the packaged `serve` graph. This Phase 00 review defines the expected result but does not claim code-level proof yet.
