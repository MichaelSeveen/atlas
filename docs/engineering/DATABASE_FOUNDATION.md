# Atlas database foundation

S05 adds feature-free PostgreSQL controls for `FND-021`, `FND-025`, and `FND-060..064`. It creates only the `atlas_foundation` control schema used for migration history, permission probes, and a synthetic recovery marker. There is no customer, identity, wallet, ledger, balance, journal, payment, transfer, outbox, or other product schema.

## Boundaries and roles

The local environment generates a distinct credential for each database identity:

| Role | Allowed foundation use | Explicitly denied or disabled |
|---|---|---|
| `atlas_migration` | apply reviewed migrations and own foundation schema objects | superuser, role creation, database creation, replication, bypass RLS |
| `atlas_api` | bounded DML and migration-history reads | schema/table create, alter, drop, grants, `SET ROLE atlas_migration` |
| `atlas_worker` | bounded DML and migration-history reads | schema/table create, alter, drop, migration-role escalation |
| `atlas_reporting_read` | read only | writes, DDL, temporary tables |
| `atlas_break_glass` | may assume migration role only during an explicit bounded activation | login is expired by default and re-expired after the drill |
| `atlas_backup` | physical base backup and WAL streaming | ordinary application and migration ownership |

The original S04 PostgreSQL identity remains a local bootstrap identity so existing named volumes can be upgraded without losing their generated credential. It is not passed to the API or worker.

## Migration policy

Each `db/migrations/*.sql` file has closed metadata covering lock risk, representative data, query-plan review, space risk, forward fix, rollback, lock timeout, and statement timeout. `db/migrations/MANIFEST.sha256` defines the complete released inventory. `dbctl verify` rejects changes, deletions, unmanifested files, reordering, malformed metadata, embedded transaction control, privileged SQL, and product terms.

The runner applies one migration per transaction with `lock_timeout=500ms` and `statement_timeout=5s`, and records its exact checksum. It may target only the configured database or the two fixed throwaway S05 lane databases. Application startup never applies migrations. Released files are corrected by appending a new migration.

## Commands

Static verification:

```powershell
pwsh -NoProfile -File ./scripts/verify-s05.ps1
go run ./cmd/dbctl verify --migration-dir db/migrations
pwsh -NoProfile -File ./scripts/test-s05-migration-canary.ps1
```

Local database lifecycle:

```powershell
pwsh -NoProfile -File ./scripts/s05.ps1 -Action Up
pwsh -NoProfile -File ./scripts/s05.ps1 -Action Migrate
pwsh -NoProfile -File ./scripts/s05.ps1 -Action Verify
pwsh -NoProfile -File ./scripts/s05.ps1 -Action BackupRestore
pwsh -NoProfile -File ./scripts/s05.ps1 -Action Down
```

`Verify` applies migrations idempotently, exercises real PostgreSQL roles, migrates empty and previous-version throwaway databases, forces a bounded lock failure, and confirms real NATS JetStream. `BackupRestore` creates and verifies a physical base backup, archives WAL, deletes a synthetic marker after the target time, restores into the separate internal-only recovery service, and proves the restored marker and migration checksum.

The full command is:

```powershell
pwsh -NoProfile -File ./scripts/verify-s05.ps1 -Live
```

Pass `-ContainerRuntime docker` to the PowerShell commands when Docker Compose is the selected provider.

## Failure posture

The API readiness probe uses its application credential and a 750 ms deadline to require migration version 2 with the exact released checksum. Connectivity, authentication, missing schema, timeout, and checksum mismatch all produce the same topology-free not-ready result; liveness and version remain independent.

Migration failures never trigger an automatic destructive down migration. Follow [Migration failure](../runbooks/MIGRATION_FAILURE.md). Suspected backup corruption follows [Backup corruption or restore failure](../runbooks/BACKUP_CORRUPTION.md). General connectivity and readiness handling remains in [Database unavailable](../runbooks/DATABASE_UNAVAILABLE.md).

## Honest limitations

- The local backup and WAL volumes are not encrypted at rest. ADR 0013 therefore limits `FND-064` satisfaction to the synthetic ADR 0008 reference platform; the first product durable state, reference deployment, or backup encryption/key-custody change must add the stronger recovery evidence.
- No product schema or representative product dataset exists; S05 representative data is confined to the permission and recovery probes.
- No outbox, inbox, idempotency, object, key, or synthetic financial flow exists to replay or reconcile after restore.
- Backup age, pool, lock, and restore telemetry/alerts remain S06; protected CI enforcement remains S07; clean-machine encrypted recovery acceptance remains S08.
- The current Windows host required direct in-VM `podman-compose` because the host Podman Compose transport is unhealthy. The repository commands are still the canonical procedure and require clean-host revalidation.
