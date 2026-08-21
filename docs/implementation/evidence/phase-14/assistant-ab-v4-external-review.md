# Assistant A/B V4 Post-result External Review

- Date: 2026-08-22
- Phase: 14
- Scope: advisory review of the completed locator-only V4 result
- Product change in this checkpoint: none
- Paid/provider action: none
- Corpus change: none

## Evidence presented

The same English review packet was sent to the existing ChatGPT and Grok
side-panel conversations. It identified the experiment as a local auxiliary
code-search MCP diagnostic and supplied the frozen V4 controls and results:

- all 24 paired turns were valid and both arms blindly graded 12/12 complete;
- compact search emitted 72,172 structured bytes and no text or source body;
- cidx event payload was 82.9% below V3, while the official paired model-token
  median was 1.044 and only 4/12 pairs were non-increasing;
- treatment performed 23 searches and 39 `read_span` attempts, including six
  failed ranges and four exact successful rereads;
- first-search complete exact locator coverage was 7/12, so a smaller `k` was
  not established as safe; and
- the requested decision was exactly one bounded V5 intervention, with no new
  corpus, paid embedding, hybrid search, ranking change, or bundled changes.

The reviewers were asked to choose one of `PROCEED_V5_ORCHESTRATION`,
`CHANGE_V5_INTERVENTION`, or `STOP_AFTER_V4`, provide exact model-facing
wording, preserve or reject each frozen control explicitly, and state the
evidence that would accept or reject the intervention.

## ChatGPT disposition

`PROCEED_V5_ORCHESTRATION`

The review concluded that V4 is sufficient to test assistant orchestration
next because response compaction preserved correctness and removed the known
transport excess, while the remaining directly observed losses are repeated
searches, repeated reads, failed ranges, refinements, and low locator use. It
explicitly did not infer that orchestration is the only token driver, that a
smaller `k` is safe, or that retrieval, ranking, or server behavior should
change.

Its important measurement clarification is adopted: failure to follow the V5
prompt is an outcome of the prompt intervention, not a harness-invalid or
ungradable turn. Correct execution, isolation, timeout, schema, and artifact
failures retain their existing operational treatment.

## Grok disposition

`PROCEED_V5_ORCHESTRATION`

The review independently attributed the next bounded test to the same visible
assistant behaviors and approved the proposed prompt-only protocol. It also
rejected a simultaneous `k`, retrieval, ranking, or `read_span` change and
required the full contemporaneous 24-turn pair rather than reuse of V4
baseline turns.

Grok proposed zero identical searches, zero identical reads, at most one
out-of-range read, a treatment search-call median no greater than two, and
higher locator citation utilization as mechanism evidence. These are reported
alongside the already frozen correctness and token gates; they do not replace
or relax those gates after observing V4.

## Accepted single V5 intervention

Only the assistant-facing orchestration paragraph changes:

> If the cidx MCP tools are available, begin with exactly one cidx `search`
> call using `max_inline_bytes=0`. Treat search only as locator discovery. Do
> not repeat an identical search. For any selected locator, first call
> `read_span` using exactly the returned `start_line` and `end_line`; do not
> widen the range before reading it. Perform at most one additional refined
> search, and only when you can identify specific material evidence that is
> still missing. Do not repeat an identical `read_span`. Stop requesting
> additional source once every material answer claim has direct cited
> repository evidence.

This paragraph is one prompt-level policy intervention. Its component clauses
jointly constrain the already observed search-to-evidence journey and are not
separate product or retrieval changes.

## Frozen V5 guardrails

V5 must preserve V4's:

- approved chi and React Hook Form repository revisions and local states;
- versioned 12-task panel, required evidence, order, and arm schedule;
- Codex model/version, reasoning effort, isolation, timeout, and token source;
- baseline and treatment task prompt outside the new paragraph;
- locator-only MCP contract, `k=10`, FTS planner, candidate generation,
  ranking, and `read_span` semantics;
- blind grading and arm-blind mechanical journey reducer; and
- complete 24-turn contemporaneous execution with no selective replacement.

Prompt-policy noncompliance remains in all outcome denominators. It is recorded
mechanically and does not invalidate an otherwise operationally valid turn.

## V5 decision evidence

Correctness remains a hard preservation boundary: no complete-to-noncomplete
reversal, no new material false claim, and no loss of required-group evidence
coverage. The predeclared token gate also remains visible and unchanged:
paired dual-complete model-total median ratio at most 0.85 and at least 8/12
non-increasing pairs.

Mechanism evidence is reported separately:

- prompt adherence by clause;
- identical search and identical `read_span` counts;
- initial exact-range and out-of-range read counts;
- per-task and aggregate search/read calls;
- locator duplicate exposure, selection/citation utilization, and false leads;
- successful-read precision, evidence citation utilization, and redundant
  source delivery; and
- official paired model-total and uncached-input comparisons.

If mechanism waste falls while correctness is preserved but the frozen token
gate still fails, V5 establishes only that the prompt changes tool behavior.
It does not establish cidx token benefit or release readiness. If the assistant
does not follow the policy, the intervention itself is rejected as ineffective
without changing the retrieval system to compensate.

## Decision

The two independent reviews and the local first-loss attribution agree. Freeze
V5 with the paragraph above as the sole change, record exact identities, run
the full pair once, and grade it before selecting any third experiment. Do not
lower `k`, clamp ranges server-side, enable hybrid/dense, or acquire another
corpus at this boundary.
