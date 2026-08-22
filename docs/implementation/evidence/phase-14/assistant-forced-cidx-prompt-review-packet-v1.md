# External Review Packet — Forced cidx Prompt Diagnostic V1

Please review the following completed, immutable experiment as an independent
technical critic. Separate measured facts from inference. Do not propose
repairing, regrading, or rerunning this frozen result.

## Owner question

The owner asked for a paired Codex CLI test of two prompt conditions:

- neutral: cidx is available but the prompt gives no cidx instruction;
- directed: the same prompt additionally says to use `cidx.search` instead of
  `rg`, `grep`, `git grep`, `find`, `fd`, or ordinary repository search, then
  use `cidx.read_span` for selected locators.

Does cidx improve the assistant's journey when it is actually exercised, and
what should be improved next?

## Controls

- 30 unchanged paired questions: 10 Go, 10 TypeScript, 10 TSX; six question
  shapes; 60 isolated Codex CLI turns.
- `gpt-5.6-sol`, high reasoning, read-only sandbox, same answer schema.
- Both arms expose byte-identical cidx `status`, `search`, `read_span`, and
  `reindex` tools and the same ordinary tools.
- Local FTS only; no Voyage credential, embedding, dense/hybrid query, source
  mutation, or cidx database mutation.
- The exact prompt paragraph is the sole paired intervention.
- 60/60 turns valid, zero timeout/control/state failure.
- Arm-blind grading: one no-tool grader call per corpus, all 60 IDs covered,
  then one canonical aggregate.
- This is a non-promotion diagnostic. It cannot establish a mandatory product
  role, optional adoption, or release readiness.

## Results

### Quality

- neutral: 29 complete, 1 partial, 0 incorrect; 39/39 required groups;
  4 unsupported material claims out of 115.
- directed: 29 complete, 1 partial, 0 incorrect; 39/39 required groups;
  1 unsupported material claim out of 104.
- One neutral partial converted to directed complete; one neutral complete
  regressed to directed partial because a cited range omitted lines needed for
  one internal-mechanism claim.

### Adoption and policy

- neutral used cidx in 0/30 tasks and made 62 ordinary `rg` discovery actions.
- directed used cidx search and read in 30/30 tasks, made zero ordinary
  discovery actions, and made 110 searches plus 119 reads.
- 26/30 were mechanically compliant. The other four all used `read_span`, but
  not with an exact previously returned locator range. There was no shell cidx,
  discovery-before-cidx, failed read, or omitted hash.

### Retrieval and evidence

- Of 27 positive/answerable tasks, 23 had complete required-locator coverage
  in the first search results and all 27 eventually did.
- First useful locator rank median was 1 among 26 tasks with a useful first-
  search locator.
- All 27 positive tasks acquired complete required cidx-read evidence.
- The three without positive evidence were verified-absence questions and
  graded complete in both arms.
- Directed combined unique source: 196,476 bytes versus neutral 532,030
  (-63.1%). Repository-output proxy: 641,426 versus 1,251,377 (-48.7%).
- Directed known-file reads reacquired 39,416 bytes already supplied by cidx,
  94.1% of its ordinary-read source bytes.

### Orchestration and tokens

- Repository actions: 104 neutral versus 254 directed (+144.2%).
- 20/30 directed tasks refined after the first search; 80 refinements; 1,181
  locator occurrences and 770 unique locators.
- Most searches explicitly requested `k=10` despite a server default of 5.
- Directed MCP result bytes: 566,458, including 558,250 structured bytes.
- Official model-total sum: 2,736,827 neutral versus 4,178,972 directed
  (+52.7%). Paired ratio median 1.404; only 6/30 non-increasing.
- Cached input rose 71.2%, uncached input rose only 1.8%, and output rose 19.7%.
  This is interpreted as evidence of extra inference/tool turns revisiting
  context rather than raw source dumping alone.

## Proposed interpretation for critique

1. The exact directed prompt is a successful tool-routing mechanism but a
   failed blanket token-efficiency policy.
2. On this panel, token growth is not principally caused by missing target
   locators. It is caused by iterative search/read decisions and partial source
   reacquisition after useful candidates appear.
3. cidx shows bounded value: it narrows code scope substantially, preserves
   aggregate outcomes, and reduces unsupported claims, but the current two-call
   search/read interaction is not economical when repeated many times.
4. Ranking should not be tuned first from this result. The next design should
   focus on one compact candidate pass, explicit refinement/stop semantics, an
   easier selected-locator handoff, complete line-addressable bounded evidence,
   session-only range deduplication, and possibly bounded multi-span reads.
5. Ranked top-k FTS should not be treated as proof of repository-wide absence;
   negative/exhaustive questions need ordinary literal search or an explicit
   exhaustive cidx contract.
6. After a concrete interface decision, the next assistant experiment should
   make cidx known but freely selectable. Another cosmetic forced-prompt rerun
   is not justified.

## Requested review

Return:

1. `ACCEPT`, `ACCEPT_WITH_CORRECTIONS`, or `REJECT` for the proposed
   interpretation;
2. any factual or causal overclaim, citing the metric that contradicts it;
3. the most likely reason model-total tokens rose despite lower source/output
   bytes;
4. the smallest product/interface change that directly targets the measured
   loss without combining unrelated ranking, dense, corpus, or provider
   changes; and
5. what the next optional-use experiment must measure and what it must not
   claim.
