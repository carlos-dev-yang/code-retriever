# Forced cidx Prompt Diagnostic Scorer Generalization V1

- Date: 2026-08-23
- Phase: 14
- Scope: `scripts/score-assistant-ab.py` only
- Experiment class: non-promotion, forced-cidx prompt-policy diagnostic

## Implemented scorer boundary

The scorer preserves the legacy fixed `baseline`/`cidx_fts` path for V3–V6
and passive-v1 availability artifacts. A separate
`forced-cidx-policy-v2` path activates only when that manifest protocol is
declared.

For policy-v2, the scorer resolves the two declared arm IDs from the manifest
rather than the legacy arm constant. It derives the neutral reference as the
sole cidx-exposed arm with an empty prompt suffix and the directed treatment
as the sole cidx-exposed arm with a nonempty prompt suffix. The run manifest
must bind the same ordered arm IDs and the exact
`assistant_session_policy_trace.py` module name, schema version, and SHA-256,
plus the delegated `assistant_session_trace.py` module/schema/SHA-256. The
scorer validates both live files before packet preparation or aggregation.

Before blind packet creation, every stored policy-v2 trace is rebuilt from its
events and final response with the frozen policy trace builder. A byte-different
rebuild, protocol/schema mismatch, or builder-hash mismatch stops processing.
The journey freeze records that trace identity along with its own digest.

Policy-v2 packets contain only the task/question, frozen truth, cited source
excerpts, assistant response, and operational-gradeability marker. They do
not include arm identity, prompt policy, cidx/ordinary tool use, token data,
or execution/search order. The blind key remains outside the packet.

The strict output adapter is retained only as the manifest-recorded generation
identity (`blind_grade_output_adapter.path` and `.sha256`), and its live digest
is checked before packet preparation. It is not used to
accept a grade semantically: schema-version-2
grades still pass the existing fail-closed `validate_passive_grade` checks for
group identity/status, valid citation indices, material-claim support,
uniqueness, and claim-finding consistency.

## Policy-v2 aggregate surfaces

The new aggregate and paired rows report separate, unweighted surfaces for:

- blind answer outcomes, required-group coverage, material claims, and
  paired complete conversions/regressions;
- official input/cached/uncached/output/model-total tokens;
- cidx tool/search/read counts, exact selected-locator reads, read overlap,
  locator/evidence coverage and precision, and source bytes;
- ordinary and ambiguous repository-discovery counts by command family,
  known-file reads, discovery timing, incomplete cidx attempts,
  non-temporal cidx/ordinary overlap, and temporal cidx-to-ordinary source
  reacquisition;
- directed-arm mechanical compliance and every independent failure reason; and
- Go, TypeScript, TSX, and frozen primary question-shape slices.

The result carries `scope=non_promotion_forced_cidx_prompt_diagnostic`,
`promotion_inference=NOT_APPLICABLE`, and `weighted_score=NOT_REPORTED`.
It makes no release, voluntary-adoption, optional-use, or product-role claim.

## Focused checks run

- `python3 -m py_compile scripts/score-assistant-ab.py`
- `python3 scripts/score-assistant-ab.py --help`
- `git diff --check -- scripts/score-assistant-ab.py`
- Read-only scorer-context/grade/journey replay for preserved V3, V4, V5, and
  V6 runs. No aggregate, packet, grade, or report was regenerated.
- Read-only passive-v1 availability context replay. Its retained grade output
  was not accepted or repaired; the recorded noncanonical grade remains a
  semantic rejection.
- In-memory policy-v2 arm-resolution probe for `neutral_cidx` and
  `directed_cidx`, plus a body-free policy-trace rebuild and existing reducer
  read replay over a preserved V6 event/final pair.
- Main-agent follow-up checks revalidated syntax/help/whitespace, required the
  exact neutral/directed arm IDs, bound the delegated trace-builder identity,
  checked the strict adapter digest, and added all-pair token-ratio reporting
  separately from the dual-complete efficiency slice.

## Checks not run

- No assistant/model or blind-grader call, corpus/source operation, provider
  request, schema-probe execution, or new scored turn.
- No runner, trace module, schema, manifest, status ledger, or existing
  evidence artifact was modified.
- No full-project validation, test-code addition, commit, or push.
