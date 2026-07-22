# Phase 00 acceptance procedure

## Scope

S08 is the final planned Phase 00 implementation slice. It reviews all 37 foundation requirements, with focused acceptance for FND-010, FND-020..024, FND-040..043, and FND-064. It adds no product endpoint, event, schema, broker stream, identity exchange, financial behavior, worker job, or wallet UI.

The authorization and financial boundaries do not change. The only live state used by this procedure is generated synthetic local configuration, the feature-free PostgreSQL migration catalogue, and the synthetic recovery marker. No money, tenant, customer, merchant, identity-document, provider, inbox, outbox, object-reference, idempotency, or journal state exists to restore or reconcile.

## Reviewer commands

Static acceptance, including the earlier build/test/contract checks and evidence-tamper canaries:

```powershell
pwsh -NoProfile -File ./scripts/verify-s08.ps1
```

Full local acceptance with history scanning, four SBOM surfaces, hardened images, the synthetic stack, the exported golden trace, collector outage, real PostgreSQL/NATS migration lanes, isolated backup/restore, the constrained-pool test, and a bounded exit-zero Bun shutdown:

```powershell
pwsh -NoProfile -File ./scripts/verify-s08.ps1 -Live -History -SupplyChain -ContainerRuntime podman
```

After S08 is committed and the worktree is clean, add `-CleanClone`. The clean-clone lane checks out the exact current revision into a disposable ignored directory and reruns static S08 acceptance. This is a clean-tree reproduction on the same host; it is not independent clean-machine evidence.

The release workflow runs the same full command with Docker and `-CleanClone` on its fresh hosted runner. It is ref-guarded to protected `main` or a `v*` tag, and the preflight precedes registry authentication or any artifact publication. Do not treat that wiring as hosted proof until an explicitly authorized release run succeeds and its digests, signatures, attestations, and sanitized logs are retained.

The procedure always tears down the live foundation in a `finally` path. Restore uses the internal-only `postgres-restore` service and never targets the active PostgreSQL namespace.

## Tests most agents skip

| Test | S08 disposition | Evidence or trigger |
|---|---|---|
| #1 deleted-history secret | PASS | S07 complete-history scan and disposable deleted-commit canary |
| #2 log injection | PASS | S06 CRLF and forged-field sink tests |
| #3 migration-lag readiness | PASS | S03 readiness/liveness separation test |
| #4 claimed outbox row survives worker crash | NOT_APPLICABLE | No outbox table, claim protocol, broker consumer, or worker job exists in Phase 00. Implement this with the first transactional-outbox slice; inventing a row lease now would create forbidden product semantics. |
| #5 unsafe production configuration | PASS | S04 development-key, environment, and wildcard-origin canaries |
| #6 logout/back-forward cache | PASS | S04 React function-component session tests and synthetic-shell evidence |
| #7 live contract examples | PASS | S07 canonical OpenAPI examples against the real feature-free API handler |
| #8 long migration lock | PASS | S05 `ADV-REL-008` bounded abort and rollback proof |
| #9 isolated restore | PASS | S05 physical backup, WAL archive, internal restore, migration checksum, and pre-deletion marker |
| #10 constrained-pool race | PARTIAL until the S08 branch runs in hosted Linux | Local S08 passed 24 concurrent real PostgreSQL readiness checks with a one-connection pool. This CGO-disabled host cannot supply `-race`; the updated PR integration lane requires it. |

## Applicable adversarial review

- `ADV-REL-007` is exercised by the one-connection concurrent readiness test; work completes within a bounded deadline and the pool releases its connections.
- `ADV-REL-008` is exercised by the real long-lock migration abort and recovery lane.
- The current API compatibility facet of `ADV-REL-009` is exercised by the canonical Git-baseline contract comparison. No rolling product API or event consumer exists.
- `ADV-REL-001..003`, `ADV-REL-005`, `ADV-DR-001..004`, and skipped test #4 require product outbox, provider, report/object, key, inbox, or workflow state that Phase 00 deliberately forbids.
- `ADV-REL-004`, `ADV-REL-006`, `ADV-REL-010`, and `ADV-DR-005` have only foundation topology/runbook or primitive coverage; their product degraded-session, expiry/order, and failover semantics remain later-phase work.

Expected absences are acceptance findings, not passing simulations.

## Evidence and completion rule

`scripts/test-s08-evidence-integrity.ps1` validates the closed S01–S08 catalogue against exact SHA-256 digests and the current source identity. It then mutates a disposable artifact and a disposable source revision and requires both to fail. The catalogue contains only sanitized repository evidence; runtime credentials, raw scanner databases, GitHub tokens, customer data, and connection strings are excluded.

Phase 00 is complete only when every requirement-specific gap is either satisfied or resolved by an accepted, scope-compatible deviation; the relevant traceability rows and evidence catalogue must identify the exact committed source. EVD-P00-S08-008 now proves the full acceptance command on a genuinely independent clean supported machine, active hosted branch rules, and digest-promoted signed/provenanced artifacts with retained SBOMs. ADR 0012 permits the explicitly recorded synthetic-only solo-maintainer deviation without pretending self-review is independent; genuine code-owner approval remains mandatory before any policy revalidation trigger. S08 hosted-release closure is complete, but the overall Phase 00 completion claim remains blocked by the six partial requirements listed in `IMPLEMENTATION_STATUS.md`.
