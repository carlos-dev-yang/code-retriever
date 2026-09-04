# 14. Packaging and MCP Host Integration

- Status: `blocked` — the default-1024/int8 package checkpoint and Phase
  13's structured-only locator search/source-only `read_span` contract remain
  accepted. Assistant V4–V6 is closed without an efficiency claim and is not
  reopened. The owner selected a new host-decided, stage-separated
  search-to-evidence direction; its documentation and 30-question/60-turn,
  FTS-only evaluation design received matching ChatGPT and Grok
  `FINAL_ACCEPT`. The neutral-interface and passive harness trace/reducer
  implementation checkpoint is now complete. Its neutral 30-pair run completed
  with zero cidx adoption. The forced prompt diagnostic and its restored
  awareness-versus-trust successor are complete. Trust-priority improved blind
  completeness and narrowed source but increased repository actions and model
  usage. External review identifies evidence acquisition/orchestration after
  useful locator selection as the measured next boundary without isolating
  stopping from legitimate dependency expansion. The owner has authorized one
  terminal, provider-free bounded multi-locator compatibility experiment in
  reopened Phase 13. Phase 14 does not run concurrently and resumes only after
  that branch is retained or rejected. Paid provider work remains unopened;
  official Phase 12 and release-candidate promotion remain separately blocked.

## 2026-08-22 V4 freeze checkpoint

The [Version 4 plan](ASSISTANT-AB-TEST-PLAN-V4.md) and manifest are frozen.
Their questions, corpora, prompt, 12-task schedule, and arm order equal V3;
locator-only structured search is the sole treatment change. Exact native
Codex 0.148.0, corpus/state, four-tool schema, and representation-lock
preflight passed. Identities and the no-scored-turn boundary are recorded in
[the V4 freeze evidence](evidence/phase-14/assistant-ab-v4-freeze.md).

## 2026-08-22 V4 result checkpoint

All 24 V4 task turns were valid and both arms blindly graded 12/12 complete.
Compact search returned 72,172 structured bytes, zero text, and zero source;
total cidx event payload fell 82.9% from V3. The paired model-token median was
nevertheless 1.044, only 4/12 tasks were non-increasing, and uncached input
sum increased 9.8%. The first residual is 23 searches plus 39 reads, including
six failed ranges, four identical successful rereads, and macro read precision
0.586. Exact metrics and artifact identities are in the
[V4 result](evidence/phase-14/assistant-ab-v4-result.md). Do not change `k`,
retrieval, or `read_span` semantics before the required post-result review.

## 2026-08-22 V4 post-result review checkpoint

ChatGPT and Grok independently returned `PROCEED_V5_ORCHESTRATION` after
receiving the same fixed V4 evidence. They approved one assistant-facing
paragraph that requires an initial locator search with `max_inline_bytes=0`,
forbids identical searches and reads, starts evidence reads at exact returned
ranges, permits at most one specifically justified refinement, and stops after
material claims have direct evidence. The [external review record](evidence/phase-14/assistant-ab-v4-external-review.md)
freezes the wording and guardrails. Prompt noncompliance is a measured outcome
of V5 rather than an automatic invalidation. Current caller-selected `k`
semantics with default 10, retrieval/ranking, wire, read semantics,
corpus/tasks, and grading remain unchanged.

## 2026-08-22 V5 freeze checkpoint

The [Version 5 plan](ASSISTANT-AB-TEST-PLAN-V5.md), manifest, and
[freeze evidence](evidence/phase-14/assistant-ab-v5-freeze.md) are complete.
V4/V5 controls, question sources, corpora, and all ordered task records are
exactly equal; the sole prompt difference is the reviewed orchestration
paragraph. Reducer v3, both corpus states, native Codex, cidx/MCP binaries,
tool schema, and structured representation are frozen. Corpus/tool preflight
and both unscored schema probes passed. No scored V5 turn has run.

## 2026-08-22 V5 result checkpoint

All 24 V5 turns were valid and both arms blindly graded 12/12 complete with all
30 required-group records covered. The intervention reduced cidx search/read
calls from V4's 62 to 38, eliminated invalid ranges, made all 16 first
path/hash reads use exact locator ranges, and reduced gross read source bytes
76.7%. Treatment model-total sum fell 6.9% and uncached input fell 34.4%, but
the paired model-total median was 0.954 with 6/12 non-increasing; the frozen
0.85 and 8/12 gate therefore remains unmet.

The only failed V5 reads were three requests that copied locator path and lines
but omitted required `expected_sha256`; the same ranges then succeeded with
the hash. Exact evidence and the bounded one-field V6 candidate are in the
[V5 result](evidence/phase-14/assistant-ab-v5-result.md). Do not change
retrieval, `k`, wire semantics, corpus, or questions before post-result review.

## 2026-08-22 V5 post-result review checkpoint

ChatGPT and Grok both returned `PROCEED_V6_HASH_FIELD`. The
[review record](evidence/phase-14/assistant-ab-v5-external-review.md) accepts
one exact `expected_sha256` prompt sentence and no other change. V6 retains all
V5 product, corpus, model, order, reducer, grading, and token controls. It is
the final permitted run in this corrective sequence; no V7 follows regardless
of its outcome.

## 2026-08-22 V6 freeze checkpoint

The [V6 plan](ASSISTANT-AB-TEST-PLAN-V6.md), manifest, and
[freeze evidence](evidence/phase-14/assistant-ab-v6-freeze.md) are complete.
Exact comparison proves one prompt-sentence insertion and equality of every
V5 control, question source, corpus, and ordered task record. Corpus/state/tool
preflight and both unscored schema probes passed. No scored V6 turn has run.

## 2026-08-22 V6 result checkpoint

All 24 V6 turns were valid. The required-hash mechanism passed with 24/24
successful reads, zero omitted hashes, and zero failed/recovery calls.
Correctness preservation failed: baseline remained 12/12 complete while
treatment was 11 complete and one partial because `rhf-x03-form-submit` added
one material claim not established by its cited excerpts. The 11
dual-complete pairs have model-total median ratio 0.977 and 6/11
non-increasing, so the token gate also fails. Exact evidence is in the
[V6 result](evidence/phase-14/assistant-ab-v6-result.md). Final external
interpretation closes V4–V6; no V7 follows.

## 2026-08-22 V6 final external review checkpoint

ChatGPT and Grok agree on all fixed gates and the first-loss classification:
the hash mechanism passes, correctness and paired tokens fail, X03 is final
claim/evidence discipline after successful navigation, and 12/12 mechanical
adherence is too narrow to establish efficient orchestration. Their only
label difference is whether observed locator behavior merits an explicit
“bounded optional value” suffix. The reconciled decision closes V4–V6 with no
efficiency claim and leaves optional locator value unpromoted. See the
[review record](evidence/phase-14/assistant-ab-v6-external-review.md).

## 2026-08-22 V4–V6 closure checkpoint

The [closure and handoff](evidence/phase-14/assistant-ab-v4-v6-closure.md)
records all three improvements, their independent correctness/mechanism/token
decisions, matching external interpretation, prohibited claims, preserved
commit history, and next owner choices. The compact locator and required-hash
mechanics remain accepted. Mandatory cidx-first token efficiency and final
correctness preservation are not established. No rerun, regrade, or V7 is
pending.

## 2026-08-22 Search-to-evidence design checkpoint

The owner accepted the
[host-decided search-to-evidence design](ASSISTANT-SEARCH-EVIDENCE-FLOW-DESIGN.md)
as the next planning direction. It does not label cidx secondary-only or force
it to run first. It preserves the existing nine-field locator response and
four-tool MCP, then separates compact candidate discovery, complete selected
parent evidence, and only claim-required dependency/context expansion.

The initial study is 30 frozen question pairs and 60 assistant turns over the
existing approved Go, TypeScript, and TSX corpora. It is FTS-only and
provider-free, leaves cidx use to the assistant, retains no-use treatment
turns in the primary denominator, and records exploration through a passive,
non-model-facing session trace. It adds no tool, database authority, session
store, signature, neighbor metadata, dependency hint, repository, or paid
query operation. ChatGPT and Grok both returned `FINAL_ACCEPT`; the complete
[external review record](evidence/phase-14/assistant-search-evidence-flow-external-review.md)
contains their corrections and cautions. The subsequent owner instruction
opens Section 14 Step 2 only; question construction and execution remain
separate. The phase dependency topology is unchanged.

## 2026-08-22 Search-to-evidence Step 2 implementation checkpoint

The owner-opened Step 2 boundary is complete. The four MCP tool names and
input schemas are unchanged; description text now states capabilities without
forcing cidx-first or secondary-only use. The harness writes a versioned,
body-free passive trace only under ignored evaluation artifacts. It observes
cidx search/read order, requested and effective FTS authority, canonical
locators, exact source-conformant reads, ordinary repository inspection,
attributed and unattributed output bytes, source-range overlap, no-use, and
paired exploration. Blind-grade schema v2 adds post-turn material-claim units
and evidence references without changing legacy v1 grades.

There is no new MCP tool, public input/output field, retrieval behavior,
production SQLite table/write, server-side conversation state, question,
scored assistant turn, or provider operation. Temporary replay proved V3–V6
aggregate JSON byte-identical and reports identical except for expected local
temporary-path lines. Focused existing Go checks, script compilation, real-log
trace/reducer probes, MCP representation preflight, schema/claim-contract
probes, and independent code review are recorded in the
[Step 2 implementation evidence](evidence/phase-14/assistant-search-evidence-flow-step-2-implementation.md).
Step 3 remains a separate owner decision.

## 2026-08-23 Forced cidx prompt diagnostic entry

The owner explicitly opened a separate paired mechanism diagnostic after the
neutral availability run selected cidx in zero of 30 treatment turns. Both new
arms expose the same four cidx tools and frozen FTS state. The sole paired
difference is one exact prompt paragraph: the directed arm must locate code
through `cidx.search` and inspect selected locators through `cidx.read_span`
instead of using `rg` or equivalent ordinary repository-search commands.

This does not change the product role or reopen V4-V6. Prompt noncompliance is
retained as an outcome rather than an invalid or retried turn. The result may
measure cidx behavior under explicit host policy, but it cannot establish
voluntary adoption, optional-use marginal value, or promotion. The exact arms,
prompt, metrics, controls, and stop rules are in the
[Forced cidx Prompt Diagnostic V1](ASSISTANT-FORCED-CIDX-PROMPT-EVALUATION-V1.md).
Execution is complete. All 60 scored turns were valid, cidx use moved from
0/30 in the neutral arm to 30/30 in the directed arm, and blind outcomes were
29 complete plus one partial in each arm. Directed cidx reduced combined
unique source bytes from 532,030 to 196,476 and unsupported claims from four
to one, but repository actions rose from 104 to 254 and the paired model-total
ratio median was 1.404. The first measured loss is repeated search/read and
source reacquisition after useful locators, not missing positive targets. See
the immutable
[result](evidence/phase-14/assistant-forced-cidx-prompt-result-v1.md). No
previous assistant turn or grade was repaired or rerun.

## 2026-08-23 awareness/trust follow-up reset

The owner rejected and hard-reset the follow-up work that attempted to add a
predecessor re-audit, semantic packet/reviewer pipeline, dedicated runner,
large scorer expansion, mandatory 60-session self-report, and another
60-context semantic pass. No successor primary turn ran, and no product or
retrieval behavior from that work remains active.

The owner-required [awareness vs trust-priority plan](ASSISTANT-CIDX-AWARENESS-TRUST-EVALUATION-V1.md)
has been restored after the reset. It preserves concise MCP descriptions, the
two prompt arms, question-level grading, deterministic journey facts, and
selective post-run diagnostic questions without restoring the rejected audit
or semantic-review infrastructure.
The [incident record](evidence/phase-14/assistant-cidx-followup-overbuild-incident.md)
documents the removed scope and prevention rules. The restored plan has now
passed its bounded external review and authorizes only the minimal existing-
path implementation and preflight before a scored-run freeze. It must reuse
the existing runner, trace, and grading paths by default.

## 2026-08-23 awareness/trust result checkpoint

The restored experiment was frozen at
`caa57f906d9a3e6219d24c8fe71914bf82bd22ec`. Both schema probes passed and 60
primary cells ran exactly once. One aware-choice task timed out at the frozen
600-second boundary and remains ungradable without retry. Blind grading and
the single canonical aggregate are complete.

Trust-priority produced 30/30 complete answers, 39/39 required groups, and no
unsupported claim. Aware-choice produced 28 complete, one partial, one
ungradable timeout, 37/39 groups, and one unsupported claim. On the 29 valid
paired efficiency records, trust-priority reduced unique source to median
0.546 but raised repository actions to median 1.300 and model-total usage to
median 1.189. cidx calls increased from 102 to 220, including 95 searches and
125 reads in the trust arm.

The [result](evidence/phase-14/assistant-cidx-awareness-trust-result-v1.md)
therefore accepts a quality and code-scope benefit but rejects an overall
efficiency claim. The [post-result review](evidence/phase-14/assistant-cidx-awareness-trust-result-external-review-v1.md)
agrees that the dominant measured loss after useful locator selection is
evidence acquisition/orchestration. It corrects any stronger stopping claim:
the trace does not separate unnecessary continuation from legitimate
dependency expansion.

The smallest reviewed successor candidate is one bounded request over a small
set of already returned locators, returning separate line-addressable evidence
units. This is not implemented or authorized because it changes the public MCP
contract. Retrieval, ranking, provider work, new repositories, and semantic
review remain outside this result.

- Prerequisite: reconciled locator-only `13-cli-and-mcp`
- Followed by: v1 release-candidate validation
- Design source: `local-code-search-mcp-v1-design-r4.md` sections 1–3 and 7–10
- Evaluation authority: [EVALUATION-CONTRACT.md](EVALUATION-CONTRACT.md)

## Context Recovery Checklist

Read the [implementation index](README.md), [execution guide](EXECUTION-GUIDE.md), [evaluation contract](EVALUATION-CONTRACT.md), and [project status](STATUS.md) before resuming.

- Confirm Phase 01 recorded the SQLite/Tree-sitter bindings, FTS5/CGO policy, and candidate platforms; Phase 13 must have frozen public CLI, stdio, and exactly four MCP tools.
- Re-check that the artifact bundles FTS5 and Go/TypeScript/TSX grammars, needs no runtime dependency download for free FTS, and serves one explicit root per process.
- Re-check project-scoped host setup, stdout protocol purity, stderr diagnostics,
  locator-only search, the 64 KiB default / 1 MiB absolute `read_span` source
  ceiling, no read-span line cap, and environment-only `VOYAGE_API_KEY`
  forwarding.
- Re-check that serving/package smoke does not open the source bank or lab DB, mutate host config or hooks, promise unverified platforms, or invent fixed-model/external-vector policy.
- Read the accepted
  [search-to-evidence design](ASSISTANT-SEARCH-EVIDENCE-FLOW-DESIGN.md) and its
  [external review](evidence/phase-14/assistant-search-evidence-flow-external-review.md)
  before changing an assistant interface, harness, question set, or metric.
- Preserve product host choice: cidx may be first, main, occasional, or unused. In the
  initial availability study, no-use treatment turns remain in the primary
  denominator and cidx-user-only slices are descriptive.
- Treat the 2026-08-23 directed-cidx experiment as a bounded non-promotion
  prompt-policy diagnostic only. Both arms must expose cidx, prompt violations
  remain in the denominator, and its result cannot replace optional-use
  evidence.
- Keep the first live session ledger passive, non-model-facing, absent from
  product/SQLite/server state, and unable to alter ranking, tool calls, prompts,
  or MCP behavior. Retain only a truth-free, body-free snapshot in ignored
  local evaluation artifacts for reproducibility. Populate usefulness and
  claim-support annotations only after blind grading.
- Do not add signature, neighbor, or dependency metadata before the initial
  measured first loss shows that the existing nine locators plus `read_span`
  are insufficient.
- Treat the forced first cidx call in diagnostic V3 as a bounded causal
  intervention only. It does not override the non-forced release-candidate
  assistant protocol above.
- Before the 30-question study, freeze the current configured default `k=5`
  and maximum 20, source-first questions, execution identities, pure-capability
  tool descriptions, provider-free enforcement, and metric authorities. Do
  not call transcript envelope bytes model-visible unless the host proves
  ingestion.
- If the relation completion series reaches assistant evaluation, add the
  separately frozen closure, body-free hints plus existing `read_span`, and
  closure-plus-hints development arms from
  [`RELATION-EVIDENCE-COMPLETION-PLAN.md`](RELATION-EVIDENCE-COMPLETION-PLAN.md).
  These arms are independent of server-push precision, remain non-product
  until measured, and do not add an MCP tool.
- Stop if dependency licensing, FTS/grammar reproducibility, schema compatibility, root semantics, or a host-specific config format is unverified. Do not claim inferred support.
- Before pausing, update build/host evidence and this decision log, then update [STATUS.md](STATUS.md) with verified and unverified targets, risks, and next action.

## 1. Objective

Produce one `cidx` executable containing FTS5 SQLite and Go/TypeScript/TSX Tree-sitter grammars, with explicit project-scoped stdio MCP registration for each supported host.

The deployment must preserve:

- FTS-only index/search needs neither runtime downloads nor an API key.
- SQLite FTS5 and grammars do not depend on accidental system installation.
- One MCP process handles one explicit root and does not rely on host cwd.
- cidx does not modify host settings, register user scope, or put secrets in project config.
- Production serve does not open the product source bank or lab DB.

Release-candidate evidence must also measure the marginal effect of making cidx
available beside an assistant's existing file, symbol, compiler, and test
tools. The host remains free to use cidx as its first, main, occasional, or
unused repository-search path; forced invocation and cidx-user-only slices do
not replace the intent-to-treat product comparison.

## Current local accepted checkpoint

The local darwin/arm64 target was rebuilt and verified from clean provenance
`5f4955e1499ee8896be5c825ef0fb9b3a52abb70`. The current ignored archive,
checksums, runtime facts, installed-binary 1024/int8 and provider-free 512/int8
materialization, retired-profile rejection, source-bank-free four-tool MCP,
and retained transcripts are recorded in
[current int8 package evidence](evidence/phase-14/int8-profile-package-reconciliation.md).
The earlier checkpoint remains [historical evidence](evidence/phase-14/revision-4.md).
Neither checkpoint verifies another OS/architecture or host, code signing,
notarization, assistant usefulness, official retrieval evaluation, or
`release_candidate`.

## 2. Scope and Non-goals

### In scope

- Release builds for Phase 01's SQLite/Tree-sitter bindings.
- FTS5 compile/runtime checks and bundled Go/TS/TSX grammars.
- Binary archives for verified OS/architecture pairs.
- Version/build metadata, checksums, and license notices.
- Clean-environment offline FTS smoke verification.
- Project-scoped host documentation and examples.
- Explicit `--root`, with PATH and absolute-binary variants.
- Safe API-key environment forwarding.
- Manual composition with existing post-commit hooks.
- Upgrade/schema compatibility and failure guidance.
- Paired assistant-task execution through actually verified hosts under frozen model, prompt, tools, budgets, corpus, and task truth.
- Task success, requirement coverage, evidence utilization, false leads, tool/time/token/cost observations, and operation failures for all three product arms.

### Out of scope

- Automatically editing, merging, or deleting host config.
- `cidx install`/`uninstall` and user/global MCP registration.
- Remote MCP, HTTP, centralized service, or multi-repository routing.
- Runtime grammar/model/binary downloads, automatic updates, or a daemon.
- Promising every package manager or installing/overwriting Git hooks.
- Claiming code signing/notarization without verification.
- Final long-term model distribution or external-vector contracts.
- Using assistant results to retune confirmation labels, retrieval settings, body budgets, or promotion margins.

### Explicit policy non-goals

This phase chooses neither a permanent bundled/pinned embedding profile nor an external vector-supply architecture. It packages only current v1: official direct Voyage AI API, initially validated `voyage-code-4`, default 1024/optional 512 serving profile, fixed cidx-owned production int8 storage, product document source bank, and separate evaluation lab. Binary/256 are preserved documents only and are not package smoke options.

Do not add provider plugins, vector import formats, model bundles, or speculative extension points. Decide long-term policy later through a separate design when real deployment requirements exist.

## 3. Prerequisites

- Phase 01 decided SQLite binding, FTS5 inclusion, CGO policy, Tree-sitter binding, and candidate platforms.
- Phase 02 froze production/lab schemas and migration boundary.
- Phase 13 froze public/development CLI, stdio, and four MCP schemas.
- Phase 14 owns build/version/schema/FTS and bundled-grammar capability reporting before packaging claims are accepted.
- `cidx serve --root` resolves config and DB below that source project's `.cidx`; portable DB metadata contains no machine-path binding.
- Exact host names and versions used for verification can be recorded.

## 4. Invariants

1. Release FTS5 does not depend on system SQLite build flags.
2. No grammar is downloaded on first execution or parse.
3. Parser/FTS implementation IDs accurately affect profile fingerprints.
4. One invocation serves one explicit root.
5. Project host config states command and `--root`; cwd inference is not required.
6. v1 never edits host files programmatically.
7. Config and examples contain no API-key literal.
8. Official support documentation covers project scope only.
9. Release `serve` neither creates nor opens `.cidx/db/embeddings.db` or any `.cidx/test/` evaluation state.
10. Unsupported schema/config/profile fails clearly rather than being silently migrated or ignored.
11. Bad checksum, corrupt archive, or missing execute permission is never reported as success.
12. Product-usefulness and promotion runs do not require or force a cidx call;
    no-use is a valid observed outcome. A separately owner-authorized forced
    prompt diagnostic remains non-promotion and preserves noncompliance.
13. Required assistant task failures and timeouts remain in denominators, and an unexecuted optional arm is `NOT_OBSERVED`, not zero.
14. Paired assistant claims require the same assistant model/version, prompt, existing tools, task order policy, context/tool budgets, corpus snapshot, and expected outcomes except for the declared cidx arm.
15. MCP body and `read_span` byte limits retain the Phase 13 64 KiB default and 1 MiB absolute ceiling; there is no separate read-span line-count limit.

## 5. Implementation Packages, Files, and Artifacts

```text
internal/buildinfo/info.go        # version, commit, target, dependency IDs
internal/runtimecheck/check.go    # disposable SQLite FTS5/WAL and bundled-grammar probes
cmd/cidx/main.go
docs/install.md
docs/hosts.md
docs/hooks.md
.github/workflows/release.yml
```

Exact tooling may follow repository conventions, but it must produce:

```text
cidx_<version>_<os>_<arch>.<archive>
checksums.txt
LICENSE
THIRD_PARTY_NOTICES
build-manifest.json
<state_root>/evaluations/<run-id>/assistant-observations.jsonl
<state_root>/evaluations/<run-id>/promotion-result.json
```

`build-manifest.json` records binary version, source commit, target, Go version, SQLite binding/version/FTS capability, Tree-sitter binding and grammar IDs, and CGO/static-link policy. It contains no credentials, source bodies, or vectors.

Assistant-run artifacts use the Phase 02 evaluation contracts and the immutable artifact layout in `EVALUATION-CONTRACT.md`. They record task IDs, required evidence/actions, presented and used result IDs, first loss, task/test outcome, tool calls, timings, tokens/cost, failures, and checksums. They do not copy source bodies, query vectors, secrets, or machine-specific corpus paths.

Required types:

```text
BuildInfo
  Version / Commit / BuildTime(optional, with reproducibility caveat)
  TargetOS / TargetArch
  SQLiteImplementationID / GrammarImplementationIDs / ChunkerImplementationIDs

RuntimeCapabilities
  FTS5Available / RegisteredLanguages
  ProductionSchemaRange / LabSchemaRange(dev only)

HostLaunchContract
  Command / Args[serve, --root, explicit root]
  Environment variable names, never values
  Transport=stdio
```

## 6. Distribution, Host API, and CLI Contract

### 6.1 Binary distribution

The user flow is:

1. Obtain the verified target artifact and verify its checksum.
2. Run `cidx init` and `cidx index` in the Git repository.
3. Register `cidx serve --root <absolute-repository-root>` in project-scoped MCP config.

The binary may be on PATH or referenced by an absolute path. Document both, including host-correct handling for spaces.

### 6.2 Runtime capability checks

- `init`, `index`, and `serve` check required FTS5 capability early.
- If config requires an absent Go/TS/TSX grammar, fail before parsing rather than deferring.
- If build implementation IDs conflict with the DB profile, direct the user to status/reconciliation.
- Never repair a missing dependency by downloading it at runtime.

### 6.3 Project-scoped host configuration

For every actually verified host/version, `docs/hosts.md` records project-scope location, stdio command/args, explicit root, PATH and absolute-binary examples, stdout/stderr rules, lifecycle/restart instructions, discovery of exactly four tools, API-key-free FTS smoke steps, the retained v1 `max_inline_bytes` compatibility input, locator-only output, and safe `VOYAGE_API_KEY` forwarding for hybrid.

Do not abstract unrelated JSON/JSONC/YAML host formats behind a generic merger. Record verification date and host version because upstream formats change.

### 6.4 Git hooks

Hooks are optional. Show only how a user manually composes this call into an existing post-commit hook or `core.hooksPath` flow:

```text
cidx index --reason commit
```

Beside the example, state that it reads the live worktree at execution time, not HEAD blobs. After a partial commit it may therefore include remaining uncommitted and untracked non-ignored code. Explain the user's choice for hook failure behavior so an index failure does not retroactively damage the commit.

### 6.5 Upgrade

- Distinguish binary replacement from repository DB migration/reconciliation.
- Do not migrate schemas while serve is handling requests.
- If a new binary requires migration or reindex, report it through status or a clear startup error.
- Do not infer downgrade support; fail closed outside the supported schema range.
- Evaluation state is not a production-upgrade prerequisite. The product source-bank schema is checked separately from serving `index.db` and is never opened by `serve`.

## 7. Configuration and Change Impact

Packaging never bakes project profile values into a binary. Runtime resolves `.cidx/config.json` through the Phase 02 loader.

| Value | Authority | Distribution impact |
| --- | --- | --- |
| SQLite/Tree-sitter implementation IDs | Build manifest/code | Artifact change may require reconciliation |
| Supported OS/architecture | Release matrix | Directly determines artifact availability |
| Model/serving-dimension/codec | Project config plus `ModelSpec` | Source is `voyage-code-4` 1024; profile rules apply |
| Hard absolute safety caps | Named binary constants | Belong to artifact version |
| Project MCP hard max/search policy | Project config | Applies after serve restart; no reindex |
| Repository root | Host `--root` argument | Explicit per process |
| API key | `VOYAGE_API_KEY` environment | Secret value never enters config/fingerprint |

Do not hard-code environment paths or credentials in repository examples. The official endpoint is a code-owned Voyage adapter constant; provide no custom `base_url`, endpoint override, or secret literal.

## 8. Ordered Implementation Checklist

1. Document the release target matrix and CGO/static-link policy from Phase 01.
2. Freeze build flags/dependencies that include FTS5 and grammars.
3. Implement `BuildInfo` and early runtime capability checks.
4. Build each target artifact in a clean environment.
5. Create archives, executable permissions, checksums, and license notices.
6. Unpack into a fresh environment and verify without system SQLite/grammar installation.
7. In a small Git repository, run key-free/network-free `init -> index -> serve -> status/search` FTS smoke.
8. Verify startup failure for unsupported config/schema/root mismatch.
9. Check paths with spaces and non-ASCII characters.
10. Connect project-scoped examples to every claimed host/version.
11. Verify discovery and smoke behavior for the four MCP tools.
12. Verify stdout protocol purity and stderr diagnostics in each host.
13. Review environment examples for secret literals.
14. Document manual hook composition without overwriting existing hooks.
15. Document upgrade, migration-required, and unsupported-downgrade flows.
16. Include build manifest plus supported/unverified platform lists in release notes.
17. Confirm no accidental import API or promise for fixed-model/external-vector policy.
18. Freeze user-approved assistant tasks, expected file/symbol/edit/test outcomes, assistant/model/tool/budget controls, and the three comparison arms before execution.
19. Run paired existing-tools-only, plus-lexical-cidx, and plus-hybrid-cidx arms without requiring the assistant to call cidx.
20. Preserve every success, no-use, failure, and timeout in denominators; record evidence utilization, false leads, first useful result, task outcome, tool calls, time, tokens, and cost.
21. Apply the already-frozen assistant-use gates without tuning retrieval or labels, and write a new immutable `scope=release_candidate` promotion result referencing the Phase 12 core result and Phase 13/14 artifact digests.

## 9. Failure, Rollback, Concurrency, and Security

### Failure and recovery

- Do not execute an artifact with a mismatched checksum; instruct the user to obtain it again.
- Do not claim a target when its build failed. List only successful artifacts.
- FTS5/grammar check failure stops startup rather than silently disabling lexical search.
- Because the user edits host config, cidx has no host file to roll back.
- Never delete or reinitialize a DB automatically when a new binary cannot open its schema.
- Upgrade docs distinguish retaining an older artifact from DB backup/recovery and do not make destructive commands the default.

### Concurrency

- One host process handles one root.
- Multiple processes on one root rely on Phase 05/10 locks and SQLite writer policy.
- Packaging adds no global lock or daemon singleton.
- Host termination follows Phase 13 cancellation/commit semantics.

### Security and supply chain

- Publish archive and checksum within a documented trust boundary and record provenance.
- Review third-party licenses and source-distribution obligations.
- Keep secret values out of project examples.
- Absolute `--root` does not expand authority; runtime source-root canonicalization and path validation still apply. The canonical path is not persisted in SQLite.
- Warn about symlink confusion and a writable untrusted PATH.
- Runtime downloads or executes no arbitrary code, grammar, or model.
- Provide no undocumented external-vector path into production storage.
- Treat evaluation tasks and assistant transcripts as local sensitive artifacts. Do not publish source, prompts, or tool outputs unless separately reviewed.

## 10. Validation Scenarios

This file defines a plan and creates no test code or release artifact.

1. A supported artifact creates an FTS5 table without system SQLite.
2. With network blocked, it parses Go/TS/TSX and performs FTS search.
3. Manifest dependency IDs match runtime reports.
4. A checksum mismatch cannot proceed as successful installation.
5. In a clean repository without `VOYAGE_API_KEY`, the project host discovers exactly four tools.
6. A different host cwd still opens only the explicit root.
7. DB root mismatch fails closed.
8. Paths containing spaces or non-ASCII characters work in a verified host.
9. Concurrent reindex from two host processes does not damage search snapshots.
10. stdout contains no nonprotocol logs.
11. Config and docs contain no credential literal.
12. Serve works without a source bank or lab DB and neither creates nor opens one.
13. Unsupported schema/config/architecture reports an actionable error.
14. Hook docs accurately explain live-worktree semantics and require no automatic hook mutation.
15. Package/API surfaces make no external-vector-import or fixed-model-bundle promise.
16. The three assistant arms use identical frozen controls except for cidx availability and retain failed/timed-out tasks in denominators.
17. A task in a cidx arm may complete without calling cidx; the run records no-use rather than forcing or discarding it.
18. Presented-gold utilization, false leads, correct file/symbol/edit/test outcomes, and first-loss attribution reconcile with the underlying MCP locators and selected `read_span` evidence.
19. Confirmation output cannot change dimension, codec, RRF, candidates, body budget, labels, or margins.

## 11. Completion Evidence

The current local darwin/arm64 implementation and operational subset is
accepted in [current int8 package evidence](evidence/phase-14/int8-profile-package-reconciliation.md).
The separately bounded assistant interface/trace Step 2 implementation is
accepted in the
[search-to-evidence Step 2 evidence](evidence/phase-14/assistant-search-evidence-flow-step-2-implementation.md).
The owner-authorized awareness-versus-trust-priority successor is complete.
Its [execution freeze](evidence/phase-14/assistant-cidx-awareness-trust-freeze-v1.md),
[result](evidence/phase-14/assistant-cidx-awareness-trust-result-v1.md), and
[post-result review](evidence/phase-14/assistant-cidx-awareness-trust-result-external-review-v1.md)
reuse the existing 30 questions, four tools, FTS state, runner, passive trace,
and blind grading path. This is non-promotion evidence and does not close the
release gates below.
The remaining items below are promotion/release gates, so the phase stays
open rather than `done`.

- Actual supported OS/architecture artifacts and checksums.
- Build manifest and third-party notices.
- Offline FTS5 and bundled-grammar smoke record.
- Clean-environment init/index/serve/FTS transcript.
- Project-scoped result for every verified host/version.
- Space/non-ASCII/root-mismatch results.
- stdout/stderr captures.
- Schema upgrade/error-flow record.
- Minimal serving run without `VOYAGE_API_KEY`, source bank, or lab DB.
- Explicit list of unsupported or unverified platforms/hosts.
- Release-surface review confirming fixed-model/external-vector policy remains out of scope.
- Frozen assistant task/control manifest and paired records for existing tools only, plus lexical cidx, and plus hybrid cidx.
- Per-task and aggregate task success, requirement coverage, evidence utilization, false leads, correct edit/test outcome, tool/time/token/cost, failure, and no-use evidence with explicit denominators.
- Corrected assistant journey reducer, host result-representation compatibility
  evidence, locator/evidence stage metrics, and compact-response V4 diagnostic.
- New immutable `scope=release_candidate` promotion result referencing the Phase 12 core result and showing every applicable assistant/host gate, or an explicit `NOT_PROMOTION_READY` with failed cohorts and first-loss evidence.

Never report an unverified target as supported.

## 12. Handoff

Release-candidate verification receives artifacts/checksums/build manifest, the verified platform and host matrix, install/project/hook/upgrade docs, four-tool smoke steps, API-key-free FTS path, explicit paid-hybrid environment/cost guidance, paired marginal-usefulness evidence, the updated promotion result, and known packaging limitations and unresolved policy items.

If operational feedback establishes a real requirement for permanent model pinning or external vector supply, create a new architecture decision first. Do not reinterpret the v1 import/packaging contract retroactively.

## 13. Decision Log

| Decision | Rationale | Revisit when |
| --- | --- | --- |
| Compare concise awareness with cidx priority and bounded trust on the existing 30-pair panel | The completed run shows better completeness and source narrowing but more actions and model usage; it measures evidence-acquisition/orchestration loss without isolating stopping from legitimate dependency expansion | Complete; preserve as non-promotion evidence |
| Keep a bounded multi-locator evidence request as the next candidate, not an authorized change | The trust arm made 125 individual reads with more duplicates and overlap after useful locators; one bounded request targets round trips while preserving separate line-addressable evidence units | The owner chooses the bound and versioned public-interface shape |
| Permit one owner-directed forced-cidx prompt diagnostic outside promotion | Zero adoption left locator/evidence behavior unobserved; a same-tools neutral-versus-directed pair isolates prompt policy while preserving noncompliance and does not redefine the product role | The diagnostic is complete or an optional-use product comparison is prepared |
| Bundle FTS5 and grammars | Keep the free core independent of system installs and runtime downloads | Platform constraints block real distribution |
| Officially document project scope only | Reduce wrong-root and multi-repository confusion | Safe user-scope root routing is designed |
| Require explicit `--root` | Avoid accidental host cwd | Core v1 invariant |
| Do not edit host config | Avoid host-specific merge/removal risk | A verified installer is separately designed |
| One process per repository | Keep state, DB, and authority boundaries simple | Multi-repository service is separately designed |
| No runtime dependency download | Preserve offline operation, reproducibility, and supply-chain boundary | An explicit plugin system is approved |
| Fixed model versus external supply stays out of scope | Do not mix initial lab needs with long-term distribution | Real production requirements appear |
| No speculative vector import format | Avoid unvalidated integrity/security compatibility | A separate ADR and provenance design are approved |
| Evaluate marginal assistant value through host-decided availability | Existing tools remain the baseline, while the treatment makes the neutral four-tool cidx interface available without forcing or demoting it; all treatment turns stay in the intent-to-treat denominator | A later frozen intervention tests an explicit host policy |
| Keep first-run session accounting passive and host-side | Measure duplicate exploration, evidence acquisition, and claim support without adding a server authority or changing assistant behavior | A separately approved orchestration study freezes active guidance as its sole intervention |
| Complete Step 2 without changing the public MCP schema | Neutral descriptions plus a body-free ignored-artifact trace measure availability and evidence flow while preserving four tools, locator-only search, source-only reads, and SQLite authority | A measured first loss and separate owner decision authorize Step 3 or reopen the wire |
| Runtime checks use disposable local state | FTS5/WAL and all embedded grammars must fail before repository mutation or production migration, without downloads or repairs | A future runtime changes the bundled dependency boundary |
| Owner selected Apache-2.0; root license and local package checkpoint recorded | The unmodified root `LICENSE` supplies cidx's terms while third-party notices remain separate; local verification is limited to darwin/arm64 | Another release target, distribution policy, or owner terms require review |
| CLI-only provenance report | Build facts are needed for package verification, while Phase 13's MCP `serverInfo` and four-tool surface remain frozen | The MCP version contract is separately revised |
| Verify both current product profiles from one source bank | Default 1024/int8 and explicit 512/int8 must work in the installed binary without making source storage a serving dependency | The source/serving contract changes |
| Keep Binary/256 package checks negative-only | Retired experimental profiles remain historical evidence and cannot regain an executable product entry point | A new measured design decision explicitly reauthorizes them |
| Separate locator search from evidence reads in the next assistant diagnostic | V3 top ranks often contained useful code, but source-bearing search results, repeated diagnostics, and later reads overlapped; preserving caller-selected `k` while compacting results isolates the response contract before changing retrieval | Compact V4 evidence shows a different first loss or the public contract cannot preserve required safety/provenance |
