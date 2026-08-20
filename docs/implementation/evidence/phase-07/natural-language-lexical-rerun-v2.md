# Natural-Language Lexical Planner v2 — Sequential Correction and Rerun

- Date: 2026-08-20
- Final code commit: `2e1a270631286015c768fda379c23924fea8a5a0`
- Question sets: unchanged chi/RHF `critical-general-v2`
- Repositories: existing approved chi v5.3.1 and React Hook Form v7.85.0 checkouts only
- Provider operations: zero
- Evidence class: provider-free calibration diagnostic; not promotion evidence

## 1. Scope

This work corrected the diagnosed natural-language lexical admission defect and
verified each correction before proceeding. It did not edit a question, truth
span, cohort, prior result, repository, embedding, dense rank, or public MCP
tool. Every successful rerun has a new immutable run ID and is registered with
its exact question version, code commit, and artifact SHA-256.

The question was deliberately narrow: can the local lexical lane admit and
rank agent-style natural-language queries after removing global all-token
`AND`, while preserving exact-identifier behavior and safe FTS grammar?

## 2. Sequential correction record

### 2.1 Planner and safe descriptive admission

Commit `1e8fad9` introduced planner version 2. Natural-language descriptive
terms are safely quoted and joined with `OR`; raw input is never executable
FTS syntax. Exact/normalized symbol, path, and descriptive FTS candidates are
independent local lanes. Their parent ranks are fused deterministically, and
the dense provider remains independent in hybrid mode.

Focused normal/race tests, vet, build, format, and MCP/evaluation integration
checks passed before any real-corpus run.

### 2.2 Artifact integrity discovered by the first real preflight

The first attempted artifact write failed closed because the FTS candidate did
not carry the authoritative indexed source SHA even though the joined chunk
row had it. No failed artifact was published. Commit `dcdfd78` propagated the
source hash into lexical candidates and preserved the existing corruption
checks.

### 2.3 First successful rerun and PascalCase regression

The `*-lexical-planner-v2-1` pair completed at commit `dcdfd78`:

| Corpus | CompleteRequirementHit@5 |
| --- | ---: |
| chi | 11/18 |
| React Hook Form | 18/26 |
| combined | **29/44** |

It exposed one regression in the previously passing `Router interface` case:
PascalCase was not recognized as a possible code anchor.

### 2.4 Eager PascalCase correction and diagnostic overreach

Commit `d352015` treated every leading-capital code-shaped token as an anchor.
The `*-lexical-planner-v2-2` pair restored the case and reached `30/44`, but
inspection showed that ordinary sentence-initial words such as `How` were
being reported as syntactic anchors. Retrieval improved, but the planner's
explanation was too broad.

### 2.5 Final index-resolved weak anchor

Commit `2e1a270` made leading-capital single-word candidates weak. A weak
candidate becomes a symbol anchor only when the pinned index snapshot contains
a matching symbol. Thus `Router interface` remains mixed, while ordinary
agent prose remains descriptive unless the repository supplies lexical
evidence. The final `*-lexical-planner-v2-3` pair reproduced `30/44` with the
intended query shapes and zero execution failure.

## 3. Immutable run lineage

| Run pair | Code commit | chi artifact SHA-256 | RHF artifact SHA-256 | Combined Complete@5 |
| --- | --- | --- | --- | ---: |
| planner v2 pre-Pascal | `dcdfd7825978b73f60b7c59c74ad6b67ce8bb077` | `ac87741c73f864c49aa507e1d3c524b557cf1a3ccec5f269b7df5ca41ca0073b` | `4564ab63d915a79f3dc60375f12ded49b646207331e0379dae3e88f5d686f60d` | 29/44 |
| planner v2 eager-Pascal | `d352015eed3c2f5930719b20371b53cae591d6cd` | `b5b9295317f593fa4836466d0448e10a1d76cc9e8b500d8c10ce3decaa4b1aac` | `61a1bef0534bbbd523c45c4a4b7a85677b94caf9ae6eb574e38d0ff050e2e8df` | 30/44 |
| planner v2 final | `2e1a270631286015c768fda379c23924fea8a5a0` | `c4724b6b753538623005bbdb3e8a65deb233a05220e356ac34d8cc65a75f56ae` | `61eb323dfd3fbcaf6ce1871fbb414dc8c80178d967882d45ab9151a252481a28` | **30/44** |

The tracked registry is
[`question-set-run-registry-v1.json`](../../../../testdata/retrieval/question-set-run-registry-v1.json).
The run JSON remains in ignored project-local evaluation state.

## 4. Final result

Primary metric is `CompleteRequirementHit@5`.

| Arm | Overall | Go | TypeScript | TSX |
| --- | ---: | ---: | ---: | ---: |
| historical all-token-AND FTS | 10/44 (22.7%) | 6/18 | 2/16 | 2/10 |
| historical any-token simple control | 28/44 (63.6%) | 9/18 | 11/16 | 8/10 |
| final lexical planner v2 | **30/44 (68.2%)** | **12/18** | **9/16** | **9/10** |

The final planner retained all ten historical FTS passes and recovered twenty
previous failures. There was no old-pass/new-fail regression.

| Critical cohort | Historical FTS | Simple control | Final planner v2 |
| --- | ---: | ---: | ---: |
| lexical anchor | 10/12 | 10/12 | **12/12** |
| semantic only | 0/24 | 13/24 | **15/24** |
| mixed signal | 0/8 | 5/8 | **3/8** |
| multi requirement | 0/4 | 2/4 | **1/4** |
| contract disambiguation | 2/5 | 1/5 | **2/5** |

The hard-negative denominator is only one. Its misleading target remained
absent from the top five, but its required answer was also not complete. This
is not enough evidence for a general hard-negative claim.

## 5. Admission and ranking are now separable

The previous planner returned zero candidates for all 32 semantic/mixed
queries. The final planner produced at least one local candidate for all 44
queries.

| Stage | Result |
| --- | ---: |
| nonzero final local candidate set | **44/44** |
| all required groups present within candidate depth 20 | **38/44** |
| required groups present at candidate depth 20 | **42/48** |
| all required groups present in returned top five | **30/44** |
| complete at depth 20 but displaced beyond top five | **8/44** |
| incomplete already at candidate depth 20 | **6/44** |

The all-token-AND zero-admission defect is therefore corrected. The remaining
fourteen incomplete top-five cases are not one failure class: six still need
better lexical candidate generation or another independent provider, while
eight entered the candidate pool and need ranking/depth analysis.

Final inferred query shapes were six anchor, nine descriptive, and twenty-nine
mixed queries. The symbol lane was nonempty for 25/44 and descriptive FTS for
44/44. The path lane was nonempty for 0/44 because this v2 set contains no
query that the planner classifies as an explicit path expression. Existing
synthetic path-lane checks pass, but real-corpus path usefulness is unmeasured
by this run and must not be reported as either success or failure.

## 6. What is corrected and what remains

Corrected at this boundary:

- global inferred-term `AND` for agent prose;
- unsafe conflation of candidate admission with BM25 rank quality;
- weak symbol treatment for code-shaped anchors;
- path/symbol/descriptive candidate-order separation;
- local lane fusion and shared hybrid/MCP diagnostic propagation;
- missing indexed source hash in real lexical artifacts; and
- misleading eager PascalCase query classification.

Not corrected or claimed:

- lexical planner v2 is not a semantic replacement for dense retrieval;
- mixed-signal, multi-requirement, and contract-disambiguation performance is
  still weak in this small calibration set;
- no current-v2 paid dense or hybrid query run was authorized;
- no real path-shaped v2 query exercised the path lane;
- no assistant A/B has yet measured answer correctness, source-reading
  reduction, token use, or false leads; and
- this exposed calibration set cannot establish `core_retrieval` or a release
  threshold.

The next owner-directed work is paired assistant A/B on these existing
repositories. It should compare the same fixed assistant, prompt, task truth,
budget, and ordering with and without the four-tool MCP. It must not reinterpret
this lexical score as an assistant-use result and must not add a repository as
part of this handoff.

## 7. Checks and excluded operations

Checks actually run before the final artifacts:

```text
go test -count=1 ./internal/config ./internal/profile ./internal/symbol ./internal/store ./internal/search/lexical ./internal/search ./internal/eval ./internal/devlab ./internal/mcp
go test -count=1 -race ./internal/symbol ./internal/search/lexical ./internal/store ./internal/search ./internal/eval ./internal/devlab ./internal/mcp
go vet ./internal/symbol ./internal/search/lexical ./internal/store ./internal/search ./internal/eval ./internal/devlab ./internal/mcp
go build -o /tmp/cidx-lexical-planner-v2 ./cmd/cidx
git diff --check
```

The final evaluation binary reported clean VCS revision
`2e1a270631286015c768fda379c23924fea8a5a0` with `vcs.modified=false`.
No Voyage request, API-key access, document/query embedding, new repository,
question edit, prior-result overwrite, broad full-project test, or assistant
execution occurred.
