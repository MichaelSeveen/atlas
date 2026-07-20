# ADR 0003 — Use a transactional outbox and idempotent consumers

- **Status:** Accepted
- **Date:** 2026-07-20
- **Related requirements:** FND-040, LED-015, ITR-031, REL-001 through REL-005

## Context

Atlas must publish events after durable business changes. Writing the database and broker in separate uncoordinated steps can lose an event or publish a fact whose transaction later rolls back. Distributed transactions add complexity and are not required for the reference architecture.

## Decision

Write an outbox record in the same PostgreSQL transaction as authoritative state. A worker leases unpublished records, publishes to an at-least-once broker, and marks progress. A crash after publish but before acknowledgement can duplicate delivery; consumers must use durable inbox/deduplication and domain idempotency.

## Rules

- Outbox payload is the exact versioned event or stable reference needed to produce it deterministically.
- Publication order is not globally guaranteed.
- Per-aggregate version/partition allows consumers to identify gaps/regressions.
- Consumer handler transaction includes inbox record plus local effect where possible.
- Dead-letter is a diagnostic state, not silent disposal.
- Replay is named, scoped, authorized, audited, and reported.
- Outbox/inbox retention and archive must preserve recovery needs.

## Failure cases explicitly supported

- API dies after business commit before publisher sees row.
- Publisher dies after claiming row.
- Broker accepts event, publisher dies before marking sent.
- Consumer executes then dies before broker acknowledgement.
- Broker outage creates backlog.
- Restore causes old events/checkpoints to reappear.

## Consequences

- Duplicate events are normal and tested.
- Publication is eventually consistent.
- Database outbox growth/claiming must be monitored.
- “Exactly once” is not claimed.

## Rejected alternatives

- Direct broker publish after commit.
- Broker publish before commit.
- XA/distributed transaction for portfolio scope.
- Change-data-capture introduced before outbox needs demonstrate it.

## Verification

Failpoints around commit, claim, publish, and acknowledgement; duplicate/out-of-order/replay consumer tests; backlog recovery; post-restore reconciliation.
