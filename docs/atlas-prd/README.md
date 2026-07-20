# Atlas Wallet & Financial Operations Platform

## Enterprise-grade portfolio PRD and verification pack

Atlas is a security-first, multi-currency wallet and money-movement platform backed by an immutable double-entry ledger, deterministic risk controls, provider simulation, merchant APIs, settlement, reconciliation, reporting, and operational tooling.

This is not an MVP brief, a generated backlog dump, a licensed financial product, or a claim of regulatory/security certification. It is a purpose-built implementation and evidence plan for demonstrating financial-system judgement with a React + TypeScript frontend and Go backend.

## Pack at a glance

- 14 ordered implementation phases, each with its own acceptance gate and X content pillars.
- 399 stable product/control requirements.
- 60 explicit threat scenarios.
- 154 centralized adversarial tests, in addition to phase-specific tests.
- 30-path OpenAPI 3.1.1 reference contract and 17-message AsyncAPI 3.0.0 event contract.
- 70 evidence-led X content items mapped to implementation phases.
- Architecture, security, privacy, reliability, DR, API, testing, governance, ADR, and evidence templates.

Counts describe this PRD pack, not implemented product capability.

## Reference implementation

- **Frontend:** React + TypeScript. Vue 3 can replace React, but do not build both.
- **Backend:** Go modular monolith with separate API, worker, simulator, migration, and controlled admin entry points.
- **Primary database:** PostgreSQL; SQL-first financial write path and database role separation.
- **Messaging:** transactional outbox plus an at-least-once broker when asynchronous fan-out begins.
- **Ephemeral state:** Redis only for rate limiting, short-lived coordination, and caching—never financial or authorization truth.
- **Object storage:** encrypted storage for synthetic evidence, statements, provider files, and signed manifests.
- **Observability:** OpenTelemetry-compatible traces, metrics, and structured logs.
- **Contracts:** source-controlled OpenAPI and AsyncAPI, validated and compatibility-checked in CI.
- **External dependencies:** deterministic KYC/payment/settlement simulators; no real money, cardholder data, or real identity documents.

## How to execute the project

1. Read the product charter, scope guardrails, glossary, requirements gates, and roadmap.
2. Read every architecture document and accepted ADR before feature code.
3. Implement phases in order. Later UI work must not invent semantics that the ledger/state/security model has not defined.
4. Treat OpenAPI, AsyncAPI, database migrations, permission catalog, posting templates, threat register, and runbooks as first-class source artifacts.
5. A phase is not complete until its acceptance scenario, adversarial tests, operations path, evidence, and content artifacts are complete.
6. Keep requirement IDs stable and update the traceability/evidence/claims ledgers as implementation evolves.
7. Publish only evidence-backed statements: diagrams, exact journal flows, failure demonstrations, security denials, benchmarks with workload, game days, and postmortems.

## Recommended repository layout

```text
apps/
  web/                 # React or Vue customer/merchant/workforce surfaces
cmd/
  api/
  worker/
  simulator/
  migrate/
internal/
  identity/
  customer/
  kyc/
  ledger/
  wallet/
  risk/
  transfers/
  merchant/
  providers/
  settlement/
  reconciliation/
  cases/
  reporting/
  audit/
  platform/
contracts/
  openapi/
  asyncapi/
db/
  migrations/
  queries/
evidence/
docs/
  adr/
  runbooks/
  threat-models/
```

The exact folders may change, but dependency and data ownership boundaries may not become implicit.

## Document map

### 00 — Master product definition

- [`00_PRODUCT_CHARTER.md`](00-master/00_PRODUCT_CHARTER.md)
- [`01_SCOPE_AND_NON_GOALS.md`](00-master/01_SCOPE_AND_NON_GOALS.md)
- [`02_DOMAIN_GLOSSARY.md`](00-master/02_DOMAIN_GLOSSARY.md)
- [`03_REQUIREMENTS_AND_QUALITY_GATES.md`](00-master/03_REQUIREMENTS_AND_QUALITY_GATES.md)
- [`04_ROADMAP_AND_DEPENDENCIES.md`](00-master/04_ROADMAP_AND_DEPENDENCIES.md)

### 01 — Architecture and control foundations

- [`00_SYSTEM_ARCHITECTURE.md`](01-architecture/00_SYSTEM_ARCHITECTURE.md)
- [`01_SECURITY_AND_TRUST_MODEL.md`](01-architecture/01_SECURITY_AND_TRUST_MODEL.md)
- [`02_DATA_ARCHITECTURE_AND_LEDGER_MODEL.md`](01-architecture/02_DATA_ARCHITECTURE_AND_LEDGER_MODEL.md)
- [`03_API_AND_EVENT_STANDARDS.md`](01-architecture/03_API_AND_EVENT_STANDARDS.md)
- [`04_RELIABILITY_OBSERVABILITY_DR.md`](01-architecture/04_RELIABILITY_OBSERVABILITY_DR.md)
- [`05_PRIVACY_AND_REGULATORY_ALIGNMENT.md`](01-architecture/05_PRIVACY_AND_REGULATORY_ALIGNMENT.md)
- [`06_ARCHITECTURE_DECISIONS.md`](01-architecture/06_ARCHITECTURE_DECISIONS.md)

### 02 — Ordered implementation phases

- [`PHASE-00_ENGINEERING_FOUNDATION.md`](02-phases/PHASE-00_ENGINEERING_FOUNDATION.md)
- [`PHASE-01_IDENTITY_ACCESS_TENANCY.md`](02-phases/PHASE-01_IDENTITY_ACCESS_TENANCY.md)
- [`PHASE-02_CUSTOMER_KYC_PRIVACY.md`](02-phases/PHASE-02_CUSTOMER_KYC_PRIVACY.md)
- [`PHASE-03_LEDGER_CORE.md`](02-phases/PHASE-03_LEDGER_CORE.md)
- [`PHASE-04_WALLETS_BALANCES_HOLDS.md`](02-phases/PHASE-04_WALLETS_BALANCES_HOLDS.md)
- [`PHASE-05_RISK_POLICY_AND_LIMITS.md`](02-phases/PHASE-05_RISK_POLICY_AND_LIMITS.md)
- [`PHASE-06_INTERNAL_TRANSFERS.md`](02-phases/PHASE-06_INTERNAL_TRANSFERS.md)
- [`PHASE-07_EXTERNAL_MONEY_MOVEMENT.md`](02-phases/PHASE-07_EXTERNAL_MONEY_MOVEMENT.md)
- [`PHASE-08_MERCHANT_PAYMENTS_AND_WEBHOOKS.md`](02-phases/PHASE-08_MERCHANT_PAYMENTS_AND_WEBHOOKS.md)
- [`PHASE-09_SETTLEMENT_AND_RECONCILIATION.md`](02-phases/PHASE-09_SETTLEMENT_AND_RECONCILIATION.md)
- [`PHASE-10_OPERATIONS_FINANCE_SUPPORT_CASES.md`](02-phases/PHASE-10_OPERATIONS_FINANCE_SUPPORT_CASES.md)
- [`PHASE-11_STATEMENTS_REPORTING_AUDIT_DATA_RIGHTS.md`](02-phases/PHASE-11_STATEMENTS_REPORTING_AUDIT_DATA_RIGHTS.md)
- [`PHASE-12_RELIABILITY_DR_PERFORMANCE_SECURITY.md`](02-phases/PHASE-12_RELIABILITY_DR_PERFORMANCE_SECURITY.md)
- [`PHASE-13_PORTFOLIO_RELEASE_EVIDENCE.md`](02-phases/PHASE-13_PORTFOLIO_RELEASE_EVIDENCE.md)

### 03 — API and event contracts

- [`openapi.yaml`](03-contracts/openapi.yaml)
- [`asyncapi.yaml`](03-contracts/asyncapi.yaml)
- [`API_DOCUMENTATION_GUIDE.md`](03-contracts/API_DOCUMENTATION_GUIDE.md)
- [`ERROR_CATALOG.md`](03-contracts/ERROR_CATALOG.md)
- [`EVENT_CATALOG.md`](03-contracts/EVENT_CATALOG.md)
- [`WEBHOOK_SECURITY.md`](03-contracts/WEBHOOK_SECURITY.md)

### 04 — Verification

- [`TEST_STRATEGY.md`](04-testing/TEST_STRATEGY.md)
- [`ADVERSARIAL_TEST_CATALOG.md`](04-testing/ADVERSARIAL_TEST_CATALOG.md)
- [`PERFORMANCE_AND_CHAOS_PLAN.md`](04-testing/PERFORMANCE_AND_CHAOS_PLAN.md)
- [`SECURITY_VERIFICATION_PLAN.md`](04-testing/SECURITY_VERIFICATION_PLAN.md)

### 05 — Build-in-public content

- [`X_CONTENT_SYSTEM.md`](05-content/X_CONTENT_SYSTEM.md)
- [`CONTENT_CALENDAR.md`](05-content/CONTENT_CALENDAR.md)

### 06 — Governance and evidence

- [`CONTROL_BASELINE.md`](06-governance/standards/CONTROL_BASELINE.md)
- [`REQUIREMENTS_TRACEABILITY.csv`](06-governance/REQUIREMENTS_TRACEABILITY.csv)
- [`THREAT_REGISTER.csv`](06-governance/THREAT_REGISTER.csv)
- [`RISK_REGISTER.md`](06-governance/RISK_REGISTER.md)
- [`DEFINITION_OF_DONE.md`](06-governance/DEFINITION_OF_DONE.md)
- [`CLAIMS_LEDGER.csv`](06-governance/CLAIMS_LEDGER.csv)
- [`EVIDENCE_INDEX.md`](06-governance/EVIDENCE_INDEX.md)
- [`adrs/`](06-governance/adrs/)
- [`PACK_VALIDATION_REPORT.md`](PACK_VALIDATION_REPORT.md)
- `MANIFEST.sha256` — integrity digests for the packaged source files

### Templates

- [`PHASE_TEMPLATE.md`](templates/PHASE_TEMPLATE.md)
- [`ADR_TEMPLATE.md`](templates/ADR_TEMPLATE.md)
- [`RFC_TEMPLATE.md`](templates/RFC_TEMPLATE.md)
- [`INCIDENT_POSTMORTEM_TEMPLATE.md`](templates/INCIDENT_POSTMORTEM_TEMPLATE.md)

## Project-wide non-negotiable invariants

1. No code path mutates a financial balance without a balanced journal entry or explicitly modelled reservation.
2. Financial amounts use integer minor units and explicit currency metadata; APIs use strings to preserve cross-language exactness.
3. Posted journal entries are immutable. Corrections use linked compensating entries.
4. Money-moving writes are idempotent, concurrency-safe, auditable, and recoverable after ambiguous failure.
5. External timeouts do not become definitive failures without evidence.
6. Every privileged action has an identified actor, authorization decision, purpose/reason, correlation, approval where required, and immutable audit evidence.
7. Sensitive data is minimised, classified, masked, encrypted, retained by policy, and excluded from logs/events/client storage unless explicitly safe.
8. At-least-once event delivery is assumed; consumers are duplicate-, replay-, and ordering-aware.
9. Operators correct systems through permissioned domain commands—not direct database edits or arbitrary status fields.
10. Security controls are executable requirements, not badges or README claims.
11. No scale, uptime, security, compliance, or “enterprise-grade” implementation claim is public without reproducible evidence and limitations.
12. The portfolio environment never handles real money, real cardholder data, or real identity documents.

## Strongest reviewer journey

A reviewer should be able to follow one synthetic merchant payment through:

1. authenticated, authorized, idempotent API command;
2. risk decision and funds reservation;
3. simulated provider acceptance followed by timeout;
4. durable `outcome_unknown` state and safe client status UX;
5. delayed signed provider callback;
6. exactly one controlled ledger journal and balance projection;
7. transactional outbox and duplicate-safe event consumers;
8. signed merchant webhook with a forced ambiguous delivery retry;
9. provider settlement file import;
10. reconciliation, including a deliberate one-minor-unit mismatch;
11. finance exception resolution with maker-checker adjustment;
12. customer/merchant statement and complete operations/audit timeline;
13. restore/replay verification showing no duplicate economic effect.

That story is the centre of the portfolio. Secondary features exist to strengthen it, not dilute it.

## Things deliberately not worth the time

- building both React and Vue implementations;
- real card or bank integrations and compliance theatre;
- cryptocurrency/blockchain immutability claims;
- premature microservices, Kafka, Kubernetes, or multi-region active-active writes;
- generic AI chatbot or autonomous fraud/money decisions;
- arbitrary journal editor or “edit transaction status” admin tools;
- stock/crypto charts and decorative analytics unrelated to operations;
- production-scale claims from toy data/hardware;
- endless visual polish before correctness, failure, and operator states work;
- generated tests that repeat implementation logic or assert only status codes.

## Completion standard

Atlas is complete only when a reviewer can:

- trace every financial state to holds, journals, provider evidence, settlement, reconciliation, and audit;
- reproduce duplicate, timeout, out-of-order, race, stale approval, tampering, abuse, and restore scenarios;
- inspect machine-readable API/event contracts and their conformance evidence;
- use customer, merchant, risk, finance, support, and security workflows;
- verify authorization, masking, step-up, approvals, key/signature, supply-chain, and recovery controls;
- reproduce benchmark/security/DR claims from documented environment and revision;
- see explicit known limitations and non-goals.
