# Codex MCP Result Representation Probe

- Date: 2026-08-22
- State: accepted host-conformance checkpoint; product default not changed by
  this checkpoint alone
- Host: Codex CLI `0.149.0-alpha.4`
- Model: `gpt-5.6-sol`, high reasoning
- Corpus/task: existing approved chi snapshot, `chi-new-router`
- Provider, network tool action, source mutation, and paid embedding: none
- Promotion authority: none

## Question

The V3 event stream contained equivalent JSON in both MCP `content[].text` and
`structuredContent`. Event presence did not prove which representation Codex
could consume. Before removing the duplicate, the host had to demonstrate a
complete search-to-read journey with each representation independently.

## Probe contract

The development-only assistant MCP launcher accepted one of:

- `dual`: text plus structured payload;
- `text`: one JSON text content block and no structured payload; or
- `structured`: an empty required `content` array plus one structured payload.

The existing isolated assistant runner propagated this selection explicitly
and recorded it in the run command. Product CLI/MCP configuration did not gain
a public option.

Both probes used the same frozen V3 task prompt and controls, an isolated clean
source copy, an isolated copied index, FTS-only search, and the normal
hash-guarded `read_span`. Each probe contained one treatment turn only. The
purpose was compatibility, not an efficiency comparison.

## Results

| Representation | Valid execution | First action | Search calls | `read_span` calls | Final schema |
| --- | --- | --- | ---: | ---: | --- |
| structured-only | yes | cidx search | 2 | 2 | valid |
| text-only | yes | cidx search | 1 | 2 | valid |

The structured-only event records had `content=[]` and a structured object for
every cidx result. The text-only records had one text block and no structured
object. In both cases the assistant extracted paths, line ranges, and indexed
hashes from search and used those exact values in successful `read_span`
calls. There were no control violations, timeouts, final-output errors, or
source/index mutations.

The different call counts and token totals are not compared. They are two
stochastic turns rather than a paired efficiency experiment.

Retained event SHA-256 values:

- structured-only:
  `dd02d4b05aa1751dcf3fb5c46225c6a6a73da4d11e971431f71f2ba76f5f3524`;
- text-only:
  `431ebacb9062ab5555b9b4842142a4cc675cacd03637dd9839d4beab868bf33e`.

Probe binary SHA-256 values:

- cidx: `fe8db0a04b0dccc6865ee5a1258c8f4f5b2b8773252d924651bd6a00d413ca8c`;
- assistant MCP launcher:
  `084f44a32dc818c95bcbe354b5bf7088728e225f5f75aadd434298a6387352f1`.

The raw local probe artifacts remain ignored because they contain model and
source output. Their hashes and reproducible controls are recorded here.

## Decision

Use structured-only results for the verified Codex diagnostic and product
default:

- it removes the equivalent JSON text copy;
- it retains the typed result object;
- `content=[]` preserves the required MCP result-content member; and
- the verified Codex host consumed it through a complete search-to-read
  journey.

This decision verifies Codex only. Another host is not called supported until
the same conformance behavior is checked there. A future host that cannot
consume structured results requires an explicit compatibility design; it does
not silently restore dual payloads for every host.

## Validation performed

- focused `internal/mcp` tests;
- cidx and assistant MCP launcher builds;
- direct MCP preflight support for all three development representations;
- one isolated structured-only Codex task turn;
- one isolated text-only Codex task turn;
- event-shape inspection for content count and structured type;
- successful downstream hash-guarded reads; and
- artifact SHA-256 capture.

Not performed:

- no scored A/B batch;
- no paired token claim;
- no locator-only projection yet;
- no non-Codex host validation;
- no provider call; and
- no promotion or release claim.

## Handoff

Make structured-only the production server default, project shared ranked hits
to the frozen locator fields, request zero search body bytes, and validate
rank/identity invariance across retained `max_inline_bytes` values. The
development representation flag remains confined to the assistant experiment
launcher.
