# Relation Calibration Stage E/F Evidence — Revision 4

- Date: 2026-08-19
- Scope: calibration-only relation evidence review and fixed-cell evaluation
- Code boundary: `ba44fabac49d257323909ea118c66b9d8a053b9a`
- Product profile: 1,024-dimensional int8 serving vectors
- Provider operations in Stage E/F: zero
- Selection result: `NO_POLICY_SELECTED_EVALUATION_ONLY`
- Semantic family: `NOT_OPENED_NO_FINITE_CELL_MANIFEST`

## Decision

The 40-query go-git, Zustand, and Memos calibration unit is complete and is
now closed to tuning. It proves that the bounded relation frontier contains
additional source evidence that can complete some requirements missed by the
protected dense top five. It does not select a serving policy and does not
prove that an assistant will inspect the hint, call `read_span`, or improve its
final answer.

The review labels are evaluation evidence only. They are not written to a
project index and are not required when cidx is initialized in another
repository. Runtime graph facts remain compiler/AST-derived and LLM-free.

## Immutable inputs and review authority

The accepted Stage A artifacts for the three owner-selected repositories were
reproved before review preparation. The label-blind emission plan contains
1,000 query/cell rows over 40 queries. The source-complete blinded universe
contains 616 unique parent attachments and 1,115 relation attachments.

Two independent AI passes covered the whole universe:

| Pass | Parent grade 0/1/2 | Relation grade 0/1/2 | Parent HN | Relation HN | Raw SHA-256 |
| --- | ---: | ---: | ---: | ---: | --- |
| ChatGPT | 278 / 257 / 81 | 536 / 450 / 129 | 17 | 11 | `814e6cffe098539f0cf3e7dbed3139e23a866e013a0cfd03c29f277dccc4e8a0` |
| Grok | 325 / 221 / 70 | 671 / 383 / 61 | 9 | 33 | `1c258dd18fbc7d38c3655bea8cb62547765146902ef434f54babed92350594ce` |

The Grok pass required a mechanical relation-target ceiling normalization for
66 rows: 55 grades were clamped to their target parent, nine grade-2 group sets
were intersected with the target parent groups, and two rows were downgraded
because no direct group remained. The normalization record has SHA-256
`7021e5031680a9788d48284ce2d2d65715629ee07e08b440128f7a7cd034c47d`;
it changes no source judgment beyond the predeclared target ceiling contract.

The two passes left 102 source conflicts: 26 parent rows and 76 relation rows.
A third Terra source-only pass inspected the bound source without rank, score,
arm, cell, or policy information and resolved them as 72 grade-2 and 30
grade-1 rows, with no grade-0 or hard-negative row. Its artifact is
`adjudications-terra-source-v1.json`, SHA-256
`c68e423a54515f77d1718eee17add10687042c66dc486ce912366924ea09d2e8`.
Wrapper, test, and caller relations received grade 1 unless the exact target
and edge directly supplied the required group.

The owner adopted the whole reconciled digest with zero per-row override under
the solo-project `NO_INDEPENDENT_HUMAN_REVIEW` limitation. The adoption input
has SHA-256
`d2c785bbec349677e365b2deaa90f24156d58816cd16123da49f088017be10db`.

## Binding and deterministic publication

| Binding | Digest |
| --- | --- |
| prelabel emissions | `48a91595e4c300b428eeb8f4443b4a938309bcbd68a1268df552d2cc80f82ba9` |
| prepared universe | `c686fe8c73f411709049369cad1f9be64671eb2c5092fae474fd2eb914fa1be0` |
| reconciliation | `19cc7b081e45bbac9e68b2b7b6e6e0f21293f64b6eeffc97a1e5ebadeab8f722` |
| frozen labels | `002a30b08e137467896df63f2e5da8bf176c965f06c6a164aee7fd4db565a19b` |

The frozen result contains 342/202/72 parent labels and 687/318/110
relation labels at grades 0/1/2. It contains zero final hard negatives. That
zero is a zero denominator, not evidence that the relation path is safe.

Stage F was run twice from the clean code boundary. The output directories are
byte-identical and contain:

- 1,000 query/cell records: 360 closure and 640 hint records;
- 1,025 scope/cell aggregates;
- 3,534 delivery aggregates;
- exactly 40 queries and 25 predeclared cells;
- no winner, weighted score, ranking, or selected policy.

The accepted Stage F file hashes are:

| File | SHA-256 |
| --- | --- |
| `artifact-checksums.json` | `b07c0071d43a003545bc6e885306c421e70ef4c7aa4ddf7dde8304f0e72c8c73` |
| `cell-aggregates.jsonl` | `c9963ee6d65832779d4972afc8dcf3c0f733eb8aa34e70c7db91e40f5fe69cbb` |
| `delivery-aggregates.jsonl` | `3ad7e17e2fe653924df31cfb2f8d7447989dbec4f7444e6a5b2cb9c63504896f` |
| `per-query-cell.jsonl` | `240e2fe62ddf6ffc3eea3b356a0de2c102643d86a5f71df2a07585e188c0fe23` |
| `selection.json` | `7789a3cc30970b58f4501f9f2a687deea30a6a2fa96abe37810d110c8ec1dc12` |

The portable scored artifacts contain no absolute path, credentials, source
body, query/document vector bytes, or provider secret. The protected product
dense top-five identity and order remain unchanged.

## Measurements

The 40 queries contain 61 required groups. The protected dense top five
already completes 52 groups and 31 queries.

| Diagnostic cell | Emitted parents | Payload bytes | Complete groups | Complete queries | Parent grade 2 / support / noise | Delivery useful / support / noise |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| closure count 2, 2,048 body bytes | 65 | 19,873 | 52 -> 57 | 31 -> 36 | 6 / 27 / 32 | 18 / 49 / 57 |
| hint count 4, 4,096 disclosure bytes | 150 rows, 133 parents | 96,316 | 52 -> 58 | 31 -> 37 | 29 / 64 / 40 | 34 / 81 / 40 |

The closure cell improves five queries, all in the relation-challenge slice,
but also produces ten noise-only queries. The hint cell improves six queries:
five relation-challenge and one naturalistic query. It has no noise-only query,
but still exposes 40 noisy parent hints; absence of a noise-only query is not
absence of noise.

Increasing the hint count beyond four at the 4,096-byte cap exposes all 201
parents and 57 noisy parents while leaving completeness at 58 groups and 37
queries. Likewise, closure cells larger than count 2 / 2,048 bytes add noise
without adding completeness. These are diagnostic comparisons, not chosen
budgets.

Across all 25 cells, a per-query upper envelope can complete 38 of 40 queries.
It is not a combinable policy result because it chooses a different cell after
seeing each answer. Every cell still misses:

- `gg-g09-rename-change`
- `me-x02-memo-editor`

## What this proves

1. The accepted graph/frontier can expose source evidence omitted by dense
   top five without changing the dense result.
2. Small contract closure and bounded relation hints each recover required
   evidence on the frozen calibration unit.
3. The effect is concentrated in the relation-challenge cohort; the measured
   naturalistic gain is small.
4. The complete process is provenance-bound, provider-free after retrieval,
   deterministic, and separated from the production search/MCP path.
5. Runtime projects do not need these review labels. The labels establish a
   benchmark reference for this one design experiment.

## What this does not prove

1. No server-push or hint policy is selected. The continuous semantic family
   was not opened because no finite executable cell manifest was frozen.
2. Zero frozen hard negatives does not establish a safety rate.
3. No assistant behavior was observed: hint inspection, `read_span` use,
   correct or wrong edge following, final-answer quality, context efficiency,
   latency, and token use remain unmeasured.
4. This calibration cannot establish cross-repository generalization,
   confirmation, Phase 12 `core_retrieval`, product integration, or Phase 14
   `release_candidate` scope.
5. The 38/40 upper envelope is descriptive only and cannot be promoted as a
   policy result.

## Validation boundary

The final implementation boundary passed:

- `go test -count=1 ./...`;
- `go test -count=1 -race ./...`;
- `go vet ./...`;
- `go build ./...` and a clean `-trimpath` build;
- `go mod tidy -diff` and `go mod verify`;
- formatting and `git diff --check`;
- production dependency proof that search, MCP, store, and vector packages do
  not import `internal/relationdiag`;
- unchanged four-tool MCP contract;
- clean runtime capability checks for FTS5, WAL, Go, TypeScript, TSX, and the
  supported schema range.

The adoption-preparation CLI defect found during real execution was fixed at
`ba44fab`: `freeze --adjudication` now builds the owner-adoption template from
the exact adjudicated reconciliation instead of the unadjudicated baseline.
Independent Terra review returned `CLEAR` and the live owner-adoption gate was
reproved. No Voyage or other network/provider operation was performed.

## Stop and handoff

This 40-query calibration unit is permanently closed. Do not add cells,
rescore it, alter its labels, or derive thresholds from it.

The next design boundary must first freeze either one exact policy or an
explicit no-policy contract without reopening this calibration. A separate
assistant-use A/B must then compare unchanged dense search, bounded closure,
and body-free hints plus existing `read_span`. Formal claims require a distinct
unexposed confirmation unit after that choice is immutable.
