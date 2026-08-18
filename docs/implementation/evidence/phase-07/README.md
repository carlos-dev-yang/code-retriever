# Phase 07 Lexical Evaluation Evidence

- Phase: `07-lexical-evaluation`
- State: `in_progress` — the default `1024/int8` and compact `512/int8`
  product boundary is reconciled; the 32-case chi/RHF calibration set is
  frozen, adopted, and replayed; a separate unexposed promotion-capable
  confirmation set remains outstanding.
- Date: 2026-08-18

Current product authority: [Retired Vector Profiles and Evidence Boundary](../../RETIRED-VECTOR-PROFILES.md).

Current frozen calibration checkpoint: [dual-AI calibration freeze and
provider-free replay — Revision 4](dual-ai-calibration-freeze-r4.md).

Current relation-graph conclusion: [compiler-resolved relation/usage graph
diagnostic — Revision 4](relation-usage-graph-diagnostic-r4.md). Exact relation
recovery passed, but the fixed label-blind selector preserved `30/32` rather
than closing G09/X08, so no production graph integration is authorized.

Current relation-metadata conclusion: [AST/compiler edge-metadata and
conditional graph-first diagnostic — Revision 4](relation-edge-metadata-diagnostic-r4.md).
Metadata dense-first recovers G09 and reaches `31/32`, but X08 remains a
relation-admission loss. The predeclared graph-first crossover adds no complete
answer and fails the chi `walkXFF` safety gate. Metadata remains
development-sidecar evidence; neither selector is authorized for production.

### 2026-08-17 solo-project relevance-authority revision

The owner explicitly replaced the unavailable human-pass gate after separate
side-panel review by ChatGPT and Grok. Both reviewers accepted the same strict
solo-project contract:

```text
protocol_version     = owner-adopted-dual-ai-v1
relevance_authority  = OWNER_ADOPTED_DUAL_AI_REVIEW
review_validation    = NO_INDEPENDENT_HUMAN_REVIEW
```

The owner is the governance authority, not a hidden third relevance reviewer.
ChatGPT and Grok independently inspect separately shuffled, source-complete
packets with arm, rank, score, prior labels, experiment results, the other
review, and owner preference hidden. Every relation needs source attestation;
grade-2/group conflicts must be reconciled; grade 1 requires dual agreement;
and no-answer/hard-negative labels require corpus-wide evidence plus both
reviews. The owner adopts or rejects the reconciled digest as a whole. Any
relation-level override reopens both reviews.

This authority may support internal calibration, confirmation, and later
`core_retrieval` evidence when every other frozen gate passes. It must never be
called `HUMAN_REVIEWED`, and every derived artifact carries
`NO_INDEPENDENT_HUMAN_REVIEW`. A label correction after confirmation exposure
permits provider-free diagnostic rescoring but cannot restore that confirmation
set's promotion authority; a new unexposed confirmation unit is required.

All older sections below remain accurate historical provenance. Their
statements that AI work was advisory or that human passes were pending describe
the authority in force at that checkpoint; they do not override this current
contract. The old 191-chi/281-RHF packets also predate the current profile/pool
boundary and cannot be adopted directly. Fresh current pools and both passes
were subsequently completed in the frozen calibration checkpoint linked
above.

The live evaluation wire now enforces this boundary. A frozen case requires
the exact protocol, authority, and validation constants; two distinct reviewer
identities; one SHA-256 per review artifact; and an owner-adoption artifact
SHA-256. Draft cases cannot claim frozen authority. Official run manifests may
carry the same authority tuple, while promotion contracts and results require
it; `CorePromotionEvidence` rejects disagreement among the confirmation
manifest, contract, and result. The former same-person
`solo_review_limitation` escape hatch is removed.

One focused contract boundary passed without a credential, corpus operation,
provider request, or network access:

```text
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 ./internal/evalcontract ./internal/eval ./internal/devlab
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 -race ./internal/evalcontract ./internal/eval
env -u VOYAGE_API_KEY GOPROXY=off go vet ./internal/evalcontract ./internal/eval ./internal/devlab
env -u VOYAGE_API_KEY GOPROXY=off go build ./...
go mod tidy -diff
gofmt -l internal/evalcontract internal/eval
jq -e . schemas/evaluation/*.json
git diff --check
```

All commands passed. That earlier checkpoint proved authority framing and
propagation only. The later frozen checkpoint supplies the missing current
packets, completed passes, reconciliation, owner adoption, and replay proof.

The dated sections below describe the measurements and decisions as they were
made. References to an active Binary profile or a 256 candidate are historical
provenance, not a current runtime or evaluation option.

Current real-data audit: [chi/RHF structural audit — Revision 4](chi-rhf-structural-audit-r4.md).

Current cohort-authoring worksheet: [chi/RHF behavior-cohort working set — Revision 4](chi-rhf-working-cohort-r4.md).

Current document-capture gate: [chi/RHF document-capture approval packet — Revision 4](chi-rhf-document-capture-approval-r4.md).

Consumed query-evaluation gate: [chi/RHF query-evaluation approval packet — Revision 4](chi-rhf-query-evaluation-approval-r4.md).

Current exploratory diagnosis: [chi/RHF exploratory query results — Revision 4](chi-rhf-exploratory-query-results-r4.md).

Current measured cohort decision: [chi/RHF cohort score review — Revision 4](cohort-score-review-r4.md).

Current measured retrieval loop: [provider-free FTS decisions and provenance-safe Voyage comparator — Revision 4](measured-retrieval-loop-r4.md).

Historical isolated codec diagnostic: [paired f32/binary/int8 top-20 diagnostic — Revision 4](codec-top20-diagnostic-r4.md).

Historical 512-dimensional follow-up: [512-dimensional int8 diagnostic — Revision 4](codec-int8-512-diagnostic-r4.md).

Historical 256-dimensional follow-up: [256-dimensional int8 diagnostic — Revision 4](codec-int8-256-diagnostic-r4.md).

Historical five-profile conclusion and current decision input: [five-profile cohort and answer comparison — Revision 4](five-profile-cohort-comparison-r4.md).

### 2026-08-17 five-profile consolidation

The immutable 1,024-, 512-, and 256-dimensional rankings were replayed without
a Voyage call and without FTS, RRF, or codec-rank fusion. Across all 32 draft
questions, every profile retained every direct answer by top 20. `512-int8`
had the strongest shallow result: 31/32 direct and complete answers by top 5,
all 32 questions with source-reviewed useful code by top 5, and `.9969`
same-run f32 top-20 retention. Its two complete SQLite files are 13.53% larger
than the fresh provider-free 1,024-binary controls. `256-int8` is only 5.44%
larger than binary but falls to 28/32 direct answers by top 5 and places RHF
T12's direct implementation at rank 14. This measurement established 512/int8
as a useful compact target; the later owner decision selected 1024/int8 as the
ordinary default and retained 512/int8 as the provider-free compact option.
Its recorded Binary and 256 comparisons remain document evidence only and are
not executable product arms.

### 2026-08-17 256-dimensional int8 follow-up

The final compact-dimension series completed with 32 successful Voyage query
operations, 646 tokens, zero retry/failure, zero document operations, no FTS,
and no RRF. Against same-run exhaustive 256-f32, int8 retained `.9917` of chi
and `.9950` of RHF top-20 membership, with zero top-1 mismatch and every direct
answer retained. Provider-free 256-int8 materialization halved vector payload
again relative to 512, but reduced the two complete SQLite files by only a
combined `7.12%`. Source inspection found more shallow-rank variability,
especially RHF T12, while all direct answers remained present by top 20.
Because the fresh query-vector hashes differ between checkpoints, this is not
a causal same-query dimension comparison. The result keeps 512/int8 preferred
and 256/int8 as a viable memory-constrained alternative; neither is production
selection or promotion evidence.

### 2026-08-17 512-dimensional int8 follow-up

The clean-provenance 512-dimension series completed over the same chi/RHF raw
document banks and questions. It used 32 successful Voyage query operations,
646 tokens, zero retry/failure, zero document operations, no FTS, and no RRF.
Against same-run exhaustive 512-f32, int8 retained `1.0000` of chi and `.9950`
of RHF top-20 membership. A complete source-backed advisory review of all 640
512-int8 relations found comparable neighborhoods to the earlier 1,024 run.
Actual provider-free SQLite materialization halved int8 vector payload and
reduced the two complete databases by a combined `26.48%`. The result advances
512/int8 as the compact candidate but is not label freeze or promotion proof.

### 2026-08-17 paired codec diagnostic entry gate

The user opened one additional measured diagnostic before human label freeze:
isolated `target_f32`, active `binary`, and candidate `int8` dense rankings at
depths 1/5/10/20 for the same chi/RHF cases. Production remains
`serving_dimensions=1024, storage_codec=binary`. Candidate int8 document
vectors are derived locally once from the existing raw f32 bank, and each case
must make exactly one fresh Voyage query-embedding operation whose ephemeral
f32 result is reused by all three arms. The comparison performs the shared
segment-to-semantic-parent collapse but no FTS or RRF. Artifact review will
keep human relevance, complete requirements, hard negatives, top-20 parent
inspection, and f32 codec fidelity separate. No success or metric result is
claimed at this entry boundary.

Implementation now provides a dedicated `cidx dev retrieval evaluate --mode
codec` path. It opens one vector-only production snapshot for the whole corpus
run, loads/transforms the compatible raw f32 document bank once, locally
encodes the candidate int8 bank once, and performs exactly one accounted
Voyage query operation per case. Each successful query prepares one binary
bit representation and one int8 quantized representation independently; the
binary and int8 scorers share neither encoded values nor score arithmetic.
Only the common target-f32 input, active snapshot identity, and established
segment-to-parent collapse are shared. The dedicated artifact contains
top-20 parent and segment rankings, per-query metrics at 1/5/10/20, language
and cohort summaries, separate binary/int8 fidelity-to-f32, and provider usage.
It contains no FTS, RRF, query vector, raw document vector, source body,
credential, or absolute checkout path. Existing production and historical
retrieval paths remain available unchanged. The real chi/RHF apply and source
accuracy review were pending at that implementation checkpoint.

The implementation boundary passed one consolidated validation after the
vector-only snapshot and dedicated scorers were complete: full `go test
-count=1 ./...`; focused race runs for vector/search/eval/devlab/store; full
`go vet ./...`; `go build ./...`; `go mod tidy -diff`; formatting; and diff
checks. No new test code, provider call, corpus mutation, profile activation,
or production-vector write was performed during implementation validation.

The first real chi apply stopped after its first successful query embedding and
before artifact publication because top-20 exposed two distinct chunk ranges
with the same portable `path + indexed hash + qualified symbol` parent key.
RHF was not started. The failed process persisted no query vector or partial
artifact; that one extra provider operation is recorded here with provider
token usage unavailable. The dedicated path now ranks every collapsed chunk,
keeps the best-scoring chunk for each portable semantic-parent identity, and
only then fills 20 unique parents. Segment rankings continue to preserve the
actual distinct segment/chunk observations. Focused search/eval/devlab tests,
build, formatting, and diff checks passed after the correction.

The corrected clean executable at commit
`11f3046a16c73a618a4d9847295a942f99db8868` then completed both corpora.
The successful series made 32 query operations, validated all 32 responses
without retry or failure, observed 646 tokens, and accounted USD `0.00007752`.
It made no document provider call and persisted no query vector. The immutable
artifacts show that int8 retained `.9958` of chi and `.9925` of RHF f32 top-20
membership with zero top-1 mismatch; binary retained `.7042` and `.7575`, with
top-1 mismatch `.25` and `.15`.

A rank/score/codec-hidden source review covered every relation in the pooled
top-20 union: 311 chi and 499 RHF question–chunk relations. This single-root
review is advisory, not a human label freeze. It confirms that binary's
occasional Hit@1/Hit@5 improvements coexist with materially lower useful
top-20 recall and pooled NDCG, while int8 tracks f32 almost exactly. Exact run
IDs, checksums, direct metrics, source-review metrics, and the decision to keep
all codec rankings isolated are recorded in the linked codec diagnostic.

### 2026-08-17 historical pre-run FTS decision boundary

Repeated AND/OR, OR/minimum-two-token, and OR 5:1/5:5 experiments are now
closed. Safe OR 5:1 materially opens the natural-language pool but saturates
candidate depth; minimum-two-token admission provides no required-group gain,
and 5:5 exchanges Go/TSX correctness for a few TypeScript gains. Neither
candidate advances. The linked evidence records the complete paired metrics,
rollback reasons, metric-advisor guidance, and independent final-direction
review.

At this boundary, the planned run used one explicit development-only OR policy
whose fingerprint covered every lexical control. The new manifest bound clean executable/code,
corpus/query/document-bank/profile/materialization/retry/retrieval identities,
zero document calls or historical rank reuse, the USD cap and pricing identity,
actual usage/cost, and all planned/observed stages. Public Search, CLI, and MCP
remained unchanged. The implementation checkpoint itself made no provider call;
the completed run is recorded immediately below.

### 2026-08-17 completed Voyage run and AI-advisory replay

The clean comparator at commit `70bbf1c3b67aa79eaaff4fba495ddbc4e805b6df`
ran 12 chi and 20 RHF queries once with `voyage-code-4`, source/serving 1,024,
and binary storage. All 32 responses validated with zero retry or failure, 646
provider-reported tokens, USD `0.00007752` accounted cost, zero document
provider operations, and no query-vector persistence. Exact run IDs, artifact
hashes, stage metrics, language/cohort exchanges, and the rejected fusion
decision are in the linked measured-loop evidence.

The refreshed rank/score/arm-hidden pools contain 191 chi and 281 RHF
relations. ChatGPT and Grok independently covered both pools completely. Chi
has one complete reconciled AI-advisory label map; all existing direct truth
parents remain direct. RHF has a complete reconciled direct map matching all
22 existing direct parents and two unreconciled grade-0/grade-1 support maps.
Its NDCG is therefore reported as a two-endpoint label-sensitivity range at
query, language, cohort, and global levels. No midpoint or single RHF
full-label digest is emitted.

These outputs are `AI_ADVISORY_CALIBRATION_REPLAY` only. User delegation
authorized review execution and sanitized public-data exchange, not human
label adoption. Formal freeze, calibration selection, confirmation, promotion,
and release authority remain false. No additional Voyage call is required if
only human labels change.

### 2026-08-16 measured cohort review and chi draft-v3 boundary

The immutable exploratory ranks and clean simple-control runs were regrouped
without another provider request. The accepted RHF T10 truth correction makes
its recorded f32/binary ranks 2 and 3 complete, leaving four binary failures:
chi G07 and RHF T01/X01/X08. All four remain because they separately expose a
real multi-parent, orchestrator, wrapper, or thin-type retrieval boundary; no
new question is added. Chi G12 alone receives source-faithful draft-v3 wording
that removes the unsupported implication of global ordering across Go map
keys. The old v2 G12 ranking and pool remain historical and must not be claimed
as a v3 measurement. Exact cohort tables, failure decisions, dataset hash, and
next refresh boundary are recorded in the linked review.

### 2026-08-16 provider-free simple-control implementation and measurement checkpoint

The accepted evaluation-only simple control is implemented and has run against
both draft-v2 corpus datasets. `ProductionStore.SemanticParentsSnapshot`
copies the active generation, manifest, and the authoritative semantic-parent
stored fields (path, indexed hash, language, kind, symbol, qualified symbol,
signature, exact source body, and ranges) in one read transaction, closes that
transaction, and only then permits local control evaluation. The control uses
the stable-deduplicated union of `symbol.ClassifyQuery` identifier tokens then
text tokens under the resolved query limits. A parent is admitted when any token is
present in the normalized path/symbol/qualified-symbol/signature/body union.
It orders admitted parents by exact normalized qualified symbol, exact
normalized symbol, path-token match, distinct matched-token count, then
normalized path, ranges, raw qualified symbol, indexed content hash, then raw
repository-relative path in bytewise UTF-8/Go-string ascending order. Its
versioned fingerprint seals those exact fields, admission, rank,
tie, and the authoritative `config.SymbolNormalizerID` normalizer policy.

`cidx dev retrieval evaluate --mode simple` is provider-free, requires a
manifest and draft dataset, and rejects both `--apply` and `--inventory-only`.
It reproves the all-file `IndexSnapshot` generation, manifest, production
schema, and index profile against the semantic-parent snapshot and resolved
config before corpus-file verification, so parentless files cannot be omitted.
It writes only a separate immutable ignored simple-control artifact with
complete finite per-case metric maps and mutually consistent corpus/label and
algorithm-bound candidate-policy controls. Every artifact case must retain
exactly `min(admitted_candidate_count, candidate_k)` ranked hits, and every
metric depth is bounded by the captured `return_k`. It emits no FTS `StageTrace` or FTS
first-loss semantics. It does not change public MCP/schema, production FTS,
BM25, aliases, weights, embeddings, or corpus state.

Both runs used a binary built from clean commit
`d343e12c36c2d17e40c00fe2fab445299f151715`, reported
`source_modified=false`, and ran with `VOYAGE_API_KEY` removed. Chi run
`chi-simple-v2-d343e12-1` recorded Hit@1 `2/12`, Hit@5 `5/12`, simple-run
SHA-256 `ab629635f94be0a90f683181899e983c3ec97a4e61222117eda7faa1dc10bc83`.
RHF run `rhf-simple-v2-d343e12-1` recorded Hit@1 `7/20`, Hit@5 `14/20`,
simple-run SHA-256
`cf01829cfe62308aa5a1546eaebb5050e1f3211d52f8f6a4ce40829c2d0b75a8`.
These values diagnose lexical/name-path separability and are not pass/fail
thresholds.

### 2026-08-16 accepted draft-v2 label boundary

The versioned `behavior-*-draft-v2.json` datasets keep the same 12 chi and 20
RHF questions, texts, languages, and cohorts. Chi G09 adds only the accepted
`middleware.walkXFF` grade-0 hard negative; RHF T10 makes `module.PathImpl` and
`module.PathInternal` grade-2 requirements and public `module.Path` grade-1
support. The affected RFC 8785-framed case digests are
`a969ea05c99b2ed5ba1842006db66e7f60acb7af1294ca7d12095fc33ee6674d`
and `6cb7985bc4b3c56b207492e123b45551e0b14bc0d2bc08e86eaa45dab026aed1`.
All 32 case digests reproduced; focused devlab/eval/evalcontract tests, vet,
build, JSON validation, and diff checks passed. An independent Terra review
reported no findings and confirmed v1 preservation plus the absence of a
human-pass, freeze, or promotion overclaim. These draft files still require the
provider-free simple pool and two separated human review passes.

The accepted Phase 04 correction is now reflected in provider-free
generation-3 indexes: chi has 452 parents/621 segments and RHF has 322
parents/492 segments; 57 RHF production anonymous default-export functions now
have deterministic path-derived retrieval labels, the three observed overload
sets each collapse to one parent, and source-span mismatches are zero. New
source-body-free inventory hashes and the current no-network document plan are
recorded in the structural audit. The side-panel-reviewed 32-case cohort is now
bound to exact generation-3 identities and deterministic case digests. The
official live Voyage pages confirm `voyage-code-4`, its supported dimensions,
$0.12 per million-token price, and 200-million-token free allowance. The user
approved the exact document-only capture under a $5 account billing limit on
2026-08-16. A clean post-approval preflight reproduced the exact plan. The
capture then persisted 619/619 chi and 492/492 RHF raw 1024-f32 vectors with
zero failures and 331,513 provider-reported tokens. Local materialization
published complete 1024/binary coverage for 621/621 chi and 492/492 RHF
segments. No query embedding occurred.

### 2026-08-16 explicit exploratory-query authorization

After the project-local source/state migration was accepted at `e06e28a`, the
user instructed Codex to goal the remaining work and continue. The instruction
authorizes only the existing two-invocation packet: 12 chi plus 20 RHF draft
exploratory queries under a combined $0.01 ceiling. Provider-free preflight
reproduced both original plan JSON digests byte-for-byte, so no corpus,
dataset, generation, manifest, raw count, profile, token ceiling, or operation
count changed. The next boundary is a clean-provenance repeat followed by that
single bounded series; no repeat, formal calibration, confirmation, mixed
corpus, promotion, or assistant operation is implied.

### 2026-08-16 document and query-plan boundary

Read-only lab and production checks proved one `voyage-code-4` response model,
1024 dimensions/4,096 raw bytes, zero capture failures, published
materialization runs, one binary codec, 128 stored bytes per vector, and no
missing segment vector. Post-capture plans have 1,111 raw hits and zero paid
misses. The provider-free retrieval preflight then fixed 12 chi plus 20 RHF
query operations with a 3,698-token conservative ceiling. The separate query
packet now records the later explicit one-series authorization under its $0.01
ceiling; credentials and document approval alone did not authorize it.

### 2026-08-16 behavior-binding boundary checks

The two behavior datasets contain 32 cases with 32 matching RFC 8785-framed
case digests. A read-only SQL join proved that every unique direct/support span
resolves to exactly one generation-3 production parent. The focused boundary
validation then passed:

```text
gofmt -w internal/devlab/lexical.go internal/devlab/lexical_test.go
go test -count=1 ./internal/devlab ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store
go test -count=1 -race ./internal/devlab ./internal/eval ./internal/evalcontract
go vet ./internal/devlab ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store
go build ./internal/devlab ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store
jq -e . testdata/retrieval/behavior-*-draft-v1.json
go mod tidy -diff
git diff --check
```

No repository-wide test, corpus mutation after generation 3, provider call,
API-key read, query embedding, materialization, label freeze, or promotion run
occurred.

### 2026-08-16 clean provider-free behavior baseline

The clean build at commit `2a08df7d465f72c939a3b0e85b28d8b5ca000cdf`
ran the production FTS path once against each new behavior dataset. Both runs
completed without operation failures, but natural-language behavior questions
returned no candidates: chi was 0/12 at Hit@5 and RHF was 0/20 at Hit@5, with
every first loss recorded as `FTS_CANDIDATE_MISS`. This is the lexical baseline,
not a dense or hybrid result and not a reason to tune an unfrozen simple-search
policy before the separately approved document/query embedding stages.

| Run | Cases | Mean returned | Hit@5 | `run.json` SHA-256 |
| --- | ---: | ---: | ---: | --- |
| `chi-behavior-fts-g3-1` | 12 Go | 0 | 0 | `1134ef35cdcc0cbd7c415feb7c2ab6f849e76dc15cd224b133f1554dfe1a928d` |
| `rhf-behavior-fts-g3-1` | 12 TypeScript + 8 TSX | 0 | 0 | `31ba8d90986b838fe6a95e9f514795c57b4d11bc04c5e56ca2fb8623f8acac84` |

The ignored immutable artifacts remain under each checkout's
`.cidx/lab/evaluations/runs/` directory. No provider, API key, query embedding,
or label freeze was involved.

## Implemented infrastructure

- `internal/eval` owns strict portable corpus-manifest and evaluation-dataset JSON loading, RFC 8785 canonical fingerprints, and validation of corpus provenance, complete commits, SPDX-form license declarations, language slices, repository-relative roots, selection patterns, and deterministic selected-content hashes.
- Local bindings are intentionally separate from portable manifests. The conventional binding loader accepts only the ignored `.cidx/lab/corpora.local.json`; an explicit checkout path takes precedence. Checkout verification is local and read-only: canonical worktree root, origin, exact commit, clean status, root license record, symlink-free selected files, declared language slices, and selected-content hash.
- Metrics reuse `internal/evalcontract.EvaluationCase` truth values. They implement OR alternatives within a requirement group, AND across groups, graded relevance/NDCG without duplicate gain, Hit/Recall/MRR, requirement coverage, complete requirement hit, known-hard-negative observations, returned counts, explicit answerable/required/hard-negative denominators, operation-failure retention, first-loss counts, and Go/TypeScript/TSX/mixed summaries.
- The lexical runner is an adapter over the production lexical search interface; it has no alternate FTS implementation. Its injected production truth inventory uses one narrow read-only store transaction to pin metadata and enumerate authoritative source chunks. It rejects stale/missing required alternatives, relevance judgments, and hard negatives before metrics exist. Successful searches must match that pin; a drift yields `NON_REPRODUCIBLE_RUN`. Required search failures stay in the denominator as typed FTS-candidate failures without serializing raw errors.
- Every ranked case carries a valid shared `StageTrace`: source/parser truth is present after preflight, FTS presence follows results, operation failures terminate at `fts_candidate`, later unrequested stages are explicit `NOT_OBSERVED`, and abstainable cases retain empty groups.
- Immutable artifacts publish `run.json` and `summary.md` through a new temporary directory plus atomic rename, then write `artifact-manifest.json` as the completion marker. They reject existing run IDs, forged summaries, missing/duplicate query results, invalid traces, invalid portable hits/ranks, unsafe portable data, and inconsistent corpus/dataset/generation pins.
- Manifest include/exclude rules use a validated whole-segment `**` matcher, including zero-segment matching for patterns such as `**/*.go`; excludes always take precedence.

## Checks actually run

The focused simple-control implementation boundary passed on 2026-08-16:

```text
gofmt -w internal/store/search_snapshot.go internal/store/search_snapshot_test.go internal/symbol/normalize.go internal/eval/run.go internal/eval/simple.go internal/eval/simple_artifact.go internal/eval/simple_test.go internal/devlab/cli.go internal/devlab/lexical.go internal/devlab/lexical_test.go
go test -count=1 -race ./internal/store ./internal/eval ./internal/devlab ./internal/symbol
go vet ./internal/store ./internal/eval ./internal/devlab ./internal/symbol
go build ./cmd/cidx ./internal/store ./internal/eval ./internal/devlab ./internal/symbol
gofmt -l internal/store/search_snapshot.go internal/store/search_snapshot_test.go internal/symbol/normalize.go internal/eval/run.go internal/eval/simple.go internal/eval/simple_artifact.go internal/eval/simple_test.go internal/devlab/cli.go internal/devlab/lexical.go internal/devlab/lexical_test.go
git diff --check
```

The focused cases cover stable query-token deduplication and query-limit
rejection, `ANY` admission across every permitted normalized field, exact/path/
matched-token/stable-identity ordering including raw-qualified and normalized-path
collision fixtures,
admitted-count versus returned-cap behavior, generation/manifest pin
propagation and all-file/profile/schema parity, sealed exact fingerprint
stability, signature-only admission, separate artifact immutability plus forged
metric-map, contradictory-control, incomplete-pool, and over-depth rejection,
source-body snapshot copying,
and simple-mode rejection of `--apply` and `--inventory-only`. No corpus state, provider,
network, API key, embedding, lab, production FTS, MCP, or public schema action
ran.

The independent Terra/high review initially found missing DB-profile reproof,
contradictory artifact-control acceptance, an under-specified fingerprint,
incomplete captured-pool validation, and missing focused field coverage. The
implementation now reproves schema/index profile, cross-checks every artifact
pin, seals token construction and the total tie order, requires exactly
`min(admitted,candidate_k)` hits at bounded metric depths, and carries the
corresponding fixtures. Final re-review and the main focused boundary commands
above reported no findings.

Codex repeated the focused boundary validation on 2026-08-15 before the
infrastructure commit. All commands below passed.

```text
gofmt -w internal/eval
go test -count=1 ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store
go test -count=1 -race ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store
go vet ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store
go build ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store
gofmt -l internal/eval internal/evalcontract internal/search/lexical internal/store
git diff --check
```

The focused tests cover portable/duplicate-field manifest rejection, local Git binding verification including wrong-content-hash and dirty-worktree rejection, whole-segment glob selection/exclusion safety (including nested `**` and empty excludes), OR/AND multi-span requirements, duplicate-parent max-grade NDCG, wrong indexed-hash rejection, exact hard-negative denominators across answerable and abstainable cases, full language/cohort summaries, retained operation failures and valid abstainable traces, production truth snapshots, inventory preflight, generation drift, caller cancellation, and atomic immutable artifact publication including forged-summary, missing metric, and rank rejection.

## Historical pre-resume checks not run and blocker

- No corpus beyond the two explicitly authorized public checkouts was selected, downloaded, cloned, copied, or embedded. No provider, API key, paid operation, or public MCP contract change ran.
- No official frozen run, human-reviewed labels, simple-search comparison baseline, or promotion evidence exists. The recorded corpus verification and repeatability evidence is draft-smoke-only.
- At this historical checkpoint a deterministic simple-search baseline was intentionally pending. That blocker is superseded by the accepted, fingerprinted control and clean draft-v2 measurements recorded above.
- The direct `internal/eval` source has no `embedclient`, `lab`, or provider import. A full Go dependency listing still reaches `embedclient` indirectly through the existing shared `config`/`evalcontract` package graph; that pre-existing shared-contract coupling must be split before a strict transitive dependency-boundary claim can be made.

## Next action

Keep the exposed 32-case calibration set immutable. Use its accepted
`1024/int8` dense baseline and separate FTS control as Phase 12 calibration
inputs without reopening the rejected RRF weight branch. Independently author
a new unexposed promotion-capable confirmation set, then apply the same
dual-AI/owner-adoption protocol before any confirmation run. Do not call
Voyage again for the closed calibration set while its corpus, questions,
retrieval policies, and embeddings are unchanged.

The ignored source-link review views are
`.cidx/test/states/chi/evaluations/review/pass1-v2-chi-review.md` and
`.cidx/test/states/react-hook-form/evaluations/review/pass1-v2-rhf-review.md`.
They contain no machine grades, lane names, scores, or original ranks.

## Authorized corpus-resume smoke

The user explicitly authorized acquisition of exactly these public checkouts;
no other corpus was selected or acquired:

- `go-chi/chi` v5.3.1 at `8b258c7bb28f97a5f2a856ff7ef962578fec9215`,
  MIT, root tree `7ccb2269b57183ac3a741f269c0da31fd03ad035`;
- `react-hook-form/react-hook-form` v7.85.0 at
  `371432c39271aab739358d19c406793771565ab3`, MIT, root tree
  `688906c5842a0d71051154343e993adb525e688f`.

Tracked portable manifests are
`testdata/retrieval/corpora/go-chi-chi-v5.3.1.json`
(`18cd5cf433ee0af47a212e6111dcd1d65f6104baa28bb528b2ec93d9afec36b9`)
and `testdata/retrieval/corpora/react-hook-form-v7.85.0.json`
(`e94f0861e6ac0c864524a23edc4bcb0ddc69a3848ef0f9c962f0b675bfde81a8`).
Their verifier-selected content hashes are respectively
`892e79de9e8c522fe3ccf6b0731a3798d0f2c67a18f1b4162685c4843245af5d`
for 78 Go files (largest 58,795 bytes) and
`717caa8346fd5a0b1a7ca69df63bf1ac8477f7c8770f1e67fa7b1fad58df132b`
for 237 `src` TypeScript/TSX files (largest 142,806 bytes).

The React Hook Form checkout is full and clean. Its checkout-local ignored
`.cidxignore` excludes the only outside-`src` TS/TSX roots at this commit:
`app`, `e2e`, `examples`, `scripts`, and `playwright.config.ts`; `.cidx/`
and `.cidxignore` are only in that checkout's `.git/info/exclude`. Each
checkout has an ignored local `.cidx/lab/corpora.local.json` self-binding.

Free local initialization/index/status records:

- chi generation 1: 78 files, 452 chunks, 621 segments, manifest
  `6bd4db89ee1a9cba70f69e125a803d147dbc0d92c95ef59b44be2dcb54302a29`;
- react-hook-form generation 1: 237 files, 275 chunks, 416 segments, manifest
  `54f6b1387ae989b1e49bdf21d3ed96189e76fb5b61b74ca282a2617c57f88b8a`.

`cidx dev retrieval evaluate --mode lexical --inventory-only` wrote ignored,
source-body-free inventory packets with SHA-256
`c6a8661e9cde7ff269d69311593411930def0f69a7c1816b9e919bfbce7cadab`
(chi) and
`d65595cfa76480278f4e71734d6a4802b8bb4129944a3384c2fccedb3e80781d`
(react-hook-form). The mode opens `app.OpenLocal` and the production store
only; it neither opens `lab.Store`, reads `VOYAGE_API_KEY`, contacts a
provider/network, nor permits `--apply`.

The tracked datasets
`testdata/retrieval/lexical-go-chi-v5.3.1-draft.json`
(`e03894820a25eecb5527049ece7d10da39da58f719c629fa3e5bc11f47ca22c4`)
and `testdata/retrieval/lexical-react-hook-form-v7.85.0-draft.json`
(`393a7562bffbe5ce3fb018b9438aaf6633c7f5c26ba4418521b74e3c3ae6df80`)
contain six cases each. Every case is explicitly `review.state=draft` with
`machine-draft/user-review-pending`; neither file is frozen, official,
promotion-capable, or usable for tuning. Every draft case digest is the
SHA-256 of RFC 8785 canonical `EvaluationCase` JSON with its `digest` field
empty; `internal/devlab.DraftCaseDigest` verifies this preparation framing
before lexical execution.

Ranking-blind review packets retain corpus/dataset digests, query text,
class, answerability/cardinality, proposed file/symbol/span/hash and
alternatives, ambiguity/rationale, accept/reject/revise/adjudicate actions,
and reviewer/timestamp/independent-source-verification fields. Their packet
digests are `d1fcc6415c6c98a73ebd95b8367fefb804830b75da8b44b45f6f5cc66e1417fc`
(chi) and `e6ab27964f0e90f52e496948723eeb518e88911e73273b23233dfa773a3f6f6a`
(react-hook-form). Both declare `dataset_status=DRAFT`,
`dataset_role=CALIBRATION_SMOKE`,
`label_authority=MACHINE_PREPARED_UNREVIEWED`,
`human_review_status=PENDING`, `run_authority=EXECUTION_ONLY`, and
`evidence_class=PIPELINE_AND_REPLAY_DIAGNOSTIC`; they also declare
`promotion_eligible=false`, `confirmation_eligible=false`,
`retrieval_arm=PROVIDER_FREE_LEXICAL_ONLY`, and `paid_provider_calls=0`.

The reviewed implementation was committed as
`28d0d6a1d93949c2151ca388a8f4b7739c7edc81`, then rebuilt with
`-trimpath -buildvcs=true` from a clean worktree. `cidx version --json` and
`go version -m` both reported that full revision and
`source_modified=false`. Reindex reused all 78 chi and 237 react-hook-form
files without changing either generation or manifest.

The clean binary published immutable ignored smoke pairs
`chi-draft-smoke-28d0d6a-{1,2}` and
`rhf-draft-smoke-28d0d6a-{1,2}` through `eval.WriteRunArtifact`; every run
records the exact implementation revision above. The first `run.json` digest
for each corpus is respectively
`284c9d07c8bd7bb40a7449a6118398db99a58976a370406b5a589daa742fa723`
and `003c13c4d0a7e9dbc46abc096e4783244f8b01b97bd4e9c7177451d17ee2dc30`.
The second-run digests are
`bdd9c8988c6e806cd0f0151bd5fa0309b5a2e58d2668c136a462ff7b0d32daee`
and `61ab6cb04c681df3d62b3d28caeed15a56fc5efa8f32b809e87b26600a2c1aac`;
their clock/run identities differ, while the framed results and summaries are
byte-identical. Earlier dirty-worktree runs are superseded execution
diagnostics and are not clean-provenance evidence.

Lexical artifact execution fails closed unless build metadata has a canonical
full lowercase hex VCS revision (40-character SHA-1 or 64-character SHA-256)
and `source_modified=false`. It also rejects any non-`draft` review state, so
this smoke command cannot stamp frozen labels as draft authority.
Inventory-only preparation remains available. A future official baseline is a
separate follow-up after human review and baseline-policy freeze.

Repeated runs produced identical diagnostic replay values using exactly
`sha256(jq -c '{results,summary}' output including LF)`:
`7b31f9cd6eac758601d99988da5c691ec462ece96b82ceb5efc0eff163242937`
(chi) and
`e869a7e0718d693958d5b34878c0604d2ace6d683cf272ee9d10ab22cabf9c3b`
(react-hook-form). The smoke denominators are 12 total: 6 Go, 4 TypeScript,
2 TSX, 0 hard-negative, and 0 confirmation. Hard-negative and confirmation
metrics are `NOT_OBSERVED`, never reported as zero. Observed metrics mean only
agreement with unreviewed draft labels under a lexical smoke configuration.
The valid conclusion is limited to provider-free lexical-path execution and
identical ranking replay for exact draft inputs; this is not a quality claim or
promotion evidence.

## Resume checks actually run

```text
go build -o /tmp/cidx-phase07 ./cmd/cidx
cidx init --serving-dim 256 --codec binary              # each authorized checkout
cidx index --root <checkout> --reason manual
cidx status --root <checkout> --json
cidx dev retrieval evaluate --mode lexical --inventory-only ...
cidx dev retrieval evaluate --mode lexical --run-id <fresh-id> ... # twice per corpus
gofmt -w internal/devlab/lexical.go internal/devlab/lexical_test.go
go test -count=1 ./internal/devlab ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store
go test -count=1 -race ./internal/devlab ./internal/eval ./internal/evalcontract
go vet ./internal/devlab ./internal/eval ./internal/evalcontract
go build -o /tmp/cidx-phase07-provenance ./cmd/cidx
git diff --check
```

All listed checks passed. No paid document/query embedding, lab database,
provider call, API-key read, promotion, simple-search comparison, or official
evaluation run occurred.

An independent Terra/high review found and rechecked three resume-boundary
defects: dirty or noncanonical executable provenance, unconditional draft
authority for non-draft inputs, and packet child-symlink escape. The accepted
tree fails closed on all three, and the final re-review reported no findings.
Codex then ran the single final commit-boundary validation:

```text
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 ./internal/devlab ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store ./internal/app ./internal/index
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 -race ./internal/devlab ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store
env -u VOYAGE_API_KEY GOPROXY=off go vet ./internal/devlab ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store
env -u VOYAGE_API_KEY GOPROXY=off go build -trimpath -buildvcs=true -o /tmp/cidx-phase07-boundary ./cmd/cidx
go mod tidy -diff
gofmt -l internal/devlab/cli.go internal/devlab/lexical.go internal/devlab/lexical_test.go
jq -e . testdata/retrieval/corpora/*.json testdata/retrieval/*draft.json
git diff --check
```

All boundary checks passed. The accepted implementation is committed before
new smoke artifacts are generated so their `code_commit` is truthful.

The clean post-commit checkpoint then ran:

```text
test -z "$(git status --porcelain)"
env -u VOYAGE_API_KEY GOPROXY=off go build -trimpath -buildvcs=true -o /tmp/cidx-phase07-clean ./cmd/cidx
cidx version --json
go version -m /tmp/cidx-phase07-clean
cidx index --root <checkout> --reason manual             # both checkouts
cidx status --root <checkout> --json                     # both checkouts
cidx dev retrieval evaluate --mode lexical --inventory-only ...
cidx dev retrieval evaluate --mode lexical --run-id <fresh-id> ... # two per corpus
jq -c '{results,summary}' <run.json> | shasum -a 256
cmp <framed-run-1> <framed-run-2>
```

The environment omitted `VOYAGE_API_KEY`; no provider or network operation
was needed. The repeat comparisons and all recorded checksum assertions
passed. The independent Terra/high reviewer then matched all four clean run
artifacts, artifact manifests, status counts, replay hashes, and ledger values
and reported no findings.

`internal/devlab/lexical_test.go` covers the draft-digest framing, lexical
mode/apply and inventory flag rejection, clean canonical code-provenance and
draft-only smoke enforcement, conventional artifact-root validation, and
descriptor-bound packet writes. The latter uses Go 1.26 `os.OpenRoot` with
root-relative directory creation, exclusive temporary creation, hard-link
publication, and reads; both `inventory` and `review` external-child-symlink
attempts are rejected without an outside write. It also covers atomic
source-body-free inventory replay/collision handling and review-packet
decoding for authority/floor fields plus `id`, `text`, `language`, and
`answer_mode`. The direct lexical source has no `lab.Open`, `VOYAGE_API_KEY`,
provider, or HTTP dependency.

## 2026-08-16 authorized exploratory-query checkpoint

The exact approved series ran once from clean implementation commit
`59b1cd61ec990c56cea275f5ac1b258e7eb5332a`. Clean-binary preflight reproduced
the frozen portable plan hashes before the first request. The provider returned
32/32 validated query responses with 32 attempts, zero retries/failures, and
636 provider-reported total tokens (221 chi and 415 RHF). Query vectors remained
ephemeral: the lab schema contains document-vector blobs only, while the usage
artifact stores query IDs, counts, terminal status, and observed tokens.

The immutable ignored run references and entry-list checksums are:

- chi: `evaluations/retrieval-7e5731ed1222a6aa432da84f`,
  `7a538245b3e74f106cdf318e31843a798670e1c9a9ff095bd1025e9ade812967`;
- react-hook-form: `evaluations/retrieval-20417011198b38cad4a1af2b`,
  `8dfd28b7d8e8de082ad9e0a964566758af69aa8e0c08261badf976344637651f`.

All 16 listed files in each artifact passed an independent SHA-256 check, and
the matching `evaluation_runs` rows are `complete` with the same operation and
token totals. These are explicitly non-promotion draft/calibration-preparation
artifacts; no repeat, formal calibration, confirmation, mixed-language, or
assistant operation is authorized by this checkpoint.

Diagnosis also found an internal trace inconsistency for segmented parents:
rankings and metrics used semantic-parent identities correctly, but the stage
trace compared required parent spans with smaller segment byte ranges. A narrow
Phase 12 correction now uses recorded parent coordinates and is documented in
its Revision 4 evidence. Focused normal/race tests, vet, build, formatting,
module, and diff checks passed at this correction boundary.
The immutable exploratory artifacts are preserved as diagnostic input and are
not rewritten. At that draft-v2 checkpoint the three material
T10/G09/simple-policy decisions were accepted and only two separated human
passes remained. The measured cohort review above now supersedes the G12
question boundary: chi v3 needs its provider-free simple/opened-arm refresh
before those passes. No provider call is required merely to inspect the
already-recorded rankings and metrics.

The final opened-arm top-5 results were pooled into ranking-blind ignored
pass-1 packets. Chi contributes 91 unique parents and 133 query-parent
judgments; RHF contributes 78 and 175. All 308 relations remain unreviewed in
the packets, including every draft-v2 truth parent even when it missed every
retrieved top-5 arm. The packets
retain verified source spans and exact parent bodies while omitting arm, score,
and rank. Their paths, digests, coverage, completed simple-search addition, and
unopened int8 status are recorded in the exploratory-results evidence below.
The earlier ignored advisory overlays cover only the 205 four-arm relations
with proposed 0/1/2 grades and required-group assignments. They are explicitly
withheld from the label-free first-pass packet, do not cover the 103
simple-only additions, and cannot be merged automatically.

The complete metrics, stage diagnosis, accepted source-backed label corrections,
accepted simple-baseline policy, and representative-intent cohort rule are recorded in
[`chi-rhf-exploratory-query-results-r4.md`](chi-rhf-exploratory-query-results-r4.md).
