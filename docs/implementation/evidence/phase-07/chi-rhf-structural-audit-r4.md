# Phase 07 chi/RHF Structural Audit — Revision 4

- Status: `blocked_at_document_provider_decision`
- Date: 2026-08-16
- Scope: provider-free corpus, inclusion, parser/chunker, semantic-parent, segment, and document-plan inspection
- Working profile: `target_segment_bytes=1024`, source dimensions `1024`, serving dimensions `1024`, storage codec `binary`
- Paid provider calls: `0`

## Authority and claim

This is a real-data structural checkpoint for the two user-approved pinned
checkouts. It is not frozen-label lexical evidence, dense-quality evidence, or
promotion evidence. No API key was read and no network or provider operation
ran.

The tracked corpus manifests still select all 78 Go files in chi and all 237
TypeScript/TSX files below `src` in react-hook-form. Tests and examples remain
in the approved universes; this audit did not silently narrow either manifest.

## Reverified corpus and profile

| Corpus | Commit | Tree | Files | Clean | Manifest |
| --- | --- | --- | ---: | --- | --- |
| go-chi/chi v5.3.1 | `8b258c7bb28f97a5f2a856ff7ef962578fec9215` | `7ccb2269b57183ac3a741f269c0da31fd03ad035` | 78 Go | yes | `6bd4db89ee1a9cba70f69e125a803d147dbc0d92c95ef59b44be2dcb54302a29` |
| react-hook-form v7.85.0 | `371432c39271aab739358d19c406793771565ab3` | `688906c5842a0d71051154343e993adb525e688f` | 183 TypeScript + 54 TSX | yes | `54f6b1387ae989b1e49bdf21d3ed96189e76fb5b61b74ca282a2617c57f88b8a` |

This section records the pre-correction generation-2 baseline. The earlier
smoke checkouts used 256 serving dimensions. Their ignored local
configs were changed to the user-confirmed 1,024/binary working profile. A
provider-free index apply activated generation 2 for both projects, reused
every file, and preserved both source manifests. The applied fingerprints are:

```text
index            8bf19c762c3d23c42caba28aab16bc6ff61a7706ff93114c7170bb7095ee70aa
canonical text   eabf6198a0be430d4d5c15f1f036d99e8fbde12381a060be0debb34c211c7016
source 1024-f32  923a0b84bf40d3880b5a081861ed6e17208380a5fa152847d66d2b98f222c0b3
space 1024       44a781a4c328e4376b910a21fb3f7b96990b95c85f08e3e32ea8a9a2fe52b4e2
storage binary   19a53bbed1f4840ababa2caf9b0544d6b81e0d66aa052dc70780de55a54c200e
```

Status reported no dirty, stale, unindexed, deleted, or index-error source
files. Re-running the manifest/index verifier also proved selected-file and
indexed-file parity.

## Historical generation-2 parent and segment inventory

| Observation | chi | react-hook-form |
| --- | ---: | ---: |
| Files | 78 | 237 |
| Semantic parents | 452 | 275 |
| Functions | 263 | 58 |
| Methods | 130 | 0 |
| Types | 59 | 217 |
| Segment rows | 621 | 416 |
| Unique canonical document inputs | 619 | 416 |
| Chunks without a segment | 0 | 0 |
| Source-body/file-span mismatches | 0 | 0 |
| Invalid segment/projection ranges | 0 | 0 |
| Canonical inputs over the 256 KiB request cap | 0 | 0 |

The two duplicate chi segment rows are three occurrences of one identical
canonical input inside `middleware_test.Example_clientIP`; hash/byte
deduplication correctly produces 619 paid document keys from 621 segment rows.

### Historical generation-2 1,024-byte target observations

The persisted contiguous `display_start_byte`/`display_end_byte` span is not
the provider input and must not be used as the size statistic: later segments
retain a parent-oriented display prefix. Canonical document bytes are rebuilt
from the segment projections plus versioned path/kind/symbol/signature framing.

| Canonical-input bytes | chi | react-hook-form |
| --- | ---: | ---: |
| Minimum | 98 | 116 |
| Mean | 770.7 | 1,004.3 |
| p50 | 674 | 752 |
| p95 | 1,581 | 2,668 |
| Maximum | 3,624 | 6,049 |
| Unique inputs above 1,024 | 188 / 619 | 168 / 416 |
| Total unique input bytes | 475,564 | 417,789 |

Projection bytes before canonical metadata have p50 values 513 and 566,
p95 values 1,425 and 2,262, and maximum values 3,490 and 5,579 respectively.
There are 73 chi and 113 react-hook-form projections above the target. Manual
inspection shows the large production cases are indivisible top-level AST
statements or members, including nested closure declarations inside
`createFormControl`, `useFieldArray`, and `useController`; retaining them whole
matches the accepted oversize-unit contract. Other inputs cross 1,024 only
after versioned canonical metadata is added. The target is therefore an
AST-packing target, not a hard provider-input ceiling.

## Historical generation-2 inclusion distribution

| Corpus area | Files | Chunks | Unique inputs | Canonical bytes |
| --- | ---: | ---: | ---: | ---: |
| chi production | 35 | 224 | 257 | 173,623 |
| chi tests | 24 | 144 | 274 | 258,694 |
| chi examples | 19 | 84 | 88 | 43,247 |
| RHF production | 107 | 253 | 394 | 410,625 |
| RHF tests | 113 | 7 | 7 | 2,756 |
| RHF type tests | 17 | 15 | 15 | 4,408 |

This evidence records the distribution without changing the approved corpus.
The current full-project plan remains representative of ordinary cidx source
discovery; later reports must separate production, test, and example cohorts
so test-heavy chi cannot hide production behavior.

## Concrete findings

### Fixed during the audit: generation-keyed inventory publication

The inventory packet included `generation` in its content but its immutable
filename used only corpus ID plus source manifest. Moving from generation 1 to
generation 2 with unchanged source content therefore reproduced
`immutable packet collision`. The working implementation now includes
generation in the inventory reference. Provider-free reruns succeeded and
published:

```text
go-chi-chi-v5.3.1-g2-6bd4...302a29.json
  sha256 bc01e4d8805c25dfd7de1b43edd6f951e5879497bf061f748f23e36d85cae693
react-hook-form-v7.85.0-g2-54f6...f88b8a.json
  sha256 9d1e27277a9d145358ec19d6363b5ab805ab050b8d4e2f375bf92da9dc8fae94
```

The path remains immutable for one exact generation/manifest pair, so a
same-key content mismatch still fails closed.

### Accepted correction before document capture: anonymous default exports

RHF had 179 indexed files with no semantic parent. Of those, 119 are tests or
type tests and several are re-export/value-only modules. The original
parentless-file audit found 51 production files whose implementation is an
anonymous default-export arrow function. The corrected inventory later showed
six more such functions in files that already had type parents, for 57
production functions total. Examples include `validateField`, `generateId`,
`createSubject`, `getProxyFormState`, and many predicate/array utilities.

Phase 04 deliberately fixed anonymous default exports as unsupported because
they lacked a source-declared stable name. On this real corpus that choice
removes core behavior from FTS and document embedding entirely. The user
accepted using the repository-relative filename stem as a stable retrieval
symbol while preserving the exact source text and
path/range persistence identity, then increment the executable chunker
version and fully reindex before freezing the document universe. A synthetic
shared `default` symbol would preserve bodies but provide poor symbol
discrimination. Leaving the policy unchanged would require admitting that
these function bodies are outside the v1 searchable unit.

The user accepted this rule on 2026-08-16. No paid capture may proceed until
the versioned full reindex proves the resulting parent/segment universe stable.

### Fixed contract, live defect: overload groups split into duplicate parents

Behavior-cohort authoring exposed a second real-corpus structural issue. The
current index contains:

| File | Logical symbol | Persisted function parents | Cause |
| --- | --- | ---: | --- |
| `src/useWatch.ts` | `module.useWatch` | 8 | associated JSDoc occurs between overload signatures |
| `src/utils/insert.ts` | `module.insert` | 3 | exported default bodyless declarations are not classified as bodyless overload signatures |
| `src/__typetest__/form.test-d.ts` | `module.mockZodResolver` | 2 | a comment separates a bodyless signature from the implementation |

Phase 04 already requires a contiguous same-owner overload set and its
implementation to form one logical parent. These comments describe individual
signatures; they are not unrelated declarations and must not split result
identity. Likewise, a named function declaration with no body is bodyless
regardless of the Tree-sitter node-kind label. This is an implementation defect
under the existing contract, not a policy choice or a retrieval score to tune
around. It must be repaired and included in the same versioned full reindex as
the anonymous-default decision.

### Independent advisory review

The same aggregate-only design question was reviewed in the existing ChatGPT
and Grok side-panel conversations. Neither adviser received credentials,
source bodies, local paths, or private repository data. Both recommended:

- include the anonymous default-export functions under a deterministic,
  versioned path-derived retrieval label rather than a shared `default` label
  or continued exclusion;
- keep path/range/source bytes as exact source identity;
- retain the already-approved complete corpus universe and report production,
  test, example, and type-test cohorts separately; and
- regenerate all parent/input counts, fingerprints, token estimates, and
  request groups after the chunker change before producing an approval packet.

One adviser proposed a new dedicated alias field. The live v1 model already
has both `symbol` and `qualified_symbol`, and both are retrieval/display fields
used by FTS and canonical document text. Adding a third alias field would
therefore expand the production schema, FTS, evaluator, and MCP contracts
without improving the immediate identity guarantee. The narrower recommended
v1 change is:

```text
symbol            = normalized filename stem
qualified_symbol  = module.<normalized repository-relative path without extension>
source identity   = path + indexed content hash + byte range
```

This is an explicit, versioned retrieval-label exception for a top-level
anonymous default-export function-like declaration. It does not insert a name
into source text and does not permit synthetic names for anonymous callbacks,
destructuring bindings, computed properties, or other expressions. The user
confirmed this narrower existing-field contract on 2026-08-16. Phase 04
completed the extraction/overload/version correction and focused boundary
validation; Phase 07 resumed after the full provider-free reindex.

## Accepted generation-3 correction results

The corrected executable uses TypeScript chunker ID
`typescript-tsx-tree-sitter-0.23.2-jsdoc-class-fields-path-defaults-overloads-v2`
and index chunker version `2`. Both authorized corpora were rebuilt in full
with no provider, API key, or network operation:

| Corpus | Files rebuilt | Chunks | Segments | Index errors |
| --- | ---: | ---: | ---: | ---: |
| chi | 78 | 452 | 621 | 0 |
| react-hook-form | 237 | 322 | 492 | 0 |

RHF now has 57 path-labeled production function parents. Files without any
semantic parent fell from 179 to 128, accounting for the original 51-file gap;
the other six new functions coexist with type parents in their files.
`useWatch`, `insert`, and `mockZodResolver` each have exactly one persisted
function parent. Database `source_body` versus current indexed file-span
mismatches are zero in both corpora. The active index profile fixes
`target_segment_bytes=1024` and `chunker_version=2`.

New source-body-free inventory packets are:

```text
inventory/go-chi-chi-v5.3.1-g3-6bd4...302a29.json
  sha256 e85e524232cf6bc3af20cea7c7c8cff84696600749d718a86e74041f19da385d
inventory/react-hook-form-v7.85.0-g3-54f6...f88b8a.json
  sha256 876edcbbb824b2a6c86f04cad5ef82909ee6d4fe2413a27e3da21d797746cf01
```

## Current no-network capture plan

The corrected generation-3 universe has no raw-vector hits or terminal
failures. `EstimatedTokens` is deliberately one token per UTF-8 input byte, so
the value is also the exact aggregate canonical input byte count and a
conservative token ceiling:

| Corpus | Active distinct | Paid misses | Conservative token upper bound | Synchronous request groups |
| --- | ---: | ---: | ---: | ---: |
| chi | 619 | 619 | 475,564 | 5 |
| react-hook-form | 492 | 492 | 804,258 | 5 |

These values are diagnostic only and are not permission to call Voyage.

## Commands and checks actually run

```text
git -C <checkout> rev-parse HEAD
git -C <checkout> rev-parse HEAD^{tree}
git -C <checkout> status --porcelain
cidx index --root <checkout> --reason manual
cidx status --root <checkout> --json
cidx dev retrieval evaluate --mode lexical --inventory-only ...
cidx dev embeddings capture --root <checkout>        # plan only; no --apply
sqlite3 <production-db> <read-only structural audit queries>
env -u VOYAGE_API_KEY GOPROXY=off go build ...
go test -count=1 ./internal/devlab ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store
go test -count=1 -race ./internal/devlab ./internal/eval ./internal/evalcontract
go vet ./internal/devlab ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store
go build ./internal/devlab ./internal/eval ./internal/evalcontract ./internal/search/lexical ./internal/store
jq -e . testdata/retrieval/behavior-*-draft-v1.json
go mod tidy -diff
git diff --check
```

The SQL audit checked source-body bytes against their current file spans using
SQLite `readfile`, parent/segment coverage, projection containment, profile
JSON, unique canonical hashes, AST-target distributions, and corpus-area
counts. No test suite ran during the initial structural audit; the focused
evaluation packages were run once later at the completed behavior-binding
boundary. No provider/network operation ran at either boundary.

## Next gate

1. Bind the reviewed behavior cohort against the generation-3 parent universe
   with exact paths, content hashes, byte ranges, kinds, and digests.
2. Finalize the spend ceiling from current official provider pricing.
3. Return for explicit document-capture approval only after the exact final
   inputs, bytes, request count, and spend ceiling are known.
