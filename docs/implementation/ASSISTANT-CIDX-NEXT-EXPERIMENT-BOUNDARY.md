# Assistant cidx Next-Experiment Boundary

- Status: owner-retained design note; not execution authorization
- Date: 2026-08-23
- Phase: 14
- Baseline: [Forced cidx Prompt Diagnostic V1 result](evidence/phase-14/assistant-forced-cidx-prompt-result-v1.md)
- Supersedes: the removed awareness/trust audit, semantic-sidecar, and dedicated-harness work

## Purpose

The completed forced-cidx diagnostic already establishes the only baseline
needed for a successor experiment: directed cidx use narrowed delivered source
but increased repository actions and model tokens. Do not re-audit that closed
run before testing a smaller host-instruction change.

This document preserves the only three ideas retained from the rolled-back
follow-up work. It does not preserve or authorize its audit scripts, semantic
review pipeline, dedicated runner, scorer expansion, schemas, manifests, or
generated artifacts.

## 1. Concise MCP descriptions

The following descriptions may be evaluated in a later, separately authorized
experiment. They are not applied to the product by this document.

### `search`

> Locate relevant functions, methods, and types from a natural-language,
> identifier, or path query. Returns ranked locators without source text;
> inspect selected locators with `read_span`. Omit `k` to use the configured
> default. Ranked top-k results do not establish repository-wide absence.

### `read_span`

> Return the complete current source for one selected locator's exact path,
> range, and indexed hash. This source is evidence for that exact range and
> does not establish behavior in an unread dependency.

## 2. Two prompt arms

Both arms must expose the same ordinary repository tools, the same four cidx
tools, the same FTS state, and the same concise tool descriptions.

### Shared awareness paragraph

> `cidx.search` and `cidx.read_span` are available for repository code
> discovery. `cidx.search` returns ranked function, method, and type locators
> for natural-language, identifier, or path queries; `cidx.read_span` returns
> the exact indexed source for a selected locator. Use cidx or ordinary
> repository tools according to the task.

### Trust-priority addition

Only the treatment arm adds:

> When locating repository code, prefer `cidx.search` over `rg` or equivalent
> ordinary text search. Treat a successful `cidx.read_span` result as
> authoritative for its exact path, range, and indexed hash, and do not
> reacquire the same or overlapping source merely for reassurance. Start with
> the configured default candidate count. Refine only for a missing answer
> requirement or a concrete ambiguity. Use ordinary search when the path is
> already known, an exhaustive literal or absence check is required, or cidx
> evidence leaves a specific unresolved context or dependency gap.

This contrast measures the complete treatment paragraph only. It does not
separately attribute an effect to priority, trust, stopping, or exception
wording.

## 3. Optional post-run diagnostic

Post-run questions are diagnostics, not primary metrics and not a mandatory
follow-up for every session. Select traces only after the primary run because
they contain a concrete investigation target such as repeated search,
overlapping source reacquisition, unusually high repository-action count, or
an unresolved evidence gap.

Ask only what is relevant to that trace:

- Why was cidx selected or not selected?
- Why was each search after the first useful locator necessary?
- Did `read_span` omit adjacent or dependency context needed for the answer?
- Why was already delivered or overlapping source inspected again?
- Did an ordinary repository tool appear more trustworthy or convenient?
- What smallest response or interface change would have ended exploration
  sooner?
- Which statements are directly observed facts and which are inferences?

Self-report never changes the primary answer, grade, action count, source-byte
measurement, or token measurement.

## Minimal future execution boundary

A future owner-authorized experiment must stay within this boundary:

1. reuse the existing approved 30-question set and frozen truth;
2. reuse and minimally generalize the existing `run-assistant-ab.py`, passive
   trace, policy classifier, and blind-grade path instead of creating a
   dedicated runner or scorer;
3. execute 30 paired questions and 60 primary turns with a balanced first-arm
   order;
4. report answer grades, cidx and ordinary search/read actions, unique and
   overlapping source, and official token usage as separate measures;
5. use deterministic trace analysis only for primary results;
6. perform optional post-run questioning only on selected traces; and
7. obtain one bounded pre-run plan review and one post-result interpretation
   review. Advisory review must not recursively expand the experiment.

Do not add a predecessor re-audit, per-action semantic review, mandatory
60-session self-report, new repository, dense/HNSW work, new MCP tool, ranking
change, or paid provider operation. Any such expansion requires a new explicit
owner decision before implementation.
