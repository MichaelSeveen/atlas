# P01-S04 account-enumeration differential pre-commit verification

## Evidence identity

- **Evidence ID:** `EVD-P01-S04-ACCOUNT-ENUMERATION`
- **Phase/slice:** `PHASE-01_IDENTITY_ACCESS_TENANCY` / `P01-S04c`
- **Source:** `UNCOMMITTED_WORKTREE(base=744d1e1f1ede231b00897a0ee10d3342ece15495)`
- **Observed date:** 2026-07-27
- **Environment:** Windows host, PowerShell 7, Go 1.25.12, Bun 1.3.0,
  Podman local synthetic stack with PostgreSQL and Keycloak 26.7.0
- **Synthetic scenario:** three isolated identity populations, one configured synthetic username
  and one deterministic absent username per population, one warm-up per arm, then nine
  interleaved observations per arm

This is additive pre-commit evidence. It does not alter the earlier S04 core, CI remediation,
Gosec remediation, or step-up records.

## Scope and boundaries

This focused slice exercises `IAM-005`, `THR-037`, `THR-039`, and `ADV-IAM-010` at the synthetic
identity-provider authentication boundary. Atlas exposes no username-taking login or recovery
operation: `/v1/auth/login` selects only an identity population and redirects to the configured
provider. ADR 0014 delegates recovery to the IdP and requires Atlas callback/error behavior to
remain non-enumerating.

The test therefore obtains a fresh Atlas OIDC transaction and Keycloak form for every observation,
submits the same definitely-wrong password for the known and absent usernames, and measures the
complete credential-submission response. It does not add an endpoint, persist a credential, invoke
a financial action, emit an event, or create a worker input.

## Implemented controls

- Each observation uses a fresh OIDC transaction and isolated HTTP cookie container.
- Authorization and form-action URLs must remain inside the exact loopback Atlas/realm allowlist.
- Both arms must return HTTP 200 without a callback redirect and expose exactly the approved
  generic `Invalid username or password.` copy.
- Known/absent attempts are interleaved in alternating order after warm-up to reduce drift bias.
- The nine-sample matrix fails if the median delta exceeds 150 ms, the median ratio exceeds 2.5
  when both medians are at least 5 ms, or the p95 delta exceeds 500 ms.
- Output contains only population labels, sample counts, bounded aggregates, status, and PASS/FAIL.
  It never prints submitted credentials, OIDC state, cookies, authorization codes, subjects, or
  principal/session identifiers.
- The test runs before the successful-login journeys in the full S04 live orchestrator, so its
  repeated failure behavior cannot be hidden by an existing provider session.

## Reproduction and observed result

Commands completed for this evidence version:

```text
pwsh -NoProfile -File ./scripts/test-p01-s04-account-enumeration.ps1
pwsh -NoProfile -File ./scripts/p01-s04.ps1 -ContainerRuntime podman
pwsh -NoProfile -File ./scripts/verify-p01-s04.ps1 -Live -ContainerRuntime podman
```

The first focused run passed all three population matrices. The complete live orchestration then
tore down and rebuilt the local stack, applied and revalidated seven migrations, passed real
PostgreSQL role, session-concurrency, base-backup, WAL, and isolated-PITR checks, configured all
three Keycloak realms, reran this differential matrix, and completed the existing customer,
merchant, and workforce journeys.

The rebuilt-stack observations were:

| Population | Median delta | Median ratio | p95 delta | Result |
|---|---:|---:|---:|---|
| customer | 21.7 ms | 1.18 | 17.7 ms | PASS |
| merchant | 4.8 ms | 1.04 | 80.8 ms | PASS |
| workforce | 6.5 ms | 1.05 | 1.4 ms | PASS |

Every arm returned HTTP 200, the same generic error copy, no callback redirect, and a uniform
repeated-attempt outcome. The terminal labels were
`p01_s04_account_enumeration_matrix=PASS(copy=status=timing-bounded,repeated-attempt-outcome=uniform)`
and `p01_s04_live=PASS`.

After the new evidence catalogue was bound, the complete verifier repeated the static Go/Bun,
contract, mutation, evidence-integrity, and S03 regression gates before rebuilding and rerunning
the live composition. Its population median deltas were 47.8 ms, 1.2 ms, and 2.8 ms; its p95
deltas were 42.1 ms, 48.1 ms, and 29.4 ms. It ended with
`p01_s04_live_verification=PASS` and `p01_s04_verification=PASS`.

## Contract defect retained as a blocker

The intended preceding checkpoint—administrator security revocation—cannot be implemented without
inventing transport semantics. `PHASE-01_IDENTITY_ACCESS_TENANCY.md` requires administrator
security revocation in `IAM-004`, while its API surface and OpenAPI expose only
`DELETE /v1/sessions/{session_id}`. That operation is explicitly
“Revoke one session owned by the current principal.” ADR 0014 simultaneously says OpenAPI owns the
complete BFF surface and forbids implementation from filling contract gaps. `P01-D15` records the
required additive contract correction; the owner-only operation was not silently overloaded.

## Sanitization

This artifact includes only public tool versions, source identities, bounded test labels,
synthetic population names, public route/error names, aggregate durations, and configuration
semantics. It excludes credentials, database URLs, authorization codes, state/nonce/PKCE values,
cookies, session or principal identifiers, tokens, provider payloads, raw HTML, SQL error detail,
WAL/data pages, and real identity/provider data.

## Limitations

- Pre-commit, same-host synthetic evidence; hosted Linux and independent review are not claimed.
- The statistical bounds are a deterministic local regression gate, not a universal constant-time
  or production latency claim.
- Atlas does not own a recovery endpoint. A real provider's recovery, throttling, localization,
  bot defense, and population-discovery posture require a later provider-selection evidence set.
- Repeated attempts had the same outcome in this bounded matrix; no production rate limiter,
  anomaly backend, or abuse-control claim is made.
- `IAM-005` remains `Planned` phase-wide.
- Administrator security revocation is blocked on `P01-D15`; the full browser
  cache/storage/BFCache matrix also remains open.

Revalidate by 2026-10-27 or on any OIDC/login/recovery contract, Keycloak realm, identity-provider
version, timing threshold/sample method, rate-control, evidence policy, or runtime change,
whichever occurs first.
