# Product charter

## 1. Product name

**Atlas Wallet & Financial Operations Platform**

## 2. Product thesis

Consumer financial interfaces are only the visible edge of a much larger system. The durable engineering value lies in preserving financial correctness while authentication changes, providers time out, messages duplicate, settlements disagree, staff investigate incidents, and customers dispute outcomes.

Atlas is designed to demonstrate that complete system.

## 3. Problem statement

Most portfolio wallets model a transfer as a row changing from `pending` to `successful` and a balance column decreasing. That design does not answer critical questions:

- What is the source of financial truth?
- Can two concurrent requests spend the same funds?
- What happens when a provider succeeds after the API timed out?
- How is a duplicate webhook handled?
- How does finance prove that provider settlement agrees with internal books?
- How does support explain an outcome without unrestricted access to customer data?
- How are corrections made without rewriting history?
- How are privileged actions controlled and reviewed?

Atlas exists to answer those questions with executable evidence.

## 4. Product objective

Build a multi-tenant financial platform that supports customer wallets, internal and simulated external money movement, merchant collection flows, double-entry accounting, transaction risk controls, settlement, reconciliation, audit, statements, and role-specific operations interfaces.

The product must remain correct under retries, concurrency, partial failure, delayed events, out-of-order events, key rotation, provider disagreement, data restore, and privileged misuse attempts.

## 5. Target users

### Customer

- Completes identity and account setup.
- Holds balances in supported currencies.
- Transfers money internally and to simulated external beneficiaries.
- Sees precise pending, available, completed, reversed, and failed states.
- Downloads statements and raises transaction disputes.

### Merchant developer

- Creates payment intents through a documented API.
- Uses scoped credentials.
- Receives signed webhooks.
- Replays and investigates webhook deliveries.
- Initiates permitted refunds.

### Merchant operator

- Reviews payments, fees, refunds, settlements, and webhook health.
- Exports operational data without exposing unnecessary personal information.

### Risk analyst

- Reviews deterministic risk factors and policy versions.
- Resolves cases using evidence and reasoned decisions.
- Tests rules against historical synthetic data.

### Operations specialist

- Searches transactions and customers under strict authorization.
- Views end-to-end timelines.
- Performs safe, approved compensating actions.
- Records notes and escalations.

### Finance operator

- Imports provider files.
- Runs deterministic reconciliation.
- Investigates breaks.
- Closes settlement periods using maker-checker controls.
- Reviews trial balances and journal exports.

### Security and audit reviewer

- Reviews immutable audit history, access decisions, privileged activity, control evidence, data flows, and incident records.

## 6. Core product capabilities

1. Identity, authentication, session, organization, role, permission, and step-up controls.
2. Customer profile, KYC tier simulation, consent, privacy notices, retention, and data-right workflows.
3. Multi-currency chart of accounts and immutable double-entry journal.
4. Wallet account lifecycle, synchronously maintained balance projection, holds, releases, and captures.
5. Deterministic transaction limits and risk decisions with human review.
6. Internal transfers with concurrency control and idempotency.
7. Simulated external bank transfer provider adapters and ambiguous-outcome recovery.
8. Merchant payment intents, refunds, API credentials, signed requests, and signed webhooks.
9. Settlement batching, provider statement import, matching, exception management, and period close.
10. Role-specific customer, merchant, operations, finance, support, risk, and audit interfaces.
11. Statements, reports, exports, audit evidence, and data-right fulfilment.
12. Observability, SLOs, alerting, backup, point-in-time recovery, chaos testing, security verification, and incident response.

## 7. Product principles

### Correctness before convenience

A delayed but explainable transaction is preferable to a fast, unrecoverable double debit.

### Security is an architectural property

Authentication, authorization, data minimisation, encryption, audit, safe defaults, and recovery are designed before feature implementation.

### Financial history is append-only

The system records what happened, including errors and corrections. It does not rewrite the past to produce a cleaner screen.

### Every state has an owner and explanation

A status must identify the responsible subsystem, permitted next transitions, timeout policy, customer-facing copy, operational action, and financial effect.

### Operations are first-class product users

A system that moves money but cannot investigate, reconcile, or correct outcomes is incomplete.

### Evidence over adjectives

“Secure,” “scalable,” “reliable,” and “enterprise-grade” are forbidden as unsupported descriptions. Each must map to controls, tests, measured targets, or incident exercises.

### Deliberate architecture over service count

The reference architecture is a modular monolith with clear bounded contexts and transactional guarantees. A service is extracted only when a measured scaling, isolation, ownership, or deployment constraint justifies it.

## 8. Success criteria

Atlas succeeds when the repository demonstrates:

- ledger invariants enforced in database and application layers;
- safe money movement under concurrent and duplicate requests;
- explicit handling of provider ambiguity and event duplication;
- separate customer, merchant, workforce, and machine identities;
- least-privilege permissions and maker-checker approval for high-risk actions;
- deterministic and rerunnable reconciliation;
- auditable operational workflows;
- privacy-by-design data handling;
- reproducible restore, chaos, load, and security verification evidence;
- API and event contracts that are understandable without source-code inspection;
- frontend workflows that explain financial state rather than hiding it behind generic success toasts.

## 9. Portfolio positioning statement

> Atlas is a security-first wallet and financial-operations platform built in Go and React. It uses an immutable double-entry ledger, synchronously maintained spendable balances, idempotent payment state machines, transactional outbox delivery, deterministic risk controls, provider simulation, automated reconciliation, and audited operations workflows. The repository includes threat models, architecture decisions, machine-readable contracts, adversarial tests, restore evidence, and reproducible performance results.

## 10. Legal and operational boundary

Atlas is an educational portfolio system. It must not:

- accept real customer funds;
- connect to production banking or card networks;
- store real identity documents or biometric information;
- claim PCI, ISO, SOC, regulatory, or legal compliance;
- present simulated risk outcomes as legally sufficient AML decisions;
- be deployed as a real financial service without licensed partners, legal review, compliance ownership, security assessment, and production operations capability.
