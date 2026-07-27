# Session compromise and revocation

Scope: Phase 01 Atlas opaque application sessions backed by PostgreSQL. This runbook does not
cover future financial commands, real identity data, or production incident response.

## Triage

1. Preserve the reported application session ID, request/correlation IDs, bounded timestamps,
   principal population, and safe client label. Never collect the cookie verifier, CSRF token,
   OIDC code, PKCE value, upstream token, or browser storage dump containing secrets.
2. Review append-only Audit facts and the `phase-01-identity-session` dashboard. Distinguish one
   session, all sessions for one principal, and a broader provider incident.
3. Treat a privilege, tenant, or membership authorization-version change as immediate invalidation.
   Redis or browser state must never override PostgreSQL authority.

## Containment

1. For self-service containment, revoke the selected session or use revoke-all with a unique
   idempotency key. Include the current session when compromise is suspected.
2. A current-session revocation must clear the host-only cookie. Reusing the prior cookie must
   return `401`; retries with the same request/key must not duplicate the durable Audit effect.
3. Administrative revocation uses
   `POST /v1/security/sessions/{session_id}/revocations` only from an active workforce
   `platform_administrator` session with `security_review` purpose and a fresh
   phishing-resistant step-up bound to `identity.session.admin_revoke`. Use a new idempotency key
   and one closed reason code. Do not reuse the self-owned route, edit session rows directly, or
   broaden the API database role.
4. If the database is unavailable, fail closed. Do not move revocation truth to Redis or process
   memory.

## Recovery and verification

Run `pwsh -NoProfile -File ./scripts/verify-p01-s04.ps1 -Live`. Confirm revoke-one, concurrent
revoke-all and administrator-revocation replay, changed-request conflict, action-bound step-up
denial, atomic Audit persistence and outage rollback, old-cookie rejection, and restored revoked
authority. Preserve new evidence rather than overwriting historical reports.

Known limitation: Phase 01 local evidence is synthetic and same-host; no real account recovery,
provider-wide logout, managed secret custody, or production incident-response claim is made.
