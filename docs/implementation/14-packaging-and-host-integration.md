# 14. Packaging and MCP Host Integration

- Status: `blocked` — the current default-1024/int8 and provider-free
  compact-512 local package checkpoint is accepted; official Phase 07/12,
  assistant-use, and release-candidate gates remain externally blocked
- Prerequisite: `13-cli-and-mcp`
- Followed by: v1 release-candidate validation
- Design source: `local-code-search-mcp-v1-design-r4.md` sections 1–3 and 7–10
- Evaluation authority: [EVALUATION-CONTRACT.md](EVALUATION-CONTRACT.md)

## Context Recovery Checklist

Read the [implementation index](README.md), [execution guide](EXECUTION-GUIDE.md), [evaluation contract](EVALUATION-CONTRACT.md), and [project status](STATUS.md) before resuming.

- Confirm Phase 01 recorded the SQLite/Tree-sitter bindings, FTS5/CGO policy, and candidate platforms; Phase 13 must have frozen public CLI, stdio, and exactly four MCP tools.
- Re-check that the artifact bundles FTS5 and Go/TypeScript/TSX grammars, needs no runtime dependency download for free FTS, and serves one explicit root per process.
- Re-check project-scoped host setup, stdout protocol purity, stderr diagnostics, the 64 KiB default / 1 MiB absolute `max_inline_bytes` ceiling, no read-span line cap, and environment-only `VOYAGE_API_KEY` forwarding.
- Re-check that serving/package smoke does not open the source bank or lab DB, mutate host config or hooks, promise unverified platforms, or invent fixed-model/external-vector policy.
- Re-check the frozen assistant-task controls and three product arms: existing tools only, existing tools plus lexical cidx, and existing tools plus hybrid cidx. Never force a cidx call.
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

Release-candidate evidence must also measure cidx's marginal usefulness beside an assistant's existing file, symbol, compiler, and test tools. It does not treat a cidx-only assistant or forced cidx invocation as the product.

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
12. Assistant usefulness runs do not require or force a cidx call; no-use is a valid observed outcome.
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

For every actually verified host/version, `docs/hosts.md` records project-scope location, stdio command/args, explicit root, PATH and absolute-binary examples, stdout/stderr rules, lifecycle/restart instructions, discovery of exactly four tools, API-key-free FTS smoke steps, required `max_inline_bytes`, and safe `VOYAGE_API_KEY` forwarding for hybrid.

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
18. Presented-gold utilization, false leads, correct file/symbol/edit/test outcomes, and first-loss attribution reconcile with the underlying MCP result IDs and body packages.
19. Confirmation output cannot change dimension, codec, RRF, candidates, body budget, labels, or margins.

## 11. Completion Evidence

The current local darwin/arm64 implementation and operational subset is
accepted in [current int8 package evidence](evidence/phase-14/int8-profile-package-reconciliation.md).
The remaining items below are promotion/release gates, so the phase stays
`blocked` rather than `done`.

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
- New immutable `scope=release_candidate` promotion result referencing the Phase 12 core result and showing every applicable assistant/host gate, or an explicit `NOT_PROMOTION_READY` with failed cohorts and first-loss evidence.

Never report an unverified target as supported.

## 12. Handoff

Release-candidate verification receives artifacts/checksums/build manifest, the verified platform and host matrix, install/project/hook/upgrade docs, four-tool smoke steps, API-key-free FTS path, explicit paid-hybrid environment/cost guidance, paired marginal-usefulness evidence, the updated promotion result, and known packaging limitations and unresolved policy items.

If operational feedback establishes a real requirement for permanent model pinning or external vector supply, create a new architecture decision first. Do not reinterpret the v1 import/packaging contract retroactively.

## 13. Decision Log

| Decision | Rationale | Revisit when |
| --- | --- | --- |
| Bundle FTS5 and grammars | Keep the free core independent of system installs and runtime downloads | Platform constraints block real distribution |
| Officially document project scope only | Reduce wrong-root and multi-repository confusion | Safe user-scope root routing is designed |
| Require explicit `--root` | Avoid accidental host cwd | Core v1 invariant |
| Do not edit host config | Avoid host-specific merge/removal risk | A verified installer is separately designed |
| One process per repository | Keep state, DB, and authority boundaries simple | Multi-repository service is separately designed |
| No runtime dependency download | Preserve offline operation, reproducibility, and supply-chain boundary | An explicit plugin system is approved |
| Fixed model versus external supply stays out of scope | Do not mix initial lab needs with long-term distribution | Real production requirements appear |
| No speculative vector import format | Avoid unvalidated integrity/security compatibility | A separate ADR and provenance design are approved |
| Evaluate marginal assistant value | cidx is an auxiliary tool, so existing tools remain the product baseline and cidx use must not be forced | The product role changes |
| Runtime checks use disposable local state | FTS5/WAL and all embedded grammars must fail before repository mutation or production migration, without downloads or repairs | A future runtime changes the bundled dependency boundary |
| Owner selected Apache-2.0; root license and local package checkpoint recorded | The unmodified root `LICENSE` supplies cidx's terms while third-party notices remain separate; local verification is limited to darwin/arm64 | Another release target, distribution policy, or owner terms require review |
| CLI-only provenance report | Build facts are needed for package verification, while Phase 13's MCP `serverInfo` and four-tool surface remain frozen | The MCP version contract is separately revised |
| Verify both current product profiles from one source bank | Default 1024/int8 and explicit 512/int8 must work in the installed binary without making source storage a serving dependency | The source/serving contract changes |
| Keep Binary/256 package checks negative-only | Retired experimental profiles remain historical evidence and cannot regain an executable product entry point | A new measured design decision explicitly reauthorizes them |
