# Relation Selection and Packaging Plan

- Status: closed experiment; adopted evaluation-only sibling 4/4096; one-hop default push rejected; confirmation not authorized from this unit
- Date: 2026-08-19
- Owning phase: 07
- Product baseline: exhaustive `1024/int8` retrieval, four MCP tools, no
  in-process LLM
- Current authorization: evaluation-only mechanical packaging on the closed
  40-query unit
- Not authorized: assistant A/B, new corpus, Voyage, production
  search/MCP/store changes, a fifth MCP tool, or reopening either closed
  calibration

Handoff and accepted evidence:
[`RELATION-ASSISTANT-VALIDATION-HANDOFF.md`](RELATION-ASSISTANT-VALIDATION-HANDOFF.md),
[Stage E/F](evidence/phase-07/relation-calibration-stage-ef-r4.md),
[overlap/selection diagnostic](evidence/phase-07/relation-overlap-noise-diagnostic-r4.md).

## 1. Product role

cidx is a lightweight local search MCP. It returns ranked code parents from
already-indexed AST/FTS state and explicit embeddings. It does not write
answers. Hosts decide how to read the payload.

The current job is:

1. get required parents into the payload;
2. collapse overlapping graph neighborhood into grouped results;
3. keep isolated irrelevance out of the default payload;
4. report limits so a caller can see what was omitted.

## 2. Fixed conclusions

Do not reopen these:

- dense top five remains protected primary rank;
- compiler-resolved `CALLS`, `TYPE_REF`, and `MEMBER_OF` facts stay an
  evaluation sidecar until a packaging contract is frozen;
- graph-first, frequency, popularity, and Pareto admission are rejected;
- automatic closure-body push is not a default product path;
- relation text is not embedded;
- the 32-case and 40-query sets are closed to tuning;
- assistant final-answer A/B is deferred and is not a packaging gate.

## 3. Diagnosed first loss

On the closed 40-query unit, protected dense top five completes 31/40
queries and 52/61 groups.

| Loss class | Count | Meaning |
| --- | ---: | --- |
| sibling in an already-retrieved file | 6 | packaging |
| different file, dense rank 14 | 1 | nearby cross-file |
| different file, dense rank 40 | 1 | farther cross-file |
| different file, dense rank 134 | 1 | report as limit |

Hint-cell extras are 133 unique parents; 13 are isolated noise. File
collapse leaves zero hint queries that emit only isolated noise. Closure
body push still emits five isolated-noise queries and is not the first arm.

## 4. Next experiment

Answer exactly this:

> Holding dense top-five identity and order fixed, does mechanical same-file
> sibling packaging raise required-parent selection without raising isolated
> noise, and does organized one-hop clustering add the remaining nearby
> cross-file parents without dumping isolated hops?

It is not intended to answer assistant-answer quality, semantic-threshold
fitting, return-k retuning as a hidden policy, or promotion.

### 4.1 Arms

Freeze these arms before the first scored output. Do not add an arm after
inspection.

| Arm | Payload | Purpose |
| --- | --- | --- |
| A | protected dense top five only | baseline |
| B | A plus same-file siblings of those five parents | sibling packaging |
| C | A plus organized one-hop clusters, isolated hops omitted | neighborhood graph |
| D | B plus C | only if B and C each add completeness without isolated-noise regression |

Primary identity and order stay identical in every arm. Extra parents may
only be appended after the five, grouped by file then symbol.

### 4.2 Same-file siblings (arm B)

Eligible sibling:

- same repository-relative path as a protected top-five parent;
- current semantic parent, complete body;
- absent from the protected five;
- parent-deduplicated;
- independent of query ID and of G09/X02 special cases.

Caps, both required:

- maximum distinct extra siblings per request;
- maximum aggregate extra body bytes.

When a cap, missing body, or cycle check fails, record an omission reason.
There is no recursive type walk.

Calibration may compare only a finite predeclared grid. Suggested first
grid, not yet selected:

```text
count ∈ {2, 4, 8}
bytes ∈ {2048, 4096, 8192}
```

### 4.3 Organized one-hop (arm C)

Start from the accepted bounded frontier. Do not rebuild the graph or
change the top-two-per-bucket cap.

Emit file/symbol clusters, not raw relation rows:

- group by target path, then qualified symbol;
- keep relation kind, direction, tier, role, and occurrence count as
  cluster metadata;
- drop isolated hops from the default payload and count them;
- do not attach source bodies unless the target is also an arm-B sibling
  under that arm’s caps.

Caps, both required:

- maximum distinct cluster files per request;
- maximum serialized cluster bytes.

Isolated hop means a target parent that is not the same file, same symbol,
or one hop from gold/primary under the frozen diagnostic buckets. The
scored implementation cannot read labels; it must use a label-free proxy
frozen before execution:

```text
keep a one-hop target only when
its file is a protected-top-five file
or its source parent is a protected-top-five parent
```

That proxy is weaker than the labeled diagnostic and must be reported as
such. Do not sneak frozen grades into the emitter.

### 4.4 Metrics

Report raw paired outcomes. No weighted quality score.

- complete queries and complete required groups per arm;
- missing-group class: `SIBLING_NOT_PACKAGED` / `NEEDED_FILE_ABSENT`;
- extra unique parents, files, and serialized bytes;
- isolated-hop count omitted and, for diagnostic replay only, labeled
  isolated-noise remaining;
- paired A-loss-to-B/C-win and A-win-to-B/C-loss;
- omission reasons;
- primary top-five identity equality.

Completeness uses the frozen topology groups. Do not retune labels.

### 4.5 Limitation fields in the payload

Every packaged result records, without LLM prose:

- `primary_count`;
- `extra_sibling_count` / `extra_sibling_bytes` / sibling omissions;
- `cluster_file_count` / `cluster_bytes` / isolated hops omitted;
- dense rank of any still-missing required parent when the coordinator
  scores offline;
- `NEEDED_FILE_ABSENT` when the missing parent’s file is not in the
  primary files.

The MCP product path does not gain these fields until a later explicit
wire design. This experiment may emit them only in evaluation artifacts.

## 5. Implementation boundary

Confine code to:

- `internal/relationdiag` packaging derivation and artifact publication;
- `internal/devlab` development command;
- tracked evidence and this plan.

Do not modify production index, store, vector scoring, FTS/RRF, MCP
schemas, or the four-tool registry.

## 6. Decision gates

These are design-continuation gates, not promotion gates.

Return `CONTINUE_SIBLING_PACKAGING` only if all are true:

1. arm B converts at least four of the six same-file misses into complete
   required groups;
2. arm B causes zero completeness regression on the 31 baseline-complete
   queries;
3. labeled isolated-noise extras do not exceed arm A (which is zero extras)
   by more than the predeclared sibling cap, and isolated-noise *files*
   added by B are reported;
4. primary top five is unchanged.

Return `CONTINUE_ONE_HOP_CLUSTERS` only if all are true:

1. arm C converts at least one of the two nearby cross-file misses
   (`gg-g07` rank 14, `gg-g08` rank 40) without claiming `gg-g09`;
2. zero completeness regression on baseline-complete queries;
3. default payload contains zero isolated hops under the label-free proxy;
4. isolated hops omitted remain visible in limitation fields.

Return `STOP_DEFAULT_GRAPH_PUSH` when neither B nor C passes. Keep the
sidecar. Do not put graph extras into production search.

Return `INCONCLUSIVE` rather than tuning when bindings, primary equality, or
caps fail. An inconclusive run requires a newly frozen repeat contract.

`gg-g09` remaining incomplete is not a packaging failure.

## 7. After this experiment

Live closed-unit result: `CONTINUE_SIBLING_PACKAGING`. One-hop failed.

Adopted evaluation contract:
[`relation-sibling-packaging-adopted-v1.json`](../../testdata/retrieval/relation-sibling-packaging-adopted-v1.json)
(digest `d0b288b321cee2b60a794a0a38d7134395381491c9ede8b02d1af09ff2d65250`).

Sibling count 4 / 4096 bytes is evaluation-only and stays off the production
MCP wire until a separate product design is approved. Default one-hop push is
rejected. Arm D is not authorized. Confirmation still requires owner-selected
repositories and a new unexposed unit; this document does not select them.

Do not start assistant A/B as a consequence of these results.

Owner review record:
[`OWNER-REVIEW-INDEX.md`](OWNER-REVIEW-INDEX.md),
[`remaining-work-review-handoff-r4.md`](evidence/revision-4/remaining-work-review-handoff-r4.md).

## 8. Artifact layout

The frozen tracked contract is
[`testdata/retrieval/relation-packaging-experiment-contract-v1.json`](../../testdata/retrieval/relation-packaging-experiment-contract-v1.json)
(digest `cb726ace5f81d980260a8111520d5b2f00f9318f128682f3ddc6cc8ff7a54c28`).
Implementation evidence:
[packaging experiment](evidence/phase-07/relation-packaging-experiment-r4.md).

Keep generated state ignored:

```text
.cidx/test/experiments/relation-packaging-v1/
  experiment-contract.json
  arm-inputs/
  per-query-results.jsonl
  cell-aggregates.jsonl
  limitation-report.json
  decision.json
  report.md
  artifact-checksums.json
```

Portable reports must contain no credentials, absolute paths, source
bodies, or vector bytes.

## 9. Explicit prohibitions

The next implementation must not:

- run assistant A/B or add a fifth MCP tool;
- change production search ranking, RRF, FTS, dimension, or codec;
- reopen, rescore, or add cells to the 32-case or 40-query units;
- fit semantic margins or choose a cell from the 38/40 envelope;
- auto-push closure bodies as the default arm;
- hide omitted hops; they belong in limitation fields;
- call Voyage or read `.cidx/credentials.env`;
- claim confirmation, `core_retrieval`, `release_candidate`, or
  generalization.
