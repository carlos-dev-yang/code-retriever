# Forced cidx Prompt Diagnostic V1

- Status: `frozen_for_execution`; schema probes and execution pending
- Date: 2026-08-23
- Phase: 14
- Experiment class: paired, non-promotion prompt-policy diagnostic
- Question set: unchanged `assistant-availability-chi-rhf-v1`, 30 questions
- Corpora: existing approved chi v5.3.1 and react-hook-form v7.85.0
- Provider boundary: FTS-only, no Voyage credential or request
- Product authority: none; this diagnostic cannot replace optional-use evidence

## 1. Owner question

When the same Codex CLI assistant has the same four cidx tools in both arms,
does an explicit instruction to use cidx instead of `rg` for repository code
search improve answer quality, evidence acquisition, exploration scope, or
official token use compared with the current neutral prompt?

The diagnostic isolates prompt policy. It does not compare tool availability:
cidx is present in both arms with byte-identical descriptions, schemas,
configuration, index, and source state.

## 2. Arms and exact prompt difference

- **`neutral_cidx`:** the existing neutral prompt and the four cidx tools.
- **`directed_cidx`:** the same rendered prompt plus the exact policy paragraph
  below, with the same four cidx tools.

The exact directed-arm addition is:

```text
For repository code discovery, use cidx.search rather than rg, grep, git grep, find, fd, or other ordinary text/file search. Use cidx.read_span to inspect a locator you select; ordinary tools remain available only for non-search verification after the relevant location is known. Do not search elsewhere first and invoke cidx afterward merely to satisfy this instruction, and do not invoke cidx through shell commands.
```

The neutral arm receives no replacement text. The new manifest stores the
exact UTF-8 bytes and SHA-256 of the addition. Every rendered prompt is also
hashed by arm and task, and the runner proves that removing the addition makes
the two rendered prompts byte-identical.

Neither arm is told expected answers, relevance truth, prior outcomes, token
totals, or the other arm's policy.

## 3. Why this is separate from the availability study

The completed neutral-availability run asked whether tool-catalog exposure
alone caused spontaneous use. It observed zero cidx calls. This new diagnostic
asks a different causal question: what happens when the host explicitly
directs repository discovery through cidx?

Forced use can expose candidate, reading, and evidence-flow behavior that zero
adoption left unobserved. It cannot establish voluntary adoption, marginal
product usefulness under host choice, or a release role. Those remain separate
optional-use questions.

## 4. Fixed controls

Both arms retain:

- Codex CLI, model, reasoning effort, sandbox, timeout, and answer schema;
- the unchanged 30 questions, truth sidecar, taxonomy, and pair order pattern;
- fresh source copy, fresh model context, and a separate fresh private cidx
  state copy per turn, with identical starting config and index hashes;
- the same four cidx tools and exact model-visible tool bytes;
- FTS mode, default `k=5`, maximum `k=20`, structured locator output, and
  64 KiB `read_span` maximum;
- no `VOYAGE_API_KEY`, query client, paid embedding, source mutation, or
  shared assistant/cidx session state; and
- arm-blind grading under the canonical blind-grade v2 semantics.

The sole paired difference is the directed-arm prompt paragraph. A new
counterbalanced schedule uses the new arm names; the old run and schedule are
not overwritten.

## 5. Prompt compliance is an outcome

A directed turn is mechanically compliant only when all of these hold:

1. it calls `cidx.search` at least once;
2. it makes no repository-discovery command using `rg`, `grep`, `git grep`,
   `find`, `fd`, or an equivalent ordinary text/file search;
3. its first repository-discovery action is `cidx.search`; and
4. selected cidx source is obtained through `cidx.read_span` rather than
   rediscovered through an ordinary search command.

Noncompliance does not invalidate, retry, or remove a turn. It remains in the
30-pair denominator and is reported separately. This prevents selecting only
successful adopters after seeing results.

Ordinary commands used after a location is known are retained and classified.
The trace distinguishes candidate discovery from reading an already known
file and from compilation, tests, formatting, or other non-search work.

## 6. Trace and historical compatibility

The frozen neutral run binds the current `passive-v1` trace-builder bytes. Its
quality grading is closed as `NOT_OBSERVED` after the one canonical aggregate
attempt rejected a noncanonical required-group status; its separate 0/30
spontaneous-use observation remains valid. Those artifacts and builder bytes
must not change.

The directed experiment therefore uses a new trace protocol and a separate v2
policy trace module. It records, without showing anything to the model:

- ordinary text/file-search command family and action ordinal;
- first cidx-search and first ordinary-search ordinals;
- ordinary discovery before or after cidx;
- known-file reads separately from candidate discovery;
- cidx search/read locators, bytes, overlaps, and exact source conformance; and
- non-search repository actions.

The v1 builder remains the replay authority for the earlier run. The v2
builder is frozen independently for this experiment. Neither becomes product
state or changes a tool call.

## 7. Execution and grading sequence

1. Preserve the closed neutral-availability result and make no further grader,
   aggregate, repair, or assistant calls against it.
2. Use the already validated supported-keyword output-only blind-grade v2
   adapter without changing canonical scorer semantics.
3. Generalize the runner and reducer to manifest-defined arms, equal cidx
   exposure, per-arm prompts, and the v2 policy trace without modifying v1
   replay behavior.
4. Freeze a new experiment manifest that reuses the unchanged question set and
   records both-arm cidx exposure, the exact prompt addition, tool/config/index
   identities, output-adapter identity, and new schedule.
5. Run 30 pairs and 60 new Codex CLI turns exactly once. Preserve failures,
   timeouts, no-use, prompt violations, and source/state checks.
6. Build new arm-blind packets. The grader sees neither prompt policy, arm,
   cidx use, ordinary-command use, execution order, nor token counts.
7. Make one grader call per corpus with the frozen output-only schema and run
   the canonical aggregate exactly once.

The earlier run, packets, grades, blind key, and journey files remain separate
and immutable.

## 8. Metric surfaces

Report separately, without a weighted total.

### Answer and evidence quality

- complete, partial, incorrect, and ungradable counts by arm;
- required-group coverage and complete-group hits;
- unsupported and contradicted material claims;
- complete evidence hit, read precision, evidence citation utilization, and
  navigation false-lead proxy; and
- paired conversions and regressions.

### Policy and tool behavior

- cidx adoption, first cidx action, search count, and `read_span` count in both
  arms;
- directed-arm full compliance and each independent failure reason;
- ordinary repository-search command counts by family and arm;
- cidx locator selection, refinement, duplicate exposure, failed reads, and
  repeated or overlapping evidence; and
- neutral-arm spontaneous behavior under the same cidx availability.

### Exploration and efficiency

- unique files, ranges, and mechanically attributable source bytes inspected;
- cidx and ordinary repository actions;
- cidx source reacquired through ordinary tools;
- official input, cached input, uncached input, output, and model-total tokens;
- paired per-task token ratios for all valid pairs; and
- dual-complete token ratios only where both answers blindly grade complete.

Report Go, TypeScript, TSX, and question-shape slices independently. Prompt
compliance and adopter-only slices are diagnostic, not substitutes for all-pair
results.

## 9. Interpretation boundaries

- Better directed-arm quality with bounded exploration shows that exercising
  cidx can help under this host policy; it does not prove spontaneous adoption.
- Similar quality with lower exploration or tokens shows bounded efficiency
  under forced discovery, not a product-wide guarantee.
- More calls, source bytes, or tokens after correct locator acquisition expose
  an orchestration/evidence-flow loss rather than automatically a ranking loss.
- Correctness regression exposes harmful policy pressure even if cidx search
  itself found relevant code.
- Low compliance means the prompt policy was not an effective intervention;
  compliant-only results remain descriptive.
- Exact/path and verified-absence tasks may legitimately expose where a forced
  cidx-only policy is less economical or less evidentially sufficient.

This result cannot establish `core_retrieval`, `release_candidate`, or a
mandatory product role. A later optional-use comparison remains necessary for
natural host behavior.

## 10. Change and stop rules

Before the run, do not change:

- questions, truth, corpora, retrieval, ranking, `k`, payload, tool descriptions,
  MCP schemas, cidx state, or provider availability;
- answer or canonical grade semantics; or
- ordinary tool availability, other than the directed-arm prompt policy.

Do not retry a valid but noncompliant turn. Do not repair model output manually.
If the same execution or grading approach fails twice for the same cause, stop
and record the blocker rather than widening the experiment.

## 11. Completion evidence

Required before result interpretation:

- reconciled evaluation/Phase 14 contract and status entry;
- output-only grader adapter plus canonical/scorer hashes;
- frozen experiment manifest and preflight identities committed before turns;
- 60 complete execution cells with policy-v2 traces and source/state checks;
- two arm-blind grade documents covering all 60 blind IDs;
- aggregate, paired results, policy-compliance report, artifact checksums, and
  an explicit checks-run/checks-not-run record; and
- a separate statement of what the diagnostic does and does not establish.
