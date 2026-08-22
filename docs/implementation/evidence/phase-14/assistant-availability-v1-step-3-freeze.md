# Assistant Availability V1 — Step 3 Freeze Evidence

- Phase: 14, Step 3
- Status: complete; executable freeze committed before scored execution
- Frozen at: 2026-08-22T12:29:05Z
- Scored assistant turns observed while authoring or reviewing: none
- Provider activity: none; FTS-only preflight with no `VOYAGE_API_KEY`
- Promotion scope: none; calibration and product-direction evidence only

## Outcome

The host-decided cidx availability experiment is frozen as a new versioned
series. It compares the same Codex CLI task with and without the four cidx MCP
tools. The treatment prompt does not mention cidx, require an initial cidx
call, reward tool adoption, or impose a cidx-specific stopping rule. A
treatment turn that does not call cidx remains in the 30-pair intent-to-treat
denominator.

The executable set contains 30 questions and 60 future scored turns:

| Slice | Questions | Shape distribution |
| --- | ---: | --- |
| Go / chi | 10 | 2 exact, 2 semantic, 2 multi-hop, 2 contract, 1 ambiguous, 1 verified absence |
| TypeScript / react-hook-form | 10 | 2 exact, 2 semantic, 2 multi-hop, 2 contract, 1 ambiguous, 1 verified absence |
| TSX / react-hook-form | 10 | 2 exact, 2 semantic, 2 multi-hop, 2 contract, 1 ambiguous, 1 verified absence |

The fixed schedule has 15 baseline-first and 15 cidx-first pairs. Each arm
receives a fresh source copy and fresh model context. Treatment cidx state is
also copied per turn and remains outside the assistant working directory.

## Model-visible tool contract

The server still exposes exactly `read_span`, `reindex`, `search`, and
`status`. No fifth tool, input-shape change, output-payload change, retrieval
change, or SQLite change was introduced. The descriptions now state the
actual selection workflow and field contracts:

- `search` accepts natural language, identifiers, or path-oriented queries and
  returns only nine-field ranked locators, never source text;
- `read_span` consumes a selected locator's path, inclusive line range, and
  `indexed_sha256` and returns the complete validated source range;
- `status` is an optional freshness diagnostic and is explicitly not required
  before ordinary lookup; and
- `reindex` is for source drift, is unnecessary for ordinary lookup, and is
  provider-free.

The runner now binds taxonomy, truth-sidecar, question-source, individual case,
composite question-set, description, input-schema, and functional-output
identities before a turn. Its functional probe performs provider-free
`search`, `read_span`, and `reindex(dry_run=true)` calls and rejects field-set
or locator-identity drift.

The independent code review found that the first implementation only recorded
the frozen tool hashes and did not reject live drift. It also found that the
manifest claimed default `k=10` while both approved evaluation configs actually
use `return_k=5`. Before commit, the runner was corrected to compare all live
tool-contract identities, freeze and byte-check each corpus config and SQLite
index, compare the complete search/MCP blocks, and enforce the effective
default `k=5` plus 64 KiB read maximum. Negative probes confirmed that a changed
description hash or config hash is rejected before run creation.

Final tool-contract identities:

| Contract | SHA-256 |
| --- | --- |
| Complete tool definitions | `a94df8cdf396b187d5cf6cfe4f8422b1fdd702a909f5fc2b9bda63411f4b0b69` |
| Descriptions | `2a7d3897b1438de3e45310d37c0d03052bb7f4ad4264da17ff7ade006ad9e7c9` |
| Input schemas | `c06571f982f38f1e274d79d1e1c3d46624066d56e98c8e73f441e59bd358b27d` |
| Functional output probe | `6441f100d0fb69d558011314ae9aa34076b1fb0823d604edfbb5651379c4a266` |

## Source-first review and corrections

Questions and truth were authored by direct inspection of the approved pinned
sources. Parser inventory was used only after authoring to encode exact parent
symbols, byte ranges, line ranges, and file hashes. No cidx rank, candidate,
score, prior assistant answer, or scored outcome informed the set.

Two independent high-reasoning reviewers checked every case against source.
The first two rounds found and caused correction of:

- an overbroad `Form` error claim and omitted accepted-response callback;
- missing `matchAcceptEncoding` evidence;
- a TypeScript multi-hop question whose wording did not require its dependency
  groups;
- omitted external-control replacement behavior in `useForm`;
- omitted nested control context in `FormProvider`;
- omitted inherited `UseFormStateProps` evidence;
- two undeclared taxonomy modifiers;
- an undocumented composite question-set hash;
- stale taxonomy hashes inside both question-source files; and
- a `Form` claim that depended on unstated `handleSubmit` semantics. The claim
  was narrowed to behavior directly established by `module.Form`, avoiding a
  56 KiB helper parent that the question did not need.

Both final reviewers then returned `ACCEPT` with all 30 cases, 44 source spans,
three repository-wide absence claims, case/file/composite digests, neutral
prompt, and 15/15 schedule verified. Review artifacts retain every round:

- final Sol: `6587f8e130f389592a2853a65c7cd62b16e6d94bf7fbff3621964e930b46ef23`;
- final Terra: `6af0c312c30ae9b933f4437370c041bdba620267dcb86e2527169cfe76ea1535`;
- owner adoption: `b49a42433d3a7a846e924cccecacc029f39639afbd8df73817c54b5975289e51`.

Authority is explicitly `OWNER_ADOPTED_DUAL_AI_REVIEW` with
`NO_INDEPENDENT_HUMAN_REVIEW`; it must not be described as human review.

## Frozen identities

| Artifact | SHA-256 |
| --- | --- |
| Evaluation design | `513b8f1bb3fb67d4698ccc82678bcd075582db247fb8c8ef133adcddaf7d7cc7` |
| Taxonomy | `7989919031881a6a44062a52304dd14eb1955883a508fe47bda2c128d04bd986` |
| Go question source | `b773a57540d07283463859f6824990aa2308119ad6490dadb9f399326fdbc737` |
| TypeScript/TSX question source | `229cf8588cf5bb9ad6c4f2dce3fe26d1b3839f4c6e931a063e0df3956117da9c` |
| Assistant-invisible truth sidecar | `d73cd8aeeeda9a549744560ba6814e1a91abc6e50ea2ec32ae71351745a11e57` |
| Composite question-set identity | `c0cece49865c0d27773129ce373b801ed0187a3df830a1867dff6935d9a72947` |
| Executable manifest file | `a5b8367c9275a71526e75f2f8ec1cbf8db0ea8ba296f6ec2a6ab7c5bc09282a7` |
| Runner | `f0fcce8420584f62a32f2dd6c082cc2f3fac5c77fb94a719a77af913101063bf` |

## Checks run

- Direct source inspection of every required claim and all reviewer findings.
- Repository-wide non-test scans for JSON auto-binding in chi, browser-storage
  persistence in react-hook-form, and WebSocket/EventSource behavior in
  react-hook-form.
- Evaluation-contract validation and independent digest reproduction for all
  10 Go and 20 TypeScript/TSX cases in draft and frozen states.
- `go test ./internal/mcp` after the model-visible description changes.
- Rebuilt the cidx CLI and isolated MCP launcher.
- Final provider-free manifest, corpus, index, tool-list, input-schema, output-
  field, locator/read identity, and dry-run reindex preflight.
- Positive and negative freeze checks: the exact tool/config state passed,
  while an in-memory description-hash drift and config-hash drift were both
  rejected before execution.
- One unscored answer-schema probe per arm. Both returned the exact requested
  object, made zero repository or MCP calls, and completed successfully under
  Codex CLI `0.149.0-alpha.4.1`.
- JSON parsing, Python compilation of the runner/reducer/scorer, formatting,
  and whitespace checks.

## Checks not run and next boundary

No scored assistant task, blind grade, aggregate comparison, paid embedding,
hybrid query, new corpus, broad release matrix, push, or deployment ran in
Step 3.

Step 4 may start only after this freeze is committed. It must execute the
frozen 60 turns once, preserve no-use and failed outcomes, complete arm-blind
grading before joining journey data, and report correctness, evidence quality,
tool adoption, exploration volume, redundant/post-sufficient work, and paired
official token deltas without a weighted total score.
