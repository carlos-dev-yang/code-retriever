# Assistant Availability V1 Question-Set Review Packet

- Review stage: pre-execution Step 3
- Scored assistant output observed: no
- cidx search output consulted during authoring: no
- Provider operation: none
- Reviewer role: independent source-first evaluation reviewer

## Files under review

- `docs/implementation/ASSISTANT-AVAILABILITY-EVALUATION-V1.md`
- `testdata/retrieval/assistant-availability-question-shapes-v1.json`
- `testdata/retrieval/question-set-go-chi-v5.3.1-assistant-availability-v1.json`
- `testdata/retrieval/question-set-react-hook-form-v7.85.0-assistant-availability-v1.json`
- `testdata/retrieval/assistant-availability-chi-rhf-v1-truth.json`
- `testdata/retrieval/assistant-availability-chi-rhf-v1.json`
- `schemas/evaluation/assistant-question-set-review.schema.json`

Frozen source candidates are the existing local approved snapshots:

- `.cidx/test/corpora/chi` at commit
  `8b258c7bb28f97a5f2a856ff7ef962578fec9215`;
- `.cidx/test/corpora/react-hook-form` at commit
  `371432c39271aab739358d19c406793771565ab3`.

## Review instructions

Review all 30 questions before any scored run. Use direct source inspection
only. Do not invoke cidx, read its search results, inspect prior assistant run
outputs, modify files, or use the network.

Verify:

1. exactly 10 Go, 10 TypeScript, and 10 TSX questions;
2. the required 2/2/2/2/1/1 primary-shape distribution in every language
   slice;
3. that each question is answerable from its required groups, or is a genuine
   repository-level no-answer case;
4. every required path, file SHA-256, parent byte range, qualified symbol, and
   grade-2 judgment against the pinned source;
5. that each required group is necessary and sufficient for the frozen
   material claims, with no missing dependency evidence;
6. that hard negatives are plausible false leads but do not establish the
   premise;
7. the three verified-absence claims with repository-wide non-test source
   searches;
8. that the shared prompt does not force, reward, discourage, or even name
   cidx and applies identically to both arms;
9. exactly 15 baseline-first and 15 cidx-first pairs, with every case present
   once and question digests matching its source file; and
10. that no question, truth, prompt, or schedule choice depends on an observed
    assistant outcome.

Use severity:

- `P1`: invalidates the comparison or source truth;
- `P2`: materially weakens grading, evidence sufficiency, or a required
  question-shape cell;
- `P3`: nonblocking clarity or maintenance improvement.

Return `ACCEPT` only when no P1 or P2 correction remains. Return
`ACCEPT_WITH_CORRECTIONS` when the listed corrections can make the set
executable without redesign. Return `BLOCK` when source truth or the paired
estimand cannot be repaired within Step 3.

The response must match
`schemas/evaluation/assistant-question-set-review.schema.json` exactly.
