# ADR 0001 — Start with a boundary-enforced modular monolith

- **Status:** Accepted
- **Date:** 2026-07-20
- **Decision owners:** Architecture and platform
- **Related requirements:** FND-001, FND-002, FND-003

## Context

Atlas contains multiple domains—identity integration, customer/KYC, ledger, wallets, risk, transfers, merchant payments, providers, settlement, reconciliation, cases, reporting, and audit. Splitting them into independently deployed services at the start would introduce network failure, distributed transaction, event-version, deployment, observability, secret, and operational burdens before domain invariants are proven.

A single unstructured codebase would also be weak: it would permit arbitrary imports, database access, shared models, and bypass of ledger/authorization boundaries.

## Decision

Build a Go modular monolith with separate deployable process entry points:

- `api` for synchronous commands/queries;
- `worker` for outbox, provider, webhook, reporting, and scheduled jobs;
- `simulator` for deterministic external-provider scenarios;
- optional migration/admin binaries with narrowly controlled credentials.

Each domain owns its package, schema namespace/table set, public application interfaces, errors, and events. Cross-module calls use explicit interfaces/application services. Static dependency tests reject forbidden imports. Only the owning module writes its authoritative tables.

## Required boundaries

- Ledger tables are writable only through the ledger application/database role path.
- Identity/authorization context is passed explicitly; repositories cannot infer tenant from global state.
- Cross-domain read models are separate from write ownership.
- No shared “models” package containing mutable domain entities.
- No module imports another module’s internal persistence package.
- Outbox/event contracts are versioned even when producer and consumer deploy together.

## Extraction criteria

A module may become a service only when evidence shows at least one of:

- materially different scaling/resource isolation needs;
- independent security/trust boundary;
- independent availability/release cadence with clear ownership;
- technology constraint impossible to meet in-process;
- demonstrated organizational need.

Extraction requires an ADR covering data ownership, consistency, failure, compatibility, observability, deployment, and recovery.

## Consequences

### Positive

- Financial transactions can remain local and atomic.
- Domain boundaries are visible and testable.
- Local development and failure reproduction are simpler.
- Reviewers can inspect architecture judgement without microservice theatre.

### Negative

- A bad module can still affect process resources.
- Deployment cadence is shared.
- Boundary discipline depends on static checks, reviews, DB roles, and ownership.

## Rejected alternatives

- **Microservices from day one:** rejected because it expands failure/operations surface without proven need.
- **Single layered CRUD application:** rejected because domain ownership and financial controls would be bypassable.
- **Serverless function per endpoint:** rejected because transaction/state/worker cohesion and local reproducibility would suffer.

## Verification

- Forbidden-import test.
- Schema/write-ownership database role tests.
- Architecture dependency graph in CI.
- One end-to-end transfer trace proving boundaries.
