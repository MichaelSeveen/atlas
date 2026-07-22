# Phase 00 — Secure engineering foundation

## Outcome

Create a reproducible engineering platform where security, contracts, migrations, observability, evidence, and recovery are enforced before financial features exist.

## Why this phase is high-signal

Most portfolio projects begin with screens and bolt on CI, security scans, tracing, or backups later. Atlas begins with the mechanisms that make future claims verifiable. A reviewer should see a delivery system capable of safely changing a financial application—not only an application that currently works.

## Dependencies

None.

## Product and engineering deliverables

### Repository and module boundaries

- `FND-001` Create the repository structure in the roadmap document.
- `FND-002` Define module dependency rules and enforce them with tests or static checks.
- `FND-003` Create separate Go entry points for API, worker, and provider simulator.
- `FND-004` Choose React or Vue once. Reference implementation uses React + TypeScript.
- `FND-005` Create a typed money primitive, opaque identifier primitive, UTC clock abstraction, actor context, correlation context, and domain error vocabulary.
- `FND-006` Ban floating-point money and direct `time.Now()` use in domain code.

### Local development environment

- `FND-010` One command provisions PostgreSQL, Redis, broker, object storage, telemetry collector, identity provider, API, worker, simulator, and web application.
- `FND-011` Seed deterministic synthetic tenants, users, accounts, and provider scenarios.
- `FND-012` Local environment has no dependency on production credentials or real services.
- `FND-013` Destructive reset requires explicit environment confirmation.

### CI/CD

- `FND-020` Pull requests run Go tests with race detector where appropriate, frontend tests, type checking, lint, contract lint, migrations, security scans, and secret scans.
- `FND-021` Integration tests use real PostgreSQL and broker containers, not mocked repositories.
- `FND-022` Generate SBOMs for backend, frontend, and container images.
- `FND-023` Build immutable images tagged by source revision and digest.
- `FND-024` Sign release artifacts and attach build provenance.
- `FND-025` Run database migration checks against empty and previous-release schemas.
- `FND-026` Require code-owner review for ledger, authorization, migrations, cryptography, CI, and deployment paths. In explicitly synthetic solo-maintainer mode, an accepted ADR may defer independent approval only while protected pull requests, required automated gates, sensitive-change declarations, fresh-context self-review, and prohibition/revalidation triggers are enforced; this is an accepted deviation, not independent-review evidence.
- `FND-027` Detect OpenAPI/AsyncAPI breaking changes.

### Environments

- `FND-030` Define local, test, staging, and production-reference configurations.
- `FND-031` No environment shares signing, encryption, database, identity, or merchant credentials.
- `FND-032` Environment banners and synthetic-data labels are visible in every non-production UI.
- `FND-033` Feature flags identify owner, expiry, default, risk classification, and rollback behaviour.

### Observability

- `FND-040` Propagate request ID, correlation ID, and trace context from edge through API, worker, simulator, database spans, and events.
- `FND-041` Structured logs have source redaction and injection-safe encoding.
- `FND-042` Establish baseline RED metrics, database pool metrics, queue lag, worker retries, and build/deploy metadata.
- `FND-043` Create a “golden synthetic request” trace used as an observability smoke test.

### Security baseline

- `FND-050` Complete system context, data-flow, trust-boundary, and initial STRIDE threat model.
- `FND-051` Define data classification and logging rules.
- `FND-052` Establish secret-management abstraction and rotation procedure.
- `FND-053` Add secure HTTP headers, body limits, timeouts, CORS defaults, and safe error handling.
- `FND-054` Pin and verify third-party dependencies; document update policy.
- `FND-055` Create vulnerability disclosure and dependency emergency-update runbooks.

### Database foundation

- `FND-060` Separate migration, API, worker, reporting-read, and break-glass database roles.
- `FND-061` Application roles cannot alter schema.
- `FND-062` Migration files are immutable after release.
- `FND-063` Every migration has lock-risk analysis, forward-fix plan, and test against representative data.
- `FND-064` Configure backup and point-in-time recovery in the reference environment.

## API foundation

Initial endpoints:

- `GET /health/live` — process liveness only.
- `GET /health/ready` — dependency readiness without leaking internals.
- `GET /version` — source revision, contract version, build time, no secrets.

Create common OpenAPI components for:

- problem details;
- request and correlation IDs;
- idempotency header;
- cursor pagination;
- money;
- actor/session metadata;
- ETags/resource versions.

## Frontend foundation

- Application shells for customer, merchant, and workforce routes.
- Route-level error boundaries and safe recovery.
- Central request client generated or verified against OpenAPI.
- Design tokens, accessible form primitives, tables, status chips, dialogs, toasts, timelines, and empty/error/loading states.
- No financial colour convention without text/icon equivalent.
- Secure handling of cached queries on logout and tenant switching.
- Development-only mock mode must be visually unmistakable and cannot silently replace contract tests.

## Tests most agents will skip

1. Secret scanner catches a seeded canary secret in a deleted Git history commit.
2. Log test proves CRLF/newline and structured-field injection cannot forge entries.
3. Readiness fails when database migrations are behind, while liveness remains healthy.
4. Worker crash while holding a claimed outbox row does not permanently strand it.
5. Configuration rejects production mode with development keys or wildcard origins.
6. Browser logout clears query cache and sensitive pages do not reappear through back-forward cache.
7. Contract examples execute against a live test server.
8. Migration test simulates long lock acquisition and verifies timeout/abort behaviour.
9. Restore a backup into an isolated environment and compare seed checksums.
10. Race detector and concurrent integration tests run under constrained database pool sizes.

## Security review

- Threats: leaked development secret, vulnerable dependency, poisoned build, unsafe default config, log data leakage, migration privilege abuse, environment confusion.
- Required evidence: threat register rows, SBOM, signed image, provenance file, secure-header scan, secret-scan report, restore log.

## Observability and runbooks

Runbooks:

- failed deployment;
- migration failure;
- secret exposure;
- dependency emergency patch;
- database unavailable;
- broker backlog;
- telemetry pipeline unavailable;
- rollback versus forward-fix decision.

## Acceptance gate

Phase 00 passes only when a fresh machine can clone the repository, verify toolchain prerequisites, start the platform, run all checks, execute a traced request, create a signed build, restore the database, and reproduce the evidence using documented commands.

## X content pillars

### Pillar A — “Why I did not start my fintech project with the wallet screen”

Thread beats:

1. Screens are the least dangerous part of money movement.
2. Show the trust-boundary diagram.
3. Explain why build provenance, migrations, secrets, and restore are product requirements.
4. Demonstrate a deliberately leaked canary secret blocked by CI.
5. End with the exact phase gate.

### Pillar B — “A modular monolith is not the beginner version of microservices”

- Show bounded contexts and allowed dependency graph.
- Explain which operations need one database transaction.
- List extraction criteria.
- Show one static dependency rule failing.

### Pillar C — “The first production feature I built was a restore”

- Explain why backup configuration is not recovery evidence.
- Show point-in-time restore steps.
- Run invariant and checksum verification.
- Publish measured RTO for the test environment without inflating it into a production claim.

### Short-form posts

- A screenshot of the same request across browser, API, outbox, worker, and simulator trace.
- “Five things Redis is forbidden to own in this wallet.”
- “Why every feature flag in a financial system needs an expiry and rollback plan.”

## Do not waste time on

- elaborate landing page;
- multiple frontend frameworks;
- Kubernetes before a reproducible container deployment exists;
- a dozen microservices;
- cloud-vendor-specific complexity before local recovery works;
- perfect visual branding;
- claiming SLSA, SOC, ISO, or PCI compliance from automated scans.
