# ADR 0008 — Local/reference dependencies use a reversible synthetic platform

- **Status:** Accepted
- **Date:** 2026-07-21
- **Owners:** Platform owner
- **Related requirements/threats:** FND-010 through FND-013, FND-030, FND-031; THR-019, THR-020, THR-025, THR-042, THR-045, THR-060

## Context

Phase 00 needs a complete, reproducible environment without introducing real services, credentials, product behavior, Kafka, Kubernetes, or a custom identity provider. Broker, object storage, telemetry, and identity choices must remain replaceable because production topology is not yet selected.

## Decision

Use a Compose-compatible local/reference topology with PostgreSQL as the future authoritative store, ephemeral Redis, NATS with bounded JetStream storage, MinIO, the OpenTelemetry Collector, and Keycloak. Run API, worker, simulator, and web as separate non-root processes. Bind host ports to loopback, generate local credentials outside version control, import three isolated synthetic identity realms, and require typed synthetic-only configuration.

This accepts the products only for local/reference foundation use. It does not select a production broker, object store, IdP deployment model, secrets manager, or telemetry backend. No broker stream, application bucket, identity exchange, database schema, or product endpoint is authorized by this ADR.

## Consequences

- A single reversible command can exercise the complete process/dependency boundary.
- Deterministic failure and readiness checks no longer require real vendors.
- Compose/provider and image availability are local bootstrap dependencies.
- NATS, MinIO, and Keycloak semantics must not leak into domain APIs.

## Migration and rollback/exit strategy

Stop without data loss using Compose down. Remove only the resolved `atlas-local` namespace after exact confirmation. Replace any local/reference dependency behind configuration and adapters with a superseding ADR; no product data migration exists in S04.

## Verification and evidence

Validate four closed configurations, unique credential references/fingerprints, full-stack readiness/restart/teardown, public health minimization, real-endpoint canaries, realm discovery, broker/object-store health, and contained reset failure cases. Revisit before S07 production-reference hardening.
