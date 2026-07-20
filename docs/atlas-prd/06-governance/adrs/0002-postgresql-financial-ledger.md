# ADR 0002 — PostgreSQL is the financial system of record

- **Status:** Accepted
- **Date:** 2026-07-20
- **Related requirements:** LED-001 through LED-053, FND-060 through FND-064

## Context

Atlas needs atomic, durable, constraint-backed posting; concurrency control; immutable history; efficient financial queries; backup/PITR; and a system that can be independently verified. A portfolio project must show these properties directly rather than delegate correctness to a black-box balance API.

## Decision

Use PostgreSQL as the authoritative store for:

- chart of accounts;
- journals and postings;
- synchronous account-balance projection;
- holds/reservations and financial command states;
- idempotency and business uniqueness;
- transaction outbox;
- reconciliation and accounting-period controls.

Each journal balances per currency, uses positive integer minor-unit postings, and updates its projection atomically. Posted rows are not mutable by application roles. Corrections are linked compensating journals.

## Transaction strategy

- Explicit transactions with bounded timeouts.
- Deterministic account lock ordering.
- Row locks and/or serializable isolation selected per command with whole-command bounded retry.
- Unique constraints for economic identity and provider references.
- Database constraints/permissions complement application validation.
- Independent rebuild uses postings as source and compares exact projections.

## Not decided by this ADR

- Exact partitioning; it requires measured need.
- Whether reporting uses a replica/read model later.
- Specific managed PostgreSQL vendor.

## Consequences

### Positive

- Local atomicity covers journal, balance, state, idempotency, and outbox.
- Strong constraints and role permissions provide defense in depth.
- Backup/restore and SQL inspection are demonstrable.

### Negative

- Hot accounts can create contention.
- Schema/migration discipline is critical.
- Cross-region active-active financial writes are not provided by default.

## Rejected alternatives

- **Redis balance as truth:** lacks the required durable accounting history and recovery model.
- **Event broker as ledger:** broker retention/order does not replace accounting constraints and queries.
- **Blockchain:** does not solve private authorization, accounting model, reconciliation, or operational correction needs.
- **Float current-balance column only:** cannot prove history or exactness.

## Verification

- Property/model/mutation tests.
- Application DB role update/delete denial.
- Concurrency and commit-response-loss tests.
- Independent projection and trial balance verification.
- Isolated point-in-time restore with full reconciliation.
