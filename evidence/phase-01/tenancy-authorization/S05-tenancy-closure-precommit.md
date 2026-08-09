# P01-S05 tenancy closure — pre-commit evidence

- **Evidence ID:** `EVD-P01-S05-TENANCY-CLOSURE`
- **Phase/slice:** `PHASE-01_IDENTITY_ACCESS_TENANCY` / `P01-S05`
- **Source:** `UNCOMMITTED_WORKTREE(base=d9634ecbf054c893a072221eb544074e9ffc1a1c)`
- **Environment:** isolated local Podman compose project on the synthetic reference stack
- **Virtual test time:** fixed UTC clocks in repository tests; no production or real-identity data
- **Reproduce:** `pwsh -NoProfile -File ./scripts/verify-p01-s05.ps1 -Live -ContainerRuntime podman`
- **Observed result:** `PASS`
- **Revalidate by:** 2026-11-09

## Scope proved

The cumulative verifier passed builds, vet, all Go tests, canonical OpenAPI/AsyncAPI lint,
migration-manifest verification and mutation canaries, Bun tests/build, the S02–S04 regression
chain, and the real PostgreSQL S05 suite. The live suite used the generated `atlas_api` and
`atlas_migration` roles against migration version 12 and the deterministic Phase 01 seed chain.

The bounded evidence supports:

- tenant/global-scope table inventory and tenant-leading repository predicates;
- explicit active-organization selection with zero-grace session and CSRF rotation;
- masked, count-free member pagination with authorization before cursor evaluation;
- 72-hour hash-only invitation issuance, closed delegation, recipient-bound bootstrap
  authentication, single-use concurrent acceptance, and Audit-atomic membership/session commit;
- exact strong membership ETags, actor/tenant-bound idempotency, immediate authority-version
  increments, and zero-grace target-session invalidation for direct viewer/operator role changes;
- durable direct viewer/operator removal, identical-request convergence to one mutation plus one
  replay, changed-request conflict, and no authorized stale-tab role mutation after revocation;
- simultaneous role-change-versus-removal serialization: exactly one mutation commits and only its
  immutable replay record exists;
- Audit-outage rollback for tenant switch, invitation, role-change, and removal mutations;
- canonical Unicode NFKC/case-fold projection for repository-owned organization fixtures plus
  real PostgreSQL rejection of normalized-name and UTS39-skeleton collisions; and
- bounded role/removal telemetry labels, mass-membership-mutation alert metadata, and the linked
  compromise/revocation runbook.

## Requirement and risk links

- **Requirements:** `IAM-010..015`; organization-bound evidence for `IAM-020..026` remains
  cumulative input to S06 rather than a claim of phase-wide authorization closure.
- **Threats:** `THR-005`, `THR-006`, `THR-038`, `THR-041`, `THR-044`, `THR-057`, `THR-058`.
- **Adversarial tests:** `ADV-IAM-001`, `ADV-IAM-007`, `ADV-IAM-008`, and phase tests-most-agents-skip
  1, 3, and 11 at the explicitly implemented boundaries.

## Failure posture

All sensitive S05 mutations recheck session, tenant, role, permission, membership/resource state,
version, and replay state inside serializable PostgreSQL transactions. Audit persistence failure
rolls back the product mutation. Response-loss retries return durable state without repeating an
effect. Redis and browser state do not authorize a tenant operation or restore a revoked grant.

## Sanitization

This report contains only public source identities, closed synthetic operation/role names,
bounded PASS statements, migration version/checksum identity, requirement/threat/test identifiers,
and control semantics. It excludes database URLs and credentials, cookies, CSRF and idempotency
values, invitation material, OIDC state/nonce/PKCE/code/token data, principal/session/member/
decision identifiers, raw SQL errors, WAL/data pages, network addresses, and real identity data.

## Limitations

- Organization creation is not a Phase 01 HTTP operation. The NFKC/case-fold projector validates
  repository-owned fixtures and PostgreSQL enforces both stored projections; a future creation
  owner must supply and test the complete UTS39 skeleton generator before accepting external names.
- Administrator role transitions remain fail-closed pending S07 typed maker-checker execution.
  Administrator removal additionally remains fail-closed pending an exact fresh-step-up and
  last-administrator policy.
- S05 evidence is same-host synthetic evidence, not a real IdP, production deployment, independent
  review, constant-time proof, or financial behavior claim.
- Product frontend and cross-tab durable-server recovery journeys remain S09 closure work.
