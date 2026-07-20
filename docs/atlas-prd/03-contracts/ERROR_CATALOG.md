# Atlas API error catalogue

## Contract

Every non-2xx response uses RFC 9457 Problem Details with `application/problem+json`. The stable machine contract is `type` plus `code`; titles and human detail may evolve. Clients must not branch on prose.

```json
{
  "type": "https://atlas.example/problems/idempotency-conflict",
  "title": "Idempotency key conflict",
  "status": 409,
  "code": "IDEMPOTENCY_KEY_REUSED_WITH_DIFFERENT_REQUEST",
  "detail": "The key was already used with a different request payload.",
  "instance": "/v1/transfers/tr_01J...",
  "request_id": "req_01J...",
  "correlation_id": "cor_01J...",
  "errors": [
    {"pointer": "/amount/amount_minor", "reason": "does_not_match_original_request"}
  ],
  "retryable": false
}
```

## Disclosure rules

- Do not include stack traces, SQL, table names, internal hostnames, provider credentials, raw policy expressions, token claims, sensitive customer data, or existence-revealing detail.
- Authentication and recovery errors use deliberately non-enumerating messages.
- Authorization failures may be returned as `404` where object existence is itself sensitive.
- Provider-specific errors are normalized for clients but preserved in encrypted operational evidence.
- `request_id` is safe for support. Internal execution identifiers are not necessarily client-visible.

## HTTP and retry rules

| Status | Meaning | Client behaviour |
|---|---|---|
| 400 | Malformed syntax or unsupported request shape | Correct request; do not retry unchanged |
| 401 | No valid authenticated principal | Re-authenticate; never retry indefinitely |
| 403 | Authenticated but action not permitted | Do not retry without changed authority/state |
| 404 | Resource absent or intentionally concealed | Do not infer existence |
| 409 | State, concurrency, duplicate, or idempotency conflict | Inspect stable `code`; retry only when documented |
| 412 | Failed `If-Match`/precondition | Re-read resource and make an explicit decision |
| 422 | Semantically invalid command | Correct business input |
| 423 | Account/resource restricted or locked by policy | Follow case/review path |
| 429 | Rate or business-flow limit | Respect `Retry-After`; do not fan out retries |
| 500 | Unclassified server fault | Retry only idempotent requests with bounded backoff |
| 502 | Upstream returned invalid/failure response | Treat financial outcome as potentially ambiguous |
| 503 | Temporarily unavailable | Bounded retry; high-risk flows may require status query |
| 504 | Upstream timeout | Never assume money movement failed; query command status |

## Stable catalogue

### Request and contract

| Code | Status | Retryable | Meaning |
|---|---:|---:|---|
| `REQUEST_MALFORMED` | 400 | No | Body/query/header cannot be parsed |
| `REQUEST_VALIDATION_FAILED` | 422 | No | One or more semantic fields invalid |
| `UNSUPPORTED_MEDIA_TYPE` | 415 | No | Content type not accepted |
| `UNSUPPORTED_API_VERSION` | 400 | No | Requested version unavailable |
| `REQUEST_TOO_LARGE` | 413 | No | Body or upload exceeds explicit limit |
| `PRECONDITION_REQUIRED` | 428 | No | Sensitive mutation requires `If-Match` |
| `PRECONDITION_FAILED` | 412 | No | Supplied resource version is stale |

### Authentication, session, and authorization

| Code | Status | Retryable | Meaning |
|---|---:|---:|---|
| `AUTHENTICATION_REQUIRED` | 401 | No | No valid session or machine credential |
| `SESSION_EXPIRED` | 401 | No | Absolute/idle lifetime elapsed |
| `SESSION_REVOKED` | 401 | No | Session explicitly invalidated |
| `STEP_UP_REQUIRED` | 403 | No | Higher assurance required; challenge metadata may be returned |
| `CSRF_VALIDATION_FAILED` | 403 | No | Browser mutation failed anti-CSRF control |
| `INSUFFICIENT_SCOPE` | 403 | No | Machine credential lacks scope |
| `ACTION_NOT_AUTHORIZED` | 403 | No | Policy denied action |
| `TENANT_CONTEXT_INVALID` | 403 | No | Principal cannot act in selected tenant |
| `MAKER_CANNOT_APPROVE` | 403 | No | Request creator cannot approve own request |
| `APPROVAL_REQUIRED` | 409 | No | Action is staged pending checker decision |
| `APPROVAL_EXPIRED` | 409 | No | Approval no longer executable |
| `APPROVAL_PAYLOAD_CHANGED` | 409 | No | Payload hash differs; create a new request |

### Idempotency and duplicate control

| Code | Status | Retryable | Meaning |
|---|---:|---:|---|
| `IDEMPOTENCY_KEY_REQUIRED` | 400 | No | Endpoint requires a key |
| `IDEMPOTENCY_KEY_INVALID` | 400 | No | Key fails format/length policy |
| `IDEMPOTENCY_REQUEST_IN_PROGRESS` | 409 | Yes | Original request has not reached a terminal response |
| `IDEMPOTENCY_KEY_REUSED_WITH_DIFFERENT_REQUEST` | 409 | No | Same key, different canonical request hash |
| `DUPLICATE_BUSINESS_REFERENCE` | 409 | No | Economic command already exists |
| `DUPLICATE_PROVIDER_REFERENCE` | 409 | No | Provider event/reference already consumed |

### Customer, KYC, restrictions, and risk

| Code | Status | Retryable | Meaning |
|---|---:|---:|---|
| `CUSTOMER_NOT_ELIGIBLE` | 422 | No | Lifecycle/KYC state disallows feature |
| `KYC_REQUIRED` | 403 | No | Required tier not met |
| `KYC_REVIEW_PENDING` | 409 | Yes | Review not complete |
| `ACCOUNT_RESTRICTED` | 423 | No | Restriction blocks action |
| `TRANSACTION_LIMIT_EXCEEDED` | 422 | No | Amount/count/balance limit exceeded |
| `RISK_REVIEW_REQUIRED` | 409 | No | Case created; command is not silently accepted |
| `RISK_POLICY_DECLINED` | 422 | No | Deterministic policy declined command |
| `BENEFICIARY_COOLING_OFF` | 409 | Yes | Activation delay has not elapsed |
| `BENEFICIARY_NOT_ACTIVE` | 422 | No | Beneficiary unavailable for transfer |

### Ledger, wallet, and money

| Code | Status | Retryable | Meaning |
|---|---:|---:|---|
| `CURRENCY_UNSUPPORTED` | 422 | No | Currency absent from allowed catalog |
| `AMOUNT_INVALID` | 422 | No | Zero, negative, malformed, or exceeds bounds |
| `CURRENCY_MISMATCH` | 422 | No | Account/request currencies do not match |
| `INSUFFICIENT_AVAILABLE_BALANCE` | 422 | No | Available funds are insufficient |
| `WALLET_FROZEN` | 423 | No | Wallet cannot initiate transaction |
| `HOLD_NOT_ACTIVE` | 409 | No | Hold already captured/released/expired |
| `HOLD_AMOUNT_EXCEEDED` | 422 | No | Capture exceeds remaining reservation |
| `LEDGER_TEMPLATE_VIOLATION` | 500 | No | Internal posting template violated an invariant |
| `LEDGER_ACCOUNT_CLOSED` | 409 | No | Account cannot receive ordinary postings |
| `LEDGER_PERIOD_CLOSED` | 409 | No | Effective period rejects posting |
| `LEDGER_REVERSAL_NOT_PERMITTED` | 409 | No | Lifecycle/accounting policy blocks reversal |
| `FINANCIAL_INVARIANT_UNAVAILABLE` | 503 | Yes | Protective circuit blocks money movement after integrity alert |

### Transfers and payment lifecycle

| Code | Status | Retryable | Meaning |
|---|---:|---:|---|
| `TRANSFER_STATE_CONFLICT` | 409 | No | Transition invalid for current state |
| `TRANSFER_OUTCOME_UNKNOWN` | 202 | Yes | External outcome ambiguous; use status endpoint |
| `TRANSFER_ALREADY_TERMINAL` | 409 | No | Terminal command cannot be repeated differently |
| `QUOTE_EXPIRED` | 409 | No | Obtain a new quote |
| `QUOTE_MISMATCH` | 409 | No | Quote does not match command/beneficiary |
| `PAYMENT_INTENT_STATE_CONFLICT` | 409 | No | Operation invalid in current state |
| `CAPTURE_AMOUNT_EXCEEDED` | 422 | No | Capture exceeds capturable amount |
| `REFUND_AMOUNT_EXCEEDED` | 422 | No | Refund exceeds refundable amount |
| `REFUND_STATE_CONFLICT` | 409 | No | Refund already terminal or conflicting |

### Provider and asynchronous processing

| Code | Status | Retryable | Meaning |
|---|---:|---:|---|
| `PROVIDER_UNAVAILABLE` | 503 | Yes | Adapter cannot accept new work |
| `PROVIDER_TIMEOUT_AMBIGUOUS` | 504 | Yes | Provider outcome unknown; no blind alternate charge |
| `PROVIDER_RESPONSE_INVALID` | 502 | Yes | Response failed contract validation |
| `PROVIDER_CALLBACK_SIGNATURE_INVALID` | 401 | No | Callback authentication failed |
| `PROVIDER_CALLBACK_REPLAYED` | 409 | No | Replay window/reference check failed |
| `PROVIDER_EVENT_OUT_OF_ORDER` | 202 | Yes | Event retained/reconciled under ordering policy |
| `ASYNC_COMMAND_ACCEPTED` | 202 | Yes | Command accepted and status resource returned |
| `DEPENDENCY_DEGRADED` | 503 | Yes | Required dependency unavailable under policy |

### Merchant API and webhooks

| Code | Status | Retryable | Meaning |
|---|---:|---:|---|
| `API_CREDENTIAL_REVOKED` | 401 | No | Credential no longer valid |
| `WEBHOOK_ENDPOINT_UNVERIFIED` | 409 | No | Endpoint must complete challenge |
| `WEBHOOK_DESTINATION_NOT_ALLOWED` | 422 | No | SSRF/network destination policy rejected URL |
| `WEBHOOK_REPLAY_NOT_ALLOWED` | 409 | No | Delivery is ineligible or already replayed under policy |
| `WEBHOOK_RATE_LIMITED` | 429 | Yes | Endpoint delivery pressure exceeded |
| `SIGNATURE_VERSION_UNSUPPORTED` | 400 | No | Verification version not supported |

### Settlement, reconciliation, reports, and exports

| Code | Status | Retryable | Meaning |
|---|---:|---:|---|
| `SETTLEMENT_FILE_ALREADY_IMPORTED` | 409 | No | Exact digest/provider identity already ingested |
| `SETTLEMENT_FILE_SCHEMA_INVALID` | 422 | No | File cannot be normalized safely |
| `RECONCILIATION_RUN_IN_PROGRESS` | 409 | Yes | Existing run owns scope |
| `RECONCILIATION_EXCEPTION_UNRESOLVED` | 409 | No | Period/batch cannot close |
| `REPORT_NOT_READY` | 409 | Yes | Asynchronous report still processing |
| `EXPORT_EXPIRED` | 410 | No | Generate a new short-lived grant |
| `EXPORT_SCOPE_DENIED` | 403 | No | Requested fields exceed permission/purpose |
| `CSV_FORMULA_SANITIZED` | 200 | No | Optional warning metadata; dangerous formula prefix escaped |

## Financial ambiguity contract

For a command that may have reached an external rail, the API does not return a definitive failure solely because its network call timed out. It returns either:

- `202` with a durable command resource in `pending_provider_confirmation`/`outcome_unknown`; or
- a `504` Problem Details response that includes a safe `command_id` and status URL.

Clients must query the status resource using the original command identifier and must not create a new command with a different idempotency key.

## Versioning

- Additive codes may be introduced without a major API version.
- Existing code meaning, status, and retryability are compatibility commitments.
- Removal or semantic reuse requires a breaking contract version.
- Internal provider codes never become permanent public contract by accident.
