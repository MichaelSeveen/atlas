# ADR 0015 — Administrator session revocation is a separate workforce security command

- **Status:** Accepted
- **Date:** 2026-07-27
- **Owners:** Identity, security, platform, and audit owners
- **Related requirements/threats:** IAM-004; IAM-006; IAM-007; IAM-024; SEC-GEN-001; SEC-GEN-004; SEC-GEN-008; AUD-GEN-001; API-GEN-001; THR-006; THR-007; THR-018; THR-037; THR-058
- **Supersedes/superseded by:** Additively corrects the administrator-revocation omission in ADR 0014 and its OpenAPI surface. It does not weaken or replace ADR 0014’s self-session, authorization, step-up, tenancy, or audit rules.

## Context

`IAM-004` requires administrator security revocation with audit. ADR 0014 declared the Phase 01
application surface complete, but its only single-session route,
`DELETE /v1/sessions/{session_id}`, is explicitly restricted to a session owned by the current
principal. Reusing that route would silently change its ownership and concealment semantics.
Implementing an uncontracted administrator path would violate the canonical-contract boundary.

## Decision

Identity owns a distinct additive command:
`POST /v1/security/sessions/{session_id}/revocations`. It accepts only the closed
`security_review` purpose and one of `compromised_session`, `suspected_account_takeover`, or
`workforce_security_response`. The command may target customer, merchant, or workforce
application sessions. It neither accepts arbitrary text nor creates a general-purpose support
search surface.

The actor must be an active workforce principal with
`identity.sessions.revoke_admin`, phishing-resistant assurance, and a step-up completed no more
than five minutes earlier for the exact action `identity.session.admin_revoke`. CSRF and a valid
idempotency key are mandatory. Permission, actor session, assurance binding, step-up freshness,
and authorization version are re-read under PostgreSQL locks before commit; no authorization
cache or IdP role claim participates.

Target lookup occurs only after actor authorization. A missing target returns
not-found-or-concealed. An already revoked target is a successful no-op, but the original
authorized command still receives a durable decision and Audit fact. Idempotent replay returns
the original outcome and decision ID; reuse with a different target, purpose, or reason conflicts.

Every authorized execution, permission denial, step-up denial, and authorized concealed-target
decision has an opaque authorization decision ID. The authorization result, any session
revocation, the append-only Audit fact, and the idempotency replay fact commit in one PostgreSQL
transaction. If Audit or PostgreSQL persistence fails, no revocation commits and the API returns
service unavailable. The route publishes no event and creates no worker or broker obligation.

## Consequences and residual risk

- Owner self-revocation semantics remain unchanged and cannot acquire cross-principal authority.
- Global workforce security can revoke a session across tenant boundaries, but receives no
  session-discovery or sensitive-field response from this command.
- A new step-up session and new idempotency key are required after a stale-step-up denial.
- The synthetic reference IdP is evidence of protocol handling, not a production workforce
  identity or phishing-resistant authenticator selection.
- Independent review remains unavailable under ADR 0012 and is not claimed.

## Failure, migration, and rollback

Migration 000008 adds nullable action-bound step-up evidence to application sessions and a
hash-only administrator-revocation replay table. It does not rewrite existing sessions; those
sessions have no action binding and therefore fail closed for this command. Released migrations
are forward-fixed. Rollback disables the route before a corrective migration and never restores
a revoked session, removes an Audit fact, or relaxes permission, purpose, step-up, and
idempotency checks.

## Verification

- `go run ./cmd/contractctl lint docs/atlas-prd/03-contracts/openapi.yaml docs/atlas-prd/03-contracts/asyncapi.yaml`
- `go test ./tests/contract -count=1`
- `go test ./internal/identity/... ./cmd/api/internal/server -count=1`
- `go test ./internal/architecture -count=1`
- `pwsh -NoProfile -File ./scripts/verify-p01-s04.ps1 -Live -ContainerRuntime podman`
