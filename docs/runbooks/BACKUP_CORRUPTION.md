# Backup corruption or restore failure

## Scope

This runbook covers the S05 local/reference physical PostgreSQL backup, WAL archive, and isolated point-in-time restore drill. Developer volumes are not encrypted; do not represent this procedure as production disaster recovery.

## Immediate containment

1. Do not restore over the active database and do not delete the suspected backup or WAL set.
2. Stop promotion of the restore instance. Preserve the source revision, PostgreSQL image tag, backup start/end UTC times, target time, manifest checksum, bounded logs, and failure stage.
3. Never attach runtime environment files, credentials, connection strings, data pages, WAL contents, or unsanitized row data to evidence.
4. If the active database is healthy, leave it unchanged. If it is unavailable, also follow `DATABASE_UNAVAILABLE.md`.

## Triage

- If `pg_verifybackup` fails, quarantine that base backup and select a previously verified set; do not waive verification.
- If WAL is missing, identify the first missing segment and preserve archive-command state. Do not invent a later recovery target.
- If recovery cannot reach the target time, keep the restore isolated and collect only bounded server diagnostics.
- If post-restore checksums or the recovery marker differ, treat the restore as invalid even if PostgreSQL accepts connections.

## Recovery and verification

Create a fresh exact restore volume and rerun `scripts/s05.ps1 -Action BackupRestore`. Success requires all of the following: base-backup verification, archived WAL presence, successful target-time recovery, promotion out of recovery, two migration records, the exact released checksum, and the pre-deletion synthetic marker. The active database must still have the marker absent.

Product schema, object storage, encryption-key access, outbox/inbox/idempotency replay, and financial invariant checks do not exist in S05. Their absence blocks `FND-064` completion and the final Phase 00 recovery claim.
