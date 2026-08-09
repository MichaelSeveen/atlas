# P01-S08 merchant API credential closure — pre-commit evidence

- **Evidence ID:** `EVD-P01-S08-CREDENTIAL-CLOSURE`
- **Phase/slice:** `PHASE-01_IDENTITY_ACCESS_TENANCY` / `P01-S08`
- **Source:** `UNCOMMITTED_WORKTREE(base=cdff61932a007bc3953e2429f649c4643d452f96)`
- **Environment:** isolated local Podman PostgreSQL/Redis stack plus repository-local Go caches
- **Determinism:** synthetic principals, injectable UTC clocks, fixed policy dimensions, and bounded concurrent actors
- **Reproduce:** `pwsh -NoProfile -File ./scripts/verify-p01-s08.ps1 -Live -ContainerRuntime podman`
- **Observed result:** `PASS`
- **Revalidate by:** 2026-11-09

## Boundary and decisions

Identity owns credential metadata, non-recoverable verifier state, lifecycle, authentication, and
the PostgreSQL mutation-replay records. The API owns strict HTTP parsing and machine-principal
composition. Audit records authorized and denied lifecycle attempts in the caller transaction.
Redis stores only reconstructible abuse counters; PostgreSQL is consulted for credential status,
tenant, environment, audience, scope, expiry, verifier algorithm/version, and revocation on every
authentication attempt.

The sole Phase 01 machine scheme is `Authorization: AtlasKey <key-id>.<secret>`. A secret contains
256 random bits encoded as unpadded base64url. Only its SHA-256 verifier and an eight-character
non-secret hint are stored. The successful initial create or rotate response shows the full secret
once; an idempotent response-loss retry returns the same durable result with the secret omitted.
List/read operations never expose verifier material, and simultaneous browser-cookie plus AtlasKey
authentication is rejected rather than choosing one identity implicitly.

The only machine scope is `identity:read`, the audience is `atlas-api`, and the environment must
match the running API. It currently authorizes only `GET /v1/me`; it cannot authorize browser,
organization-mutation, approval, wallet, or money-movement operations. Create, rotate, and revoke
require an active merchant-security-administrator session, the exact action-bound five-minute
step-up, idempotency, a versioned authorization decision, and atomic Audit.

Rotation creates the replacement before marking the old credential `rotating`, permits the old
credential for exactly ten minutes, and supports immediate explicit old-key revocation. The
credential row lock linearizes concurrent rotations and revocations. A PostgreSQL savepoint owns
each provisional mutation claim: a losing or invalid target rolls back only that claim before the
outer transaction commits the denied Audit fact. The API role therefore has no delete privilege on
credentials or replay history.

## Scope proved

The cumulative verifier binds the S04–S07 regression chain, migration 15 and policy seed v4,
OpenAPI/architecture/secret-boundary checks, builds, observability mutation canaries, evidence
integrity, real database roles and isolated backup/WAL/PITR recovery, and focused PostgreSQL and
Redis integration. The focused evidence proves:

- exact create replay returns redacted metadata and the original decision without recoverable
  secret material;
- wrong verifier, tenant-derived credential identity, environment, audience, scope, status,
  expiry, algorithm, or version denies with the generic machine-authentication failure;
- old and replacement credentials authenticate during overlap, explicit revocation takes effect
  on the next PostgreSQL-backed request, and expiry is terminal;
- simultaneous different rotations produce one winner and one audited conflict, with no durable
  losing mutation claim or orphan replacement;
- lifecycle Audit failure rolls back the credential and replay record, after which the same
  idempotent request can safely succeed;
- credential, tenant, and HMAC-derived network dimensions are enforced atomically in Redis, while
  Redis outage enters stricter bounded in-process limits and capacity exhaustion fails closed;
- new-network anomaly and authentication/rate/fallback metrics use closed labels without tenant,
  key, secret, network, or other unbounded identifiers; and
- backup and isolated PITR restore preserve a revoked synthetic credential, migration/seed
  checksums, least-privilege grants, and the absence of plaintext secret columns.

## Requirement, threat, and adversarial links

- **Requirements:** `IAM-040..044`; `IAM-006` and `IAM-020..026` are revalidated at the lifecycle
  and machine-authentication boundaries.
- **Threats:** `THR-020`, `THR-023`, `THR-024`, `THR-041`, `THR-044`, `THR-056..058`.
- **Risks:** `RSK-009`, `RSK-017`, and `RSK-024` are addressed at this bounded synthetic surface;
  programme-wide residual risk remains.
- **Adversarial tests:** strict-scheme/downgrade fuzzing, mass-assignment and duplicate-header
  rejection, wrong-binding matrix, response-loss replay, simultaneous rotation, old/new overlap,
  immediate revocation, Audit outage, Redis outage, counter isolation, and bounded-capacity
  fail-closed behavior.

## Failure posture and operations

Malformed or ambiguous credentials, unknown keys, mismatched bindings, expired/revoked keys, stale
authority, absent step-up, conflicts, unavailable PostgreSQL, and exhausted fallback capacity deny
safely. Redis loss never makes a credential authoritative and produces the stricter fallback signal.
Failure before commit has no lifecycle effect; response loss after commit is resolved through the
durable idempotency result without redisclosing the secret.

The compromise runbook permits creation disablement, explicit revocation, replacement, bounded
signal inspection, and recovery rehearsal. It forbids secret recovery, verifier edits, direct
status edits, replay-record deletion, Redis-based re-enablement, and restoring an older active key.
`P01-S08-ALR-001..003` cover authentication anomaly, rate rejection, and fallback operation. No
deployed pager, SIEM, or production response exercise is claimed.

## Sanitization

This report contains only public source identities, closed statuses/scopes/actions, aggregate PASS
results, migration identity, and requirement/threat/control semantics. It excludes full secrets,
verifiers, database/Redis credentials and URLs, cookies, CSRF/idempotency material, network
signals, opaque runtime identifiers, raw SQL errors, WAL/data pages, and real identity data.

## Limitations

- Evidence is same-host synthetic proof with local PostgreSQL and Redis; it is not an independent
  review, production capacity claim, compliance certification, or deployed alert-routing claim.
- Local fallback counters are deliberately stricter and process-local. They preserve fail-closed
  abuse posture during Redis loss but do not claim globally exact distributed quotas.
- No managed production secret manager, merchant OAuth client-credentials flow, financial scope,
  worker, event, browser credential-management UI, or product SDK exists in this slice.
- Administrator membership removal remains fail-closed pending the exact fresh-step-up and
  last-administrator policy. Phase 01 remains open for P01-S09 acceptance and closure.
