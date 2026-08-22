# Assistant Availability V1 Blind-Grading Closure

- Date: 2026-08-23
- Status: `grading_protocol_invalid_no_quality_result`
- Run: `assistant-availability-chi-rhf-v1-run-001`
- Assistant turns: unchanged, 60/60 valid
- Promotion authority: none

## Outcome

The run's execution and zero-adoption observation remain valid, but blind
answer quality is not available. The output-only grading recovery reached one
v2 envelope per corpus, then the unchanged local scorer stopped on the first
Go required-group row because the model emitted status `satisfied` instead of
the canonical enum `covered|missing|invalid_evidence`.

This is a grading-document contract failure. It is not a low retrieval score,
an assistant answer failure, or evidence that cidx helped or harmed quality.
No aggregate or result report was written.

## Preserved attempts

The initial output-adapter Go call is retained separately because it copied
the packet's root version 1. Its content did not enter scoring. The symmetric
root-envelope recovery then produced root-v2 documents for Go and React Hook
Form with the expected 20 and 40 rows and no grader tool use.

| Artifact | SHA-256 | Disposition |
| --- | --- | --- |
| Initial Go output | `9831a588bb67024c2d6b43d0161a1844e5d4fafd4f5102b007e1da7601e00a9f` | rejected root v1 |
| Initial Go events | `44db62c2a6645ef6064d32703ed16021f8d022f286c9aa3cbacb38c71451dc75` | preserved |
| Initial Go stderr | `7fa3a2f4c0f19ca443efd39ad6bb4938a6073d21c0351daef33a89c6cc57114e` | preserved |
| Recovery Go output | `9ffd8b7f3a1d44c98e1308e9064eec4f207b0d91aa7caf7323818a5e7a8842ff` | rejected canonical status enum |
| Recovery Go events | `1ca46d0cb06f92bda7563ecde4e4eae53ea1a912f5e4b2dab2eefdb59a15b30d` | preserved |
| Recovery Go stderr | `6702fb1133541f0e975009134a5d2c156b46a000050a744351e555131da2ded2` | preserved |
| Recovery React Hook Form output | `d11537e4fe4a08d2fe2f5ad9248f8aa3a4ea9032880068679a6edbaa7ee8896a` | unaggregated because run validation failed |
| Recovery React Hook Form events | `bc607089745a311bebd5b64b287475055a8d6f5669ba321c9fcc5cedb4835e0b` | preserved |
| Recovery React Hook Form stderr | `4242ab688ba6a7ee4a421b513a39a593062c439daedb933eea748de1278dcbee` | preserved |

The recovery calls used 65,165 input / 7,991 output tokens for Go and
157,640 input / 20,929 output tokens for React Hook Form. These are grader
costs, not assistant-arm costs, and are excluded from all A/B efficiency
metrics.

## Exact aggregate stop

The scorer identity was still
`7b393f4f52760d47eafa6ac2d55047dea6b53022a6758f8a85a7f988bcc9c538`.
The aggregate command ran exactly once and stopped before producing an
aggregate at:

```text
ScoreError: invalid required-group grade for blind-5b35b6ac5ef5
```

Inspection limited to contract diagnosis showed that the Go document used
`satisfied` for its required-group status values. The canonical contract and
scorer allow only `covered`, `missing`, or `invalid_evidence`. The document was
not normalized, edited, mapped, or resubmitted.

## Decision boundary for the directed prompt experiment

Do not call either corpus grader again for this availability run. It closes
with answer quality `NOT_OBSERVED` while preserving its 0/30 spontaneous-cidx
adoption result.

The new neutral-cidx versus directed-cidx experiment is an independent paired
run with its own blind key, packets, journeys, and grades. It may proceed after
a new output adapter revision has both:

1. endpoint-proven enum constraints for envelope version, outcome, group
   status, and claim classification; and
2. an unscored representative probe that passes the unchanged local semantic
   validator.

The new run must not reuse, normalize, compare against, or choose between any
grade row from this closed attempt. Its two arms provide their own paired
quality baseline, so the old run's missing quality aggregate is not silently
substituted into the new comparison.

## Checks actually run

- one root-envelope qualifying call per corpus under the frozen recovery;
- exact root key, root version, corpus ID, row-count, and no-tool checks;
- unchanged scorer hash verification;
- exactly one aggregate invocation;
- first-loss diagnosis after the fail-closed exception;
- independent grading-procedure and code-boundary review.

## Checks not run

- no second aggregate;
- no manual grade repair or another grader call;
- no correctness, paired complete, claim-support, or promotion conclusion;
- no directed-cidx assistant turn as part of this closure;
- no embedding provider action.
