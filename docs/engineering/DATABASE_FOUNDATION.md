# Atlas database foundation and Phase 01 identity persistence

S05 established the feature-free PostgreSQL controls for `FND-021`, `FND-025`, and `FND-060..064`.
P01-S03 deliberately activates the ADR 0013 `first-product-schema` and
`first-product-durable-state` triggers by adding only the ADR 0014-owned `atlas_identity` and
`atlas_audit` namespaces. P01-S04 extends only those namespaces with OIDC login transactions,
encrypted application sessions, replay records, and revocation authority. There is still no
wallet, ledger, balance, journal, payment, transfer, outbox, or other financial state.

## Boundaries and roles

The local environment generates a distinct credential for each database identity:

| Role | Allowed foundation use | Explicitly denied or disabled |
|---|---|---|
| `atlas_migration` | apply reviewed migrations/seeds and own foundation, Identity, and Audit schema objects | superuser, role creation, database creation, replication, bypass RLS |
| `atlas_api` | table-specific Identity reads/writes and insert-only Audit writes | policy/seed/scope-registry writes, Audit read/update/delete, DDL, grants, migration-role escalation |
| `atlas_worker` | Phase 00 permission-probe DML only | Identity/Audit access, product writes, DDL, migration-role escalation |
| `atlas_reporting_read` | Phase 00 foundation reads only | Identity/Audit access, writes, DDL, temporary tables |
| `atlas_break_glass` | may assume migration role only during an explicit bounded activation | login is expired by default and re-expired after the drill |
| `atlas_backup` | physical base backup and WAL streaming | ordinary application and migration ownership |

The original S04 PostgreSQL identity remains a local bootstrap identity so existing named volumes can be upgraded without losing their generated credential. It is not passed to the API or worker.

## Migration policy

Each `db/migrations/*.sql` file has closed metadata covering lock risk, representative data,
query-plan review, space risk, forward fix, rollback, lock timeout, and statement timeout.
`db/migrations/MANIFEST.sha256` defines the six-file-pair released inventory. `dbctl verify`
rejects changes, deletions, unmanifested files, reordering, malformed metadata, embedded
transaction control, privileged SQL, unratified schemas, and financial terms.

The runner applies one migration per transaction with `lock_timeout=500ms` and
`statement_timeout=5s`, and records its exact checksum. The separate seed runner validates its
closed manifest and applies one JSON document in a transaction. That document is fixed at
`2026-07-26T00:00:00Z`, is SHA-256-bound to the canonical 23-permission/13-role policy, maps the
three local Keycloak subjects to synthetic Atlas principals, and includes two tenants, two
memberships, one workforce role, one revoked session recovery canary, and one Audit fact.
Application startup applies neither migrations nor seeds.

## Commands

Static verification:

```powershell
pwsh -NoProfile -File ./scripts/verify-s05.ps1
go run ./cmd/dbctl verify --migration-dir db/migrations
pwsh -NoProfile -File ./scripts/test-s05-migration-canary.ps1
pwsh -NoProfile -File ./scripts/verify-p01-s03.ps1
```

Local database lifecycle:

```powershell
pwsh -NoProfile -File ./scripts/s05.ps1 -Action Up
pwsh -NoProfile -File ./scripts/s05.ps1 -Action Migrate
pwsh -NoProfile -File ./scripts/s05.ps1 -Action Verify
pwsh -NoProfile -File ./scripts/s05.ps1 -Action BackupRestore
pwsh -NoProfile -File ./scripts/s05.ps1 -Action Down
```

`Verify` applies migrations and seeds idempotently, exercises real PostgreSQL roles, migrates
empty and Phase-00-version throwaway databases, rejects duplicate subjects and cross-population
memberships, proves a tenant-leading repository query against real PostgreSQL, forces a bounded
product-table lock failure, and confirms real NATS JetStream. `BackupRestore` creates and verifies
a physical base backup, archives WAL, mutates the revoked-session canary after the target, restores
into the separate internal-only recovery service, and proves the restored migration/seed
checksums, product rows, grants, Audit fact, and revoked authority.

The full command is:

```powershell
pwsh -NoProfile -File ./scripts/verify-s05.ps1 -Live
```

Pass `-ContainerRuntime docker` to the PowerShell commands when Docker Compose is the selected provider.

## Failure posture

The API readiness probe uses its application credential and a 750 ms deadline to require
migration version 6 with the exact released checksum. Connectivity, authentication, missing
schema, timeout, and checksum mismatch all produce the same topology-free not-ready result;
liveness and version remain independent.

Migration failures never trigger an automatic destructive down migration. Follow [Migration failure](../runbooks/MIGRATION_FAILURE.md). Suspected backup corruption follows [Backup corruption or restore failure](../runbooks/BACKUP_CORRUPTION.md). General connectivity and readiness handling remains in [Database unavailable](../runbooks/DATABASE_UNAVAILABLE.md).

## Honest limitations

- The local backup and WAL volumes are not encrypted at rest. P01-S04 revalidates the current
  synthetic product state only; a reference deployment or backup encryption/key-custody change
  still requires stronger recovery evidence and independent review at the ADR 0012 triggers.
- The S04 core HTTP/OIDC/application-session boundary invokes Identity and Audit persistence, but
  contracted step-up idempotency, live higher-assurance completion, admin security revocation,
  authorization evaluation, approval, credential, and frontend product behavior remain absent.
- No outbox, inbox, idempotency, object, key, or synthetic financial flow exists to replay or reconcile after restore.
- Bounded verifier signals cover migration, seed, lock, role, restore, identity operations, and
  provider requests. Audit persistence is exercised synchronously by revocation; deployed alert
  routing and authorization-decision telemetry remain future owner-slice work.
- The current Windows host required direct in-VM `podman-compose` because the host Podman Compose transport is unhealthy. The repository commands are still the canonical procedure and require clean-host revalidation.
