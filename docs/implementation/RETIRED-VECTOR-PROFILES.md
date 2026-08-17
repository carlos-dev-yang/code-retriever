# Retired Vector Profiles and Evidence Boundary

- Status: normative v1 product boundary
- Date: 2026-08-17
- Supersedes: the earlier v1 product support for cidx sign-binary storage and
  256 serving dimensions

## Product profiles

Production cidx supports one cidx-owned storage codec, `int8`, and exactly two
serving dimensions:

| Profile | Product status | Purpose |
| --- | --- | --- |
| `1024/int8` | default | ordinary development, evaluation, initialization, embedding, and serving |
| `512/int8` | supported explicit option | compact target rematerialized from the same 1024-f32 source bank |
| `1024/f32` | evaluation reference only | exhaustive evidence reference; never production storage |
| any `binary` profile | retired evidence only | historical comparison and preserved artifacts |
| any 256-dimensional profile | retired evidence only | historical comparison and preserved artifacts |

The public CLI exposes `--serving-dim <1024|512>` and no codec selector.
Configuration and profile fingerprints still record the code-owned int8 codec
ID so database compatibility remains explicit. The user cannot select another
codec. Provider requests remain source-1024 float and local code reduces to the
selected serving dimension before int8 quantization.

All ordinary implementation and evaluation fixtures use `1024/int8`. A
`512/int8` profile is an explicit supported compact option. Document-role
1024-f32 is durably stored in the product source bank so either target can be
materialized locally. Query f32 remains request-local; source f32 is never a
search fallback or active serving representation.

## Retired evidence boundary

Existing Binary/256 reports, checksum manifests, ignored SQLite states, and
source-review packets remain preserved as historical evidence. They are not
rewritten to look like int8 product results and are not deleted merely because
the profile is retired.

Binary and 256 are removed from config resolution, `init`, materialization,
embedding publication, runtime vector scan, package smoke verification, and
current-profile evaluation. They are not hidden flags and are not accepted
legacy values. A retired profile must never become active through a default,
fallback, internal codec branch, development command, or direct store API.

Any future reproduction that would create a new Binary or 256 artifact
requires all of the following before execution:

1. a new explicit user approval naming the retired profile and bounded purpose;
2. a separately designed development-only evidence tool that is not part of
   the current product executable or packages;
3. a recorded authorization reference and immutable output location; and
4. an evidence note that the profile is unsupported by the product and cannot
   vote for activation without a new product-contract decision.

The current source tree does not retain an executable Binary codec or
Binary/256 comparison path merely for convenience. Historical algorithm and
format details live in immutable reports and version control. If an old
schema migration must recognize a historical literal to identify an
incompatible database, that literal is migration metadata only: no current
writer, decoder, scorer, or evaluator may accept it.

## Existing repository migration behavior

- A historical config containing `binary` or dimension 256 is rejected as an
  unsupported retired profile; neither value is part of the current config
  model or CLI.
- cidx never deletes or rewrites the old config or database automatically.
- The owner creates or selects a current 512/int8 or 1024/int8 state. Its
  storage fingerprint differs from retired states and requires local
  rematerialization from a compatible product source f32 bank or a separately approved
  document embedding.
- AST, FTS, source chunks, and canonical input identities remain reusable when
  their own profiles are compatible.
- Label-only replay of immutable historical rankings remains provider-free.

## Evaluation rule

Current calibration and confirmation use `1024/int8` unless a frozen plan
explicitly declares the supported `512/int8` arm. Human relevance remains the
product-quality reference. When codec or dimensional fidelity evidence is
needed, use exhaustive f32 as a separate reference and never combine ranks
across profiles with RRF.
