# Audit persistence unavailable

## Scope

This runbook covers the synchronous, append-only P01-S03 Audit persistence boundary. No product
HTTP mutation invokes it yet. When a later privileged or denied high-risk action does, Audit
failure must deny the action and the caller-owned PostgreSQL transaction must not commit.

## Immediate containment

1. Deny the privileged/high-risk action with the safe service-unavailable contract. Never commit
   the domain mutation without its required Audit fact.
2. Keep the database/readiness policy fail-closed. Do not redirect Audit to logs, Redis, a file, an
   unratified event, or an operator SQL statement.
3. Preserve source revision, migration/seed checksums, UTC time, safe correlation/decision IDs,
   action class, and the bounded database error class. Exclude actor, tenant, subject, verifier,
   SQL, credential, and before/after values.
4. Verify that `atlas_api` retains INSERT but not SELECT/UPDATE/DELETE on
   `atlas_audit.audit_events`, and that worker/reporting roles retain no Audit access.

## Diagnose and recover

Run `go run ./cmd/dbctl verify --migration-dir db/migrations`, the focused Audit package tests,
and the real PostgreSQL role matrix. If a migration or checksum differs, follow
`MIGRATION_FAILURE.md`. If PostgreSQL is unavailable, follow `DATABASE_UNAVAILABLE.md`.

Resume the caller only after a synthetic transaction proves:

- the domain mutation and Audit insert commit together;
- a forced Audit insert failure rolls both back;
- the application role cannot update/delete/read Audit facts directly;
- the safe response contains no internal detail.

P01-S03 proves the persistence and rollback boundary only. Runtime metrics, alert thresholds, and
an HTTP failure exercise must be added by the first slice that invokes this recorder.
