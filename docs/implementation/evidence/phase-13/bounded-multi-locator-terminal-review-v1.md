# Bounded Multi-Locator Terminal Review V1

- Date: 2026-09-05
- Phase: 13
- Reviewed result: `bounded-multi-locator-result-v1.md`
- Status: `ACCEPT_REJECT_BATCH_V2`

## Review sources

The terminal result was reviewed independently in two ways:

1. a terra/high read-only arithmetic and contract review against the frozen
   plan and ignored raw artifacts; and
2. a separate ChatGPT review in the existing `Improve Retrieval Design`
   conversation, using the complete frozen design, execution counts, gate
   outcomes, and grading failure.

Both reviews accepted `REJECT_BATCH_V2 / RETAIN_SCALAR_V1`. Neither review is
promotion evidence; the frozen plan, raw records, and local validators remain
the authority.

## Reproduced facts

- Trace arithmetic reproduces `107→112` searches, `132→75` read calls,
  `239→187` total cidx calls, `263→203` repository actions, and
  `163,062→166,980` combined unique source bytes.
- The mechanism denominator is 30 reference-defined eligible tasks; 21/30
  eliminated at least one read round trip and the paired median difference is
  `-2`.
- Duplicate and overlap rates increase when computed per task, which is the
  frozen gate definition.
- Exactly 58 blind rows pass the frozen semantic validator. They contain one
  partial-to-complete conversion and one complete-to-partial treatment-arm
  regression. That regression alone fails the no-regression gate.
- The two rejected grade rows are the same FieldArray question in both arms.
  Its 509-line product-valid evidence conflicts with the historical 500-line
  grader excerpt rule. This is a grader-contract failure with a missed
  preflight boundary, not a product `read_span` failure.

## Corrections and claim boundary

The eligibility description now follows the frozen definition: two valid
locators must be visible before first successful source acquisition; both do
not have to be read later. Duplicate counts are explicitly per-task rather
than global reuse across questions.

The result may say that the batch-capable arm reduced read round trips and
repository actions. It may not claim that batch calls caused every reduction
or caused the isolated unsupported assertion. It also may not claim aggregate
quality preservation or end-to-end token efficiency because the official
aggregate and dual-complete efficiency result remain `NOT_OBSERVED`.

## Accepted handoff

Preserve the closed experiment, restore scalar v1, fix the grader contract
only before a future evaluation, and stop assistant-interface iteration here.
The next product dependency is the owner-gated Phase 12 confirmation. Phase 14
packaging and release-candidate evidence follows only after an immutable core
result exists.
