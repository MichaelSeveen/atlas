# Atlas domain event catalogue

## Purpose

This catalogue defines the asynchronous facts emitted after durable state changes. Events are integration contracts—not an escape hatch around domain ownership or database invariants.

## Delivery model

- Transactional outbox written atomically with the authoritative state change.
- At-least-once publication and delivery.
- No global total-order claim.
- Ordering is defined only within an aggregate/partition key and only where the broker configuration supports it.
- Consumers own an inbox/deduplication record keyed by event ID and consumer.
- Event handlers must survive duplicates, redelivery after timeout, replay, and delayed/out-of-order events.
- Financial truth remains in PostgreSQL ledger and business state; broker retention is not a ledger.

## Standard envelope

| Field | Type | Rule |
|---|---|---|
| `spec_version` | string | Envelope contract version, initially `1.0` |
| `event_id` | opaque ID | Globally unique and immutable |
| `event_type` | string | Past-tense fact with major version suffix |
| `occurred_at` | RFC 3339 timestamp | Domain state transition time |
| `recorded_at` | RFC 3339 timestamp | Durable persistence time |
| `producer` | string | Owning module/process |
| `tenant_id` | opaque ID/null | Required for tenant-scoped events |
| `aggregate_type` | string | E.g. `transfer`, `wallet`, `payment_intent` |
| `aggregate_id` | opaque ID | Partition/deduplication context |
| `aggregate_version` | integer | Monotonic optimistic-concurrency version |
| `correlation_id` | opaque ID | End-to-end business flow |
| `causation_id` | opaque ID/null | Command/event that caused this fact |
| `traceparent` | string/null | W3C trace propagation, not authorization |
| `actor` | object | Minimal actor type/ID; no sensitive claims |
| `data` | object | Versioned event payload |

## Classification rules

- Events contain identifiers and non-sensitive snapshots needed for deterministic consumption.
- Never include access tokens, secrets, raw identity documents, full bank account numbers, unrestricted addresses, provider credentials, or unredacted free-form operator notes.
- A downstream consumer that needs sensitive data must make an authorized point-in-time API call and accept that access can be denied.
- Event schemas are compatibility-tested. Fields are added as optional before becoming required in a future major version.

## Core event inventory

### Ledger

| Event | Partition key | Required data | Consumers |
|---|---|---|---|
| `ledger.journal.posted.v1` | journal ID | journal ID, template, business reference, currencies, posting count, posted time | audit, notifications, reporting, verification scheduler |
| `ledger.journal.reversed.v1` | original journal ID | original/reversal IDs, reason category, approval ID | audit, operations, reporting |
| `ledger.period.closed.v1` | book + period | book, period, close type, manifest reference | reporting, audit |
| `ledger.verification.failed.v1` | book + currency | verification run, variance class, affected scope | incident automation only; no public notifications |

`ledger.journal.posted.v1` must not expose unrestricted posting lines. Authorized consumers query the ledger read API or a protected reporting replica.

### Wallets and holds

| Event | Partition key | Required data |
|---|---|---|
| `wallet.created.v1` | wallet ID | owner type/ID, currency, status |
| `wallet.status_changed.v1` | wallet ID | prior/new status, reason category, restriction/case ID |
| `wallet.hold.created.v1` | wallet ID | hold ID, amount, currency, purpose, expiry, business reference |
| `wallet.hold.captured.v1` | wallet ID | hold ID, captured/remainder amounts, journal ID |
| `wallet.hold.released.v1` | wallet ID | hold ID, released amount, reason category |
| `wallet.balance_projection_updated.v1` | wallet ID | balance version and safe totals; internal-only |

### Internal and external transfers

| Event | Partition key | Required data |
|---|---|---|
| `transfer.created.v1` | transfer ID | type, amount, currency, source/destination references, initial state |
| `transfer.risk_decided.v1` | transfer ID | decision, rule-set version, case ID if any; no secret rules |
| `transfer.funds_reserved.v1` | transfer ID | hold ID, amount, expiry |
| `transfer.provider_submitted.v1` | transfer ID | attempt ID, provider alias, submitted time |
| `transfer.status_changed.v1` | transfer ID | previous/new state, reason category, sequence |
| `transfer.completed.v1` | transfer ID | journal IDs, completed time, safe counterparty snapshot |
| `transfer.failed.v1` | transfer ID | failure category, funds disposition |
| `transfer.reversed.v1` | transfer ID | reversal journal, reason, approval/case references |

### Merchant payments and refunds

| Event | Partition key | Required data |
|---|---|---|
| `payment_intent.created.v1` | payment intent ID | merchant, amount, currency, capture method, expiry |
| `payment_intent.status_changed.v1` | payment intent ID | previous/new state, amount capturable/received |
| `payment_intent.captured.v1` | payment intent ID | capture ID, amount, fee, journal IDs |
| `refund.created.v1` | refund ID | payment ID, amount, reason category |
| `refund.status_changed.v1` | refund ID | previous/new state, provider attempt, journal ID |
| `merchant.settlement_obligation_changed.v1` | merchant ID | currency, delta, business reference; internal finance only |

### Provider adapter

| Event | Partition key | Required data |
|---|---|---|
| `provider.attempt.created.v1` | attempt ID | operation, provider, request fingerprint, deadline |
| `provider.attempt.status_changed.v1` | attempt ID | previous/new state, normalized result, provider reference hash |
| `provider.callback.accepted.v1` | provider event ID | provider alias, event type, matched object, signature key ID |
| `provider.callback.rejected.v1` | provider event ID/digest | rejection category only; security channel |
| `provider.outcome.ambiguous.v1` | attempt ID | attempt, next query time, age |

### Webhooks

| Event | Partition key | Required data |
|---|---|---|
| `webhook.delivery.scheduled.v1` | endpoint ID | delivery ID, merchant event ID, endpoint ID, attempt time |
| `webhook.delivery.succeeded.v1` | endpoint ID | delivery ID, HTTP class, latency, signing key ID |
| `webhook.delivery.failed.v1` | endpoint ID | delivery ID, normalized error, next attempt, attempt count |
| `webhook.endpoint.disabled.v1` | endpoint ID | reason category, last failure time |

### Settlement and reconciliation

| Event | Partition key | Required data |
|---|---|---|
| `settlement.file.imported.v1` | provider file ID | digest, provider, period, row count, object reference |
| `settlement.batch.created.v1` | batch ID | provider, currency, gross, fees, net, expected settlement date |
| `settlement.batch.closed.v1` | batch ID | close manifest, approvals, totals |
| `reconciliation.run.completed.v1` | run ID | scope, counts by result, totals, ruleset version |
| `reconciliation.exception.created.v1` | exception ID | class, amount/currency, internal/provider references |
| `reconciliation.exception.resolved.v1` | exception ID | resolution type, journal/approval/case references |

### Risk, compliance, operations, and audit

| Event | Partition key | Required data |
|---|---|---|
| `risk.case.opened.v1` | case ID | subject references, trigger classes, priority |
| `risk.case.decision_recorded.v1` | case ID | decision, reviewer, policy version, reason category |
| `customer.restriction.changed.v1` | customer ID | prior/new restriction, case/approval, effective time |
| `operations.case.created.v1` | case ID | case type, priority, subject references |
| `operations.case.transitioned.v1` | case ID | previous/new state, actor, version |
| `audit.event.recorded.v1` | audit stream partition | audit event ID and classification; internal pipeline |
| `data_rights.request.status_changed.v1` | request ID | request type, previous/new state, due date |
| `statement.generated.v1` | statement ID | period, format set, content digest, object references |

## Compatibility policy

1. `event_type` major version changes only for breaking semantics.
2. Producers never silently change amount units, timestamp meaning, identifier scope, or enum semantics.
3. Consumers ignore unknown optional fields and reject unknown major versions.
4. New enum values are treated as `unknown` unless the consumer explicitly requires fail-closed behaviour.
5. Event fixture corpus contains every historical schema version still replayable.
6. CI validates AsyncAPI, examples, backward compatibility, and that code-generated types have no uncommitted drift.

## Replay policy

- Replays run in a named replay context with initiator, reason, scope, expected consumers, and expiry.
- Consumers distinguish initial delivery from replay only for observability; financial effect remains idempotent.
- Replays cannot bypass authorization to fetch protected data.
- A replay report records source offset/range, emitted count, deduplicated count, failures, and final checkpoints.
- Destructive “reset consumer and replay everything” is prohibited without a rehearsed runbook and isolated verification.

## Event acceptance evidence

For every event type, the implementation must provide:

- schema and example;
- producer transaction boundary;
- partition key;
- data classification;
- consumer list and owner;
- duplicate handling test;
- out-of-order handling test where relevant;
- replay test;
- dead-letter/operator procedure;
- dashboard for publication age and consumer lag.
