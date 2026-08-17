# Phase 07 Relation/Usage Graph Diagnostic — Revision 4

- State: accepted negative calibration result; not confirmation or promotion evidence
- Date: 2026-08-18
- Implementation commit: `02834052921116a6341c44d7f7fd7e51f6a87005`
- Corpora: `go-chi-chi-v5.3.1`, `react-hook-form-v7.85.0`
- Retrieval lane: frozen exhaustive `1024/int8` top 20; no new query embedding
- Selection policy: `one-hop-kind-order-type-call-member-one-bundle-v2`
- Body policy: `related-complete-parent-2x1024-v1`

## Decision

The compiler-resolved sidecar is technically viable as a relation-fact store,
but the fixed label-blind one-bundle selector did not improve answer
identification. The current diagnostic therefore does **not** authorize a
production graph integration, public search change, MCP change, RRF arm, or
serving-schema change.

The result separates two questions that were previously mixed:

1. Can cidx recover exact code relations? **Yes.** The pinned G09, X08, T09,
   and T10 call/type probes all passed at exact parent and occurrence byte
   ranges, including G09 reverse caller lookup.
2. Can one fixed relation-kind/rank order choose the answer-bearing relation
   for a natural-language query? **No.** It preserved the existing `30/32`
   complete-at-five result without closing either G09 or X08.

This is a useful negative result. The missing layer is query-conditioned
relation admission and evidence-group selection, not another parser, another
lexical engine, or a larger graph. Any later experiment must be separately
designed and must not retune this exposed 32-case calibration set into
confirmation evidence.

## Frozen inputs

| Input | SHA-256 |
| --- | --- |
| chi frozen dataset | `34d95e76d57d88be57cdf23f341c10724dd42fcfe213786b8620595a0ae9c1e1` |
| RHF frozen dataset | `e5c93b9e7823e155b0c31e7b2994ba1ccf96880fcad5e680bc7a46adbcbd8ecf` |
| chi frozen 1024/int8 replay | `90efcc02c9c4e826515ad56d5d3a96104782840503e110e3835b24880cd50bb5` |
| RHF frozen 1024/int8 replay | `5909878346400b307d1f97baea1f5ce939b0ae10ad722d4ef8799e8f57b67bd4` |
| exact relation probes | `df802d1109fe177846d76fb3ee3a012a53ccb07c21275fd4d81cbb4ef3e7c133` |

The replay loader revalidated corpus ID, dataset source SHA-256, canonical
dataset fingerprint, frozen review authority, exact 20-depth query/rank set,
and profile lane before selection. Labels were loaded only after Stage A facts,
Stage B selection, and related-body packaging were frozen.

## Implementation boundary

The implementation adds only development surfaces:

- `internal/relationdiag` for Tree-sitter frontier extraction, compiler-backed
  Go and TypeScript resolution, immutable SQLite publication, and replay;
- `cidx dev relations build` and `cidx dev relations evaluate`;
- a pinned TypeScript `6.0.3` helper/lock and ignored local materialization;
- exact tracked chi/RHF relation probes.

The sidecar stores forward `CALLS`, `TYPE_REF`, and `MEMBER_OF` occurrences and
derives reverse lookup from the same rows. Every occurrence retains an explicit
resolution outcome. It never imports into production search, MCP, production
schemas, vector storage, or provider code.

Publication is generation- and provenance-bound. The builder verifies every
indexed source hash and semantic-parent body, records the cidx executable,
actual Go executable/environment, x/tools, Node, TypeScript helper/lock/config,
and TypeScript runtime payload, then repeats live source/toolchain/index proofs
immediately before atomic publication. Evaluation verifies the artifact
checksum entry set before opening the immutable database.

The fixed Stage B order is only:

```text
TYPE_REF -> CALLS -> MEMBER_OF
anchor dense rank
endpoint dense rank
occurrence byte
stable identities
```

Direct-call versus value-flow-call priority was deliberately not invented:
the current compiler resolver has no authoritative fact for that distinction.
Dense top five stays byte-identical. At most one bundle and two complete
related parents of at most 1,024 bytes each may be added. Selection never sees
labels, query ID outcomes, or required groups.

## Clean graph evidence

The accepted executable reports:

```text
commit          02834052921116a6341c44d7f7fd7e51f6a87005
source_modified false
go_version      go1.26.4
target          darwin/arm64
```

The resolver ran under an independently recorded local Go command
(`GOTOOLCHAIN=local`, `GOFLAGS=`), Node `v24.11.1`, and TypeScript `6.0.3`.
The build executable SHA-256 was
`bbf8fde380a6bf3e18baaec69d82b56dede763db382cf9a533e77e81341dffcc`.

| Corpus | Files | Parents | Occurrences | Resolved unique | Logical graph SHA-256 | Database SHA-256 |
| --- | ---: | ---: | ---: | ---: | --- | --- |
| chi | 78 | 452 | 6,737 | 1,740 | `927bd63204a24e408590ef38070eb67ea6c39a07b69ff5f2c25f8630467070d1` | `6218c338fbe061a83b19f44d01fe6aed6b9fd2907d763e6440226b904fbde364` |
| RHF | 237 | 322 | 26,011 | 3,267 | `ef17e4f24f7ee0cbc5d2791a3a3efc153b1c950b36d18bb4232abd70d79361ac` | `cd0621fb8d90932b4ecf94482b6244d8c93ecefdde3ae0a992e590a91c7e16fd` |

Local ignored graph references:

```text
.cidx/test/states/chi-1024-int8/evaluations/relation-graph-chi-clean-0283405
.cidx/test/states/react-hook-form-1024-int8/evaluations/relation-graph-rhf-clean-0283405
```

Graph manifest/checksum SHA-256 values:

| Corpus | Graph manifest | Artifact checksums |
| --- | --- | --- |
| chi | `831e68f37c52e12be06ac248b30bd71fb25a1ee27e92cab02005c8dac5c4952c` | `628f5b39695298c043da3e665a031c233b406f32600578a98aa7fff88d5ad385` |
| RHF | `d5e325d49e6e2c57ab8695ef3a0fcdd711cafaeb5eb541dc180e1a94bc7dcae1` | `3b54081c58bfe78573be242fc164edb7bdfbeff4d56f57b06cb9ef3355de4390` |

## Exact relation proofs

All tracked probes passed with the declared cardinality:

- G09 `middleware.RealIP --CALLS--> middleware.realIP`, occurrence
  `middleware/realip.go:1111..1120`, one match;
- G09 reverse lookup `middleware.realIP --CALLED_BY--> middleware.RealIP`,
  backed by the same stored occurrence, one match;
- X08 `module.FormState --TYPE_REF--> module.FormStateProps`, occurrence
  `src/formStateSubscribe.tsx:688..702`, one match;
- T09 `module.createFormControl --TYPE_REF--> module.Control`, occurrence
  `src/logic/createFormControl.ts:44054..44061`, one match;
- T10 `module.PathInternal --TYPE_REF--> module.PathImpl`, occurrences
  `src/types/path/eager.ts:1431..1439`, `1510..1518`, and `1585..1593`,
  three matches.

These are exact compiler-resolved facts, not qualified-symbol text matches.
The probe contract binds corpus, source/target path, indexed content hash,
qualified symbol, parent byte ranges, direction, occurrence ranges, and exact
cardinality.

## Retrieval result

| Corpus | Queries | Baseline complete | Augmented complete | First loss for the remaining case |
| --- | ---: | ---: | ---: | --- |
| chi | 12 | 11 | 11 | G09: `RELATION_ADMISSION` |
| RHF | 20 | 19 | 19 | X08: `RELATION_ADMISSION` |
| total | 32 | 30 | 30 | both answer relations were reachable but not selected |

No existing complete case regressed because dense top five was protected.
There were zero attached declared hard negatives and zero `walkXFF`
attachments, so both runs remained diagnostically eligible. Eligibility only
means the fixed experiment was safe to interpret; it does not mean that it
improved retrieval.

### G09

Stage A enumerated 88 one-hop facts from the fixed dense top-20 seeds and did
contain the exact `RealIP -> realIP` answer relation. The selector instead
chose a rank-1 test-parent call to another grade-0 test parent. That added body
was 1,555 bytes and was correctly omitted as `BODY_TOO_LARGE`. No answer body
was added, so complete-at-five stayed false. `walkXFF` remained the protected
rank-5 hard negative but was not selected as a related attachment.

### X08

Stage A enumerated 944 one-hop facts and contained the exact
`FormState -> FormStateProps` relation. The selector instead chose a
rank-1 `useFormState` self-parent type reference; both endpoints were already
the same primary parent, so it added no related body. Complete-at-five stayed
false.

The large `88` and `944` reachable sets explain why a graph alone cannot
identify the answer. Connectivity recovers evidence candidates, but a single
relation-kind/rank order has no representation of the question's requested
behavior or required evidence group.

## Artifact identities

Local ignored diagnostic references:

```text
.cidx/test/states/chi-1024-int8/evaluations/relation-diagnostic-chi-clean-0283405
.cidx/test/states/react-hook-form-1024-int8/evaluations/relation-diagnostic-rhf-clean-0283405
```

| Corpus | Run manifest | Aggregate metrics | Per-query trace | Probe results | Artifact checksums |
| --- | --- | --- | --- | --- | --- |
| chi | `2fe9ea7d766193a480b8fe527f0bc200453a7c6358337ceea06344f305f4e650` | `974e8926b8feba8d4a823df71c1426618aca193d085fa6b9d2bf20d37afd1d86` | `6a1b050307fb573c397244ea971d1c57d25888dfe5b66feb87a5ca46ccd01696` | `8781b2a913c5efd07672fa18a21210d729758c3d7346dfdb15f4051a50dab0ad` | `e332d1cd580a957da1cdde6233f384affc552c50de59449c899669c4d2d9e544` |
| RHF | `2d6b0b2128557b03b7e16d67e0590a8b79b4b0a57d3221dbf2ce2891baa79369` | `f5976f031781328e942d50c875cc0c2431ea4045c230a9900a3554d91b414e77` | `d334173b68c3303e12a4ec1f84a9134da48d2f2ce4b8e46fdc0074c5fc3a1546` | `713a4d6d6a8984bd1021f4879e7ab0cbea07a6e55ee00f1bd0ae2874586a3d03` | `8bbe2e5c08c40346bcd544f43c5b1e380d1a1b33de459cbac41395e92d2f137e` |

The earlier dirty smoke and the clean accepted run produced byte-identical
aggregate metrics and per-query traces for both corpora.

## Validation and review

One main commit-boundary validation passed:

```text
GOPROXY=off go test -count=1 -race ./internal/relationdiag ./internal/devlab
GOPROXY=off go test -count=1 ./...
GOPROXY=off go vet ./internal/relationdiag ./internal/devlab
GOPROXY=off go build ./cmd/cidx
node --check tools/relationdiag/typescript-resolver.mjs
sh -n scripts/materialize-relation-typescript.sh
jq -e . testdata/retrieval/relation-probes-chi-rhf-v1.json
go mod tidy -diff
gofmt -l <owned Go files>
git diff --check
```

The final Terra review is `CLEAR`. It verified production isolation,
generation/source/toolchain reproof, exact-universe TypeScript resolution,
compiler-backed membership, checksum consumption, corpus-scoped exact probes,
first-loss attribution, hard-negative gates, the implemented v2 selection
order, and explicit-root workspace behavior. A separate actual invocation from
`/tmp` with `--root` successfully built the chi graph, proving that development
source and state resolution do not depend on the caller's current directory.

The pinned TypeScript package was materialized once into ignored local state
during implementation preparation. The accepted graph and replay runs were
offline, removed `VOYAGE_API_KEY` from their environment, made no Voyage call,
made no document or query embedding request, changed no production database,
and incurred zero provider cost.

## Handoff

Keep the relation sidecar and this negative evidence as a development tool.
Do not add its current selector to production.

A later proposal is justified only if it introduces an independently specified
query-conditioned admission layer that can choose coherent evidence groups
without labels, keeps the exact resolver/provenance contracts, and is evaluated
first on a newly authored unit rather than tuned until G09/X08 pass. The next
formal Phase 07 requirement remains a separately frozen, unexposed confirmation
set; this diagnostic does not satisfy or weaken that gate.
