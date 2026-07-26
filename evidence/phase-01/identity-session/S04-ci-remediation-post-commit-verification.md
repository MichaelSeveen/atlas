# P01-S04 CI remediation post-commit verification

## Evidence identity

- **Evidence ID:** `EVD-P01-S04-CI-REMEDIATION`
- **Phase/slice:** `PHASE-01_IDENTITY_ACCESS_TENANCY` / `P01-S04`
- **Implementation revision:** `57ea819b8c4e585cd4df8e6eafa1535d61a7b526`
- **Implementation tree:** `8e691e28edffc5c502bf3df147076f86736a893c`
- **Implementation commit time:** `2026-07-26T21:22:32+01:00`
- **Observed date:** 2026-07-26
- **Environment:** Windows host, PowerShell 7, Go 1.25.12, Bun 1.3.0, Podman/WSL,
  repository-pinned PostgreSQL, NATS, and Keycloak images

This additive evidence version revalidates the S04 core checkpoint after correcting the two
failed PR gates at pushed revision `67a238efa84814355ee16fe9924eb4435f38f5ef`. Historical S04
evidence remains unchanged.

## Change and boundary review

- Git mode for `db/tools/apply-phase-01-seeds.sh` changed from `100644` to `100755`, matching
  its direct execution from the Linux database test.
- The container-invoked script inventory now rejects a missing executable bit for that seed
  loader.
- Gitleaks has a match-scoped exception for the exact pair of unpublished synthetic event names
  that the generic API-key rule misclassified. No file, directory, rule, or secret-value
  allowlist was added.
- `golang.org/x/net` moved from `v0.52.0` to `v0.53.0`, which fixes `GO-2026-4918`;
  its required `golang.org/x/sys` dependency moved from `v0.42.0` to `v0.43.0`.
- Direct OIDC imports were normalized by `go mod tidy`; no application code changed.
- OpenAPI, AsyncAPI, database schema, migrations, seeds, authorization behavior, session
  behavior, idempotency/concurrency semantics, and financial boundaries are unchanged.

Requirements revalidated: `FND-020`, `FND-021`, `FND-025`, `FND-026`, `FND-054`, and the
existing S04 boundary for `IAM-001..007`, `IAM-020..021`, and `IAM-025`.

Threats revalidated: `THR-006..007`, `THR-020`, `THR-037`, `THR-039`, `THR-044`, `THR-056`,
and `THR-058`. No threat row or IAM requirement changes status.

## Root-cause confirmation

GitHub Actions run `30216709777` contained five jobs. Two failed:

1. `static-contracts-secret-history` stopped at the ADR 0012 governance step because PR #25
   retained six unchecked sensitive-change declarations.
2. `postgres-nats-migration-lanes` reached the real Phase 01 database test and failed with exit
   code 126 when Linux directly invoked `/database/tools/apply-phase-01-seeds.sh`.

After the governance blocker was reproduced, the complete local history gate also exposed a
latent false positive in committed test data and `GO-2026-4918` in `x/net v0.52.0`. Both were
fixed before the PR declaration was marked complete.

## Reproduction and observed result

```text
pwsh -NoProfile -File ./scripts/verify-s05.ps1 -Live -ContainerRuntime podman
pwsh -NoProfile -File ./scripts/verify-s07.ps1 -History
go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...
pwsh -NoProfile -File ./scripts/test-s08-constrained-pool.ps1
pwsh -NoProfile -File ./scripts/p01-s04.ps1 -ContainerRuntime podman
pwsh -NoProfile -File ./scripts/verify-p01-s04.ps1 -Live -ContainerRuntime podman
```

Observed committed-source results:

- real PostgreSQL migrations, roles, empty/previous lanes, deterministic Phase 01 seed
  idempotence, tenant/population/OIDC constraints, Audit atomicity, bounded lock abort, and real
  NATS JetStream passed;
- isolated PITR restored the exact product identity state and preserved revoked session
  authority; the final evidence-aware run observed a 30-second RTO;
- the disposable real PostgreSQL session repository passed its concurrency and atomic Audit
  assertions;
- the rebuilt synthetic stack passed API readiness/version, web, Keycloak, broker, and object
  storage smoke checks;
- customer and merchant completed callback, secure opaque-cookie, current-principal,
  session-inventory, CSRF denial, step-up initiation, logout, and old-cookie rejection checks;
- workforce baseline authentication remained denied and issued no application session;
- the constrained one-connection PostgreSQL test passed locally; the required race-detector
  proof remains assigned to the GitHub Linux job;
- Gitleaks passed worktree and 58-commit history scans, including the deleted-history-secret
  canary;
- Govulncheck reported zero called vulnerabilities with Go 1.25.12 and `x/net v0.53.0`;
- Go build/vet/test, contract lint, migration verification, Bun typecheck/test/build, and
  evidence mutation canaries passed.

## Failure and recovery observations

- The sandbox initially denied WSL access before container start; rerunning with the required
  host permission passed.
- Two Podman build attempts encountered external `proxy.golang.org` stream/connection resets.
  A bounded host-network cache fill completed the identical pinned module layer; the subsequent
  normal compose build also completed successfully.
- One intermediate restore attempt exited before its assertions; the immediately following
  clean topology run passed the complete PITR verification and preserved revoked authority.
- No failed attempt changed contracts, migrations, seed contents, or product behavior. Database
  test targets were bounded and disposable; the final synthetic stack is healthy.

## Sanitization

This artifact contains only source identities, public dependency versions and vulnerability IDs,
closed operation/population labels, bounded PASS results, counts, and durations. It excludes
credentials, database URLs, authorization codes, state/nonce/PKCE values, cookies, session
identifiers, access or refresh tokens, raw provider payloads, internal credential-bearing URLs,
SQL error detail, WAL/data pages, and real identity/provider data.

## Limitations

- Independent human review is unavailable and is not claimed under ADR 0012.
- GitHub Linux supplies the required race, Gosec, CodeQL, and container/supply-chain verdicts;
  the hosted rerun was pending when this local evidence version was written.
- The WSL compose fallback does not forward the host source-revision build argument, so local
  images retain the conservative `development` label even though the evidence ancestry guard
  binds code and configuration to `57ea819b8c4e585cd4df8e6eafa1535d61a7b526`.
- Existing S04 limitations remain: contracted step-up `Idempotency-Key` replay/conflict
  semantics, successful live higher-assurance completion, admin security revocation, bounded
  account-enumeration timing, and the complete browser cache/storage/BFCache matrix are open.
- No real IdP, real identity data, production secret custody, reference deployment, deployed
  alert routing, authorization evaluator, approval, API credential, event, worker input, or
  financial state exists.

Revalidate by 2026-10-26 or on any OIDC, dependency, scanner policy, cookie/session,
migration/seed/policy/schema/role, telemetry, recovery, container, or runtime change, whichever
occurs first.
