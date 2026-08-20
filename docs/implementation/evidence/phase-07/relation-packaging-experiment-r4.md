# Phase 07 Relation Packaging Experiment — Revision 4

- Date: 2026-08-19
- Owning phase: 07
- Authority: `OWNER_ADOPTED_DUAL_AI_REVIEW / NO_INDEPENDENT_HUMAN_REVIEW`
- Product baseline: exhaustive `1024/int8` dense top five, four MCP tools, no in-process LLM
- Provider operations: zero
- Assistant tasks: not run

## Decision

The mechanical packaging contract is frozen and executable. It is evaluation-only.

Owner index: [`OWNER-REVIEW-INDEX.md`](../../OWNER-REVIEW-INDEX.md).

Tracked contract:

```text
testdata/retrieval/relation-packaging-experiment-contract-v1.json
kind     cidx.relation_packaging.experiment_contract.v1
digest   cb726ace5f81d980260a8111520d5b2f00f9318f128682f3ddc6cc8ff7a54c28
```

Frozen decision cells:

| Arm | Payload | Decision cell |
| --- | --- | --- |
| A | protected dense top five | identity/order only |
| B | A plus same-file siblings | count 4 / 4096 body bytes |
| C | A plus organized one-hop clusters | 4 files / 4096 serialized bytes |
| D | B plus C | not authorized |

## Live closed-unit result

Provider-free replay of the frozen 40-query unit at the predeclared decision cells:

```text
cidx dev relations packaging
decision                 CONTINUE_SIBLING_PACKAGING
sibling_gate             true     5/6 named sibling misses recovered
one_hop_gate             false    nearby 2/2 recovered, but Arm C also completes gg-g09
primary_equal            true
baseline complete        27/40    topology source-parent coverage
arm B decision complete  32/40
arm C decision complete  36/40
gg-g09 in Arm A          incomplete
labeled isolated extras  0 at Arm B decision
provider operations      0
```

Local artifacts:

```text
.cidx/test/experiments/relation-packaging-v1/
```

Stage F / overlap / completion checksums matched the handoff before this run.

Named sibling recoveries at count 4 / 4096 bytes:

| Query | Arm A | Arm B | Note |
| --- | --- | --- | --- |
| `me-x02-memo-editor` | miss rank 7 | recovered | 4 extras, 1212 bytes |
| `me-x05-ai-provider-contract` | miss rank 10 | recovered | 4 extras, 644 bytes |
| `zu-t08-create-bound-contract` | miss rank 12 | recovered | 4 extras, 437 bytes |
| `me-x06-navigation-item` | miss rank 20 | recovered | 4 extras, 487 bytes |
| `me-g06-schedule-matchers` | miss rank 22 | recovered | 4 extras, 590 bytes |
| `gg-g06-commit-object` | miss rank 9 | **not recovered** | same file has ~141 extras; needed parent is outside the first 8 by symbol order |

Named cross-file at 4 files / 4096 cluster bytes:

| Query | Arm B | Arm C | Note |
| --- | --- | --- | --- |
| `gg-g07-diff-header-contract` | miss | recovered | rank 14, `NEEDED_FILE_ABSENT` on A/B |
| `gg-g08-topological-node` | miss | recovered | rank 40 |
| `gg-g09-rename-change` | miss | **recovered** | rank 134; every C grid cell emits `object.Change`. Forbidden claim. |

Arm C therefore fails its predeclared gate even though nearby recovery meets the numeric floor. Do not retune the keep proxy on this closed unit. Arm D is not authorized.

Topology completeness (27/40) is stricter than Stage F query completeness (31/40). Stage F counts a required group complete when a frozen grade-2 label on a protected top-five parent covers that group. This experiment counts a group complete only when one of its topology `source_parent_ids` is in the payload. Four additional topology misses exist (`gg-g03`, `gg-g11`, `gg-g12`, `me-t08`); they are not in the frozen sibling-miss ID list and did not vote for the sibling gate. Two of them (`gg-g03`, `me-t08`) become complete at sibling count 8.

An earlier fixture of the diagnosed 9-miss shape returned `CONTINUE_BOTH_FOR_ONE_COMBINED_TEST` and remains engine/gate proof only. The live unit supersedes it as measurement.

## What was implemented

- Canonical contract in `internal/relationdiag` plus the tracked JSON mirror.
- Label-free emitter: same-file sibling packaging with greedy skip-oversize byte admission; one-hop keep proxy `target file is a primary file OR source parent is a primary parent`.
- Coordinator scoring uses frozen topology groups for completeness and frozen labels only to classify extras as labeled isolated noise.
- Development command `cidx dev relations packaging`.
- Artifact layout from [RELATION-PACKAGING-NEXT.md](../../RELATION-PACKAGING-NEXT.md).

Production search, MCP, store, vector scoring, FTS/RRF, and the four-tool registry are unchanged and do not import `internal/relationdiag`.

## Checks run

```text
go test -count=1 ./internal/relationdiag ./internal/devlab
go test -count=1 -race ./internal/relationdiag ./internal/devlab
go vet ./internal/relationdiag ./internal/devlab
go build ./...
gofmt -l internal/relationdiag/packaging.go internal/relationdiag/packaging_test.go internal/devlab/relation_packaging.go
go list -deps ./internal/search ./internal/mcp ./internal/store ./internal/vector ./internal/app ./internal/cli
# no cidx/internal/relationdiag
git diff --check
env -u VOYAGE_API_KEY go run ./cmd/cidx dev relations packaging ...
# fails closed: missing .cidx
```

Focused packaging tests cover decision-cell `CONTINUE_BOTH`, count-cap miss of a deep sibling, primary-mismatch `INCONCLUSIVE`, label isolation, isolated-hop omission, and portable artifacts.

## Checks not run

- Full `./...` race.
- Assistant A/B.
- Any Voyage operation.
- Arm D (not authorized because one-hop failed).

## Exact next action

Keep sibling count 4 / 4096 bytes as the frozen evaluation packaging cell. Do not put it on the production MCP wire. Do not run Arm D. Do not retune one-hop keep-proxy or caps on this closed unit. Do not start assistant A/B. A later product design may attach same-file siblings under those caps; one-hop default push is rejected here because the label-free proxy always admits `gg-g09`.
