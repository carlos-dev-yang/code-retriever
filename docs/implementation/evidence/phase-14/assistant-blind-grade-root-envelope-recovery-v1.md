# Assistant Blind Grade Root Envelope Recovery V1

- Date: 2026-08-23
- Status: `frozen_before_qualifying_calls`
- Protocol ID: `blind-grade-v2-root-envelope-recovery-v1`
- Run: `assistant-availability-chi-rhf-v1-run-001`
- Scope: rejected generation-format attempt and symmetric recovery
- Promotion authority: none

## Mechanical failure

The first real Go call under the output-adapter freeze returned the required
root fields, expected corpus ID, 20 rows, and v2 grade-row fields, but copied
the input packet's `schema_version: 1` into the output envelope. Passive-trace
grading requires a v2 envelope. The unchanged scorer therefore rejects the
document before semantic scoring.

The collision is visible in the frozen input: its root version identifies the
grading **packet** format, while the generation prompt did not explicitly
distinguish that number from the required grade-result envelope version. This
is a generation-contract defect, not a grade outcome.

No React Hook Form call and no aggregate command ran after detection. The Go
files remain byte-for-byte evidence and are classified
`REJECTED_BEFORE_SEMANTIC_SCORING`; their grade rows cannot enter a score.

| Rejected Go artifact | SHA-256 |
| --- | --- |
| Raw output | `9831a588bb67024c2d6b43d0161a1844e5d4fafd4f5102b007e1da7601e00a9f` |
| Events | `44db62c2a6645ef6064d32703ed16021f8d022f286c9aa3cbacb38c71451dc75` |
| Stderr | `7fa3a2f4c0f19ca443efd39ad6bb4938a6073d21c0351daef33a89c6cc57114e` |

The host's immediate structural inspection also printed the rejected outcome
counts before this recovery freeze. Those counts are not used to choose or
alter the recovery instruction. The new instruction changes only the root
version distinction, is applied identically to both corpora, and the next
qualifying outputs are accepted or rejected solely by the precommitted checks
below regardless of whether their outcomes improve or regress.

## Frozen correction

The original packets, prompts, blind key, journey, canonical schema, output
adapter, scorer, model, reasoning effort, and isolation settings remain
unchanged. One exact 494-byte wrapper is appended after each frozen prompt:

```text
OUTPUT ENVELOPE INVARIANT — APPLIES ONLY TO YOUR RESPONSE:
The GRADING PACKET above is input data. Its `schema_version: 1` identifies the packet format; it is not the format version of your response. Return a new grade-result object with exactly the root fields `schema_version`, `corpus_id`, and `grades`. The response `schema_version` MUST be the JSON integer `2`. Copy `corpus_id` exactly from the input packet. Do not copy the packet object or its `schema_version: 1` into your response.
```

| Input | SHA-256 |
| --- | --- |
| Wrapper | `9aa1937b9617fafea1edfafb71544ea760bce116b73e94a9675b317eb9ac3791` |
| Frozen Go prompt | `7ae089732ff60a924644a8211b373c0df2f8c7a62aa661d7a849eed93af250ad` |
| Combined Go prompt | `24799dcc7d73733a77473e77e4619185e4b5b32f167c2aa3996a08efc234e2bc` |
| Frozen React Hook Form prompt | `f1b7f9b1adc8effbaea4753df29e2b1571d3979d893a3d9a8715c699fd186a1b` |
| Combined React Hook Form prompt | `f3012ccafa88f09442448f95a9f11b7b40e3b564844157ae4874624c0b2ce2f3` |

The wrapper contains no outcome, evidence, group, claim, arm, retrieval,
ranking, token, or tool-use instruction. The output adapter stays on its
endpoint-proven structural keyword subset; the local scorer remains the v2 and
semantic authority.

## Qualifying-call rule

After this freeze, make exactly one qualifying call for Go and exactly one for
React Hook Form using the same wrapper and isolated CLI configuration. The
rejected Go call is not a qualifying grade. There was no earlier React Hook
Form model output.

For each new raw output, the host checks only:

1. exact root key set `schema_version`, `corpus_id`, and `grades`;
2. JSON integer `schema_version == 2`;
3. exact expected corpus ID;
4. exact row count, 20 for Go and 40 for React Hook Form; and
5. no tool, command, MCP, OAuth, or file-access event.

A raw output passing those checks is moved byte-for-byte into its canonical
`grades-{corpus_id}.json` path. No JSON field is edited. After both pass, the
unchanged aggregate runs exactly once and performs full fail-closed validation
of IDs, groups, evidence indexes, material claims, and coverage.

If either qualifying call repeats the envelope error, fails, or is rejected by
the canonical scorer, preserve it and stop. Do not revise the wrapper or call
that corpus again.

## Checks actually run before freeze

- confirmed the rejected Go root is version 1 with the expected corpus and 20
  rows;
- confirmed every rejected row has the v2 field set, without treating that as
  semantic acceptance;
- confirmed React Hook Form was not called and aggregate was not run;
- hashed the rejected artifacts, wrapper, frozen prompts, and exact combined
  prompt byte streams;
- obtained independent grading-procedure and code-boundary review.

## Checks not run

- no qualifying recovery call;
- no canonical grade validation or aggregate;
- no directed-cidx assistant turn;
- no provider embedding or query operation.
