# Phase 00 int8 product-profile reconciliation — Revision 4

- Date: 2026-08-17
- Authority: explicit user product decision
- Product codec: cidx-owned `int8` only
- Product serving dimensions: 1024 default, 512 explicit compact option
- Retired evidence profiles: every Binary and 256-dimensional profile

The five-profile chi/RHF comparison established the decision input. The user
then explicitly removed Binary and 256 from the service while requiring their
existing evidence to remain intact and any future use to require confirmation.

The canonical and evaluation contracts now point to
[`RETIRED-VECTOR-PROFILES.md`](../../RETIRED-VECTOR-PROFILES.md). Ordinary
config, CLI, storage publication, runtime scan, evaluation, packaging smoke,
and fixtures must use int8 and only 1024 or 512 dimensions. Ordinary tests use
1024. Document-role 1024-f32 is preserved in the product source bank for
provider-free target rematerialization; exhaustive query/reference f32 remains
non-serving evidence.

Existing historical reports and ignored local databases are not rewritten or
deleted. Current source removes Binary/256 from settings, materialization,
search, and evaluation instead of retaining an internal access path. A
historical config or database is incompatible, and current work requires a
new int8 state plus local rematerialization or separately approved embedding.

Affected phase boundaries requiring implementation or revalidation are 02,
05, 07, 09, 10, 11, 13, and 14. Phase 00's semantic decision is complete; the
ledger keeps downstream phases gated until their own changes and checks land.
