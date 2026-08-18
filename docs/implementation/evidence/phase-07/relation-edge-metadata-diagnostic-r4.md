# Phase 07 Relation-Edge Metadata Diagnostic — Revision 4

- Phase: `07-lexical-evaluation`
- State: accepted calibration diagnostic; not confirmation or promotion evidence
- Date: 2026-08-18
- Implementation commits:
  - `a5efa7832148a6ddffeb600a0ba7b22089ead9a6` — metadata-v2 sidecar and frozen selectors
  - `c197cdafa93852df2c1463d2636378caae288130` — non-null empty TypeScript metadata arrays
- Clean executable commit: `c197cdafa93852df2c1463d2636378caae288130`
- Independent implementation and artifact review: Terra `CLEAR`

## Decision boundary

The owner rejected relation-text embedding. This diagnostic therefore adds
only mechanically derived AST/compiler metadata to the isolated development
relation sidecar. It does not change production indexing, FTS, vector search,
RRF, MCP, questions, labels, document vectors, or the protected dense top
five.

Two policies were fixed before their results were inspected:

1. `query-edge-metadata-dense-first-v1` starts from the frozen
   `dense_1024_int8` top 20 and orders exact one-hop relation facts by query
   features and frozen edge metadata.
2. `query-edge-metadata-graph-first-dense-crossover-v1` was conditional. It
   ran only because X08 remained a relation-admission loss after the first
   arm. It starts from the parent-deduplicated union of frozen FTS and
   simple-control top five, admits the best metadata-prefix tier, and only
   then reranks that tier by the existing frozen dense ordinal.

Neither policy uses RRF, a fresh query vector, a relation embedding, or a
provider request. Both reuse immutable ranks already present in the frozen
calibration replay.

## Stored metadata

The v2 sidecar keeps the existing exact `CALLS`, `TYPE_REF`, and `MEMBER_OF`
facts. Each occurrence additionally records:

- syntactic zone: signature, body, type body, or initializer;
- occurrence role: free/method/value call, parameter/return/field/alias/
  heritage/argument/local type use, or member declaration/receiver;
- flow role: return, assignment, condition, argument, declaration, or none;
- execution mode: direct, deferred, concurrent, or awaited;
- control context: branch, loop, switch, try/catch, or none;
- occurrence file role: production, test, example, or benchmark;
- normalized nearby identifier tokens; and
- deterministic ordinal inside the enclosing semantic parent.

`parent_traits` separately freezes each semantic parent's file role and
mechanically detected Go `Deprecated:` or TypeScript `@deprecated` status.
Selection reads these stored traits but does not read source bodies. Complete
source bodies are opened only after the relation selection has frozen and are
subject to the existing two-parent, 1,024-byte-per-parent packaging cap.

The canonical policy fingerprint covering both selection tuples, keyword
features, caps, trait rules, enums, and packaging rule is:

```text
b984d5eb21241760435da01b45a74ebffd79479175dd325ed17c3941f2a8e133
```

## Real build chronology

The first clean implementation commit built the chi graph, then exposed a real
RHF wire defect before graph publication: candidates without nearby context
identifiers encoded `null`, while the pinned TypeScript helper requires an
array. The process published no RHF graph or evaluation artifact. The fix
makes empty context encode as `[]` and rejects nil at the Go validation
boundary. Terra re-reviewed that fix as `CLEAR`. Both final graphs were then
rebuilt from the later clean commit.

No corpus source, question, label, index, or document/query vector changed
during this correction.

## Frozen inputs

| Input | chi | React Hook Form |
| --- | --- | --- |
| Replay SHA-256 | `90efcc02c9c4e826515ad56d5d3a96104782840503e110e3835b24880cd50bb5` | `5909878346400b307d1f97baea1f5ce939b0ae10ad722d4ef8799e8f57b67bd4` |
| Dataset SHA-256 | `34d95e76d57d88be57cdf23f341c10724dd42fcfe213786b8620595a0ae9c1e1` | `e5c93b9e7823e155b0c31e7b2994ba1ccf96880fcad5e680bc7a46adbcbd8ecf` |
| Dataset fingerprint | `c89ff2760445205937ec2a556d29d8b5a177ef371468ed8616b6221550f620d2` | `558f6b84185ba6dbea55dac975284311510e73f563bf42512737d005f79e0cda` |
| Query-feature SHA-256 | `b8ff6cacd50139344601917238df7a696233f94be8c80e8dced79f09c134cc64` | `43035ca36583c335a494ecfd14714056d34e9716370b261f072018a46db0b305` |

The tracked exact relation-probe input SHA-256 is
`df802d1109fe177846d76fb3ee3a012a53ccb07c21275fd4d81cbb4ef3e7c133`.
All declared forward and reverse probe cardinalities passed.

## Graph results

| Corpus | Parents / traits | Occurrences | Files | Resolved `CALLS` / `TYPE_REF` / `MEMBER_OF` | Logical graph SHA-256 | Database SHA-256 |
| --- | ---: | ---: | ---: | ---: | --- | --- |
| chi | 452 / 452 | 6,737 | 78 | 1,213 / 419 / 108 | `21f8fa8613ed6794d50d76702939150828d17cc9e1f20e93d0931d6bda6eab29` | `dc2a999547f362e671110935e6de00bb08790e30fc4f85fcbb5965f15f3c1b93` |
| React Hook Form | 322 / 322 | 26,011 | 237 | 389 / 2,878 / 0 | `40b85a5872d3c9a36a8ae73a22c689c915dee75a8ba5564d0cbe15b212a66ee8` | `0aca67eb6307ad89fa94ad05beda04284bdb72850a80f1e84b803b966ce74e1c` |

The zero resolved TypeScript `MEMBER_OF` facts are reported rather than hidden.
The useful X08 relation is an exact compiler-resolved `TYPE_REF`; this
diagnostic does not depend on membership recovery for its conclusion.

The metadata tables increased the chi relation DB from about 4.4 MiB to 5.1
MiB and the RHF DB from about 14 MiB to 15 MiB. These are ignored development
sidecars, not production-memory indexes.

## Diagnostic results

The baseline column is the frozen 1,024-dimensional int8 dense top five. The
augmented column preserves that top five exactly and evaluates the complete
related parents appended by the selected relation bundle.

| Arm | Corpus | Baseline complete | Augmented complete | Remaining first loss | Gate |
| --- | --- | ---: | ---: | --- | --- |
| metadata dense-first | chi | 11/12 | **12/12** | none | eligible |
| metadata dense-first | RHF | 19/20 | **19/20** | X08 `RELATION_ADMISSION` | eligible |
| graph-first crossover | chi | 11/12 | **12/12** | none | **ineligible: `walkXFF` attached to G05** |
| graph-first crossover | RHF | 19/20 | **19/20** | X08 `RELATION_ADMISSION` | eligible |

Across both corpora, metadata dense-first improves complete answer evidence
from `30/32` to `31/32`. The conditional graph-first crossover also ends at
`31/32`; it changes 12 selected bundles but adds no complete answer and creates
one explicit chi safety-gate failure.

### G09: metadata is useful

For `chi-g09-client-ip`, metadata dense-first selects the verified
`RealIP -> realIP` call. The decisive occurrence is a production initializer
inside a branch with assignment flow and the nearby identifiers
`fn/ip/real/rip`. Both complete parents fit the body budget. They are both
required, both grade 2, and change the case from incomplete at five to complete
after the two related parents are attached. No `walkXFF` or hard negative is
attached.

This is positive evidence that syntactic edge metadata can distinguish a
useful relation after dense localization.

### X08: metadata is not a sufficient relevance selector

The graph contains the exact required
`FormState -> FormStateProps` `TYPE_REF` at the pinned source span. It is
reachable in both policies. The selected fact instead connects
`UseFormStateProps` to the unrelated `FormState` type. Both selected endpoints
are grade 0.

For graph-first, the wrong fact has metadata prefix
`[0,-4,-2,-6,0,0,0]`; the exact required fact has
`[0,-3,-6,-4,0,0,0]`. The frozen lexicographic contract prioritizes nearby
context-token overlap before endpoint-name overlap, so the required fact is
excluded from the best metadata tier before dense ordinal reranking. This is
an observed admission-policy limitation, not a parser, compiler-resolution,
graph-reachability, or embedding-coverage failure.

The exposed 32 cases must not now be used to swap those key priorities. Any
new selector needs a newly versioned calibration unit and later unexposed
confirmation evidence.

### Graph-first crossover is rejected

Graph-first does not repair X08. It also selects
`middleware.walkXFF` for `chi-g05-middleware-order`; the complete 834-byte
parent is attached even though it is grade 0 and explicitly protected by the
walkXFF gate. Therefore graph-first is not merely neutral: it is ineligible
under the predeclared safety rule.

The result rejects graph-first-before-dense as a production direction under
this fixed policy. It does not reject graph navigation as an evidence assembly
primitive after a stronger query-conditioned admission decision.

## Immutable artifact references

Ignored local references:

```text
.cidx/test/states/chi-1024-int8/evaluations/relation-graph-chi-metadata-v2-c197cda
.cidx/test/states/react-hook-form-1024-int8/evaluations/relation-graph-react-hook-form-metadata-v2-c197cda
.cidx/test/states/chi-1024-int8/evaluations/relation-diagnostic-chi-metadata-dense-first-v1-c197cda
.cidx/test/states/react-hook-form-1024-int8/evaluations/relation-diagnostic-react-hook-form-metadata-dense-first-v1-c197cda
.cidx/test/states/chi-1024-int8/evaluations/relation-diagnostic-chi-metadata-graph-first-v1-c197cda
.cidx/test/states/react-hook-form-1024-int8/evaluations/relation-diagnostic-react-hook-form-metadata-graph-first-v1-c197cda
```

Canonical artifact entry-list checksums:

| Artifact | Checksum |
| --- | --- |
| chi graph | `864bc9d78ee835a8769b1755d0a03a19ad2f08670160df534c43d091901492a2` |
| RHF graph | `74c7398f3124276cb4695b2591a090e5e06e38803ad16ee829f8387d72951153` |
| chi metadata dense-first | `727755aa28b8469aafaf83983d5f6e58a19a587ddf75790ad82e708a4c4a6438` |
| RHF metadata dense-first | `495a8890da2852004f425531d3a56a6a2f359af3227bdb1e451e63ef91e125a9` |
| chi graph-first crossover | `8f0e39b6244b96b2b29228636b1d856d95cbc5ea6d204ba8a31004fb8862d9ce` |
| RHF graph-first crossover | `14cbd5eb547dad511ed95db11476edd0a72993393b292ad15afb76ccbd5eced9` |

Every listed entry digest and canonical entry-list checksum was independently
recomputed. No artifact contains an absolute path, raw query vector, relation
embedding, credential, or provider operation. Every run manifest records
`zero_provider_operations=true`, the clean evaluator commit, frozen input
digests, the selected policy, the shared policy fingerprint, and the exact
primary top-five proof.

## Validation and review

Before the implementation commit, the main agent ran the single code boundary:

```text
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 -race ./internal/relationdiag ./internal/devlab
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 ./...
env -u VOYAGE_API_KEY GOPROXY=off go vet ./internal/relationdiag ./internal/devlab
env -u VOYAGE_API_KEY GOPROXY=off go build ./cmd/cidx
node --check tools/relationdiag/typescript-resolver.mjs
jq -e . testdata/retrieval/relation-probes-chi-rhf-v1.json
go mod tidy -diff
gofmt -l internal/relationdiag internal/devlab/relations.go
git diff --check
```

All checks passed. The real RHF wire correction then received focused tests,
build, Node syntax, formatting, diff checks, and a separate Terra `CLEAR`.
Terra's final read-only audit of all six clean artifacts also returned
`CLEAR`.

## Accepted conclusion

The experiment closes the owner's requested bounded sequence:

1. strengthen graph metadata without relation embeddings;
2. measure dense-first use of that metadata;
3. because one admission loss remained, measure graph-first crossover; and
4. inspect whether either method better identifies the actual answer evidence.

Metadata is retained in the development sidecar because it recovered G09 and
provides useful first-loss evidence. Neither selector is authorized for
production. Graph-first crossover is rejected. The remaining X08 gap is a
query-conditioned evidence-admission problem, and the next valid evaluation
step is a new calibration unit followed by the already-required unexposed
confirmation set—not tuning this exposed set and not adding relation
embeddings.
