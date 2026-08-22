# Assistant Blind Grade Strict Output Adapter V3

- Date: 2026-08-23
- Status: `endpoint_probe_passed_ready_for_new_run_freeze`
- Scope: passive blind-grade output structure and future packet-prompt wording only
- Provider action: none
- Promotion authority: none

## Boundary

`assistant-blind-grade-v2-output.schema.json` remains frozen and unchanged.
The canonical `assistant-blind-grade.schema.json` and
`load_grades`/`validate_passive_grade` remain the semantic authority. This
revision adds a separate generation adapter for a later qualifying call; it
does not modify any ignored, frozen packet, prompt, journey, grade, run, or
trace artifact.

## Strict adapter

`schemas/evaluation/assistant-blind-grade-v2-strict-output.schema.json` uses
the previously accepted structural keyword subset (`type`, `properties`,
`required`, `additionalProperties`, and `items`) plus `enum` constraints. It
requires:

- root `schema_version` to be the integer `2`;
- each grade `outcome` to be `complete`, `partial`, `incorrect`, or
  `ungradable`;
- each required-group `status` to be `covered`, `missing`, or
  `invalid_evidence`; and
- each material-claim `classification` to be `observed`, `derived`, or
  `unresolved`.

The adapter deliberately contains no `allOf`, `oneOf`, `$ref`, or
`uniqueItems`. Those semantic relationships—including uniqueness, group
coverage, evidence-index validity, required support for observed/derived
claims, and claim-list consistency—remain fail-closed in the unchanged local
scorer.

| Artifact | SHA-256 |
| --- | --- |
| Strict v2 output adapter | `0bda4226b9fba3b82e3d7503ab83acfaf9ae8c8cea8e9f81f6124d97244261cd` |

## Future prompt generation

`scripts/score-assistant-ab.py` now names the two independent versions when it
generates a future passive blind-grade prompt:

- `PACKET_SCHEMA_VERSION=1` labels the input grading packet; and
- `GRADE_ENVELOPE_VERSION=2` requires the new output grade-result envelope.

This generation-only wording does not rewrite existing ignored packets or
their frozen prompts. It does not change non-passive v1 prompt generation.
The closed Availability V1 grading attempt is not called again; this revision
is an input to the independent neutral-cidx versus directed-cidx run.

## Checks actually run

- Parsed the frozen adapter, strict adapter, and canonical grade schema as
  JSON.
- Audited the strict adapter recursively: it contains only the previously
  accepted structural keywords plus `enum`; no `allOf`, `oneOf`, `$ref`,
  `uniqueItems`, or `const` appears.
- Removed only `enum` values from the strict adapter and confirmed structural
  equality with the frozen v2 output adapter.
- Confirmed the strict root version, outcome, group-status, and claim-
  classification enums exactly align with the canonical v2 contract.
- Ran `python3 -m py_compile scripts/score-assistant-ab.py`.
- Verified the future passive prompt contains both named version constants and
  that packet generation still writes `PACKET_SCHEMA_VERSION`.
- Ran one isolated, unscored Codex CLI structured-output probe with plugins,
  remote plugins, apps, MCP apps, and MCP servers disabled. The endpoint
  accepted every enum constraint and returned root version 2, outcome
  `complete`, group status `covered`, and claim classification `observed`.
- Inspected probe events and stderr; no tool, command, MCP, OAuth, or file-
  access event occurred.
- Ran `git diff --check`.

| Endpoint probe artifact | SHA-256 |
| --- | --- |
| Events | `c9ee53ac05fe5d723709a52d66f98cb44749351181648e3b54b6821a616ae50b` |
| Final JSON | `a999ea404759384828ce04ce1c9dfabcc0b590cf04295b129a1d2c81ddf04625` |
| Stderr | `94b68a7219b9773fb04d88fb1fb63dc3f732540d061bb5ed9ad0f6ccb6fef037` |

## Checks not run

- The optional Python `jsonschema` package is not installed, so no separate
  third-party JSON Schema engine ran; no dependency was installed or network
  action taken. The recorded direct audit checks the exact adapter subset and
  its canonical enum alignment instead.
- no blind grade, semantic validation, or aggregation;
- no runner/trace-arm generalization;
- no provider, corpus, or source-repository operation.

## Handoff

Freeze this adapter hash into the independent neutral-cidx versus directed-
cidx manifest. Use it only for that new run's blind grades; do not call the closed
Availability V1 graders again. Preserve raw output byte-for-byte and send it
to the unchanged canonical scorer. A schema-shaped response is not semantic
acceptance, and a failed qualifying call must be preserved without adapter
revision or retry.
