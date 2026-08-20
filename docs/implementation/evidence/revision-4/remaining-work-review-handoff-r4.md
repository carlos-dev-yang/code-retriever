# Remaining-Work Review Handoff — Revision 4

- Date: 2026-08-19
- Status: owner review; not `core_retrieval` or `release_candidate`
- Authority: `OWNER_ADOPTED_DUAL_AI_REVIEW` / `NO_INDEPENDENT_HUMAN_REVIEW`
- Provider operations in this freeze: zero
- Assistant A/B: not run, not authorized by this document

Owner-facing index (overview, live numbers, document map):
[`OWNER-REVIEW-INDEX.md`](../../OWNER-REVIEW-INDEX.md).

This file is the stop-and-review record after every internally completable
follow-up from the closed packaging replay. A later owner pass should be able
to resume without chat history.

## 1. What is already a product

The corpus-independent implementation through the local Phase 14 package
checkpoint is accepted. cidx is a local auxiliary search MCP:

- free AST/FTS indexing of the live worktree;
- optional Voyage `voyage-code-4` embeddings;
- default serving `1024/int8`, explicit compact `512/int8`;
- durable document 1024-f32 in the product source bank;
- query f32 never persisted;
- SQLite as the only search authority;
- MCP tools exactly `status`, `search`, `read_span`, `reindex`.

Phases `00–06`, `08–11`, and `13` are `done`. Phase 14 has a local
darwin/arm64 archive from provenance `5f4955e1499ee8896be5c825ef0fb9b3a52abb70`.
That archive is not `release_candidate`.

Binary and 256-dimensional code paths are absent from the product. Their
reports remain historical evidence only.

## 2. What this freeze completed

### 2.1 Packaging experiment (closed 40-query unit)

Live command:

```text
env -u VOYAGE_API_KEY go run ./cmd/cidx dev relations packaging \
  --contract testdata/retrieval/relation-packaging-experiment-contract-v1.json \
  --output-dir .cidx/test/experiments/relation-packaging-v1
```

Preflight hashes matched the Stage E/F handoff (Stage F A/B byte-identical,
selection `NO_POLICY_SELECTED_EVALUATION_ONLY`, overlap diagnostic v2
`33a91723549c12486da93c07a638907537ff5065ad7375432109cbb19939656d`, frozen
labels `002a30b08e137467896df63f2e5da8bf176c965f06c6a164aee7fd4db565a19b`).

Decision:

```text
CONTINUE_SIBLING_PACKAGING
sibling_gate             true     5/6 named same-file misses
one_hop_gate             false    nearby 2/2, but gg-g09 completed in every C cell
primary_equal            true
topology-complete A/B/C  27 / 32 / 36 of 40
gg-g09 still incomplete on A and B
labeled isolated extras  0 on B decision cell
provider operations      0
```

Local artifacts remain ignored under
`.cidx/test/experiments/relation-packaging-v1/`.
`decision.json` SHA-256
`49f2d483fb5d75a18854c36de33cf583d2b7c62f648737112faff33b59b3b5ae`.

Exact named outcomes and the 27-versus-31 denominator note are in
[packaging experiment evidence](../phase-07/relation-packaging-experiment-r4.md).

### 2.2 Adopted evaluation contract

Tracked, evaluation-only:

```text
testdata/retrieval/relation-sibling-packaging-adopted-v1.json
kind    cidx.relation_packaging.adopted_evaluation_contract.v1
digest  d0b288b321cee2b60a794a0a38d7134395381491c9ede8b02d1af09ff2d65250
```

Adopted:

- same-file sibling extras, count 4, 4096 body bytes, greedy skip-oversize;
- dense top-five identity and order remain protected.

Rejected / not authorized:

- default one-hop graph push;
- Arm D combination;
- production graph path;
- MCP wire change;
- assistant final-answer A/B;
- retuning either closed calibration.

`gg-g06-commit-object` remains a recorded sibling-cap limitation: the needed
parent is rank 9 in the same file, but that file has on the order of 141
extras and symbol order does not reach it inside count 8.

## 3. What was not done, and why

These are external gates. They were not skipped for convenience.

| Work | Why it did not run |
| --- | --- |
| Unexposed confirmation unit | Owner must select new repositories. This agent must not choose, clone, or embed a corpus. The 32-case chi/RHF set and the 40-query relation set are closed to confirmation use. |
| Official Phase 12 `core_retrieval` | Requires that confirmation unit, frozen margins, and a paid query-embedding approval after the policy is sealed. Corpus-independent adapters already exist. |
| Phase 14 `release_candidate` | Requires official Phase 12 core evidence plus frozen assistant/host inputs. Local package smoke is not that evidence. |
| Assistant-use A/B | Owner deferred it. Host/model/prompt variance cannot close the current retrieval gate. |
| Production sibling MCP field | Adopted cell is evaluation-only until a separate product wire design is approved. |
| One-hop proxy retune (rank or role filter) | Would be post-hoc tuning on a closed unit. Needs a new frozen contract and a new unexposed set. |

Phase 07 therefore cannot be marked `done`. Formal completion still requires
the unexposed confirmation set. After this freeze the operational state is
`blocked` on owner corpus selection, not on missing packaging code.

## 4. Closed calibration that must not be reused as confirmation

Do not rescore, add cells, or retune:

1. chi v5.3.1 + React Hook Form v7.85.0 — 32 frozen questions,
   `owner-adopted-dual-ai-v1`, dense `1024/int8` baseline Complete@5 `30/32`,
   tested 1:1 and FTS1:dense2 RRF rejected.
2. go-git v5.19.1 + Zustand v5.0.14 + Memos v0.30.0 — 40-query Stage E/F,
   `NO_POLICY_SELECTED_EVALUATION_ONLY`.

Graph admission policies measured on (1) are rejected for product use:
graph-first, frequency, popularity, Pareto, bridge-only, automatic closure
body push, and relation-text embedding.

## 5. Owner decisions required before work resumes

Answer these before any new implementation or paid call:

1. Confirmation repositories: which open-source roots, pinned commits, and
   licenses? Do not reuse the closed calibration identities.
2. Confirmation floor: Phase 07 still names a promotion-capable set of at
   least 90 answerable queries (30 Go / 30 TypeScript / 30 TSX), 18 verified
   abstainable/hard-negative queries, and 10 cases in every critical cohort.
   Confirm or explicitly replace that floor before authoring.
3. Whether sibling 4/4096 should later become a product MCP payload field.
   Default remains no.
4. Whether assistant-use A/B is reopened as a Phase 14 host experiment.
   Default remains deferred.
5. Paid query-embedding approval is requested only after confirmation labels,
   margins, arms, and the promotion contract are sealed.

## 6. Confirmation intake (do not execute yet)

When the owner names repositories, the next engineer should:

1. Commit portable manifests only (URL, commit, SPDX, language slice,
   include/exclude, expected tree/content hash). No absolute paths.
2. Bind checkouts through ignored `.cidx/test/corpora.local.json`.
3. Verify clean worktree, pinned commit, content hash, and license before
   indexing.
4. Index locally (`cidx index` / MCP `reindex`) with zero Voyage calls.
5. Freeze draft questions before any score exposure. Do not rewrite closed
   calibration failures into confirmation wording.
6. Request document embeddings only after an explicit document-spend gate.
7. Request query embeddings only after labels, policy, and margins are sealed.
8. Review under `owner-adopted-dual-ai-v1` with whole-digest adoption.
9. Run paired confirmation against the frozen product profile `1024/int8`
   (optional explicit `512/int8` compact arm from the same source bank).
10. Publish `scope=core_retrieval` only if every Phase 12 hard gate passes.

Sibling packaging, if tested at all on confirmation, must use the adopted
4/4096 cell without grid expansion. One-hop default push stays off.

## 7. Checks run for this freeze

```text
WRITE_ADOPTED_PACKAGING_CONTRACT=1 \
  go test -count=1 ./internal/relationdiag -run TestCanonicalAdoptedPackagingContract
go test -count=1 ./internal/relationdiag ./internal/devlab
go test -count=1 -race ./internal/relationdiag ./internal/devlab
go vet ./internal/relationdiag ./internal/devlab
go build ./...
gofmt -l internal/relationdiag/packaging.go \
  internal/relationdiag/packaging_adopted.go \
  internal/relationdiag/packaging_test.go \
  internal/devlab/relation_packaging.go
go list -deps ./internal/search ./internal/mcp ./internal/store ./internal/vector \
  ./internal/app ./internal/cli
# no cidx/internal/relationdiag
```

Live packaging replay and Stage F hash reproduction are recorded in the
packaging experiment evidence. They were not re-run for this freeze.

## 8. Checks not run

- New corpus selection, clone, index, or embed.
- Confirmation questions or dual-AI review.
- Phase 12 official promotion contract/result.
- Phase 14 host/assistant tasks, signing, notarization, non-darwin builds.
- Production search/MCP schema change.
- Voyage operations.

## 9. Read these on resume

1. `AGENTS.md`
2. `docs/implementation/STATUS.md` (authoritative phase state)
3. this handoff
4. [adopted sibling contract](../../../testdata/retrieval/relation-sibling-packaging-adopted-v1.json)
5. [packaging experiment](../phase-07/relation-packaging-experiment-r4.md)
6. [EVALUATION-CONTRACT.md](../../EVALUATION-CONTRACT.md)
7. [Phase 12](../../12-retrieval-evaluation.md) if confirmation repositories exist
8. [Phase 14](../../14-packaging-and-host-integration.md) only after official core evidence

Exact next action after this file: wait for owner review of §5. Do not author
confirmation questions, do not touch production search, and do not start
assistant A/B from this handoff.
