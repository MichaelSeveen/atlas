# P01-S04 step-up idempotency and assurance post-commit verification

## Evidence identity

- **Evidence ID:** `EVD-P01-S04-STEP-UP-IDEMPOTENCY-ASSURANCE-POSTCOMMIT`
- **Phase/slice:** `PHASE-01_IDENTITY_ACCESS_TENANCY` / `P01-S04b`
- **Implementation revision:** `015911fdc586b4e7b65a80d29cdf06b799e37fc4`
- **Implementation tree:** `586fe6fb1ab7a8d020c4f21803bc8f560aa7d9c2`
- **Implementation commit time:** `2026-07-27T03:52:14+01:00`
- **Evidence-selection revision:** `0e3b4ad256c5950163d30d871a6b7c37e81b4e64`
- **Evidence-selection tree:** `79f27dc43b69426171b2770cc92d4e87a66f9ee1`
- **Evidence-selection commit time:** `2026-07-27T03:53:49+01:00`
- **Observed date:** 2026-07-27
- **Environment:** Windows host, PowerShell 7, Go 1.25.12, Bun 1.3.0, Podman/WSL,
  repository-pinned PostgreSQL and synthetic Keycloak 26.7.0
- **Verification worktree:** `UNCOMMITTED_WORKTREE(base=0e3b4ad256c5950163d30d871a6b7c37e81b4e64)`
  containing only additive post-commit evidence and permitted status/index updates

The Phase 01 evidence ancestry guard proves that the declared source is an ancestor of the
verification worktree and that every later path is within the explicit post-commit evidence
allowlist. Artifact tamper, stale source, duplicate evidence ID, and unsafe path mutations remain
fail-closed. No code, migration, contract, identity configuration, runtime environment, or test
changed after implementation revision `015911f`.

## Reproduction

```text
pwsh -NoProfile -File ./scripts/verify-p01-s04.ps1 -Live -ContainerRuntime podman
pwsh -NoProfile -File ./scripts/verify-s07.ps1 -History
```

## Observed result

PASS for the committed P01-S04b step-up checkpoint.

The canonical live gate:

- passed the complete Go/Bun/static/contract/mutation/evidence/S03 regression set;
- verified migration 7, empty/previous/idempotent lanes, tenant/population/OIDC/replay
  constraints, real database roles, lock abort, NATS, base backup, WAL, and isolated PITR;
- passed the disposable real PostgreSQL simultaneous-owner, in-progress, lease-reclaim,
  stale-owner fencing, exact-replay, changed-request, session, revocation, and Audit assertions;
- rebuilt API and web images from the committed implementation;
- revalidated supported three-realm Keycloak LoA flows;
- completed customer and merchant exact replay/conflict plus LoA 2 session/CSRF rotation with
  pre-step-up cookie rejection and logout revocation;
- kept workforce baseline authentication fail-closed and issued no workforce application
  session; and
- terminated with `p01_s04_live=PASS`, `p01_s04_live_verification=PASS`, and
  `p01_s04_verification=PASS`.

The S07 history gate scanned the worktree and 65 commits without a leak, detected the deleted
secret history canary, and reported zero called vulnerabilities. Race, Gosec, CodeQL, and hosted
enforcement remain explicitly unavailable or unverified on this CGO-disabled Windows host and
are required on the protected Linux workflow rather than claimed locally.

## Failure retention and correction

The pre-commit report retains both material failed attempts. One stopped on a transient WSL
container teardown error. A clean rerun then exposed nondeterministic simultaneous-claim behavior
under the original hashed advisory-lock election. That gate remained failed. Unique-scope
`INSERT ... ON CONFLICT` serialization plus `SELECT ... FOR UPDATE` replaced the election while
challenge-request rotation remained the stale-owner fencing token. Three independent disposable
PostgreSQL repetitions, the corrected pre-commit full gate, and this post-commit full gate passed.

## Sanitization

This artifact contains only source identities, public tool versions, bounded test labels,
synthetic population names, public route/error names, checksums, counts, and configuration
semantics. It excludes credentials, database URLs, authorization codes, state/nonce/PKCE values,
cookies, session or principal identifiers, tokens, provider payloads, internal credential-bearing
URLs, SQL error detail, WAL/data pages, and real identity/provider data.

## Limitations

- Same-host synthetic local/reference evidence; independent human review is unavailable and not
  claimed under ADR 0012.
- Level 2 is fresh synthetic password confirmation, not MFA or phishing resistance.
- Local developer backup/WAL volumes are unencrypted.
- Administrator security revocation, bounded account-enumeration timing, and the complete browser
  cache/storage/BFCache matrix remain open.
- No real IdP, real identity data, production secret custody, reference deployment, deployed alert
  routing, authorization evaluator, approval, API credential, event, worker input, or financial
  state exists.

Revalidate by 2026-10-27 or on any idempotency scope, OIDC/session, Keycloak flow, migration,
database role, contract, authorization, evidence policy, or runtime change, whichever occurs
first.
