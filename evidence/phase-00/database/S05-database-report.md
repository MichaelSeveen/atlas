# S05 database roles, migration safety, and recovery report

- **Evidence ID:** EVD-P00-S05-001
- **Created:** 2026-07-21T09:57:00Z
- **Source revision:** `UNCOMMITTED_WORKTREE(base=199b86113a9f0fcda323ae2775acf026b521067e)`
- **Base tree:** `c3d7ecf3a9dca6ee19baa47787decf4c9e7d9b81`
- **Requirements:** FND-021, FND-025, FND-060, FND-061, FND-062, FND-063; partial FND-064
- **Threats:** THR-007, THR-014, THR-015, THR-018, THR-019, THR-020, THR-027, THR-040, THR-051, THR-054, THR-060
- **Named skipped tests:** #8 application role cannot perform DDL; #9 long-lock migration aborts safely
- **Adversarial coverage:** `ADV-REL-008`; recovery-mechanics facets of `ADV-DR-001..005` that are applicable before product schema, keys, objects, inbox/outbox, or financial state exist
- **Seed:** fixed `s05-pitr-marker` and feature-free permission probes only
- **Result:** PASS for S05 requirement-scoped database mechanics, with FND-064 explicitly partial
- **Revalidate by:** 2026-08-21 or on any Go/PostgreSQL/pgx/migration/role/compose/recovery change

## Scope and boundaries

S05 introduced no API contract change, product endpoint, product schema, broker stream, identity integration, wallet UI, financial amount, journal, provider behavior, or worker job. The only schema is `atlas_foundation`, containing migration history, permission probes, and a synthetic recovery marker.

Authorization is confined to PostgreSQL role capabilities. Financial boundaries, idempotency, concurrency, before/after-commit financial failures, and replay are not represented because there is no financial command or durable product state. The lock test exercises schema-change concurrency only.

## Configuration identity

| Artifact | SHA-256 |
|---|---|
| `deploy/local/compose.yaml` | `f02db7df40c6030861e0cc5704ce32291173a1d9804f71a361d3ddf55dfcb020` |
| `db/migrations/MANIFEST.sha256` | `d745a31e80c684dcfb2a4dd86938a322ac7694afa083b3b229f4a95f9f306d12` |
| `deploy/environments/local.json` | `86a922c9c288fbc6c64bcefba22ddc75d15e4d7fadbdfbc5fc34f08c574f33cc` |
| `deploy/environments/test.json` | `bb9810e6cbfc582b830d481699c5189996014e2c7381cb3b77a94954f19f03f1` |
| `deploy/environments/staging.json` | `9736d262b2d141915fbca33adb3ded338dfd9c584fa5512f7d7362de4a180871` |
| `deploy/environments/production-reference.json` | `f6e5071468e2914b3846f17e391fa8d9f4f1b79a422d5e65b663d41c30f13a96` |

Runtime credential values and the ignored runtime environment file are deliberately excluded. Generated local credentials were validated as distinct by purpose and environment; evidence contains neither values nor reversible fingerprints.

## Reproduction

```powershell
pwsh -NoProfile -File ./scripts/verify-s05.ps1
pwsh -NoProfile -File ./scripts/s05.ps1 -Action Up
pwsh -NoProfile -File ./scripts/s05.ps1 -Action Verify
pwsh -NoProfile -File ./scripts/s05.ps1 -Action BackupRestore
pwsh -NoProfile -File ./scripts/test-s04-live.ps1
pwsh -NoProfile -File ./scripts/s05.ps1 -Action Down
```

Expected: builds/tests pass; two migrations match the closed manifest; both seeded manifest violations are killed; migration execution is repeatable; empty and version-1 databases reach version 2; application/reporting role escalation is denied; break-glass expires; a migration blocked by a three-second exclusive lock fails before the lock is released and leaves no column; real JetStream is present; the API is ready only with the current schema; base backup and WAL verification pass; the isolated restore contains the pre-deletion marker and exact checksum.

Observed results are preserved in `S05-static-verification.txt`, `S05-live-database.txt`, and `S05-recovery.txt`. The lock attempt aborted in one second. The verified base backup took six seconds. These local timings are diagnostic measurements, not production RTO/RPO claims.

## Requirement results

| Requirement | Observed evidence | State |
|---|---|---|
| FND-021 | Real PostgreSQL role/migration/lock/recovery tests and real NATS JetStream check; mocks cannot satisfy the live scripts | Satisfied locally; protected CI wiring remains S07 |
| FND-025 | Empty and previous-version throwaway databases migrated and cleaned up; repeated current migration was idempotent | Satisfied locally; protected CI wiring remains S07 |
| FND-060 | Distinct migration/API/worker/reporting/break-glass credentials and real allow/deny matrix | Satisfied |
| FND-061 | API/worker/reporting create/alter/drop/grant/temp/escalation attempts denied | Satisfied |
| FND-062 | Closed released SHA-256 inventory; changed SQL and deleted metadata canaries killed | Satisfied pre-commit; post-commit immutability proof required after commit |
| FND-063 | Mandatory closed risk metadata; representative foundation rows; 500 ms lock timeout; failed ALTER rolled back and database remained usable | Satisfied for the feature-free schema |
| FND-064 | Verified physical backup, WAL archive, isolated target-time restore, migration checksum, and recovery marker | Partial: local storage is unencrypted and no product/object/key/replay checks exist |

## Sanitization and limitations

Transcripts contain only stable pass markers, versions, counts, checksums, bounded durations, and HTTP status codes. They exclude secret values, runtime configuration contents, connection strings, SQL errors, row data, container logs, host names beyond the documented loopback topology, and identity material.

The exact host `scripts/verify-s05.ps1 -Live` path remains limited by the pre-existing unhealthy Windows Podman transport. Static verification ran through the exact repository command; live commands ran through the same Compose file and scripts using in-VM `podman-compose`. Clean supported-host proof remains S08.

This report is pre-commit. It is bound to the named base commit plus an uncommitted worktree, so it cannot provide a final Git tree or implementation-commit identity. After an authorized commit, rerun the verifier from a clean tree and add a new post-commit evidence artifact; do not overwrite this report.

`FND-064` remains partial because the named volume is not encrypted at rest and there is no product schema, object/key material, outbox/inbox/idempotency state, or financial flow to verify after restore. S05 does not claim production backup durability, RPO/RTO, disaster recovery, or Phase 00 completion.
