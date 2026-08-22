# Forced cidx Prompt Diagnostic V1 Result

- Date: 2026-08-23
- Run: `assistant-forced-cidx-prompt-chi-rhf-v1-run-001`
- Manifest: `assistant-forced-cidx-prompt-chi-rhf-v1`
- Scope: 30 paired tasks, 60 Codex CLI turns, non-promotion prompt-policy diagnostic
- Arms: `neutral_cidx` and `directed_cidx`
- Retrieval/provider boundary: local FTS only; no Voyage credential, dense query, hybrid search, or provider call

## Decision

The directed paragraph successfully changed repository discovery from `rg` to
cidx, and cidx reduced the unique code surface while preserving aggregate
blind answer outcomes. It did **not** reduce official model-token use. The
dominant measured loss is turn and evidence-flow multiplication after useful
locators are available, not a failure to retrieve the required locations.

| Surface | Neutral | Directed cidx | Result |
| --- | ---: | ---: | --- |
| Complete / partial | 29 / 1 | 29 / 1 | aggregate correctness preserved |
| Required groups covered | 39/39 | 39/39 | equal |
| Unsupported material claims | 4/115 | 1/104 | directed arm lower |
| Combined unique source bytes | 532,030 | 196,476 | -63.1% |
| Repository-output proxy bytes | 1,251,377 | 641,426 | -48.7% |
| Repository tool actions | 104 | 254 | +144.2% |
| Model-total token sum | 2,736,827 | 4,178,972 | +52.7% |
| Paired model-total ratio median | — | 1.404 | 6/30 non-increasing |

The prompt is therefore a successful routing intervention and a failed token-
efficiency policy in its present form. This result does not demote cidx to a
secondary product role and does not authorize a mandatory cidx-only host
policy. It identifies the next interface problem to solve.

## Frozen execution and grading integrity

The only model-visible paired difference was the directed arm's exact prompt
suffix. Both arms exposed byte-identical `status`, `search`, `read_span`, and
`reindex` definitions, the same source snapshot and FTS index, and the same
model, effort, sandbox, answer schema, and ordinary tools.

- the frozen preflight and both arm schema probes passed before scored turns;
- all 60 scored turns completed exactly once and were valid;
- zero turn timed out or returned a nonzero process exit;
- zero control violation, source mutation, or cidx database mutation occurred;
- the isolated source and state copies were removed after every capture;
- the 60 blind IDs were split into 20 Go and 40 React Hook Form entries;
- exactly one tool-free grader call per corpus produced 20/20 and 40/40
  matching grade IDs;
- grader event logs contain no tool event and no error line; and
- the canonical aggregate was invoked exactly once after both grade documents
  passed identity checks.

The Go grader used 61,086 input and 11,569 output tokens. The React Hook Form
grader used 143,638 input and 17,781 output tokens. These grading costs are not
attributed to either assistant arm.

## Blind answer and evidence quality

| Arm | Complete | Partial | Incorrect | Ungradable | Required groups | Unsupported | Contradicted |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `neutral_cidx` | 29 | 1 | 0 | 0 | 39/39 | 4 | 0 |
| `directed_cidx` | 29 | 1 | 0 | 0 | 39/39 | 1 | 0 |

There was one conversion and one regression:

- `avail-go-contract-throttle`: neutral `partial` to directed `complete`. The
  directed answer acquired support for the defaults and literal error cases
  that were unsupported in the neutral answer.
- `avail-go-contract-route-outcome`: neutral `complete` to directed `partial`.
  The directed answer covered the routing contract but asserted an internal
  leaf-detection/allowed-method mechanism whose cited tree excerpts omitted
  the relevant lines.

The unsupported-claim rate fell from `3.48%` to `0.96%`. This is encouraging
evidence that bounded selected code can improve claim discipline, but one
paired conversion and one paired regression do not establish a general
correctness gain.

Go was 9 complete and 1 partial in each arm. TypeScript and TSX were each
10/10 complete in both arms. Every language slice retained all frozen required
groups.

## Prompt and tool behavior

The intervention strongly controlled tool selection:

- neutral spontaneous cidx use: `0/30` tasks;
- directed cidx search and read use: `30/30` tasks;
- neutral ordinary discovery: 62 actions, all `rg`;
- directed ordinary discovery: zero actions;
- directed cidx calls: 229 total, comprising 110 searches and 119 successful
  `read_span` calls; and
- shell-invoked cidx attempts: zero.

Mechanical directed-policy compliance was `26/30`. The four failures were all
`no_selected_locator_read_span`: the assistant did use `read_span`, but the
range it supplied was not an exact previously returned locator range. No
failure involved ordinary discovery before cidx, shell cidx, a failed read,
or a missing required hash. The affected tasks were:

- `avail-go-multihop-real-ip`;
- `avail-go-exact-router-contract`;
- `avail-ts-multihop-subject-contract`; and
- `avail-ts-contract-dotted-paths`.

This is primarily an input-contract/usability observation. It should not be
reported as four search failures.

## Locator quality versus orchestration loss

The frozen panel separates locator acquisition from what the assistant did
after acquisition.

### Locator acquisition

- 23 of 27 answerable tasks had every required locator represented in the
  first search result set;
- all 27 answerable tasks had complete required-locator coverage after any
  refinements;
- among the 26 tasks with a first-search useful locator, the median first
  useful rank was 1: rank 1 in 20 tasks, rank 2 in 3, rank 3 in 2, and rank 8
  in 1;
- 27 answerable tasks reached complete cidx-read evidence; and
- the three tasks without a complete-evidence hit were the three frozen
  verified-absence/hard-negative questions, which have no positive target
  group and still graded complete in both arms.

This bounded evidence does not prove universal FTS quality, but it rules out a
missing-target explanation for the aggregate token increase on this panel.

### Search and read multiplication

The directed arm made 254 repository actions versus 104 in the neutral arm.
Its per-task median was 8.5 actions versus 3.0. The cidx path separated
candidate lookup from source acquisition, but the assistant frequently turned
that useful separation into many model/tool round trips:

- 20/30 tasks refined after their first search;
- those tasks made 80 refined searches, for 110 searches overall;
- search responses exposed 1,181 locator occurrences and 770 distinct
  locators;
- 119 reads followed, but only 86 were exact reads of a selected locator;
- read precision was `0.437` macro; and
- five successful read ranges overlapped, with one exact duplicate range.

Most searches explicitly requested `k=10` even though the frozen server
default was 5. One attempt requested an invalid `k=50`, then retried at 20.
The prompt required cidx but did not give a stopping condition, initial depth,
or refinement rule.

### Reacquisition after cidx

Fifteen directed tasks also used 25 ordinary known-file reads after locations
were known. These were allowed verification actions rather than prohibited
discovery. They exposed 41,875 unique source bytes, of which 39,416 bytes
(`94.1%`) overlapped source already returned by cidx. That overlap equals
`20.3%` of all unique cidx source bytes in the directed arm.

The recurrence supports the owner's earlier distinction: a tiny locator-only
answer is insufficient, while indiscriminate source delivery is wasteful. The
selected evidence response must be complete and convenient enough that the
assistant can base final claims on it without reacquiring the same range.

## Why tokens rose while code bytes fell

The directed arm reduced both unique source and repository-output proxy bytes,
so the token regression is not explained by dumping more raw code.

| Official usage | Neutral | Directed | Change |
| --- | ---: | ---: | ---: |
| Input sum | 2,687,335 | 4,119,726 | +53.3% |
| Cached input sum | 1,993,216 | 3,412,992 | +71.2% |
| Uncached input sum | 694,119 | 706,734 | +1.8% |
| Output sum | 49,492 | 59,246 | +19.7% |
| Model-total sum | 2,736,827 | 4,178,972 | +52.7% |

The small uncached-input increase beside the large cached-input increase is
consistent with turn fragmentation: every extra search/read decision causes
the model to revisit the accumulated conversation, and official model-total
usage counts that cached context. The cidx arm returned 566,458 bytes of MCP
event results, including 558,250 structured bytes. Total repository-output
proxy still fell by 48.7%, but it arrived across far more inference turns.

Across all 30 pairs, the directed/neutral model-total ratio median was `1.404`
and only `6/30` pairs were non-increasing. Across the 28 dual-complete pairs,
the median was `1.390`. The sum ratio was `1.527`.

Question-shape medians reinforce the same boundary:

| Shape | Tasks | All-pair model-total ratio median |
| --- | ---: | ---: |
| Exact identifier or path | 6 | 1.282 |
| Semantic single hop | 6 | 1.392 |
| Dependency/type multi-hop | 6 | 1.746 |
| Ambiguous disambiguation | 3 | 1.839 |
| Verified absence/hard negative | 3 | 1.704 |
| Contract/lifecycle/state/error | 6 | 1.208 |

The largest ratios occurred for ambiguous `FormState` (`4.332`), Go
compression multi-hop (`4.282`), semantic `watch` (`3.069`), and Go throttle
contract (`2.955`). These sessions repeatedly reformulated queries or split
evidence into many reads. The lowest ratios were the subject contract
(`0.645`), use-form lifecycle (`0.695`), exact use-form return (`0.710`), and
dotted-path contract (`0.727`). Task-specific value exists, but the blanket
policy is not consistently efficient.

## Product and experiment interpretation

This diagnostic establishes only the following bounded facts:

1. the exact host instruction is sufficient to move Codex from 0/30 cidx use
   to 30/30 cidx use without hiding ordinary tools;
2. current FTS plus `read_span` can find and supply all frozen positive target
   groups in this panel;
3. selected cidx evidence materially narrows inspected source and is associated
   with fewer unsupported claims in this run; and
4. the current forced workflow multiplies model/tool turns enough to erase the
   source-volume savings in official model-token usage.

It does not establish spontaneous or informed optional adoption, a mandatory
product role, a general correctness improvement, dense/hybrid value, or
release/promotion readiness.

## Next design boundary

Do not tune FTS ranking first from this result. The answerable panel eventually
found 27/27 required targets, while the measured first loss is the search-to-
evidence interaction contract.

The next design discussion should preserve the two-stage goal while reducing
round trips:

1. **Candidate stage:** make the cheapest normal path one compact search with a
   small default candidate set. State that refinement is for a missing answer
   requirement, not general reassurance. Do not force `k=10` or repeated
   paraphrases when rank 1 already supplies a useful locator.
2. **Selection handoff:** make a selected locator directly consumable by
   `read_span` without manually reconstructing an exact path/range/hash tuple.
   A `chunk_id`/opaque selection handle or a ready-to-copy read request is a
   candidate, subject to an explicit public-contract decision.
3. **Evidence stage:** return enough exact, line-addressable source and bounded
   adjacent/dependency context for the final claim. Avoid both locator-only
   starvation and automatic whole-file expansion.
4. **Session-only deduplication:** let the host remember ranges already read in
   the current task and discourage exact/overlapping reacquisition. Do not add
   a persistent second authority or a complex product-wide evidence graph.
5. **Multi-span economics:** evaluate whether one bounded `read_span` request
   can carry several selected locators. This is read batching, not embedding
   batching, and directly targets the measured inference-turn multiplier.
6. **Negative/exhaustive work:** do not claim that a ranked top-k FTS result can
   prove repository-wide absence. Either retain ordinary exact search for this
   shape or design an explicit exhaustive/literal cidx contract before a
   cidx-only policy is considered.

Any schema or tool-description change is a new external-contract decision and
must not be silently implemented from this diagnostic. After that decision,
the next assistant study should expose cidx as an informed, freely selectable
tool and measure whether the model chooses it where it reduces the whole
journey. Repeating only the blanket prompt with a cosmetic wording change is
not justified.

## Artifact identities

| Artifact | SHA-256 |
| --- | --- |
| run manifest | `ea952a4617cfa3b2d697c1f7bcd6989ba0f1016ab74e262436267f5380ea7086` |
| frozen journey JSONL | `b98cafeef4a027f0b5b8f9ddb467ceded218a1e492bb96a89bc1510f9ee3cbd7` |
| journey freeze envelope | `b9e7cde13f1f9a3ccd29b2a8f3e89fb8e5682cb20afded69a77b1b1e60e539ba` |
| Go grading packet | `c0c97f3cc47e5a81360fb148ebe90a472cf2c32e9979e1a26d181688a071ee41` |
| React Hook Form grading packet | `aa6c854d7f8c571fb68990116330093e0641f76f7526a5cd3c4efba8104b6488` |
| Go blind grades | `a0c0006df0720853a54188a6a184bb192113291e0be613a29edfbc7b9f7e9fea` |
| React Hook Form blind grades | `9a6518ed1815e66e6d0c2783c73ee6e2d046463f9bba0692bb2b26321a4913ee` |
| Go grader events | `3f21509c10813467ded961906ff039a2c0c77f93e036e131769d33a895368e8d` |
| React Hook Form grader events | `7d4c2f5645c1b55a6ee32952c383915a18b3a39b451f761fcb9fe058cd6f75ad` |
| paired results | `4462f3532e79e2a3bc140cb0c9f70932af8a8a9b221cd706586515593681d05b` |
| aggregate | `7d0156a0c8a73d1be01476dd1b69f8bcf3ab498fd3743786a038f02615458920` |
| generated report | `b5e2247a1209e53e3ce92aeea8406c41590e1cf382b6e3d0343f2de4701daa21` |

## Checks run and not run

Checks run:

- frozen preflight and both unscored arm schema probes;
- 60 scored observations and before/after source/database identity audit;
- frozen trace, manifest, runner, scorer, and output-adapter identity checks;
- blind packet leakage audit and exact grade-ID coverage;
- tool-free grader event and stderr audit;
- one canonical aggregate; and
- read-only per-task review of outcome reversals, prompt failures, largest token
  ratios, search refinements, reads, candidate counts, and source reacquisition.

Checks not run:

- no broad product test suite, because no product implementation changed after
  the frozen execution commit;
- no paid embedding, dense, hybrid, ANN, or provider evaluation;
- no new repository or question version; and
- no promotion or release-candidate gate.
