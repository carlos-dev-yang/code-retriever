# Product Source-Vector Bank Decision

- Status: accepted owner decision
- Date: 2026-08-17
- Related product profile: default `1024/int8`, optional compact `512/int8`

## Accepted decision

Ordinary document embedding always preserves the provider's immutable
1024-dimensional f32 response so a project can change its active target between
1024/int8 and 512/int8 without another provider call. The owner selected
1024/int8 as the default and 512/int8 as the compact option on 2026-08-17.

## Pre-reconciliation implementation baseline

The shared transformer already derives a target vector by taking the leading
target dimensions of the 1024-f32 source and then L2-normalizing them.
Before this accepted change, development evaluation persisted immutable 1024-f32 document vectors in
`<state_root>/raw/embeddings.db`, and its materializer can generate a selected
target profile locally.

The pre-change public embedding path had no lab dependency,
transforms each provider response directly into the active production codec,
publishes that target vector, and retains only the raw-vector checksum. The
f32 values are then lost. Consequently, an ordinary project whose only copy is
an active int8 vector cannot currently change dimensions with exact local
rematerialization.

## Recommended product layout

```text
<state_root>/db/embeddings.db
  immutable document source bank
  key: source-profile fingerprint + canonical-input SHA-256
  value: validated 1024-dimensional f32 + provider provenance

<state_root>/db/index.db
  active AST, FTS, segment keys, and exactly one target vector set
  target: 1024/int8 by default or explicit compact 512/int8
```

The source bank is product state, not an evaluation lab. Search, serve, status,
and MCP do not open it. Only explicitly applied document embedding and local
target materialization use it. Evaluation metadata, query accounting, review
artifacts, and experimental variants remain in the separate test/lab
namespace.

## Exact dimension-change flow

1. Resolve a current 1024/int8 or 512/int8 config.
2. Reconcile the new vector-space and storage fingerprints against the active
   canonical input set.
3. Read compatible 1024-f32 source rows from `embeddings.db`.
4. For every row, take the first target-dimension values, L2-normalize, and
   encode with the cidx int8 codec.
5. Atomically publish the complete target set and active profile in
   `index.db`.
6. Remove or ignore the previous target set only after successful publication.

No provider call is needed when every current canonical input has a compatible
source row. Missing or incompatible source rows are reported as pending; only
those rows need a separately approved document embedding operation.

## Why the stored 1024-int8 target is not the source

The int8 codec calculates one scale from the maximum absolute value of the
whole target vector. A 1024-int8 vector therefore uses a scale derived from all
1024 components. A directly materialized 512-int8 vector uses a scale derived
from only the first 512 components. Cutting the encoded 1024-int8 blob can
retain the wrong quantization resolution and is not equivalent to deriving
512/int8 from source f32.

The authoritative conversion is therefore:

```text
1024-f32 source -> prefix(512 or 1024) -> L2 -> int8 target
```

It is never:

```text
1024-int8 target -> truncate bytes -> 512-int8 target
```

## Compatibility and safety

- Source rows are immutable and content-addressed. Conflicting bytes for the
  same source-profile/input key fail closed.
- The source bank is not a second serving authority and is never a search
  fallback.
- Query f32 remains request-local and is never persisted.
- Binary and 256 remain absent from config, materialization, search, and
  evaluation code. Historical reports do not enter this bank.
- Git ignores both databases. Moving a project requires copying `.cidx` when
  the owner wants to retain embeddings; otherwise cidx can rebuild free AST/FTS
  state and re-embed only after approval.
- A source-bank write is durable before target publication. If publication
  fails, the successfully captured source row remains reusable and the old
  active target remains authoritative.

## Phase impact if accepted

| Phase | Required change |
| --- | --- |
| 02 | Fix 512/1024 int8 profiles and add code-owned source-bank location/identity contracts. |
| 05 | Reprove profile reconciliation and pending-source-key behavior; AST/FTS remains provider-free. |
| 08 | Separate reusable product source-bank storage from evaluation-only run/materialization metadata. |
| 09 | Materialize either supported target exclusively from compatible 1024-f32 source rows. |
| 10 | Make public embedding durably record source f32 before publishing the active int8 target; reuse hits without a provider call. |
| 11 | Keep search int8-only and source-bank-independent. |
| 13 | Expose 1024 default and 512 compact option; make dimension change plan/apply behavior explicit without a codec flag. |
| 14 | Verify a provider-free 1024-to-512 rematerialization in package smoke using a synthetic source bank. |

The rejected alternative—discarding source f32 after target publication—is not
the product contract because it would make a later dimension change require an
avoidable paid document embedding.
