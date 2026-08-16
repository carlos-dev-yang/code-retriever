# Phase 07 Measured Retrieval Loop — Revision 4

- State: `in_progress`
- Date: 2026-08-17
- Authority: calibration pool-building only
- Promotion eligible: no
- Labels: draft, two separated human passes pending
- Fixed serving profile: Voyage `voyage-code-4`, source 1,024, serving 1,024, binary

## Decision rule

A higher Hit@5, MRR, or NDCG is not sufficient. A candidate advances only
when reviewed requirement groups, complete-query coverage, known hard
negatives, evidence neighborhoods, and later-stage first loss do not regress.
The 32 cases stay in their language and cohort denominators; gains in one
slice never cancel a correctness loss in another.

## Provider-free FTS experiments

Every arm ran twice against the same chi draft-v3 and RHF draft-v2 snapshots.
Each repeat was byte-identical. Tokenization, candidate depth 20, return depth
5, exact-symbol handling, tie order, and all non-FTS lanes were held fixed.

### Production AND versus safe OR 5:1

The production all-token AND expression returned no candidates for all 32
natural-language cases. The evaluation-only safe-token OR expression produced:

| Corpus | Cases | Hit@5 | requirement coverage@5 | complete@5 | MRR@5 | provisional NDCG@5 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| chi | 12 | 0.5833 | 0.5417 | 0.5000 | 0.4583 | 0.4970 |
| RHF | 20 | 0.6000 | 0.5750 | 0.5500 | 0.5083 | 0.5085 |

All OR queries reached the candidate cap, so these values are provisional
pool-building observations rather than evidence that unrestricted OR is a
production policy.

### OR versus minimum-two-token admission

Minimum-two-token admission reduced pre-cap eligible pools, but all 32 queries
still saturated candidate depth. Required-group status was 0 improved, 32
unchanged, 0 regressed. Only chi G08 and RHF T04 changed top-five identities;
both evidence neighborhoods became less useful. The candidate was rejected.

### OR 5:1 versus OR 5:5

Eligible identities and counts were identical query by query, isolating the
field-weight change. Chi recorded 0 improved, 11 unchanged, 1 regressed; G07
lost `selectEncoder`. RHF recorded 3 improved, 15 unchanged, 2 regressed; T04
and X06 lost required implementation parents. Aggregate RHF coverage concealed
that exchange while MRR and NDCG fell. The candidate was rejected.

## Accepted next comparator

Stop FTS micro-tuning. Keep safe OR 5:1 only as an explicit development
comparator and leave production AND unchanged. Run all 32 queries freshly and
coherently through the shared Voyage query embedding, exhaustive target-f32,
active binary, parent collapse, RRF, ablation, and body-packaging code. Reuse
the already fingerprinted document bank, make zero document-provider calls,
persist no query vector, and reuse each request-local query vector only across
the arms for that same query.

The run is allowed before label freeze because it expands the blinded review
pool. Its quality numbers remain provisional and non-promotable. Assistant-use
correctness remains `NOT_OBSERVED`.

## Provenance and authorization boundary

The new development-only policy is a structured, full-field fingerprint. It
records the OR operator, query builder and normalizer versions, SQLite FTS
tokenizer/schema, fields, 5:1 resolved weights, depths, exact-symbol policy,
tie policy, and `production_policy_unchanged=true`. Production CLI, MCP, and
`Search` accept no such option.

Before a provider request, the evaluator requires:

- clean canonical VCS provenance and an executable SHA-256;
- exact corpus, generation, index manifest, profile, materialization, dataset,
  and ordered query-text identities;
- an ordered document-bank fingerprint derived from immutable raw-vector
  identities without serializing vectors;
- exactly 32 series operations, zero document operations, zero reused query
  vectors/rankings, and no query-vector persistence;
- a non-secret authorization reference, dated pricing identity, USD-per-million
  rate, conservative maximum cost, and a cap that contains that maximum.

The experiment manifest schema additionally records all profile fingerprints,
source/serving dimensions and codec, retry-policy fingerprint, actual provider
usage and accounted cost, retrieval/collapse/RRF/body policy identities,
per-query vector hashes, and planned/observed stage states. Any generation or
profile drift invalidates the run. Credentials never enter plans or artifacts.

## Review inputs

The primary metric-contract advisor recommended this coherent all-query run
instead of replaying 31 historical dense ranks with one new query. Two
independent final-direction reviews agreed to stop FTS tuning, keep OR 5:1
provisional, and proceed while tracking saturation, correctness exchanges,
hard negatives, requirement fragmentation, and lexical-to-body first loss.

## Focused implementation checks

Before the clean-provenance execution commit, the following passed:

```text
go test -count=1 ./internal/search ./internal/devlab
go test -race -count=1 ./internal/search ./internal/devlab
go vet ./internal/search ./internal/devlab
go build ./internal/search ./internal/devlab
git diff --check
```

The focused production/evaluation test proves that the public and ordinary
evaluation paths retain AND while only the fingerprinted development policy
uses safely quoted OR tokens. Mutated policy fields, incomplete authority,
and over-cap plans reject before execution. No provider, credential, corpus
mutation, document embedding, query embedding, or artifact publication was
performed at this checkpoint.

## Exact next action

Commit this provenance-safe execution boundary, build it from the clean commit,
run both provider-free dry plans, and confirm that their combined series is
12 chi plus 20 RHF operations with zero document operations and a conservative
maximum below USD 5. Only then source the ignored credential file for the two
bounded Voyage embedding-search invocations. Publish the stage-separated
scorecard, send its sanitized aggregate evidence to the metric advisor, and
obtain the final ChatGPT/Grok directional review before changing retrieval or
cohort structure.
