# ADR 0016 — Invitation acceptance uses recipient-bound bootstrap authentication

- **Status:** Accepted
- **Date:** 2026-08-08
- **Owners:** Identity, security, platform, privacy, and audit owners
- **Related requirements/threats:** IAM-002; IAM-010 through IAM-015; IAM-020 through IAM-026; SEC-GEN-001; SEC-GEN-002; SEC-GEN-004; SEC-GEN-008; AUD-GEN-001; API-GEN-001 through API-GEN-004; THR-005; THR-006; THR-038; THR-041; THR-044; THR-057; THR-058
- **Supersedes/superseded by:** Additively resolves `P01-D16` and corrects ADR 0014's invitation-acceptance bootstrap omission. It does not weaken ADR 0014's BFF, CSRF, tenant, rotation, authorization, or Audit rules.

## Context

The accepted invitation operation required an authenticated BFF session, while merchant
application sessions required an existing active membership. A first-time invited merchant could
therefore not authenticate before the operation that creates the membership. The request carried
only a bearer invitation token and did not bind it to a verified recipient identity. Treating that
token alone as tenant authority would permit forwarding or theft to grant the invited role.

## Decision

Identity adds `POST /v1/organization-invitations/{invitation_id}/authentication`. It is an
unauthenticated OIDC initiation operation that accepts the invitation token only in a
strict JSON body. The token is never placed in a URL, redirect, log, Audit fact, browser-readable
cookie, or recoverable transaction record. The single-use OIDC transaction stores only its SHA-256
digest and the opaque invitation ID and is bound to the merchant issuer, state, nonce, and PKCE.
Syntactically valid inputs receive the same redirect without revealing invitation existence.

The callback requires the merchant issuer to return `email` with `email_verified=true`. Atlas
normalizes it using the same exact local-part and lower-cased-domain rule used at invitation issue,
then compares only SHA-256 digests in constant time. The callback also rechecks the invitation ID,
token digest, pending state, expiry, and active organization under PostgreSQL locks. Any mismatch
fails as the same authentication error and creates no principal or session.

After those checks, an existing `(population, issuer, subject)` mapping is reused. If none exists,
Atlas may create one synthetic merchant principal and external-subject mapping solely for this
valid invitation flow. This is invitation-gated provisioning, not open self-registration; the
verified email is not stored recoverably and is never used as the external account key.

The callback issues a 15-minute acceptance-only BFF session with no tenant ID, role, permission,
or organization-list authority. Its opaque cookie and CSRF behavior are identical to ordinary BFF
sessions, but the durable session is bound to the invitation ID and verified-email digest. Only
`GET /v1/me`, logout, and the matching invitation-acceptance operation accept this scope. The
current-principal response exposes the Atlas principal ID, null active tenant, baseline assurance,
an empty permission list, and the short expiry; it does not disclose the invitation or email.

Acceptance requires that cookie, its memory-held CSRF token, the same invitation ID and token, and
an idempotency key. In one serializable PostgreSQL transaction Identity locks and rechecks the
principal, acceptance-only session, invitation, token digest, verified-recipient digest, expiry,
organization, and absence of an existing tenant-principal membership; creates the invited
membership; marks the invitation accepted with hash-only replay metadata; rotates to a normal
tenant-bound session with zero grace; and records Audit. No tenant authority exists before that
commit. Audit or database failure rolls back every effect.

An accepted replay by the same principal, verified recipient, token, idempotency key, and request
digest returns the durable membership and original decision ID without repeating the effect or
redisplaying session secret material. A lost rotation response invalidates the old bootstrap
cookie; after reauthentication, the same principal may replay the command or read current durable
state. Reuse by another principal or with different request material fails closed. Missing,
revoked, expired, mismatched-token, and mismatched-recipient cases use the same concealed response
surface. The flow emits no event and performs no external call inside the acceptance transaction.

## Consequences and residual risk

- The invitation token remains necessary but is not sufficient; control of the verified recipient
  identity is also required.
- Email normalization intentionally preserves local-part case. A future broader identity-linking
  policy requires a separate decision and migration; this flow does not merge principals by email.
- The synthetic provider demonstrates the claim and protocol boundary, not production email-proof
  quality, identity recovery, bot defense, or mail delivery.
- A valid but abandoned invitation authentication can leave a principal with no membership. It has
  no tenant authority and is retained under the Phase 01 identity lifecycle until a later data-rights
  policy defines cleanup.
- Independent human review remains unavailable under ADR 0012 and is not claimed.

## Failure, migration, and rollback

Migration 000010 adds hash-only invitation binding to OIDC transactions and sessions, the
acceptance-only session scope, and accepted invitation replay references. It adds nullable columns
and forward-only constraints without rewriting secret material. Rollback disables both invitation
authentication and acceptance routes; it never deletes an accepted membership or Audit fact,
restores an accepted invitation, reactivates a bootstrap session, or relaxes recipient binding.

## Verification

- `go run ./cmd/contractctl lint docs/atlas-prd/03-contracts/openapi.yaml docs/atlas-prd/03-contracts/asyncapi.yaml`
- `go test ./tests/contract -count=1`
- `go test ./internal/identity/... ./cmd/api/internal/server -count=1`
- `go test ./internal/architecture -count=1`
- Configure `ATLAS_P01_DATABASE_URL` and `ATLAS_P01_MIGRATION_DATABASE_URL` for the synthetic
  local PostgreSQL roles, then run `go test ./internal/identity/persistence -count=1`.
