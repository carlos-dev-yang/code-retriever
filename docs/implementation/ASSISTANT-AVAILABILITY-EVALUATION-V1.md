# Assistant Availability Evaluation V1

- Status: `frozen_for_execution`; Step 3 complete and no scored turn has run
- Phase: 14, search-to-evidence Step 3 followed by Step 4
- Series: new host-decided availability series; not Assistant A/B V7
- Corpora: existing approved chi v5.3.1 and react-hook-form v7.85.0 snapshots
- Provider boundary: FTS-only, no `VOYAGE_API_KEY`, no Voyage client, no paid request
- Product boundary: no retrieval, ranker, input shape, output payload, SQLite,
  or tool-count change; only model-visible descriptions become explicit

## 1. Question

When the same Codex assistant receives the same repository, prompt, ordinary
tools, model, reasoning level, isolation, and limits, does making the four cidx
MCP tools available help it reach a correct source-backed answer with a
smaller or more focused investigation journey?

The treatment is tool availability plus the accurate model-visible MCP
interface. A cidx call is never required or rewarded. A treatment turn that
does not use cidx remains valid and stays in the primary intent-to-treat
denominator.

## 2. Arms

- `baseline`: Codex CLI with its ordinary local repository tools and no MCP
  server.
- `cidx_fts`: the identical environment plus the four cidx MCP tools under the
  live neutral descriptions and strict input schemas.

The shared prompt does not name cidx, prescribe a first tool, impose a
cidx-specific stopping rule, or describe cidx as primary, secondary, or last
resort. The assistant chooses whether and when the exposed capability is
useful.

## 3. Model-visible tool interface

The live MCP `tools/list` response is the interface authority. The runner
captures the complete tool list and binds three separate SHA-256 identities:

1. complete tool definitions;
2. description text by tool name; and
3. input schemas by tool name.

The expected capability and I/O contract is:

| Tool | Input | Successful output | Important boundary |
| --- | --- | --- | --- |
| `search` | required nonempty `query`; optional `k` from 1 through 20, with frozen default 5 when omitted; optional `mode` of `fts` or `hybrid`; required nonnegative `max_inline_bytes` | one ranked `results` array; every item has `chunk_id`, `path`, `language`, `kind`, `qualified_symbol`, `start_line`, `end_line`, `indexed_sha256`, and `match_sources` | returns no source text; array order is rank; `max_inline_bytes` is retained wire input and cannot change candidates; hybrid may require a paid query embedding but is disabled in this run |
| `read_span` | repository-relative `path`, inclusive 1-based `start_line` and `end_line`, and exact `expected_sha256` from a locator | exactly `path`, `start_line`, `end_line`, `indexed_sha256`, and complete `body` when path, hash, range, and byte maximum are valid | no silent truncation; stale, absent, unsafe, or oversized input returns a typed error; configured per-call source maximum is 64 KiB |
| `status` | empty object | generation identity, desired/applied fingerprints, file/chunk/segment and vector-coverage counts, freshness/error counts, timestamps, and dirty/generation-change flags without source bodies | diagnostic only; no full file list, vectors, or source bank |
| `reindex` | optional `dry_run` boolean | dry-run returns `dry_run` plus planned file/chunk and embedding reuse/pending counts; apply returns scanned/updated/reused/deleted file counts, updated chunks, embedding counts, manifest hash, and optional activated generation | local and provider-free; it never embeds documents; scored turns must not mutate the frozen state |

The server still exposes exactly these four tools. Search produces compact
candidate locators; selected source comes only from `read_span`. Raw BM25,
dense, RRF, planner, and timing diagnostics are not model-visible evidence.
Preflight performs one provider-free `search`, reads its first locator with
`read_span`, and performs `reindex(dry_run=true)`. It rejects any field-set or
locator-identity mismatch and freezes a separate functional-output-contract
hash alongside the full definition, description, and input-schema hashes.

## 4. Provider-free enforcement

The runner must prove all of the following before execution:

- each corpus config defaults to `fts`;
- `allow_paid_query_embedding` is false;
- the isolated environment omits `VOYAGE_API_KEY`;
- no query client is injected;
- MCP discovery returns exactly four tools;
- the index is current and clean; and
- each corpus config and SQLite index byte hash matches the frozen evaluation
  state, including the complete search block, default `k=5`, and the 64 KiB
  `read_span` hard maximum; and
- requested and mechanically effective search modes are recorded in the
  passive trace.

The runner compares the live tool names, result representation, provider-
credential boundary, complete definition hash, description hash, input-schema
hash, and functional-output hash to the manifest before it creates a scored
run directory. Recording a changed contract is not sufficient; drift is a
preflight failure.

A hybrid attempt is a protocol deviation and cannot produce provider traffic.
This run does not authorize a paid dense or hybrid comparison.

## 5. Question-set contract

The composite question-set version contains 30 calibration tasks:

- 10 Go questions over chi;
- 10 TypeScript questions over react-hook-form; and
- 10 TSX questions over react-hook-form.

Each language slice contains exactly:

| Primary shape | Count |
| --- | ---: |
| exact identifier or path | 2 |
| semantic single-hop behavior | 2 |
| dependency or type multi-hop | 2 |
| contract, lifecycle, state, or error flow | 2 |
| ambiguous/disambiguation | 1 |
| verified no-answer or hard negative | 1 |

Question wording and expected claims were authored by direct inspection of the
pinned source. No cidx search result or prior assistant output was consulted.
After the expected answer was written, the frozen parser inventory supplied
only exact parent symbols, file hashes, byte ranges, and line ranges. It did
not supply rank, score, candidate sets, or assistant behavior.

Every case freezes:

- one primary shape and optional modifiers;
- exact question text and language slice;
- required evidence groups and valid alternatives;
- expected material claims;
- a reviewed hard negative for every verified no-answer case;
- source file hashes and parent byte ranges; and
- an independent case digest.

The composite `question_set_identity.sha256` is also executable, not a label.
The runner recomputes it from one UTF-8 JSON object containing the identity
`id` and `version`, taxonomy specification, ordered question-source
specifications, and ordered scheduled-question records (`sequence`, `task_id`,
`question_source_index`, `question_digest`, `corpus_id`, and `language`).
Object keys are sorted, arrays retain manifest order, JSON has no insignificant
whitespace, and non-ASCII text is emitted directly. Preflight also verifies the
taxonomy, truth-sidecar, and individual question-source file hashes, every
scheduled case digest, total case count, and language-slice counts.

Prior question versions and Assistant A/B V3-V6 artifacts remain unchanged.

## 6. Execution schedule and isolation

Run 30 pairs and 60 scored turns exactly once. The frozen schedule alternates
the first arm and therefore assigns 15 pairs baseline-first and 15 pairs
cidx-first. The second arm follows immediately for the same task.

Every turn receives:

- a fresh opaque copy of the same pinned source tree;
- a fresh model context;
- identical ordinary tools and permissions;
- no grading truth or evaluation files in its working directory; and
- for treatment only, a private copy of the same frozen cidx config and index.

No transcript, cache created by another turn, mutable index state, or cidx
conversation state is shared. Source and state digests are checked before and
after each turn. A failed, timed-out, no-use, or partial turn is preserved and
is not selectively replaced.

## 7. Shared prompt and answer contract

Both arms receive the same prompt. It asks the assistant to use whatever local
repository tools it judges useful, avoid network and mutation, stop after it
has adequate direct evidence, and return only the frozen JSON answer schema.
Every material answer claim must cite a repository-relative source range. A
claim not established by the repository must be omitted, qualified, or stated
as unresolved.

The answer schema is unchanged from the closed series. Tool-selection policy
is intentionally absent from the prompt; the live tool descriptions and input
schemas are sufficient for the model to decide whether to call cidx.

## 8. Passive observation

The passive session trace is outside the model and outside product state. It
records action order, cidx queries and locators, exact reads, verified source
ranges, overlap, ordinary repository inspections, and model-visible result
bytes. It contains no source body, grading truth, usefulness label, or claim
judgment, and it cannot reject, rewrite, or suppress any action.

Official Codex usage fields are retained separately as input, cached input,
output, and total model tokens. Wall time is observational. There is no
latency, hit-rate, adoption, or weighted-total release gate in this
calibration run.

## 9. Blind grading and reduction

After all 60 turns are immutable:

1. build arm-blind packets containing the question, answer, cited source
   excerpts, required groups, and hard negatives;
2. classify answer quality as complete, partial, incorrect, or ungradable;
3. identify every material claim as observed, derived, or unresolved and bind
   its evidence references;
4. keep unsupported and contradicted claims separate from required-group
   coverage; and
5. join grades and frozen truth to the passive journey only after both are
   fixed.

Report, without a weighted total:

- paired answer correctness and required-group coverage;
- unsupported and contradicted material claims;
- cidx adoption and no-use outcomes;
- candidate coverage, first useful rank, selection utilization, and false-lead
  proxy when search is used;
- read precision, evidence coverage, gross/unique/redundant source bytes, and
  citation utilization when source is read;
- ordinary and cidx repository exploration before and after sufficient
  evidence; and
- paired official token deltas for all 30 intent-to-treat pairs.

The cidx-user-only slice and per-shape slices are descriptive because tool use
is self-selected.

## 10. Decision boundary

This is calibration and product-direction evidence, not `core_retrieval`,
`release_candidate`, or promotion evidence. The result determines the first
measured constraint among:

- candidate miss or ranking;
- candidate found but insufficient selected source;
- required dependency or type evidence not acquired;
- duplicate or post-sufficient exploration;
- unsupported final claims; or
- no demonstrated marginal value over ordinary tools.

Do not add signatures, neighbor metadata, dependency hints, a fifth tool, an
active orchestration prompt, or paid hybrid search before that first loss is
measured and separately approved.

## 11. Step checkpoints

### Step 3 — freeze

- validate all 30 source spans and three verified-absence scans;
- review questions, claims, alternatives, and hard negatives;
- freeze the question files, truth sidecar, taxonomy, prompt, schedule, and
  manifest digests;
- run only provider-free preflight/schema probes; and
- commit the freeze evidence before scored execution.

### Step 4 — execute and grade

- record Step 4 entry in `STATUS.md`;
- execute the frozen 60 turns once;
- preserve every result;
- prepare and complete blind grade v2;
- reduce the journeys and write immutable result evidence; and
- commit the result separately from the Step 3 freeze.

## 12. Checks not implied by this plan

- no new test code;
- no new repository or corpus download;
- no corpus source mutation;
- no provider credential, embedding, or network request by cidx;
- no broad release/package matrix;
- no non-Codex host claim; and
- no push or release publication.
