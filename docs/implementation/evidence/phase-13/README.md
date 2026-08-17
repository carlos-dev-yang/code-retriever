# Phase 13 CLI and MCP Evidence

The current 2026-08-17 int8-only acceptance is recorded in
[`int8-only-cli-mcp-reconciliation.md`](int8-only-cli-mcp-reconciliation.md).
The records below remain historical evidence and do not restore their retired
profile options.

- State: historical mixed pre-R4 checkpoint in `b3a6cb1`; implementation is not accepted as Revision 4 complete. The separately owned local-only retrieval-evaluation adapter is implemented.
- Date: 2026-08-15.

The narrow Revision 4 implementation checkpoint from `6797544` is recorded
separately in [revision-4.md](revision-4.md). It was later accepted after
independent review and main commit-boundary validation; this historical record
is preserved unchanged as evidence of the earlier mixed checkpoint.

## Implemented and validated, except `init` defaults

- The production bootstrap opens only the configured root and production SQLite store. It conditionally creates a bounded Voyage client from `VOYAGE_API_KEY`; FTS startup needs neither key nor network. The serve/search/index runtime path imports no lab package.
- MCP registers only `status`, `search`, `read_span`, and `reindex`. The search adapter clamps inline bytes before delegating to the Phase 11 packager, and does not reproduce ranking or body-allocation logic.
- Status takes one short production snapshot, reports segment coverage separately from distinct canonical-input ready/pending/failed state, and keeps latest terminal failure behind ready precedence. Reindex uses the resulting read-only status view only to label reused/pending embeddings; it performs no provider call or extra publication.
- Dry-run reindex now derives planned distinct embedding keys from the prepared Phase 05 payload and current valid cache state rather than relabeling the active generation. The focused new-file fixture proves planned pending inputs increase while the active generation and persisted file set do not change.
- Desired-profile dry classification is one pinned read transaction across all batches and validates the desired profile directly, rather than relying on active metadata. Fixtures cover a profile-change reconciliation that counts only the replacement key and an inactive desired-profile cache row reused without an active segment reference.
- `read_span` checks configured source-language eligibility, repository-relative paths, lowercase hash syntax, UTF-8, observed symlinks, hashes, and ranges from one bounded read; it returns complete line slices only.
- The stdio server uses a named bounded worker queue: it continues receiving cancellation frames under saturation, rejects duplicate request IDs, serializes newline-delimited JSON frames, treats a short frame write as terminal, and closes a read-closable stdin when its outer context is cancelled. Lifecycle tests cover pre-initialize rejection, atomic initialize ownership, initialized notification, version negotiation, cancellation, saturation, out-of-order completion, and clean EOF.
- MCP request IDs accept only strings or integral numbers. Escaped-equivalent strings and equivalent integers share one canonical in-flight key for cancellation and active-collision handling; completed IDs are not retained, keeping serving memory bounded. Valid non-object JSON frames are `INVALID_REQUEST` rather than parse failures; notification methods with an ID are rejected, and initialize/tool-call `_meta` plus tools-list cursor metadata are accepted only as JSON objects under strict schemas.
- An unchanged dry-run computes the current manifest and desired-profile embedding reuse/pending plan without writing. An applied no-op keeps the current manifest and MCP omits `activated_generation` when no generation was published.
- Stable CLI JSON responses use explicit snake_case fields for status, index, and public embedding plan/result payloads; private paid authorization remains unexported.
- A real `cidx serve --root` temporary-repository smoke test performs `initialize`, `notifications/initialized`, and `tools/list`, observes exactly the four public tools, exits cleanly at EOF, and proves no `.cidx/lab` directory is created. It is distinct from the narrower production-bootstrap test.
- The MCP search adapter passes only its clamped effective budget into the Phase 11 core. The existing focused `TestBodyBudgetDoesNotChangeRankingOrInventFTSExcerpt` was rerun against the same adapter dependency set and proves zero/small/full body budgets preserve IDs, order, and count.
- Public `embed --apply` and unstable document capture use explicit `--apply` plus `VOYAGE_API_KEY`; no external-provider invocation was made during this implementation. Unstable materialization wires the existing local Phase 09 application use case. `cidx dev retrieval evaluate` now strictly validates the portable manifest, local binding/checkout, dataset truth, indexed source slice, active profile, and complete raw coverage before any provider action. Its structured plan reports query count, the conservative local token upper bound, and that a dated cost estimate is not frozen. Planning checks corpus inputs before application opening, uses `app.OpenLocal` (which neither reads `VOYAGE_API_KEY` nor creates a provider client), and uses only an existing read-only lab DB, so it does not create/migrate lab state. Applied execution uses one request-local query f32 per case, reuses the Phase 11 snapshot/collapse/RRF/body-packager for all eight Phase 12 arms, retains typed failed-arm denominators, and discards query/session caches after the final arm. Focused tests use only an in-memory fake client; official evaluation evidence remains gated on the user-approved corpus inputs and paid-query authorization.
- Applied execution now publishes a checksummed, vector-free Phase 12 execution artifact and a v4 `evaluation_runs` provenance reference. The completion marker explicitly records that it is not promotion evidence; official `CorePromotionEvidence` remains unavailable until the user supplies the approved confirmation corpus/labels and frozen promotion contract. Only client-returned query failures are retained as operational denominator observations; local snapshot, raw, validation, transform, and packaging defects abort the run.

## Revision 4 supersession

`cidx init` is held behind `config.DefaultRaw`, and the current factory returns `DEFAULT_CONFIG_VALUES_PENDING_DECISION`. Revision 4 and the user's confirmed contract now resolve those operational defaults: source 1 MiB, target segment 1 KiB, synchronous 128-input/256-KiB groups, concurrency 4, 30-second timeout, three 10/20/30-second retries, 64-KiB inline default, and independent 1-MiB source/inline ceilings. The remaining work is implementation and revalidation, not a product decision. Official Phase 12 evidence remains separately blocked on user-selected manifests/bindings, reviewed data, raw coverage, and paid-query approval.

The package-boundary decision is now implemented: lab-backed capture and materialization use cases live in `internal/devapp`, while production bootstrap remains in `internal/app`. `go list -deps ./internal/app ./internal/mcp ./internal/search ./internal/store | rg 'cidx/internal/lab'` produced no output. The one `cidx` binary still statically includes development code through its `dev` namespace, which is explicitly allowed; the `serve` runtime assembly has no lab import or lab-store operation.

## Checks run

```text
gofmt -w cmd/cidx internal/app internal/cli internal/config internal/devapp internal/devlab internal/index internal/mcp internal/search internal/store
go test -count=1 -race ./cmd/cidx ./internal/config ./internal/cli ./internal/devapp ./internal/devlab ./internal/app ./internal/mcp ./internal/index ./internal/store ./internal/search
go vet ./cmd/cidx ./internal/config ./internal/cli ./internal/devapp ./internal/devlab ./internal/app ./internal/mcp ./internal/index ./internal/store ./internal/search
go build -o /tmp/cidx-phase13 ./cmd/cidx
go build ./internal/cli ./internal/devapp ./internal/devlab ./internal/app ./internal/mcp
go list -deps ./internal/app ./internal/mcp ./internal/search ./internal/store | rg 'cidx/internal/(lab|eval)'
git diff --check
```

All listed checks passed; the dependency query produced no output. Focused fixtures cover short stdio writes; pre-initialize rejection; lifecycle version negotiation, atomic/repeated initialize handling and initialized notification; cancellation, saturation, duplicate IDs, out-of-order responses, newline-delimited stdout purity, and clean EOF; a real serve/no-lab handshake; complete multi-line read-span slicing, traversal rejection, symlink rejection, bounded reads, and exact Git-runtime dirty exclusions; the Phase 11 rank-invariant body budget; dry-run no-publish behavior; canonical/profile-change planning; and inactive desired-profile cache reuse. No user-workspace or corpus index operation, provider/API-key/network request, paid embedding, or evaluation execution was performed. Synthetic temporary-repository index and local materialization operations ran inside focused tests. The local root build artifact was removed; validation builds use an external temporary output path.

## Follow-up checks

```text
gofmt -w internal/mcp/server_test.go internal/cli/cli_test.go
go test -count=1 -race ./internal/mcp ./internal/index ./internal/cli ./internal/app ./internal/config ./internal/profile
go vet ./internal/mcp ./internal/index ./internal/cli ./internal/app ./internal/config ./internal/profile
go build ./internal/mcp ./internal/index ./internal/cli ./internal/app ./internal/config ./internal/profile
go list -deps ./internal/app ./internal/mcp ./internal/search ./internal/store | rg 'cidx/internal/(lab|eval)'
git diff --check
```

All follow-up checks passed; the dependency query again produced no output. Added focused fixtures cover canonical active-ID collisions and completed-ID reuse, invalid ID rejection, `_meta`/cursor acceptance, invalid non-object frames, notification request rejection, no-op reindex wiring, unchanged dry-run manifests and embedding plans, configured source eligibility/UTF-8/lowercase-hash read-span rejection, case-insensitive source extensions, bounded index reads, lab worktree dirtiness, typed span-error payload preservation, exact integer parsing, named server-busy code, and stable CLI snake_case encoding.

## Remaining Revision 4 work

Complete `--serving-dim`, the central default factory and config writer, line-cap-free `read_span`, and the reconciled request/profile contracts after their owning phases finish. Official Phase 12 usefulness/promotion evidence remains externally gated and is not claimed here.
