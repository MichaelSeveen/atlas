# API documentation and governance guide

## Contract-first rule

OpenAPI and AsyncAPI are reviewed product artifacts. A feature is incomplete when implementation exists without:

- operation and schema contract;
- authorization and tenant scope;
- idempotency/precondition semantics;
- errors and retryability;
- examples, including adverse examples;
- rate/resource limits;
- audit/observability expectations;
- compatibility assessment;
- contract, integration, and negative tests.

## HTTP conventions

- JSON UTF-8; reject ambiguous duplicate JSON keys in security-sensitive parsers.
- Paths use plural nouns and opaque IDs.
- Commands with complex lifecycle return a durable resource, never an unverifiable success message.
- `POST` money-moving commands require `Idempotency-Key` and canonical request hashing.
- Mutable resources that can suffer lost updates require `ETag` plus `If-Match`.
- `202 Accepted` includes status URL and command/resource ID.
- `Location` identifies newly created resource.
- Cursor pagination is stable, opaque, signed or integrity-protected, and scoped to the original filter/order/tenant.
- Timestamps are RFC 3339 UTC with explicit semantics.
- Amounts are decimal strings in integer minor units plus ISO-like currency code; never JSON number/floating point.
- Dates and reporting boundaries state timezone and inclusive/exclusive semantics.

## Standard headers

### Request

- `Authorization` for machine APIs only; browser uses secure BFF session cookie.
- `Idempotency-Key` on economic commands.
- `If-Match` on guarded mutations.
- `X-Request-Id` optional client correlation input; server validates and may replace unsafe value.
- `traceparent`/`tracestate` for observability only.

### Response

- `X-Request-Id`
- `ETag`
- `Location`
- `Retry-After` where meaningful
- `Deprecation`/`Sunset` and documentation link during retirement
- no internal topology or framework banners.

## Idempotency contract

The deduplication scope includes tenant, authenticated principal/credential, operation, key, and API version. Store:

- canonical request hash;
- command/resource ID;
- original status, headers allowlist, and response body;
- creation and terminal time;
- processing lease/version;
- retention expiry.

Same key + same canonical request returns the original contract response. Same key + different request returns a conflict. The server handles concurrent first use and crash recovery without executing the economic effect twice.

## Pagination contract

- Default and maximum page size are explicit.
- Sort order has deterministic tie-breaker.
- Cursor encodes/integrity-protects tenant, filters, order, last values, schema version, and expiry where needed.
- Authorization filtering occurs before counts and cursor creation.
- Newly inserted rows do not cause duplicates in a snapshot-consistent mode; where live pagination is chosen, duplicate/skip semantics are documented.
- Exports use asynchronous report jobs rather than unbounded list endpoints.

## API evolution

Non-breaking:

- optional response fields;
- optional request fields with safe defaults;
- new resource/operation;
- new error code only where client fallback is defined.

Breaking:

- field removal/rename/type or unit change;
- new required request field;
- enum narrowing or semantic change;
- changed authorization/tenant scope that invalidates clients;
- altered idempotency or state transition semantics;
- changed pagination order/cursor meaning;
- reusing identifiers or error codes.

All breaking changes require ADR, migration guide, compatibility period, telemetry on old-version use, and explicit sunset.

## Documentation depth for each operation

Each operation description answers:

1. Who can call it and in which tenant context?
2. What business preconditions apply?
3. What is committed synchronously?
4. What can remain asynchronous or ambiguous?
5. What is the idempotency scope and retention?
6. Which ledger/hold effects may occur?
7. Which audit event is written?
8. Which domain events are emitted?
9. Which errors are retryable?
10. What client action is safe after timeout?
11. What sensitive fields are masked or omitted?
12. What limits protect the service?

## Generated documentation

Generated reference documentation is allowed, but generated prose is not the source of truth. CI must:

- validate syntax and examples;
- lint design rules;
- detect incompatible changes;
- generate server/client types;
- fail on uncommitted generated drift;
- run conformance tests against a deployed test service;
- publish versioned docs tied to commit/release digest.

## Example quality gate

Every money-moving operation includes examples for:

- success;
- validation rejection;
- insufficient funds/limit;
- duplicate same-key replay;
- same-key different-payload conflict;
- stale precondition where relevant;
- provider ambiguous outcome;
- final status query;
- cross-tenant/object authorization rejection without leakage.
