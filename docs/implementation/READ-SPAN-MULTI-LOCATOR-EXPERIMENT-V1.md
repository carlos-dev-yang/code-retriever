# Bounded Multi-Locator `read_span` Experiment V1

- Status: `complete` — `REJECT_BATCH_V2 / RETAIN_SCALAR_V1`
- Owner: `/root`; implementation assistance: `gpt-5.6-terra` at high effort
- Affected phase: Phase 13 compatibility amendment; Phase 14 execution waits
- Provider/corpus impact: none; this experiment is FTS-only and provider-free
- Product disposition: scalar-v1 restored at `ff9d4e8`; batch-v2 preserved
  only as closed experiment evidence

## 1. Purpose

The awareness/trust experiment found that cidx could narrow unique source and
improve answer completeness, but repeated evidence reads increased repository
actions and model input. The next question is therefore not whether to tune
retrieval again. It is whether an assistant that has already selected several
useful locators can acquire their complete, line-addressable evidence in one
bounded round trip without losing answer quality.

This is one terminal compatibility experiment. It either retains the bounded
multi-locator branch for the stable contract or rejects it and restores the
unchanged scalar contract. It does not start another prompt-tuning series.

## 2. Context Recovery Checklist

Before changing code or running the experiment, read completely:

1. `EXECUTION-GUIDE.md`, `README.md`, `STATUS.md`, and
   `EVALUATION-CONTRACT.md`;
2. `13-cli-and-mcp.md` and its locator-only completion evidence;
3. `14-packaging-and-host-integration.md`;
4. the awareness/trust plan, freeze, result, external review, and overbuild
   incident; and
5. this document, including the contract, stop rules, and decision gates.

Confirm a clean worktree before implementation. Phase 13 must be
`in_progress`; Phase 14 must not run concurrently.

## 3. Fixed Scope

### In scope

- Preserve scalar `read_span` input, output, and errors exactly.
- Add one explicit, opt-in version-2 input branch to the same `read_span`
  tool.
- Accept two through four caller-selected, unique locators.
- Return separate evidence units in request order.
- Apply the existing `mcp.hard_max_inline_bytes` as one aggregate source-body
  ceiling and return no partial source on failure.
- Expose scalar-only or batch-capable schema through the existing
  evaluation-only MCP launcher so the two arms differ only in contract
  availability.
- Teach the existing trace and scorer to distinguish tool-call round trips
  from evidence units.
- Reuse the same frozen 30 questions, trust-priority instruction, corpora,
  FTS indexes, model, blind answer grader, and metric definitions.

### Out of scope

- A fifth MCP tool or a product configuration option.
- Retrieval, FTS, ranking, `k`, SQLite, indexing, embedding, provider, codec,
  or corpus changes.
- Automatic sibling/dependency expansion, range widening, merging,
  concatenation, summaries, diagnostics, or server session state.
- Proving that a locator originated from a prior search inside the stateless
  server. The host trace, not the server, checks prior locator exposure.
- A new runner, evaluator, schema family, semantic-review pipeline, or
  retrospective re-audit.
- More than this one scalar-versus-batch experiment.

## 4. Wire Contract Under Test

The scalar-v1 branch remains unchanged:

```json
{
  "path": "internal/example.go",
  "start_line": 10,
  "end_line": 30,
  "expected_sha256": "<64 lowercase hex>"
}
```

The opt-in batch branch is:

```json
{
  "input_version": 2,
  "locators": [
    {
      "path": "internal/example.go",
      "start_line": 10,
      "end_line": 30,
      "expected_sha256": "<64 lowercase hex>"
    },
    {
      "path": "internal/dependency.go",
      "start_line": 40,
      "end_line": 65,
      "expected_sha256": "<64 lowercase hex>"
    }
  ]
}
```

Its successful structured result is:

```json
{
  "input_version": 2,
  "evidence": [
    {
      "path": "internal/example.go",
      "start_line": 10,
      "end_line": 30,
      "body": "...",
      "indexed_sha256": "<64 lowercase hex>"
    }
  ]
}
```

`body` and every evidence field intentionally reuse the scalar response
vocabulary. Batch mode introduces no second evidence representation.

### Validation and atomicity

- Exactly two through four locators are required.
- Unknown or mixed scalar/v2 fields are rejected.
- A locator tuple is unique by `path`, `start_line`, `end_line`, and
  `expected_sha256`; overlapping but non-identical ranges remain valid.
- Every item receives the existing path, symlink, range, file, and hash
  validation.
- The sum of unescaped `body` bytes must not exceed the existing resolved
  `mcp.hard_max_inline_bytes`. JSON metadata and escaping remain outside this
  source-byte contract.
- All evidence is buffered. Any invalid, stale, missing, oversized, or
  aggregate-over-limit item returns a typed error with no evidence bodies.
- An item failure identifies its zero-based `locator_index`.
- Atomicity means no partial response; it does not create a multi-file
  filesystem snapshot. Each expected hash remains the freshness guard.

The batch-specific stable reasons under test are
`INVALID_LOCATOR_COUNT`, `DUPLICATE_LOCATOR`, and `BATCH_TOO_LARGE`.

## 5. Minimal Implementation Boundary

1. Keep scalar application service behavior unchanged and add a buffered
   batch method that delegates each item to it.
2. Keep exactly four registered MCP tools. Add a strict scalar/v2 schema
   branch and matching dispatch only in batch-capable mode.
3. Default the product server and ordinary launcher behavior to scalar-v1.
   Only the existing assistant-evaluation launcher accepts an experimental
   contract-mode argument.
4. Extend the existing runner with per-arm contract mode and preflight both
   schemas. Do not create another runner.
5. Flatten batch evidence units for range/coverage/byte accounting while
   retaining one read action per MCP invocation.
6. Modify existing focused tests only. Do not add a new test suite or test
   framework.

If support code becomes larger than the behavior under test, or requires a
new generalized pipeline, stop and report instead of expanding scope.

## 6. Frozen Comparison

### Arms

- **A — scalar-v1:** trust-priority prompt; scalar `read_span` only.
- **B — batch-v2:** the byte-identical trust-priority prompt; scalar and
  explicit v2 `read_span` available.

Both arms retain ordinary repository tools. Batch use is available, not
forced. Prompt wording, tool count, search contract, and all non-`read_span`
tool descriptions are equal.

### Reused inputs

- The exact 30 question IDs and truths from the accepted awareness/trust V1
  manifest.
- The same 15/15 counterbalanced order, model, reasoning effort, host,
  timeouts, answer schema, and blind grader.
- The same approved Go and TypeScript/TSX corpus bindings and FTS-only SQLite
  bytes.
- `k=5`, zero search source bytes, and the existing 64 KiB read ceiling.

The new manifest records its parent manifest digest and freezes a per-arm
MCP schema/description/functional-probe digest. No scored turn runs before a
clean freeze commit and passing provider-free preflight.

## 7. Metrics and Denominators

Primary quality remains question-level blind outcome, required-group coverage,
and unsupported claims. Thirty tasks or successful tool calls are not quality
proxies.

For the mechanism analysis:

- one `read_span` invocation is one evidence-read round trip;
- each returned member of `evidence` is one evidence unit;
- unique/gross source bytes, overlap, locator exposure, and final citations
  are computed from flattened evidence units;
- paired batch eligibility is fixed from the scalar reference arm: at least
  two distinct, valid locator tuples were exposed before its first successful
  source acquisition. This avoids defining the opportunity from treatment
  adoption; eligibility does not force batch use;
- duplicate and overlapping reads are reported separately; and
- failed, timed-out, non-adopting, and noncompliant tasks remain in their
  stated denominators.

## 8. Retain / Reject Gates

Reject immediately if the batch-capable arm loses blind task quality,
required-group coverage, or unsupported-claim safety.

Retain the branch only when all of these hold:

1. At least two thirds of eligible tasks eliminate at least one evidence-read
   round trip, and the eligible median evidence-read action count is lower.
2. Duplicate/overlap rates do not increase and evidence coverage or unique
   source bytes do not move adversely.
3. Among dual-complete pairs, median total repository actions is below `1.0`
   relative to scalar, median model-total tokens is at most `1.0`, and at
   least two thirds have non-increasing repository actions.

Interpretation is terminal:

- quality regression: reject;
- mechanism improvement without end-to-end action/token improvement: reject
  or defer outside v1, then continue to Phase 12;
- insufficient eligible adoption: reject, then continue to Phase 12; or
- all gates pass: retain the v2 branch, reconcile the stable Phase 13
  contract, then continue to Phase 12.

Do not lower gates or run another orchestration variant after viewing the
result.

## 9. Ordered Execution

1. Reconcile Phase 13/14 status and commit this plan.
2. Implement the bounded contract and focused compatibility checks.
3. Have the main agent inspect the changed paths and run focused validation.
4. Freeze the derived manifest, exact arm contracts, code identities, and
   preflight evidence in a clean commit.
5. Run the 30 paired tasks once, blind-grade with the existing path, and
   aggregate once.
6. Record one result report with artifact digests and a retain/reject
   decision.
7. Reconcile Phase 13 and unblock the next eligible phase. Do not add a new
   assistant-orchestration experiment.

## 10. Completion Evidence

- Scalar schema, output, and typed-error compatibility.
- Batch count, mixed-input, duplicate, ordering, stale-item, per-item-size,
  aggregate-size, and no-partial-body checks.
- Exactly four MCP tools and scalar product default.
- Per-arm provider-free launcher/schema/functional preflight.
- Existing focused Go tests, race check, vet/build where directly affected,
  Python compilation, and legacy trace replay.
- Frozen manifest and one-time paired result with explicit denominators.
- Checks actually run, checks not run, remaining risk, and final contract
  disposition recorded separately.

## 11. Decision Log

| Decision | Rationale | Revisit when |
| --- | --- | --- |
| Test batching inside `read_span` | The measured loss is evidence-read round trips after useful locator selection | The terminal experiment rejects it |
| Preserve scalar-v1 | Compatibility and a true control arm require it | Never during this experiment |
| Use 2–4 caller-selected locators | Bounded enough for the existing byte contract while addressing observed multi-read tasks | New independent evidence after v1 |
| Reuse the aggregate hard maximum | Avoid a second budget/config rule | The source-byte contract changes |
| Buffer then fail closed | Partial evidence would make retries and accounting ambiguous | Never for this experiment |
| Evaluation-launcher mode only | A true A/B needs different visible schemas, but an unproven product flag is not justified | The retain gate passes |
| One terminal comparison | Prevent another prompt/orchestration tuning loop | Never on this exposed dataset |

## 12. Terminal Outcome

The one frozen comparison completed on 2026-09-05. The batch-capable arm
passed the round-trip mechanism gate but failed the no-regression and
duplicate/overlap gates. The existing aggregate also exposed a symmetric
grader-contract mismatch for a product-valid 509-line span, so official full
quality and dual-complete efficiency remain `NOT_OBSERVED`. No grade was
changed and no second aggregate or prompt/orchestration variant ran.

The evaluation-only batch implementation was removed and scalar-v1 restored.
See the [terminal result](evidence/phase-13/bounded-multi-locator-result-v1.md),
[independent review](evidence/phase-13/bounded-multi-locator-terminal-review-v1.md),
and [restoration evidence](evidence/phase-13/bounded-multi-locator-scalar-restoration-v1.md).
