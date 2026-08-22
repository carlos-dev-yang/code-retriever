# Assistant Blind Grade Output Adapter V2 Freeze

- Date: 2026-08-23
- Status: `frozen_before_scored_grading`
- Scope: output-format compatibility only
- Provider action: none beyond one unscored structural probe
- Promotion authority: none

## Purpose and boundary

The completed Assistant Availability V1 run could not enter blind grades
because the Codex structured-output endpoint rejected `uniqueItems` and then
`allOf` before either corpus reached model output. The canonical blind-grade
contract and its local fail-closed scorer remain unchanged. This adapter is a
generation-only schema containing the same v2 fields in the endpoint-supported
keyword subset.

The adapter does not relax scoring. The existing scorer still validates the
canonical v2 contract, exact corpus and blind-ID coverage, unique IDs and
groups, required-group membership, evidence indexes, claim classification and
support, and aggregate completeness. A structurally generated response that
violates any of those rules fails locally rather than being repaired.

## Frozen identities

| Artifact | SHA-256 |
| --- | --- |
| Canonical blind-grade schema | `2d9729b3f38454ac863ec5898bb04148c97813deb17cd19ce8958fecceba0253` |
| Output-only v2 adapter | `61ea5b67dbddbb5bdabeb4fab5b0ae662aeb8614c1aac59d105491ce94f46652` |
| Scorer | `7b393f4f52760d47eafa6ac2d55047dea6b53022a6758f8a85a7f988bcc9c538` |
| Run manifest | `75ad824999697f2bb3580b0e1d4dfad6397a751959bfcb78d325bfa82562c438` |
| Journey freeze | `24e024f68e9fc5a3560d0fe605eec603710aa54a04e5b917799b43e9c352461c` |
| Frozen journeys | `b8e06ab2986012294440b3ca8d808545a92dca2f5fec99cd979a134bdb8e784f` |
| Blind key | `c238f9ae44b51f7437c86e2034ff32954c08f3cee1b91b0319da00c58e1c6199` |
| Go packet | `9fca08c9aca3b79f53ae81782f917cc5e887756902ecdd5402b5aacf154c3a00` |
| React Hook Form packet | `6415e4d5eed543135aed3780b31369a402f71adbd2f184ee64cd0354ca090055` |
| Go grader prompt | `7ae089732ff60a924644a8211b373c0df2f8c7a62aa661d7a849eed93af250ad` |
| React Hook Form grader prompt | `f1b7f9b1adc8effbaea4753df29e2b1571d3979d893a3d9a8715c699fd186a1b` |

## Grader execution protocol

Each corpus receives exactly one new grader model invocation after this freeze.
The frozen prompt is provided through standard input. The grader runs from a
new empty directory with Codex CLI `0.149.0-alpha.4.1`, model
`gpt-5.6-sol`, and reasoning effort `high`. User configuration and rules are
ignored; plugins, remote plugins, apps, MCP apps, and all MCP servers are
disabled; the sandbox is read-only; the session is ephemeral. The only model
output contract is
`schemas/evaluation/assistant-blind-grade-v2-output.schema.json`.

No source repository, blind key, arm identity, prior grade, or evaluation
report is exposed to the grader. No tool use, manual grade edit, missing-row
repair, retry, or replacement call is permitted. A transport or model failure
is preserved and stops aggregation for that corpus.

## Compatibility evidence

One unscored synthetic probe exercised the exact output adapter and the final
isolation configuration. It returned the requested v2 object and emitted only
one agent-message event. No tool, command, MCP, file read, or OAuth event
appeared.

| Probe artifact | SHA-256 |
| --- | --- |
| Events | `ab3b2b6173ad95d4509e7c376101c23a84cd4bba6cd2fdca4d7d5310f1f528b7` |
| Final output | `05d469b6d54805b996a199a183655306bc6a18d751024f3fd2b623e0350d7c7c` |
| Stderr | `6a9a5d686f5d67f6106b01986507c6ff28cd9446e72ecc764ab62c662cc9fcd5` |

The probe used 13,603 input and 67 output tokens. Those tokens are adapter
compatibility cost only and are excluded from the assistant A/B evidence.

## Checks actually run

- parsed the canonical and output-only schemas as JSON;
- confirmed the adapter uses only structural structured-output keywords;
- re-hashed every frozen run, journey, blind, packet, prompt, and scorer input;
- ran one unscored output-schema probe with the final isolated CLI settings;
- inspected probe events and stderr for tools, commands, MCP, OAuth, and file
  access; none occurred.

## Checks not run

- no corpus blind grader invocation;
- no local grade semantic validation;
- no availability-run aggregation;
- no directed-cidx assistant turn;
- no paid embedding or query operation.

## Handoff

Commit this adapter and freeze evidence before grading. Then make one new
isolated call for each frozen corpus prompt, preserve raw events, stderr, and
final JSON byte-for-byte, and run the unchanged local aggregate command exactly
once only after both outputs pass fail-closed validation.

## Post-freeze envelope collision

The first real Go call exposed an ambiguity not exercised by the synthetic
probe: the input packet's root `schema_version: 1` was copied into an otherwise
v2-shaped result. It is rejected before semantic scoring and does not amend
this adapter in place. The preserved failure and the symmetric, root-only
protocol correction are recorded in
[Root Envelope Recovery V1](assistant-blind-grade-root-envelope-recovery-v1.md).
