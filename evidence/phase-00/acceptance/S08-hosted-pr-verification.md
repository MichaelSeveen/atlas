# S08 hosted PR verification

- **Evidence ID:** EVD-P00-S08-003
- **Created:** 2026-07-22T17:48:06Z
- **Hosted source revision:** `10ed35b8d86a68d821c89f69822289f5ab655aa8`
- **Pull request:** [MichaelSeveen/atlas#19](https://github.com/MichaelSeveen/atlas/pull/19)
- **Workflow run:** [atlas-pr run 29943586545](https://github.com/MichaelSeveen/atlas/actions/runs/29943586545)
- **Requirements:** closes the hosted S08 race-execution facet of FND-020; supports FND-021..023, FND-025, FND-027, and FND-054; FND-020/FND-023/FND-024/FND-026 remain partial
- **Threats:** THR-009, THR-013, THR-014, THR-019, THR-030, THR-042, and THR-060
- **Result:** PASS for all five hosted jobs, including the S08 real constrained-pool race test; ruleset/review/release gates remain absent
- **Revalidate by:** 2026-08-22 or on any workflow, test, dependency, contract, migration, scanner, image, ownership, or release change

## Hosted jobs

| Job | Job ID | Duration | Result |
|---|---:|---:|---|
| `static-contracts-secret-history` | 89003294676 | 2m00s | PASS |
| `postgres-nats-migration-lanes` | 89003294629 | 1m33s | PASS |
| `codeql-go-typescript-go` | 89003294612 | 1m45s | PASS |
| `codeql-go-typescript-javascript-typescript` | 89003294642 | 57s | PASS |
| `sbom-vulnerability-license-container` | 89003294568 | 3m32s | PASS |

The run was a `pull_request` event and GitHub reports conclusion `success` for exact head `10ed35b8d86a68d821c89f69822289f5ab655aa8`.

## S08 race and constrained-pool proof

The real PostgreSQL/NATS job log records:

```text
=== RUN   TestMostAgentsSkip10ConstrainedDatabasePool
--- PASS: TestMostAgentsSkip10ConstrainedDatabasePool (0.02s)
s08_constrained_pool_connections=1
s08_constrained_pool_race=PASS
s08_named_skipped_test_10=PASS
```

This is the required Linux `go test -race` execution against the real migrated PostgreSQL service. It closes the local CGO limitation recorded in EVD-P00-S08-001 without rewriting that historical report.

## Remaining hosted gates

After the run, the repository rulesets API still returned `[]`, and the release workflow run list remained empty. The successful checks are therefore execution evidence, not required-check enforcement. No independent code-owner approval, bypass-policy proof, GHCR promotion, keyless signature, or provenance is claimed.

## Sanitization

This derivative retains public repository, revision, PR, workflow, job, duration, and bounded PASS markers only. It contains no token, credential, connection string, runtime secret, customer data, product payload, scanner database, or complete job log.
