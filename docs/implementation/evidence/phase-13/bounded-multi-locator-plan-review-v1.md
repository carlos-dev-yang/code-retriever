# Bounded Multi-Locator Plan Review V1

- Date: 2026-09-04
- Phase: 13
- Scope: one terminal, provider-free `read_span` compatibility experiment
- Disposition: `PROCEED_ONCE`

## Reviewed direction

The owner discussion and the bounded ChatGPT/Grok review converged on one
fundamental change: test whether evidence already selected by the assistant can
be acquired in fewer MCP round trips. The experiment must not tune FTS,
ranking, prompts, question labels, or assistant orchestration again.

The accepted boundary is:

- preserve the scalar request and the four-tool product default;
- expose one evaluation-only v2 request containing two to four caller-selected
  locators;
- preserve each source unit and locator identity separately and in order;
- use the existing aggregate inline-byte maximum and fail atomically;
- add no automatic sibling/dependency expansion, range widening, session
  state, fifth tool, provider call, corpus, or question; and
- compare scalar and batch-capable contracts once on the same 30 questions,
  with identical trust-priority prompts and all other controls fixed.

## Review reconciliation

The material scheduling concern was whether to defer the wire experiment until
after Phase 12. It is run now because Phase 12 confirmation still requires an
owner-selected unexposed corpus, while Phase 14 needs a stable host-facing
contract. The result is terminal: retain v2 only if quality is preserved and
evidence actions improve; otherwise keep scalar v1 and move on. No additional
prompt or orchestration variant follows this result.

External review is advisory. The executable contract and retain/reject gates
remain those in
[`READ-SPAN-MULTI-LOCATOR-EXPERIMENT-V1.md`](../../READ-SPAN-MULTI-LOCATOR-EXPERIMENT-V1.md).
