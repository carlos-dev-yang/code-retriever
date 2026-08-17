# 256-dimensional int8 follow-up — Revision 4

- Date: 2026-08-17
- Authority: calibration diagnostic only; not a human label freeze or promotion evidence
- Working baseline: `serving_dimensions=1024`, `storage_codec=binary` (unchanged)
- Evaluation executable commit: `1cc9fd5a6a340908c42556edf7bcd6208eaa2929`
- Evaluation executable SHA-256: `ff598f8438a65905c5c895eafb13f7deb78ae5f7be85bcf003e78b64d66ed846`
- Corpus/query set: pinned chi draft-v3 (12) and react-hook-form draft-v2 (20)

## 1. Experiment boundary

This final compact-dimension diagnostic used new, separate project-local
development states. It did not modify the 1,024- or 512-dimensional states,
artifacts, raw banks, or the working production profile. The shared
`prefix-l2-v1` transformer locally reduced each authorized Voyage
`voyage-code-4` 1,024-f32 document vector to 256 dimensions. The evaluator then
prepared independent 256-f32, 256-binary, and 256-int8 scorers.

Each case made one fresh Voyage query-embedding operation at source dimension
1,024. Its ephemeral response was reduced once to 256 and reused only by the
three arms in that run. The experiment made no document-provider operation,
persisted no query vector, and used no FTS or RRF. Int8 fidelity is therefore
measured against same-run exhaustive 256-f32.

None of the 32 query-vector hashes exactly matched the prior 512 or 1,024
checkpoint hashes. Cross-dimension differences consequently include both the
serving-dimension change and a fresh provider response; they are not causal,
same-query dimension deltas. A later formal dimension choice must reuse one
source query vector across every compared dimension in one run.

## 2. Execution and accounting

| Corpus | Run | Raw documents | Queries | Valid / retry / failed | Tokens | Cost |
| --- | --- | ---: | ---: | --- | ---: | ---: |
| chi | `codec-1449cbef0e5dfec9acb6601d` | 619 | 12 | 12 / 0 / 0 | 231 | USD 0.00002772 |
| react-hook-form | `codec-bd5e9d3441b6f3292737a74d` | 492 | 20 | 20 / 0 / 0 | 415 | USD 0.00004980 |

The complete series used 32 logical query operations, 32 provider attempts,
646 provider-reported tokens, zero retry or failed attempts, and USD
`0.00007752`. Both artifacts report `source_modified=false` and zero document
provider operations.

Artifact roots:

- `.cidx/test/states/chi-256/evaluations/codec-1449cbef0e5dfec9acb6601d`
- `.cidx/test/states/react-hook-form-256/evaluations/codec-bd5e9d3441b6f3292737a74d`

Their aggregate artifact checksums are
`7b51974dfdc6540e7521ef136057aff8c33adfd1fa3c8e263df65d8a9508ac6e`
and `8357c139b622a263c460387b351f68d5def2a3b87415275f76de2516db4dc945`.
The compatible raw document-bank fingerprints remain
`6a9ef289e86dcadb2dd6e8de8703650160c32c54c0f3b2dd2451b9cce8608efe`
and `59b34a78c59e34323eabcb222441de7802012364be7cbdbff550e48d623f2f27`.

## 3. Same-run codec fidelity

| Corpus | Codec | Top-20 retention | Top-1 mismatch | Mean displacement | Missing f32 top-20 items | Direct-gold retention |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| chi | binary | .5375 | .6667 | 4.5642 | 111 | .9167 |
| chi | int8 | .9917 | .0000 | .1434 | 2 | 1.0000 |
| react-hook-form | binary | .5850 | .4000 | 4.1280 | 166 | .9000 |
| react-hook-form | int8 | .9950 | .0000 | .1029 | 2 | 1.0000 |

The four int8 membership changes occur only at the top-20 boundary. Source
inspection found no lost direct answer:

- chi G01 exchanges an unrelated regexp-recursion test for the route-pattern
  accessor; neither is the direct pooled-context implementation;
- chi G09 exchanges one newer XFF test for one newer RemoteAddr test; the
  deprecated `RealIP` answer remains rank 7;
- RHF T03 exchanges the `UseFieldArrayUpdate` operation type for the
  `FieldArray` value type, both adjacent support rather than the hook body; and
- RHF X04 exchanges validation-rule extraction for `FormProvider`; neither is
  the `Watch` render-prop implementation, while the replacement is at least
  context support.

Thus the 256 int8 quantizer itself caused no observed answer loss relative to
same-run 256-f32. Binary again changes a much larger part of the neighborhood.

## 4. Draft direct-truth metrics

| Corpus | Checkpoint | Hit@1 | Hit@5 | Hit@10 | Hit@20 | Complete@5 | Complete@20 | MRR@5 | NDCG@20 |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| chi | 256 int8 | .5833 | .9167 | 1.0000 | 1.0000 | .9167 | 1.0000 | .7222 | .7787 |
| chi | 512 int8 reference | .7500 | 1.0000 | 1.0000 | 1.0000 | 1.0000 | 1.0000 | .8542 | .8607 |
| chi | 1024 int8 reference | .6667 | .9167 | 1.0000 | 1.0000 | .9167 | 1.0000 | .7917 | .8266 |
| react-hook-form | 256 int8 | .5000 | .8500 | .9000 | 1.0000 | .8500 | 1.0000 | .6667 | .7838 |
| react-hook-form | 512 int8 reference | .4000 | .9500 | .9500 | 1.0000 | .9500 | 1.0000 | .6225 | .7563 |
| react-hook-form | 1024 int8 reference | .4000 | .9500 | .9500 | 1.0000 | .9500 | 1.0000 | .6308 | .7679 |

At 256, int8 and same-run f32 have identical Hit, completeness, and MRR at
every recorded cutoff. RHF NDCG@20 differs only from `.7840` to `.7838` because
of within-pool order changes.

The practical shallow-rank misses deserve source-backed interpretation:

- chi G09 places deprecated `middleware.RealIP` at rank 7. Ranks 1–6 are the
  newer client-IP API and tests, including the declared `walkXFF` hard negative
  at rank 5. This is a real top-five intent-separation miss.
- RHF T01 places direct `useForm` at rank 6. Its top five include useful
  lifecycle pieces such as `createFormControl` and `getProxyFormState`, so the
  neighborhood is useful even though the complete answer misses top five.
- RHF T12 places direct `createSubject` at rank 14. `createFormControl` at rank
  1 is a major consumer of the subject, but it does not implement observer
  add/notify/unsubscribe/clear. This is the clearest 256 tail-placement loss.
- RHF X08 places direct `FormStateProps` at rank 14. `FormState` at rank 3
  consumes the contract and `UseFormStateProps` at rank 2 supplies its base
  fields but not the render callback. The results are useful adjacent code,
  while the exact declaration remains too deep; this issue also existed in the
  larger-dimension checkpoints.

The 256 RHF language slices are TypeScript Hit@1/5/10/20
`.5000/.8333/.9167/1.0000` and TSX `.5000/.8750/.8750/1.0000`. Every existing
question has a direct answer by top 20. These 32 draft questions remain too
small for a production dimension decision.

## 5. Storage measurement

Separate provider-free int8 states were materialized from the same raw banks
at 1,024, 512, and 256 dimensions. Int8 stores one vector byte per dimension,
so its vector blob is exactly quartered from 1,024 to 256 and halved from 512
to 256.

| Corpus | Vectors | 1024-int8 DB | 512-int8 DB | 256-int8 DB | 512 to 256 DB reduction |
| --- | ---: | ---: | ---: | ---: | ---: |
| chi | 619 | 2,428,928 B | 1,794,048 B | 1,667,072 B | 126,976 B / 7.08% |
| react-hook-form | 492 | 1,871,872 B | 1,368,064 B | 1,269,760 B | 98,304 B / 7.19% |
| combined | 1,111 | 4,300,800 B | 3,162,112 B | 2,936,832 B | 225,280 B / 7.12% |

Combined vector blobs are 1,137,664 B at 1,024, 568,832 B at 512, and 284,416
B at 256. The complete 256 databases are 31.71% smaller than the complete
1,024-int8 controls, not 75% smaller, because source, chunk, segment, FTS,
schema-page, and row metadata do not shrink with vector dimension.

## 6. Decision

1. Preserve the working `1024/binary` baseline and every prior checkpoint.
2. Accept `256/int8` as a viable memory-constrained diagnostic profile: its
   same-run quantization fidelity remains excellent and all direct answers
   survive by top 20.
3. Keep `512/int8` as the preferred compact candidate for the next frozen
   evaluation. On these complete databases, 256 saves only another `7.12%`
   beyond 512 while the observed shallow results are less stable, especially
   RHF T12.
4. Do not attribute every 256-versus-512 rank change to dimension because the
   query source vectors differ. Any final dimension selection must compare all
   dimensions from one shared source query vector per case.
5. Do not activate either int8 candidate in production, fuse codec ranks, or
   repeat Voyage when only labels change. Human label freeze and confirmation
   remain required.

## 7. Boundary validation

No new test code was added. Before the provider run, focused devlab, search,
vector, and devapp tests, vet, build, module-diff, formatting, and diff checks
passed. After artifact publication, the one final boundary passed:

- `go test -count=1 ./...`;
- `go vet ./...`, `go build ./...`, and `go mod tidy -diff`;
- `git diff --check` and a tracked-diff credential-assignment scan;
- structural JSON/JSONL parsing and exact byte-size/SHA-256 verification for
  every entry in both artifact checksum manifests;
- SQLite `PRAGMA integrity_check` for both 256 binary states, both 256 int8
  controls, and both copied raw banks;
- exact 619/492 vector counts, dimension 256, int8 codec identity, and blob
  totals 158,464/125,952 bytes; and
- clean executable commit/source-modified identity plus executable SHA-256.

An initial combined validation wrapper exited after its successful full-test
stage because of a shell assertion in the later verification block. It made no
provider call. Each remaining build, artifact, database, config, and executable
check was then rerun separately and passed.
