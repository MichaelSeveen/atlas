# P01-S03 Identity/Audit persistence and recovery evidence

## Evidence identity

- **Evidence ID:** `EVD-P01-S03-PERSISTENCE`
- **Phase/slice:** `PHASE-01_IDENTITY_ACCESS_TENANCY` / `P01-S03`
- **Observed date:** 2026-07-26
- **Source identity:** `UNCOMMITTED_WORKTREE(base=2884484a99eeb2b846a56c90177163e37e419d11)`
- **Environment:** Windows host, PowerShell 7, Go 1.25.12, Podman/WSL,
  `postgres:18.4-alpine` at the repository-pinned digest
- **Seed/time:** `atlas-phase01-identity-v1`; virtual time `2026-07-26T00:00:00Z`
- **Seed SHA-256:** `e5a8ab37437edad69ed655e6589efffd824ca4b9151b6f9d9358632bf1f13d6c`
- **Policy SHA-256:** `fbdb484af7c0feda9304caa76b3e6cefa2be98e39c713f9bad3cdb76553ff23a`
- **Migration release:** version 4; current checksum
  `8331fa74149830bc1ae0f258d48df6db8e3db83b8bbf9254812ac684b2b2fc21`

## Scope and expected result

S03 was expected to introduce only ADR 0014-owned Identity/Audit persistence, deterministic
synthetic product seeds, explicit tenant query boundaries, real PostgreSQL role constraints, and
product-state recovery evidence. It was also expected to consume the ADR 0013
`first-product-schema` and `first-product-durable-state` triggers without rewriting Phase 00
historical evidence.

Expected:

- four checksum-bound forward migrations, upgrading cleanly from the Phase 00 version-two state;
- a closed registry for every Identity/Audit table with tenant, mixed, or documented global scope;
- canonical 23-permission/13-role policy loaded from a separately checksum-bound seed;
- unique external subjects and database-enforced tenant/principal/role population consistency;
- one tenant repository whose signature and SQL both require tenant context;
- API table-specific Identity access and insert-only Audit access; no product access for
  worker/reporting roles; disabled break-glass by default;
- audit insert failure rolls back its caller transaction;
- bounded product-table lock failure leaves no partial DDL;
- physical backup, archived WAL, isolated target-time restore, exact checksums/product rows, and
  revoked authority preserved after a post-target resurrection failpoint;
- no HTTP handler, OIDC exchange, Redis authorization truth, event/outbox, worker input, frontend
  product behavior, or financial state.

## Observed result

PASS.

- Empty and Phase-00-version migration lanes reached version 4; reapplication remained
  idempotent.
- The seed loaded twice into a throwaway database with one application record. Exact observed
  catalogue counts were 23 permissions, 13 roles, 119 role-permission bindings, and six
  delegation bindings. Synthetic product counts were two tenants, three principals, three
  external subjects, two tenant memberships, one workforce role assignment, one revoked session,
  and one Audit fact.
- All 11 product tables matched the closed scope registry. Composite foreign keys rejected a
  customer principal inserted as a merchant member. The unique
  `(population, issuer, subject)` constraint rejected a duplicate Keycloak subject.
- The first repository returned the seeded membership only for its owning tenant and returned the
  same concealed not-found error for a valid cross-tenant membership ID. Removing
  `tenant_id = $1` killed the focused policy test.
- `atlas_api` could read authorized Identity tables and insert an Audit fact inside a rolled-back
  transaction. It could not read/update/delete Audit, mutate role/scope catalogues, create/alter
  tables, or assume the migration role. Worker/reporting roles could not access Identity/Audit.
  Break-glass succeeded only during the bounded activation and was expired again.
- A forced invalid membership after an Audit insert rolled back the Audit fact. A locked
  `atlas_identity.memberships` table caused the DDL canary to abort in one second with no added
  column.
- `pg_verifybackup`, WAL archive, and isolated PITR passed. The restore reached promotion in 55
  seconds, retained four exact migration records, seed metadata, three principals, two
  memberships, one Audit fact, and the expected grants. A session made active after the target
  restored as revoked; the active database was returned to revoked state after the failpoint.
- The actual pgx membership repository passed against the application role on the real local
  PostgreSQL instance.

The initial live iterations failed closed before seed commit when the migration role lacked
temporary-table permission and when psql/PLpgSQL seed variables were not bound safely. The role
grant and loader were corrected, the seed manifest digest was regenerated, and the entire live
sequence was rerun to PASS. No partial seed application record or authority row committed during
those failed transactions.

## Reproduction and observed commands

```text
go run ./cmd/dbctl verify --migration-dir db/migrations
migration_count=4
current_version=4
current_checksum=8331fa74149830bc1ae0f258d48df6db8e3db83b8bbf9254812ac684b2b2fc21
migration_manifest=PASS

pwsh -NoProfile -File ./scripts/p01-s03.ps1 -ContainerRuntime podman
database_empty_and_previous_lanes=PASS
phase01_identity_seed_idempotence=PASS
phase01_identity_tenant_predicate=PASS
phase01_identity_population_constraints=PASS
phase01_audit_atomicity=PASS
database_role_matrix=PASS
database_long_lock_abort=PASS elapsed_seconds=1
database_integration_broker=REAL_NATS_JETSTREAM
database_base_backup=PASS duration_seconds=19
database_wal_archive=PASS
database_isolated_pitr_restore=PASS product_identity_state=verified revoked_authority=preserved
database_restore_rto_seconds=55
p01_s03_real_postgres_repository=PASS
p01_s03_live=PASS
```

The complete static/live reproduction command is:

```text
pwsh -NoProfile -File ./scripts/verify-p01-s03.ps1 -Live -ContainerRuntime podman
```

## Requirements, threats, ownership, and failure posture

- **Foundational requirements evidenced:** `IAM-004`, `IAM-010..012`, `IAM-020..021`, `IAM-025`.
  All remain `Planned` phase-wide; this report proves only their current persistence/control
  boundary.
- **Revalidated foundation requirements:** `FND-011`, `FND-064`.
- **Threats exercised:** `THR-005..006`, `THR-018`, `THR-020`, `THR-040..041`, `THR-044`,
  `THR-058`, `THR-060`.
- **Owners:** Identity owns principal/subject/tenant/role/session persistence; Audit owns the
  append-only facts and recorder; Data/Platform own migrations, roles, backup/WAL/PITR, and the
  bounded operational scripts.
- **Before commit:** migration, seed, population, Audit, and lock failures roll back. Seed rerun
  never overwrites authority drift.
- **After commit:** there is no product response path yet. PITR proves exact durable state and that
  post-target authority activation does not appear at the recovery target.
- **Forward fix:** once released, add ordered migrations or a new seed version. Never edit a
  released SQL, metadata, seed, or checksum history entry.

## Sanitization and limitations

This artifact contains only synthetic fixed IDs, checksums, bounded result labels, counts, and
durations. It excludes runtime environment files, credentials, database URLs, cookie/verifier
values, SQL error details, subjects beyond the repository-owned synthetic mapping, WAL contents,
and data pages.

Limitations:

- pre-commit, same-host local/reference evidence; no independent human review is claimed;
- backup/WAL developer volumes are unencrypted;
- no product handler invokes Identity or Audit persistence;
- no real IdP/provider, identity data, production secret custody, reference deployment, runtime
  Audit metric/alert, event, worker input, approval, API credential, or financial state exists;
- S04 must add OIDC/session concurrency, fixation, expiry, CSRF, rotation, revocation, outage,
  telemetry, and HTTP failure evidence before any authentication/session capability claim.

Revalidate by 2026-10-26 or on any migration/seed/policy/schema/role/recovery/container/runtime
change, whichever occurs first.
