# Assistant Search-to-Evidence Flow Design

- Status: `implementation_step_2_complete`; Step 3 remains owner-gated
- Date: 2026-08-22
- Owner direction: replace the closed mandatory-cidx-first experiment with a
  host-decided, stage-separated search-to-evidence design
- Product scope: local repository MCP for Go, TypeScript, and TSX
- Implementation authority: Section 14 Step 2 is implemented and validated:
  neutral tool descriptions plus passive harness trace/reducer work. Question
  construction, scored runs, new test code, and provider operations remain
  outside this completed checkpoint
- Provider action authorized by this document: none
- Historical boundary: Assistant A/B V4-V6 remains closed and unchanged; this
  document is not V7 and does not regrade or rerun that series

## 1. Decision Summary

cidx must not be product-labeled as a tool that an AI may use only as a minor
auxiliary. It also must not force itself to be the first or primary search path.
The MCP exposes a precise local retrieval capability; the calling AI or host
decides whether a given task is best served by cidx, ordinary file/symbol tools,
or both.

The next design target is an evidence funnel:

```text
user question
-> host decides whether cidx is useful
-> compact ranked candidate locators
-> host selects the smallest plausible candidate set
-> complete source evidence for selected candidates
-> only the directly required neighboring/type/dependency evidence
-> answer whose material claims are bounded by observed evidence
```

The optimization target is the complete investigation journey, not the byte
size of a single MCP response. Search should narrow the repository with small,
accurate candidate cards. Once a candidate is selected, cidx should make it
cheap to obtain enough complete evidence that the AI does not repeat the same
work with ordinary tools or infer dependency behavior it has not inspected.

## 2. Why the V4-V6 Result Does Not Set the Product Role

The closed series tested a narrow host policy: every treatment task began with
cidx, and later versions constrained how the assistant searched and read.
Those experiments established useful mechanics but did not establish that the
mandatory-first policy was consistently token-efficient.

The result does not establish any of the following:

- that cidx should be restricted to rare or secondary use;
- that a host should always use cidx first;
- that locator-only navigation is sufficient evidence for an answer;
- that the local FTS or dense retrieval architecture is unnecessary; or
- that smaller search payloads necessarily reduce end-to-end model tokens.

The most important V6 failure occurred after successful navigation. cidx found
the correct `Form` code, but the answer asserted semantics of a dependency that
had not been inspected. That exposes an evidence-acquisition gap, not a target
locator failure. A design that only minimizes locator bytes can therefore make
the AI perform a second exploration or encourage an unsupported inference.

## 3. Goals and Priority Order

1. Preserve or improve answer correctness and material-claim support.
2. Narrow a repository to a small set of relevant functions, methods, or types.
3. Deliver complete selected evidence at a bounded, useful volume.
4. Reduce duplicate exploration across cidx and ordinary repository tools.
5. Reduce end-to-end source exposure, tool work, and model tokens when the task
   benefits from cidx.
6. Measure whether the AI elects to use cidx when it is freely available.

Low tool adoption is not itself a failure. A small number of precise uses can
be valuable if they shorten difficult investigations or improve correctness.
Conversely, a high call count is not success if the calls do not reduce the
investigation scope.

## 4. Fixed Product Boundaries

This direction preserves the following v1 invariants:

- SQLite remains the sole persistent retrieval authority.
- One search observes one committed active generation.
- MCP continues to expose exactly `status`, `search`, `read_span`, and
  `reindex`; this design adds no fifth tool.
- Free AST/FTS indexing and FTS search remain local and provider-free.
- Hybrid search remains explicit. Its vector comparison is local, but a new
  natural-language query requires a paid Voyage `voyage-code-4` query
  embedding in the same space as the stored document vectors.
- Search ranking and result identity do not depend on a source-body budget.
- `read_span` remains live-file, path-safe, all-or-nothing, and guarded by the
  indexed whole-file SHA-256.
- The repository config owns the per-call server hard maximum. The calling AI
  or host owns any smaller per-session source/context budget.
- Raw BM25, dense, RRF, and planner diagnostics remain internal retrieval and
  evaluation data. They are not source evidence and are not included in the
  normal assistant response.

No server-side conversation store, persistent evidence graph, or second
in-memory retrieval authority is introduced.

## 5. Three-Stage Interaction Contract

### 5.1 Stage A: candidate discovery

`search` answers only: **where should the AI inspect next?**

The initial response remains a ranked, deduplicated list of compact canonical
locators:

```text
chunk_id
path
language
kind
qualified_symbol
start_line
end_line
indexed_sha256
match_sources
```

The array order is rank. `match_sources` is routing provenance, not proof that
the code satisfies the question. Raw scores, planner terms, signatures, source
bodies, and profile diagnostics remain absent.

The first 30-question run freezes the current configured default `k=10` and
absolute maximum 20. The caller may still select a valid `k`, and that choice
is observed rather than silently rewritten. Compact locator volume is already
small enough that lowering the default before measurement would risk candidate
loss without solving the observed evidence problem. A later default-`k` change
is a separate measured intervention.

### 5.2 Stage B: selected evidence acquisition

The AI selects one or more plausible locators and calls `read_span` with the
exact `path`, parent range, and `indexed_sha256` copied from each locator.
`read_span` returns the complete selected range when it fits the configured
hard maximum. The normal evidence unit is therefore one complete named
function, method, or type, not a tiny snippet chosen only to save tokens.

If a semantic parent exceeds the per-call maximum, the caller may request
ordered, non-overlapping subranges under the same hash. The server does not
silently truncate or automatically stream an arbitrary partial body.

### 5.3 Stage C: justified evidence expansion

After reading a selected parent, the AI classifies the remaining uncertainty:

- **No unresolved material claim:** answer and stop.
- **Same-file lifecycle or wrapper context:** read only the non-overlapping
  adjacent range needed for that claim, or locate the neighboring named parent
  through another compact search.
- **Referenced type or external dependency:** search the observed identifier or
  qualified name, then read the selected dependency parent.
- **Ambiguous declaration/implementation pair:** search again with the exact
  symbol/path evidence learned from the first body.
- **Potential conflict or hard negative:** inspect the smallest competing
  locator set needed to resolve the conflict.

A new search or read is justified by an unresolved answer claim, not by a
generic desire to become more certain. There is no global cap on distinct
locators because a legitimate multi-hop task may require several. Duplicate
and overlapping work is controlled separately so a cap for one pathological
task cannot damage other tasks.

## 6. Evidence Sufficiency and Stopping

The host should treat every material answer statement as one of:

- `observed`: directly supported by a read source range;
- `derived`: a bounded inference whose premises are all observed; or
- `unresolved`: requires another read or must be omitted/qualified.

The answer may stop when all statements required by the question are observed
or safely derived and no unresolved statement is planned for inclusion.

During evaluation, the blind-grade packet records one
`observed | derived | unresolved` classification and supporting source ranges
for every material final-answer claim. This classification is assigned after
the turn; it is not supplied to the assistant while it works.

This rule specifically prevents the V6 `Form` failure mode. Seeing a call to a
dependency does not establish that dependency's validation, transformation,
error, or state semantics. The AI must either read the dependency or avoid that
claim. It should not read the dependency if the answer does not need the claim.

## 7. Session-Local Evidence Ledger

The duplicate-control and claim boundary belong to the calling session or the
evaluation harness, not SQLite. A minimal ephemeral ledger is sufficient:

```text
SearchObservation
  query / mode / k / ordered locator keys

CandidateObservation
  locator key / selected-for-read | unread

ReadObservation
  path / indexed_sha256 / requested range / delivered bytes
  overlap with prior reads

PostGradeAnnotation
  accepted-evidence locator/range / cited range
  stable local claim id / observed | derived | unresolved
  supporting read keys
```

Canonical keys are derived from data already on the MCP wire:

```text
locator key = indexed_sha256 + path + qualified_symbol + parent range
read key    = indexed_sha256 + path + requested range
```

The live ledger may exist only in the assistant harness, host adapter, or
current agent context. It never becomes product or server session state, never
writes to SQLite, is never consulted by the MCP server, and has zero effect on
ranking, locator identity, or `read_span` behavior. For reproducible evaluation
only, the harness retains a truth-free snapshot and its digest under the
ignored local run artifacts. That snapshot contains coordinates, hashes,
counts, and byte accounting but no copied source body; it is not a serving
authority and is never supplied to the model.

For the first 30-question evaluation it is strictly passive and non-model-
facing. It records searches, reads, bytes, ranges, overlap, and action order but
does not reject or suppress a call, inject reminders, rewrite prompts, alter a
tool response, or tell the assistant that evidence is complete. Candidate
usefulness and material-claim support are joined to the passive trace only
after blind grading.

A later host-orchestration study may separately test active duplicate warnings
or claim-state guidance. That would be a new intervention, not part of the
initial tool-availability estimate and not a server-side graph product.

## 8. Tool Descriptions and Host Behavior

The MCP descriptions are pure capability statements with no usage preference:

- `search`: returns ranked function, method, and type locators for a repository
  query without source text; hybrid mode may require an explicit paid Voyage
  query embedding when enabled.
- `read_span`: returns a complete current source range when the path, range,
  indexed SHA-256, and configured byte maximum are valid.

Tool-description text is model-visible interface state. Its exact bytes and
digest are versioned and frozen with the tool schema in every assistant run.

The host prompt must not require a cidx call. It also must not describe cidx as
last-resort or secondary-only. The assistant is free to:

- answer a trivial direct lookup with existing tools and never call cidx;
- use cidx as the first and main repository narrowing tool;
- combine cidx candidates with language-server, compiler, test, or file tools;
  or
- return to cidx for a dependency after ordinary inspection reveals a useful
  identifier.

The treatment being evaluated is **tool availability plus an accurate
interface**, not mandatory tool use.

The shared task prompt may require source-backed material claims for both arms,
but it contains no cidx-specific usage or stopping rule. Candidate-selection
and evidence-expansion guidance in this design is not injected as a treatment-
only orchestration paragraph in the first evaluation.

## 9. Source-Volume Contract

The server enforces only safety maxima. It must not push every candidate body
or guess the caller's total context allowance.

- Candidate search: compact locators only.
- Evidence read: one complete caller-selected range, subject to the configured
  `mcp.hard_max_inline_bytes` per-call maximum and 1 MiB code-owned ceiling.
- Session total: chosen by the calling AI/host and frozen by an evaluation
  manifest when comparing arms.
- Rank and locator identity: independent of source-volume settings.

The desired outcome is not minimum source bytes at any cost. It is high useful
evidence density: enough complete code to avoid duplicate work, with unrelated
candidate bodies and diagnostics excluded.

## 10. FTS and Dense Sequencing

The next assistant-flow experiment is FTS-only and provider-free. This isolates
candidate selection and evidence acquisition from paid-query availability. Its
frozen execution config sets the default mode to FTS, disables paid query
embedding, exposes no `VOYAGE_API_KEY` or query client, and records requested
and effective mode plus fallback reason. A hybrid request cannot create
provider traffic; it deterministically remains unavailable or falls back to
FTS under the existing product contract and is recorded as a protocol
deviation. No credential is present to make the boundary dependent on prompt
compliance.

Dense/hybrid remains a planned complementary lane, not a discarded feature:

1. improve and measure the FTS candidate/evidence flow;
2. establish whether compact candidate selection and complete evidence reduce
   the assistant's exploration scope;
3. only with separate paid-query approval, run the same stage metrics with
   hybrid availability; and
4. compare FTS-only, dense contribution, and fused results without letting RRF
   hide a broken lane.

Document embeddings already exist in the Voyage vector space. A changing
natural-language query still needs a Voyage query embedding to participate in
that same space; the subsequent exhaustive vector scan and RRF are local.

## 11. Initial Implementation Boundary

The first implementation candidate intentionally uses the existing public
nine-field `search` locator and existing `read_span` source contract. It needs
no new database table, relation graph, MCP tool, or source-bearing search
response.

The first changes to evaluate are limited to:

1. neutral, pure-capability MCP tool descriptions;
2. a passive, non-model-facing session trace in the assistant evaluation
   harness;
3. post-turn blind claim-support classification; and
4. stage-separated journey reduction and reporting.

Potential wire additions are deferred until this flow shows a measured first
loss that existing fields cannot resolve. If needed, test one addition at a
time in this order:

1. compact function/type signature in candidate cards, if locator ambiguity
   causes excess reads;
2. same-file adjacent parent locators in a `read_span` response, if
   non-overlapping local context repeatedly requires ordinary rescans; and
3. stable dependency locator hints, only if a provider-free, source-derived
   implementation can be proven useful without becoming a graph authority.

Any such addition changes the public MCP schema, reopens Phase 13, and requires
focused compatibility and representation validation before assistant use.

## 12. Thirty-Question Assistant Availability Evaluation

The next evaluation is a new series, not V7. It uses the already approved Go
and TypeScript/TSX repositories; no new repository is required.

### 12.1 Question set

Create a new versioned 30-question set and preserve the existing 12-question
sets and results unchanged. Every run records the exact question-set ID,
version, canonical digest, corpus commit/tree/content identities, model,
prompt, tool schemas, and reducer version.

Use 10 questions per language slice: Go, TypeScript, and TSX. Within each
slice include:

| Question shape | Count | Purpose |
| --- | ---: | --- |
| exact identifier or path lookup | 2 | Observe when ordinary tools may be cheaper and whether cidx abstains naturally |
| semantic single-hop behavior | 2 | Test natural-language narrowing to one parent |
| dependency or type multi-hop | 2 | Test selected evidence plus justified expansion |
| contract, lifecycle, state, or error flow | 2 | Test complete evidence and claim boundaries |
| ambiguous/disambiguation case | 1 | Test competing locators without a global read cap |
| verified no-answer or hard-negative case | 1 | Measure false leads and unnecessary exploration |

Questions should vary in wording and required evidence count. Do not add a
repository merely to fill cells, and do not rewrite earlier results. A changed
question set receives a new version and new run IDs.

Question authoring is source-first. Authors inspect the frozen repository
source directly and do not consult cidx retrieval results or prior assistant
outputs while constructing the new set. Exact wording, required groups,
accepted alternatives, hard negatives, grading truth, and source identities
freeze before any scored assistant or cidx output is observed. Every question
has exactly one primary shape from the table; multi-hop, contract,
disambiguation, and hard-negative properties may also be recorded as
nonexclusive modifiers.

### 12.2 Arms and execution

Run 30 frozen pairs, 60 assistant turns total:

- **A — existing tools:** the fixed Codex CLI repository tools without cidx.
- **B — existing tools plus cidx:** the same model, reasoning effort, task
  prompt, budgets, corpus snapshot, and ordinary tools, with the four cidx MCP
  tools available under the neutral descriptions in Section 8.

Do not require or reward a cidx call. No-use treatment turns remain valid and
stay in the intent-to-treat denominator. Randomize or counterbalance arm order
under a frozen schedule. Blind grading sees task truth, answer, and cited
evidence but not arm, cidx use, token counts, or execution order.

The primary comparison is all 30 frozen pairs. Results limited to treatment
turns that elected to use cidx are descriptive because tool selection is not
random and creates selection bias.

Every scored execution starts with a fresh opaque source copy and fresh model
context. Treatment state is a private copy of the same frozen cidx index; no
state, transcript, or cache created by another turn is visible. Grading truth
and evaluation artifacts are outside the assistant-visible filesystem.
Ordinary repository tools, environment, task/context budgets, timeout, and
permissions are identical; only the four frozen cidx tools and their frozen
descriptions/schemas differ. The manifest binds the Codex CLI, model/reasoning,
cidx binary, config, tool interface, prompt, repository snapshot, index, and
reducer digests. Each pair runs consecutively under the predeclared
counterbalanced schedule, and neither arm is selectively retried or replaced.

### 12.3 Metrics

Report these surfaces separately; do not create a weighted total.

**Answer quality**

- complete/partial/incorrect/ungradable;
- required-group coverage;
- correct file, symbol, behavior, and requested action;
- unsupported and contradicted material-claim counts;
- unsupported-material-claim rate, independent of required-group coverage;
- `observed | derived | unresolved` status for every material claim; and
- citation-to-claim support.

**Candidate navigation, only when cidx is used**

- first-search and any-search required-locator coverage;
- first useful locator rank;
- unique locator count, duplicate exposure, and selected-locator utilization;
- refined-search count and the evidence reason for each refinement; and
- false-lead candidate inspections.

**Evidence acquisition**

- complete selected-parent evidence;
- dependency/type evidence acquired when the answer relies on it;
- successful, failed, duplicate, and overlapping reads;
- gross and unique cidx source bytes;
- read precision and evidence-to-citation utilization; and
- first point at which every required claim had sufficient evidence.

**End-to-end exploration scope**

- unique repository files, named parents, and source ranges inspected;
- unique source bytes delivered by cidx and ordinary tools;
- ordinary grep/search/file-read actions before and after useful cidx evidence;
- repeated acquisition of code already delivered by cidx; and
- total repository tool calls.

**Assistant availability and efficiency**

- cidx adoption rate and first-use position by question shape;
- no-use outcomes;
- official input, cached-input, output, and total model tokens;
- wall time and provider usage as observations, not current release gates; and
- paired per-task deltas rather than aggregate totals alone.

### 12.4 Metric authority and denominators

Use the existing canonical locator and evidence sets from
`EVALUATION-CONTRACT.md`. For task `q`:

```text
Uq = unique locators returned by all cidx searches
Rq = unique locators whose ranges were successfully read
Eq = unique source ranges actually delivered to the model
Cq = unique source ranges cited in the final answer
Gq = frozen required evidence groups and reviewed alternatives
Mq = material final-answer claims identified by the blind grader
```

The passive runner mechanically supplies calls, arguments, ordered locators,
successful responses, actual model-visible output bytes, time ordering, and
range overlap. The frozen truth supplies accepted locator/range alternatives.
The blind grader alone supplies material-claim units and claim support.

| Measurement | Numerator / denominator | Authority |
| --- | --- | --- |
| Locator selection utilization | unique returned locators in `Rq` / `|Uq|` | mechanical trace |
| Read precision | successful read ranges intersecting a reviewed accepted alternative / all successful read ranges | trace + frozen truth |
| Evidence requirement coverage | groups in `Gq` intersecting `Eq` / `|Gq|` | trace + frozen truth |
| Citation utilization | unique delivered ranges intersecting `Cq` / `|Eq|` | trace + final citations |
| Navigation false-lead proxy | read locators neither accepted nor cited / `|Rq|` | trace + frozen truth + citations |
| Duplicate locator exposure | repeated canonical locator occurrences after the first / all locator occurrences | mechanical trace |
| Redundant source ratio | gross delivered source bytes minus unique delivered bytes / gross delivered bytes | mechanical trace |
| Unsupported-claim rate | unsupported claims in `Mq` / `|Mq|` | blind grade; `NOT_OBSERVED` when `|Mq|=0` |

“First useful locator rank” means the first locator that matches a frozen
accepted alternative, not the first item later mentioned by the assistant.
“Complete evidence” means every frozen group intersects actually delivered
source. “First sufficient-evidence point” is the earliest ordered event where
that condition becomes true; final material-claim support remains a separate
blind-grade outcome. Visible-but-uncited source is not automatically a false
lead, so the navigation false-lead value is an operational proxy rather than
proof that every counted read was semantically useless. Model-visible bytes
are counted from actual tool/shell result payloads
delivered by the host, never inferred from command text alone.

## 13. Evaluation Interpretation

This 30-question series is calibration and product-direction evidence, not a
release or retrieval-promotion result. It should answer four separate
questions:

1. Does cidx make the correct target easier to locate?
2. Once located, does the staged flow provide enough evidence for the answer?
3. Does it reduce the total repository exploration scope without increasing
   unsupported claims?
4. When free to choose, on which question shapes does the AI use cidx and gain
   value?

Do not conclude that cidx failed merely because every retrieval query is not
perfect or because the AI skips it on easy tasks. Do not claim success from
adoption, locator rank, response-byte reduction, or aggregate token reduction
alone.

The next product decision should be based on paired correctness plus the
location of the first loss:

- candidate miss/ranking;
- candidate selected but insufficient source evidence;
- required dependency not acquired;
- duplicate or post-sufficient exploration;
- unsupported final claim; or
- no measured advantage over existing tools.

Numeric promotion margins, if later needed, are calibrated and frozen before
an independent confirmation set. This exposed 30-question series cannot tune
itself into promotion evidence.

## 14. Implementation and Validation Sequence

### Step 1 — contract reconciliation

- Accept this reviewed direction.
- Update the canonical product-role wording so it describes host choice rather
  than a mandated auxiliary or first-tool role.
- Keep the current four-tool and SQLite boundaries.

Evidence: documentation cross-reference and conflict audit.

### Step 2 — interface and session ledger

- Revise tool descriptions without changing schemas.
- Add passive session-trace/reducer fields to the assistant harness.
- Keep the trace non-model-facing; defer active duplicate or claim guidance to
  a separately frozen orchestration study.
- Prove no production SQLite write or server conversation state is introduced.

Evidence: exact schema comparison, focused host representation probe, and
ledger fixture/reducer report. Test-code authorization must be obtained before
creating new tests; existing checks may be run.

### Step 3 — question-set preparation and freeze

- Prepare the 30 questions against the existing approved corpora.
- Author source-first without cidx results, then review required groups, valid
  alternatives, material-claim truth, and hard negatives.
- Version and freeze the question set and execution schedule before scored
  assistant output is observed.

Evidence: manifests, digests, review/adoption record, and no scored-turn gate.

### Step 4 — paired execution and blind grading

- Run the 60 turns once under the frozen controls.
- Preserve no-use, failure, and timeout results.
- Blind grade answer quality and claims, then reduce the journeys.

Evidence: immutable run records, grades, stage metrics, artifact checksums, and
per-pair report.

### Step 5 — first-loss decision

- Decide whether the next constraint is retrieval quality, candidate metadata,
  evidence expansion, host orchestration, or no demonstrated marginal value.
- Add signature/neighbor/dependency metadata only if the measured first loss
  requires it.
- Request separate approval before any paid hybrid run.

Evidence: explicit decision with rejected alternatives and remaining risks.

## 15. Non-goals

- No V7 rerun or regrade of V4-V6.
- No new evaluation repository in this stage.
- No forced cidx-first or cidx-last policy.
- No global maximum on distinct evidence locators.
- No source body in the candidate-discovery response.
- No raw planner/score diagnostics presented as evidence.
- No server-persisted session, evidence graph, or second authority.
- No new MCP tool or automatic filesystem watch.
- No paid query embedding or hybrid execution without explicit approval.
- No claim that token reduction is more important than correctness and
  sufficient evidence.

## 16. Questions Submitted for External Review

External reviewers should challenge the design on these exact points:

1. Are the existing nine locator fields sufficient for first-stage candidate
   selection, or is a compact signature necessary before the 30-question run?
2. Can exact `read_span` plus refined symbol/path search provide adequate
   dependency evidence without same-file neighbor metadata?
3. Does the passive session-local ledger measure duplicate exploration without
   adding server complexity or constraining legitimate multi-hop tasks?
4. Are answer correctness, exploration scope, evidence sufficiency, adoption,
   and official tokens separated well enough to diagnose the first loss?
5. Does any part of the design accidentally force cidx use or demote it to a
   secondary-only role?
6. What is the smallest material correction required before implementation?

## 17. External Review Record

The same complete draft was sent to the existing ChatGPT and Grok side-panel
conversations. Both returned `ACCEPT_WITH_CORRECTIONS`, found no fundamental
conflict with the four-tool/SQLite/paid-provider contracts, and agreed that the
current nine locator fields and exact `read_span` are sufficient for the first
evaluation.

The reconciled corrections are now incorporated:

- execution isolation and source-first question precommitment;
- a passive, non-model-facing initial session trace, with active orchestration
  deferred;
- pure-capability, versioned tool descriptions;
- explicit metric numerators, denominators, and authorities;
- independent unsupported-claim and claim-support accounting;
- executable FTS-only/no-credential enforcement; and
- the first-run `k=10`, absolute-max-20 configuration freeze.

The revised complete document was returned to both conversations. ChatGPT and
Grok independently returned `FINAL_ACCEPT`. Their nonblocking cautions retain
post-grade claim fields outside the live trace, keep the first-sufficient-
evidence event secondary to blind claim support, treat false-lead accounting as
an operational proxy, keep cidx-user-only slices descriptive, and require any
future active orchestration to be a separate frozen intervention. These
clarifications are incorporated above. The detailed packet, findings,
dispositions, and final replies are retained in
[`evidence/phase-14/assistant-search-evidence-flow-external-review.md`](evidence/phase-14/assistant-search-evidence-flow-external-review.md).

## 18. Step 2 Implementation Record

Section 14 Step 2 is complete under the exact boundary above. The public MCP
still exposes four tools and the same input schemas. Only model-visible
capability descriptions changed. The assistant harness now retains a passive,
body-free trace in ignored evaluation state, verifies returned `read_span`
identity and source bytes, conservatively attributes ordinary-tool output, and
reduces candidate, evidence, exploration, adoption, and blind claim-support
surfaces separately.

The trace is not sent to the model, is not written to production SQLite, and is
not server conversation state. It records no frozen truth or source body.
Truth and claim grades are joined only after the trace and blind grading inputs
are frozen. Historical V3–V6 aggregates and reports remain unchanged under
protocol-specific replay.

Exact code scope, validation commands, compatibility results, checks not run,
and the remaining Step 3 gate are recorded in the
[Step 2 implementation evidence](evidence/phase-14/assistant-search-evidence-flow-step-2-implementation.md).
