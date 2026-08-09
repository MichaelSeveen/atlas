# Authorization incident and invalidation

Owner: Identity on-call. Scope: Phase 01 server-side Identity authorization, tenant concealment,
purpose-bound masking, PostgreSQL authority versions, and application-session invalidation. This
runbook does not authorize direct database edits, a Redis authorization cache, or any financial
operation.

## Detection and evidence safety

- `P01-S06-ALR-001` detects a bounded aggregate spike in rejected organization-member requests
  returning the uniform concealed `404` response. `P01-S05-ALR-001` detects a burst of successful
  membership role changes or removals. Review the corresponding operation-count and duration
  panels without adding tenant, actor, user, request, decision, email, or resource identifiers to
  metric labels.
- Preserve source revision, migration and policy-seed digests, UTC interval, affected route and
  environment, bounded response status/code, and the decision IDs already returned to the caller
  or recorded in append-only Audit. Do not collect cookies, CSRF/idempotency/invitation material,
  raw or canonical email, SQL text, browser tokens, or real identity data.
- Distinguish malformed requests, concealed cross-tenant identifiers, missing permission or
  purpose, stale authorization/session version, inactive resource state, and dependency failure.
  External errors must remain generic; policy internals belong only in bounded source-controlled
  tests and safe Audit reason codes.

## Containment

1. Disable the affected route or apply a narrower versioned policy if exploitation is plausible.
   Never fail open, synthesize a grant, restore a removed role, or move authoritative decisions to
   Redis, browser state, process memory, logs, or an unratified event.
2. For suspected role or membership compromise, use the supported session and membership
   revocation paths in `SESSION_COMPROMISE_AND_REVOCATION.md`. Administrator mutations that are
   not yet backed by their required maker-checker or last-administrator policy remain fail-closed.
3. Confirm that the canonical permission/role/purpose/field catalogue matches the seeded
   PostgreSQL bindings exactly. An unknown or partial catalogue entry must deny; do not edit a
   released migration or seed to restore service.
4. Preserve append-only Audit facts and decision linkage. If required Audit persistence is
   unavailable, follow `AUDIT_PERSISTENCE_UNAVAILABLE.md` and deny the privileged disclosure or
   mutation rather than committing without evidence.

## Invalidation and recovery

1. A committed role, restriction, membership, or session-assurance change must advance the
   authoritative version and revoke or rotate affected application sessions in the same bounded
   protocol. Every API instance rechecks PostgreSQL; no invalidation message or cache flush is
   required in Phase 01.
2. Verify that an old cookie/session is rejected on two independent API pools and that a replacement
   session receives only the current role's field mask. Confirm cross-tenant known and absent IDs
   retain the same safe status/body/page shape and bounded timing behavior.
3. Run
   `pwsh -NoProfile -File ./scripts/verify-p01-s06.ps1 -Live -ContainerRuntime podman` and confirm
   the authorization matrix, unknown-entry mutation, authorization-before-cursor ordering,
   purpose-bound masked field, Audit rollback, interleaved concealment timing, and multi-instance
   invalidation checks pass.
4. Forward-fix catalogue or application defects additively with an explicit source revision,
   regression test, Audit/traceability update, and fresh sanitized evidence. Re-enable a route only
   after the exact affected revision passes the cumulative verifier.

Known limitation: evidence is synthetic and same-host. The current Phase 01 surface has one bounded
member-list operation and no search, suggestion, or autocomplete operation; their absence is not a
claim about future product behavior. No deployed pager, SIEM, production identity provider, real
identity data, or independent incident-response certification is claimed.
