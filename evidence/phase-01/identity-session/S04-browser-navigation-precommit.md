# P01-S04 browser logout/navigation pre-commit verification

## Evidence identity

- **Evidence ID:** `EVD-P01-S04-BROWSER-NAVIGATION`
- **Phase/slice:** `PHASE-01_IDENTITY_ACCESS_TENANCY` / `P01-S04`
- **Source:** `UNCOMMITTED_WORKTREE(base=744d1e1f1ede231b00897a0ee10d3342ece15495)`
- **Observed date:** 2026-07-27
- **Environment:** Windows host, Bun 1.3.0, React 19.2.7, rebuilt Podman web image,
  Chromium through the Codex in-app browser
- **Requirements/threats/tests:** `IAM-002`, `IAM-004`, `THR-006`, `THR-037`,
  `ADV-IAM-009`, Phase 01 “most agents skip” back-forward-cache test

## Boundary

The three React routes remain synthetic shell UI and are deliberately detached from product APIs
until P01-S09 selects and generates the OpenAPI client. This evidence therefore proves only the
browser-side cache/storage/navigation boundary. Server cookie issuance, CSRF, logout revocation,
old-cookie rejection, and `Cache-Control: no-store` are proved separately by the S04 Go/HTTP/live
suite.

## Implemented controls

- Logout clears the in-memory synthetic query cache before rendering signed-out state.
- A tab-scoped `atlas.synthetic.signed-out=1` marker prevents shell resurrection after a document
  reload. It is a non-sensitive guard, not a credential, principal, permission, or server session.
- Atlas-owned local storage remains empty. The storage diagnostic permits only that one
  session-storage marker and fails closed on any unknown Atlas-owned key.
- Reload, back, forward, and a direct protected-route navigation re-evaluate the signed-out guard.
- `pageshow.persisted` and `PerformanceNavigationTiming.type` distinguish BFCache restoration from
  an ordinary or history reload without exposing browser state in telemetry.

## Reproduction and observations

The final frontend source passed:

```text
bun test
bun run build
```

Observed result: 8 tests, 28 expectations, zero failures; the production build completed.

The first container rebuild after the browser change retained a material lifecycle failure:
the prior web container stopped in 8224 ms, exceeding the repository’s 8000 ms bound. No timeout
was widened. The same unmodified rebuild command was repeated and the final image stopped in
7276 ms, then reported `s04_environment_up=PASS`.

The real browser observations on that final image were:

| Scenario | Safe observation | Result |
|---|---|---|
| New customer shell | Browser credential storage reported absent | PASS |
| Logout | `/signed-out`; local Atlas entries `0`; session Atlas entries `1`; unexpected entries `false` | PASS |
| Navigate to `/`, then Back | `/signed-out`; restore kind `history-reload`; signed-out `true`; actor shell absent | PASS |
| Further Back / Forward / Reload | `/signed-out`; signed-out `true`; actor shell absent | PASS |
| Direct navigation to `/customer` | Redirected to `/signed-out`; signed-out `true`; actor shell absent | PASS |

The history traversal reloaded the no-store document rather than restoring a BFCache actor shell.
At no point did the browser expose or persist an Atlas cookie value, access token, refresh token,
CSRF token, OIDC state, authorization code, PKCE value, principal identifier, or permission.

## Sanitization and limitations

Only route names, boolean state, safe storage-entry counts, restore classification, public tool
versions, bounded shutdown durations, and test totals are retained. Browser storage values,
cookies, credentials, protocol material, provider payloads, raw HTML, screenshots, and real
identity data are excluded.

This is same-host synthetic shell evidence, not a product-client or real-account browser journey.
It proves the negative browser boundary without claiming production browser coverage, a real IdP,
MFA, phishing-resistant workforce authentication, or independent review.

Revalidate by 2026-10-27 or on any web routing, cache header, client-storage, logout/session,
generated-client, browser-runtime, or container-lifecycle change, whichever occurs first.

