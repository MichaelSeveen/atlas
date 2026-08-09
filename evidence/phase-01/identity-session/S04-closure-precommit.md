# P01-S04 identity/session closure pre-commit verification

## Evidence identity

- **Evidence ID:** `EVD-P01-S04-CLOSURE`
- **Phase/slice:** `PHASE-01_IDENTITY_ACCESS_TENANCY` / `P01-S04`
- **Source:** `UNCOMMITTED_WORKTREE(base=744d1e1f1ede231b00897a0ee10d3342ece15495)`
- **Observed date:** 2026-07-27
- **Environment:** Windows host; Go 1.25.12; Bun 1.3.0; React 19.2.7; Podman;
  PostgreSQL 18.4; NATS JetStream; Keycloak 26.7.0; Chromium through the Codex
  in-app browser
- **Requirements:** `FND-011`, `FND-025`, `FND-060..064`, `IAM-001..007`,
  foundation for `IAM-020..021` and `IAM-025`
- **Threats/tests:** `THR-006..007`, `THR-018`, `THR-020`, `THR-037`,
  `THR-039`, `THR-044`, `THR-056`, `THR-058`, `THR-060`,
  `ADV-IAM-009..010`, and the Phase 01 back-forward-cache “test most agents
  skip”

This report is additive. It does not replace the S04 core, CI remediation,
Gosec remediation, step-up, account-enumeration, browser-navigation, or
administrator-revocation evidence versions.

## Closure boundary

P01-S04 closes the approved OIDC BFF and durable application-session slice:
population-specific login and callback, server-side session cookies, current
principal, session inventory, self revocation, scoped step-up, workforce
security revocation, provider/account-enumeration behavior, and the synthetic
browser logout/navigation boundary.

It adds no wallet, ledger, money movement, event, worker, approval, API
credential, or financial state. The React routes remain synthetic shells
without a generated product API client.

## Reproduction

The authoritative command completed:

```text
pwsh -NoProfile -File ./scripts/verify-p01-s04.ps1 -Live -ContainerRuntime podman
```

The same source state also completed the focused real-database and recovery
commands:

```text
pwsh -NoProfile -File ./scripts/p01-s04-session-repository.ps1 -ContainerRuntime podman
pwsh -NoProfile -File ./scripts/s05.ps1 -Action Verify -ContainerRuntime podman
```

## Observed result

- Every Go package, `go vet`, OpenAPI/AsyncAPI lint, migration checksum,
  architecture policy, seeded mutation, and evidence-integrity canary passed.
- Bun reported 8 tests, 28 expectations, and zero failures; the minified
  browser bundle completed.
- Migration 8, the released immutable identity seed v1, and additive policy
  seed v2 passed current, empty, previous-version, and repeated-application
  lanes.
- PostgreSQL role isolation, bounded lock abort, real NATS integration, base
  backup, WAL archive, and isolated PITR restore passed. Restored identity
  state retained both seed applications and revoked authority.
- The disposable real-PostgreSQL session suite passed concurrent exact
  administrator revocation, stable replay identity, changed-request conflict,
  immediate target rejection, stale action-bound step-up denial, and
  Audit-outage rollback.
- The rebuilt ten-service environment passed API, web, Keycloak, broker, and
  object-storage readiness.
- Customer and merchant login, current-principal, session inventory, exact
  step-up replay, higher-assurance cookie/CSRF rotation, old-cookie rejection,
  and logout revocation passed.
- Workforce baseline assurance was denied and no application session was
  issued.
- The browser logout, reload, history traversal, forward navigation, and
  direct protected-route matrix retained signed-out state with no actor shell
  or browser credential material.

The final known/absent-user matrix used nine interleaved observations per arm:

| Population | Median delta | Median ratio | p95 delta | Result |
|---|---:|---:|---:|---|
| customer | 13.3 ms | 1.06 | 12.7 ms | PASS |
| merchant | 35.6 ms | 1.24 | 73.0 ms | PASS |
| workforce | 10.5 ms | 1.07 | 37.0 ms | PASS |

Every arm returned the same HTTP 200 generic error outcome without an Atlas
callback. The complete run ended with:

```text
p01_s04_account_enumeration_matrix=PASS(copy=status=timing-bounded,repeated-attempt-outcome=uniform)
p01_s04_real_postgres_session_repository=PASS
p01_s04_live=PASS
p01_s04_live_verification=PASS
p01_s04_verification=PASS
```

## Retained failures and corrections

Two material failures were retained during closure rather than waived:

1. The web lifecycle gate initially reported 24,633 ms because it timed the
   complete Compose provider command. The control now sends the bounded stop
   directly to the identified web container and leaves Compose to perform the
   subsequent teardown. The same source path observed exit 0 in 7,662 ms
   against the unchanged 8,000 ms requirement.
2. The first migration-8 upgrade rejected a changed checksum for released seed
   `atlas-phase01-identity-v1`. That seed was restored byte-for-byte to digest
   `e5a8ab37437edad69ed655e6589efffd824ca4b9151b6f9d9358632bf1f13d6c`.
   Additive seed `atlas-phase01-identity-policy-v2` now verifies that exact
   predecessor and advances only the permission/role policy checksum. Upgrade,
   fresh, replay, backup, and restore lanes then passed.

## Sanitization and limitations

Only public versions, source identity, route/population names, bounded result
labels, aggregate timing, safe browser counts/navigation classifications, and
artifact digests are retained. Credentials, database URLs, authorization
codes, state/nonce/PKCE values, cookies, CSRF/idempotency values,
session/principal/decision identifiers, provider payloads, SQL error detail,
WAL/data pages, screenshots, and real identity data are excluded.

This is same-host synthetic evidence. Local Keycloak uses fresh password
confirmation for customer/merchant level 2 and does not prove MFA or
phishing-resistant workforce authentication. The administrator route’s
action-bound workforce authority is proved in a disposable real PostgreSQL
store, while HTTP transport is proved separately by handler tests. No
real-provider, production incident-response, multi-region, independent-review,
constant-time, or production rate-control claim is made.

Revalidate by 2026-10-27 or on any identity contract/policy, OIDC provider,
session/cookie/step-up/revocation behavior, seed/migration, browser lifecycle,
PostgreSQL role/recovery, timing method, or evidence-policy change, whichever
occurs first.
