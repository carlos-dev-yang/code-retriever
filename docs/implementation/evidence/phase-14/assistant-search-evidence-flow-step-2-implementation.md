# Assistant Search-to-Evidence Flow Step 2 Implementation

- Date: 2026-08-22
- Phase: 14, bounded Step 2 checkpoint
- State: `complete`; Phase 14 remains `in_progress`
- Owner: `/root`
- Implementation review: Terra high implementation agent, independent Sol high
  review agent, and direct main-agent source/structure review
- Product retrieval or ranking change: none
- Public MCP input-schema change: none
- New MCP tool, server session state, or SQLite authority: none
- Question-set, scored assistant turn, provider request, or paid operation: none
- Test code created: none

## Authorized boundary

This checkpoint implements only Section 14 Step 2 of the accepted
[search-to-evidence design](../../ASSISTANT-SEARCH-EVIDENCE-FLOW-DESIGN.md):

1. replace model-visible usage-policy descriptions with neutral capability
   descriptions while preserving exactly four public tools and their input
   schemas;
2. add a passive, non-model-facing, body-free session trace to the ignored
   assistant-evaluation state;
3. reduce navigation, evidence, exploration, voluntary adoption, and claim
   support as separate surfaces; and
4. version the blind-grade claim contract without changing historical V3-V6
   interpretation.

Question construction, a new assistant run, active orchestration, MCP response
metadata, provider traffic, and any production state remain outside this
checkpoint.

## Implementation

### Public MCP interface

`internal/mcp/schema.go` still registers only `status`, `search`, `read_span`,
and `reindex`. `search` describes locator-only results and the explicit paid
Voyage boundary for hybrid mode. `read_span` describes the existing exact,
hash-guarded, byte-bounded source read. Neither description tells the model
when or how often to use cidx.

The live MCP preflight observed:

```text
tools                       read_span, reindex, search, status
input-schema SHA-256        1a7edb66e784d531b401176507caef6867373dc9b39b2222ce83a41eda3d0819
description SHA-256         6c6a4a579fe065e35d5d8d925eb218b7d5b240dea4350110a02cd01621b5c96f
provider credential present false
```

The input-schema digest is byte-identical to the preserved V6 interface. Tool
descriptions and input schemas are now bound separately in every applicable
run manifest, so a description-only intervention is not mistaken for a schema
change.

### Passive trace

`scripts/assistant_session_trace.py` is the single mechanical trace builder.
It observes only:

- cidx `search` calls and returned locators;
- cidx `read_span` requests, delivered identity, result-envelope size, and
  source conformance; and
- ordinary repository-inspection events, output byte counts, verified source
  ranges, and source paths.

The retained trace contains no source body, raw shell command, frozen truth,
grade, or material-claim judgment. Raw commands are replaced by SHA-256. A
successful read requires exact requested/delivered path, line range, and
indexed SHA-256 identity plus an exact byte-for-byte match to the current
source span. The trace is an ignored evaluation artifact, is never sent to the
model, and creates no product-server or SQLite session state.

Ordinary-tool attribution is deliberately conservative. Each output parser is
gated by the command family that can produce that format. Mixed sequential
streams remain unattributed unless every stream has the same mechanically
verifiable producer. Every attributed line is checked against the bound source
checkout; ambiguous matches remain unattributed. This prevents source text
that merely resembles `path:line:content` from being counted as another file.

### Runner and reducer

`scripts/run-assistant-ab.py` resolves one explicit trace protocol, fails
closed on unknown protocols, writes the passive trace only for `passive-v1`,
and binds the trace protocol, schema version, and builder digest. Historical
protocols do not acquire a new trace artifact. The FTS preflight removes the
Voyage credential and records both requested and mechanically authoritative
effective mode.

`scripts/score-assistant-ab.py` keeps protocol-specific historical dispatch:

- V3: historical report without assistant stages;
- V4: historical locator and evidence stages;
- V5/V6: historical orchestration, locator, and evidence stages; and
- `passive-v1`: voluntary adoption plus separate navigation, evidence,
  exploration, and claim-support surfaces.

For a passive run, scoring requires the stored trace identity to match the run
manifest and a deterministic rebuild to be exactly equal before grading is
joined. Baseline and treatment exploration are reported separately. No-use
treatment tasks remain in the intent-to-treat denominator. The reducer does
not infer why a query was refined or whether unattributed ordinary output was
source.

### Blind grade v2

`schemas/evaluation/assistant-blind-grade.schema.json` retains readable v1 and
adds a separate v2 envelope. Passive runs require v2. Every material claim has
a unique ID, exact text, `observed | derived | unresolved` classification, and
evidence-index references. Observed or derived claims require at least one
reference. Runtime validation additionally requires every referenced evidence
item to resolve to a real in-root source file and valid line excerpt; duplicate
group IDs, evidence indices, support references, claim IDs, and finding IDs are
rejected. Unsupported and contradicted claim IDs are disjoint and remain
separate from required-group coverage.

## Main-agent structure review

The main agent did not rely on the implementation-agent summary. It searched
the complete Go registry, runner, trace builder, scorer, schemas, and Phase 14
contracts directly and followed these dependency edges:

```text
MCP registry -> runner preflight and manifest identities
raw Codex events -> mechanical session trace -> frozen trace identity
frozen source truth + blind grades -> protocol reducer -> aggregate/report
```

The trace builder owns truth-free event mechanics. The runner owns isolated
execution and artifact capture. The scorer owns the later truth/grade join and
keeps legacy protocol branches explicit. Product Go/SQLite code does not
import or depend on the evaluation trace. Keeping the passive reducer in the
existing scorer avoids introducing a second aggregation authority and
preserves historical report dispatch; the separate trace module contains the
new reusable mechanical parsing boundary.

The first independent review found legacy dispatch, attribution, read
identity, effective-mode, and blind-grade contract issues. The Terra agent and
main agent corrected them. The final review then found two narrower P2 issues:
command-agnostic ordinary-output parsing and support references to evidence
without a valid source excerpt. The main agent corrected both and added the
duplicate-ID/reference guards described above.

The final bounded Sol recheck exercised numbered, unnumbered, filename-bearing,
stdin, recursive, and mixed sequential search shapes plus exact, reordered,
missing, unknown, and duplicate required-group sets. It reported no remaining
actionable P1 or P2 finding.

## Validation evidence

Checks actually run:

- Python syntax compilation of the runner, trace builder, and scorer;
- JSON parsing of the blind-grade schema and repository whitespace validation;
- focused Go tests for `internal/mcp` and `internal/evalcontract`;
- the `internal/mcp` race check;
- focused static analysis for `internal/mcp` and the assistant MCP adapter;
- current-binary MCP initialize/list/status preflight with exactly four tools,
  a clean index, no Voyage credential, and separately computed tool,
  description, and input-schema digests;
- synthetic exact-source probes for `cat`, `head`, `tail`, single-file search,
  command-family isolation, ambiguous output, and the former false
  `cat a.ts -> target.ts` attribution case;
- blind-grade v2 positive and negative probes for missing source range,
  duplicate group ID, duplicate support reference, missing support, and an
  unknown claim finding;
- deterministic aggregate/report replay on disposable copies of V3, V4, V5,
  and V6; and
- a read-only structural trace probe over all 24 preserved V6 cells.

Replay results:

| Historical run | `aggregate.json` | `report.md` |
| --- | --- | --- |
| V3 | byte-identical | identical after excluding the two expected temporary absolute-path footer lines |
| V4 | byte-identical | identical after excluding the two expected temporary absolute-path footer lines |
| V5 | byte-identical | identical after excluding the two expected temporary absolute-path footer lines |
| V6 | byte-identical | identical after excluding the two expected temporary absolute-path footer lines |

The V6 structural probe built 24 traces from 67 ordinary inspections and 24
`read_span` results. All 24 reads passed exact identity and source-body
conformance. It recorded 625,038 ordinary output bytes, mechanically
attributed 493,329 of them, and mapped 289,267 source bytes. These values
validate instrumentation against known logs; they are not a new score, a new
assistant run, or a reinterpretation of V6.

Final focused command results:

```text
go test -count=1 ./internal/mcp ./internal/evalcontract   PASS
go test -count=1 -race ./internal/mcp                    PASS
go vet ./internal/mcp ./scripts/assistant-ab-mcp         PASS
Python syntax compilation                                PASS
blind-grade JSON parse                                   PASS
git diff --check                                         PASS
```

## Artifact identities

```text
internal/mcp/schema.go
  46a91ba8c207315680aae241b20061a2cf3fdfaf514f51a22fefc45cbb9dd551
scripts/assistant_session_trace.py
  12ca918216a7e6767de5d33b1c41d48f9b412745d19798aa485c72e78ab72410
scripts/run-assistant-ab.py
  a9a4a9a12992a83813a26bff9ca1e6b90903b3a2a7e769e34319aca537d0129a
scripts/score-assistant-ab.py
  7b393f4f52760d47eafa6ac2d55047dea6b53022a6758f8a85a7f988bcc9c538
schemas/evaluation/assistant-blind-grade.schema.json
  2d9729b3f38454ac863ec5898bb04148c97813deb17cd19ce8958fecceba0253
```

## Checks not run and remaining gate

- No broad full-project test suite or packaging/release matrix was run.
- No new test file or fixture was added.
- No new question, corpus, scored assistant turn, blind grade, provider call,
  embedding, production database write, or release/promotion decision was
  created.
- No active duplicate warning, stopping instruction, claim-state guidance,
  signature card, neighbor metadata, dependency hint, or public schema field
  was introduced.

Step 2 is complete but does not complete Phase 14. The exact next action is an
owner decision on Step 3. If separately authorized, Step 3 prepares and freezes
the source-first 30-question version without running scored turns.
