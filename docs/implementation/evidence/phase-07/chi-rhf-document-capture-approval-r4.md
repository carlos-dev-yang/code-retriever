# Phase 07 chi/RHF Document-Capture Approval Packet — Revision 4

- Status: `awaiting_user_decision`
- Prepared: 2026-08-16
- Operation covered: document embedding only
- Query embedding covered: no
- Provider calls made while preparing this packet: `0`

## 1. Fixed input universe

Both repositories are on clean, pinned, user-approved checkouts and corrected
generation-3 production indexes. The document universe is the set of distinct
active canonical segment inputs, not source files or semantic parents.

| Corpus | Generation | Manifest SHA-256 | Parents | Segments | Distinct document inputs |
| --- | ---: | --- | ---: | ---: | ---: |
| chi v5.3.1 | 3 | `6bd4db89ee1a9cba70f69e125a803d147dbc0d92c95ef59b44be2dcb54302a29` | 452 | 621 | 619 |
| react-hook-form v7.85.0 | 3 | `54f6b1387ae989b1e49bdf21d3ed96189e76fb5b61b74ca282a2617c57f88b8a` | 322 | 492 | 492 |
| **Total** | — | — | **774** | **1,113** | **1,111** |

Two chi segments share canonical bytes, which is why 621 segments reduce to
619 distinct inputs. RHF has no duplicate canonical input in this generation.

## 2. Exact local request plan

`embed.ConservativeInputTokenUpperBound` charges one token per UTF-8 byte.
Therefore the existing plan's `EstimatedTokens` is both the exact aggregate
canonical input byte count and a deliberately conservative token ceiling. It
is not a claim about Voyage tokenization.

| Corpus | Raw hits | Paid misses | Exact canonical bytes | Conservative token ceiling | Synchronous requests |
| --- | ---: | ---: | ---: | ---: | ---: |
| chi | 0 | 619 | 475,564 | 475,564 | 5 |
| react-hook-form | 0 | 492 | 804,258 | 804,258 | 5 |
| **Total** | **0** | **1,111** | **1,279,822** | **1,279,822** | **10** |

The request contract is fixed at at most 128 inputs and 256 KiB of UTF-8 input
per synchronous request, at most four requests in flight, a 30-second attempt
timeout, and initial request plus at most three transient retries after
10/20/30 seconds. `Retry-After` is honored when longer. No asynchronous Batch
API is used.

## 3. Fixed vector/profile contract

```text
provider/model       Voyage AI / voyage-code-4
input_type           document
source dimensions    1024
provider output      float
truncation            false
segment target        1024 bytes at AST boundaries
serving dimensions    1024
storage codec         binary
reducer               prefix-l2-v1
normalizer            l2-v1
metric                cosine
```

Raw f32 document vectors are written only to ignored development-lab state.
After complete capture, local materialization transforms them to the selected
1024-dimensional binary representation and publishes only that cidx-owned
representation to production. Query vectors are not part of this approval.

## 4. Official model and price verification

The official [Voyage pricing](https://docs.voyageai.com/docs/pricing) and
[embedding model](https://docs.voyageai.com/docs/embeddings) pages were checked
on 2026-08-16 against their live HTML. They confirm `voyage-code-4` with a
32,000-token per-input context, output dimensions 1024 (default), 256, 512, and
2048, and intended use for code retrieval and coding agents. The official price
is $0.00012 per thousand tokens / $0.12 per million tokens, with the first 200
million tokens free per account.

An initial documentation-search extract was stale and omitted this model. A
direct read of the current official page HTML exposed the rows above and is the
authority for this corrected packet; the earlier absence conclusion is
withdrawn.

At $0.12 per million tokens, this packet's conservative 1,279,822-token
single-pass ceiling is $0.15357864 after the free allowance is exhausted. A
theoretical four-full-attempt ceiling (initial attempt plus all three retries)
is 5,119,288 tokens or $0.61431456. The proposed operation approval ceiling is
therefore $1.00. If the account still has its published free allowance, the
actual billed amount should be $0; provider-reported usage and account billing
remain the final operational evidence.

## 5. Decision required

The user must explicitly approve the canonical `voyage-code-4` document-only
capture of 1,111 inputs / 1,279,822 conservative tokens / 10 planned synchronous
requests under the proposed $1.00 ceiling before any provider call. Query
embedding is excluded and requires a later separate approval.

Supplying an API key alone is not approval. Until the bounded approval is
recorded, the command remains plan-only and no code or query text is sent to
Voyage.

## 6. Post-approval sequence

1. Re-run both plans and require the generation, manifest, distinct-input,
   byte, and request counts above to match exactly.
2. Capture chi and RHF document inputs with immediate per-group lab persistence
   and resumable failure records.
3. Record actual provider token usage and request/failure counts separately
   from the conservative local ceiling.
4. Materialize and activate the fixed 1024/binary profiles locally.
5. Run exploratory retrieval against the 12 Go, 12 TypeScript, and 8 TSX draft
   behavior cases to refine cohort direction without deleting difficult cases.
6. Ask for a separate paid-query approval before any hybrid/dense query run.
7. Freeze labels only after exploratory pooling and the required formal review
   passes; mixed-language work remains deferred until chi/RHF closure.

This packet is not promotion evidence and authorizes no operation by itself.
