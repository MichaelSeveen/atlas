# P01-S04 administrator session-revocation pre-commit verification

## Evidence identity

- **Evidence ID:** `EVD-P01-S04-ADMIN-SESSION-REVOCATION`
- **Phase/slice:** `PHASE-01_IDENTITY_ACCESS_TENANCY` / `P01-S04`
- **Source:** `UNCOMMITTED_WORKTREE(base=744d1e1f1ede231b00897a0ee10d3342ece15495)`
- **Observed date:** 2026-07-27
- **Environment:** Windows host, Go 1.25.12, disposable real PostgreSQL 18.4 through Podman
- **Requirements/threats:** `IAM-004`, `IAM-006`, `IAM-025`, `FND-011`, `FND-064`,
  `THR-006`, `THR-007`, `THR-018`, `THR-037`, `THR-058`

## Contract and authorization boundary

ADR 0015 resolves `P01-D15` additively with
`POST /v1/security/sessions/{session_id}/revocations`. The existing
`DELETE /v1/sessions/{session_id}` remains current-principal-owned.

The command accepts only:

- an active workforce application session;
- `identity.sessions.revoke_admin`;
- purpose `security_review`;
- one of `compromised_session`, `suspected_account_takeover`, or
  `workforce_security_response`;
- phishing-resistant assurance and a step-up no more than five minutes old, bound to the exact
  `identity.session.admin_revoke` action;
- a CSRF token and 16–128 character idempotency key.

The PostgreSQL transaction re-locks the actor session and authoritative workforce role/version,
then checks permission and step-up evidence before target lookup. Missing targets are concealed
until that authorization succeeds. The revocation, stable decision ID, idempotency result, and
append-only Audit fact commit together. Audit failure rolls the target mutation back.

## Schema and failure model

Released migration 000008 adds nullable paired `step_up_action` /
`step_up_verified_at` evidence to sessions and creates a global hash-only administrator-revocation
replay table. Existing sessions have no action binding and fail closed. API access is
`SELECT, INSERT`; update/delete remain denied. Worker and reporting roles cannot read the table.
The migration is forward-only and is included in empty/previous/current, role, backup/WAL, and
isolated-PITR verification.

## Reproduction and observed result

These focused commands passed:

```text
go test ./internal/identity/... ./cmd/api/internal/server ./tests/contract ./internal/platform/migration -count=1
pwsh -NoProfile -File ./scripts/p01-s04-session-repository.ps1 -ContainerRuntime podman
```

The unit/HTTP/contract suites proved closed request decoding, CSRF/idempotency transport,
authorization-decision headers, purpose/reason rejection, action propagation from the OIDC
transaction into the rotated session, route inventory, telemetry labels, and additive OpenAPI
compatibility.

The disposable real-PostgreSQL test applied all eight migrations and the canonical seed, then
proved:

- two concurrent identical administrator commands produce exactly one original execution and one
  replay with the same decision ID;
- changing the request digest under the same idempotency key returns conflict;
- the target verifier immediately observes `SESSION_REVOKED`;
- a phishing-resistant workforce session without the exact fresh action binding is denied and its
  replay returns the same step-up-required decision ID;
- permission/assurance/authority decisions occur before target lookup;
- an injected Audit-recorder outage returns service unavailable, stores no decision/replay fact,
  and leaves the target session active;
- the successful execution and denied high-risk attempt each produce exactly one Audit fact.

The script ended with:

```text
database_migrations=PASS target=atlas_p01_s04_session_test
phase01_identity_seed=PASS target=atlas_p01_s04_session_test
ok github.com/MichaelSeveen/atlas/internal/identity/persistence
p01_s04_session_test_database=drop_PASS
p01_s04_real_postgres_session_repository=PASS
```

## Sanitization and limitations

Only public versions, contract names, bounded result classes, source identities, and synthetic
control semantics are retained. Database URLs/passwords, cookies, CSRF tokens, idempotency keys,
session/principal/decision values, provider material, SQL errors, row data, and real identities
are excluded.

The local Keycloak workforce profile deliberately cannot claim a production phishing-resistant
authenticator. The action-bound workforce session used by the real-store test is created through
the application store in a disposable seeded database; HTTP transport is separately covered by
the handler test. This is synthetic same-host security evidence, not a real-provider,
multi-region, production incident-response, or independent-review claim.

Revalidate by 2026-10-27 or on any session, step-up, permission/role, purpose/reason,
administrator route, idempotency, Audit, migration, PostgreSQL, or recovery change, whichever
occurs first.
