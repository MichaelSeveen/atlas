# ADR 0004 — Browser applications use a backend-for-frontend session

- **Status:** Accepted
- **Date:** 2026-07-20
- **Related requirements:** IAM-001 through IAM-007

## Context

React/Vue browser clients need secure authentication and step-up flows. Storing bearer access/refresh tokens in JavaScript-readable storage increases exposure to XSS and complicates revocation, tenant context, and token handling. Atlas also serves machine integrations with different credential needs.

## Decision

Customer, merchant-workforce, and platform-workforce browser applications use a backend-for-frontend (BFF):

- OAuth/OIDC protocol and tokens are handled server-side.
- Browser receives a Secure, HttpOnly, host-bound, SameSite session cookie.
- Cookie-authenticated mutations require CSRF protection.
- Session state records identity population, tenant, assurance, authentication time, idle/absolute expiry, and revocation version.
- Session rotates at login, step-up, privilege change, and tenant switch.
- High-risk actions require recent step-up and server-side policy.

Merchant/server integrations use scoped machine credentials/tokens with audience, expiry, rotation, and revocation; they do not reuse browser session semantics.

## Consequences

- Browser JavaScript cannot directly read OAuth tokens.
- BFF is a security-critical boundary and must scale/recover with the application.
- CSRF and session-store/validation controls are mandatory.
- Frontend navigation must handle session expiry and return-to-flow without duplicating commands.

## Rejected alternatives

- Access/refresh tokens in localStorage/sessionStorage.
- A single authentication realm and role model for every actor.
- UI-only step-up flags.

## Verification

Browser storage inspection, XSS/CSRF tests, session fixation/rotation/revocation, issuer/audience/nonce/PKCE checks, role/tenant change while tabs are open, step-up expiry at submission.
