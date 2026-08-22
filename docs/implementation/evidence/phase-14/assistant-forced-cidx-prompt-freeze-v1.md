# Forced cidx Prompt Diagnostic V1 — Freeze Evidence

- Date: 2026-08-23
- Phase: 14
- Scope: 30-pair, 60-turn, provider-free, non-promotion prompt diagnostic
- Manifest: `testdata/retrieval/assistant-forced-cidx-prompt-chi-rhf-v1.json`
- Preflight artifact: ignored local state at
  `.cidx/test/assistant-ab/runs/assistant-forced-cidx-prompt-v1-preflight-final-seal`

## Frozen comparison

The existing 30 questions, question digests, truth, corpora, source commits,
tree hashes, prompt template, cidx indexes, retrieval configuration, Codex
model/reasoning settings, answer schema, MCP result representation, and four
model-visible tool definitions are unchanged from the closed availability
manifest.

Both arms expose cidx with canonical-identical MCP configuration:

- `neutral_cidx`: empty suffix, SHA-256
  `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`;
- `directed_cidx`: exact design paragraph, SHA-256
  `d2f2e0150569473b284648a699bfec487514759462eed678f47a3ce625fe2352`.

For all 30 tasks, removing the exact directed suffix produces the neutral
prompt byte for byte. The schedule is 30 pairs with 15 neutral-first and 15
directed-first tasks. Noncompliance is retained, never retried or excluded.

## Bound implementation

- policy trace:
  `2174d948e2282d9ffd2f4f1c6bac29736b1424dff4217d169a31b5aee9db2d6a`
- delegated passive trace:
  `12ca918216a7e6767de5d33b1c41d48f9b412745d19798aa485c72e78ab72410`
- runner:
  `5166077ff51238ac2db30043cac779fbf99d7b0b6675a0a43b6553e8d7d48cab`
- scorer:
  `235a0bd7a8347affeadbae21e2845fe8bbf676066aa96467d29bb6eb2a7d2786`
- strict output adapter:
  `0bda4226b9fba3b82e3d7503ab83acfaf9ae8c8cea8e9f81f6124d97244261cd`
- canonical blind-grade schema:
  `2d9729b3f38454ac863ec5898bb04148c97813deb17cd19ce8958fecceba0253`

Runner preflight validates the live runner/scorer and both trace builders.
The scorer independently revalidates them and requires the run's runner hash
to match this freeze.

## Seal preflight

The final external-review manifest hash was
`e5beaa8b4d1fcbd26f1c4f0e443a8ba8a307c14d6c532a504eec58e244203023`.
After the independent reviewer returned `ACCEPT`, the only manifest changes
were the status, freeze timestamp, and review disposition. The executable
manifest SHA-256 is
`764f16fb4e61c4b3a48881223f47df64d161e9dd011428bf609f754cb98d7a9c`.
Provider-free preflight completed successfully with:

- run-manifest SHA-256
  `8ca8b24c3040914d4c2bc7e21a1c11ca3ea4d9dbb23f7b6c44ae82236357a26a`;
- tool-schema SHA-256
  `8a94f0f73810b562016c7f073d3bbcdcd0c9a9b5a664d5c29032cd3229601153`;
- tool-definition SHA-256
  `a94df8cdf396b187d5cf6cfe4f8422b1fdd702a909f5fc2b9bda63411f4b0b69`;
- description SHA-256
  `2a7d3897b1438de3e45310d37c0d03052bb7f4ad4264da17ff7ade006ad9e7c9`;
- input-schema SHA-256
  `c06571f982f38f1e274d79d1e1c3d46624066d56e98c8e73f441e59bd358b27d`;
- functional-output probe SHA-256
  `6441f100d0fb69d558011314ae9aa34076b1fb0823d604edfbb5651379c4a266`;
- no provider credential observed; and
- both corpus commit/tree/config/index identities matched.

## Review findings closed

The independent code review initially found missing sole-intervention checks,
protocol/arm coupling, event-identity mapping, ambiguous command handling,
chronology, delegated trace identity, schedule validation, execution-code
identity, stale metadata, and wrapper-form shell cidx detection. Each finding
was reproduced or inspected, corrected, and rechecked before this freeze.
The final independent review returned `ACCEPT` with zero remaining blocking
findings after verifying mandatory dual trace roles, live execution identities,
suffix-only intervention, equal MCP controls, 30/15/15 scheduling, and wrapper
shell-cidx handling.

## Checks actually run

- Python compilation and CLI help for runner, trace, and scorer.
- Repository whitespace check.
- Byte-equal legacy `baseline`/`cidx_fts` Codex command construction.
- Read-only V3–V6 scorer-context replay.
- Policy fail-closed probes for arm IDs, MCP deltas, protocol mismatch,
  suffix/removal mismatch, execution hashes, and schedule omissions/mismatches.
- Focused policy-trace probes for incomplete calls, exact event identity,
  command ambiguity, source-path attribution, chronological locator selection,
  overlap versus reacquisition, body-free output, and direct/wrapped shell cidx.
- Real frozen-corpus/MCP preflight over chi and React Hook Form.

## Checks not yet run

- No new schema-probe Codex turn or scored assistant turn.
- No blind grader call, packet preparation, or aggregate.
- No Voyage/provider request and no paid embedding/query action.

The next permitted sequence is: change only the manifest status/freeze seal,
commit it, rerun frozen preflight and two schema probes, then execute the 60
turns exactly once if those probes pass.
