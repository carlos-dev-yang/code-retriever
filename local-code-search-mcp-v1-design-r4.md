# Local Code Search MCP v1 — Final Target Contract (Revision 4)

- Status: final implementation target; Revision 4 reconciliation is in progress
- Date: 2026-08-15
- Binary name: `cidx`
- Previous canonical draft: [`local-code-search-mcp-v1-design-r3.md`](local-code-search-mcp-v1-design-r3.md)
- Detailed implementation plan: [`docs/implementation/README.md`](docs/implementation/README.md)
- Evaluation contract: [`docs/implementation/EVALUATION-CONTRACT.md`](docs/implementation/EVALUATION-CONTRACT.md)

Revision 4 is the normative v1 target. Earlier revisions remain design history. If an earlier revision, phase document, implementation, or completion record conflicts with this document, Revision 4 wins and the affected phase must be reconciled before implementation resumes.

## 1. Product boundary

`cidx` is a small, repository-local MCP search assistant for Go, TypeScript, and TSX source code. It complements file readers, `rg`, language servers, compilers, and tests; it does not replace them.

The v1 retrieval unit is a named function, method, or type. Local AST extraction and SQLite FTS5 indexing are free. Dense retrieval is optional and uses the official Voyage AI embeddings endpoint with `voyage-code-4`. Paid document or query embeddings are never hidden inside indexing or FTS search.

The stable MCP surface contains exactly four tools:

- `status`
- `search`
- `read_span`
- `reindex`

The default serving behavior is:

- search mode: `fts`
- production vector codec: cidx-owned `binary`
- alternative vector codec: cidx-owned `int8`
- one active vector profile per repository
- no HNSW or other ANN index

No numeric hit-rate, MRR, p50, p95, or maximum-latency target is a v1 release promise. These values are measured per corpus and language before later optimization decisions.

## 2. Runtime architecture

The MCP host starts `cidx serve` as a small long-lived Go stdio process for one explicit repository root. The process parses JSON-RPC/MCP messages and coordinates bounded work; it is not an HTTP daemon.

SQLite is embedded through the selected Go driver and stored under the repository's `.cidx` directory. Users do not install a separate SQLite service. Persistent AST chunks, FTS rows, segment metadata, active vectors, profiles, and generation metadata live in `.cidx/index.db`.

The Go process may hold bounded request buffers, decoded query vectors, and SQLite/OS page-cache state. It must not load the whole corpus or maintain a second authoritative in-memory index. Restarting the process must recover all production state from `index.db`.

Tree-sitter grammars for Go, TypeScript, and TSX are packaged with the executable. Language-specific chunkers share a common chunk/projection/segment contract because each grammar represents declarations, methods, overloads, fields, and type bodies differently. Native compiler ASTs are not a planned automatic fallback. A native AST adapter may replace a language adapter only after language-specific evaluation demonstrates a material Tree-sitter failure and the same common contract can be preserved.

## 3. Indexing and freshness

`cidx index` and MCP `reindex` are separate from `search`.

An index run:

1. validates the explicit repository root and configuration;
2. enumerates tracked plus untracked, non-ignored eligible source files;
3. reads each selected file once for its content hash and parse input;
4. parses changed files into function, method, and type chunks;
5. creates local FTS documents and embedding segments;
6. prepares changes outside the SQLite write transaction; and
7. publishes one atomic active generation.

The indexed source is the live working tree read during the run, not the Git HEAD snapshot. Git commit and dirty state are diagnostic metadata only. `search` never runs `index` implicitly. `index` and `reindex` never call Voyage.

Search ranking and returned indexed bodies come from one pinned SQLite generation. Live filesystem checks only annotate freshness after ranking. `read_span` reads the current file and requires an expected indexed hash; it fails on stale or missing files instead of returning a misleading range.

## 4. Source, chunk, and segment size contracts

These limits have different purposes and must not share one configuration field.

| Contract | Final v1 value | Meaning |
| --- | ---: | --- |
| Eligible source file default and absolute ceiling | 1 MiB | Parser/indexing safety boundary for one source file |
| Semantic chunk byte cap | none | A named function, method, or type remains one parent chunk |
| Segment target | 1,024 bytes | Packing target for embedding input units |
| Segment evaluation candidates | 768, 1,024, 1,536 bytes | Values evaluated before later tuning |
| MCP inline-body server default | 64 KiB | Aggregate source bytes returned inline by one `search` call |
| MCP inline-body executable ceiling | 1 MiB | Absolute serving safety ceiling, independent of the source-file ceiling |

The 1 MiB source-file value is an eligibility ceiling, not an expected allocation per file and not a performance guarantee. Generated, minified, dependency, vendored, and explicitly ignored files remain excluded independently of size.

There is no configurable `max_chunk_bytes` in the final target. A large semantic parent is stored and ranked as one chunk. Its embedding input may be divided into several segments, and its response body remains governed by the caller's inline budget.

`target_segment_bytes` is a byte-based packing target, not a hard split boundary. Segment construction follows AST statement/member boundaries and never cuts arbitrary UTF-8 bytes. One indivisible AST unit may exceed the target and remains one oversize segment with an explicit diagnostic.

The earlier 200–400 token description is a quality hypothesis for typical code, not a deterministic byte-to-token mapping. Local indexing must not depend on a provider tokenizer. Phase 12 records actual token observations for the selected corpora and compares the 768/1,024/1,536-byte candidates without treating any value as a universal token guarantee.

## 5. Embedding and vector profile

The only v1 provider/model pair is `voyage-official` / `voyage-code-4`.

Both document and query requests explicitly use:

- `output_dimension=1024`
- `output_dtype=float`
- `truncation=false`
- `input_type=document` for indexed segments
- `input_type=query` for search queries

The provider's 1024-dimensional float response is the source representation. Production applies the same local transform to documents and queries:

```text
1024-dimensional provider float
-> take the leading serving_dimensions components
-> L2 normalize
-> encode or prepare with the selected cidx codec
```

The final configuration name is `serving_dimensions`, with allowed values `256`, `512`, or `1024`. The CLI name is `--serving-dim`.

`serving_dimensions` is the vector length used by every document vector, query vector, blob validator, and scanner in the active repository profile. It is not a source-code line range, directory scope, or search boundary. Source scope is determined separately by enabled languages, Git/ignore rules, file eligibility, and the explicit repository root.

Production stores one active cidx-owned codec only:

- `binary` is the default;
- `int8` is the only v1 alternative;
- provider-side binary/int8 output is not treated as either cidx codec;
- production contains no persistent f32/f16 vector column.

The initial development/evaluation workflow may preserve document-role 1024-dimensional f32 in a physically separate lab database. This is a test asset used to rematerialize and compare serving dimensions/codecs without repeated document-embedding charges. Runtime `serve` and `search` never open the lab database, query f32 is never persisted, and the lab is not a permanent multi-profile serving system.

## 6. Synchronous request and retry policy

v1 does not use Voyage asynchronous Batch Inference. A local coding assistant must not trade a small embedding bill for a job that may complete hours or days later.

The regular synchronous embeddings endpoint still accepts multiple inputs in one HTTP request. This request grouping is retained because it reduces round trips without changing the operation into an asynchronous batch job.

Provider references: [Embeddings API](https://docs.voyageai.com/reference/embeddings-api), [quickstart request grouping example](https://docs.voyageai.com/docs/quickstart-tutorial), and [rate limits](https://docs.voyageai.com/docs/rate-limits). Public provider tables may lag newly released models, so cidx does not copy an undocumented `voyage-code-4` aggregate token cap from another model.

Final request-policy defaults:

| Setting | Value |
| --- | ---: |
| `embedding.request.max_inputs` | 128 |
| `embedding.request.max_total_input_bytes` | 256 KiB |
| `embedding.request.max_concurrency` | 4 |
| `embedding.request.timeout_seconds` | 30 |
| `embedding.retry.max_retries` | 3 |
| retry waits | 10 s, 20 s, 30 s |

`max_total_input_bytes` is a conservative local request-construction boundary. It is not a provider token limit and must not be named or reported as one. Requests also obey the provider's validated per-input/context constraints and fail closed when a capability is unknown.

The retry schedule is staged linear backoff, not mathematical exponential backoff. The operation makes one initial attempt and at most three retries after 10, 20, and 30 seconds. Only transient transport failures, timeouts, HTTP 408, 429, and 5xx responses are retryable. Authentication, model, dimension, invalid input, and deterministic validation errors fail immediately. A valid `Retry-After` longer than the configured wait wins, and cancellation interrupts both requests and waits.

## 7. Search response and `read_span`

`search.max_inline_bytes` remains a required per-call input. It limits only the aggregate raw UTF-8 source bytes placed in `results[].body`; it never changes candidate selection, scores, rank, or the up-to-`k` result identities.

A value of zero requests metadata and ranges without inline source. The initial 64 KiB server value is a serving safety choice—roughly sixty-four 1 KiB target segments—not a host token-budget guarantee. Evaluation records truncation/omission and follow-up reads before any later adjustment.

The server calculates:

```text
effective_max_inline_bytes = min(request.max_inline_bytes, 64 KiB)
```

The configured 64 KiB value may be lowered operationally but never raised above the separate 1 MiB executable ceiling. A request above the server value is explicitly reported as clamped. Metadata remains available when a body is omitted.

The final target has no `max_read_span_lines` and makes no unsupported 400-line claim. Line count is a caller selection, not a reliable payload-size measure. `read_span` validates:

- eligible repository-relative regular file;
- current hash equals `expected_sha256`;
- valid inclusive line range;
- UTF-8 source;
- whole-file size is within the 1 MiB source-file ceiling; and
- the requested response fits the server byte ceiling.

It returns the complete requested line range or a typed error such as `FILE_STALE`, `FILE_NOT_FOUND`, or `SPAN_TOO_LARGE`; it never silently truncates. A single source line larger than the response ceiling cannot be paginated by the v1 line-only API and must be read with another file tool.

## 8. Configuration authority

All modules consume one validated immutable `ResolvedConfig`; they do not independently reread JSON or duplicate semantic constants.

The final user-facing shape includes these concepts:

```json
{
  "index": {
    "max_source_file_bytes": 1048576,
    "target_segment_bytes": 1024
  },
  "embedding": {
    "model": "voyage-code-4",
    "serving_dimensions": "<one of 256, 512, 1024>",
    "storage_codec": "binary",
    "request": {
      "max_inputs": 128,
      "max_total_input_bytes": 262144,
      "max_concurrency": 4,
      "timeout_seconds": 30
    },
    "retry": {
      "max_retries": 3,
      "wait_seconds": [10, 20, 30]
    }
  },
  "search": {
    "default_mode": "fts"
  },
  "mcp": {
    "hard_max_inline_bytes": 65536
  }
}
```

The serving dimension is required rather than silently defaulted. `cidx init --serving-dim` or the evaluation workflow selects one supported value, and repository-specific evaluation may later change it through explicit reconciliation. `source_dimensions=1024`, supported serving dimensions, provider endpoint, algorithm/version IDs, schema versions, and executable absolute ceilings are code-owned model/protocol contracts rather than arbitrary user values.

Change impact remains explicit:

| Change | Required action |
| --- | --- |
| source eligibility, parser/chunker, segment target, FTS format | local reindex |
| canonical embedding input or model/source role | paid document re-embedding unless compatible raw exists |
| serving dimensions, reducer/normalizer, binary/int8 codec | local rematerialization when compatible lab raw exists; otherwise paid embedding |
| request grouping, retry, FTS defaults, RRF, return count, inline byte limits | restart/reload only; no reindex or re-embedding |
| database schema | explicit migration |

## 9. Evaluation and promotion

Go, TypeScript, TSX, and mixed repositories are reported separately because parser and source-shape failures are language dependent. The user selects and approves pinned open-source corpora; cidx records URL, commit, license evidence, file slices, hashes, and local bindings. It never silently chooses, downloads, updates, indexes, or embeds an external corpus.

Evaluation separates, rather than blends:

- operational correctness and failures;
- human-judged retrieval usefulness;
- FTS, dense, provider-union, parent-collapse, RRF, body-packaging, and assistant first-loss stages;
- exhaustive serving-dimension f32 versus binary/int8 representation fidelity;
- storage, memory, request count, latency, and cost observations.

There is no weighted total score. HNSW recall and ANN tuning are excluded because v1 scans all eligible stored vectors. Failures and timeouts stay in required denominators. Calibration selects candidate settings; frozen confirmation evidence supports promotion. A metric observed during development does not become a release threshold retroactively.

The initial matrix includes:

- segment targets: 768, 1,024, 1,536 bytes;
- serving dimensions: 256, 512, 1024;
- cidx codecs: binary and int8;
- FTS, dense, and RRF hybrid lanes;
- per-language and mixed-corpus slices;
- inline-body and follow-up `read_span` behavior.

## 10. Implementation stop point and resumption rule

This revision finalizes the target contract only. It does not claim that the current worktree implements the renamed fields, limits, retry waits, or removed line/chunk caps.

Before implementation resumes:

1. update the central config schema/defaults and semantic profile names;
2. reconcile every affected phase against this revision;
3. identify migrations or compatibility handling for any existing local database/config;
4. rerun only the phase-boundary validations invalidated by the contract change; and
5. keep documentation, code, and completion evidence in separate intentional commits.

No release or usefulness claim is valid until the selected corpora and paid evaluation actions are separately approved and the evaluation contract is executed.
