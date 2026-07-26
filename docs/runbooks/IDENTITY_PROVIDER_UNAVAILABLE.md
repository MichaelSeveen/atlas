# Identity provider unavailable

Scope: Phase 01 synthetic Keycloak OIDC only. Atlas application sessions remain authoritative in
PostgreSQL; Keycloak tokens are never browser session state or durable authorization truth.

## Detection

- `S04-ALR-001` observes a sustained error ratio on bounded discovery/token request metrics.
- Login, callback exchange, and step-up return a safe retryable service-unavailable problem.
- API liveness and existing low-risk application-session checks may remain healthy. Provider
  availability is deliberately outside authoritative readiness.

## Response

1. Confirm `/health/live`, then inspect the `phase-01-identity-session` dashboard by bounded
   operation, population, and outcome. Never inspect or export tokens, codes, cookies, PKCE,
   credentials, or raw provider payloads.
2. Confirm whether discovery, token exchange, or both are failing. Validate all three exact
   synthetic realm discovery documents without changing issuer, client, redirect, or PKCE policy.
3. Block new login and step-up by leaving the fail-closed response in place. Do not bypass fresh
   assurance, mint an Atlas session locally, accept a browser bearer token, or weaken workforce
   assurance.
4. Existing customer/merchant sessions may continue only for the already-authorized low-risk
   session/current-principal operations. No high-risk action may proceed without fresh step-up.
5. Restore the local synthetic provider with `scripts/s04.ps1 -Action Up`, then run
   `scripts/configure-p01-s04-keycloak.ps1` and the three
   `scripts/test-p01-s04-oidc-http.ps1` population probes.
6. If disposable provider state is inconsistent, use the exact guarded local reset only after
   confirming that local synthetic volumes are the intended target.

## Recovery

Require provider metrics to return to normal and rerun
`pwsh -NoProfile -File ./scripts/verify-p01-s04.ps1 -Live`. Confirm no session was issued for a
failed callback and no baseline workforce session was created.

Never bypass realm separation, create a custom IdP, claim production identity readiness, or treat
synthetic Keycloak users as authorization evidence.
