# 12. Retrieval Evaluation

- Status: `planned`
- Prerequisites: `07-lexical-evaluation`, `08-raw-embedding-lab`, `09-vector-materialization`, `11-vector-and-hybrid-search`
- Followed by: `13-cli-and-mcp`, `14-packaging-and-host-integration`
- Design source: `local-code-search-mcp-v1-design-r3.md` sections 7.4 and 14
- Evaluation authority: [EVALUATION-CONTRACT.md](EVALUATION-CONTRACT.md)

## Context Recovery Checklist

Read the [implementation index](README.md), [execution guide](EXECUTION-GUIDE.md), [evaluation contract](EVALUATION-CONTRACT.md), and [project status](STATUS.md) before resuming.

- Confirm the user has selected each open-source corpus and approved the exact tracked manifest before any checkout, download, indexing, or paid embedding occurs.
- Confirm each tracked corpus manifest records `corpus_id`, upstream URL, pinned commit, SPDX/license evidence, language slices, expected clean-tree/content hash, root subdirectory, and include/exclude policy; it must not contain an environment-specific absolute path.
- Confirm an ignored `.cidx/lab/corpora.local.json` binding or explicit CLI input maps each `corpus_id` to the user's existing checkout, and verify its commit, cleanliness, and content hash before a run.
- Re-check that all compared profiles use the same approved document raw bank, manifest, canonical input set, question/answer dataset, and production search implementation, and that aggregate metrics retain separate Go, TypeScript, TSX, and mixed counts and denominators.
- Re-check source 1024, targets 256/512/1024, nonpersistent query f32, cidx-owned `binary` and `int8` candidates, binary as the production default, and one active serving profile per run.
- Re-check the frozen calibration/confirmation split, exact denominators, required-group labels, first-loss enum, FTS/dense lane traces, target-f32 codec reference, RRF ablations, and promotion contract before any applied run.
- Stop if corpus approval, license evidence, pinned revision, clean-tree/content hash, ground truth, raw coverage, or fingerprint compatibility is unresolved. Never choose, download, update, index, or embed a corpus on the user's behalf without explicit approval.
- Before pausing, update run/evidence records and this decision log, then update [STATUS.md](STATUS.md) with approved corpus IDs, checked hashes, incomplete work, and the exact next action.

## 1. Objective

Sequentially compare supported target dimensions and the cidx-owned `binary` and `int8` codecs against the same `voyage-code-4` 1024-dimensional document raw bank and the same question/answer dataset. Each run evaluates exactly one `ServingVectorProfile` selected by `.cidx/config.json` and records evidence for the next configuration decision. Binary remains the resolved default unless the user explicitly selects int8.

Produce a stage-separated scorecard and hard-gate result rather than a weighted total. Human relevance determines usefulness; exhaustive target-dimension f32 determines representation fidelity. Preserve FTS-only, dense-only, collapse, RRF, and body-survival evidence so a correct hybrid result cannot hide a broken lane or downstream loss.

The evaluation corpus is a user-selected, pinned open-source project. cidx consumes approved manifests and existing local checkouts; this phase does not search for, select, download, or silently update projects.

For each run, explicitly request queries from Voyage AI with `output_dimension=1024`, `output_dtype=float`, `input_type=query`, and `truncation=false`, transform them into the same vector space, and use them only in memory. Even fixed evaluation-query vectors are never persisted.

## 2. Scope and Non-goals

### In scope

- User-approved open-source corpus manifests and local checkout bindings.
- Reproducible retrieval datasets with expected symbols/files.
- Target-f32 baselines derived from one document raw bank.
- Current production selected-codec results.
- FTS-only, vector-only, and FTS+vector RRF comparison.
- FTS/dense candidate union plus hybrid-without-FTS and hybrid-without-dense ablations.
- f32/binary/int8 score and ranking observations.
- Parent-collapse, RRF contribution/rescue/harm, body-package survival, first-loss, and per-cohort evidence.
- Sequential target-dimension and codec comparisons.
- Free planning separated from paid query apply.
- Frozen promotion contract/result plus immutable traces, metrics, and checksummed artifacts.

### Out of scope

- Selecting, cloning, downloading, updating, indexing, or embedding a corpus without explicit user approval.
- Committing local absolute checkout paths or vendoring full third-party repositories into this project.
- Query-vector tables or caches.
- Concurrent runtime A/B serving of multiple candidate profiles.
- Automatic config edits or automatic profile promotion by evaluation.
- Importing vectors for models absent from the raw bank.
- Fixed latency/hit-rate release thresholds or SLAs.
- Learned rerankers, LLM judges, generated-answer evaluation, or permanent product operation of the raw bank.

## 3. Prerequisites

- Phase 07 provides the shared dataset format and FTS baseline runner.
- The user has selected the corpus, approved its pinned manifest and license use, and made an approved checkout available locally.
- The local binding resolves `corpus_id` to a path outside the tracked manifest; the checkout matches pinned commit, clean-tree rule, expected content hash, root subdirectory, and include/exclude language slice.
- Phase 08 completely covers the approved evaluation manifest and source profile in its raw document bank.
- Phase 05 records active fingerprints and segment keys for current config; Phase 09 publishes its serving vectors.
- Phase 11 exposes the actual vector scorer, aggregation, and RRF as reusable entry points.
- The dataset records corpus/revision, query ID/text, expected file/symbol, and permitted answer sets.
- `VOYAGE_API_KEY` and paid-query permission are needed only for `--apply`.

## 4. Invariants

1. No corpus network access, checkout creation/update, indexing, or embedding occurs until the user explicitly approves that corpus and action.
2. A tracked manifest contains portable identity and integrity data, never an absolute local path; local bindings are ignored or passed explicitly.
3. Every dimension/codec comparison uses the same approved corpus commit, clean/content hash, document source fingerprint, manifest, and canonical input set.
4. One run evaluates one current production `ServingVectorProfile`.
5. Changing a candidate requires an explicit config edit, `cidx index` reconciliation, then `cidx dev embeddings materialize --activate`.
6. Evaluation never writes config or changes the active serving profile.
7. Each run creates a fresh query from current `voyage-code-4` at 1024 dimensions and never persists it or reuses it in another run.
8. Within one run, one ephemeral target query may be reused across f32 baseline, the active codec scan, and hybrid variants, then discarded.
9. Document target-f32 baselines apply the current reducer/normalizer to raw documents in memory and never enter production storage.
10. Production binary/int8 variants read only the current active serving profile.
11. Document and query both start at source 1024 and use the same prefix plus L2 transform.
12. Runs differing in corpus/commit/content hash, manifest, dataset, config, raw profile, or code version are not labeled as like-for-like.
13. Failed or missing queries are never silently removed from metric denominators.
14. Hit rate and latency are observations, not predeclared automatic candidate gates.
15. Validate query response model/count/indexes, 1024 dimensions, and finite values; omit `encoding_format`.
16. Production binary/int8 representations are Phase 09 cidx codecs, not Voyage provider-quantized output.
17. Human-gold retrieval and target-f32 codec fidelity are independent metrics; neither substitutes for the other.
18. Calibration may select settings and margins but cannot vote for promotion. Confirmation cannot tune any setting, label, budget, or margin.
19. Required query/API failures and timeouts remain in denominators with an explicit failure stage. Optional unrequested stages are `NOT_OBSERVED`, not zero.
20. Activation and a valid serving profile are lifecycle evidence only, not quality admission.

## 5. Implementation Packages, Files, and Types

| Package/file | Responsibility |
| --- | --- |
| `eval/corpora/*.json` | Tracked, approved, portable corpus manifests; no local paths |
| `.cidx/lab/corpora.local.json` | Ignored local `corpus_id` to checkout-path binding |
| Phase 02 `internal/evalcontract` | Shared dataset, stage trace, failure, artifact, and promotion wire contracts |
| Phase 07 `internal/eval/{ground_truth,metrics,report}.go` | Shared truth mapping, metric calculations, and report logic |
| `internal/eval/corpus_manifest.go` | Strict manifest loading, license/revision/hash policy |
| `internal/eval/corpus_binding.go` | Resolve explicit/ignored local checkout binding |
| `internal/eval/retrieval_runner.go` | Orchestrate lexical/target-f32/current-codec/hybrid variants |
| `internal/eval/retrieval_metrics.go` | Human retrieval, requirement, survival, first-loss, and target-f32/current-codec fidelity metrics |
| `internal/eval/variants.go` | Variant definitions and common rank adapter |
| `internal/eval/promotion.go` | Frozen gate contract, paired comparison, and promotion result |
| `internal/lab/eval_documents.go` | Target-f32 view over the raw bank |
| `internal/lab/eval_runner.go` | One-current-config development run |
| `internal/lab/evaluations.go` | Vector-free run provenance in the Phase 02 lab schema |
| `internal/app/dev_evaluate.go` | Plan/apply, confirmation, progress, and result for Phase 13 |

Call the Phase 11 implementations directly; do not copy a similar evaluation-only algorithm.

Core types reuse Phase 02 `EvaluationCase`, `RequiredGroup`, `ExpectedAlternative`, relevance grades, cohorts, split, stage, first-loss, artifact, and promotion types plus Phase 07 truth/metric services. Phase 12 adds only retrieval calculations such as `EvaluationPlan`, `EvaluationRun`, `CaseRanking`, `CodecFidelity`, `LaneContribution`, and `MetricSummary`. `RetrievalVariant` includes `fts`, `target_f32`, `serving_active_codec`, each supported RRF arm, provider union, and the two lane-ablation hybrids. The run manifest records whether the active codec is binary or int8.

## 6. Schema, API, and CLI

### Portable corpus manifest

The repository tracks one reviewed manifest per approved corpus. Minimum fields are:

```json
{
  "corpus_id": "user-approved-stable-id",
  "upstream_url": "https://approved.example/repository.git",
  "pinned_commit": "full-commit-id",
  "license_spdx": "SPDX-ID",
  "license_evidence": "upstream-relative-license-path-or-reviewed-note",
  "root_subdir": ".",
  "language_slices": ["go", "typescript", "tsx"],
  "include": ["approved/globs/**"],
  "exclude": ["approved/exclusions/**"],
  "expected_tree_hash": "reviewed-tree-or-slice-hash",
  "expected_content_hash": "canonical-selected-content-hash",
  "clean_tree_required": true
}
```

- `upstream_url` is provenance, not permission to fetch.
- `pinned_commit` is a full immutable revision, never a branch or tag alone.
- Record license/SPDX and evidence before paid processing; a missing or incompatible license is a stop condition.
- `root_subdir`, include/exclude patterns, and language slices define the exact indexed subset.
- Define canonical path ordering, file-byte hashing, line-ending treatment, and symlink rejection once so `expected_content_hash` is reproducible.
- A changed upstream revision or manifest needs fresh user approval; never update it automatically.

### Local checkout binding and preflight

Do not put a machine-specific path in a tracked manifest. Resolve it through either:

```json
{
  "user-approved-stable-id": "/user/local/checkout/path"
}
```

stored in ignored `.cidx/lab/corpora.local.json`, or an explicit CLI `--corpus-path`. Before plan or apply:

1. Resolve and canonicalize the local path without following an escaping symlink.
2. Verify repository origin when available, exact pinned commit, and the manifest's clean-tree rule.
3. Apply `root_subdir`, include/exclude, and language slices.
4. Compute and match the expected tree/content hash.
5. Report mismatches without modifying, cleaning, resetting, fetching, or checking out the repository.

### Dataset contract

The tracked dataset implements `EVALUATION-CONTRACT.md`: answer mode, language/cohorts, OR alternatives inside AND requirement groups, durable file/hash/symbol/span truth, relevance grades, hard negatives, review passes, and calibration/confirmation split. A `mixed` case must intentionally cover cross-language retrieval. A query, label, review, or slice change creates a new digest; no prior query vector or promotion vote is reused.

Confirmation is promotion-capable only when it satisfies the documented dataset/review floor. Smaller data is marked smoke-only. No-answer and hard-negative labels require corpus-wide evidence and a second independent review/pass.

### Lab evaluation metadata

`.cidx/lab/embeddings.db` `evaluation_runs` stores no vectors. It records run/state, corpus ID and manifest fingerprint, upstream pinned commit, verified content hash, repository identity, index generation/manifest SHA-256, dataset fingerprint/count, source-space-storage-serving fingerprints, raw-bank fingerprint/coverage, build version, actual query input-token usage and outcomes, and report path/checksum.

Query text remains only in the dataset; the DB stores a query ID or hash. It has no query-f32 blob column.

### Artifacts

```text
.cidx/lab/evaluations/<run-id>/
  run-manifest.json
  per-query-trace.jsonl
  fts-candidates.jsonl
  dense-segment-candidates.jsonl
  collapsed-parent-candidates.jsonl
  rrf-results.jsonl
  inline-body-packages.jsonl
  per-query-metrics.jsonl
  aggregate-metrics.json
  cohort-language-report.json
  first-loss-report.json
  provider-usage.json
  implementation-audit.json
  promotion-contract.json
  promotion-result.json
  report.md
  artifact-checksums.json
```

Artifacts include sufficient corpus, profile, and raw-document provenance but no query vector or raw document bytes. `rankings.json` uses query IDs rather than copying text. Mark incomplete runs explicitly and never compare them automatically with complete runs.

### Development CLI

```text
cidx dev retrieval evaluate --corpus-manifest <path> --dataset <path>
cidx dev retrieval evaluate --corpus-manifest <path> --corpus-path <approved-local-path> --dataset <path>
cidx dev retrieval evaluate --corpus-manifest <path> --dataset <path> --apply
```

- Default execution performs corpus/config/DB/raw coverage/dataset validation and estimates query tokens/cost; it makes no network call and does not index or embed.
- An existing approved local binding may replace `--corpus-path`; the tracked manifest itself never holds the path.
- `--apply` is a separate explicit approval for paid query embedding. Prior corpus approval does not imply paid-call approval.
- If the current profile/key differs, return `PROFILE_RECONCILIATION_REQUIRED` and make no API call.
- If the key matches but current vectors are absent, return `MATERIALIZATION_REQUIRED` and point to development materialize or public embed.
- If the raw bank does not completely cover the verified evaluation manifest, return `RAW_COVERAGE_INCOMPLETE`.
- A command evaluates only current config. It is an unstable development surface, not MCP or public compatibility surface.
- This command never clones, fetches, checks out, cleans, or updates a corpus. Any such action requires a separate user-approved workflow outside this command.

### Sequential comparison workflow

Phase 13 eventually exposes the commands below. Phase 12 itself can call the same application services through a development harness and does not require the final CLI/MCP adapter.

```text
0. user approves corpus manifest and supplies a matching local checkout binding
1. verify corpus commit, clean-tree rule, selected-content hash, license, and slices
2. cidx dev embeddings capture --apply      # one approved 1024-f32 document capture
3. select candidate A in config
4. cidx index                               # reconcile serving key when needed
5. cidx dev embeddings materialize --activate
6. cidx dev retrieval evaluate --corpus-manifest ... --dataset ... --apply
7. select candidate B in config
8. cidx index
9. cidx dev embeddings materialize --activate
10. cidx dev retrieval evaluate --corpus-manifest ... --dataset ... --apply
11. compare complete runs only when their compatibility fingerprints match
```

Target or codec changes make zero additional document API calls. Queries are requested fresh in each evaluation run.

### Variants and observations

Each run evaluates FTS-only, target-f32 vector-only, active-codec vector-only, provider union, FTS+f32 RRF, FTS+active-codec RRF, and both lane-ablation hybrids. Binary and int8 use separate profile runs because one active profile never mixes codecs. All compatible arms use the same ephemeral target query and frozen candidate/collapse/RRF/body policy.

Compute the exact formulas in `EVALUATION-CONTRACT.md`: Hit/Recall/MRR/NDCG, requirement coverage and complete hit, stage survival/loss, first loss, target-f32 top-k/gold retention, missing neighbors, rank displacement, pairwise inversion, tie diagnostics, RRF lane overlap/contribution/rescue/harm, and body-package survival. Keep raw BM25, f32, binary, int8, and RRF scores on separate scales.

Report every metric by Go, TypeScript, TSX, mixed, and critical cohort with denominators and failures. An aggregate is allowed only alongside slices. Numeric noninferiority margins come from repeated cidx calibration baselines and are frozen in `promotion-contract.json` before confirmation; no foreign threshold is copied.

## 7. Configuration and Change Impact

Evaluation reads current model, target dimensions, reducer, normalizer, metric, storage codec, and search candidate/return/RRF/FTS-weight config. `voyage-code-4` is initially the only validated model; `ModelSpec` supplies source 1024 and current target from `{256,512,1024}`.

Do not guess a code-4 token ceiling from another model. Plan with the actual dataset estimate and conservative `embedding.batch` policy, clearly labeled as project policy.

Corpus manifest and dataset paths are development inputs. Artifact location is fixed under `.cidx/lab/evaluations/`. Do not duplicate dimension, reducer, normalizer, metric, or codec in a lab-specific config or CLI override.

| Change | Like-for-like status | Required work |
| --- | --- | --- |
| Target dimensions/codec | Candidate comparison | Reconcile index, materialize, run again |
| Reducer/normalizer/metric | Different spaces; compare only approved combinations | Materialize and rerun |
| Dataset question/answer | Dataset incompatible | New fingerprint and query run |
| Corpus commit/content hash/include slice | Corpus incompatible | New user approval, raw coverage, and series |
| Index manifest | Corpus snapshot incompatible | New raw coverage and series |
| RRF/candidates/FTS weights | Retrieval-policy comparison | New run with explicit difference |

## 8. Ordered Implementation Checklist

1. Define strict portable corpus-manifest loading and canonical manifest fingerprinting.
2. Implement ignored/explicit local binding without writing an absolute path to tracked files.
3. Verify user approval marker, license evidence, origin, pinned commit, cleanliness, selected files, and content hash before any run.
4. Freeze dataset schema, strict loading, required-group/relevance matching, review records, calibration/confirmation assignment, and canonical digest.
5. Verify repository, corpus manifest, index manifest, source profile, and raw coverage.
6. Check current config against active serving profile before query APIs.
7. Implement query-count and token/cost planning with no network access.
8. Connect `--apply` to the Voyage 1024 query request and shared transformer, validating role and response.
9. Make it impossible for a query vector to escape request/run memory.
10. Stream or batch target-f32 document views from the raw bank.
11. Reuse Phase 11 aggregation, tie-break, codec-aware scan, FTS, collapse, and RRF for every standalone, union, hybrid, and ablation arm.
12. Record success, API failure, fallback, missing-answer, stage survival, and first-loss outcomes explicitly.
13. Implement per-language/cohort denominators, required-failure inclusion, aggregate-with-slice-count rules, and incomplete-run policy.
14. Compute paired human-gold metrics and target-f32-versus-active-codec retention, displacement, inversion, missing-neighbor, tie, and determinism diagnostics. Treat binary/int8 cross-run deltas as paired only when query-vector hashes and all other controls match.
15. Write artifacts to a temporary location and atomically publish with a completion marker.
16. Store only vector-free provenance and artifact checksums in `evaluation_runs`.
17. Freeze development plan/apply request and summary contracts for Phase 13.
18. Compare complete runs only when compatibility fingerprints and paired controls allow it.
19. Implement the frozen zero/100% correctness gates and calibrated paired-margin gates from `EVALUATION-CONTRACT.md`.
20. Publish an immutable `scope=core_retrieval` `PROMOTION_EVIDENCE_READY` only when every core gate passes; otherwise publish `NOT_PROMOTION_READY` with failed gates. Do not imply that assistant/host release-candidate gates have run.
21. Hand the reviewed target/codec choice to the config decision log; never apply it automatically.

## 9. Failure, Rollback, Concurrency, and Security

### Failure and rollback

- Corpus approval, license, path, origin, revision, cleanliness, slice, or hash failure stops before indexing, API calls, or DB writes; the tool never repairs the checkout.
- Preflight failure performs no query API call and no runtime write.
- Record an API-failed case and its denominator treatment; never skip it silently.
- Never persist query vectors on success, failure, or cancellation.
- If generation, manifest, corpus content, or active profile changes during a run, mark it incomplete and exclude it from complete comparisons.
- Artifact failure leaves only temporary output and writes neither completion marker nor success row.
- Evaluation changes neither config nor serving vectors, so production rollback is unnecessary.

### Concurrency

- Allow one applied evaluation per repository under a development evaluation lock.
- Hold no SQLite write transaction during query APIs.
- Do not take index/search application mutexes; compare start/end corpus, index manifest, and profile fingerprints to detect change.
- Use the Phase 11 snapshot contract for scans; a long evaluation must not serialize ordinary MCP search.

### Security and licensing

- Require explicit user approval before acquiring or processing third-party source.
- Treat upstream URL as provenance, not download authority.
- Disclose before apply that dataset questions are sent to Voyage AI.
- Never write credentials, query f32, or raw/source vector bytes to artifacts or DBs.
- Do not duplicate query text outside the user-managed dataset.
- Limit errors and diagnostics to path, symbol, and range; exclude complete source bodies.
- Exclude `.cidx/lab/evaluations` and `.cidx/lab/corpora.local.json` from Git and release artifacts.

## 10. Validation Scenarios

- An unapproved manifest or absent license evidence stops before corpus access or API calls.
- A local binding mismatch in commit, dirty state, root slice, or content hash reports the mismatch without changing the checkout.
- A tracked manifest containing an absolute local path is rejected.
- Planning makes zero query calls and reports expected count, tokens, and cost.
- Applied evaluation leaves no query f32 row in either schema.
- f32 and the active binary or int8 variant reuse the same ephemeral target query within a run.
- Runs for targets A and B record the same approved raw-bank and document-key set.
- Document API calls remain zero between codec/dimension candidates.
- Changing a question changes the dataset fingerprint.
- Changing source model prevents reuse of the raw bank.
- Config/materialization mismatch fails before query embedding.
- A concurrent index or corpus change marks the run incomplete.
- Reports expose failed-query denominator treatment.
- Reports expose Go, TypeScript, TSX, and mixed results separately and cannot present an aggregate without slice counts.
- Target-f32 and active-codec ranking entries align by query/document ID within every run; sequential binary/int8 runs declare whether query-vector hashes permit a direct paired delta.
- Low or high latency/hit rate never automatically selects a profile.
- Standalone FTS and dense lane regressions remain visible even when RRF retrieves the correct parent.
- Provider union, collapse, RRF, package, and first-loss requirement coverage is monotonic.
- Target-f32 fidelity and human relevance can disagree without either metric overwriting the other.
- Calibration output cannot vote for promotion, and confirmation cannot modify settings or margins.
- Required failed calls remain in quality, latency, token, and cost denominators.

## 11. Completion Evidence

- Approved portable manifest and separate ignored local-binding example.
- License, commit, clean-tree, slice, and content-hash verification record.
- Dataset schema/fingerprint and a complete run manifest with corpus/raw/profile provenance.
- Per-language/cohort Hit/Recall/MRR/NDCG, requirement coverage, survival, first-loss, and hard-negative results for every standalone/union/hybrid/ablation arm.
- Same-run paired target-f32/current-codec retention, missing-neighbor, displacement, inversion, tie, human-gold, determinism, and codec-integrity evidence, plus clearly classified sequential binary/int8 comparison status.
- RRF lane overlap, contribution, rescue, harm, rank movement, and broken-lane detection evidence.
- Body-package survival, fidelity, density, duplicate, and omission-reason evidence produced by the shared packaging core available at this phase.
- Sequential reports for at least two approved target/codec candidates when the user requests that comparison.
- Per-run query calls/tokens/cost and proof of zero new document calls.
- Inspection proving no query vector in DBs or artifacts.
- Incomplete classification for a concurrent manifest/content change.
- Human decision record for selected config and rationale.
- Frozen `promotion-contract.json`, `scope=core_retrieval` `promotion-result.json`, first-loss report, implementation audit, and artifact checksums.

## 12. Handoff

Write the selected `embedding.target_dimensions`, reducer, normalizer, and metric to the single project config authority only after human review. Keep `storage_codec=binary` unless the user explicitly chooses the supported `int8` alternative after reviewing the comparison. Then run `cidx index` to reconcile serving keys and either development materialize activation or public embed apply to prepare rows before Phase 13 and Phase 14 verification.

If results are inconclusive, record candidates, why they remain undecided, and what additional user-approved corpus or questions are required. Do not invent a default.

## 13. Decision Log

- Open-source corpora are selected and sampled by the user; tooling only consumes explicitly approved pinned manifests and local bindings.
- Tracked manifests are portable and contain no local absolute path; ignored bindings map IDs to local checkouts.
- The initial 1024-f32 document bank is reusable for setup evaluation only, not permanent product workflow.
- Query f32 is never stored, even for fixed questions, and is not a cache foundation.
- Candidates are evaluated sequentially by changing current config; runtime still serves one profile.
- The f32 baseline exists only in development-run memory.
- Source output stays explicitly 1024; only targets 256, 512, and 1024 are candidates.
- Target dimension is chosen after measurements. Storage defaults to binary; int8 requires an explicit user selection after comparison.
- Hit rate and latency are observations, not predeclared gates.
- Evaluation remains an unstable development surface outside MCP/public compatibility.
- There is no weighted total quality score; promotion is the conjunction of applicable hard gates.
- Human relevance and target-f32 codec fidelity are independent references.
- FTS and dense standalone evidence is mandatory around RRF; fusion cannot admit a broken lane.
- HNSW/ANN recall, `ef_search`, graph health, and ANN tuning are excluded.
- Activation follows evaluation and never substitutes for it.
