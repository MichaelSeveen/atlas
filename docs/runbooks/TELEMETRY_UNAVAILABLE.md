# Telemetry unavailable

Owner: platform-on-call. Severity: ticket for isolated loss; page if loss hides a simultaneous authoritative-service incident.

1. Confirm API liveness/readiness and authoritative PostgreSQL state independently. Do not make the application unready solely because the collector is unavailable.
2. Check collector process health, OTLP port, memory limit, export queue/backpressure, and recent configuration/revision. Do not enable verbose payload capture or log credentials while debugging.
3. Bound the incident window and affected services using process build markers and container stdout. Treat missing telemetry as missing evidence, not evidence of success.
4. Restore the collector or roll back its configuration. Application export queues are bounded and may drop telemetry; do not retry unboundedly.
5. Run the S06 golden synthetic request and verify API, readiness, and database span linkage plus RED/database/build metrics.
6. Record the gap, cause, sanitized evidence, and whether any security or financial review needs reconstruction from authoritative sources.

Never lower redaction, cardinality, TLS, or environment controls to recover visibility.
