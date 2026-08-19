# Phase 07 Relation Overlap and Search-Selection Diagnostic — Revision 4

- Date: 2026-08-19
- Artifact version: `relation-overlap-noise-diagnostic.v2`
- Scope: provider-free replay of the closed 40-query Stage E/F unit
- Code / labels: unchanged accepted boundary `ba44fab`; frozen digest
  `002a30b08e137467896df63f2e5da8bf176c965f06c6a164aee7fd4db565a19b`
- Provider operations: zero
- Assistant tasks: not run
- Local artifact:
  `.cidx/test/experiments/relation-overlap-noise-v1/diagnostic.json`
  SHA-256 `33a91723549c12486da93c07a638907537ff5065ad7375432109cbb19939656d`

## Decision

Assistant final-answer A/B is not the next graph test. The current test is
required-parent selection and residual isolated noise.

On this closed unit:

1. Dense top five already returns a file that contains the missing required
   parent for six of nine incomplete queries. The miss is a sibling symbol.
2. Three incomplete queries need a different file. Those required parents sit
   at dense ranks 14, 40, and 134.
3. Most parents Stage F counted as noise sit next to gold or primary (same
   file, same symbol, or one hop). Isolated irrelevance is the minority.
4. Organizing hint-cell extras by file leaves zero noise-only queries.

This is not a selected product policy and not confirmation. It does not
reopen the 32-case or 40-query questions.

## Method

Replay only accepted artifacts: `prepared.json`, `frozen-ba44`, Stage F
`per-query-cell.jsonl`, and the three Stage A `semantic-parent-scores.jsonl`
files. Attachment identities joined 616/616.

Missing groups are defined only from query topology:

```text
a required group is missing
when none of its source_parent_ids is in protected_top5_parent_ids
```

That definition matches the nine Stage F baseline-incomplete queries. Frozen
grade-2 `required_group_ids` on review attachments are used only to classify
neighborhood, not to declare a miss recovered.

Each review parent is placed in exactly one bucket:

| Bucket | Meaning |
| --- | --- |
| `direct` | frozen grade 2 |
| `support_near_gold` | grade 1 and same file / symbol / one-hop as gold |
| `support_other` | grade 1, not adjacent to gold |
| `overlap_near_gold` | grade 0 adjacent to gold |
| `overlap_near_primary` | grade 0 adjacent to protected top five only |
| `isolated_noise` | grade 0 and not adjacent to gold or primary |

A file is then a gold neighborhood, a support/primary neighborhood, or an
isolated-noise file.

### Correction

An earlier draft of this report said dense top 10 would recover five of nine
misses. That mixed frozen grade-2 covering parents into the denominator and
is withdrawn. Topology-only recovery is the table below.

## Search selection

Protected dense top five completes 31/40 queries and 52/61 groups. The nine
incomplete queries have nine missing groups.

| Missing group | Dense rank | Already in a top-five file? |
| --- | ---: | --- |
| `me-x02-memo-editor` / `module.MemoEditor` | 7 | yes |
| `gg-g06-commit-object` / `git.Repository.CommitObject` | 9 | yes |
| `me-x05-ai-provider-contract` / `module.LocalAIProvider` | 10 | yes |
| `zu-t08-create-bound-contract` / `module.UseBoundStore` | 12 | yes |
| `me-x06-navigation-item` / `module.NavLinkItem` | 20 | yes |
| `me-g06-schedule-matchers` / `scheduler.fieldMatcher` | 22 | yes |
| `gg-g07-diff-header-contract` / `diff.File` | 14 | no |
| `gg-g08-topological-node` / `commitgraph.CommitNode` | 40 | no |
| `gg-g09-rename-change` / `object.Change` | 134 | no |

Six of nine misses are sibling-symbol packaging failures. Three are
cross-file retrieval failures. `gg-g09` is far from the dense head and is a
limitation to report, not a graph-admission target for this unit.

Topology-only dense top-k recovery of the nine missing required parents:

| k | Recovered misses |
| ---: | ---: |
| 7 | 1/9 |
| 9 | 2/9 |
| 10 | 3/9 |
| 12 | 4/9 |
| 14 | 5/9 |
| 20 | 6/9 |
| 22 | 7/9 |
| 40 | 8/9 |
| 134 | 9/9 |

Raising return k is a descriptive option, not a chosen policy. Same-file
sibling packaging is the cheaper first mechanical test because it can reach
the six already-retrieved files without changing rank.

## Graph overlap versus isolated noise

Review universe (uncapped, 616 parents / 1,115 relation rows):

| Bucket | Parents |
| --- | ---: |
| direct | 72 |
| support near gold | 171 |
| support other | 31 |
| overlap near gold | 120 |
| overlap near primary | 98 |
| isolated noise | 124 |

Relation rows collapse to 447 distinct targets (2.49 rows/target). 307
targets have more than one relation row.

Representative Stage F cells, completeness unchanged from the accepted run:

| Cell | Queries | Groups | Unique parents | Isolated parents | Isolated share | Isolated-noise files | Gold-neighborhood files |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| closure 2 / 2,048 | 36/40 | 57/61 | 65 | 15 | 0.231 | 15 | 30 |
| hint 4 / 4,096 | 37/40 | 58/61 | 133 | 13 | 0.098 | 11 | 65 |

After file collapse, the hint cell has zero queries whose extra parents are
only isolated noise. Closure still has five such queries, because
contract-type body push can attach a file that is not next to gold or
primary:

- `me-g02-connect-routes`
- `me-g07-markdown-recursion`
- `me-x02-memo-editor`
- `me-x07-toolbar-selector`
- `zu-t06-subscribe-selector`

Stage F’s “32 noisy closure parents / 40 noisy hint parents” mixes two
things. The larger part is overlapping neighborhood that should be grouped.
The smaller part is isolated irrelevance that should be omitted or reported
as a limitation.

## Limitation signals that need no LLM

- required parent rank and whether it is inside the returned set;
- omitted-by-count and omitted-by-byte reasons;
- distinct-file count after collapse;
- isolated-hop count not admitted;
- `SIBLING_NOT_PACKAGED` versus `NEEDED_FILE_ABSENT`.

Those are search/graph packaging facts. They are not assistant-answer grades.

## What this authorizes

Evaluation-only follow-up on frozen labels, specified in
[`RELATION-PACKAGING-NEXT.md`](../../RELATION-PACKAGING-NEXT.md):

1. Mechanical same-file sibling packaging around protected top five.
2. Mechanical one-hop organization as file/symbol clusters, keeping isolated
   hops out of the default payload.
3. Separate reporting of selection rate and residual isolated-noise rate.

## What this does not authorize

- assistant A/B, a fifth MCP tool, or production search/MCP/store changes;
- rescoring or adding cells to the closed 32-case or 40-query units;
- choosing a semantic margin, a new return k, or the 38/40 envelope as policy;
- claiming confirmation, `core_retrieval`, or generalization.

## Checks run

- identity join of review attachments to Stage A parent scores: 616/616;
- Stage F completeness reproduced for both probe cells (36/40 and 37/40);
- Stage F `selection.json` SHA-256
  `7789a3cc30970b58f4501f9f2a687deea30a6a2fa96abe37810d110c8ec1dc12`;
- Stage F `per-query-cell.jsonl` SHA-256
  `240e2fe62ddf6ffc3eea3b356a0de2c102643d86a5f71df2a07585e188c0fe23`;
- no provider credentials read; no production package edited.

## Checks not run

- new executable packaging implementation;
- race/vet/build (no product code change);
- any assistant or Voyage operation.
