# P01-S04 synthetic OIDC and application-session core evidence

## Evidence identity

- **Evidence ID:** `EVD-P01-S04-CORE`
- **Phase/slice:** `PHASE-01_IDENTITY_ACCESS_TENANCY` / `P01-S04` core checkpoint
- **Observed date:** 2026-07-26
- **Source identity:** `UNCOMMITTED_WORKTREE(base=2884484a99eeb2b846a56c90177163e37e419d11)`
- **Environment:** Windows host, PowerShell 7, Go 1.25.12, Podman/WSL, repository-pinned
  PostgreSQL and Keycloak images
- **Seed/time:** `atlas-phase01-identity-v1`; fixed seed time `2026-07-26T00:00:00Z`;
  injected clocks for lifetime/timing boundaries
- **Migration release:** version 6; current checksum
  `b8c46daa86ff72667161201fc494fb296325737330d988117fd1e76c62f6e9a0`
- **Identity provider:** deterministic local Keycloak realms and runtime-only synthetic test users;
  the generated test credential was rotated after the browser check and was never added to
  source or evidence

## Scope and expected result

This checkpoint was expected to compose only the ADR 0014-owned OIDC BFF and durable
application-session boundary against the P01-S03 Identity/Audit stores.

Expected:

- exact population-specific issuer, audience, redirect URI, callback parameter, state, nonce,
  PKCE S256, token timing, signing algorithm, and signing-key validation;
- one-use bounded server-side login transactions with no tokens in browser-readable storage;
- opaque Secure/HttpOnly/SameSite application cookies backed by encrypted refresh-token storage;
- population-specific idle/absolute lifetimes and zero-grace session rotation;
- current-session inventory, logout, self-revoke, and concurrent revoke-all with synchronous,
  atomic Audit facts and replay/conflict behavior;
- step-up initiation requires CSRF and forces a fresh higher-assurance provider request;
- provider outage preserves an already-valid low-risk session while denying new authentication
  and step-up;
- bounded identity/provider metrics, source-redacted logs, alert/dashboard catalogue entries, and
  incident/revocation runbooks;
- real PostgreSQL role, concurrency, migration, backup/WAL/PITR, and restored-revocation proof;
- no authorization evaluator, approval, API credential, worker/event behavior, financial state,
  or wallet UI.

## Observed result

PASS for the stated core checkpoint.

- OIDC unit and HTTP matrices rejected wrong issuer/audience/state/nonce/redirect/timing,
  non-RS256 tokens, unknown signing keys, callback query ambiguity, transaction replay, and old
  cookies. Discovery and token endpoints use separately validated internal transport URLs while
  preserving the exact public issuer.
- Customer and merchant local Keycloak journeys completed login, callback, current-principal,
  current-session inventory, CSRF denial, fresh step-up initiation, logout, and old-cookie
  rejection. Cookies were Secure, HttpOnly, and SameSite=Lax; response bodies contained no access
  or refresh token. Workforce baseline login was denied and created no application session.
- A real in-app browser completed the synthetic customer login journey. The runtime-only test
  credential was then rotated through the repository-owned Keycloak configuration command and
  the customer/workforce HTTP journeys were rerun successfully. No credential value is retained
  here.
- Population lifetime and rotation tests used injected clocks. A successful synthetic
  higher-assurance callback rotated the application session and rejected the prior cookie in the
  service test boundary. The live provider journey currently proves only the fresh step-up
  request parameters.
- The disposable `atlas_p01_s04_session_test` database exercised the actual API-role repository.
  Two concurrent identical revoke-all calls produced exactly one original commit and one replay;
  a sequential identical call replayed the committed result, a changed request conflicted, and
  the Audit count stayed atomic. The database is dropped in a `finally` path.
- Logout, revoke-one, and revoke-all commit application-session authority and synchronous Audit
  facts in one transaction. Revoke-all serializes per principal with a transaction-scoped
  advisory lock and does not require broad table-update authority.
- An injected identity-provider outage preserved an existing valid low-risk current-session read
  while new login and step-up failed closed. The provider-unavailable and session-compromise
  runbooks describe the same boundary.
- Identity operation metrics use the closed operation set `login`, `callback`, `current`,
  `logout`, `session_list`, `session_revoke`, `session_revoke_all`, and `step_up`. Provider
  request metrics use only `discovery`/`token`, population, and `ok`/`error` dimensions. Focused
  metric and alert-cardinality tests passed.
- Six migration pairs passed manifest verification. The full real database role/migration/lock,
  NATS, base-backup, WAL, and isolated PITR sequence passed; the observed restore RTO was 32
  seconds and revoked authority remained revoked at the recovery target.
- API code reaches persistence only through Identity/Audit application boundaries; focused
  architecture tests reject direct command-to-persistence imports.

Initial iterations failed closed when the API role lacked database `CONNECT`, when `FOR SHARE`
required an unintended table-wide update grant, and when a stale container image was reused.
The bootstrap grant, principal-only column grant, advisory-lock repository transaction, and
forced bounded rebuild orchestration were corrected before the passing runs. No failed
revocation/audit transaction committed partial authority state.

## Reproduction and observed commands

```text
go run ./cmd/dbctl verify --migration-dir db/migrations
migration_count=6
current_version=6
current_checksum=b8c46daa86ff72667161201fc494fb296325737330d988117fd1e76c62f6e9a0
migration_manifest=PASS

pwsh -NoProfile -File ./scripts/p01-s04-session-repository.ps1 -ContainerRuntime podman
p01_s04_real_postgres_session_repository=PASS

pwsh -NoProfile -File ./scripts/s05.ps1 -Action Verify -ContainerRuntime podman
database_empty_and_previous_lanes=PASS
database_role_matrix=PASS
database_long_lock_abort=PASS
database_integration_broker=REAL_NATS_JETSTREAM
database_base_backup=PASS
database_wal_archive=PASS
database_isolated_pitr_restore=PASS product_identity_state=verified revoked_authority=preserved
database_restore_rto_seconds=32

pwsh -NoProfile -File ./scripts/s04.ps1 -Action Up -ContainerRuntime podman
s04_environment_up=PASS

pwsh -NoProfile -File ./scripts/test-p01-s04-oidc-http.ps1 -Population customer
pwsh -NoProfile -File ./scripts/test-p01-s04-oidc-http.ps1 -Population merchant
pwsh -NoProfile -File ./scripts/test-p01-s04-oidc-http.ps1 -Population workforce
p01_s04_customer_http=PASS
p01_s04_merchant_http=PASS
p01_s04_workforce_baseline_denied=PASS
```

The complete static/live reproduction command is:

```text
pwsh -NoProfile -File ./scripts/verify-p01-s04.ps1 -Live -ContainerRuntime podman
```

## Requirements, threats, ownership, and failure posture

- **Requirements evidenced at this checkpoint:** `IAM-001..007`, with session-boundary
  foundations for `IAM-020..021` and `IAM-025`. Every IAM row remains `Planned` phase-wide.
- **Revalidated foundation requirements:** `FND-011`, `FND-021`, `FND-025`, `FND-040..043`,
  `FND-053`, `FND-060..064`.
- **Threats exercised:** `THR-005..008`, `THR-018`, `THR-020`, `THR-023`, `THR-037..039`,
  `THR-044`, `THR-056..058`, `THR-060`.
- **Owners:** Identity owns provider mapping, login transactions, application sessions, and
  revocation; Audit owns append-only facts and its recorder; Platform/Data own HTTP, database
  roles/migrations, telemetry, Keycloak orchestration, and recovery controls.
- **Authorization boundary:** possession of an application session is authentication only.
  No role/permission evaluator or authorization decision is inferred from a browser cookie.
- **Financial boundary:** no money amount, wallet, balance, reservation, journal, payment,
  transfer, payout, refund, beneficiary, or other financial state/operation is introduced.
- **Idempotency/concurrency:** revoke-all has durable request-digest replay and conflict behavior
  with per-principal transaction serialization. Step-up currently declares but does not yet
  implement its contracted `Idempotency-Key` replay boundary.
- **Before commit:** invalid provider data, expired/replayed login state, session expiry,
  revocation conflict, audit failure, or database failure commits no new session authority.
- **After commit:** session/revocation state is durable; repeated revoke-all retrieves the
  committed result. Response-loss recovery for step-up is still open.
- **Forward fix:** released migrations are immutable. Correct behavior with ordered migrations
  and additive evidence; never edit released migration/seed history.

## Sanitization and limitations

This artifact contains only synthetic population labels, closed operation names, checksums,
bounded result labels, counts, and durations. It excludes runtime environment files, credentials,
database URLs, authorization codes, state/nonce/PKCE values, cookies, session identifiers, access
or refresh tokens, raw provider payloads, internal URLs containing credentials, SQL error detail,
WAL/data pages, and real identity/provider data.

Limitations:

- pre-commit, same-host local/reference evidence; no independent human review is claimed;
- local backup/WAL developer volumes are unencrypted;
- step-up does not yet enforce/replay/conflict on its contracted `Idempotency-Key`;
- a successful live higher-assurance/ACR completion is not configured or proven; only live
  initiation and synthetic service-level successful rotation are proven;
- admin security revocation is absent;
- stable safe authentication error copy is proven, but the bounded existing/absent account timing
  differential is not complete;
- browser login is proven once, but the complete cache/storage/BFCache/logout navigation matrix
  has not been rerun against the S04 product session;
- no real IdP, real identity data, production secret custody, reference deployment, deployed
  alert routing, authorization evaluator, approval, API credential, event, worker input, or
  financial state exists.

Revalidate by 2026-10-26 or on any OIDC client/provider configuration, cryptography, cookie,
session, migration/seed/policy/schema/role, telemetry, recovery, container, or runtime change,
whichever occurs first.
