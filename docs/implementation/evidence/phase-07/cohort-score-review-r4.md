# Phase 07 chi/RHF Cohort Score Review — Revision 4

- Status: `diagnostic_complete_question_update_recorded`
- Date: 2026-08-16
- Claim scope: draft calibration question shaping only
- Fixed profile: 1,024-byte segments, 1,024 source/serving dimensions, binary codec, top 5

## 1. Measurement basis

No provider request was repeated for this review. It reuses the immutable
Voyage embedding-search rankings already recorded for 12 chi and 20 RHF
questions, together with the clean provider-free simple-control runs. The
original search artifacts remain unchanged:

- chi: `retrieval-7e5731ed1222a6aa432da84f`;
- RHF: `retrieval-20417011198b38cad4a1af2b`.

Together they contain 32/32 validated query responses, 636 observed input
tokens, zero retries, and zero failed attempts.

The Voyage rankings were produced before the accepted draft-v2 truth correction.
RHF T10 is therefore reinterpreted offline against the unchanged recorded top
five: `PathImpl` and `PathInternal` are ranks 2 and 3 in both serving f32 and
binary, so T10 is complete under current truth. Chi G09's added `walkXFF` hard
negative is absent from the recorded binary top five. No query vector or
ranking was regenerated, and no immutable artifact was rewritten.

Production FTS returned zero candidates for all 32 natural-language questions
under its all-token `AND` policy. The recorded hybrid arm therefore equals its
dense arm. This is a lane diagnosis, not an authorization to tune FTS.

## 2. Cohort observations

Values are `Hit@5`; the last two columns are binary requirement coverage and
binary complete-requirement rate. Cohorts overlap and must not be summed.

### chi

| Cohort | n | Simple | f32 | Binary | Binary coverage | Binary complete |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `signal:mixed` | 3 | 100.0% | 66.7% | 100.0% | 100.0% | 100.0% |
| `signal:semantic_only` | 9 | 22.2% | 100.0% | 100.0% | 94.4% | 88.9% |
| `task:delegated_or_cross_parent_flow` | 5 | 40.0% | 80.0% | 100.0% | 90.0% | 80.0% |
| `task:lifecycle_state_error_or_configuration` | 3 | 66.7% | 100.0% | 100.0% | 100.0% | 100.0% |
| `task:single_parent_behavior` | 4 | 25.0% | 100.0% | 100.0% | 100.0% | 100.0% |

### react-hook-form, corrected draft-v2 truth

| Cohort | n | Simple | f32 | Binary | Binary coverage | Binary complete |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `signal:mixed` | 5 | 60.0% | 80.0% | 80.0% | 80.0% | 80.0% |
| `signal:semantic_only` | 15 | 73.3% | 100.0% | 86.7% | 86.7% | 86.7% |
| `task:delegated_or_cross_parent_flow` | 7 | 85.7% | 100.0% | 71.4% | 71.4% | 71.4% |
| `task:interface_type_or_api_contract` | 3 | 33.3% | 66.7% | 66.7% | 66.7% | 66.7% |
| `task:lifecycle_state_error_or_configuration` | 5 | 60.0% | 100.0% | 100.0% | 100.0% | 100.0% |
| `task:single_parent_behavior` | 5 | 80.0% | 100.0% | 100.0% | 100.0% | 100.0% |

The corrected TypeScript binary result is Hit@5 `11/12`, requirement coverage
`11/12`, and complete requirements `11/12`. TSX remains `6/8` for all three.
These small, structurally different slices do not establish a language effect.

## 3. Failure decisions

All four current binary failures remain in the working cohort:

| Case | Observation | Decision |
| --- | --- | --- |
| `chi-g07-compression-selection` | binary keeps `selectEncoder` at rank 1 but loses the separately required `isCompressible`; f32 contains both | `KEEP` — real two-parent completeness and codec-boundary case |
| `rhf-t01-form-control-lifecycle` | binary returns creation/proxy/state helpers but loses required root orchestrator `useForm`; f32 has it at rank 5 | `KEEP` — real orchestration and binary-displacement case |
| `rhf-x01-controller-render-prop` | binary returns `useController` and related helpers but loses the `Controller` wrapper; f32 has it at rank 4 | `KEEP` — real wrapper-versus-delegate case |
| `rhf-x08-form-state-props` | `FormStateProps` is outside both dense top fives while supporting `FormState` ranks second | `KEEP` — real thin public type-contract miss shared by f32 and binary |

Low score alone is not a removal rule. A Grok advisory recommended dropping
X08 only because neither dense arm returned it; that recommendation conflicts
with the recorded difficult-case contract and is rejected. A separate ChatGPT
advisory recommended keeping all four and distinguished binary displacement
from the shared X08 dense miss. Both advisory outputs remain non-authoritative;
the source basis and measured removal effect above control the decision.

No new question is added before the next measured run. The observed failures
already cover multi-parent completeness, root orchestration, TSX wrapper
delegation, and a thin TSX type contract without quota-padding edge cases.

## 4. chi G12 wording correction

The previous question said “the first matching header middleware”. The source
iterates distinct header keys through a Go map, whose iteration order is not a
global registration-order contract. The current chi working dataset is now:

- `testdata/retrieval/behavior-go-chi-v5.3.1-draft-v3.json`;
- file SHA-256 `7918210192b558240c5347fd74afc18d97f840837798334427038cc2cd39e505`;
- G12 case digest `0d1edd584a89cd4393abbbd4ae7399ac74bd9cb1d2cf811ab43c40d8c8feeeba`.

The revised question is:

> How does header routing evaluate exact or wildcard header matchers, invoke a
> matching middleware, and use the default route when none match?

Version 2 and its rankings/pools remain historical diagnostic evidence. The
new wording has not been query-embedded or scored, so the old G12 result must
not be relabeled as a v3 measurement. Before human label freeze, refresh the
provider-free simple result and the opened-arm pool for this changed query;
any new paid query operation remains a separately recorded bounded action.

## 5. Boundary conclusion

The corpus/parser/chunker/segment structure is not reopened: every required
parent exists in the exhaustive inventory, and the misses begin at top-five
ranking. Keep the fixed 1,024/binary working profile, keep all current cases,
add none, and use the next measured run to observe the single G12 wording
change rather than repeating advisory label debates.
