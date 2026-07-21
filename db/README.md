# Database foundation

This directory owns Phase 00 database controls only. It contains no wallet, ledger, identity, payment, customer, or other product schema.

- `roles/` bootstraps distinct migration, API, worker, reporting-read, backup, and disabled break-glass identities in the synthetic PostgreSQL cluster.
- `migrations/` contains ordered forward migrations plus mandatory lock, timeout, representative-data, space, query-plan, forward-fix, and rollback metadata.
- `MANIFEST.sha256` is the released-history boundary. A released SQL or metadata file is never edited, removed, or reordered; corrections use a new migration.

The repository-owned `dbctl verify` command checks the closed migration inventory and its checksums before any database command runs. PostgreSQL applies each migration with `ON_ERROR_STOP`, an explicit transaction, `lock_timeout=500ms`, and `statement_timeout=5s`.

Local recovery uses PostgreSQL physical base backup, WAL archiving, `pg_verifybackup`, and a separate internal-only restore service. The developer volumes are not encrypted at rest, so this is recovery-mechanics evidence rather than completion of production-reference encrypted backup controls.
