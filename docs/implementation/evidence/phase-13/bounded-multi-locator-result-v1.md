# Bounded Multi-Locator `read_span` V1 Result

- Date: 2026-09-05
- Phase: 13
- Run: `assistant-read-span-multi-locator-chi-rhf-v1-run-001`
- Status: `COMPLETE_REJECT_BATCH_V2`
- Stable product disposition: retain scalar-v1 and remove the evaluation-only
  batch branch
- Promotion inference: `NOT_APPLICABLE`

## Result in one paragraph

The bounded request proved its narrow mechanism but failed the frozen terminal
contract. The batch-capable arm used a real v2 request on 19/30 tasks and cut
`read_span` invocations from 132 to 75, total cidx calls from 239 to 187, and
all repository actions from 263 to 203. It eliminated at least one evidence
round trip on 21/30 reference-defined eligible tasks, with paired median read
and repository-action differences of `-2`. Paired model-total ratio was
`0.983`. However it delivered slightly more evidence units and unique source,
and duplicate/overlap rates rose. One individually valid blind pair also
regressed from complete to partial because the batch arm made an unsupported
repository-wide negative-search claim. The full grade aggregate did not
complete because both arms of one separate task exposed the same pre-existing
500-line evaluator mismatch. Since v2 may be retained only when every gate
passes, it is rejected; this run does not authorize another prompt or
orchestration variant.

## Execution and grading accounting

All 60 primary cells ran exactly once and were valid; none timed out or was
retried. The scorer then froze 60 arm-blind entries and deterministic journeys.
One blind grader call returned 20 Go rows and one returned 40 TypeScript/TSX
rows. Both files had the exact expected blind-ID sets and no grader tool use.

The one permitted aggregate invocation stopped before writing an aggregate,
paired-result file, or generated report:

```text
ScoreError: invalid required-group grade for blind-de857ef759a2
```

Read-only first-loss diagnosis found two rejected rows, both for
`avail-tsx-multihop-field-array`, one per arm. Each primary answer cited the
complete `useFieldArray` range at lines 47–555. Product `read_span` is
line-cap-free under its byte ceiling, but the existing blind packet marks a
citation over 500 lines as invalid. The blind grader correctly classified the
group as `invalid_evidence` and identified evidence index 3; the frozen local
validator rejects every invalid source index even when the group status is
`invalid_evidence`.

This is a pre-existing evaluator contract mismatch, not evidence that either
MCP contract returned the wrong source. The two raw rows both intended the
same partial result, so the mismatch is symmetric, but full-run answer quality
remains `NOT_OBSERVED` under the frozen aggregate. The grade files were not
edited, normalized, resubmitted, or selectively recalled, and no second
aggregate was run.

## Provider-free mechanism result

These trace measures are descriptive and do not substitute for the failed
official quality aggregate.

| Measure | Scalar-v1 | Batch-v2 capable |
| --- | ---: | ---: |
| Tasks with a real batch invocation | 0/30 | 19/30 |
| cidx searches | 107 | 112 |
| `read_span` invocations | 132 | 75 |
| Delivered evidence units | 132 | 137 |
| Batch invocations | 0 | 34 |
| Total cidx calls | 239 | 187 |
| All repository actions | 263 | 203 |
| Ordinary discovery actions | 13 | 10 |
| Combined unique source bytes | 163,062 | 166,980 |
| Ordinary reacquired cidx source bytes | 9,016 | 1,695 |
| Exact duplicate evidence units | 8/132 (6.1%) | 12/137 (8.8%) |
| Overlapping successful evidence units | 8/132 (6.1%) | 13/137 (9.5%) |

All 30 scalar tasks met the predeclared reference eligibility rule: at least
two distinct valid locators were available before the first successful source
acquisition. The rule does not require every exposed locator to be read later;
11 tasks later read only one of those initially available locators. The
batch-capable arm eliminated one or more evidence round trips on 21/30 and the
eligible median read difference was `-2`.

Duplicate and overlap numerators above are computed within each task. They do
not count reuse of the same source range by separate questions, which is not a
duplicate acquisition within one assistant journey.

Across all 30 operationally comparable pairs:

- repository-action ratio median: `0.730`; difference median: `-2`;
  non-increasing: `27/30`;
- model-total token ratio median: `0.983`; non-increasing: `16/30`;
- unique-source ratio median: `1.000`; non-increasing: `21/30`;
- summed model-total tokens: `4,045,297` scalar versus `4,139,265` batch;
  and
- summed unique source grew by 3,918 bytes (`+2.4%`).

The core effect is therefore real and narrow: batching reduced evidence
acquisition round trips and ordinary fallback reacquisition. It did not reduce
the number of distinct evidence units, it added five searches, and it did not
prevent duplicate or overlapping selection.

## Blind-quality stop

Of the 58 grade rows that independently passed the frozen semantic validator,
one pair converted partial to complete and one pair regressed complete to
partial. The regression was `avail-ts-negative-local-storage`: the scalar
answer was complete, while the batch-capable answer added an uncited claim
about a repository-wide literal-search result and was partial with one
unsupported claim. This individually valid regression is sufficient to fail
the predeclared no-regression gate even though the raw, non-authoritative
30-row outcome counts happened to be symmetric.

## Terminal gate disposition

| Gate | Result | Evidence |
| --- | --- | --- |
| Quality preservation | **Fail** | One valid complete-to-partial regression; full-run aggregate also unavailable because of the symmetric evaluator mismatch |
| Eligible round-trip reduction | Pass | 21/30 eliminate at least one; median read difference `-2` |
| Duplicate/overlap and evidence volume | **Fail** | Duplicate and overlap rates increase; total evidence units and unique source increase |
| End-to-end actions/tokens | Descriptively passes, formally incomplete | All-pair action median `0.730`, token median `0.983`, actions non-increasing 27/30; frozen dual-complete aggregate unavailable |

The retain rule is conjunctive. Passing the mechanism surface cannot override
the quality and evidence-discipline failures. The v2 branch is rejected and
scalar-v1 remains the stable four-tool contract.

## Artifact seal

Ignored local artifacts remain the detailed authority.

| Artifact | SHA-256 |
| --- | --- |
| Run manifest | `c89633974c608500ad620b410629b91597272a96233707b8d70b82ae8fb74bba` |
| Frozen journey | `2f5d1f9e0a33991599e8a23ed2646803ae53675d0c41134f652dc9fb6fb55ccd` |
| Blind key | `f8f194b197bdedd7a596fea58f0fafab6e48d02a69f1a8a53589cd05d08221cd` |
| Go packet / grades | `3d8cfcfb7c3f8b6a51b7efc8ee9ef4d220eefaf203222144f9e461e4dc78e1b3` / `fb11121645c0b8c1c4b43d7cd559138e5a42687fed855d06435ca37b804ceb80` |
| RHF packet / grades | `25681be8b5aafe1296d1b9df8d488979a43b8a3091f813262ecb424dc5d9bb5d` / `85d47664a3b6a2c8dcd8378988893f96bdf6937eeeb0b76aeb0fdaf6fb8a7e96` |
| Final 449-file run inventory | `099dffafc5a6753bbd4036878b29bd610921675657600eb7b2f17e1e9b81a660` |

## Handoff

The evaluation-only batch-v2 implementation was removed in `ff9d4e8` while
the plan, frozen manifest, raw artifacts, and this result remain preserved.
Focused scalar MCP tests, race/static/build checks, and byte-equivalent replay
of all 60 historical scalar policy traces passed. Phase 13 is complete again;
the exact restoration evidence is in
[`bounded-multi-locator-scalar-restoration-v1.md`](bounded-multi-locator-scalar-restoration-v1.md).

Before any future assistant evaluation, prospectively reconcile the
line-cap-free product source contract with the grader's 500-line excerpt
validator. Do not apply that correction to this closed run. No further
assistant prompt/orchestration experiment is authorized here; the next product
dependency is the owner-gated Phase 12 confirmation corpus and freeze.

The arithmetic, denominators, terminal gate application, and minimal handoff
were independently rechecked. That review also limits causal language to the
batch-capable treatment: this experiment does not prove that batching caused
the isolated unsupported claim. See
[`bounded-multi-locator-terminal-review-v1.md`](bounded-multi-locator-terminal-review-v1.md).
