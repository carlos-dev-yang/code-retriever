# Phase 07 Relation Frontier-Cap Diagnostic — Revision 4

Date: 2026-08-18

Status: accepted calibration diagnostic; no product policy adopted

## Boundary

The owner authorized one provider-free test of a bounded relation-graph
frontier. The stored v3 graph, frozen dense rankings, questions, labels,
primary top five, body policy, and production search remained unchanged.

Both arms used the same label-blind frontier:

1. select two parents from the frozen dense top 20 with the existing anchor
   token-coverage order;
2. enumerate resolved outgoing and incoming one-hop occurrences;
3. remove self traversal and collapse occurrences to canonical typed/tier
   edges;
4. bucket by `anchor ordinal × direction × structural tier`;
5. retain the existing bidirectional-specificity top two in each bucket;
6. reserve a direct selected-anchor bridge inside an eligible two-edge bucket;
7. canonical-union without backfill, with a hard maximum of 32 edges.

`anchor-frontier-cap-only-v1` selected the first deterministic frontier edge.
`anchor-frontier-bridge-abstention-v1` used the identical frontier but emitted
only a direct edge between the two selected anchors; otherwise it recorded
`NO_DIRECT_ANCHOR_BRIDGE`. The two-anchor, top-two, tier order, 32-edge ceiling,
bridge reservation, and two-parent/1,024-byte body controls are provisional
calibration controls, not product thresholds.

The numeric design was reviewed with `kb-guide` before implementation. It
required bucket-first selection, canonical union without backfill, bridge
reservation inside the hard cap, cap-only versus bridge-only ablation, full
32-query denominators, and separate complexity and relevance reporting.

## Implementation and validation

The accepted implementation is clean commit
`770ff8e0c6c151791d5599bbdf68bd730dab7e99`. The clean executable reports
`source_modified=false` and has SHA-256
`808d5c2381932c1ad9c750968341f09b1889d5ec6d2f81ee3b14bd715320c505`.

The main commit-boundary checks passed:

```text
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 -race ./internal/relationdiag ./internal/devlab
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 ./...
env -u VOYAGE_API_KEY GOPROXY=off go vet ./internal/relationdiag ./internal/devlab
env -u VOYAGE_API_KEY GOPROXY=off go build -o /tmp/cidx-relationdiag-frontier-boundary ./cmd/cidx
go mod tidy -diff
node --check tools/relationdiag/typescript-resolver.mjs
jq -e . testdata/retrieval/relation-probes-chi-rhf-v1.json
gofmt -l internal/relationdiag internal/devlab/relations.go
git diff --check
```

All checks passed. Independent Terra review found one initial reservation defect:
the first version inserted bridges only under the global limit instead of
occupying an eligible bucket slot. The accepted code reserves before union,
keeps every bucket at two or fewer, records the exact displaced edge, and
abstains on assignment overflow. Re-review was `CLEAR`.

## Immutable execution

The ignored plan is
`.cidx/test/experiments/relation-frontier-cap/relation-frontier-cap-series-v1-770ff8e.json`
with SHA-256
`dcd1a5e81cb630844c979524a92da58f6d10968d384c2c0f2d5e79c299e710ab`.

The original graph directories contained SQLite `-wal`/`-shm` side files from
earlier read activity, which are deliberately absent from their immutable
three-entry checksum manifests. The run therefore read snapshot directories
containing only the four published files: `relations.db`, graph manifest,
resolution summary, and checksum manifest. Database, logical graph, resolver,
corpus, and semantic-parent hashes are unchanged. No graph was rebuilt or
mutated.

Eight fresh artifacts were written: chi and RHF, cap-only and bridge-only,
each with an initial and repeat run. Key initial hashes are:

| Corpus / arm | Artifact checksums | Frontier trace | Aggregate |
| --- | --- | --- | --- |
| chi cap | `9afbc0f5bd7793ab465ba8b97463fb73b853e232e24831a6be3023426f9bb785` | `90342f973955b53aa5bb6d1123c27e6afd26780ae4b756deefa2511a71367e60` | `9aafc4956e18fc8c1a044956a2254575c8250706c67a903cc7522663e85991a0` |
| chi bridge | `9528b9aa8de41a957a1d13c884f2a22c8e982ae8167c7d10e01845af4d783474` | `0f20e3e4e53a7249ac2b4dcd4a9f9182a17630ea02513ad6a9c1c6b588d086f9` | `e90b0439243f1896d078376a733bc5af64feb20d5c83f906e79c1090c55e2c58` |
| RHF cap | `6ab0e1cac1d2022c71a1822ffb1ddc151bce6f446a03bc3ae5551cd4fabd5979` | `a70090f35db0b5c6e3fb23e22ca6aba8229176e3e4cb2f3288c512a5f78dd21c` | `1f39ab220e75da58ba5efaea28a31da7fa1c6131f1b4893a058722d07b164493` |
| RHF bridge | `9586f7fd61264ad684a1ca043fd3f3f09f182e20312280e1534d4150f0d5dfbd` | `2a756859177c485dcb8aabcf6b0e2d1f3dabdd54bcf3f7a48f2e94f848939229` | `4fe179517edb3ac086145c1ba249f91b9bb2f3540ca24c131cb90aeb5cf09a12` |

Every repeat has the same frontier trace, per-query trace, aggregate, primary
top-five proof, bundle, and related-body hash as its initial run. Cap and
bridge arms have the same per-query final-frontier digest and primary proof.
All eight manifests bind the clean implementation, exact graph/replay/dataset,
calibration-only authority, and zero provider operations. Independent Terra
artifact review was `CLEAR`; no absolute machine path, hard-negative
attachment, or `walkXFF` attachment was found.

## Complexity result

The bucket cap, not the global 32-edge ceiling, performed the material pruning:

| Corpus | Raw occurrences | After self removal | Bucket-distinct after repeated-occurrence collapse | After bucket cap | After canonical dedupe | Per-query final min/max | Queries at 32 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| chi | 119 | 115 | 79 | 56 | 51 | 1 / 8 | 0 |
| RHF | 1,290 | 711 | 362 | 158 | 153 | 2 / 11 | 0 |

Chi removed 4 self facts, collapsed 36 repeated occurrences, truncated 23
bucket views, and removed 5 post-cap cross-bucket duplicates. RHF removed 579
self facts, collapsed 349 repeated occurrences, truncated 204 bucket views,
and removed 5 post-cap duplicates.

Direct selected-anchor bridges existed in 4 chi queries (5 canonical edges)
and 6 RHF queries (6 edges). Every bridge already survived its bucket top two;
no real query exercised forced reservation, displacement, or overflow. Those
paths are core-tested but remain unmeasured on these corpora. The global
32-edge ceiling was non-binding for all 32 queries, so this run does not prove
that 32 is a sufficient product threshold.

## Relevance result

| Corpus / arm | Complete before → after | Emitted / abstained | Useful / noise-only emitted queries | G2 / G1 / G0 / unreviewed attachments | Body bytes |
| --- | ---: | ---: | ---: | ---: | ---: |
| chi cap | `11/12 → 12/12` | `12 / 0` | `6 / 6` | `3 / 6 / 11 / 4` | 11,090 |
| chi bridge | `11/12 → 11/12` | `4 / 8` | `2 / 2` | `1 / 3 / 4 / 0` | 4,200 |
| RHF cap | `19/20 → 20/20` | `20 / 0` | `6 / 14` | `6 / 1 / 19 / 14` | 28,422 |
| RHF bridge | `19/20 → 20/20` | `6 / 14` | `4 / 2` | `3 / 2 / 7 / 0` | 6,268 |

The cap-only arm selected exactly the same relation ID for all 32 queries as
the earlier unconditional bidirectional-specificity arm. It therefore proves
that the frontier can be bounded without changing this calibration result, but
it does not improve relevance: 20/32 emitted queries still contain only
grade-0 or unreviewed evidence.

Bridge-only abstention reduced emission to 10/32 and noise-only emission to
4/10. It preserved RHF X08 and selected the exact
`FormState -> FormStateProps` `TYPE_VALUE_PARAMETER` contract. It failed chi
G09: the useful `RealIP -> realIP` call leads from a selected anchor to a
graph-only endpoint, not between the two selected dense anchors. G09 is thus
an admission/abstention loss, not a reachability or cap loss. Bridge-only is
also not sufficient precision evidence because four emitted bundles remain
noise-only.

Primary top-five identity, order, score, and body proofs are byte-identical in
all arms. Cap-only reaches `32/32` formal completeness; bridge-only reaches
`31/32`. No one was dropped from the denominator.

## Decision

- Retain `top_n_per_bucket=2` only as a provisional development complexity
  control. It worked on chi/RHF, but its relevance value and generality are not
  established.
- Retain the global 32-edge ceiling as a safety ceiling whose practical limit
  was not exercised here.
- Reject cap-only as a relevance policy; it reproduces the prior unconditional
  selector and its noise.
- Reject bridge-only as a general admission policy; it loses G09 and still
  emits noise. Preserve direct-anchor connectivity only as a coherence feature
  and ablation signal.
- Do not integrate either arm into production search, schema, MCP, FTS/RRF,
  vector scoring, or packaging defaults.

The bounded metric review permits at most one final exposed-data probe before
freezing a new unit: admit a direct selected-anchor bridge, or a unique
same-tier Pareto winner from a selected anchor to a graph-only endpoint using
source focus, no-greater source target diversity, and no-greater target
incoming-source fan-in, with at least one strict improvement. Multiple
nondominated candidates must abstain; no learned numeric margin or query/corpus
exception is allowed. Regardless of that result, stop tuning on these 32
queries. Product use still requires a separately frozen calibration unit and
the unexposed confirmation set.
