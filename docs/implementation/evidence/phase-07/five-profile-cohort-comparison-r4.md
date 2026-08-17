# Five-profile cohort and answer comparison — Revision 4

- Date: 2026-08-17
- Authority: calibration diagnostic; not human-frozen truth or promotion evidence
- Corpora: pinned chi v5.3.1 and react-hook-form v7.85.0
- Questions: 32 draft behavior cases (Go 12, TypeScript 12, TSX 8)
- Profiles: `1024-f32`, `1024-binary`, `1024-int8`, `512-int8`, and `256-int8`
- Retrieval boundary: independent exhaustive dense ranks only; no FTS, RRF, or cross-codec fusion
- Provider activity for this consolidation: none

## 1. Scope and interpretation

This report consolidates the already-published immutable codec-ranking
artifacts. It performs no new Voyage operation: document vectors and top-20
rankings already existed, and the comparison only reads those stored results.
The original three dimension checkpoints each used one successful Voyage query
operation per case, but query-vector bytes were intentionally not persisted.

The five arms are never merged. Each question is evaluated independently
against one codec and serving dimension, so an improvement in one arm cannot
be supplied by another arm. The report deliberately omits a weighted overall
score: direct-answer retrieval, complete multi-parent requirements,
source-reviewed usefulness, f32 codec fidelity, and storage are separate
decision dimensions.

The three serving-dimension checkpoints used fresh provider responses. All 32
query-vector hashes differ between the 1,024, 512, and 256 checkpoints.
Cross-dimension results are therefore practical observed checkpoints, not a
causal paired dimension experiment. Within a checkpoint, int8 fidelity is
paired correctly against the same ephemeral source query vector and exhaustive
target-dimension f32 arm.

The direct truth is the current draft dataset. “Useful” below is a complete
source-backed advisory review of the pooled result parents, not the two-pass
human authority still required for label freeze.

## 2. Overall direct-answer result

| Profile | Hit@1 | Hit@5 | Hit@10 | Hit@20 | Complete@5 | Complete@20 | MRR@5 | MRR@20 | NDCG@5 | NDCG@20 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 1024-f32 | 16/32 | 30/32 | 31/32 | 32/32 | 30/32 | 32/32 | .6911 | .6975 | .7702 | .7888 |
| 1024-binary | 17/32 | 29/32 | 31/32 | 32/32 | 28/32 | 32/32 | .6875 | .6973 | .7444 | .7784 |
| 1024-int8 | 16/32 | 30/32 | 31/32 | 32/32 | 30/32 | 32/32 | .6911 | .6975 | .7713 | .7899 |
| **512-int8** | **17/32** | **31/32** | **31/32** | **32/32** | **31/32** | **32/32** | **.7094** | **.7122** | **.7801** | **.7954** |
| 256-int8 | 17/32 | 28/32 | 30/32 | 32/32 | 28/32 | 32/32 | .6875 | .7016 | .7456 | .7819 |

The profiles agree more than the aggregate differences first suggest:

- 15 of 32 direct answers are rank 1 in all five profiles;
- 27 are present in every profile's top 5;
- 30 are present in every profile's top 10; and
- all 32 are present in every profile's top 20.

`512-int8` has the strongest shallow direct result in this draft set. It does
not dominate every query or cohort, but it is the only profile with 31 direct
answers and 31 complete requirement sets in the first five results.

## 3. Result by language

| Profile | Language | Cases | Hit@1 | Hit@5 | Hit@10 | Hit@20 | Complete@5 | MRR@5 | NDCG@20 |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 1024-f32 | Go | 12 | .6667 | .9167 | 1.0000 | 1.0000 | .9167 | .7917 | .8237 |
| 1024-f32 | TypeScript | 12 | .4167 | 1.0000 | 1.0000 | 1.0000 | 1.0000 | .6694 | .7690 |
| 1024-f32 | TSX | 8 | .3750 | .8750 | .8750 | 1.0000 | .8750 | .5729 | .7663 |
| 1024-binary | Go | 12 | .6667 | 1.0000 | 1.0000 | 1.0000 | .9167 | .7847 | .8074 |
| 1024-binary | TypeScript | 12 | .5000 | .9167 | 1.0000 | 1.0000 | .9167 | .6944 | .7902 |
| 1024-binary | TSX | 8 | .3750 | .7500 | .8750 | 1.0000 | .7500 | .5313 | .7171 |
| 1024-int8 | Go | 12 | .6667 | .9167 | 1.0000 | 1.0000 | .9167 | .7917 | .8266 |
| 1024-int8 | TypeScript | 12 | .4167 | 1.0000 | 1.0000 | 1.0000 | 1.0000 | .6694 | .7690 |
| 1024-int8 | TSX | 8 | .3750 | .8750 | .8750 | 1.0000 | .8750 | .5729 | .7663 |
| **512-int8** | **Go** | **12** | **.7500** | **1.0000** | **1.0000** | **1.0000** | **1.0000** | **.8542** | **.8607** |
| **512-int8** | **TypeScript** | **12** | **.4167** | **1.0000** | **1.0000** | **1.0000** | **1.0000** | **.6486** | **.7438** |
| **512-int8** | **TSX** | **8** | **.3750** | **.8750** | **.8750** | **1.0000** | **.8750** | **.5833** | **.7751** |
| 256-int8 | Go | 12 | .5833 | .9167 | 1.0000 | 1.0000 | .9167 | .7222 | .7787 |
| 256-int8 | TypeScript | 12 | .5000 | .8333 | .9167 | 1.0000 | .8333 | .6667 | .7682 |
| 256-int8 | TSX | 8 | .5000 | .8750 | .8750 | 1.0000 | .8750 | .6667 | .8072 |

The language result explains why one aggregate is insufficient. Binary is
strong on Go direct Hit@5, but it is weakest on the TSX slice. At 256, TSX
Hit@1 improves while TypeScript shallow completeness falls. `512-int8` keeps
all Go and TypeScript direct answers in the first five while retaining the
same persistent TSX limitation as 1,024 f32/int8.

## 4. Result by cohort

### 4.1 Signal cohort

| Profile | Signal | Cases | Hit@1 | Hit@5 | Hit@10 | Complete@5 | MRR@5 | NDCG@20 |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 1024-f32 | mixed | 8 | .3750 | .7500 | .8750 | .7500 | .5625 | .7138 |
| 1024-f32 | semantic | 24 | .5417 | 1.0000 | 1.0000 | 1.0000 | .7340 | .8139 |
| 1024-binary | mixed | 8 | .5000 | .8750 | .8750 | .8750 | .6875 | .7940 |
| 1024-binary | semantic | 24 | .5417 | .9167 | 1.0000 | .8750 | .6875 | .7732 |
| 1024-int8 | mixed | 8 | .3750 | .7500 | .8750 | .7500 | .5625 | .7138 |
| 1024-int8 | semantic | 24 | .5417 | 1.0000 | 1.0000 | 1.0000 | .7340 | .8153 |
| **512-int8** | **mixed** | **8** | **.5000** | **.8750** | **.8750** | **.8750** | **.6667** | **.7698** |
| **512-int8** | **semantic** | **24** | **.5417** | **1.0000** | **1.0000** | **1.0000** | **.7236** | **.8040** |
| 256-int8 | mixed | 8 | .5000 | .7500 | .8750 | .7500 | .6250 | .7520 |
| 256-int8 | semantic | 24 | .5417 | .9167 | .9583 | .9167 | .7083 | .7918 |

### 4.2 Task cohort

| Profile | Task | Cases | Hit@1 | Hit@5 | Hit@10 | Complete@5 | MRR@5 | NDCG@20 |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 1024-f32 | delegated/cross-parent | 12 | .5000 | .9167 | 1.0000 | .9167 | .6486 | .7850 |
| 1024-binary | delegated/cross-parent | 12 | .5000 | .8333 | 1.0000 | .7500 | .6458 | .7691 |
| 1024-int8 | delegated/cross-parent | 12 | .5000 | .9167 | 1.0000 | .9167 | .6486 | .7878 |
| **512-int8** | **delegated/cross-parent** | **12** | **.5833** | **1.0000** | **1.0000** | **1.0000** | **.7389** | **.8405** |
| 256-int8 | delegated/cross-parent | 12 | .5833 | .8333 | 1.0000 | .8333 | .6944 | .8088 |
| 1024-f32 | interface/type/API | 3 | .0000 | .6667 | .6667 | .6667 | .3333 | .5675 |
| 1024-binary | interface/type/API | 3 | .0000 | .6667 | .6667 | .6667 | .3333 | .5440 |
| 1024-int8 | interface/type/API | 3 | .0000 | .6667 | .6667 | .6667 | .3333 | .5675 |
| 512-int8 | interface/type/API | 3 | .0000 | .6667 | .6667 | .6667 | .2778 | .5128 |
| 256-int8 | interface/type/API | 3 | .0000 | .6667 | .6667 | .6667 | .3333 | .5543 |
| 1024-f32 | lifecycle/config | 8 | .6250 | 1.0000 | 1.0000 | 1.0000 | .7917 | .8175 |
| **1024-binary** | **lifecycle/config** | **8** | **.7500** | **1.0000** | **1.0000** | **1.0000** | **.8542** | **.8605** |
| 1024-int8 | lifecycle/config | 8 | .6250 | 1.0000 | 1.0000 | 1.0000 | .7917 | .8175 |
| 512-int8 | lifecycle/config | 8 | .7500 | 1.0000 | 1.0000 | 1.0000 | .8438 | .8441 |
| 256-int8 | lifecycle/config | 8 | .6250 | .8750 | .8750 | .8750 | .7500 | .7888 |
| 1024-f32 | single-parent | 9 | .5556 | 1.0000 | 1.0000 | 1.0000 | .7778 | .8422 |
| 1024-binary | single-parent | 9 | .5556 | 1.0000 | 1.0000 | 1.0000 | .7130 | .7959 |
| 1024-int8 | single-parent | 9 | .5556 | 1.0000 | 1.0000 | 1.0000 | .7778 | .8422 |
| 512-int8 | single-parent | 9 | .4444 | 1.0000 | 1.0000 | 1.0000 | .6944 | .7863 |
| 256-int8 | single-parent | 9 | .5556 | 1.0000 | 1.0000 | 1.0000 | .7407 | .8156 |

All profiles reach every direct answer by top 20, including the thin
`interface/type/API` cohort. Its zero Hit@1 and only 2/3 Hit@10 across every
profile show a query/representation boundary that a codec choice alone does
not solve. The direct props contract in X08 is consistently outranked by its
hook and adjacent types.

## 5. Source-reviewed answer usefulness

The following asks a different question from direct Hit@K: does the returned
code actually help answer the behavior question, even when it is a caller,
consumer, helper, or adjacent contract rather than the declared direct parent?

| Profile | Useful top-1 | Useful Hit@5 | Useful results in top 5 | Useful precision@5 |
| --- | ---: | ---: | ---: | ---: |
| 1024-f32 | 26/32 | 32/32 | 81/160 | .5063 |
| 1024-binary | 25/32 | 32/32 | 78/160 | .4875 |
| 1024-int8 | 26/32 | 32/32 | 81/160 | .5063 |
| **512-int8** | **27/32** | **32/32** | **81/160** | **.5063** |
| 256-int8 | 27/32 | 31/32 | 76/160 | .4750 |

This prevents a misleading conclusion from direct rank alone. Binary's G09
direct answer improves from rank 8 to rank 2, but its overall useful top-5
neighborhood is weaker and it loses one of G07's two mandatory answer groups.
At 256, T12 has no source-reviewed useful result in the first five; the direct
`createSubject` implementation falls to rank 14. `512-int8` is the only compact
arm that combines the best useful top-1 count with useful top-5 coverage for
all 32 questions.

## 6. Per-question direct answer placement

Ranks are ordered `1024-f32 / 1024-binary / 1024-int8 / 512-int8 / 256-int8`.
“Top-1 interpretation” records whether a repeated non-direct result is still
useful support rather than treating every non-exact first result as equivalent.

### 6.1 chi / Go

| Case | Direct implementation | Direct rank by profile | Top-1 interpretation |
| --- | --- | --- | --- |
| G01 | `chi.Mux.ServeHTTP` | 1 / 4 / 1 / 4 / 3 | Direct in two arms; remaining first results are route/context support. |
| G02 | `chi.Mux.routeHTTP` | 2 / 1 / 2 / 1 / 2 | `chi.node.findRoute` is first in three arms and is useful route-selection support. |
| G03 | `chi.node.FindRoute` | 2 / 3 / 2 / 2 / 2 | Internal `chi.node.findRoute` is first in all five and implements the underlying lookup. |
| G04 | `chi.Mux.Mount` | 1 / 1 / 1 / 1 / 1 | Exact direct answer in all five. |
| G05 | `chi.chain` | 2 / 3 / 2 / 2 / 3 | `chi.Mux.With` is first in all five and usefully consumes the chain. |
| G06 | `middleware.BasicAuth` | 1 / 1 / 1 / 1 / 1 | Exact direct answer in all five. |
| G07 | `Compressor.selectEncoder` + `compressResponseWriter.isCompressible` | 1 / 1 / 1 / 1 / 1 | A direct parent is first in all five, but binary returns only one of the two required groups by top 5. |
| G08 | `middleware.Recoverer` | 1 / 1 / 1 / 1 / 1 | Exact direct answer in all five. |
| G09 | `middleware.RealIP` | 8 / 2 / 8 / 1 / 7 | Binary/512 rescue the deprecated implementation; newer client-IP code dominates other arms. |
| G10 | `middleware.ThrottleWithOpts` | 1 / 1 / 1 / 1 / 1 | Exact direct answer in all five. |
| G11 | `middleware.Timeout` | 1 / 1 / 1 / 1 / 1 | Exact direct answer in all five. |
| G12 | `middleware.HeaderRouter.Handler` | 1 / 1 / 1 / 1 / 1 | Exact direct answer in all five. |

G09 contains the current explicit hard negative, `walkXFF`. It enters top 5
for f32, both larger int8 arms, and 256-int8. Binary keeps it outside top 10
but it returns by top 20. This is a real intent-separation signal, not a reason
to merge binary and int8 ranks.

### 6.2 react-hook-form / TypeScript and TSX

| Case | Direct implementation | Direct rank by profile | Top-1 interpretation |
| --- | --- | --- | --- |
| T01 | `useForm` | 5 / 9 / 5 / 5 / 6 | `createFormControl` is first in all five and is useful lifecycle support; binary and 256 miss the direct top 5. |
| T02 | `useController` | 2 / 2 / 2 / 2 / 2 | `createFormControl` is first in all five and is useful coordinator support. |
| T03 | `useFieldArray` | 1 / 1 / 1 / 1 / 1 | Exact direct answer in all five. |
| T04 | `getDirtyFields` | 2 / 2 / 2 / 2 / 2 | `markFieldsDirty` is first in all five and is direct helper support. |
| T05 | `deepEqual` | 1 / 1 / 1 / 1 / 1 | Exact direct answer in all five. |
| T06 | `cloneObject` | 1 / 1 / 1 / 1 / 1 | Exact direct answer in all five. |
| T07 | `getFieldValue` | 2 / 1 / 2 / 2 / 1 | Direct first in binary/256; `createFormControl` is useful support in the others. |
| T08 | `createFormControl` | 1 / 1 / 1 / 1 / 1 | Exact direct answer in all five. |
| T09 | `Control` | 2 / 2 / 2 / 3 / 2 | `createFormControl` is first in all five and is the main contract consumer. |
| T10 | `PathImpl` + `PathInternal` | 2 / 2 / 2 / 2 / 2 | `ArrayPathImpl` is first in all five and is adjacent recursive type support. |
| T11 | `validateField` | 1 / 1 / 1 / 1 / 1 | Exact direct answer in all five. |
| T12 | `createSubject` | 3 / 3 / 3 / 4 / 14 | `createFormControl` is useful subject consumer, but 256 loses a useful shallow answer. |
| X01 | `Controller` | 4 / 7 / 4 / 3 / 2 | `useController` is first in four arms and directly supplies the render-prop values. |
| X02 | `FormProvider` + `useFormContext` | 1 / 1 / 1 / 1 / 1 | A direct answer is first and both required groups fit top 5 in every arm. |
| X03 | `Form` | 1 / 1 / 1 / 1 / 1 | Exact direct answer in all five. |
| X04 | `Watch` | 2 / 2 / 2 / 2 / 1 | Direct first at 256; `useWatch` is first and useful in the other arms. |
| X05 | `FieldArray` | 2 / 2 / 2 / 2 / 2 | `useFieldArray` is first in all five and directly supplies the operations. |
| X06 | `FormState` | 3 / 4 / 3 / 3 / 3 | `useFormState` is first in all five and is direct subscription support. |
| X07 | `Form` | 1 / 1 / 1 / 1 / 1 | Exact direct answer in all five. |
| X08 | `FormStateProps` | 13 / 17 / 13 / 11 / 14 | `useFormState` is first in all five; adjacent code is useful, but the exact props/render contract needs top 20. |

The persistent hard cases have different causes:

- G09 is an old-versus-new API intent-separation case and includes a declared
  hard negative.
- G07 is multi-parent completeness: binary can rank one direct parent first
  while still failing the full answer.
- T01 and X01 often return the implementation they delegate to before the
  public wrapper; those neighborhoods remain useful.
- T12 is a genuine 256 shallow-rank regression.
- X08 is a cross-profile type/props retrieval limitation. Every profile finds
  adjacent hook/type code early, but none retrieves the exact props contract
  in top 10.

## 7. Codec fidelity against exhaustive f32

| Profile | Paired f32 reference | Top-20 retention | Top-1 mismatch | Missing f32 top-20 memberships |
| --- | --- | ---: | ---: | ---: |
| 1024-f32 | reference | 1.0000 | .0000 | 0 |
| 1024-binary | same-run 1024-f32 | .7375 | .1875 | 168 |
| 1024-int8 | same-run 1024-f32 | .9938 | .0000 | 4 |
| 512-int8 | same-run 512-f32 | .9969 | .0313 | 2 |
| 256-int8 | same-run 256-f32 | .9938 | .0000 | 4 |

The 512 top-1 mismatch is one near-tie in chi G01 and is not an observed
usefulness loss. Int8 at every tested dimension closely reproduces its
same-run f32 neighborhood. Binary does not: it remains a valid independent
retriever, but it is not a high-fidelity approximation of f32.

## 8. Storage

A fresh provider-free 1,024-binary control was materialized from the existing
raw banks so complete SQLite size is measured on the same project-local layout
as the int8 controls. This operation made no Voyage call.

| Profile | Bytes/vector | Combined vector blobs | Combined SQLite files | Versus 1024-binary DB |
| --- | ---: | ---: | ---: | ---: |
| 1024-binary | 128 | 142,208 B | 2,785,280 B | baseline |
| 1024-int8 | 1,024 plus scale/norm | 1,137,664 B | 4,300,800 B | +54.41% |
| 512-int8 | 512 plus scale/norm | 568,832 B | 3,162,112 B | +13.53% |
| 256-int8 | 256 plus scale/norm | 284,416 B | 2,936,832 B | +5.44% |
| 1024-f32 raw bank | 4,096 | 4,550,656 B | 7,778,304 B isolated raw DB | development-only reference |

The f32 database is not a production serving comparison; raw f32 is isolated
development storage and never a runtime dependency. The complete int8 DB
reductions are 26.48% from 1,024 to 512, 7.12% from 512 to 256, and 31.71%
from 1,024 to 256. SQLite pages for source, chunks, segments, FTS, and metadata
do not shrink with vector dimension, which is why halving the vector payload
does not halve the whole database.

The f32 reference row uses the original isolated raw banks:

- `.cidx/test/states/chi/raw/embeddings.db`
- `.cidx/test/states/react-hook-form/raw/embeddings.db`

Local provider-free binary control paths:

- `.cidx/test/states/chi-1024-binary-control/db/index.db`
- `.cidx/test/states/react-hook-form-1024-binary-control/db/index.db`

## 9. Decision

1. Keep `1024-f32` as the exhaustive quality/fidelity reference only. Its
   4-byte payload is too large for the intended local production profile.
2. Carry `512-int8` as the preferred candidate into human-frozen calibration
   and confirmation. It has the strongest combined direct and useful shallow
   result here, near-f32 codec fidelity, and a complete DB only 13.53% larger
   than 1,024-binary.
3. Keep `256-int8` as the memory-constrained alternative. It is only 5.44%
   larger than the binary control and retains all direct answers by top 20,
   but it has four direct top-5 misses and the material T12 shallow loss.
4. Keep `1024-binary` as the unchanged working production baseline until the
   human-label and confirmation gates are complete. Its smallest vector
   representation is valuable, but its neighborhood fidelity, TSX result,
   G07 complete-answer loss, and source-reviewed precision do not support
   choosing it solely from the current draft Hit@K.
5. Do not use RRF or combine codec result lists. A serving profile selects one
   dimension and one codec, and the five evidence arms remain isolated.
6. Do not repeat Voyage when only labels or reporting change. A future final
   causal dimension comparison should persist one authorized source query
   response in-memory long enough to reduce it to 1,024/512/256 within one run.

This is a provisional profile direction, not production activation. The next
quality action remains separated human source review and provider-free replay
of the same immutable rankings.

## 10. Evidence and validation

Immutable ranking roots:

- 1,024 chi: `.cidx/test/states/chi/evaluations/codec-6532381320e61bc733552d95`
- 1,024 RHF: `.cidx/test/states/react-hook-form/evaluations/codec-a54f687f939e5b2588963905`
- 512 chi: `.cidx/test/states/chi-512/evaluations/codec-234846ae3107925a85cf0d59`
- 512 RHF: `.cidx/test/states/react-hook-form-512/evaluations/codec-02232abdd7d4428f39a9b874`
- 256 chi: `.cidx/test/states/chi-256/evaluations/codec-1449cbef0e5dfec9acb6601d`
- 256 RHF: `.cidx/test/states/react-hook-form-256/evaluations/codec-bd5e9d3441b6f3292737a74d`

The consolidation parses those immutable JSON/JSONL files, uses the original
draft required groups and cohort tags, and uses the complete advisory review
map for actual source usefulness. No production state, dataset, ranking,
label, source file, or query vector was changed. No new test code was added.

Final provider-free validation passed:

- every file listed by the six artifact checksum manifests matched its exact
  byte size and SHA-256, and every JSON/JSONL file parsed;
- direct Hit and complete-requirement counts were recomputed from all five
  representations and matched the tables in this report;
- SQLite `PRAGMA integrity_check` returned `ok` for both binary controls and
  all six 1,024/512/256 int8 controls;
- vector rows were exactly 619 chi plus 492 RHF in every production control,
  with expected dimensions, codec IDs, and blob totals; and
- the two original f32 raw banks also returned `integrity_check=ok` and exactly
  619/492 1,024-dimensional rows.
