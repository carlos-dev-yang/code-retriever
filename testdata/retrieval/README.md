# Provider-free lexical smoke inputs

These portable manifests name the two public repositories explicitly selected
and authorized by the user. They contain no checkout path or source body.

Local checkout bindings live only in each checkout's ignored
`.cidx/lab/corpora.local.json`. The verifier checks origin, pinned commit,
clean status, root tree, the manifest's selected-content SHA-256, root license
evidence, language slices, and production-index file parity before it opens
the lexical runner.

`go-chi-chi-v5.3.1` selects all 78 Go files (`<= 58,795` bytes each).
`react-hook-form-v7.85.0` selects the 237 TypeScript/TSX files below `src`
(`<= 142,806` bytes each); its full local checkout has an ignored
`.cidxignore` that excludes the exact outside-`src` TS/TSX roots `app`,
`e2e`, `examples`, `scripts`, and `playwright.config.ts` so the production
index has identical source coverage.

Draft smoke datasets are deliberately `review.state=draft`, identify the
machine draft author and `user-review-pending`, and are not frozen,
official, promotion-capable, or suitable for tuning. Do not authorize a paid
document or query embedding operation from these files.
