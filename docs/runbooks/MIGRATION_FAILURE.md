# Migration failure or unsafe lock

## Scope

This runbook covers the S05 feature-free PostgreSQL migration foundation. It does not authorize product data repair, direct status edits, or an automatic down migration.

## Immediate containment

1. Stop the rollout and keep affected instances out of traffic through readiness. Do not bypass migration-state checks.
2. Preserve the source revision, migration manifest digest, target environment, UTC time, safe database error class, and migration version. Never capture credentials, connection strings, SQL parameters, or row data in evidence.
3. Confirm `GET /health/live` independently. A live process with not-ready migration state is expected.
4. Do not edit an already-released SQL or metadata file and do not manually rewrite `atlas_foundation.schema_migrations`.

## Diagnose safely

Run `go run ./cmd/dbctl verify --migration-dir db/migrations` before touching the database. If validation fails, restore the reviewed source state or append a reviewed forward fix. If PostgreSQL reports a lock or statement timeout, inspect only environment-private lock metadata, cancel the migration session if it remains active, and verify the transaction rolled back before retrying.

Use the role matrix to confirm the migration was not attempted with API, worker, reporting, or break-glass credentials. Break-glass must remain expired unless an approved incident owner activates it for a bounded period and records purpose and expiry.

## Rollback versus forward fix

Use a forward fix by default. Consider rollback only when the migration metadata explicitly says reversal is safe, the failed transaction did not commit, and the reversal cannot reinterpret or lose committed data. A partially committed or ambiguous result blocks traffic until database history and schema evidence establish one state.

## Recovery verification

Before resuming rollout:

1. run the manifest validator and migration canaries;
2. apply migrations twice and confirm idempotence;
3. confirm the exact version/checksum using the application role;
4. run the real role matrix and bounded lock test;
5. verify API readiness returns 200 without diagnostic detail;
6. attach sanitized revision-bound evidence and record any forward fix as a new migration.
