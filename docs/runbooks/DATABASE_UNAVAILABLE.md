# Database unavailable or migration state unverified

## Scope

This runbook defines the safe readiness posture for the PostgreSQL adapter and migration-state
probe after the P01-S04 core checkpoint. Synthetic Identity/Audit rows and the OIDC/session BFF
now exist, but automated failover and production recovery are not implemented.

## Expected external behavior

- `GET /health/live` remains `200 {"status":"alive"}` while the process can serve requests.
- `GET /health/ready` returns a generic `503 DEPENDENCY_DEGRADED` when a required dependency is unavailable, migrations are behind, or migration state cannot be verified.
- The readiness response must not contain a dependency name, host, port, schema, migration version, SQL, credential, or stack trace.
- `GET /version` remains available with safe build metadata.

## Initial response

1. Remove the instance from traffic using readiness; do not disable or bypass the check.
2. Confirm liveness separately. A liveness failure is a process incident, not a database diagnosis.
3. Confirm the deployed source and contract revisions using `/version`.
4. Use environment-private diagnostics to distinguish connectivity, credential/role, capacity, TLS, migration lag, or migration-state-read failure. Never paste credentials, connection strings, SQL payloads, customer data, or internal topology into tickets or public evidence.
5. If migrations are behind or the checksum differs, stop rollout and follow `MIGRATION_FAILURE.md`. Do not mark the service ready manually and do not edit released history.

## Recovery verification

Restore readiness only after the real application-role checker proves both required dependency
availability and migration version 6 with its exact released checksum. Then verify, in order:

1. readiness changes to `200`;
2. liveness remained healthy unless the process was intentionally restarted;
3. source/contract/build metadata matches the intended deployment;
4. no sensitive diagnostic detail appeared in response bodies or retained evidence.

## Telemetry degradation

The database readiness count/duration, pool metrics, bounded identity-operation metrics, and
bounded provider-request metrics remain authoritative only for their documented source.
Migration, seed, role, lock, and restore scripts emit bounded PASS/failure-class signals without
tenant, actor, subject, SQL, or credential fields. Deployed alert routing and
authorization-decision metrics do not exist; do not claim that coverage early.

## Escalation and evidence

The platform owner owns this foundation response. Preserve UTC timestamps, source/configuration revision, safe request/correlation IDs, readiness transitions, actions, and outcome. Sanitize before attaching evidence. Use `BACKUP_CORRUPTION.md` only for physical-backup, WAL, or isolated-restore failures.
