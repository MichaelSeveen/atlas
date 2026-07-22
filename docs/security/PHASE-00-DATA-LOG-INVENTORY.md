# Phase 00 data and log inventory

This inventory implements the Phase 00 facet of FND-051. Product fields do not exist yet and must be inventoried before later-phase implementation.

## Classification baseline

| Class | Phase 00 meaning | Examples | Sink rule |
|---|---|---|---|
| Public | Intended for public display | Contract version, documented operational route names | May be returned by the documented interface |
| Internal | Operational metadata whose disclosure is low risk but unintended | Module, event, outcome, status, bounded route/method, build revision/time | Structured sinks only; no free-form copies |
| Confidential-pseudonymous | Linkable operational identifier with no direct identity semantics | Request, correlation, and trace IDs | Logs/traces only, 30-day foundation target, never metric labels |
| Restricted | Credentials, key material, personal data, financial data, raw payloads | Local generated passwords and future product data | Forbidden from Phase 00 logs, traces, metrics, errors, fixtures, browser storage, and evidence |

The retention values below are engineering targets for the reference environment, not legal-retention conclusions. A later legal/purpose review is required before product data exists.

## Allowed structured-log fields

The executable policy is `internal/platform/logging.FieldPolicies`. The only accepted fields are timestamp, stable event, severity, module, outcome, bounded HTTP method/route/status, request/correlation/trace IDs, and source revision. Internal fields target 30 days except revision metadata at 90 days; pseudonymous IDs target 30 days.

There is deliberately no free-form message, error, URL, query, header, body, payload, actor, tenant, account, email, name, credential, or key field. Actor and tenant fields will be added only after Phase 01 supplies typed pseudonymous identifiers and authorization context.

## Telemetry inventory

| Signal | Allowed content | Forbidden content | Retention/export |
|---|---|---|---|
| Logs | Closed fields described above | Raw errors, headers, bodies, queries, secrets, PII, financial metadata | JSON stdout; sink retention must enforce the field policy |
| Traces | Stable span names; bounded route/method/outcome/status; validated request/correlation IDs; PostgreSQL system/operation/namespace | SQL text, connection strings, credentials, query parameters, actor/customer/product values | OTLP collector; 30-day foundation target |
| Metrics | Only labels in `deploy/observability/catalog.json` | Request/correlation/trace/actor/tenant/user/account/email identifiers; raw routes; arbitrary error strings | OTLP collector; aggregate operational retention |
| HTTP problems | Stable public type/title/status/code, pseudonymous request/correlation IDs, retryable flag | Internal error text, dependency address, stack, schema, SQL, credential | Response only; `Cache-Control: no-store` |
| Evidence | Revision, command, seed, expected/observed result, digest, limitation, sanitized collector excerpts | Environment files, generated passwords, local key material, personal data | Versioned append-only repository artifacts |

## Source-redaction and injection policy

Values are rejected before encoding if they are outside a closed allowlist or strict identifier grammar. JSON encoding provides a single record boundary, and a rejected record writes zero bytes. Redaction does not depend on a downstream regex. The S06 named skipped test injects CRLF plus a forged JSON event and verifies that the sink receives no forged record. Evidence scanning uses synthetic canary strings only.
