# Final Implementation-to-Design Review: Int8 Source Profile

- Status: accepted for the corpus-independent cidx v1 implementation scope;
  no implementation finding remains in the reviewed change set
- Date: 2026-08-17
- Reviewed head: `ec9c733`
- Current product authority: default 1024/int8, explicit compact 512/int8,
  durable document source-1024 f32, and no Binary/256 product path
- Promotion status: not promotion-ready; Phase 07 owner-adopted dual-AI label
  freeze, official Phase 12, paired assistant evidence, and release-candidate
  scope remain gated

## Authority and scope

The original Revision 4 design remains the base architecture. Its earlier
Binary/256 capability statements are superseded for the current product by the
owner's 2026-08-17 decision recorded in the implementation index,
[retired-profile contract](../../RETIRED-VECTOR-PROFILES.md), and
[source-bank decision](../../SOURCE-VECTOR-BANK-DECISION.md). Historical
five-profile measurements remain evidence; they are not executable options.

This review covers the committed reconciliation from config through package:

| Boundary | Commit |
| --- | --- |
| config/profile authority | `969bd89` |
| index reproof | `bbf3bb6` |
| product source-bank split | `15863af` |
| int8 materialization | `7db6f35` |
| document orchestration | `a677449` |
| query/search cleanup | `95e1c00` |
| evaluation adapter | `cc6e098` |
| CLI/MCP surface | `97959cf` |
| package verifier | `5f4955e` |
| package evidence/status | `ec9c733` |

It does not re-score chi/RHF, alter labels, call Voyage, run an assistant, or
claim official promotion.

## Contract-to-implementation matrix

| Current contract | Implementation observation | Evidence |
| --- | --- | --- |
| 1024/int8 is the default | `DefaultServingDimensions=1024`; default codec is the sole `int8` codec | Phase 02 and Phase 13 current evidence |
| Only 1024 and 512 are selectable | the model registry exposes exactly `[1024, 512]`; config, CLI, and evaluation schemas validate the same set | Phase 02, Phase 12, and installed-package config captures |
| Provider output remains source-1024 f32 | the product source-bank schema requires 1024 dimensions, 4096-byte little-endian f32, checksum, SHA-256, source profile, and canonical input hash | Phase 08 source-bank evidence |
| 512 needs no new document embedding when compatible source rows exist | the shared transform performs prefix selection followed by L2 normalization; document apply reads validated source rows, int8-encodes, and publishes without entering the provider plan | Phase 09/10 evidence and Phase 14 provider-free 512 run |
| One active serving profile | production rows and snapshots are keyed/reproved against current source, vector-space, storage, and active-profile fingerprints | Phase 05/09/10 evidence |
| Search is int8-only | vector scan prepares one int8 query, rejects a non-int8 active row, and uses the dedicated reconstructed-int8 cosine path | Phase 11 current evidence |
| Binary/256 are not product capabilities | no Binary codec/score/materializer/config enum or 256 serving value exists in non-test product code; CLI/package checks are rejection-only | Phase 00 retired-profile evidence and Phase 14 current evidence |
| Source storage is not serving authority | ordinary open/serve builds search from production `index.db`; source-bank opening is confined to explicit document embedding orchestration | Phase 08/13 evidence and relocated package smoke |
| Project state is local and portable | ordinary state resolves below `<source>/.cidx`, production is `.cidx/db/index.db`, and source rows are `.cidx/db/embeddings.db`; source identity is not an absolute persisted path | project-local layout and source-bank evidence |
| Stable MCP remains four tools | installed binary and schema expose exactly `status`, `search`, `read_span`, and `reindex` | Phase 13 and Phase 14 current evidence |
| Package reproduces both current profiles | clean darwin/arm64 archive materialized 3/3 at 1024/int8 and then 3/3 at 512/int8 from the same source rows with zero Voyage requests/tokens | Phase 14 current evidence |

## Static and artifact review performed

- Scanned `cmd`, `internal`, evaluation schemas, and package scripts for retired
  codec IDs, Binary transform/score APIs, candidate aliases, legacy flags, and
  256 serving values. Non-product matches were limited to rejection fixtures,
  SHA-256, 256 KiB request limits, standard-library binary encodings, and
  historical/evidence text.
- Inspected current config/model registry, shared prefix/L2 transform, int8
  codec and prepared-query scorer, source-bank schema, document source reuse,
  search scan, CLI help, and four-tool MCP schema.
- Parsed every evaluation JSON schema. Current paired-control dimensions are
  exactly 1024 or 512 with source dimension fixed to 1024.
- Rechecked the ignored clean-provenance archive and verifier evidence recorded
  in [Phase 14 current evidence](../phase-14/int8-profile-package-reconciliation.md).
- Confirmed the committed workspace is clean except for pre-existing user-owned
  `.gitignore` and `.omo/` changes, which were not staged or modified by this
  reconciliation.

No broad test suite was repeated for this read-only final review. Each owning
phase already performed its one focused commit-boundary validation, and the
final installed-package boundary exercised the assembled behavior.

## Linkage versus runtime access

The single `cidx` executable includes public embedding and development command
code, so Go dependency inspection naturally shows source-bank and lab packages
linked into that executable. This is not a second serving authority. The
ordinary `serve/search/MCP` assembly does not call the source-bank opener; the
installed-package relocation fixture removed `embeddings.db` and all
development state and still completed status plus the four-tool MCP smoke
without creating either. The accepted invariant is runtime access isolation,
not whole-binary symbol exclusion.

## Result and remaining work

The current corpus-independent implementation matches the superseded int8-only
product contract. Phase 07 now resumes under `owner-adopted-dual-ai-v1`; it
must propagate `NO_INDEPENDENT_HUMAN_REVIEW` and must not claim human review.

The remaining gates are evidence work, not hidden implementation completion:

1. independent ChatGPT/Grok chi/RHF source passes, reconciliation, owner adoption, and digest freeze;
2. immutable rank replay and official Phase 12 `core_retrieval` result;
3. frozen paired assistant/host evidence;
4. Phase 14 `scope=release_candidate` decision.

Until those inputs exist, Phase 07, official Phase 12, and Phase 14 promotion
scope correctly remain `blocked`.
