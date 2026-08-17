# Phase 07 Dual-AI Calibration Freeze and Provider-Free Replay

- State: accepted calibration checkpoint; not confirmation or promotion evidence
- Date: 2026-08-17
- Corpora: `go-chi-chi-v5.3.1`, `react-hook-form-v7.85.0`
- Serving lane: default `1024/int8`
- Review protocol: `owner-adopted-dual-ai-v1`
- Relevance authority: `OWNER_ADOPTED_DUAL_AI_REVIEW`
- Review limitation: `NO_INDEPENDENT_HUMAN_REVIEW`

## Outcome

The 12 chi and 20 react-hook-form behavior questions are now closed as one
32-case calibration set. ChatGPT and Grok independently reviewed every pooled
query-parent relation from source-complete, shuffled packets with rank, score,
lane, prior labels, experiment results, the other pass, and owner preference
hidden. Source-backed reconciliation resolved every direct/group disagreement.
The owner then adopted both reconciled label payloads as a whole without a
relation-level override.

The frozen datasets are:

- `testdata/retrieval/behavior-go-chi-v5.3.1-calibration-frozen-v1.json`
  — 12 Go cases, dataset fingerprint
  `c89ff2760445205937ec2a556d29d8b5a177ef371468ed8616b6221550f620d2`,
  file SHA-256
  `34d95e76d57d88be57cdf23f341c10724dd42fcfe213786b8620595a0ae9c1e1`;
- `testdata/retrieval/behavior-react-hook-form-v7.85.0-calibration-frozen-v1.json`
  — 12 TypeScript and 8 TSX cases, dataset fingerprint
  `558f6b84185ba6dbea55dac975284311510e73f563bf42512737d005f79e0cda`,
  file SHA-256
  `e5c93b9e7823e155b0c31e7b2994ba1ccf96880fcad5e680bc7a46adbcbd8ecf`;
- `testdata/retrieval/reviews/owner-adoption-chi-rhf-calibration-v1.json`
  — whole-digest adoption, file SHA-256
  `96a8a2bc2e90ea66c3441d89215db3fd2c9b95600bcd57b488cf7b012acf3015`.

All 32 case digests were recomputed with the evaluation contract's RFC 8785
framing. Every case is `split=calibration`, every case is frozen under the same
authority tuple, every required parent is present in the indexed inventory,
and every declared hard negative has an explicit grade-0 judgment. The set is
large enough to compare the current chi/RHF retrieval behavior, but it is below
the promotion-capable confirmation floor and must not vote for promotion.

## Review closure

### chi

- pool: 513 relations across 234 unique semantic parents;
- complete passes: ChatGPT `513/513`, Grok `513/513`;
- initial exact agreement: 396; reconciled differences: 117;
- final grades: 467 grade 0, 32 grade 1, 14 grade 2;
- ChatGPT review SHA-256:
  `c11779745ffc211beffdaae570c8c0cc80a4c2ab5c802ea6933a910bd251cf59`;
- Grok review SHA-256:
  `d28a0a8b0456427095d706a1a8babf8c45a8632fae17471377fb2d241ba7d749`;
- reconciliation SHA-256:
  `d0389a9df56f0cec0643124e6e4444cbbb2943b6f4697dea1f93a15aaae703d3`;
- reconciled-label payload SHA-256:
  `a302d3b8800e6d30072d620f06ccf597a907c18f9987221d147df32f120e559e`.

The material direct/group disagreement was G09. Source inspection preserved
two required operations: `middleware.realIP` for trusted-header selection and
`middleware.RealIP` for `RemoteAddr` mutation. `walkXFF` remains a reviewed
grade-0 misleading candidate. Dense retrieves a direct G09 parent by top 5,
but does not retrieve both required operations by top 20; this is retained as
a parser/chunker/retrieval diagnostic rather than rewritten after seeing the
result.

### react-hook-form

- pool: 735 relations across 181 unique semantic parents;
- complete passes: ChatGPT `735/735`, Grok `735/735`;
- initial exact agreement: 597; reconciled differences: 138;
- final grades: 669 grade 0, 44 grade 1, 22 grade 2;
- all 20 required-group sets retain at least one grade-2 alternative;
- ChatGPT review SHA-256:
  `83201d000e45bb050e79eae08be3442b931831bde26d8930eb24aa9cfc721220`;
- Grok review SHA-256:
  `3d651e6de1b6ddd9773016b06410abbfd0fcca93ac1798ff9636139e6fab7749`;
- reconciliation SHA-256:
  `de30be4dfefc742e7d9904dd5710f66f1dbc8816660a152a7922e96c4217ac0e`;
- reconciled-label payload SHA-256:
  `682d5aef014d60be62e45030129d26fb3515a4e78f7ae8728be80c14f16c43d9`.

One malformed candidate-ID emitted by a reviewer was bound to the only exact
source candidate after proving all semantic fields identical; no grade,
rationale, group, or source field changed. Grade 2 is retained only on dual
consensus, grade 1 only on dual agreement, and required groups are the
reconciled intersection.

## Provider-free replay

The frozen labels were replayed against the already immutable ranks. No
credential was read, no corpus or query was sent to Voyage, no query embedding
was repeated, and no production ranking/configuration was changed. The ignored
local replay artifacts are:

- `.cidx/test/experiments/review-union/current/chi-frozen-replay.json`,
  SHA-256
  `90efcc02c9c4e826515ad56d5d3a96104782840503e110e3835b24880cd50bb5`;
- `.cidx/test/experiments/review-union/current/rhf-frozen-replay.json`,
  SHA-256
  `5909878346400b307d1f97baea1f5ce939b0ae10ad722d4ef8799e8f57b67bd4`.

The fixed lanes are the accepted simple control, FTS, exhaustive current
`1024/int8` dense rankings, equal `1:1` RRF, and one bounded `FTS 1 : dense 2`
RRF probe. All RRF arms use `k=60`; no per-query exception is present.

| Corpus / lane | Complete@5 | Hit@5 | Complete@20 | Hit@20 | NDCG@5 | NDCG@20 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| chi dense 1024/int8 | 11/12 | 11/12 | 11/12 | 12/12 | 0.718862 | 0.809029 |
| chi FTS | 6/12 | 7/12 | — | — | 0.486561 | — |
| chi RRF 1:1 | 10/12 | 11/12 | — | — | 0.667021 | — |
| chi RRF FTS1:dense2 | 11/12 | 11/12 | — | — | 0.676379 | — |
| RHF dense 1024/int8 | 19/20 | 19/20 | 20/20 | 20/20 | 0.758856 | 0.815379 |
| RHF FTS | 11/20 | 12/20 | — | — | 0.463057 | — |
| RHF RRF 1:1 | 14/20 | 15/20 | — | — | 0.593404 | — |
| RHF RRF FTS1:dense2 | 17/20 | 18/20 | — | — | 0.664677 | — |

Across all 32 cases, dense has `30/32` Hit@5 and Complete@5, `32/32`
Hit@20, and `31/32` Complete@20. RHF TS is `12/12` Complete@5 and TSX is
`7/8`; X08 becomes complete by top 20. chi G09 is the only incomplete dense
case at top 20 and its reviewed misleading `walkXFF` appears in the top 5.
Those two cases stay in the calibration set because they reveal different
structural limits.

The weighted probe recovered some equal-RRF losses, but still regressed from
dense on RHF T09 and T10. It also worsened the reviewed grade mix in the RHF
top-5 pool. The predeclared no-dense-regression and evidence-usefulness gates
therefore failed. This is a result-level failure, not a low-score-only
decision.

## Accepted direction

The metric packet was reviewed by the measurement guide first, then by
ChatGPT and Grok for the product direction. All three independently converged
on the same bounded decision:

1. retain `1024/int8` dense as the current retrieval-quality calibration
   baseline;
2. retain FTS as a separate lexical control;
3. reject the tested equal `1:1` and `FTS 1 : dense 2` RRF calibration arms;
4. stop weight tuning unless a structural change, a new candidate generator,
   or new frozen labels creates a genuinely new experiment;
5. do not generalize this result into a claim that lexical retrieval can never
   add value.

The direction-review artifact is
`.cidx/test/experiments/review-union/current/calibration-direction-review.json`,
SHA-256
`0c82f50200c4c3a25f332c0781f264ccb5aff113ca98c08ea031cdfbf9a85443`.
This decision records calibration evidence only. It does not silently change
the already accepted production Phase 11 policy; a serving-policy change needs
its own design and implementation boundary.

## Boundary validation

One credential-free, network-disabled phase-boundary pass completed after the
freeze. It regenerated both frozen datasets from their own case payloads,
recomputed all case digests and dataset fingerprints, byte-compared the
outputs, checked the adoption/replay/direction hashes above, parsed every new
tracked JSON file, and then ran:

```text
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 ./...
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 -race ./internal/evalcontract ./internal/eval ./internal/devlab
env -u VOYAGE_API_KEY GOPROXY=off go vet ./internal/evalcontract ./internal/eval ./internal/devlab
env -u VOYAGE_API_KEY GOPROXY=off go build ./...
go mod tidy -diff
gofmt -l internal/evalcontract internal/eval internal/devlab
git diff --check
```

Every command passed. No provider, credential, corpus mutation, production
state mutation, or new retrieval experiment was part of this validation.

## Remaining boundary

The chi/RHF calibration questions and labels are closed. They must not be
edited to improve these exposed results. Mixed-language work remains deferred
until after this two-corpus checkpoint, as directed by the owner. A future
promotion-capable confirmation set must be independently authored, remain
unexposed while margins/policies are selected, meet the evaluation contract's
language/cohort floors, and then receive the same review/adoption protocol.
Body packaging and assistant usefulness are `NOT_OBSERVED` in this replay and
remain later Phase 12/14 work.
