# Assistant cidx Follow-up Experiment Overbuild Incident

- Date: 2026-08-23
- Phase: 14
- Disposition: removed from active `main` by owner-authorized hard reset
- Reset target: `a155825eb11d7bd49e2054db0bb8d74a40095e9d`
- Product/retrieval impact: none
- Scored successor turns: none

## Summary

After the forced-cidx prompt diagnostic completed, its successor was intended
to compare a short cidx-awareness prompt with a cidx-priority and bounded-trust
prompt. The work expanded into a promotion-grade audit and provenance system
before the new primary experiment began.

The expansion was not proportionate to a non-promotion assistant diagnostic.
The owner rejected it and directed a hard reset rather than a revert series or
incremental repair. Only the concise MCP descriptions, two prompt paragraphs,
and optional post-run questions survive in
[the next-experiment boundary](../../ASSISTANT-CIDX-NEXT-EXPERIMENT-BOUNDARY.md).

## Removed work

The active branch removed eight commits after the accepted forced-prompt result:

- `a96dcb7` and `19e8460`: oversized successor plan and contract expansion;
- `385b4be`: 1,404-line predecessor journey re-audit;
- `13324e6`: semantic packet builder and semantic schemas;
- `31c1e61`: isolated semantic reviewer runner;
- `69eb33b`: provider-schema repair;
- `b85ed3f`: field-local semantic projection; and
- `145dafc`: semantic-audit closure and reporting.

The reset also discarded the uncommitted dedicated awareness/trust runner,
semantic-review runner, semantic and self-report schemas, manifest, MCP
description edit, and approximately 5,800 lines of awareness/trust scorer
expansion. Ignored packet variants, semantic outputs, draft preflights,
binaries, and generated Python caches were removed from the workspace and
placed outside the repository temporarily for recoverability.

## What consumed the time

The removed work added approximately 6,350 committed lines and approximately
10,100 more uncommitted lines. It included:

- seven semantic packet preparation variants;
- one 60-cell semantic attempt rejected before model work by unsupported
  response-schema forms;
- a second 60-context semantic pass over 358 historical actions;
- a 251-finding field-level normalization layer;
- a second mandatory 60-context semantic pass designed for the future run;
- mandatory original-session self-report for all 60 future cells;
- multiple artifact seals, digests, identity joins, and review-specific
  schemas; and
- repeated external and internal review rounds that continued to add contract
  requirements after the experiment was already non-promotional.

No successor primary A/B turn was executed despite this work.

## Root causes

1. A closed predecessor result was treated as needing a new semantic audit
   instead of being used directly as the baseline observation.
2. A bounded diagnostic inherited release/promotion-grade provenance and
   sealing requirements without an owner decision that required that rigor.
3. Existing runner, isolation, trace, classifier, and grading capabilities
   were duplicated instead of minimally generalized.
4. Side-panel review was allowed to recursively expand scope rather than being
   advisory within a frozen owner boundary.
5. Implementation size was not surfaced before it grew far beyond the primary
   experiment. The added complexity introduced its own runtime and contract
   defects.

## Required prevention rules

- A completed experiment remains closed unless the owner explicitly requests a
  re-audit. Its result may be cited without rebuilding its evaluation system.
- Before adding a new evaluation runner, scorer, schema family, model-review
  pass, or more than a small generalization of existing infrastructure, report
  the proposed files, model-call count, and scope to the owner.
- Non-promotion assistant diagnostics use deterministic raw traces and existing
  blind grading by default. Semantic model review requires explicit approval.
- Post-run agent questioning is selective diagnostic follow-up unless the owner
  explicitly requires a full denominator.
- One pre-run external review round may identify concrete blockers. After the
  owner boundary is reconciled, new advisory suggestions are recorded as later
  work rather than automatically implemented.
- If evaluation-support implementation becomes larger or more complex than the
  product behavior under test, stop before expanding and request an owner
  decision.

## Recovery and current boundary

The hard reset restored the repository to the immutable forced-prompt result.
No product code, public MCP contract, ranking, corpus, embedding, or provider
operation from the rejected work remains active. The next experiment is not
authorized merely by the retained design note; implementation resumes only
after the owner approves its minimal execution boundary.

## Checks performed

- resolved the exact reset commit before mutation;
- staged the three retained design elements outside the repository and verified
  their digest before reset;
- reset `main` directly to the named commit without a revert series;
- removed the rejected untracked files and generated local experiment state
  from the workspace; and
- verified that no scored successor run existed.

No force-push, remote mutation, product test, provider call, or successor model
run was performed as part of the reset.
