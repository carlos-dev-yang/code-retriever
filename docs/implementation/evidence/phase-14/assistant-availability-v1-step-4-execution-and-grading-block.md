# Assistant Availability V1 Step 4 Execution Checkpoint

- Date: 2026-08-22
- Status: `execution_complete_grading_blocked`
- Run: `assistant-availability-chi-rhf-v1-run-001`
- Entry commit: `500fe27`
- Manifest: `testdata/retrieval/assistant-availability-chi-rhf-v1.json`
- Provider action: none; FTS-only and no provider credential
- Promotion authority: none

## Execution result

The frozen 30-pair schedule ran exactly once. All 60 scored turns completed
without timeout, final error, source/state mutation, or control violation. No
turn was retried or replaced.

The treatment exposed the four frozen cidx MCP tools and their exact live
descriptions and schemas, but the assistant did not call cidx in any of the 30
treatment turns. Every treatment answer instead used ordinary repository
inspection. This is an adoption/interface observation. It is not evidence
about cidx candidate quality, selected-source quality, or retrieval efficiency,
because no treatment turn consumed a cidx result.

| Arm | Valid turns | cidx users | MCP calls | Ordinary commands | Input | Cached input | Uncached input | Output | Model total |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| baseline | 30/30 | 0 | 0 | 101 | 2,662,122 | 1,982,464 | 679,658 | 45,921 | 2,708,043 |
| cidx FTS available | 30/30 | 0 | 0 | 104 | 2,786,356 | 2,071,552 | 714,804 | 47,200 | 2,833,556 |

The treatment totals are retained as intent-to-treat observations. They must
not be labelled cidx transport cost or cidx token effect when adoption is zero.
Correctness and paired dual-complete token analysis remain pending blind
grading.

## Frozen post-run artifacts

The ignored local run is under
`.cidx/test/assistant-ab/runs/assistant-availability-chi-rhf-v1-run-001`.
Preparation rebuilt all 60 session traces from raw events and final outputs,
then froze arm-opaque grading packets before any arm key was used for scoring.
No packet answer disclosed `cidx`, `MCP`, or `read_span` usage.

| Artifact | SHA-256 |
| --- | --- |
| `run-manifest.json` | `75ad824999697f2bb3580b0e1d4dfad6397a751959bfcb78d325bfa82562c438` |
| `grading/journey-freeze.json` | `24e024f68e9fc5a3560d0fe605eec603710aa54a04e5b917799b43e9c352461c` |
| `grading/journey-frozen.jsonl` | `b8e06ab2986012294440b3ca8d808545a92dca2f5fec99cd979a134bdb8e784f` |
| `grading/blind-key.json` | `c238f9ae44b51f7437c86e2034ff32954c08f3cee1b91b0319da00c58e1c6199` |
| Go blind packet | `9fca08c9aca3b79f53ae81782f917cc5e887756902ecdd5402b5aacf154c3a00` |
| React Hook Form blind packet | `6415e4d5eed543135aed3780b31369a402f71adbd2f184ee64cd0354ca090055` |

## Blind-grading block

Two grader launches per corpus stopped before a model response and produced no
grade file. The first launch used the committed blind-grade schema
`2d9729b3f38454ac863ec5898bb04148c97813deb17cd19ce8958fecceba0253`.
The structured-output endpoint rejected JSON Schema `uniqueItems`. A second
launch removed only the three `uniqueItems` keywords in a temporary schema
variant, preserving scorer-side uniqueness checks; the endpoint then rejected
`allOf`. The committed schema was restored byte-for-byte after the failed
compatibility probe.

| Failure artifact | SHA-256 | Pre-model error |
| --- | --- | --- |
| Go original-schema events | `c44ba86203aea733bc712becc7a3842973cb525b420e5d0ede33ead0858cfc92` | `uniqueItems is not permitted` |
| RHF original-schema events | `f599a263ccfacefeeb29937a734427c3621016c17c5682f7413a696efd6a1d90` | `uniqueItems is not permitted` |
| Go schema-fix events | `7bd8d62db5574a77566321ce4f8e11bf8a3aa17f0f81665dd746581cc7f5738c` | `allOf is not permitted` |
| RHF schema-fix events | `3aeff8a6ebb88f6412d304d1275965c34ce35faac439f1535eb5877c992fde2a` | `allOf is not permitted` |

These are format negotiation failures, not blind-grade results. Repeating the
same grader path is stopped. No blind key was joined to a grade and no
aggregate or correctness result exists.

## Required decision and recommended continuation

The recommended continuation is a dedicated response-format schema for blind
grade v2. It should have a single v2 root object and use only the structured-
output-supported subset, omitting the historical v1/v2 union, `allOf`, and
`uniqueItems`. The canonical committed schema and the scorer's fail-closed
semantic validators remain authoritative for group uniqueness, claim/support
rules, and complete 60-entry coverage. The output-only adapter and both hashes
must be recorded before one new grader model call per corpus.

The alternative is unstructured JSON generation followed by local validation,
but that weakens fail-closed generation and is not recommended. No further
grader call should run until this choice is accepted.

## Checks actually run

- 60/60 required final, observation, event, and session-trace artifacts exist.
- 60/60 scored observations are valid; timeout, final-error, and control-
  violation counts are zero.
- treatment cidx use and MCP-call counts are both zero.
- blind preparation completed and deterministically rebuilt the stored traces.
- blind packets contain 20 Go and 40 React Hook Form entries and do not expose
  arm/tool-use wording.
- two preserved structured-output compatibility attempts failed before model
  output; no grade file exists.

## Checks not run

- blind answer grading;
- paired correctness, required-group, or material-claim aggregation;
- dual-complete paired token analysis;
- final Step 4 result review or promotion interpretation.
