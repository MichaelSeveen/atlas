# ADR 0010 — Native PostgreSQL migration and recovery controls

- **Status:** Accepted
- **Date:** 2026-07-21
- **Owners:** Platform owner
- **Related requirements/threats:** FND-021, FND-025, FND-060 through FND-064; THR-007, THR-014, THR-015, THR-018 through THR-020, THR-027, THR-040, THR-051, THR-054, THR-060

## Context

Phase 00 must prove migration safety, least-privilege database access, and recoverability before product or financial tables exist. The solution must stay reversible, use the selected PostgreSQL source of truth, avoid application startup migrations, and produce failure evidence against a real database. No migration framework identity or product schema is otherwise required yet.

## Decision

Use repository-owned, ordered SQL migrations with closed JSON risk metadata and a SHA-256 manifest. `dbctl verify` validates the entire released inventory before execution. A controlled admin command invokes PostgreSQL's `psql`; each unapplied migration runs in its own explicit transaction with `ON_ERROR_STOP`, bounded `lock_timeout`, bounded `statement_timeout`, and a checksum recorded in `atlas_foundation.schema_migrations`. Released SQL and metadata are append-only; corrections use a new forward migration.

The API uses pinned `pgx/v5` only for its bounded migration-state readiness query. API and worker processes do not possess migration credentials and never run schema changes. Bootstrap creates distinct migration, API, worker, reporting-read, disabled break-glass, and physical-backup roles with closed grants and distinct generated local credentials.

Use native physical `pg_basebackup`, continuous WAL archiving, `pg_verifybackup`, and a separate internal-only PostgreSQL restore service for local/reference recovery drills. Recovery targets a dedicated data volume and promotes only after reaching the recorded target time. The active database is never the restore target.

The local developer volumes are not encrypted at rest. This proves recovery mechanics but leaves encrypted backup, key-access recovery, product object/state checks, replay verification, and the final Phase 00 recovery gate outstanding.

## Alternatives considered

- A third-party migration framework was deferred because the present requirements need deterministic SQL ordering, metadata, checksums, and PostgreSQL-native failure behavior, all of which are repository-owned. Add one only through a superseding ADR when it removes demonstrated complexity.
- Application-startup migrations were rejected because they couple service availability to privileged credentials and allow ordinary runtime identities to affect schema.
- Logical dumps alone were rejected because they do not prove the required point-in-time recovery path.
- Restoring over the active local database was rejected because a failed drill could destroy the source needed for diagnosis.

## Consequences

- Migration history, risk analysis, and timeouts are reviewable without a hidden framework state machine.
- PostgreSQL client/server compatibility and native backup tooling are pinned through the exact reference image.
- A real role matrix can prove positive access and denied escalation paths.
- The migration role remains powerful and must be short-lived and controlled; application roles remain schema-inert.
- Physical recovery is PostgreSQL-specific, consistent with ADR 0002.
- Production encryption, retention, immutable storage, monitoring, and secret-manager integration are still separate decisions.

## Migration and rollback/exit strategy

Prefer a new forward-fix migration after release. Roll back only when the migration metadata identifies a safe, tested reversal and no committed data would be reinterpreted or lost. To replace the runner, first require the new tool to read or deterministically map the existing checksummed history, prove empty and previous-version lanes, and execute an isolated restore. To replace PostgreSQL, supersede ADR 0002 and provide an invariant-preserving migration and recovery plan.

## Verification and evidence

Run the static manifest validator and mutation/deletion canaries; apply twice to a real PostgreSQL container; migrate empty and previous-version databases; exercise positive and denied role paths including expiring break-glass; force a long-lock timeout; verify a physical base backup; archive WAL; and restore to an isolated target where the pre-deletion marker and migration checksums are present.
