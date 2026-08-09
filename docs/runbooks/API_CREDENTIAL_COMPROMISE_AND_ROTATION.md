# API credential compromise and rotation

## Scope and safety boundary

Use this runbook for a suspected copied merchant API credential, a new-network anomaly, sustained rate rejection, Redis rate-counter degradation, or a rotation/revocation failure. PostgreSQL is the sole credential validity, scope, tenant, environment, audience, expiry, and revocation authority. Redis and process-local counters never authorize a request.

Do not query, log, email, paste, or attempt to recover a credential secret. Atlas stores only a SHA-256 verifier and a display suffix. Do not edit credential or Audit tables directly, widen scopes, extend an overlap, restore a revoked credential, or bypass action-bound step-up.

## Triage

1. Record the alert identifier, time window, environment, source revision, and the bounded telemetry outcome. Do not copy raw `Authorization` values or source addresses.
2. Confirm API and PostgreSQL readiness. If Redis is unavailable, verify `atlas.identity.api_credential.rate_fallback.count` is increasing and that the stricter local policy is rejecting excess traffic.
3. Ask the merchant administrator to identify the credential from its name, key ID, and non-secret suffix in the credential list. Never request the secret.
4. Inspect the credential lifecycle and synchronous Audit chain through application/operator read paths. Confirm tenant, environment, status, expiry, rotation links, last-use time, and anomaly flag.

## Containment

1. For credible compromise, use the authenticated credential revocation operation with exact action-bound phishing-resistant step-up, purpose `credential_management`, CSRF, idempotency, and correlation headers.
2. If continuity is required and compromise is not established, rotate once. Deliver the new secret only through the original response and confirm the operator stored it in the intended synthetic client. The old credential remains valid for no more than ten minutes.
3. Explicitly revoke the old key as soon as the new client succeeds. Do not wait for overlap expiry after suspected compromise.
4. If Audit persistence is unavailable, stop: the lifecycle mutation must roll back. Follow `AUDIT_PERSISTENCE_UNAVAILABLE.md` and retry with the same idempotency key after recovery.

## Recovery and verification

1. Verify the revoked credential receives the generic authentication failure and cannot be re-enabled by Redis recovery or stale counters.
2. Verify the replacement accepts only `identity:read` for audience `atlas-api` in the exact configured environment.
3. Verify list metadata exposes no verifier, full secret, or raw network address. Confirm replay of create/rotate returns `secret: null` and `secret_disclosed: false`.
4. Review bounded authentication, anomaly, rejection, and fallback metrics. Credential, tenant, principal, request, and network identifiers must not appear in labels or logs.
5. Run `pwsh -NoProfile -File ./scripts/verify-p01-s08.ps1 -Live -ContainerRuntime podman` and retain sanitized revision-bound evidence.

## Escalation and follow-up

Escalate to the security and identity owners when any new-network anomaly is unexplained, multiple credentials are affected, revocation does not take effect at the PostgreSQL commit boundary, or protected data may have been read. Rotate rather than recover. Record timeline, durable decision/Audit identifiers, affected synthetic scopes, verification result, and follow-up control changes without including secrets or raw network data.
