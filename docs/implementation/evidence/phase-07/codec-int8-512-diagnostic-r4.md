# 512-dimensional int8 follow-up — Revision 4

- Date: 2026-08-17
- Authority: calibration diagnostic only; not a human label freeze or promotion evidence
- Working baseline: `serving_dimensions=1024`, `storage_codec=binary` (unchanged)
- Evaluation executable commit: `5c55b024feae5f241940ecdd1bb53fdce9e223bf`
- Evaluation executable SHA-256: `93201a50e15ea8c39d0cef30111e8e5848f436d2bb50fe3ceb7814bb7a14b083`
- Corpus/query set: pinned chi draft-v3 (12) and react-hook-form draft-v2 (20)

## 1. Experiment boundary

This follow-up used separate project-local development states and did not
modify the existing 1,024-dimensional databases or artifacts. The approved
document raw banks remain Voyage `voyage-code-4` 1,024-dimensional f32. The
shared `prefix-l2-v1` transformer reduced those documents locally to 512
dimensions, after which the dedicated codec evaluator independently prepared
512-f32, 512-binary, and 512-int8 arms.

Each case made one fresh Voyage query-embedding operation at source dimension
1,024. The same ephemeral result was locally reduced to 512 and reused within
that run's three arms. There were no document-provider operations, persisted
query vectors, FTS inputs, RRF inputs, or mixed codec scores. Int8 fidelity is
therefore measured against same-run exhaustive 512-f32. The comparison to the
earlier 1,024-dimensional result is a comparison of separate checkpoints, not
a same-query paired delta.

The codec evaluator was generalized only from serving dimension 1,024 to
serving dimensions 512 or 1,024. Source dimension 1,024, active binary
snapshot, `candidate_k >= 20`, clean provenance, provider accounting, and the
isolated scorers remain mandatory.

## 2. Execution and accounting

| Corpus | Run | Raw documents | Queries | Valid / retry / failed | Tokens | Cost |
| --- | --- | ---: | ---: | --- | ---: | ---: |
| chi | `codec-234846ae3107925a85cf0d59` | 619 | 12 | 12 / 0 / 0 | 231 | USD 0.00002772 |
| react-hook-form | `codec-02232abdd7d4428f39a9b874` | 492 | 20 | 20 / 0 / 0 | 415 | USD 0.00004980 |

The complete series used 32 logical query operations, 32 provider attempts,
646 provider-reported tokens, zero retry or failed attempts, and USD
`0.00007752`. Both artifacts report `source_modified=false`.

Artifact roots:

- `.cidx/test/states/chi-512/evaluations/codec-234846ae3107925a85cf0d59`
- `.cidx/test/states/react-hook-form-512/evaluations/codec-02232abdd7d4428f39a9b874`

Their aggregate artifact checksums are
`cfd21645befb69363ffd209d0b670610a09f870632bc92b104e03a89ab8294b6`
and `770fb1afdca42241427e677ff34665df4741f4931e294f703c1c4da209b97bf9`.
The transformed document-bank fingerprints are
`6a9ef289e86dcadb2dd6e8de8703650160c32c54c0f3b2dd2451b9cce8608efe`
and `59b34a78c59e34323eabcb222441de7802012364be7cbdbff550e48d623f2f27`.

## 3. Same-run codec fidelity

| Corpus | Serving dimension | Codec | Top-20 retention | Top-1 mismatch | Mean displacement | Missing f32 top-20 items |
| --- | ---: | --- | ---: | ---: | ---: | ---: |
| chi | 512 | int8 | 1.0000 | .0833 | .1000 | 0 |
| react-hook-form | 512 | int8 | .9950 | .0000 | .0853 | 2 |
| chi reference | 1024 | int8 | .9958 | .0000 | .0750 | 1 |
| react-hook-form reference | 1024 | int8 | .9925 | .0000 | .0508 | 3 |

The chi top-1 mismatch is `chi-g01-context-pool`. Exhaustive 512-f32 ranks
`chi.node.findRoute` first and `chi.Context.Reset` second with only
`0.00001076` score separation; 512-int8 swaps them. The prior source review
grades `findRoute` as irrelevant for this question and `Context.Reset` as
useful support, while the direct `chi.Mux.ServeHTTP` answer remains rank 4 in
both arms. The fidelity mismatch is therefore not a usefulness regression.

The two RHF membership changes occur only at the top-20 boundary in T02 and
T11. The T02 displaced and replacement parents are both grade 0. In T11,
int8 drops the f32 rank-20 `MultipleFieldErrors` type, which is useful support
for all-criteria mode, and replaces it with grade-0 `useForm`. Every declared
direct-gold parent is nevertheless retained.

## 4. Draft direct-truth metrics

| Corpus | Checkpoint | Hit@1 | Hit@5 | Hit@10 | Hit@20 | Complete@5 | Complete@20 | MRR@5 | NDCG@20 |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| chi | 512 int8 | .7500 | 1.0000 | 1.0000 | 1.0000 | 1.0000 | 1.0000 | .8542 | .8607 |
| chi | 1024 int8 reference | .6667 | .9167 | 1.0000 | 1.0000 | .9167 | 1.0000 | .7917 | .8266 |
| react-hook-form | 512 int8 | .4000 | .9500 | .9500 | 1.0000 | .9500 | 1.0000 | .6225 | .7563 |
| react-hook-form | 1024 int8 reference | .4000 | .9500 | .9500 | 1.0000 | .9500 | 1.0000 | .6308 | .7679 |

The cross-dimension changes do not justify a causal claim because query
vectors were freshly generated and the serving space changed. They do show no
material draft retrieval collapse at 512. Chi loses G01 direct rank 1 to rank
4 but gains G02 rank 2 to rank 1 and G09 rank 8 to rank 1. RHF changes several
within-top-20 positions without changing any reported hit or completeness
cutoff.

## 5. Source-reviewed usefulness

The previous blind packet covered 626 of the 640 new 512-int8 query-parent
relations. The remaining 14 relations were inspected from their exact pinned
source bodies: 10 were useful support and four were grade 0; none was a new
direct answer. The one f32-only T11 boundary parent was also inspected and
graded useful support so codec loss is not hidden from the recall denominator.
This supplemental root review is advisory and is not a substitute for the
required separated human label passes.

| Corpus | Checkpoint | K | Direct hit | Useful precision | Reviewed-pool recall |
| --- | --- | ---: | ---: | ---: | ---: |
| chi | 512 int8 | 1 | .7500 | .9167 | .2339 |
| chi | 512 int8 | 5 | 1.0000 | .5833 | .5554 |
| chi | 512 int8 | 10 | 1.0000 | .4167 | .7239 |
| chi | 512 int8 | 20 | 1.0000 | .2875 | .9603 |
| react-hook-form | 512 int8 | 1 | .4000 | .8000 | .2273 |
| react-hook-form | 512 int8 | 5 | .9500 | .4600 | .5449 |
| react-hook-form | 512 int8 | 10 | .9500 | .3050 | .6910 |
| react-hook-form | 512 int8 | 20 | 1.0000 | .2100 | .8980 |

The corresponding 1,024-int8 useful precision values were chi
`.8333/.5667/.4250/.2625` and RHF `.8000/.4700/.2850/.2100` at
1/5/10/20. Recomputing the 1,024 rankings against the same expanded relevance
pool gives reviewed-pool recall of chi
`.2200/.5415/.7426/.8828` and RHF
`.2273/.5496/.6587/.9022`. On that like-for-like denominator, 512 is higher
for chi at 1/5/20 and lower by `.0187` at 10; RHF differs by at most `.0324`
at every cutoff. The result is mixed by small amounts rather than uniformly
higher, but the actual source neighborhoods remain comparable and every query
has useful source within the top five.

Local supplemental artifacts and their SHA-256 values:

- `int8-512-supplemental-root-review.json`:
  `5d2632b83f2c5c179efdc99a96121c2d44164b07006a7787d6d562216904c067`
- `chi-int8-512-source-accuracy.json`:
  `e22e7ce0a807cf78aa41e10342c33719bffc0e37c98bde2d0f0105a5b652038d`
- `rhf-int8-512-source-accuracy.json`:
  `74bba3aaee2e4dc51f2ac0f357495a84bbc43402efe95785b48fc66fd0ab1ca8`
- `chi-int8-1024-expanded-pool-source-accuracy.json`:
  `d064797126003decb9c501290d3c8bc0efa7a25e596eb152bd868c85ff150fab`
- `rhf-int8-1024-expanded-pool-source-accuracy.json`:
  `f2eab0edd13ac16dbdf538ceb6b2ab460b02c202e86aee2d1675952a614b1c78`

## 6. Storage measurement

The production int8 payload is one signed byte per dimension plus per-vector
scale and norm metadata. Reducing 1,024 to 512 therefore halves the vector blob
from 1,024 to 512 bytes. Across the 1,111 vectors, blob payload falls from
1,137,664 to 568,832 bytes.

Separate provider-free 1,024-int8 and 512-int8 development states were built
from the same raw banks to measure the complete SQLite files:

| Corpus | Vectors | 1024-int8 DB | 512-int8 DB | Reduction |
| --- | ---: | ---: | ---: | ---: |
| chi | 619 | 2,428,928 B | 1,794,048 B | 634,880 B / 26.14% |
| react-hook-form | 492 | 1,871,872 B | 1,368,064 B | 503,808 B / 26.91% |
| combined | 1,111 | 4,300,800 B | 3,162,112 B | 1,138,688 B / 26.48% |

The whole database falls by less than 50% because source files, chunks,
segments, FTS, schema pages, and row metadata do not shrink with vector
dimension. The vector payload itself is exactly halved.

## 7. Decision

1. Preserve the existing `1024/binary` working baseline and all prior
   artifacts.
2. Carry `512/int8` forward as the preferred compact candidate for the next
   frozen calibration and confirmation checks. It halves int8 vector payload,
   reduces these complete SQLite files by about 26.5%, retains every direct
   answer by top 20, and remains nearly identical to same-run 512-f32.
3. Do not declare `512/int8` the v1 production profile from this 32-question
   draft calibration set. Separated human labels and confirmation evidence are
   still required.
4. Keep binary and int8 scorers isolated. Do not add RRF or mix their scores.
5. Label-only changes require no further Voyage query operation against these
   immutable rankings.
