# Requirements and quality gates

## 1. Requirement language

- **MUST** — release-blocking requirement.
- **SHOULD** — required unless an ADR documents a stronger alternative or an accepted risk.
- **MAY** — optional capability.

Every MUST requirement needs:

1. a stable requirement ID;
2. one or more tests or review procedures;
3. an evidence artifact;
4. a responsible module;
5. a failure or rollback strategy.

## 2. Project-wide release blockers

### Financial correctness

- `FIN-INV-001` Every committed journal balances per currency.
- `FIN-INV-002` Posted journals and postings cannot be updated or deleted by application roles.
- `FIN-INV-003` Every customer-visible financial transaction references its journal, reservation, or explicit no-posting reason.
- `FIN-INV-004` Current balance projection can be rebuilt from source records and compared without unexplained variance.
- `FIN-INV-005` Concurrent spend attempts cannot cause unauthorized negative available balance.
- `FIN-INV-006` A retry after timeout cannot create a second business effect.
- `FIN-INV-007` Cross-currency operations use explicit quotes and linked per-currency journals.

### Security

- `SEC-GEN-001` No browser access token is persisted in localStorage, sessionStorage, IndexedDB, or readable cookies.
- `SEC-GEN-002` All object-level reads and writes enforce tenant and subject authorization server-side.
- `SEC-GEN-003` Privileged workforce actions require separate workforce identity and role evaluation.
- `SEC-GEN-004` High-risk actions require step-up and, where specified, maker-checker approval.
- `SEC-GEN-005` Secrets, tokens, identity documents, raw provider payloads, and sensitive personal data are not logged.
- `SEC-GEN-006` Cryptographic keys are versioned, rotatable, access-controlled, and never committed to source.
- `SEC-GEN-007` All external callbacks and merchant webhooks have replay-resistant authentication.
- `SEC-GEN-008` API, UI, and event authorization tests include cross-tenant and horizontal privilege attacks.
- `SEC-GEN-009` Dependency, secret, static, container, IaC, and SBOM checks run in CI.
- `SEC-GEN-010` A threat model exists for every new trust boundary or high-value flow.

### Reliability

- `REL-GEN-001` Domain change and outbox record are committed atomically.
- `REL-GEN-002` Consumers tolerate duplicate and out-of-order events where ordering is not guaranteed.
- `REL-GEN-003` Jobs have bounded retries, exponential backoff, dead-letter handling, and operator visibility.
- `REL-GEN-004` Backup restore is exercised, not merely configured.
- `REL-GEN-005` Every long-running state has a watchdog, timeout policy, and operational path.
- `REL-GEN-006` Critical business metrics and technical signals have alerts and runbooks.

### Privacy and audit

- `PRV-GEN-001` Every sensitive data field has a classification, purpose, lawful-basis placeholder, retention rule, and masking policy.
- `PRV-GEN-002` Data export and deletion workflows preserve financial and audit obligations while minimising personal identifiers.
- `PRV-GEN-003` Automated risk outcomes expose factors and a human-review path for reviewable cases.
- `AUD-GEN-001` Security-relevant and privileged actions create tamper-evident audit events.
- `AUD-GEN-002` Audit events include actor, action, target, reason, decision, timestamp, source, correlation ID, and before/after references where appropriate.

### API and frontend

- `API-GEN-001` Public HTTP APIs are contract-first and documented in OpenAPI.
- `API-GEN-002` Events are documented in AsyncAPI with versioned schemas and delivery semantics.
- `API-GEN-003` Mutation endpoints document idempotency behaviour and error outcomes.
- `API-GEN-004` Errors use a consistent machine-readable problem format.
- `UX-GEN-001` Financial states have precise customer and operator copy; generic “something went wrong” is insufficient for known states.
- `UX-GEN-002` Destructive and irreversible actions show consequences, require confirmation, and prevent accidental double submission.
- `UX-GEN-003` Keyboard, screen-reader, focus, error-summary, and color-independent status behaviour are tested on critical flows.

## 3. Phase completion gate

A phase is complete only when all of the following exist:

- approved domain model and state diagram;
- updated threat model and data-flow diagram;
- database migration and rollback/forward-fix plan;
- OpenAPI and event contract changes;
- authorization matrix;
- functional UI states including loading, empty, partial, stale, denied, pending, failed, and recovered;
- database-backed integration tests;
- adversarial and concurrency tests;
- observability dashboard and alerts for the new flow;
- runbook for stuck or failed states;
- seeded demonstration scenario;
- evidence entry in the traceability matrix;
- at least one technical content artifact grounded in real implementation evidence.

## 4. Evidence types

Accepted evidence includes:

- passing test with reproducible command;
- benchmark report with hardware and dataset;
- architecture diagram linked to ADR;
- trace screenshot with redacted synthetic data;
- restore log and checksum comparison;
- threat model with completed mitigations;
- reconciliation report and deterministic rerun proof;
- recorded failure-injection demonstration;
- accessibility audit report;
- signed release provenance and SBOM.

A screenshot of a successful UI alone is not sufficient evidence for a financial or security requirement.

## 5. Pull request gate

Every pull request that changes money movement, authorization, identity, risk, audit, provider integration, or reconciliation MUST include:

- requirement IDs;
- threat and abuse-case impact;
- migration and rollback impact;
- idempotency and concurrency analysis;
- failure-before-commit and failure-after-commit analysis;
- test evidence;
- telemetry additions;
- PII/logging review;
- API compatibility statement.

## 6. Definition of “enterprise-grade” for this project

The term means the implementation has explicit controls for correctness, security, operability, recovery, governance, and change. It does not mean the project has a particular service count, cloud bill, certification logo, or volume claim.
