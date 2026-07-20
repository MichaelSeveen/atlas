# Database unavailable or migration state unverified

## Scope

This Phase 00 runbook defines the safe readiness posture before a database adapter exists. It does not claim a database, migration tool, alert, dashboard, or automated recovery is implemented.

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
5. If migrations are behind, stop rollout and follow the approved migration forward-fix/rollback procedure once S05 provides it. Do not mark the service ready manually.

## Recovery verification

Restore readiness only after the injected checker proves both required dependency availability and expected migration state. Then verify, in order:

1. readiness changes to `200`;
2. liveness remained healthy unless the process was intentionally restarted;
3. source/contract/build metadata matches the intended deployment;
4. no sensitive diagnostic detail appeared in response bodies or retained evidence.

## Telemetry degradation

S03 defines safe trace/metric recorder boundaries but ships no exporter. Failure or absence of a future telemetry sink must not make liveness fail or cause unbounded buffering. Record the gap through an environment-private operational channel, preserve request/correlation identifiers only, and do not claim observability restoration until a synthetic trace is visible end to end. S06 owns exporters, bounded buffering, dashboards, alerts, and the telemetry-pipeline runbook.

## Escalation and evidence

Until S04/S05 assigns environment and database owners, the platform owner owns this foundation response. Preserve UTC timestamps, source/configuration revision, safe request/correlation IDs, readiness transitions, actions, and outcome. Sanitize before attaching evidence.
