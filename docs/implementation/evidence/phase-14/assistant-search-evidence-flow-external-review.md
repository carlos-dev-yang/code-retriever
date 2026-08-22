# Assistant Search-to-Evidence Flow External Review

- Date: 2026-08-22
- Phase: 14 design review
- Scope: product role, search-to-evidence interaction, session accounting, and
  the next assistant-availability evaluation
- Reviewers: existing ChatGPT and Grok side-panel conversations
- Provider/API action: none
- Paid cidx operation: none
- Product code or MCP schema change: none
- Historical boundary: V4-V6 remains closed; this review does not authorize or
  define V7

## Review packet

Both reviewers received the same complete English design plus the same review
instructions. They were asked to check host freedom, compact candidate versus
sufficient evidence separation, total-journey rather than single-response
optimization, four-tool/SQLite authority, session-local accounting, metric
separation, the 30-question/60-turn design, and the explicit Voyage paid-query
boundary.

The initial transmitted design had:

```text
SHA-256  4dcb5d8f33d1dd0dbf245026b37a6bc453527131106c8f01af2b757081396756
bytes    22533
```

The final reviewed repository document is:

```text
path     docs/implementation/ASSISTANT-SEARCH-EVIDENCE-FLOW-DESIGN.md
SHA-256  9824a12d2d2ebc3fa905c83be9c49cbe146f48b112be25fdbbbc399242cbe972
bytes    29496
```

## Initial ChatGPT review

Disposition: `ACCEPT_WITH_CORRECTIONS`.

ChatGPT found the architecture coherent and found no fundamental conflict with
SQLite authority, the four-tool MCP, locator-only search, `read_span`, or the
paid-query boundary. It accepted the current nine locator fields and deferred
signature/neighbor metadata until a measured first loss.

Required corrections were:

1. Restate fresh source/context, private state, identical ordinary tools,
   hidden grading artifacts, frozen binary/interface/config identities, and
   consecutive paired execution.
2. Separate a passive first-run trace from active duplicate suppression or
   claim guidance. Otherwise the treatment would measure cidx plus a new
   orchestration layer rather than cidx availability.
3. Author and freeze questions source-first without consulting cidx output or
   prior assistant behavior.
4. Give interpretation-sensitive navigation/evidence metrics explicit
   numerators, denominators, event sources, and mechanical/blind authorities.
5. Enforce the FTS-only/no-provider condition through execution configuration,
   missing credentials, and requested/effective-mode records.

Its smallest material correction was the passive-versus-active ledger split.

## Initial Grok review

Disposition: `ACCEPT_WITH_CORRECTIONS`.

Grok likewise found no high-severity architecture finding and accepted the
nine-field candidate locator plus exact `read_span` for the first run. It
required:

1. An explicit statement that the session ledger never persists, writes
   SQLite, affects ranking, or changes MCP behavior.
2. Pure-capability tool descriptions with usage preference kept out of the
   model-visible interface.
3. Unsupported material claims as a first-class count/rate independent of
   required-group coverage.
4. A first-run freeze of configured default `k=10` and absolute maximum 20,
   with later default changes treated as separate experiments.
5. Post-turn `observed | derived | unresolved` classification for every
   material claim.

## Reconciled corrections

The complete design was revised to incorporate both sets of findings:

- the live first-run ledger is passive, non-model-facing, absent from product,
  SQLite, and server session state, and unable to influence server or assistant
  behavior; a truth-free, body-free trace snapshot may be retained only in
  ignored local evaluation artifacts for reproducibility;
- claim support and accepted-evidence annotations are joined only after blind
  grading;
- execution isolation and complete identity binding are explicit;
- question construction is source-first and pre-result;
- tool descriptions state capabilities only and are digest-bound;
- the provider-free boundary is executable, credential-free, and observable;
- first-run result-depth configuration is fixed without rewriting a caller's
  valid `k` choice;
- metric sets, formulas, denominators, and authorities are explicit; and
- unsupported claims remain independent of requirement coverage.

## Final review

The same revised complete design was returned to both reviewers with an
accept-or-block instruction.

### ChatGPT

Final disposition: `FINAL_ACCEPT`.

Nonblocking cautions:

- keep claim-state classification post-turn rather than live guidance;
- treat the defined navigation false-lead rate as a proxy, not proof that every
  uncited/non-gold read was semantically useless; and
- keep cidx-user-only and question-shape slices descriptive because adoption is
  self-selected and this series is calibration, not promotion.

### Grok

Final disposition: `FINAL_ACCEPT`.

Nonblocking cautions:

- populate selection/usefulness/claim annotations only after grading;
- keep mechanical first-sufficient-evidence time secondary to final blind claim
  support; and
- freeze any future active duplicate or claim-guidance layer as a separate
  intervention.

## Final disposition

The design is accepted for implementation planning with no unresolved external
review blocker. The reviewers agree on these immediate boundaries:

- do not add signature, neighbor, dependency metadata, another tool, or a
  server-side session store before the initial measured run shows that need;
- do not force cidx use or label it secondary-only;
- keep the initial experiment FTS-only and provider-free;
- evaluate all 30 treatment turns by intent to treat, including no-use; and
- do not interpret this calibration series as retrieval or release promotion.

This review authorizes documentation reconciliation only. Product code,
test-code creation, question-set construction, scored runs, and any paid
hybrid/query embedding remain subsequent phase work under their existing
approval and evidence rules.

## Post-review document provenance

The `FINAL_ACCEPT` dispositions above bind the reviewed design digest
`9824a12d2d2ebc3fa905c83be9c49cbe146f48b112be25fdbbbc399242cbe972`
and 29,496 bytes. After that review, the owner separately opened Step 2 and the
repository document received implementation-status, evidence-link, and
ignored-artifact trace-retention clarifications. Its current post-Step-2 digest
is:

```text
SHA-256  d4ebd3fa730e51fd4f1322b5803e86ad570e91a6037781820c397f1c06d68a5b
bytes    31024
```

Those later bytes are not represented as externally reviewed. They record the
owner's subsequent authorization and completed implementation checkpoint and
do not change the reviewed host-choice, four-tool, locator/read, passive-trace,
metric-separation, FTS-only, or provider boundary. Step 2 implementation
evidence is maintained separately.
