# Failed deployment

Owner: release-on-call with the affected service owner.

1. Stop rollout expansion and identify source revision, image digest, build time, configuration revision, environment, and failing health/metric signal.
2. Preserve logs/traces and compare migration compatibility before rollback. Never roll application code behind an irreversible schema boundary.
3. If the previous revision remains compatible, roll back to its immutable artifact; otherwise follow the migration-failure or database-recovery runbook.
4. Verify liveness, readiness, version/build markers, RED metrics, database readiness, and the S06 golden trace. Telemetry loss alone follows `TELEMETRY_UNAVAILABLE.md` and must not disguise authoritative failures.
5. Record timeline, impact, sanitized evidence, cause, rollback decision, and required prevention test.
