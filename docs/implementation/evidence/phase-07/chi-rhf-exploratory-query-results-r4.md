# Phase 07 chi/RHF Exploratory Query Results — Revision 4

- Status: `diagnostic_complete_decisions_pending`
- Run date: 2026-08-16
- Authority: draft calibration preparation only
- Serving profile: 1,024 dimensions, binary codec
- Segment target: 1,024 bytes
- Provider/model: official Voyage AI / `voyage-code-4`
- Code provenance: `59b1cd61ec990c56cea275f5ac1b258e7eb5332a`
- Trace correction: `8a9791c` and its Phase 12 evidence

## 1. Execution and integrity

The exact approved two-invocation series ran once after the clean binary
reproduced both frozen provider-free plan hashes. No retry or repeat occurred.

| Corpus | Queries | Attempts | Validated | Retries / failed | Observed total tokens | Artifact entry-list checksum |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| chi | 12 | 12 | 12 | 0 / 0 | 221 | `7a538245b3e74f106cdf318e31843a798670e1c9a9ff095bd1025e9ade812967` |
| react-hook-form | 20 | 20 | 20 | 0 / 0 | 415 | `8dfd28b7d8e8de082ad9e0a964566758af69aa8e0c08261badf976344637651f` |
| **Total** | **32** | **32** | **32** | **0 / 0** | **636** | — |

At the price frozen in the approval packet, the observed 636 tokens correspond
to `$0.00007632`; this is accounting for the approved operation, not a new price
claim. Every listed artifact file passed SHA-256 verification and the matching
lab rows are `complete`. Query-vector values are absent from production and lab
schemas and artifacts; only their SHA-256 digests appear on comparable arms.

Ignored local references:

- chi: `.cidx/test/states/chi/evaluations/retrieval-7e5731ed1222a6aa432da84f`;
- RHF: `.cidx/test/states/react-hook-form/evaluations/retrieval-20417011198b38cad4a1af2b`.

## 2. Observed top-5 retrieval metrics

These small draft sets are diagnostic only. They cannot select a codec,
establish a threshold, or vote for promotion.

| Corpus / arm | Hit@5 | Recall@5 | MRR@5 | NDCG@5 | Complete required groups |
| --- | ---: | ---: | ---: | ---: | ---: |
| chi FTS | 0/12 | 0.000 | 0.000 | 0.000 | 0/12 |
| chi target f32 | 11/12 | 0.917 | 0.792 | 0.807 | 11/12 |
| chi active binary | 12/12 | 0.958 | 0.785 | 0.785 | 11/12 |
| RHF FTS | 0/20 | 0.000 | 0.000 | 0.000 | 0/20 |
| RHF target f32 | 19/20 | 0.925 | 0.623 | 0.735 | 18/20 |
| RHF active binary | 17/20 | 0.825 | 0.621 | 0.707 | 16/20 |

The active-codec hybrid equals the active dense arm in this run because FTS
returned no candidates for every natural-language behavior question. That is a
known property of the current production FTS query contract: every normalized
query token is joined with `AND`. It is not evidence that FTS indexing, parsing,
or stored symbol labels are absent.

## 3. Stage diagnosis

Source discovery, parser/chunker truth inventory, active generation, document
raw coverage, and production vector coverage succeeded for every query. The
exhaustive dense segment inventories contain the required semantic parents.
There is therefore no current evidence to reopen the 1,024-byte segment target
or the Go/TypeScript/TSX parser/chunker contracts.

The first run revealed a trace-only defect: required parent spans were compared
with smaller segment ranges. Rankings and metrics already compared semantic
parents correctly. The correction at `8a9791c` uses each segment's recorded
parent coordinates; a real-artifact calculation found 5 affected chi dense
observations and 11 affected RHF dense observations. The immutable run is not
rewritten. Future authorized runs will publish the corrected trace.

With that correction, observed serving-profile losses begin at top-5 parent
collapse:

- chi `g07`: `selectEncoder` survives at rank 1, while the second required
  parent `isCompressible` falls outside binary top 5;
- RHF `t01`: the direct `useForm` parent is f32 rank 5 but outside binary top 5;
- RHF `t10`: `PathInternal` survives but the current second required parent
  `Path` does not; direct source review also shows the cycle guard is actually
  in `PathImpl`, so this is a label defect as well as a retrieval observation;
- RHF `x01`: the direct `Controller` parent is f32 rank 4 but outside binary
  top 5;
- RHF `x08`: `FormStateProps` is outside both top-5 arms, while its already
  judged support parent `FormState` ranks second.

Binary also rescues chi `g09` (`RealIP` at binary rank 2) after target f32 is
dominated by the newer `client_ip` implementation/tests. This, together with
the RHF regressions, confirms that sign-binary can materially reorder close
parents. The sample is too small and labels remain too provisional to make a
codec-selection claim; the user's fixed 1,024/binary working profile remains
unchanged.

Body packaging created no additional required-group loss beyond the active
top-5 ranking. All 60 chi packages were complete. RHF had 83 complete, 4
partial, and 13 budget-omitted packages among 100 returned parents, but every
required parent that survived ranking also retained the source range necessary
for its current required group. Assistant-use and assistant-resolution remain
`NOT_OBSERVED`.

## 4. Language and cohort direction

- Go binary Hit@5 is 12/12, but the five-case delegated/cross-parent cohort has
  only 0.90 mean requirement coverage because `g07` requires two parents.
- TypeScript binary Hit@5 is 11/12; TSX is 6/8. The TSX gap is concentrated in
  thin wrapper/type-contract cases rather than parser absence.
- RHF `task:interface_type_or_api_contract` is the weakest working cohort:
  Hit@5 2/3, requirement coverage 0.50, complete requirements 1/3.
- RHF `task:delegated_or_cross_parent_flow` is next: Hit@5 5/7 and complete
  requirements 5/7.
- RHF lifecycle and single-parent behavior cases are 5/5 at Hit@5 and complete
  requirement coverage under the active profile.

The direction is therefore to keep difficult multi-parent, thin-wrapper, and
type-contract questions. Removing them would hide the observed failure mode.
New confirmation questions must remain independently authored and cannot be
reworded copies of these failures.

## 5. Pooled-label corrections proposed before freeze

### T10 — revise direct truth

Current draft truth incorrectly makes public `Path` and `PathInternal` the two
direct groups. Source review at the pinned RHF commit shows:

- `module.PathImpl`, `src/types/path/eager.ts`, bytes 408–1042: constructs the
  dotted string and contains the already-seen-type guard;
- `module.PathInternal`, same file, bytes 1044–1649: dispatches tuple, array,
  and object recursion;
- `module.Path`, same file, bytes 1651–2036: public distributive wrapper.

Proposed revision: required grade-2 groups are `PathImpl` and `PathInternal`;
`Path` becomes grade-1 useful support. Keep the query text, TypeScript language,
`BEST_N`, and current task/signal tags. This creates a new draft digest.

### G09 — add one reviewed hard negative

The question explicitly asks about deprecated `RealIP`, header precedence, and
`RemoteAddr` mutation. The f32 pool instead surfaced the newer client-IP code.
`middleware.walkXFF` in `middleware/client_ip.go` (bytes 7967–8801) is a useful
misleading negative: it walks forwarded headers for the safer replacement API
but does not implement the deprecated precedence/mutation contract. Proposed
revision: retain `RealIP` as grade 2 and `realIP` as grade 1; add `walkXFF` as a
grade-0 hard negative with the distinction above. This requires corpus-wide
evidence and the later separated second review pass before freeze.

### Cases retained unchanged

- chi `g07` remains a deliberate two-required-parent compression flow;
- RHF `t01` remains direct `useForm` with already judged delegated support;
- RHF `x01` remains direct `Controller` with `useController` support;
- RHF `x08` remains direct `FormStateProps` with `FormState` support.

Their misses are useful calibration evidence, not grounds to weaken truth.

The existing Grok advisory review was rechecked against the pinned source and
actual candidate pools. It remains advisory only. In particular, its earlier
acceptance of `Path`/`PathInternal` as complete T10 truth is superseded by the
direct source location of the guard in `PathImpl`.

## 6. Deterministic simple-search decision

The simple baseline is an evaluation control, not a production search change.
The existing-field label decision remains authoritative: it uses stored `symbol`
and `qualified_symbol`; it adds no alias column or public wire.

Recommended frozen baseline:

1. scan the same semantic-parent inventory as FTS;
2. normalize query, path, symbol, qualified symbol, signature, and body with one
   versioned language-neutral identifier/text normalizer;
3. admit a parent when at least one normalized query token is literally present;
4. rank exact qualified-symbol match, exact symbol match, path match, then
   descending distinct matched-token count;
5. break ties by normalized path, parent start byte, and stable parent identity;
6. record returned count and an exact algorithm fingerprint; use no BM25,
   embedding, learned/corpus/language boost, or per-query exception.

This makes the earlier ambiguous “literal token presence” rule explicit as
`ANY` candidate admission. Requiring all natural-language tokens would simply
reproduce the current all-token FTS emptiness and would not be a useful weak
control.

The source audit also closes the remaining mechanical ambiguities in the
proposal. Query tokens are the stable-deduplicated union from the existing
`symbol.ClassifyQuery` and `identifier-split-lower-v1` normalizer. Parent fields
use the same tokenizer. Exact symbol checks compare fully normalized strings;
path match means any query token occurs in the normalized path; matched-token
count is distinct across the union of all five fields. The stable tie tuple is
normalized path, start byte, end byte, qualified symbol, then indexed content
hash. The fingerprint seals those definitions. A development-only,
generation-pinned read snapshot supplies authoritative parents and stored
bodies; no public FTS, MCP, schema, or production-ranking change is proposed.

## 7. Blind-pool coverage prepared

The exploratory top-5 outputs were combined across FTS, serving f32, active
binary, and their RRF observations, then deduplicated by semantic-parent
identity. The generated pass-1 packets remove arm identity, native score, and
original rank. Candidate order is deterministically shuffled from a recorded
seed. Each catalog entry retains the verified source path/hash, parent byte and
line range, local source path, and exact parent body.

| Corpus | Queries | Unique parents | Query-parent judgments | Already present in machine draft | Newly unjudged |
| --- | ---: | ---: | ---: | ---: | ---: |
| chi | 12 | 64 | 82 | 22 | 60 |
| react-hook-form | 20 | 58 | 123 | 34 | 89 |
| **Total** | **32** | **122** | **205** | **56** | **149** |

Ignored local packets:

- chi: `.cidx/test/states/chi/evaluations/review/pool-v1-chi-exploratory-top5.json`,
  SHA-256 `b1e2c572026ac2bd8aa61b36fead308366b14142dffe54bc1beb6857a1d18f83`;
- RHF: `.cidx/test/states/react-hook-form/evaluations/review/pool-v1-rhf-exploratory-top5.json`,
  SHA-256 `818b35cb299a3273419865aee6958886c203cf82ad640361fa0ff859ed64414e`.

Both packets are `MACHINE_PREPARED_UNREVIEWED`, contain null decisions, and
are non-promotional. They deliberately include all 205 query-parent relations:
every retrieved parent plus all 56 machine-draft truth parents, including truth
that missed every top-5 arm. The labels remain hidden so the human first pass
does not inherit those machine grades. Deterministic simple-search candidates
remain pending the policy decision below. Int8 is an unopened alternative and
is out of scope for the user-selected 1,024/binary initial grid.

These counts also correct the earlier impression that only three headline
label decisions remain. T10, G09, and the simple baseline are the policy gates;
after those gates, all 205 pooled relations still require a blind first pass and
a separated second pass before the dataset can be frozen.

Separate ignored machine-suggestion overlays prepare, but do not replace, that
human work:

- chi: `.cidx/test/states/chi/evaluations/review/machine-suggestions-v1-chi.json`,
  SHA-256 `a453e50560e722c5e64f4db667c34f3d4584f4f1d9b9e0fc1701aa0878326b4f`;
- RHF: `.cidx/test/states/react-hook-form/evaluations/review/machine-suggestions-v1-rhf.json`,
  SHA-256 `01553e8030e06ed4ba256e6317dad473cc1681865e9bc8abce99faf38505f29b`.

Across both overlays the advisory distribution is 111 grade-0, 59 grade-1,
and 35 grade-2 relations. Every grade-2 suggestion has a proposed required
group. The overlays preserve every existing machine-draft grade except the
explicit T10 `Path` grade-2 to grade-1 proposal; they also mark the T10 three
parents and G09 `walkXFF` as material decision gates. Their status is
`ADVISORY_NOT_LABEL_AUTHORITY`, and their merge policy forbids dataset changes
without explicit user acceptance and two recorded human passes. Review pass 1
uses the label-free packets above; these overlays are a later reconciliation
aid, not a way to seed the blind decisions.

## 8. Remaining decision gate

Before creating a new working-dataset digest or implementing the simple
baseline, the user must confirm:

1. the T10 direct-truth revision;
2. the G09 `walkXFF` hard-negative addition;
3. the six-point simple-search policy above.

No further provider operation is required for those decisions. After they are
accepted, update the draft dataset, generate blinded provider-free simple/FTS
pool additions, complete two separated label-review passes, and freeze the
chi/RHF calibration digest. A later paid calibration apply remains a separate
approval.
