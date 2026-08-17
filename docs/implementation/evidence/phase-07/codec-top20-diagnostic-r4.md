# Paired f32/binary/int8 top-20 diagnostic — Revision 4

- Date: 2026-08-17
- Authority: calibration diagnostic only; not a human label freeze and not promotion evidence
- Production profile: `serving_dimensions=1024`, `storage_codec=binary` (unchanged)
- Evaluation executable commit: `11f3046a16c73a618a4d9847295a942f99db8868`
- Corpus/query set: pinned chi draft-v3 (12) and react-hook-form draft-v2 (20)

## 1. Dedicated comparison boundary

`cidx dev retrieval evaluate --mode codec` is a separate vector-only pipeline.
It opens one production snapshot for the corpus run, loads the authorized raw
f32 document bank once, transforms it once, and locally encodes the candidate
int8 bank once. Each case performs one Voyage query-embedding operation. The
returned ephemeral query f32 is the only vector shared by the three arms.

The arms then diverge:

- target f32 uses exhaustive serving-dimension f32 scoring;
- active binary prepares one sign-bit query and uses the binary scorer; and
- candidate int8 prepares one int8 query and uses the int8 scorer.

Binary and int8 do not share encoded values, distance arithmetic, thresholds,
or ranks. The comparison uses no FTS and no RRF. The only common post-score
operation is the established segment-to-semantic-parent collapse, followed by
portable parent de-duplication. Existing production search and the historical
fusion evaluator were not redirected through this path.

## 2. Execution and accounting

| Corpus | Run | Queries | Valid / retry / failed | Provider tokens | Accounted cost |
| --- | --- | ---: | --- | ---: | ---: |
| chi | `codec-6532381320e61bc733552d95` | 12 | 12 / 0 / 0 | 231 | USD 0.00002772 |
| react-hook-form | `codec-a54f687f939e5b2588963905` | 20 | 20 / 0 / 0 | 415 | USD 0.00004980 |

The successful series used 32 logical query operations, 32 provider attempts,
646 provider-reported tokens, and USD `0.00007752`. It made zero document
provider calls, persisted no query vector, and left production binary active.
One earlier chi attempt made one successful query call and stopped before
artifact publication when duplicate portable parents were exposed. That call
has no provider-token observation and is excluded from the successful-series
token total, but is retained as an operational caveat rather than silently
discarded.

Successful artifact roots:

- `.cidx/test/states/chi/evaluations/codec-6532381320e61bc733552d95`
- `.cidx/test/states/react-hook-form/evaluations/codec-a54f687f939e5b2588963905`

Their aggregate checksum manifests have SHA-256
`c37317915e899c3c2594155ad19a90021d5696089b2f728a8eada51ee2e62aa5`
and `8ca0069fc10f9ebf6909917f6ae39aa39c5356c3bec56d379b941e96eb56f96e`
respectively.

## 3. Existing direct-truth metrics

| Corpus | Arm | Hit@1 | Hit@5 | Hit@10 | Hit@20 | Complete@5 | Complete@20 |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| chi | f32 | .6667 | .9167 | 1.0000 | 1.0000 | .9167 | 1.0000 |
| chi | binary | .6667 | 1.0000 | 1.0000 | 1.0000 | .9167 | 1.0000 |
| chi | int8 | .6667 | .9167 | 1.0000 | 1.0000 | .9167 | 1.0000 |
| react-hook-form | f32 | .4000 | .9500 | .9500 | 1.0000 | .9500 | 1.0000 |
| react-hook-form | binary | .4500 | .8500 | .9500 | 1.0000 | .8500 | 1.0000 |
| react-hook-form | int8 | .4000 | .9500 | .9500 | 1.0000 | .9500 | 1.0000 |

The wider depth is material: RHF X08 is not a top-10 answer in any arm, but
appears at f32/int8 rank 13 and binary rank 17. Top-20 therefore exposes a
real tail-placement failure instead of incorrectly classifying the answer as
absent from the dense candidate space.

## 4. Codec fidelity to exhaustive f32

| Corpus | Codec | Top-20 retention | Top-1 mismatch | Mean displacement | Missing top-20 items |
| --- | --- | ---: | ---: | ---: | ---: |
| chi | binary | .7042 | .2500 | 3.4293 | 71 |
| chi | int8 | .9958 | .0000 | .0750 | 1 |
| react-hook-form | binary | .7575 | .1500 | 2.7178 | 97 |
| react-hook-form | int8 | .9925 | .0000 | .0508 | 3 |

Both codecs retain every currently declared direct-gold parent somewhere in
their top 20. That fact does not make their rankings equivalent: binary
replaces roughly one quarter to three tenths of the f32 top-20 neighborhoods,
while int8 differs by only four candidate memberships across 640 positions.

## 5. Blind pooled source review

For an actual-source check, the three arm rankings were pooled by query and
de-duplicated into 311 chi and 499 RHF question–chunk relations. The review
packets hide arm, rank, score, and retrieval history and contain the exact
public source body for each extracted chunk. The single root review marked
every relation as direct (`2`), useful support (`1`), or irrelevant (`0`):

| Corpus | Relations | Unique chunks | Direct | Support | Rest grade 0 |
| --- | ---: | ---: | ---: | ---: | ---: |
| chi | 311 | 183 | 16 | 49 | 246 |
| react-hook-form | 499 | 160 | 21 | 69 | 409 |

Packet SHA-256 values are
`d8183af3085d34ad96b522fd0a08f4c2af7b93dca216e6a443b3567367b93a83`
for chi and
`e248cd3bc2a6e7dbab51852b930687a4f4feea4864deb6d1732868cde979ce67`
for RHF. Review SHA-256 values are
`ed18c2ef0832f659f11c9e74a7e40ec3af4f2c46c7c56ced42a8fad95fb6807c`
and `a0b1a8a21e1a45f383be04e1b94a1dd543563d903d290e0fb711e6651b3e3634`.
The derived accuracy artifacts are
`80b6e103fbfc3c5677f4c897a08049e6d7ce46c09caef1f32d068cbea35c8220`
and `30d353f2fc8b335172508b56def24e9d824dfd776fdb8ef30e9d8f7a3721dd6c`.

The metrics below are pooled calibration diagnostics, not corpus-wide
precision claims. `P-useful` counts both direct and support; `R-pool` uses the
reviewed union of the three arms as its denominator; NDCG uses gains 3/1/0.

| Corpus | Arm | K | Direct hit | P-useful | R-pool | Pooled NDCG |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| chi | f32 | 1 | .7500 | .8333 | .2334 | .7778 |
| chi | f32 | 5 | .9167 | .5667 | .5906 | .7631 |
| chi | f32 | 10 | 1.0000 | .4250 | .8152 | .8016 |
| chi | f32 | 20 | 1.0000 | .2625 | .9722 | .8464 |
| chi | binary | 1 | .6667 | .7500 | .2168 | .6944 |
| chi | binary | 5 | 1.0000 | .5833 | .6032 | .7343 |
| chi | binary | 10 | 1.0000 | .3750 | .7284 | .7372 |
| chi | binary | 20 | 1.0000 | .2375 | .8875 | .7860 |
| chi | int8 | 1 | .7500 | .8333 | .2334 | .7778 |
| chi | int8 | 5 | .9167 | .5667 | .5906 | .7650 |
| chi | int8 | 10 | 1.0000 | .4250 | .8152 | .8032 |
| chi | int8 | 20 | 1.0000 | .2625 | .9722 | .8480 |
| react-hook-form | f32 | 1 | .4000 | .8000 | .2314 | .5333 |
| react-hook-form | f32 | 5 | .9500 | .4700 | .5667 | .6770 |
| react-hook-form | f32 | 10 | .9500 | .2850 | .6770 | .6930 |
| react-hook-form | f32 | 20 | 1.0000 | .2125 | .9444 | .7680 |
| react-hook-form | binary | 1 | .4500 | .8000 | .2331 | .5667 |
| react-hook-form | binary | 5 | .8500 | .4300 | .5209 | .6387 |
| react-hook-form | binary | 10 | .9500 | .2850 | .6534 | .6796 |
| react-hook-form | binary | 20 | 1.0000 | .1825 | .8139 | .7246 |
| react-hook-form | int8 | 1 | .4000 | .8000 | .2314 | .5333 |
| react-hook-form | int8 | 5 | .9500 | .4700 | .5667 | .6770 |
| react-hook-form | int8 | 10 | .9500 | .2850 | .6770 | .6930 |
| react-hook-form | int8 | 20 | 1.0000 | .2100 | .9319 | .7654 |

The source review confirms the central metric warning. Chi binary raises raw
Hit@5 by moving the G09 deprecated RealIP implementation upward, but at depth
20 it retains fewer useful chunks than f32/int8 in G01, G03, G04, G05, G08,
and G10. RHF binary similarly has a slightly higher Hit@1 yet loses useful
coverage in field-array, field-value, dotted-type-path, controller, context,
submit, FormState, and render-mode cases. A larger hit number can therefore
coexist with a worse answer neighborhood.

## 6. Decision

1. Keep production `1024/binary` unchanged while calibration remains open.
2. Retain int8 as the measured candidate codec. On these fixed development
   cases it preserves f32 membership and order far more closely than binary,
   without f32's local storage cost.
3. Do not mix binary and int8 scores and do not use RRF to combine their
   rankings. Every codec remains an isolated arm.
4. Do not rewrite cohort questions from this result. The observed misses are
   ranking/tail-placement and quantization effects, not evidence of a new
   parser or chunker failure.
5. Before formal label freeze, produce two independently randomized copies of
   the de-duplicated union and complete the required separated human source
   passes. The root review above is useful calibration evidence only.
6. No further Voyage query call is needed for label changes. Recompute labels
   and metrics provider-free against the immutable rankings.

