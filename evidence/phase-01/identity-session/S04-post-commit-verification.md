# P01-S04 post-commit verification

## Evidence identity

- **Evidence ID:** `EVD-P01-S04-POSTCOMMIT`
- **Phase/slice:** `PHASE-01_IDENTITY_ACCESS_TENANCY` / `P01-S04` core checkpoint
- **Implementation revision:** `d276ad457e1ce7e3863cbd4717dfb5c432e2e29d`
- **Implementation tree:** `873cba6d4eb6f4cfc024413bfb776e085047ea23`
- **Implementation commit time:** `2026-07-26T18:35:47+01:00`
- **Observed date:** 2026-07-26
- **Environment:** Windows host, PowerShell 7, Go 1.25.12, Bun 1.3.0, Podman/WSL,
  repository-pinned PostgreSQL and Keycloak images
- **Verification worktree:** `UNCOMMITTED_WORKTREE(base=d276ad457e1ce7e3863cbd4717dfb5c432e2e29d)`
  containing only the post-commit catalogue and sidecar while the verifier ran

The Phase 01 evidence ancestry guard proved that the implementation revision was the current
`HEAD`, that every dirty path was below the explicit S04 evidence-only allowlist, and that
artifact tamper, stale source, duplicate evidence ID, and unsafe path mutations were rejected.
No code, migration, contract, configuration, runtime environment, or test changed after the
implementation commit.

## Reproduction

```text
pwsh -NoProfile -File ./scripts/verify-p01-s04.ps1 -Live -ContainerRuntime podman
```

## Observed result

PASS for the committed P01-S04 core checkpoint.

Static and build observations:

- all Go entry points built;
- `go vet ./...` and `go test ./... -count=1` passed;
- Bun tests and production build passed;
- OpenAPI and AsyncAPI lint passed;
- six released migrations verified at version 6 with checksum
  `b8c46daa86ff72667161201fc494fb296325737330d988117fd1e76c62f6e9a0`;
- S02 contract and policy mutations, S03 migration/seed/tenant/Audit mutations, Phase 00 topology
  guards, canonical-manifest checks, and all four Phase 01 evidence mutations were rejected.

Live observations:

- empty/previous/idempotent migration lanes, deterministic seed, Identity population/OIDC
  constraints, Audit atomicity, role matrix, and bounded long-lock abort passed;
- the real NATS JetStream check passed;
- base backup and WAL archive passed;
- isolated PITR restored exact Identity/Audit state and preserved revoked authority; observed RTO
  was 68 seconds;
- the disposable real PostgreSQL session repository passed concurrent identical revoke-all,
  replay, changed-request conflict, and atomic Audit assertions, then dropped its test database;
- the rebuilt stack passed API live/ready/version, web-shell, Keycloak realm, broker, and object
  storage smoke checks;
- all three Keycloak clients remained public PKCE-S256 clients with direct grants disabled;
- customer and merchant completed callback, Secure/HttpOnly/SameSite cookie, current-principal,
  current-session inventory, CSRF denial, fresh step-up initiation, logout, and old-cookie
  rejection;
- workforce baseline authentication was denied and issued no application session.

## Sanitization

This artifact contains only source identities, checksums, closed population/operation labels,
bounded PASS results, counts, and durations. It excludes runtime environment files, credentials,
database URLs, authorization codes, state/nonce/PKCE values, cookies, session identifiers, access
or refresh tokens, raw provider payloads, internal credential-bearing URLs, SQL error detail,
WAL/data pages, and real identity/provider data.

## Limitations

- same-host synthetic local/reference evidence; no independent human review is claimed;
- the evidence-only dirty worktree caused the container build to retain the conservative
  `development` OCI/application revision label even though the ancestry guard proved the code,
  migrations, contracts, and configuration were exactly implementation revision `d276ad4`;
- local backup/WAL developer volumes are unencrypted;
- step-up does not yet implement contracted `Idempotency-Key` replay/conflict behavior;
- successful live higher-assurance completion, admin security revocation, bounded
  existing/absent-principal timing, and the complete browser cache/storage/BFCache matrix remain
  open;
- no real IdP, real identity data, production secret custody, reference deployment, deployed
  alert routing, authorization evaluator, approval, API credential, event, worker input, or
  financial state exists.

Revalidate by 2026-10-26 or on any OIDC, cookie/session, migration/seed/policy/schema/role,
telemetry, recovery, container, or runtime change, whichever occurs first.
