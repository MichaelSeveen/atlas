# Atlas context index

This file is a routing table and compact working model. It does not replace the PRD. Use the linked source, stable requirement/threat ID, endpoint, event, module, or phase to load only the context needed for a task.

## Compact working model

### Mission, actors, and surfaces

Atlas demonstrates a synthetic multi-tenant wallet and financial-operations system that remains correct under retries, races, partial failure, delayed/duplicate events, provider disagreement, privileged misuse, and restore. The authoritative mission and limits are the [product charter](../atlas-prd/00-master/00_PRODUCT_CHARTER.md) and [scope/non-goals](../atlas-prd/00-master/01_SCOPE_AND_NON_GOALS.md).

| Actor | Surface | Boundary that matters |
|---|---|---|
| Customer | Customer web | BFF session; subject/tenant authorization; clear pending/ambiguous/recovered states |
| Merchant developer/system | Developer portal and public merchant API | Organization scope; machine credentials; idempotency; signed webhooks |
| Merchant operator | Merchant dashboard | Organization-scoped workforce session and masked operational views |
| Risk analyst | Workforce console | Versioned deterministic policies; explained decisions; no ledger posting authority |
| Operations/support | Workforce console | Purpose-bound search and domain commands; no arbitrary state/database edits |
| Finance operator | Finance/reconciliation console | File integrity, deterministic reruns, maker-checker adjustments and period controls |
| Security/audit reviewer | Evidence/audit views | Immutable/tamper-evident facts, least privilege, revision-bound evidence |
| Worker/provider identities | Internal jobs and simulator interfaces | Distinct rotatable machine identity, narrow scopes, replay-safe messages |

Source: [product charter §5–6](../atlas-prd/00-master/00_PRODUCT_CHARTER.md), [system architecture §2–3](../atlas-prd/01-architecture/00_SYSTEM_ARCHITECTURE.md), and [security model §3–5](../atlas-prd/01-architecture/01_SECURITY_AND_TRUST_MODEL.md).

### Architecture and ownership

- Deployable units are `api` (BFF/public API and synchronous commands), `worker` (outbox, provider, webhook, reconciliation/reporting/watchdogs), `provider-simulator` (deterministic hostile provider behavior), and `web` (React + TypeScript route shells). See [system architecture §3](../atlas-prd/01-architecture/00_SYSTEM_ARCHITECTURE.md) and [ADR 0001](../atlas-prd/06-governance/adrs/0001-modular-monolith.md).
- Bounded contexts are Identity, Customer, Ledger, Wallet, Risk, Transfer, Payment, Provider, Settlement, Reconciliation, Operations, Reporting, and Audit. Each owns writes to its authoritative tables and exposes explicit application contracts; cross-context reads do not transfer write ownership.
- PostgreSQL owns ledger, projections, holds, authorization mappings, financial commands, idempotency, inbox/outbox, jobs, reconciliation, and audit metadata (`FIN-INV-001..007`, `REL-GEN-001`). Redis is non-authoritative and ephemeral. Object storage holds checksum-addressed synthetic evidence/files. See [system architecture §7](../atlas-prd/01-architecture/00_SYSTEM_ARCHITECTURE.md) and [ADR 0002](../atlas-prd/06-governance/adrs/0002-postgresql-financial-ledger.md).
- Synchronous boundaries cover authorization and durable command acceptance, reservations, journal/projection commits, and returned durable IDs. Provider calls, webhook delivery, statements/exports, reconciliation/imports, retries, watchdogs, and audit manifests are asynchronous. `202` is valid only after durable state, reservation where applicable, audit, and outbox commit.
- Trust boundaries are untrusted browsers/merchant systems/provider callbacks, edge, API/BFF zone, domain/database financial zone, broker/worker zone, object storage, separate workforce identity, and CI/deployment plane. Every boundary needs authentication, authorization, confidentiality, integrity, replay, resource limits, safe logging, and explicit failure behavior (`SEC-GEN-001..010`).

### Invariants and contract ownership

- Journals balance per currency and posted history is immutable (`FIN-INV-001`, `FIN-INV-002`). Customer-visible financial state points to a journal, reservation, or no-posting reason (`FIN-INV-003`). Projections are rebuildable (`FIN-INV-004`).
- Concurrent spend and timeout retry cannot create unauthorized negative funds or duplicate economic effect (`FIN-INV-005`, `FIN-INV-006`). FX uses explicit quotes and linked balanced journals (`FIN-INV-007`). Money is integer minor units with explicit currency; APIs use decimal strings ([ADR 0006](../atlas-prd/06-governance/adrs/0006-integer-money-and-currency.md)).
- Browser checks never authorize. Workforce identities are separate; high-risk operations use step-up/maker-checker; sensitive data is minimized and excluded from unsafe logs/events (`SEC-GEN-001..010`, `PRV-GEN-001..003`, `AUD-GEN-001..002`).
- State and outbox commit atomically. At-least-once delivery, duplicates, replay, gaps, and relevant reordering are normal conditions ([ADR 0003](../atlas-prd/06-governance/adrs/0003-transactional-outbox.md), `REL-GEN-001..006`).
- HTTP behavior is owned by [OpenAPI](../atlas-prd/03-contracts/openapi.yaml); event behavior by [AsyncAPI](../atlas-prd/03-contracts/asyncapi.yaml) plus the [event catalogue](../atlas-prd/03-contracts/EVENT_CATALOG.md). Implementations conform to contracts; database structures do not become public schemas.

### Delivery order and obligations

Current delivery is Phase 00. Hard order is `00 → {01,03}`, `01 → 02`, `{01,02,04} → 05`, `03 → 04`, `05 → 06 → 07`, `{01,07} → 08`, `{07,08} → 09`, `{05,09} → 10 → 11 → 12 → 13`. See the [roadmap](../atlas-prd/00-master/04_ROADMAP_AND_DEPENDENCIES.md).

Explicit non-goals include real money/card/identity data, native mobile apps, React/Vue duplication, custom identity/cryptography, speculative AI decisions, blockchain/stablecoin paths, vanity dashboards, false exactly-once/compliance claims, premature microservices/Kafka/Kubernetes/multi-region writes, and universal event sourcing.

Every completed requirement needs stable IDs, tests/review, evidence, an owner, and rollback/forward-fix strategy. Every phase also needs its acceptance journey, threat/data-flow updates, contracts, authorization matrix, real-database/adversarial/concurrency tests, telemetry/alerts/runbooks, deterministic demo, traceability/evidence entry, and evidence-backed content. See [quality gates](../atlas-prd/00-master/03_REQUIREMENTS_AND_QUALITY_GATES.md) and [Definition of Done](../atlas-prd/06-governance/DEFINITION_OF_DONE.md).

## Specification routing table

| Question | Consult first | Then target |
|---|---|---|
| Product purpose, actors, success, legal boundary | [Product charter](../atlas-prd/00-master/00_PRODUCT_CHARTER.md) | [Scope/non-goals](../atlas-prd/00-master/01_SCOPE_AND_NON_GOALS.md), `RSK-001`, `RSK-007`, `RSK-014`, `RSK-020`, `RSK-029` |
| Domain terminology | [Domain glossary](../atlas-prd/00-master/02_DOMAIN_GLOSSARY.md) | Owning phase/domain state model; do not invent synonyms |
| Project-wide invariant or release gate | [Requirements and quality gates](../atlas-prd/00-master/03_REQUIREMENTS_AND_QUALITY_GATES.md) | [Traceability CSV](../atlas-prd/06-governance/REQUIREMENTS_TRACEABILITY.csv) by requirement ID |
| Phase order or extraction criterion | [Roadmap/dependencies](../atlas-prd/00-master/04_ROADMAP_AND_DEPENDENCIES.md) | [System architecture](../atlas-prd/01-architecture/00_SYSTEM_ARCHITECTURE.md), accepted ADRs |
| Module/deployable/data ownership | [System architecture](../atlas-prd/01-architecture/00_SYSTEM_ARCHITECTURE.md) | [ADR 0001](../atlas-prd/06-governance/adrs/0001-modular-monolith.md), [ADR index](../atlas-prd/01-architecture/06_ARCHITECTURE_DECISIONS.md) |
| Ledger, wallet, money, locking, transaction boundaries | [Data/ledger model](../atlas-prd/01-architecture/02_DATA_ARCHITECTURE_AND_LEDGER_MODEL.md) | `FIN-INV-*`; [ADR 0002](../atlas-prd/06-governance/adrs/0002-postgresql-financial-ledger.md); current financial phase |
| Identity, session, authorization, tenant, privileged action | [Security/trust model](../atlas-prd/01-architecture/01_SECURITY_AND_TRUST_MODEL.md) | `SEC-GEN-*`; [ADR 0004](../atlas-prd/06-governance/adrs/0004-bff-and-session-security.md); `THR-005..008`, `THR-023`, `THR-038..041`, `THR-057..058` |
| Data classification, retention, rights, regulatory wording | [Privacy/regulatory alignment](../atlas-prd/01-architecture/05_PRIVACY_AND_REGULATORY_ALIGNMENT.md) | `PRV-GEN-*`; [control baseline](../atlas-prd/06-governance/standards/CONTROL_BASELINE.md); `THR-009`, `THR-022`, `THR-026..028`, `THR-045`, `THR-050` |
| HTTP endpoint semantics | [OpenAPI](../atlas-prd/03-contracts/openapi.yaml) by path/operation | [API/event standards](../atlas-prd/01-architecture/03_API_AND_EVENT_STANDARDS.md), [API guide](../atlas-prd/03-contracts/API_DOCUMENTATION_GUIDE.md), [error catalogue](../atlas-prd/03-contracts/ERROR_CATALOG.md) |
| Event name/schema/delivery/replay | [AsyncAPI](../atlas-prd/03-contracts/asyncapi.yaml) by channel/message | [Event catalogue](../atlas-prd/03-contracts/EVENT_CATALOG.md), [API/event standards](../atlas-prd/01-architecture/03_API_AND_EVENT_STANDARDS.md), [ADR 0003](../atlas-prd/06-governance/adrs/0003-transactional-outbox.md) |
| Merchant webhook security | [Webhook security](../atlas-prd/03-contracts/WEBHOOK_SECURITY.md) | OpenAPI/AsyncAPI operation/event; `THR-011`, `THR-012`, `THR-035`, `THR-056` |
| Retry, ambiguity, observability, SLO, restore, disaster | [Reliability/observability/DR](../atlas-prd/01-architecture/04_RELIABILITY_OBSERVABILITY_DR.md) | [Performance/chaos plan](../atlas-prd/04-testing/PERFORMANCE_AND_CHAOS_PLAN.md), `REL-GEN-*`, `THR-015`, `THR-019`, `THR-025`, `THR-030`, `THR-048..049`, `THR-054`, `THR-059` |
| Test technique and CI lane | [Test strategy](../atlas-prd/04-testing/TEST_STRATEGY.md) | [Adversarial catalogue](../atlas-prd/04-testing/ADVERSARIAL_TEST_CATALOG.md) by ID; phase “tests most agents skip” |
| Security verification/release gate | [Security verification plan](../atlas-prd/04-testing/SECURITY_VERIFICATION_PLAN.md) | [Threat register](../atlas-prd/06-governance/THREAT_REGISTER.csv) by threat ID; [risk register](../atlas-prd/06-governance/RISK_REGISTER.md) |
| Operations, alerts, runbooks, evidence | Current phase observability/runbook section | [Reliability spec](../atlas-prd/01-architecture/04_RELIABILITY_OBSERVABILITY_DR.md), [evidence index](../atlas-prd/06-governance/EVIDENCE_INDEX.md), [Definition of Done](../atlas-prd/06-governance/DEFINITION_OF_DONE.md) |
| Public content or claims | [X content system](../atlas-prd/05-content/X_CONTENT_SYSTEM.md) | [Content calendar](../atlas-prd/05-content/CONTENT_CALENDAR.md), [claims ledger](../atlas-prd/06-governance/CLAIMS_LEDGER.csv); publish only after evidence |

## Phase routing table

| Phase | Source | Load when |
|---|---|---|
| 00 — Engineering foundation | [PHASE-00](../atlas-prd/02-phases/PHASE-00_ENGINEERING_FOUNDATION.md) | Current phase: repository, tooling, environments, CI, observability, security, DB foundation |
| 01 — Identity/access/tenancy | [PHASE-01](../atlas-prd/02-phases/PHASE-01_IDENTITY_ACCESS_TENANCY.md) | Customer/merchant/workforce/machine identity, BFF, authorization, approval foundations |
| 02 — Customer/KYC/privacy | [PHASE-02](../atlas-prd/02-phases/PHASE-02_CUSTOMER_KYC_PRIVACY.md) | Customer lifecycle, synthetic KYC, consent, restrictions, retention |
| 03 — Ledger core | [PHASE-03](../atlas-prd/02-phases/PHASE-03_LEDGER_CORE.md) | Chart of accounts, journal/posting, reversal, projection, period/FX foundation |
| 04 — Wallets/holds | [PHASE-04](../atlas-prd/02-phases/PHASE-04_WALLETS_BALANCES_HOLDS.md) | Wallet lifecycle, balances, reservation/capture/release, freezes |
| 05 — Risk/limits | [PHASE-05](../atlas-prd/02-phases/PHASE-05_RISK_POLICY_AND_LIMITS.md) | Versioned deterministic policies, limits, decisions, reviews |
| 06 — Internal transfers | [PHASE-06](../atlas-prd/02-phases/PHASE-06_INTERNAL_TRANSFERS.md) | Recipient resolution and atomic internal movement |
| 07 — External movement | [PHASE-07](../atlas-prd/02-phases/PHASE-07_EXTERNAL_MONEY_MOVEMENT.md) | Provider adapters, ambiguity, callbacks, payout/cash-in |
| 08 — Merchant payments/webhooks | [PHASE-08](../atlas-prd/02-phases/PHASE-08_MERCHANT_PAYMENTS_AND_WEBHOOKS.md) | Credentials, payment intents/refunds, delivery/replay/SSRF |
| 09 — Settlement/reconciliation | [PHASE-09](../atlas-prd/02-phases/PHASE-09_SETTLEMENT_AND_RECONCILIATION.md) | Immutable ingestion, deterministic matching, exceptions, period operations |
| 10 — Operations/cases | [PHASE-10](../atlas-prd/02-phases/PHASE-10_OPERATIONS_FINANCE_SUPPORT_CASES.md) | Workforce search, cases, commands, queues, break-glass |
| 11 — Statements/audit/data rights | [PHASE-11](../atlas-prd/02-phases/PHASE-11_STATEMENTS_REPORTING_AUDIT_DATA_RIGHTS.md) | Statements/reports/exports, audit search, data rights |
| 12 — Reliability/DR/performance/security | [PHASE-12](../atlas-prd/02-phases/PHASE-12_RELIABILITY_DR_PERFORMANCE_SECURITY.md) | Full game days, load/chaos/restore/security verification |
| 13 — Portfolio release | [PHASE-13](../atlas-prd/02-phases/PHASE-13_PORTFOLIO_RELEASE_EVIDENCE.md) | Reviewer journey, public claims, release evidence and limitations |

## Current implementation navigation

- [Implementation status](IMPLEMENTATION_STATUS.md)
- [Phase 00 audit and execution plan](PHASE-00-PLAN.md)
- [Synthetic local environment and S04 commands](LOCAL_ENVIRONMENT.md)
- [Agent operating guide](../../AGENTS.md)
