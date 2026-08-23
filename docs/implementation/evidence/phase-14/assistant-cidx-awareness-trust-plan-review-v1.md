# Assistant cidx Awareness/Trust Plan Review V1

- Date: 2026-08-23
- Plan: [Assistant cidx Awareness vs Trust-Priority Evaluation V1](../../ASSISTANT-CIDX-AWARENESS-TRUST-EVALUATION-V1.md)
- Final disposition: `PROCEED`
- Reviewers: ChatGPT Pro and Grok in the owner's existing side-panel chats
- Initial plan SHA-256: `02dbfea8e9a1b218b474ab334cdb3f051be45d83205b0f073e314e52a1c2b32d`
- Initial review packet SHA-256: `ffc332c26fa805544196b05097c143facd086bd2d466f4ae442b16ebb91883c6`
- Correction-confirmation prompt SHA-256: `dc38986833ddb219d78061b06920fc84f07d684554c0cde5b86099af0ee0586c`
- Accepted plan SHA-256: `096c9c96baa5f5420a51474927b828afc18aab9e32654c0fc6b7e276b9448425`
- ChatGPT chat: <https://chatgpt.com/c/6a8a9d2b-5d10-83e8-88c0-32c6d9a5adf6>
- Grok chat: <https://grok.com/c/f4c3bde5-707d-4b7e-be8d-5973b9bc03f5>

## Review boundary

Both reviewers received the same complete English plan and the same strict
review request. They could identify at most three defects that would invalidate
the simple 30-pair experiment. The request explicitly prohibited a predecessor
re-audit, semantic packet or sidecar, mandatory 60-session self-report,
dedicated runner/scorer, new schema family, new repository, dense/HNSW work,
product redesign, another arm, or recursive review.

No repository source, evaluation corpus body, credential, ignored run artifact,
or Voyage operation was sent. The packet contained only the plan and the
already documented aggregate predecessor facts.

## Initial verdicts

| Reviewer | Initial verdict | Blocking findings |
| --- | --- | --- |
| ChatGPT | `BLOCKED` | incomplete failure/efficiency denominator rules; private per-cell state and `reindex` isolation not explicit |
| Grok | `BLOCKED` | shared `read_span` description contaminated Arm A with trust/stopping guidance; incomplete paired efficiency denominator rules |

The findings were narrow contract defects in the plan. Neither reviewer asked
for new retrieval work or new evaluation infrastructure under the imposed
boundary.

## Adopted minimal corrections

### 1. Keep shared `read_span` interface-only

The shared description is now exactly:

> Return the complete current source for one selected locator's exact path,
> range, and indexed hash.

Trust, no-reassurance reacquisition, stopping, and adjacent/dependency guidance
remain solely in Arm B's suffix. This preserves Arm A as awareness plus free
choice.

### 2. Close failure and efficiency denominators

Every scheduled execution now receives an outcome. Timeout, non-zero exit,
missing/invalid output, or mechanically ungradable output is `ungradable`, is
not retried, gives zero positive-group coverage, and leaves negative-only truth
as `N/A`. Claim rates expose their gradable/material-claim denominator.

All-cell action, source, and token totals are descriptive. Paired action/source
efficiency requires two completed intact traces; token comparison additionally
requires official usage in both cells. Incomplete prefixes remain censored
diagnostics, and every excluded pair and reason is visible.

### 3. Make existing per-cell isolation explicit

Each primary cell uses its own source snapshot, SQLite state copy, state root,
and MCP process from one frozen initial binding. A model-issued `reindex` may
mutate only that cell and never supplies another cell's initial state. Initial
identity mismatch, source mutation, missing provenance, or cross-cell leakage
stops execution; a private post-turn generation change by itself does not.
Evaluation and session artifacts remain outside the visible/indexed source.

### 4. Record the minimal implementation mapping

The semantic arm labels `aware_choice` and `trust_priority` map to the existing
runner IDs `neutral_cidx` and `directed_cidx`. Shared awareness belongs in the
base prompt and Arm B alone receives the suffix. The MCP descriptions come from
the live `tools/list` response, so the only planned product-code edit is the two
description strings in `internal/mcp/schema.go`; tool names, schemas, outputs,
ranking, and behavior remain unchanged.

## Confirmation

The same exact correction summary was returned to both reviewers with a request
for only `PROCEED` or one unresolved defect from the original scope.

| Reviewer | Final verdict |
| --- | --- |
| ChatGPT | `PROCEED` |
| Grok | `PROCEED` |

No additional requirement was added after confirmation.

After confirmation, Section 16 was appended as a non-normative 1:1 index from
the owner's 17 annotations to the already reviewed sections. It adds no
behavior, metric, call, or artifact; the accepted-plan digest above includes
that index.

## Final disposition and next boundary

The plan is accepted for minimal implementation. This review does not authorize
the removed predecessor audit, semantic reviewer, dedicated runner, scorer
expansion, new corpus, retrieval change, or paid action. Before any scored turn:

1. modify only the existing description/prompt/session/trace path needed by the
   accepted plan;
2. stop if the implementation crosses the plan's size or schema boundary;
3. run focused compatibility and contract checks; and
4. freeze exact prompts, descriptions, schedule, source/index identity, model,
   and execution-code identity in a clean commit.

No scored successor turn was run during plan reconstruction or review.
