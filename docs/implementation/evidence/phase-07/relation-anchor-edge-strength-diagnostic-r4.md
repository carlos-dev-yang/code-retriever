# Phase 07 Anchor/Edge-Strength Diagnostic — Revision 4

- Phase: `07-lexical-evaluation`
- State: measured calibration diagnostic; no serving policy adopted
- Date: 2026-08-18
- Implementation commit: `dd814915902986c3fcb5a36220a35d5f8297b894`
- Dense input: frozen exhaustive `1024/int8` top 20
- Provider operations: zero
- Question, label, graph, production-search, and vector changes: none

## Purpose and boundary

The earlier v3 graph proved that RHF X08's exact
`FormState -> FormStateProps` type relation exists and carries the general
`SIGNATURE / TYPE_VALUE_PARAMETER / DECLARATION` metadata. It remained unclear
whether a directional edge-strength definition could choose that relation
without an X08-specific rule.

This experiment keeps the immutable v3 graph and the exposed 32-case
calibration replay unchanged. For each query it selects exactly two parents
from the frozen dense top 20 by normalized query-to-symbol token coverage,
enumerates both outgoing and incoming uniquely resolved one-hop edges, and
compares four predeclared strength definitions over the identical candidate
universe. It never embeds a relation, calls Voyage, changes the primary dense
top five, invokes FTS/RRF, or alters a product path.

The graph metadata separates three concepts:

1. `structural_tier` is a mechanical AST/compiler class, not a relevance
   grade: declaration contract, executable dependency, body reference, or
   declaration structure;
2. edge strength is computed only from complete-graph occurrence and distinct
   endpoint counts inside the same `(relation_kind, structural_tier)` stratum;
3. frozen relevance grades are loaded only after selection to evaluate the
   resulting attachments.

Production, test, example, and benchmark occurrence slices are recorded for
inspection but do not affect ordering. Ratios use checked integer
cross-products; no floating or weighted total score is created.

## Frozen policy arms

All arms first prefer query direction and the same structural-tier order.
They differ only in the strength fields that follow:

| Arm | Strength definition | Intended diagnostic |
| --- | --- | --- |
| raw frequency | exact edge occurrences descending | whether repetition alone identifies useful relations |
| source-normalized focus | edge/source-stratum occurrence ratio descending, then fewer source-stratum targets | whether a source concentrates on a small contract/dependency set |
| bidirectional specificity | source-normalized focus, then fewer target incoming sources and occurrences | whether a relation points to a specific rather than repository-wide target |
| incoming-popularity control | more incoming sources and occurrences, then edge frequency | negative control for hub/centrality preference |

The exact series plan was written before execution at
`.cidx/test/experiments/relation-anchor-edge-strength/relation-anchor-edge-strength-series-v1-dd81491.json`.
Its SHA-256 is
`a45ae418ca3693aa5698791fb16bbca534b0e2352d0f9e625ca86b2835bcdb06`.

## Implementation and validation

The evaluator records every deduplicated dense-top-20 anchor candidate, the
two selected anchors, directional facts, complete graph denominators, typed
ranking tuples, first differing components, admission order, attachments,
body packaging, and protected top-five body hashes. Comparator validation
proves that the serialized typed tuple reproduces the actual ranking order.

Independent Terra code review was `CLEAR` after requiring complete anchor
observability, mechanically checkable ranking tuples, separate edge/source/
target file-role slices, and minimum-rank retention for duplicate identities.

The main commit-boundary validation passed:

```text
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 -race ./internal/relationdiag ./internal/devlab
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 ./...
env -u VOYAGE_API_KEY GOPROXY=off go vet ./internal/relationdiag ./internal/devlab
env -u VOYAGE_API_KEY GOPROXY=off go build -o /tmp/cidx-relationdiag-anchor-edge-boundary ./cmd/cidx
node --check tools/relationdiag/typescript-resolver.mjs
jq -e . testdata/retrieval/relation-probes-chi-rhf-v1.json
go mod tidy -diff
gofmt -l internal/relationdiag internal/devlab/relations.go
git diff --check
```

The clean executable records commit
`dd814915902986c3fcb5a36220a35d5f8297b894`, `source_modified=false`, and
SHA-256
`14208d9042c3f55dac3fe395a9171269790fe2dad362eb7fef13d6dce0434510`.

## Aggregate measured result

Every arm raised formal complete related evidence from `30/32` to `32/32`,
preserved every primary top five, added no declared hard negative, and added no
`walkXFF` parent. That completion number is insufficient for policy adoption:
each arm always chooses a bundle, including many queries whose attachments
contain only frozen grade-0 or unreviewed evidence.

`Useful queries` below means at least one attached grade-1 or grade-2 parent;
`noise-only` means every attachment is grade 0 or unreviewed. Attachment
columns count parents rather than queries.

| Corpus | Arm | Complete before -> after | Useful / noise-only queries | Attached G2 / G1 / G0 / U | Added parents | Body bytes |
| --- | --- | --- | --- | --- | ---: | ---: |
| chi | raw frequency | `11/12 -> 12/12` | `6 / 6` | `3 / 6 / 11 / 4` | 20 | 10,964 |
| chi | source-normalized | `11/12 -> 12/12` | `6 / 6` | `3 / 6 / 11 / 4` | 20 | 10,964 |
| chi | specificity | `11/12 -> 12/12` | `6 / 6` | `3 / 6 / 11 / 4` | 20 | 11,090 |
| chi | popularity | `11/12 -> 12/12` | `6 / 6` | `3 / 6 / 12 / 3` | 19 | 10,911 |
| RHF | raw frequency | `19/20 -> 20/20` | `9 / 11` | `6 / 3 / 16 / 15` | 26 | 21,708 |
| RHF | source-normalized | `19/20 -> 20/20` | `8 / 12` | `8 / 1 / 18 / 13` | 28 | 22,789 |
| RHF | specificity | `19/20 -> 20/20` | `6 / 14` | `6 / 1 / 19 / 14` | 30 | 28,422 |
| RHF | popularity | `19/20 -> 20/20` | `10 / 10` | `7 / 3 / 12 / 18` | 26 | 18,670 |

The built-in safety gate is green because it checks declared hard negatives,
`walkXFF`, primary stability, graph/provenance binding, and execution
invariants. The stricter policy-advancement gate is not met: depending on the
arm, 6–14 queries per corpus receive no useful reviewed attachment at all.
This is the reason the formal `32/32` result is not accepted as a quality
result.

Edge-strength choice is material, especially in TypeScript/TSX. All four arms
selected the same relation for 7/12 chi queries but only 2/20 RHF queries.
Source-normalized and specificity agreed on 11/12 chi and 13/20 RHF queries;
the popularity control differed from source-normalized on 4/12 chi and 15/20
RHF queries.

## X08: exact relation versus incidental completion

For `rhf-x08-form-state-props`, the label-blind anchor step selects:

- required `FormStateProps`, dense rank 13, token coverage `3/4`;
- supporting `FormState`, dense rank 2, token coverage `2/3`.

All arms therefore make X08 complete after the related-evidence slot admits
the required rank-13 anchor. Only two arms actually select the intended
relation:

| Arm | Selected fact | Interpretation |
| --- | --- | --- |
| raw frequency | `FormStateProps -> UseFormStateProps`, `TYPE_BODY / TYPE_ALIAS`, byte 314 | completes only because the required anchor is also packaged; the selected relation is not the answer relation |
| source-normalized | `FormState -> FormStateProps`, `SIGNATURE / TYPE_VALUE_PARAMETER`, byte 688 | selects the exact contract; ratio `1/2`, two source-stratum targets, then the rank-13 endpoint wins the remaining tie |
| specificity | same exact contract | also separates the exact target's `2` incoming sources / `2` occurrences from the generic rival's `105` sources / `149` occurrences |
| popularity | `FormStateProps -> FieldValues`, `TYPE_BODY / TYPE_HERITAGE`, byte 260 | deliberately prefers the generic hub and adds an unreviewed endpoint |

The exact relation ID is
`4baa57b19ec63201ccb8af423b5fb56f4f8f8a154b0cebe4e7dffbfbbd36d43e`
at `src/formStateSubscribe.tsx:688..702`. The body for the required
`FormStateProps` parent is complete at 230 bytes in every arm.

This distinction is decisive: anchor-first localization fixes the formal X08
coverage, while source normalization and target specificity are the only
tested strengths that also identify why the two answer parents belong
together.

## G09 and safety

Every arm selects the exact `RealIP -> realIP` `CALLS` fact with relation ID
`aee07df0bb4d608d25e240e7cce47771b11683f4eb93292ff443eed390351c60`
at `middleware/realip.go:1111..1120`. Both required grade-2 parents are added,
their bodies are complete, and no arm adds `walkXFF` as related evidence. The
frozen dense top five still contains the pre-existing misleading `walkXFF`
result; this experiment neither changes nor duplicates it.

Body packaging omitted four unrelated chi bodies in every arm as
`BODY_TOO_LARGE`. RHF omitted one under raw/source/popularity and three under
specificity. Neither G09 nor X08 had a required-body omission.

## Reproducibility and immutable artifacts

Initial artifacts are under:

```text
.cidx/test/states/chi-1024-int8/evaluations/relation-diagnostic-chi-anchor-edge-{raw,source,specificity,popularity}-dd81491
.cidx/test/states/react-hook-form-1024-int8/evaluations/relation-diagnostic-rhf-anchor-edge-{raw,source,specificity,popularity}-dd81491
```

Each has a corresponding `-repeat-dd81491` directory. For all eight pairs,
the following files are byte-identical between the initial and repeat run:
`anchor-groups.jsonl`, `directional-one-hop-facts.jsonl`,
`edge-candidate-stats.jsonl`, `policy-ranking.jsonl`,
`primary-top5-proof.jsonl`, `stage-b-bundles.jsonl`,
`related-body-packages.jsonl`, and `aggregate-relation-metrics.json`.

The canonical filename/digest/byte-list SHA-256 for that deterministic subset
is:

| Corpus | Arm | Stable-subset SHA-256 |
| --- | --- | --- |
| chi | raw | `6d54bcfbf136d53974381d1ba1513a598abdbc29ad16e9554e6ebc5de830b25b` |
| chi | source | `bd5df20dd7d5e247a5ff7d56dc15fae690e97ea7f40baea7c33fb01415646e40` |
| chi | specificity | `e8d167e0b19a1508fe78e199d2921c4523758990008b2088b5647ed51a280d83` |
| chi | popularity | `0be63b33da9b58d67a95cb7d280bb95abf373806decbb17abc254ecea55cb2e9` |
| RHF | raw | `0baf7f7029a98549d8c402dfcf4fa4cf097831b3ba5456bb6bb785125f36922f` |
| RHF | source | `7ec671a751adb4e07e55f7d22992967f2cef78a0e70e00658ef07624fc760944` |
| RHF | specificity | `1c93460bc4493592f5e25f10a6af7ce354c768562837e5628895d5f6f4e91ff2` |
| RHF | popularity | `e1ea684e46ffbaa68b3b39fd4b4976530ff58d1cb34ad2e211473dd136ba9247` |

Full initial `artifact-checksums.json` file SHA-256 values are:

| Corpus | Raw | Source-normalized | Specificity | Popularity |
| --- | --- | --- | --- | --- |
| chi | `13e0cdbfa5fd9fb107c3d157ccb0f2e93830f5caf7b61ea763e642a0aebd1b5a` | `13ddd1a0a316a0a3988201cf50bd6011894ce7ca72ae560c99e7a278263671f1` | `0bed8c93b26278b33c1889a1c37bda88b9004a567183a29736a1cc4baeac802e` | `bbc8e7a86fb47a1a1ca36c11a8d30cf381ac23212a14572b60a0273a755fabe1` |
| RHF | `bb64636219fbc265b6cfd2b6c90f7ba289a6ad19089f6d885fd59436e355f88e` | `695826393a371e11e4870bac55021e3af8f056002292ba38170bee75428ad9ed` | `a1aaee2ef1386456bccb6889a8ce4304ce07e07e00caa960e647987159d14718` | `0f943de7f7e771a1232fc4c3b4164eee98563fbf6c1db16970523572b7a30be6` |

All 16 artifact entry sets were independently enumerated. Every declared
SHA-256 and byte count was recomputed and matched; no undeclared file was
present in a run directory. The artifacts contain no credential, absolute
checkout path, persisted query vector, provider operation, WAL/SHM file, or
source-body payload outside the declared bounded body package.

Independent Terra artifact review was `CLEAR`: it rechecked all 16 artifact
sets, initial/repeat stability, frozen primary top-five proofs, relation probes,
clean evaluator/graph/replay/dataset/policy bindings, zero-provider framing,
and G09/X08 plus hard-negative safety results. It agreed that none of the four
exposed calibration arms justifies production adoption.

## Decision and handoff

Do not adopt any of the four arms as a serving policy from this exposed
calibration set.

- Raw frequency is not reliable relation evidence; X08 is completed
  incidentally through its anchor.
- Incoming popularity is a negative control, not a candidate. It explicitly
  prefers generic hubs and produces the most unreviewed RHF attachments.
- Source-normalized focus produces the most RHF grade-2 attachments and
  selects the exact X08 relation, but still emits noise-only bundles for
  12/20 RHF queries.
- Bidirectional specificity supplies the clearest X08 anti-hub evidence, but
  as a global lexicographic preference it emits the most parents and bytes and
  has noise-only bundles for 14/20 RHF queries.

The next policy problem is admission/abstention, not another unconditional
edge ranking. A future versioned design should treat source-normalized focus
as base evidence and target specificity as supporting evidence, and should be
allowed to attach no bundle when structural coherence is insufficient. A
promising general signal is a verified edge connecting two independently
query-matched anchors, as occurs in both G09 and X08; it must be frozen on a
new calibration unit and validated on the separate unexposed confirmation set
rather than tuned further on these 32 exposed cases.

Phase 07 remains `in_progress`. No production schema, search, MCP, FTS/RRF,
question, label, graph, or provider change is authorized by this diagnostic.
