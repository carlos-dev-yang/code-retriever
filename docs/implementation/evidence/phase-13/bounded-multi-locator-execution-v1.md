# Bounded Multi-Locator Execution V1

- Date completed: 2026-09-05 (Asia/Seoul)
- Run ID: `assistant-read-span-multi-locator-chi-rhf-v1-run-001`
- Scope: frozen 30-pair scalar-v1 versus batch-v2 execution
- Status: `PRIMARY_EXECUTION_COMPLETE_UNGRADED`

## Execution result

The runner repeated the frozen provider-free MCP preflight and both unscored
Codex schema probes, then executed the 60 scored cells exactly once in the
predeclared alternating order.

- scored observations: `60`
- valid executions: `60`
- exit code zero: `60`
- timeouts: `0`
- scalar-v1 cells: `30`
- batch-v2-capable cells: `30`
- selective retries or replacement cells: `0`

The immutable question text, truth, task order, trust prompt, model, FTS state,
ordinary tools, byte ceiling, and grading contract were unchanged. The runner
verified each isolated source checkout remained clean after its pair.

## Local artifact identity

- Run manifest SHA-256:
  `c89633974c608500ad620b410629b91597272a96233707b8d70b82ae8fb74bba`
- Tool schema artifact SHA-256:
  `bb5b557db70c85e07eb3683643c57a55c72b44778a5e3e4a6b5bce926c060db3`
- Raw run inventory: `436` files
- Sorted file-content inventory SHA-256:
  `e8a4ae4d3ce5276924e8884a15f848beacaa6225d9d4a4b42c2c064b63d0e51d`

The run remains in the ignored local evaluation directory. These identities
bind the next blind-grade preparation to the exact raw run without committing
model transcripts or corpus source.

## Next bounded action

Use the existing scorer `prepare` path, existing strict blind-grade adapter,
and existing canonical grade schema. Do not alter questions, truth, required
groups, or primary answers. After blind grades are complete, run one aggregate
and apply the predeclared terminal retain/reject gates.

This section records the action that followed this immutable primary-execution
checkpoint. The later grading boundary and terminal disposition are recorded
separately in [`bounded-multi-locator-result-v1.md`](bounded-multi-locator-result-v1.md);
batch-v2 was rejected and scalar-v1 restored.
