# ADR 0014 — Phase 01 identity and access contracts are server-authoritative and synchronous

- **Status:** Accepted
- **Date:** 2026-07-23
- **Owners:** Identity, security, platform, data, privacy, operations, and audit owners
- **Related requirements/threats:** IAM-001 through IAM-044; SEC-GEN-001 through SEC-GEN-004; SEC-GEN-008 through SEC-GEN-010; AUD-GEN-001 through AUD-GEN-002; API-GEN-001 through API-GEN-004; THR-005 through THR-008; THR-018; THR-020; THR-023; THR-024; THR-028; THR-037 through THR-041; THR-044; THR-053; THR-056 through THR-058; THR-060
- **Supersedes/superseded by:** Specializes ADR 0001, ADR 0004, ADR 0008, ADR 0010, ADR 0012, and ADR 0013 for Phase 01. It does not select a production IdP, secret manager, deployment platform, event pipeline, or alert backend.

## Context

The Phase 01 narrative names identity populations and 30 IAM requirements, but the baseline OpenAPI defined only `GET /v1/me` from the Phase 01 application surface. It did not settle login/callback/logout transport, CSRF, tenant switching, invitation acceptance, approval creation/execution, machine-key authentication, secret-producing retries, exact session policy, or context ownership.

Implementing handlers or schema from those gaps would silently create product semantics outside the canonical contract. It would also risk activating ADR 0013 product-schema and recovery triggers without a closed authorization model.

## Decision

### Ownership

- Identity owns Atlas principals and external subject links, application sessions and assurance/revocation metadata, merchant organizations/memberships/invitations, role/permission bindings and authorization versions, and merchant API credentials.
- Operations owns generic approval requests, immutable payload binding, decisions, and execution coordination.
- Audit owns append-only access/security facts and the synchronous application API used by Identity and Operations.
- `cmd/api` owns BFF/HTTP composition only. It does not own authorization or domain state.
- Keycloak proves authentication and factors in the synthetic reference environment. PostgreSQL, not Keycloak claims or Redis, owns Atlas tenant membership and authorization truth.
- No `internal/authorization` context is created. Authorization remains an Identity application capability used through its root/application API.

The intended schema namespaces are `atlas_identity`, `atlas_operations`, and `atlas_audit`. Their creation remains P01-S03 or the later owning slice and must execute ADR 0013 revalidation.

### Browser authentication and sessions

Customer, merchant, and workforce browsers use OIDC authorization code plus PKCE through the BFF. The browser receives only `__Host-atlas_session`, with `Secure`, `HttpOnly`, `SameSite=Lax`, `Path=/`, and no `Domain`. The opaque value has 256 random bits; only a SHA-256 verifier is durable.

Cookie-authenticated mutations require `X-Atlas-CSRF-Token`. The token is delivered in a same-origin response header, held in browser memory, bound to the session rotation version, and rotated with the session. It is never an authorization input.

Credentialed browser CORS accepts only exact configured origins, emits `Vary: Origin`, permits the
CSRF header and credentials only for those origins, and applies the same headers to preflight,
success, and safe error responses. Wildcard credentialed CORS is forbidden. Same-origin remains
the deployment default.

Idle/absolute lifetimes are:

| Population | Idle | Absolute | Minimum baseline |
|---|---:|---:|---|
| Customer | 30 minutes | 12 hours | baseline |
| Merchant | 20 minutes | 8 hours | baseline |
| Workforce | 10 minutes | 1 hour | phishing-resistant |

Fresh step-up is valid for five minutes. Rotation occurs at login, step-up, privilege change, and tenant switch. The old session has no grace period. Atlas retains no upstream OIDC access, refresh, or ID token after the validated callback. A one-time login transaction may retain only the minimum protected state/nonce/PKCE material until callback or expiry.

At most ten concurrent application sessions may remain active for one principal; creating another
revokes the least-recently-seen session transactionally. OIDC time validation permits at most 60
seconds of configured clock skew and never extends application-session expiry.

During synthetic IdP outage:

- new login and new step-up fail closed;
- existing sessions receive no lifetime extension;
- existing low-risk reads may continue until their normal idle/absolute expiry with ordinary server authorization;
- high-risk actions without already-fresh assurance are denied.

Recovery remains IdP-owned. Atlas callback/error copy, status, and rate behavior are non-enumerating.

### Principals and maker-checker identity

The immutable external key is population + issuer + subject. Merchant and workforce identities also carry a synthetic immutable non-PII person anchor. Multiple external accounts with one anchor map to one Atlas principal. Maker/checker separation compares immutable Atlas principal IDs, not usernames, sessions, roles, or mutable IdP accounts. Self-registration remains disabled.

This is a synthetic portfolio control, not a claim that a real IdP’s identity proofing or employment directory has been selected.

### Tenant and authorization policy

Every tenant row carries tenant ID unless present in a closed global-scope catalogue. Every tenant repository method accepts explicit tenant context. Organization names are display data only: authorization uses opaque organization ID. Names use Unicode NFKC plus case folding, and a UTS #39 skeleton collision is rejected within the caller-visible set so a name cannot create UI authorization ambiguity.

Invitations use 256 random bits, store only SHA-256 verifier state, expire after 72 hours, are single-use, and may grant only roles in the inviter’s closed delegation set. Membership revocation increments authoritative versions in its transaction. No new mutation may commit under that membership after the revocation transaction commits.

Authorization is deny-by-default and evaluates principal, population, tenant, role, permission, resource owner/classification/version, action, field, purpose, assurance, and authorization version. Phase 01 introduces no authorization-decision cache. Every API replica reads authoritative PostgreSQL state/version; Redis cannot authorize. Filtering occurs before totals, cursors, ranks, or suggestions. Privileged and denied high-risk decisions receive decision IDs and required Audit records.

The canonical machine-readable catalogue is `03-contracts/identity-access-policy.json`.

### High-risk and future actions

Current Phase 01 actions in the catalogue require step-up. Future contact, beneficiary, payout, refund, restriction-removal, and privileged-export actions are reserved and deny until their owning phase supplies a typed contract and revalidation. Naming a reserved action does not implement or authorize it.

### Approvals

Operations exposes a typed approval application API. Phase 01 does not accept arbitrary action names or arbitrary JSON payloads. The initial executable type is the typed merchant-administrator membership-role change.

Requested typed payloads use RFC 8785 canonical JSON and SHA-256. The approval stores action type, target/version, hash/version, maker principal, checker policy, expiry, status, and reason code. Default expiry is 24 hours. A decision does not implicitly execute: execution is an explicit synchronous operation.

Checker eligibility is dynamic and re-evaluated at decision and execution; it is not a stale role
snapshot. Execution rechecks maker/checker separation, permission, fresh step-up, target
version/state, payload hash, and policy in the target application transaction. Domain mutation and
Audit record commit atomically. No event, worker, or outbox is introduced by this decision.
Response-loss retries use idempotency and the durable execution result.

### Merchant API credentials

Phase 01 defines Atlas API keys using `Authorization: AtlasKey <key-id>.<secret>`. The secret contains 256 random bits, is displayed only in the initial successful response, and stores only SHA-256 verifier state. Idempotent replay returns redacted metadata, never the secret. Loss of the first response requires replacement/rotation.

Every key is bound to the exact configured environment, tenant, and `atlas-api` audience. Entropy
source failure denies creation/rotation without storing partial credential state.

The current scope is `identity:read`, sufficient to query the authenticated principal context. Later financial scopes remain absent. Default expiry is 90 days and maximum expiry is 365 days. Rotation permits old/new overlap for ten minutes; explicit revocation ends acceptance immediately according to PostgreSQL truth.

Rate counters may use Redis, but authorization, scope, status, expiry, and revocation remain PostgreSQL-owned. Redis outage falls back to a bounded conservative process-local limiter and can never re-enable or authorize a credential.

### Audit and event boundary

Privileged mutations and their Audit facts share one PostgreSQL transaction through the Audit application service. A denied high-risk decision must persist its decision record before the final response; persistence outage keeps the action denied and returns safe service-unavailable behavior.

Phase 01 Audit facts are `confidential-security`, require `audit.events.read`, and are retained
through Phase 01. Phase 11 may pseudonymize subject identifiers under its data-rights policy but
must not delete the security facts or break their controlled internal linkage.

Phase 01 does not publish `audit.event.recorded.v1` and defines no identity/session/member/approval/credential event. The first event, consumer, stream, worker input, or outbox activates ADR 0013 `FND-040` obligations and requires a separate decision with causal propagation, delivery, replay, telemetry, and recovery evidence.

### HTTP contract and generated client

OpenAPI owns the complete application/BFF surface, including login/callback/logout, CSRF, sessions, active organization, invitations, approvals, and credentials. Secret-producing endpoints store a redacted replay representation, overriding ordinary stored-response replay only for the secret field.

The first web product consumer in P01-S09 will use a pinned OpenAPI TypeScript generator executed by Bun against the sole canonical OpenAPI. Generated output is reproducible and never hand-edited. Selecting and pinning the exact generator package remains part of that dependency-changing slice; handwritten parallel schemas are prohibited.

### Cross-phase evidence continuity

The final Phase 00 evidence catalogue remains historical and is not rebound to later work merely to satisfy freshness. Phase 01 uses a new versioned evidence policy and catalogue under `evidence/phase-01/`. Its verifier retains source binding, artifact digests, tamper canaries, and stale-source canaries. A Phase 00 regression may still fail closed on descendant product/contract change; that is a revalidation signal, not permission to rewrite Phase 00 history.

## Alternatives rejected

- Browser access/refresh tokens or readable bearer cookies.
- Keycloak roles/JWT claims as permanent Atlas authorization.
- Redis-backed session revocation or policy truth.
- One identity realm and role model for every actor.
- Arbitrary approval JSON/action names.
- Automatic approval execution hidden inside a decision request.
- Recoverable merchant API secrets or replay that redisplays them.
- Emitting identity/audit events only to make the architecture appear event-driven.
- A new authorization bounded context or direct cross-context persistence imports.
- Rebinding Phase 00 evidence without new owning evidence.

## Consequences and residual risk

- Exact policy is reviewable and machine-validated before schema or handlers.
- Synchronous PostgreSQL checks favor revocation correctness over a premature cache.
- Session/callback response loss may require reauthentication; secret response loss requires rotation. This is safer than secret recovery or session resurrection.
- Unicode/confusable rules require a maintained standards-based implementation and adversarial corpus.
- Audit/database outage denies high-risk operations and can reduce availability.
- The person anchor is synthetic. A real workforce/merchant directory and proofing model remain a real-provider decision and independent-review trigger.
- No runtime requirement is satisfied by this ADR. All IAM rows remain Planned until their owning slice passes real tests/evidence.

## Failure, migration, and rollback

This decision adds no schema, runtime handler, IdP client, credential, session, event, worker input, or financial state. Its failure mode is a contract/policy defect discovered before implementation.

Before runtime implementation, correct the additive contract and policy in a protected PR. After a client consumes the contract, preserve compatibility or version the breaking change. Once S03 creates product state, forward-fix migrations and preserve stronger ADR 0013 recovery controls. Rollback must never restore browser tokens, fail-open authorization, recoverable secrets, or asynchronous behavior without its required controls.

## Verification

- `go run ./cmd/contractctl lint docs/atlas-prd/03-contracts/openapi.yaml docs/atlas-prd/03-contracts/asyncapi.yaml`
- `go test ./tests/contract -count=1`
- `go test ./internal/architecture -count=1`
- `pwsh -NoProfile -File ./scripts/test-p01-s02-contract-canary.ps1`
- `pwsh -NoProfile -File ./scripts/verify-p01-s02.ps1`

The S02 report must state that this is contract/policy evidence only, independent review remains unavailable and unclaimed, and no Phase 01 runtime capability exists.
