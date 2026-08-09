# P01-S04 step-up idempotency and assurance pre-commit verification

## Evidence identity

- **Evidence ID:** `EVD-P01-S04-STEP-UP-IDEMPOTENCY-ASSURANCE`
- **Phase/slice:** `PHASE-01_IDENTITY_ACCESS_TENANCY` / `P01-S04b`
- **Source:** `UNCOMMITTED_WORKTREE(base=79176d8d224ebdbd07f399537cc30ad129369401)`
- **Observed date:** 2026-07-27
- **Environment:** Windows host, PowerShell 7, Go 1.25.12, Bun 1.3.0,
  Podman local synthetic stack with PostgreSQL and Keycloak 26.7.0

This is additive pre-commit evidence. It does not alter the earlier S04 core, CI-remediation, or
Gosec-remediation records.

## Scope and boundaries

The focused slice implements the existing `POST /v1/step-up/challenges` contract without adding
an endpoint or event. Identity/PostgreSQL owns the scoped idempotency reservation, normalized
request hash, processing lease, exact 201 response, and OIDC transaction. The synthetic Keycloak
provider call remains outside database transactions. There is no broker event, worker input,
financial state, or money movement.

Requirements exercised: `IAM-002`, `IAM-006`, `IAM-007`, while preserving `IAM-001`,
`IAM-020..021`, and `IAM-025`.

Threats exercised: `THR-006..007`, `THR-020`, `THR-037`, `THR-039`, `THR-044`, `THR-056`,
and `THR-058`.

## Implemented controls

- A forward-only migration adds mixed-scope, hash-only step-up replay state with a bounded
  processing lease, 24-hour key retention, exact stored authorization response, tenant/workforce
  scope constraints, and API-role `SELECT/INSERT/UPDATE` without `DELETE`.
- Same scoped key plus the same normalized action returns the original transaction ID, URL,
  expiry, status, and response body without another provider call.
- The same scoped key plus a different normalized action returns
  `IDEMPOTENCY_KEY_REUSED_WITH_DIFFERENT_REQUEST`; an active concurrent owner returns
  `IDEMPOTENCY_REQUEST_IN_PROGRESS`.
- The OIDC transaction and completed replay record commit atomically. An ambiguous completion
  failure cannot overwrite a committed response with a retryable failure.
- Lease recovery rotates the challenge-request identifier as a fencing token. A timed-out owner
  cannot complete or fail a row after a retry has reclaimed it.
- Step-up callback claims must carry higher assurance and an authentication time strictly newer
  than the five-minute freshness boundary.
- Atlas requests explicit baseline assurance for normal login and forces fresh LoA 2 for step-up
  with `prompt=login` and `max_age=0`.
- Keycloak uses supported `basic-flow` subflows. Level 2 repeats the username/password
  authenticator under forced reauthentication, with max age zero; a known user sees a
  password-confirmation form. The synthetic harness permits no more than two credential forms,
  manually preserves Secure provider cookies without printing them, and rejects any callback
  outside the allow-listed Atlas endpoint.
- Level 2 is a fresh synthetic password confirmation. This is not MFA and is not claimed to be
  phishing resistant.

## Reproduction and observed result

Completed before this evidence version:

```text
go test ./internal/platform/migration ./internal/identity ./internal/identity/persistence ./cmd/api/internal/server
go test ./internal/architecture -run TestPhase00GateClosurePolicy -count=1
pwsh -NoProfile -File ./scripts/configure-p01-s04-keycloak.ps1
pwsh -NoProfile -File ./scripts/test-p01-s04-oidc-http.ps1 -Population customer
pwsh -NoProfile -File ./scripts/test-p01-s04-oidc-http.ps1 -Population merchant
pwsh -NoProfile -File ./scripts/test-p01-s04-oidc-http.ps1 -Population workforce
pwsh -NoProfile -File ./scripts/verify-p01-s04.ps1 -Live -ContainerRuntime podman
pwsh -NoProfile -File ./scripts/verify-s07.ps1 -History
```

Observed:

- focused migration, service, persistence, and HTTP tests passed;
- exact replay remained available after the synthetic provider was made unavailable;
- changed-action reuse and stale higher-assurance authentication were rejected;
- a real PostgreSQL lease-recovery race rejected the stale owner’s late completion and allowed
  only the fenced replacement owner to publish the replay response;
- the HTTP test compared the complete replay body and `Location` value byte-for-byte;
- the Phase 00 migration/recovery guard passed after migration 7 was explicitly added to its
  closed inventory and revalidation basis;
- all three live synthetic realms accepted and revalidated supported `basic-flow` subflows;
- customer and merchant completed baseline login, exact replay/conflict, LoA 2 session/CSRF
  rotation, old-cookie rejection, and logout revocation;
- workforce baseline assurance remained fail-closed and issued no application session;
- live output explicitly recorded `mfa_claim=false`.

Final revalidation retained two material failed attempts. The first full run stopped during
teardown when WSL returned `-1` for `stop --timeout 5 web`; `podman info` then proved a healthy
runtime with no residual containers, and the same unwaived command was rerun. That clean run
exposed a nondeterministic simultaneous-claim result: one contender returned
`IDENTITY_UNAVAILABLE` instead of the contracted in-progress response. The gate remained failed.
The hashed advisory-lock election was replaced with unique-scope `INSERT ... ON CONFLICT`
serialization followed by `SELECT ... FOR UPDATE`; lease reclamation still rotates the
challenge-request fencing token. Three fresh disposable PostgreSQL repetitions and the subsequent
complete live gate passed the owner/in-progress, reclaim, stale-owner, exact-replay, and
changed-request cases.

The full `verify-p01-s04.ps1 -Live -ContainerRuntime podman` run passed. It repeated the complete
uncached Go/Bun/static/contract/mutation/evidence/S03 regression set; applied and revalidated all
seven migrations; passed real PostgreSQL role, concurrency, base-backup, WAL, and isolated-PITR
checks; rebuilt the API/web images; and completed the three-population Keycloak journeys. The
observed terminal labels were `p01_s04_live=PASS`,
`p01_s04_live_verification=PASS`, and `p01_s04_verification=PASS`.

The S07 history pass found no worktree or 63-commit history secret, detected its deleted-secret
canary, and reported zero called vulnerabilities. It correctly left race, Gosec, CodeQL, and
hosted enforcement unverified on this CGO-disabled Windows host rather than claiming them.

## Sanitization

This artifact includes only public tool versions, source identities, bounded test labels,
synthetic population names, public route/error names, and configuration semantics. It excludes
credentials, database URLs, authorization codes, state/nonce/PKCE values, cookies, session or
principal identifiers, tokens, provider payloads, SQL error detail, WAL/data pages, and real
identity/provider data.

## Limitations

- Pre-commit, same-host evidence; hosted Linux and independent-review results are not claimed.
- Level 2 uses fresh synthetic password confirmation. No MFA, phishing resistance, real IdP,
  production custody, or compliance claim is made.
- Administrator security revocation, bounded account-enumeration timing, and the full browser
  cache/storage/BFCache matrix remain later P01-S04 work.
- Independent human review is unavailable and not claimed under ADR 0012.

Revalidate by 2026-10-27 or on any idempotency scope, OIDC/session, Keycloak flow, migration,
database role, contract, authorization, or runtime change, whichever occurs first.
