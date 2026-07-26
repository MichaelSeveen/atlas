# Database and Phase 01 identity persistence

This directory retains the Phase 00 database controls and owns the P01-S03 migration, seed, role,
tenant-isolation, and recovery mechanics plus the P01-S04 OIDC transaction and durable
application-session lifecycle for the ratified `atlas_identity` and `atlas_audit` schemas. It
contains no wallet, ledger, balance, payment, transfer, outbox, or other financial state.

- `roles/` bootstraps distinct migration, API, worker, reporting-read, backup, and disabled break-glass identities in the synthetic PostgreSQL cluster.
- `migrations/` contains ordered forward migrations plus mandatory lock, timeout, representative-data, space, query-plan, forward-fix, and rollback metadata.
- `seeds/` contains the checksum-bound deterministic Phase 01 identity/authorization catalogue and
  synthetic principal, tenant, revoked-session, and audit fixtures. The loader is transactional,
  idempotent, policy-digest-bound, and fails on seeded-row drift.
- `migrations/MANIFEST.sha256` and `seeds/MANIFEST.sha256` are released-history boundaries. A
  released artifact is never edited, removed, or reordered; corrections use a new migration or
  seed version.

The repository-owned `dbctl verify` command checks the closed migration inventory and its checksums before any database command runs. PostgreSQL applies each migration with `ON_ERROR_STOP`, an explicit transaction, `lock_timeout=500ms`, and `statement_timeout=5s`.

Local recovery uses PostgreSQL physical base backup, WAL archiving, `pg_verifybackup`, and a
separate internal-only restore service. It proves migration and seed checksums, product rows,
insert-only Audit grants, and that a post-target authority resurrection cannot replace the
revoked session restored at the target. Developer volumes remain unencrypted at rest, so this is
synthetic local/reference recovery evidence, not production disaster-recovery or key-custody proof.
