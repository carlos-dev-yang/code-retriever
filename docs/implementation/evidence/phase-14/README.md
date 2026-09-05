# Phase 14 Packaging and Host Integration Evidence Index

- Phase: `14-packaging-and-host-integration`
- State: `blocked`
- Updated: 2026-09-05
- Current blocker: immutable Phase 12 `core_retrieval` confirmation result
- Authority: [Phase 14 plan](../../14-packaging-and-host-integration.md)

This directory preserves package/host evidence and the assistant-use
experiment history. None of these records independently establishes
`release_candidate`.

## Completed package boundary

- [Revision 4 evidence](revision-4.md) and
  [int8 package reconciliation](int8-profile-package-reconciliation.md)
  record the current local package boundary.
- The verified package environment is darwin/arm64 only. Other OS/architecture
  combinations, installers, host matrices, and distributable artifacts remain
  unverified until this phase resumes.
- Current product profiles are default 1024/int8 and explicit 512/int8.
  Binary/256 remain negative-only historical cases.

## Assistant experiment chronology

| Step | Question answered | Durable record | Terminal interpretation |
| --- | --- | --- | --- |
| V3 | Does the initial FTS-first MCP help answer code questions? | [result](assistant-ab-v3-result.md) | Correctness held, but source-heavy responses increased model input |
| V4 | Does locator-only `search` remove response payload waste? | [freeze](assistant-ab-v4-freeze.md), [result](assistant-ab-v4-result.md), [review](assistant-ab-v4-external-review.md) | MCP payload fell sharply; full model-token efficiency still failed |
| V5 | Can concise search/read guidance reduce repeated work? | [freeze](assistant-ab-v5-freeze.md), [result](assistant-ab-v5-result.md), [review](assistant-ab-v5-external-review.md) | Tool/source volume improved; frozen token gate still failed |
| V6 | Does explicit hash-field guidance remove failed reads? | [freeze](assistant-ab-v6-freeze.md), [result](assistant-ab-v6-result.md), [review](assistant-ab-v6-external-review.md) | Hash omissions disappeared, but one correctness regression closed the series |
| V4–V6 closure | What survives all three experiments? | [closure](assistant-ab-v4-v6-closure.md) | No general efficiency claim and no V7 |
| Neutral availability | Will the assistant discover cidx from neutral tool exposure? | [freeze](assistant-availability-v1-step-3-freeze.md), [execution/grading block](assistant-availability-v1-step-4-execution-and-grading-block.md), [closure](assistant-availability-v1-blind-grading-closure.md) | cidx adoption was 0/30; this measured discovery, not retrieval quality |
| Forced prompt | What happens when the model is directed to use cidx? | [freeze](assistant-forced-cidx-prompt-freeze-v1.md), [result](assistant-forced-cidx-prompt-result-v1.md) | Source narrowed and quality held, but action/token cost increased |
| Overbuild reset | Which follow-up infrastructure was rejected? | [incident](assistant-cidx-followup-overbuild-incident.md) | Large semantic audit/runner work was removed before a successor run |
| Awareness vs trust | Does concise priority/trust guidance help under equal tool exposure? | [freeze](assistant-cidx-awareness-trust-freeze-v1.md), [result](assistant-cidx-awareness-trust-result-v1.md), [review](assistant-cidx-awareness-trust-result-external-review-v1.md) | Completeness and source scope improved; actions/model tokens increased |
| Bounded multi-locator | Can evidence round trips be reduced by a small batch request? | [Phase 13 plan](../../READ-SPAN-MULTI-LOCATOR-EXPERIMENT-V1.md), [result](../phase-13/bounded-multi-locator-result-v1.md), [restoration](../phase-13/bounded-multi-locator-scalar-restoration-v1.md) | Mechanism improved, retention gates failed; batch-v2 removed |

Supporting interface/runner history is retained in
[the response-contract direction](assistant-response-contract-and-v4-direction.md),
[the host-decided flow design](../../ASSISTANT-SEARCH-EVIDENCE-FLOW-DESIGN.md),
[the passive trace implementation](assistant-search-evidence-flow-step-2-implementation.md),
and the versioned grading adapters in this directory. These are diagnostics and
infrastructure records, not separate product claims.

## Conclusions that remain valid

- Compact locator-first search is the accepted MCP response shape.
- Exact scalar `read_span` is the accepted source/evidence contract.
- cidx can reduce the code scope an assistant inspects and can improve answers
  on some tasks, but current evidence does not prove general end-to-end token
  or action efficiency.
- Tool adoption, locator quality, evidence acquisition, claim support, and
  final answer quality must remain separate measurements.
- No exposed assistant run authorizes retrieval retuning, another prompt
  series, a fifth tool, batch-v2, or product promotion.

## Work not complete

- No immutable Phase 12 confirmation or `core_retrieval` result exists.
- No cross-platform package/build/checksum matrix or supported installer exists.
- No complete host integration matrix has been validated.
- No final paired assistant/host result has been bound to immutable core and
  package evidence.
- Therefore no `release_candidate` result may be issued.

## Resume procedure

1. Read the central [phase history](../README.md), current
   [`STATUS.md`](../../STATUS.md), Phase 14 plan, and Phase 13 terminal result.
2. Verify an immutable Phase 12 core result exists and matches the current
   scalar four-tool contract.
3. Freeze target OS/architecture/host scope before claiming support.
4. Run only the package, offline FTS/grammar, root/path/schema/error, stdout,
   host, and final assistant checks required by the Phase 14 plan.
5. Record either a scoped `release_candidate` result or explicit failed gates.

Do not infer current support from an older local checkpoint or repair a closed
assistant run. New evidence must be versioned and appended.
