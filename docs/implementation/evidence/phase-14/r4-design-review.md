# Revision 4 Final Implementation-to-Design Review

- Review target: clean implementation commit `30748c1`.
- Reviewer: `/root/r4_final_design_review` (independent terra/high).
- Scope: canonical Revision 4 implementation and corpus-independent phase
  boundaries. Package artifacts, assistant-use runs, and promotion evidence
  are explicitly outside the accepted evidence because their prerequisites do
  not exist yet.
- Result: **no P1 or P2 implementation findings**. One P3 evidence-ledger
  staleness finding was corrected by the commit that records this review.

## Requirement mapping

| Revision 4 boundary | Implementation/evidence conclusion |
| --- | --- |
| 1 MiB source ceiling, no chunk cap, 1 KiB AST-aware segment target | Resolved configuration, Go/TS/TSX chunking, whole semantic-parent FTS, AST-boundary segment packing, and oversize-unit behavior agree with the canonical contract. |
| Synchronous 128-input/256 KiB/four-request embedding policy | Shared executor, document orchestration, and query adapter consume one validated policy with a 30-second attempt timeout and cancellation-aware 10/20/30-second transient retry waits. |
| 1024 source dimensions and project serving dimensions | Source/profile fingerprints, shared transform, binary/int8 storage, one active serving profile, and query/document parity use `serving_dimensions` 256/512/1024 without scope ambiguity. |
| SQLite authority and free local FTS | Generation publication, profile reproof, lexical snapshots, and startup FTS5/WAL checks retain SQLite as the only persistent serving authority and require no provider or key. |
| Four-tool MCP and byte-bounded bodies | The stable registry remains exactly `status`, `search`, `read_span`, and `reindex`; inline byte budgets do not change rank/identity; `read_span` has no line cap and fails all-or-nothing on an oversized response. |
| Provider-free atomic initialization | Git-root discovery, staged owner-only config, exclusive DB claim, identity-gated cleanup, and no-replace publication preserve Phase 13 failure and concurrent-writer safety. |
| Runtime/package/host boundary | Build/runtime JSON reports exact target/dependencies/schema/FTS5/WAL/grammars; the darwin/arm64 scripts fail closed on missing provenance, notices, checksum, capability, or owner license; Codex documentation is project-scoped and never embeds a secret value. |
| Evaluation and promotion isolation | Provider attempts remain separate from logical query denominators; lab state stays out of production; no synthetic/core/host result is promoted beyond its supported scope. |

## Independent checks

The reviewer ran these read-only checks with no provider, corpus, model, or
paid action:

```text
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 ./...
env -u VOYAGE_API_KEY GOPROXY=off go vet ./...
env -u VOYAGE_API_KEY GOPROXY=off go build ./...
bash -n scripts/package-local.sh scripts/verify-local-release.sh
git diff --check
```

All passed, and the worktree was clean at the reviewed commit.

## Corrected documentation finding

The Revision 4 supersession index stopped at Phase 13, and the Phase 14
working evidence still described its implementation commit as pending. Those
records now identify `30748c1` as the accepted corpus-independent Phase 14
checkpoint. No implementation change was required.

## Evidence not established

This review does not establish a package artifact, another platform or host,
signing/notarization, assistant usefulness, `core_retrieval`, or
`release_candidate` status. The exact remaining prerequisites are:

- an owner-selected cidx project `LICENSE`, followed by archive/checksum and
  unpacked offline verifier evidence;
- user-selected corpus manifests/bindings, reviewed labels, and compatible raw
  document coverage;
- an immutable Phase 12 `scope=core_retrieval` result;
- frozen assistant tasks, model/prompt/tool/budget controls, and all three
  comparison arms; and
- separate approval for paid hybrid-query embedding.

Once the license exists, only the package-artifact and host-parse outputs need
an artifact-focused recheck before updating this evidence. Official evaluation
and release-candidate work must still follow its separate approval gates.
