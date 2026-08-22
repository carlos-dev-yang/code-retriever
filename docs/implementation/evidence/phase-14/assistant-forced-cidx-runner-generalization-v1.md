# Forced cidx Prompt Diagnostic Runner Generalization

- Date: 2026-08-23
- Phase: 14, non-promotion prompt-policy diagnostic preparation
- Scope: `scripts/run-assistant-ab.py` only
- Product retrieval, MCP schema, scorer, manifest, and status change: none

## Implemented runner contract

Existing manifests without an `arms` member retain the historical
`baseline`/`cidx_fts` behavior, command construction, arm order, state-copy
rules, and passive-v1 trace identity.

A new prompt-policy manifest may instead supply exactly the two arms
`neutral_cidx` and `directed_cidx`, as either an ordered array of objects or
an object keyed by those IDs. Every arm declares:

```json
{
  "id": "neutral_cidx",
  "cidx_exposed": true,
  "prompt_suffix": "",
  "mcp": {"server_name": "cidx", "approval_mode": "approve"}
}
```

The manifest's top-level `prompt_policy` must declare
`neutral_arm_id="neutral_cidx"`, `directed_arm_id="directed_cidx"`, and the
SHA-256 of the directed suffix in `directed_suffix_sha256`. The neutral suffix
is exactly empty, the directed suffix is non-empty, and both cidx MCP objects
plus their resolved server/approval values must be canonical-identical. The
runner renders the base task prompt plus only the arm suffix and proves for
every task that removing the exact directed suffix produces the neutral
rendered prompt.

Both arms expose the frozen cidx server. Each receives a fresh copied source
worktree and an independent copied cidx state; the before/after database
digests remain in its observation. The runner probes the output schema once
for each arm. It records arm IDs, suffix and rendered-prompt hashes, the
per-task removal proof, and the trace-builder identities in the new run
manifest.

Manifest-defined arms require the `forced-cidx-policy-v2` protocol; that
protocol rejects a legacy asymmetric `baseline`/`cidx_fts` manifest. When
declared at top-level, or in `freeze` / `freeze.trace_builders`, the frozen
`policy_trace_builder_sha256` and `passive_trace_builder_sha256` values must
match their live modules before preflight proceeds. The policy run manifest
records both the policy module and its delegated passive-v1 module identities
and hashes.

For policy schedules, `schedule.pair_count`,
`schedule.neutral_first_pairs`, and `schedule.directed_first_pairs` are
required. They are fail-closed at the frozen values `30`, `15`, and `15`, and
must match the actual task and first-arm counts. Task IDs and sequences must be
unique, sequences must be contiguous, and each task resolves to exactly the
two distinct policy arms.

The policy trace protocol uses the separately owned
`assistant_session_policy_trace.py` module. Its absence is a clear fail-closed
execution error; passive-v1 continues to use only its frozen builder. Policy
compliance belongs to the policy trace and is recorded as an outcome, not used
by the runner to discard or retry a valid execution. In particular, a
`shell_cidx_attempt` is retained as policy noncompliance while legacy runs
retain their existing invalidating-control behavior.

The policy manifest must also freeze `execution_code.runner` and
`execution_code.scorer` path/SHA-256 identities. Preflight verifies both live
files and records the identities in the run manifest; the scorer independently
rechecks both files and requires the run's runner hash to equal the freeze.

## Focused checks run

- `python3 -m py_compile scripts/run-assistant-ab.py scripts/assistant_session_trace.py scripts/assistant_session_policy_trace.py`
- `python3 scripts/run-assistant-ab.py --help`
- an in-memory runner probe confirming legacy `baseline`/`cidx_fts` resolution
  and command construction; valid strict policy-arm resolution, suffix-removal
  equivalence, MCP equality, schedule counts, and dual trace identity; and
  fail-closed rejection of wrong arm IDs, suffix/hash/config/MCP mismatches,
  asymmetric protocol selection, duplicate/noncontiguous schedules, pair-count
  and first-arm-count absence or mismatches, invalid pair construction, and
  frozen policy or passive trace-hash mismatches
- `git diff --check -- scripts/run-assistant-ab.py`

## Checks not run

- No assistant/model execution, schema-probe invocation, corpus preflight,
  scorer run, provider request, test-code addition, or full-project validation.
- No manifest, schema, scorer, status, or trace-module edit was made by this
  bounded runner change.
