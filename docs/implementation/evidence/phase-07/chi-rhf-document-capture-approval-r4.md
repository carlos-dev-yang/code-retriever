# Phase 07 chi/RHF Document-Capture Approval Packet — Revision 4

- Status: `document_capture_complete`
- Prepared: 2026-08-16
- Approved: 2026-08-16 by the user, with a $5 account billing limit and an
  explicit instruction to proceed
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

## 5. Approval recorded

The user approved the canonical `voyage-code-4` document-only capture of 1,111
inputs / 1,279,822 conservative tokens / 10 planned synchronous requests on
2026-08-16. The account billing limit is $5; this packet's theoretical
four-full-attempt estimate is $0.61431456. The apply must stop before a provider
call if the generation, manifest, input, byte, request, profile, or retry plan
differs from this packet. Query embedding is excluded and requires a later
separate approval.

Supplying an API key alone was not approval; the user's explicit instruction is
the approval authority for this bounded operation.

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

This packet is not promotion evidence. It authorizes only the bounded document
capture above and does not authorize query embedding.

## 7. Approved-operation preflight

The clean binary built from commit
`1e72820f68569ec6028b6218820de13b00e36f7e` reported
`source_modified=false`. Its provider-free plans matched this packet exactly:

| Corpus | Generation | Manifest | Active distinct | Raw hits | Paid misses | Conservative tokens | Requests |
| --- | ---: | --- | ---: | ---: | ---: | ---: | ---: |
| chi | 3 | `6bd4db89ee1a9cba70f69e125a803d147dbc0d92c95ef59b44be2dcb54302a29` | 619 | 0 | 619 | 475,564 | 5 |
| react-hook-form | 3 | `54f6b1387ae989b1e49bdf21d3ed96189e76fb5b61b74ca282a2617c57f88b8a` | 492 | 0 | 492 | 804,258 | 5 |

The initial apply attempt did not start because `VOYAGE_API_KEY` was absent.
The user then placed a single credential entry in ignored local
`.cidx/credentials.env`; its mode was corrected to `0600`. The launcher read
only `VOYAGE_API_KEY`, exported it only to each cidx child process, and repeated
the exact plan checks before the first provider call. The credential value was
not printed, copied into tracked state, or passed as a CLI argument.

## 8. Completed capture and materialization

Both approved captures completed without failure or retry:

| Corpus | Run | Requested/persisted | Failed | Actual provider tokens | Raw format |
| --- | ---: | ---: | ---: | ---: | --- |
| chi | 1 | 619/619 | 0 | 131,433 | 1024 × f32, 4,096 bytes |
| react-hook-form | 1 | 492/492 | 0 | 200,080 | 1024 × f32, 4,096 bytes |
| **Total** | — | **1,111/1,111** | **0** | **331,513** | — |

Every request and response model was `voyage-code-4`, the raw encoding was
`cidx-lab-f32-le-v1`, and `capture_failures` remained empty. At the official
$0.12/million rate, observed usage is $0.03978156 after the free allowance is
exhausted and $0 while sufficient published free allowance remains.

Provider-free materialization then published the fixed
`19a53bbed1f4840ababa2caf9b0544d6b81e0d66aa052dc70780de55a54c200e`
serving profile:

| Corpus | Raw coverage | Staged/published vectors | Codec | Stored bytes/vector | Segment coverage |
| --- | ---: | ---: | --- | ---: | ---: |
| chi | 619/619 | 619 | `cidx-binary-sign-lsb-v1` | 128 | 621/621 |
| react-hook-form | 492/492 | 492 | `cidx-binary-sign-lsb-v1` | 128 | 492/492 |

Post-capture plans report all 1,111 inputs as raw hits, zero paid misses, and
zero request groups. The ignored local database SHA-256 values at this boundary
are:

```text
chi lab         b070f80b516e1b9ce577b3c505d1fb7c8fcea62b5857e1d26dfd7fa5c70873f4
chi production  1cf8451a8d6dda4cac91a6685cf1bf3efb98edfffff53d85ed333caf9e961625
RHF lab         108c20a79005c3a81edeccb385fc9afca72a2f3a0518ad5e77e58465489ab122
RHF production  9992220ce27b898fd88357dacf0bce3b338b075eff2ae3ad63ae3e8332e10958
```

No query embedding occurred. The next paid gate is the separate
[query-evaluation approval packet](chi-rhf-query-evaluation-approval-r4.md).
