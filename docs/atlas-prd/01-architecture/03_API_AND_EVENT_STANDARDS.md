# API and event standards

## 1. Contract posture

The API and event contracts are product artifacts. A reviewer should understand supported workflows, security boundaries, errors, idempotency, pagination, and lifecycle states without reading Go handlers.

- HTTP contract: OpenAPI, source-controlled and linted.
- Event contract: AsyncAPI, source-controlled and linted.
- Examples are executable in contract tests.
- Breaking changes require versioning and migration notes.

## 2. Resource conventions

- Base path: `/v1`.
- Resource names are plural nouns.
- Actions use subresources only when not naturally represented by resource state, e.g. `/refunds`, `/holds/{id}/release`, `/approvals/{id}/decisions`.
- IDs are opaque and never encode tenant or PII.
- Timestamps are UTC RFC 3339 with explicit offset.
- Monetary amounts are decimal strings representing integer minor units plus ISO currency code.
- Enums are documented and unknown future values are handled safely by clients.

## 3. Authentication surfaces

| Surface | Authentication | Session/token posture |
|---|---|---|
| Customer web | OIDC through BFF | Secure HttpOnly session cookie |
| Merchant dashboard | OIDC through BFF | organization-scoped session |
| Workforce console | separate OIDC client/realm | short session, MFA, step-up |
| Merchant server API | OAuth client credentials or signed scoped key | no browser use |
| Provider callback | signed request | timestamp, key ID, event replay check |
| Internal job | workload identity | least-privilege service scope |

## 4. Idempotency protocol

Money-moving POST endpoints require `Idempotency-Key`.

Server scope:

```text
principal + tenant + HTTP method + canonical route + idempotency key
```

The stored record includes:

- normalized request hash;
- lifecycle: `processing`, `completed`, `failed_retryable`, `failed_final`;
- resource identifier;
- response status and canonical body;
- expiry;
- correlation ID.

Rules:

- same key and same normalized request returns the original response;
- same key and different request returns conflict;
- concurrent same-key requests wait, poll, or receive a documented in-progress response without duplicating effects;
- timeout after commit remains recoverable through the key;
- provider idempotency is separate from public API idempotency;
- retention is long enough for realistic client retries and documented per endpoint.

## 5. Error model

Use problem details with:

- `type` stable URI identifier;
- `title`;
- `status`;
- `detail` safe for the caller;
- `instance` request-specific reference;
- `code` stable machine code;
- `request_id`;
- optional field errors;
- optional `retryable`, `retry_after`, and `current_state`.

Never expose stack traces, SQL, provider secrets, internal account IDs, or authorization-policy internals.

Examples of stable codes:

- `insufficient_available_funds`;
- `idempotency_key_reused`;
- `step_up_required`;
- `account_restricted`;
- `quote_expired`;
- `transition_not_allowed`;
- `provider_outcome_pending`;
- `approval_required`;
- `reconciliation_period_closed`.

## 6. Optimistic concurrency

Configuration, cases, notes, rules, and approval records use resource versions or ETags where lost updates matter.

Financial posting does not rely solely on frontend-supplied versions; server-side locks and constraints remain authoritative.

## 7. Pagination and filtering

- Cursor pagination for transaction, audit, webhook, case, and reconciliation streams.
- Stable ordering by `(occurred_at, id)` or domain sequence.
- Maximum page size enforced.
- Filters are allowlisted and indexed.
- Export is an asynchronous job, not an unbounded page size.
- Search results respect field masking and tenant scope.

## 8. API compatibility

- Additive optional fields are permitted.
- Removing, renaming, changing type, narrowing enum behaviour, or changing semantics is breaking.
- New enum values require tolerant clients or a version bump.
- Deprecations include date, replacement, telemetry, and migration guide.
- CI compares contracts against the released baseline.

## 9. Event envelope

Every domain event includes:

```json
{
  "event_id": "...",
  "event_type": "transfer.status_changed",
  "event_version": 1,
  "occurred_at": "...",
  "producer": "atlas.transfer",
  "tenant_id": "...",
  "subject_id": "...",
  "correlation_id": "...",
  "causation_id": "...",
  "traceparent": "...",
  "data": {}
}
```

Rules:

- event IDs are globally unique;
- event schemas are immutable within a version;
- delivery is at-least-once;
- consumers store inbox records transactionally with effects;
- event payloads contain minimum necessary data and avoid secret or excessive PII;
- events describe facts in past tense;
- commands are not disguised as events;
- ordering assumptions are explicit per key or absent;
- consumers handle unknown optional fields.

## 10. Outbox relay

- Domain state and outbox record commit together.
- Relay claims rows with safe concurrent worker semantics.
- Publish acknowledgement and row marking are not assumed atomic; duplicates are expected.
- Poison messages move to visible quarantine after bounded attempts.
- Outbox lag and oldest unprocessed age are monitored.
- Schema validation occurs before publish.

## 11. Webhook contract

Merchant webhooks are external event delivery, not the internal event bus exposed directly.

They have:

- curated payloads;
- stable public event names;
- signature headers;
- retry schedule;
- endpoint disable policy;
- replay tooling;
- ordering disclaimer;
- event retrieval API so a merchant can verify truth independently.

## 12. Documentation quality gate

Each operation documents:

- purpose and actor;
- permission;
- preconditions;
- state transitions;
- financial effect;
- idempotency;
- errors;
- retry advice;
- examples;
- audit effect;
- webhook/event effect;
- rate-limit class.
