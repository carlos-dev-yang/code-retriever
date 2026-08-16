# Phase 07 chi/RHF Query-Evaluation Approval Packet — Revision 4

- Status: `approved_for_one_bounded_series`
- Prepared: 2026-08-16
- Operation covered: query embedding for the 32 draft behavior cases only
- Document embedding covered: no — complete before this packet
- Mixed-language or confirmation work covered: no
- Provider calls made while preparing this packet: `0`

## 1. Completed document prerequisite

The approved `voyage-code-4` document capture completed with 1,111/1,111 raw
1024-f32 inputs persisted and zero failures. Local 1024/binary materialization
published 619 distinct chi vectors and 492 distinct RHF vectors. Production
coverage is 621/621 chi segments and 492/492 RHF segments, with no pending or
failed vector rows. Both corpora use serving profile
`19a53bbed1f4840ababa2caf9b0544d6b81e0d66aa052dc70780de55a54c200e`.

## 2. Exact provider-free query plan

The clean `source_modified=false` binary from
`1e72820f68569ec6028b6218820de13b00e36f7e` completed full corpus, dataset,
truth, raw-bank, active-profile, production-vector, and evaluation-session
preflight without a provider call.

| Corpus | Generation | Dataset SHA-256 | Query operations | Conservative query-token ceiling | Raw document inputs |
| --- | ---: | --- | ---: | ---: | ---: |
| chi | 3 | `4bbb0adc14ee9ab4679318d3a730a4adb9a56675a505569f50328eebdf42dd16` | 12 | 1,307 | 619 |
| react-hook-form | 3 | `da019071853ef7f099185aa123f14a90f9c10ad5572e53939e26b6d9ec315681` | 20 | 2,391 | 492 |
| **Total** | — | — | **32** | **3,698** | **1,111** |

The portable plan JSON SHA-256 values are
`b57c76d04dd55da08488af4170943d4bf26e9f3bf5c5672952875f77e60cbfc5`
for chi and
`d1c259f40ee262316ee2a7f2008aac057a1efd9828fadf388687ce6513882997`
for RHF.

## 3. Fixed provider contract

```text
provider/model       Voyage AI / voyage-code-4
input_type           query
source dimensions    1024
provider output      float
truncation            false
attempt timeout       30 seconds
maximum retries       3 after 10/20/30 seconds
serving transform     prefix-l2-v1 + l2-v1
evaluation codec      target f32 and active 1024/binary arms
```

Each case is one logical query operation. A validated query vector is reused
in memory across that case's retrieval arms and is never persisted. Provider
usage records logical operations, attempts, retries, validated responses,
failed attempts, and provider-reported tokens independently from retrieval
quality.

## 4. Price and approval ceiling

The dated official `voyage-code-4` price recorded in the document-capture
packet is $0.12 per million tokens after the published 200-million-token free
allowance. The conservative 3,698-token single-pass ceiling costs $0.00044376
after that allowance is exhausted. The theoretical four-full-attempt ceiling
is 14,792 tokens or $0.00177504.

The approved ceiling is **$0.01** for this complete 32-query operation. The
user's $5 account billing limit and the completed document approval did not
implicitly approve it; the explicit authorization is recorded below.

## 5. Claim boundary and authorization record

These are draft calibration questions. The resulting run may guide pooling and
cohort revision, but it is not a frozen-label, confirmation, mixed-language,
promotion, or release-candidate result. Difficult cases remain in the working
set until reviewed; the run does not authorize label deletion or threshold
tuning on confirmation data.

On 2026-08-16 the user instructed Codex to set the remaining work as a goal and
continue. This authorizes exactly the two-invocation series in this packet:
12 chi plus 20 react-hook-form draft exploratory queries, at most $0.01 total.
It does not authorize a changed dataset/profile, a repeated exploratory run,
formal calibration, confirmation, mixed-language work, or assistant use.

Immediately after that instruction, provider-free preflight at commit
`e06e28a` reproduced both packet plans byte-for-byte, including the original
plan SHA-256 values `b57c76d04dd55da08488af4170943d4bf26e9f3bf5c5672952875f77e60cbfc5`
and `d1c259f40ee262316ee2a7f2008aac057a1efd9828fadf388687ce6513882997`.
The generation, manifests, dataset digests, raw counts, query counts, token
ceilings, and serving profile remain unchanged. A clean-provenance binary must
repeat this same preflight before the first request.
