# S06 observability and security operating-baseline report

- **Evidence ID:** EVD-P00-S06-001
- **Created:** 2026-07-21T12:51:07Z
- **Source revision:** `UNCOMMITTED_WORKTREE(base=7a08056539de6d655086f7730d0cb8df3a9bb4c6)`
- **Base tree:** `bbc2b0003781851a08a01a955da1dffdad20f7d9`
- **Requirements:** partial FND-040 and FND-042; satisfied FND-041, FND-043, FND-050, FND-051, FND-052, FND-053, and FND-055
- **Threats:** executable coverage for canonical THR-001..060; immediate S06 focus THR-005, THR-009, THR-010, THR-013, THR-014, THR-019, THR-020, THR-025, THR-030, THR-031, THR-040, THR-042, THR-045, THR-048, THR-049, THR-056, and THR-060
- **Named skipped test:** #2, structured-log injection/source-redaction
- **Adversarial coverage:** `ADV-RES-003`, provider-neutral `ADV-DR-004` facets, telemetry outage, label-cardinality and ownerless-alert mutations
- **Synthetic seed:** W3C trace ID `4bf92f3577b34da6a3ce929d0e0e4736`; no customer, identity, credential, provider, or financial fixture
- **Result:** PASS for the current feature-free S06 path, with FND-040 and FND-042 explicitly partial
- **Revalidate by:** 2026-08-21 or on any Go/OpenTelemetry/pgx/collector/log-field/metric/alert/secret-policy/threat/configuration change

## Scope and boundaries

S06 adds operational safety to the foundation that actually exists: API/readiness/database request tracing, process build/lifecycle telemetry, HTTP RED and database metrics, closed-schema JSON process logs, a Phase 00 DFD/STRIDE baseline, versioned secret-provider contracts, and incident runbooks. It adds no product endpoint, schema, event, broker stream, worker job, identity exchange, provider behavior, wallet UI, cryptographic operation, or financial behavior.

Authorization remains the existing HTTP edge and PostgreSQL role boundary. There is no money, idempotent command, product concurrency, or before/after-commit financial state to test. Telemetry is deliberately non-authoritative: its queue/export failure cannot make the API unready or change database truth.

## Environment and configuration identity

- Go `1.25.7` with language baseline `1.25.0`; pgx `5.10.0`; OpenTelemetry Go/SDK/exporters `1.43.0`.
- Bun `1.3.0`; React/React DOM `19.2.7`. Web source and product behavior are unchanged by S06.
- Local live proof used Podman `5.8.2`, WSL `podman-compose` `1.5.0`, PostgreSQL `18.4-alpine`, OpenTelemetry Collector `0.155.0`, and the exact images pinned in `deploy/local/compose.yaml`.
- `deploy/local/compose.yaml`: `e01ae2212b0df81c3122a7bbdde5e2c1f5d8588721802f157dcec68bf85ca9b0`
- `deploy/local/otel-collector.yaml`: `ad0e9f605251ada97d8540796ea5ea4577d8ff0ffe9a33b09436398346d8166a`
- `deploy/observability/catalog.json`: `03ba801a3f7b28d821392d171036ae4264b9ca0365fde7baa57d965e483d10b1`
- `docs/security/PHASE-00-THREAT-COVERAGE.json`: `6870336a39a12028b8333f415c315dc1a434180e6076bea7e534f5b371f94744`

Runtime credential values, connection strings, generated environment files, container-log dumps, and reversible credential fingerprints are excluded.

## Reproduction

```powershell
pwsh -NoProfile -File ./scripts/verify-s06.ps1
pwsh -NoProfile -File ./scripts/verify-s06.ps1 -Live -ContainerRuntime podman
pwsh -NoProfile -File ./scripts/s06.ps1 -Action Down -ContainerRuntime podman
```

Expected: all Go build/vet/tests, environment and migration validation, frozen Bun install/type/test/build, threat coverage, structured-log injection, secret rotation/outage, metric catalog, and alert mutations pass. The fixed inbound parent is exported as linked API, `readiness.check`, and `database.schema_readiness` spans; the collector observes HTTP count, database pool, and build metrics. Stopping the collector leaves `/health/ready` at `200`, then the collector restarts.

Observed:

```text
s06_alert_catalog_mutations=KILLED
golden_trace_id=4bf92f3577b34da6a3ce929d0e0e4736
golden_trace_spans=api,readiness,database
telemetry_outage_readiness=200
s06_live_observability=PASS
s06_environment_verify=PASS
source_revision=UNCOMMITTED_WORKTREE(base=7a08056539de6d655086f7730d0cb8df3a9bb4c6)
s06_verification=PASS
```

The repository stack was stopped after the exercise. Named volumes were preserved; no material data was deleted.

## Requirement results

| Requirement | Observed evidence | State |
|---|---|---|
| FND-040 | Validated request/correlation/W3C context and fixed-parent continuity through exported API, readiness, and database spans | Partial: there is no event/job/worker request path to propagate through |
| FND-041 | Closed JSON records contain no free-form message/error/body/header/identity/secret field; CRLF and forged structured values write zero bytes; bootstrap and SDK/server paths cannot emit raw errors | Satisfied; deployment retention/access is revalidated in S07/S08 |
| FND-042 | Emitted HTTP RED, database readiness/pool, and revision/build metrics have a closed cardinality catalog; dashboards/alerts have owner, severity, rationale, runbook, and mutation test | Partial: queue/retry metrics are definition-only and no deployed alert engine/routing exists |
| FND-043 | Collector observes the deterministic API/readiness/database trace and expected metrics; collector outage does not affect readiness | Satisfied for the complete current synthetic request path |
| FND-050 | Current context/DFD, six trust boundaries, initial STRIDE review, and exact executable THR-001..060 links | Satisfied baseline; every new boundary must update it |
| FND-051 | Canonical policy plus documentation and executable classification/retention inventory for all accepted log fields | Satisfied and preserved |
| FND-052 | Versioned provider-neutral references, explicit environment/purpose/algorithm/floor, bounded overlap, revocation, outage/recovery, material-copy wiping, and downgrade/cross-boundary rejection | Satisfied abstraction; no managed provider/HSM/custody claim |
| FND-053 | Existing HTTP safety suite passes and legacy/raw server diagnostic output is suppressed | Satisfied and preserved |
| FND-055 | Private disclosure and dependency emergency procedures plus the synthetic tabletop below | Satisfied runbook scope |

## Synthetic dependency-emergency tabletop

Scenario `S06-TT-DEP-001` declares, solely for the exercise, that OpenTelemetry Go `1.43.0` has received a critical malicious-release advisory. This is not a statement that the real release is vulnerable. The scenario intentionally provides no verified fixed version.

1. The affected manifest paths (`go.mod`, `go.sum`) and process/export boundary are identified; the current source and configuration hashes above are preserved.
2. The runbook directs containment of affected builds/deployments and isolation of any untrusted reproduction. It does not request execution of third-party exploit code or use of real data.
3. Because no authentic upstream fixed release/checksum is supplied, the procedure refuses to invent `1.43.1`, weaken checksum/TLS/static policy, or silently retain an unsafe capability.
4. The disposition is `BLOCKED-PENDING-VERIFIED-UPSTREAM`, with security owner and module owner responsible. If exploitation exposed credentials, the secret-exposure procedure would rotate/revoke by purpose and environment.
5. A real remediation would require the smallest verified supported version, complete S06 plus full phase gates, source/revision/build-marker comparison, security/module-owner review, and new immutable evidence.

Observed result: PASS. The procedure preserves containment and evidence, fails closed when the safe version is unknown, and avoids making a false vulnerability or remediation claim.

## Sanitization, integrity, and limitations

Evidence contains stable IDs, public tool/image versions, source/configuration digests, bounded pass markers, and an explicitly synthetic advisory. It contains no secret material, connection string, customer/identity data, payload, raw error, row data, or full collector/container log.

This report is pre-commit. Its source identity is the named base plus an uncommitted worktree, so it cannot provide a final implementation commit/tree or immutable image digest. After an authorized commit, rerun from a clean tree and add a new post-commit artifact; do not overwrite this report.

The current host proof uses the repository's WSL `podman-compose` fallback. The exact repository wrapper succeeds here, but a clean supported Podman/Docker host remains an S08 acceptance gate. Local detailed collector output is test-only. There is no production telemetry backend, retention/access configuration, alert evaluator/router, managed secret provider, HSM, product trace, event/job path, queue lag, retry behavior, or compliance/security certification claim.
