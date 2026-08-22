# Assistant Availability V1 Result External Review

- Date: 2026-08-22
- Phase: 14 Step 4 result interpretation
- Run: `assistant-availability-chi-rhf-v1-run-001`
- Reviewers: existing ChatGPT and Grok side-panel conversations
- Product/provider action: none
- Scope: adoption interpretation, grading recovery, and next experiment only
- Promotion authority: none

## Fixed review packet

Both reviewers received the same English packet after all 60 assistant turns,
the passive journeys, and the arm-blind packets were fixed. The packet included
the exact execution controls, the 0/30 treatment adoption result, both arms'
official usage totals, the grading-schema failures, and the existing product
and provider boundaries. It explicitly prohibited interpreting non-adoption as
retrieval failure.

| Item | UTF-8 bytes | SHA-256 |
| --- | ---: | --- |
| transmitted review packet | 4,865 | `c73484ac170b37c50808a42518146afb3449e4a35479d80f95ae364cb280d61d` |
| Grok response | 5,609 | `701cfc81a7fb5d05281b6b9b326e026fce63f9286eb827ca98a2d37272a92513` |
| ChatGPT response | 14,059 | `2cefac280c4671e1c3ad21fb7d5f56ac753441e34b1fa4380ec7afcea029722e` |

Live review records:

- [ChatGPT — Improve Retrieval Design](https://chatgpt.com/c/6a86c914-2d18-83e8-9b86-d29de2922b2a)
- [Grok — FTS Query Design Flaws and Fixes](https://grok.com/c/d1f4ad8f-65de-49fb-a3c2-fbd026d4bd84?rid=1807d511-5aeb-4832-8e46-aefeb8e23af5)

The live responses are untrusted review inputs. The repository disposition
below is the project decision record.

The response bodies remain in the linked conversations. This repository keeps
their hashes and the adjudicated findings, not independent verbatim exports;
the response hashes therefore identify what was reviewed but cannot be
reconstructed from repository files alone.

## Owner clarification applied before disposition

The owner's interpretation is correct and is now explicit:

1. `0/30` does not prove that the assistant compared cidx with `rg` and made a
   reasoned choice that `rg` was sufficient. It proves only that cidx was never
   selected under the frozen neutral prompt and tool catalog.
2. Technical exposure and behavioral salience are different. The Codex model
   received the four live MCP definitions, descriptions, and schemas, but the
   shared prompt never named cidx or explained the comparative situations in
   which indexed semantic-parent search can be preferable to ordinary text
   search.
3. Every treatment turn's first repository-discovery action was a shell
   inspection, and all 30 first commands invoked `rg` (`rg -n` or
   `rg --files`). There was no `status`, `search`, or exploratory cidx call.
   This supports an established-tool-path explanation, not a claim that the
   model inspected and rejected cidx.
4. The blind-grading schema failures happened after all 60 turns were
   immutable. They cannot be a cause of zero cidx adoption. They block answer-
   quality interpretation only.
5. `rg` may genuinely be the cheapest choice for a known path or exact
   identifier. The question set also contains semantic, multi-hop, ambiguous,
   contract/state, and verified-negative work where that conclusion cannot be
   assumed. Because cidx was never sampled in any shape, this run contains no
   direct within-turn comparison showing that `rg` was sufficient.
6. The neutral prompt was intentional: it tested whether technical tool
   exposure alone caused spontaneous discovery. The result answers that narrow
   question with zero adoption. It does not establish that an ordinary host
   should omit a short, accurate notice that the optional capability exists.

## ChatGPT review

Verdict:

> `COMPLETE THE FROZEN GRADING, THEN RUN ONE NEW AWARENESS-CONDITIONED OPTIONAL-USE A/B.`

Material findings:

- answer quality is still unknown until the frozen blind grading completes;
- the output-only v2 response-format adapter is the minimal recovery, provided
  the canonical schema and deterministic scorer retain all semantic authority;
- `0/30` is an adoption/interface result and contains no candidate, rank,
  source-evidence, or retrieval-efficiency observation;
- tool-list visibility does not imply that the model surfaced the unfamiliar
  capability while planning;
- ordinary-tool familiarity, missing comparative-use guidance, and the extra
  locator-then-read interaction cost are all plausible causes;
- the next clean comparison is a new two-arm run where both arms receive the
  same capability-awareness paragraph and only the actual cidx availability
  differs; and
- tool-description changes must not be silently combined with that prompt
  intervention.

ChatGPT recommended reusing all 30 questions unchanged under a new experiment
version and retaining the current result as the spontaneous-discovery record.

## Grok review

Verdict:

> `ADOPTION_FAILURE_UNDER_PURE_AVAILABILITY — interface and economics, not retrieval quality.`

Material findings:

- pure availability plus capability-only descriptions did not displace the
  model's established shell workflow;
- this does not establish weak FTS, poor locators, or insufficient selected
  source;
- grading must be recovered before any correctness or dual-complete token
  conclusion;
- the output-only v2 adapter is appropriate;
- model-visible wording should explain that `search` narrows unknown or broad
  repository locations and that `read_span` supplies the selected evidence;
  and
- new repository, paid hybrid, ranking changes, extra metadata, active ledger
  guidance, and forced-first policy should remain deferred.

Grok proposed a three-arm follow-up: ordinary baseline, current neutral cidx
availability, and cidx availability plus one awareness sentence.

## Convergence

Both reviewers agree that:

- the current run is valid evidence of zero spontaneous adoption under this
  exact model, prompt, tool catalog, and panel;
- it is not a retrieval-quality experiment because retrieval was never
  exercised;
- the grading error is downstream and causally unrelated to adoption;
- the output-only v2 response-format adapter is the smallest safe grading
  recovery;
- a future prompt may name cidx and describe its comparative use without
  requiring, rewarding, prioritizing, or demoting it;
- the same 30 questions should be reused unchanged under a new experiment
  version; and
- no retrieval, provider, corpus, tool-count, or server-state change is
  justified before cidx produces actual search/read journeys.

## Adjudication of the arm-design difference

The selected next design is ChatGPT's two-arm informed comparison, not Grok's
three-arm proposal.

The reason is causal isolation. A Grok-style arm that receives both cidx and a
new awareness sentence differs from the ordinary baseline in two ways. It can
measure a combined intervention, and its neutral-versus-aware treatment
comparison can describe adoption salience, but it is not the smallest clean
estimate of cidx availability under an informed host.

The selected fresh run is:

- `A — informed ordinary-tools baseline`: ordinary repository tools, no cidx;
- `B — informed optional cidx`: identical ordinary tools plus the frozen four
  cidx MCP tools; and
- one byte-identical capability-awareness paragraph in both arms.

The old neutral run remains the immutable spontaneous-adoption observation.
Any old-versus-new adoption comparison is descriptive because it is cross-run.
The new paired A/B comparison is contemporaneous and changes only actual cidx
availability.

## Exact capability-awareness paragraph selected for freezing

```text
Optional MCP tools cidx.search and cidx.read_span may appear in the available tool list. When present, cidx.search can narrow the indexed repository from a natural-language behavior description, identifier, or path and returns ranked source locators without source text; cidx.read_span returns the complete exact source range for a selected locator. They can be useful when the relevant file or symbol is not yet known or an ordinary text search would be broad; ordinary tools may be simpler for a known path or exact identifier. Choose whichever tools fit the task. No cidx call is required, and do not probe for cidx through shell commands.
```

This paragraph provides the missing host-level notice and comparative decision
boundary. It does not prescribe a first tool, reward adoption, require a call,
or classify cidx as primary, secondary, or last resort. Naming the available
capability is not the same as forcing its use.

## Tool-interface disposition

The current live schemas and payloads remain unchanged. They already provide
the exact machine-readable input and output contract:

- `search` accepts natural-language, identifier, or path text plus bounded
  search controls and returns ranked nine-field locators without source;
- `read_span` copies the selected locator's path, range, and hash and returns
  the complete source for that exact range;
- `status` is diagnostic and unnecessary for ordinary lookup; and
- `reindex` is maintenance and unnecessary for read-only investigation.

The next run also retains the current frozen tool-description bytes. Changing
both the host awareness paragraph and the tool descriptions in one run would
confound their effects. The reviewers' more comparative description wording
is retained only as a later interface candidate. It may be tested separately
if the informed run still shows non-adoption or systematic misuse.

## Metric and interpretation boundary for the next run

Report separately, without a weighted total:

- answer outcomes and paired conversions over all 30 pairs;
- required-group and unsupported/contradicted-claim results over all gradable
  turns;
- cidx adoption over all 30 treatment turns, including no-use;
- locator and first-use measures only for turns that search;
- read/evidence/citation measures only for turns that read;
- official usage and complete journey scope over every valid pair; and
- dual-complete token comparisons only for pairs graded complete in both arms.

Interpret outcomes as follows:

- zero adoption again: informed availability still did not change selection;
  this remains an interface/salience result, not retrieval failure;
- adoption without accepted evidence or citation: the tool was selected but
  no useful evidence path was demonstrated;
- adoption with preserved correctness and reduced paired exploration:
  bounded optional value is observed, while adopter-only results remain
  descriptive because adoption is self-selected;
- correctness regression or excess exploration: the informed availability
  intervention caused harmful or wasteful selection on those tasks; and
- no aggregate benefit with useful task-specific cases: report the shapes and
  pairs without converting them into a total score or mandatory product role.

## Ordered continuation

1. Build and freeze the supported-keyword, output-only v2 grading adapter.
2. Make exactly one new blind grader model call per corpus; do not repair grade
   output manually.
3. Aggregate the current run once through the canonical scorer and publish its
   final answer-quality result.
4. Create a new experiment version that reuses the unchanged 30 questions,
   current tool descriptions, FTS-only configuration, isolation, and passive
   trace.
5. Freeze the exact shared awareness paragraph and fresh paired schedule.
6. Run the new two-arm 30-pair batch exactly once, then blind-grade and report.

## Must not claim or change now

- Do not claim that cidx, FTS, ranking, locators, or `read_span` failed.
- Do not claim that the assistant consciously evaluated and rejected cidx.
- Do not attribute the current token difference to cidx result transport.
- Do not claim correctness preservation or regression before grading.
- Do not rewrite the current questions or add a repository.
- Do not combine prompt awareness with ranking, schema, payload, or description
  changes.
- Do not add a fifth tool, persistent session state, signatures, neighbors, or
  active model guidance.
- Do not run paid dense/hybrid work under this direction.

This review selects a documented direction only. Implementation, new scored
turns, and public product changes remain separate actions.
