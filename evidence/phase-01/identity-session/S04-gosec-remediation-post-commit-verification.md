# P01-S04 Gosec remediation post-commit verification

## Evidence identity

- **Evidence ID:** `EVD-P01-S04-GOSEC-REMEDIATION`
- **Phase/slice:** `PHASE-01_IDENTITY_ACCESS_TENANCY` / `P01-S04`
- **Implementation revision:** `818f42e06f06e6e731136356e9dcbd7127237b9e`
- **Implementation tree:** `43dd74ac0e2b0bbb549184b0662e7e3d710f68ca`
- **Implementation commit time:** `2026-07-27T00:00:39+01:00`
- **Observed date:** 2026-07-27
- **Environment:** Windows host, PowerShell 7, Go 1.25.12, repository-pinned Gosec 2.25.0;
  GitHub Actions Ubuntu runner for the originating finding

This additive evidence version follows the broader CI remediation evidence without changing or
overwriting it.

## Hosted finding and root cause

GitHub Actions run `30223627970` passed four of five jobs: the real PostgreSQL/NATS migration
lane, both CodeQL languages, and the complete SBOM/vulnerability/license/container lane. The
remaining `static-contracts-secret-history` job passed governance, tests, race detection,
contract checks, Gitleaks worktree/history scans, and the deleted-secret canary before
repository-pinned Gosec 2.25.0 reported `G101` at `cmd/api/internal/server/cors.go`.

The reported map contains only public HTTP header names (`Content-Type`, `Idempotency-Key`,
`traceparent`, the Atlas CSRF header, and correlation/request IDs). It contains no credential,
credential value, secret, token, session identifier, or authentication material. `G101` was
therefore a scanner false positive caused by credential-related words in public protocol names.

## Change and boundary review

- A line-scoped `#nosec G101` annotation now explains that the map values are public HTTP
  header names rather than credentials.
- The existing credentialed-preflight test now asserts the exact canonical response header
  allowlist, including the idempotency and CSRF header names that triggered the finding.
- The allowlist remains closed and server controlled. Wildcard origins and unrecognized
  request headers remain denied.
- OpenAPI, AsyncAPI, migrations, database roles, persistence, OIDC/session behavior,
  authorization decisions, idempotency semantics, financial boundaries, and dependency
  versions are unchanged.

Requirements revalidated: `FND-020`, `FND-026`, `IAM-001..007`, `IAM-020..021`, and `IAM-025`.

Threats revalidated: `THR-007`, `THR-020`, `THR-037`, `THR-039`, and `THR-056`.

## Reproduction and observed result

```text
go test ./cmd/api/internal/server ./internal/architecture -count=1
./.tmp/s07-tools/bin/gosec.exe -quiet -exclude-generated -exclude-dir .tmp ./...
```

Observed committed-source results:

- the focused server and architecture packages passed;
- the exact pinned Gosec command that failed on GitHub exited zero with no findings;
- the CORS regression assertion preserved the exact credentialed preflight response;
- `git diff --check` passed before commit.

## Sanitization

This artifact contains only source identities, public tool versions, public HTTP header names,
bounded job/test results, and the public GitHub Actions run identifier. It excludes credentials,
database URLs, authorization codes, state/nonce/PKCE values, cookies, session identifiers,
access or refresh tokens, raw provider payloads, internal credential-bearing URLs, SQL error
detail, WAL/data pages, and real identity/provider data.

## Limitations

- Independent human review is unavailable and is not claimed under ADR 0012.
- The post-fix hosted five-job rerun was pending when this revision-bound local evidence was
  written.
- Existing S04 limitations remain: contracted step-up `Idempotency-Key` replay/conflict
  semantics, successful live higher-assurance completion, admin security revocation, bounded
  account-enumeration timing, and the complete browser cache/storage/BFCache matrix are open.
- No real IdP, real identity data, production secret custody, reference deployment, deployed
  alert routing, authorization evaluator, approval, API credential, event, worker input, or
  financial state exists.

Revalidate by 2026-10-27 or on any CORS, CSRF, idempotency, OIDC/session, scanner policy,
dependency, contract, authorization, or runtime change, whichever occurs first.
