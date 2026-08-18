# Phase 07 Graph-Only Pareto Admission Diagnostic — Revision 4

Date: 2026-08-18  
State: completed calibration diagnostic; policy rejected for product use  
Authority: `OWNER_ADOPTED_DUAL_AI_REVIEW / NO_INDEPENDENT_HUMAN_REVIEW`

## 1. Decision

The bounded relation frontier is useful as a complexity control, but the tested
admission rule is not a product policy.

The provider-free arm recovered both exposed incomplete cases and raised
formal related-evidence completeness from `30/32` to `32/32`. It did so without
changing the dense top five or attaching a reviewed hard negative. That
aggregate result is not sufficient: only `7/17` emitted bundles contained
reviewed useful evidence, while `10/17` were noise-only. The unique graph-only
Pareto branch was useful in only `1/7` emissions.

Retain as development evidence:

- compiler-resolved typed relations and their provenance;
- self-edge removal and repeated-occurrence aggregation;
- the top-two-per-anchor/direction/tier frontier as a provisional complexity
  control, not a serving threshold;
- direction, structural tier, bridge status, source focus, source diversity,
  target fan-in, and first-loss traces as diagnostic features; and
- proof that structural evidence can recover G09 and X08 while protecting the
  dense top five.

Reject for product use:

- direct-bridge admission alone;
- unique graph-only Pareto admission alone; and
- the combined bridge-or-Pareto rule measured here.

No more admission policy tuning may use these exposed 32 questions. Any later
design starts with a separately frozen development unit and then an unexposed
confirmation set.

## 2. Scope and isolation

The complete immutable relation-occurrence SQLite sidecars are retained only
as evaluation evidence. They are not imported by product indexing, storage,
search, MCP, FTS/RRF, or embedding paths. The arm reads the existing bounded
query frontier and does not rebuild or enlarge either graph.

No corpus, question, label, dense ranking, source bank, serving vector,
production database, or product configuration changed. The four executions
ran with `VOYAGE_API_KEY` absent and made zero document or query provider
operations. No query vector or relation embedding was created or persisted.

## 3. Frozen label-blind rule

Implementation policy:

```text
anchor-frontier-graph-only-pareto-v1
policy fingerprint 2ed879c70143bdd8de1287092b877013c7392fc967b64b42175a9128604083d6
```

The arm consumes only the accepted `FinalFrontier`, never the uncapped graph.

1. If the bounded frontier contains a direct edge between the two selected
   anchors, emit the first such edge under the existing deterministic frontier
   order.
2. Otherwise admit only outgoing selected-anchor edges whose endpoint is
   absent from the frozen dense top 20.
3. Within each structural tier, retain every Pareto-nondominated candidate:
   maximize exact `edge_occurrences / source_stratum_occurrences`, minimize
   `source_stratum_distinct_targets`, and minimize
   `target_incoming_stratum_distinct_sources`.
4. Dominance requires no worse values on all three dimensions and at least one
   strict improvement. Ratios use checked integer cross-products.
5. Exactly one survivor across all tiers emits. Zero records `NO_CANDIDATE`;
   two or more record `MULTIPLE_WINNERS`. Structural-tier order and stable
   identity never collapse multiple survivors.

There is no learned margin, weighted total, query/corpus/language exception,
label access, or source-body access during selection.

## 4. Implementation and validation boundary

Entry-status commit:

```text
654f58cbec6d5f51e50a6ed76fbe1f968bf0e003
```

Implementation commit:

```text
497c000bf0d3e9452fd8ff1ce9f570a3df144525
```

Only these implementation files changed:

- `internal/relationdiag/model.go`
- `internal/relationdiag/evaluate.go`
- `internal/relationdiag/evaluate_test.go`

The implementation adds an optional policy-local trace and independently
checks these equations before publication:

```text
final frontier
  = direct bridge
  + incoming exclusion
  + outgoing dense-top-20 endpoint exclusion
  + outgoing graph-only candidate

per-tier candidates = dominated + nondominated
global union = sum(per-tier nondominated)
```

Main-agent boundary commands all passed:

```text
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 -race ./internal/relationdiag ./internal/devlab
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 ./...
env -u VOYAGE_API_KEY GOPROXY=off go vet ./internal/relationdiag ./internal/devlab
go mod tidy -diff
gofmt -l internal/relationdiag
git diff --check
```

The independent Terra code review returned `CLEAR`. A main-boundary review
then strengthened bridge-case partition accounting before the final review,
which also returned `CLEAR`.

## 5. Executable and predeclared plan

Preserved clean executable:

```text
.cidx/test/bin/cidx-relationdiag-pareto-497c000
SHA-256 5ca2afe3d4012122c7399e1755d4a1d9fa33e3793aa6f47bed0cf57618547bc8
mode 0500
```

`version --json` records commit
`497c000bf0d3e9452fd8ff1ce9f570a3df144525` and
`source_modified=false`.

Predeclared ignored plan:

```text
.cidx/test/experiments/relation-graph-only-pareto/relation-graph-only-pareto-series-v1-497c000.json
SHA-256 b55be18ff7efeed99aa031e1c55a1212b3e67232e43c792503b636533a0d166a
```

The plan binds the executable, policy, exact rule, corpus manifests, graph
database and logical hashes, replay and dataset hashes, and all four run IDs.
The executable-provenance re-audit returned `CLEAR` after the exact executed
binary was preserved.

The first invocation stopped before evaluation because an older working
snapshot directory contained SQLite `-wal/-shm` side files outside its exact
published checksum entry set. It created no run. The accepted executions used
fresh evaluation-only snapshots containing only the four already published
files. Their database, manifest, logical graph, and resolution-summary bytes
are unchanged; no new WAL/SHM file appeared.

## 6. Immutable runs

| Corpus | Initial | Repeat | Queries |
| --- | --- | --- | ---: |
| chi | `relation-diagnostic-chi-graph-only-pareto-497c000` | `relation-diagnostic-chi-graph-only-pareto-repeat-497c000` | 12 |
| react-hook-form | `relation-diagnostic-rhf-graph-only-pareto-497c000` | `relation-diagnostic-rhf-graph-only-pareto-repeat-497c000` | 20 |

Run checksum roots:

| Run | Artifact checksum |
| --- | --- |
| chi initial | `248d5c8564e02d187326587217440beccbeddd420e863e8994b1883f96c39b92` |
| chi repeat | `1ae69b80badf60012c1d00c101267c4ddb873429268e080f33544966fe9fee9f` |
| RHF initial | `85b2bfa8b7ccd8d183a2d181695497c93519f897570583ad977c96003a26a43a` |
| RHF repeat | `41ae26e1d8374881f21949d5d8decb1e6c3393782d4670f6ac2212023c2b4c5f` |

Initial/repeat hashes are byte-identical for primary top-five proofs, frontier
traces, Pareto traces and denominators, bundles, related-body packages,
aggregate metrics, and probe results.

Key deterministic hashes:

| Artifact | chi | RHF |
| --- | --- | --- |
| primary top five | `0f4253dd073bd4955b8eaf415a295ba840440b7e3b2c924545991f1a824bea3d` | `71f7e8a51e61ee3381b2ba53107c456571ab5033d67b270f5d48e087bc31c71d` |
| Pareto trace | `03fdd82eae607f418994faa283720a8f0db4d212fb12856b82d69536c0ca3410` | `419490688a43412dd702009e76930f9dab23d03819d2eab72532b611d52334e5` |
| Pareto denominators | `08cb7954ce0f0903019fb86a4b2c5442b13555512859ed58092b52859c6eff60` | `3376964922564ed2014b32498671bd1815beb72e682594460c79b801052874a3` |
| bundles | `9c8be27cb803ab14b2d4871ea3d8c7f9904da41e99a41df06fe6c6e0d8a49b35` | `4f9676b47bd12cbf1ab20bfd8b96f30c29a761b7df6c6a14f5dea44532fd0d46` |
| aggregate metrics | `d59a85dce4da52017fe76311e2943453f2a44ad7ab9393f80ac9e5a1415bef97` | `4fbfe47ff9f9aa8505db2f3b6baa5692733454b1b4acd375637b38a0b02c8381` |

Primary hashes are exactly equal to the accepted frontier-cap artifacts. A
canonical comparison of every query's final-frontier digest and count is also
equal to the accepted frontier-cap run: chi proof hash
`1c7c81e165faf3528af7fc3720d376d16789c20e9d18b1305956efacad9c4f26`,
RHF proof hash
`d955a550e58c06f4fc0becfe76610c4d7380f5d259c6d64b0fc0c39ee581eb85`.

## 7. Measured frontier and admission

The cap was unchanged:

- chi: 51 final edges across 12 queries, maximum 8 per query;
- RHF: 153 across 20, maximum 11; and
- no query reached the global 32-edge ceiling.

The exact combined partition is:

```text
204 final frontier edges
= 11 direct-bridge edge views
+ 64 incoming exclusions
+ 38 outgoing dense-top-20 endpoint exclusions
+ 91 outgoing graph-only candidates
```

The 91 graph-only candidates produced 58 per-tier nondominated survivors
before the global exactly-one gate.

| Outcome | chi | RHF | Total |
| --- | ---: | ---: | ---: |
| `DIRECT_BRIDGE` | 4 | 6 | 10 |
| `ONE_WINNER` | 4 | 3 | 7 |
| `MULTIPLE_WINNERS` | 3 | 10 | 13 |
| `NO_CANDIDATE` | 1 | 1 | 2 |
| emitted | 8 | 9 | 17 |
| abstained | 4 | 11 | 15 |

## 8. Actual answer evidence

| Measurement | chi | RHF | Total |
| --- | ---: | ---: | ---: |
| baseline complete | 11/12 | 19/20 | 30/32 |
| augmented complete | 12/12 | 20/20 | 32/32 |
| useful emitted bundles | 3 | 4 | 7 |
| noise-only emitted bundles | 5 | 5 | 10 |
| hard-negative queries | 0 | 0 | 0 |
| `walkXFF` attachments | 0 | 0 | 0 |

By admission path:

- direct bridge: 10 emissions, 6 useful and 4 noise-only;
- unique graph-only Pareto: 7 emissions, 1 useful and 6 noise-only.

Attachment labels total 18 grade-0, 5 grade-1, 6 grade-2, and 5 unreviewed
parents. Three nonrequired related bodies were omitted as `BODY_TOO_LARGE`;
neither recovered requirement depended on an omitted body.

### chi G09

`chi-g09-client-ip` emitted the unique graph-only Pareto winner:

```text
RealIP -> realIP
CALLS / FORWARD
relation aee07df0bb4d608d25e240e7cce47771b11683f4eb93292ff443eed390351c60
middleware/realip.go byte 1111
```

Both attached parents are required grade-2 evidence. Their complete bodies are
872 and 351 bytes with hashes
`0486157b39400b4efc0cae6b2cd579871d4b8b8d409fd3a9c01abead04e49600`
and
`6774c1d9b88125452b6324093c876d0fba9c4211f62689deb6780301eaa5f8b6`.
The result changes incomplete to complete and does not attach `walkXFF`.

### RHF X08

`rhf-x08-form-state-props` emitted the direct selected-anchor bridge:

```text
FormState -> FormStateProps
TYPE_REF / FORWARD / DECLARATION_CONTRACT
relation 4baa57b19ec63201ccb8af423b5fb56f4f8f8a154b0cebe4e7dffbfbbd36d43e
src/formStateSubscribe.tsx byte 688
```

The required grade-2 endpoint body is complete at 230 bytes with hash
`3a1640e3b8a80792bbd0aa397883360a97dec78b16d331298a80309200e579b2`.
The result changes incomplete to complete.

## 9. Review and final boundary

The measurement guide confirmed the bounded interpretation:

- keep the typed relation and frontier machinery as calibration evidence;
- do not adopt bridge, Pareto, or the combined rule as product policy; and
- close tuning on the exposed 32 questions.

The independent Terra artifact audit verified the plan/executable/run chain,
all checksums and denominator equations, repeat determinism, accepted primary
and frontier identity, exact G09/X08 relations and bodies, zero-provider and
safety gates, clean graph snapshots, and the absence of product-path changes.
Its final result is `CLEAR`.

Phase 07 remains `in_progress`. A separately frozen unexposed confirmation set
is still required before any product relation integration or official
promotion claim.
