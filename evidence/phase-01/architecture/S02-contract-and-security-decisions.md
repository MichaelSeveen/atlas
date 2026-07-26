# P01-S02 contract and security-decision evidence

## Evidence identity

- **Evidence ID:** `EVD-P01-S02-CONTRACT`
- **Phase/slice:** `PHASE-01_IDENTITY_ACCESS_TENANCY` / `P01-S02`
- **Observed date:** 2026-07-23
- **Source identity:** `UNCOMMITTED_WORKTREE(base=2884484a99eeb2b846a56c90177163e37e419d11)`
- **Environment:** Windows host, PowerShell 7, Go 1.25.12; no live stack or external identity provider
- **Seed/time:** no runtime seed; contract-only deterministic checks
- **Decision:** ADR 0014

## Scope and expected result

S02 was expected to close the Phase 01 HTTP, ownership, session, tenancy, authorization, approval,
API-credential, audit, and cross-phase evidence decisions without creating runtime behavior or
claiming an IAM requirement implemented.

Expected:

- every canonical Phase 01 operation is represented in OpenAPI;
- cookie-authenticated mutations require the closed CSRF header contract;
- policy is default-deny, PostgreSQL-authoritative, and internally consistent;
- one-time invitation/API secrets cannot be recovered through idempotent replay;
- privileged audit is synchronous and Phase 01 publishes no new event;
- every IAM traceability row remains `Planned`;
- changed/new canonical files are bound by `MANIFEST.sha256`;
- contract/policy mutations are rejected;
- Phase 00 evidence retains its original source identity.

## Observed result

PASS.

- OpenAPI 3.1.1 contains the original 15 Phase 01 paths/17 operations plus the seven support
  paths/eight operations required to close login/callback/logout, tenant switching, invitation
  acceptance, and approval creation/execution/cancellation semantics.
- `identity-access-policy.json` closes four identity populations, principal linking, session
  lifetimes/rotation/IdP outage, tenancy and invitation rules, 23 permissions, 13 roles, field
  masking/purpose, five-minute step-up, one typed approval action, least-privilege `AtlasKey`
  credentials, synchronous Audit, and Bun/OpenAPI client posture.
- AsyncAPI is unchanged. ADR 0014 requires synchronous privileged audit in the same PostgreSQL
  transaction and rejects Phase 01 event publication without a concrete consumer/delivery need.
- All 30 `IAM-*` rows parse with ten CSV columns, have requirement-specific verification/evidence
  routes, and remain `Planned`.
- The canonical manifest covers the policy, ADR, and all changed canonical sources.
- A removed `/v1/sessions` path and an allow-by-default policy both caused their focused tests to
  fail as expected.
- The Phase 00 post-commit catalogue remains bound to
  `188578b96e5b2fe5dab27930a9e2e66f20d2ca12`; Phase 01 uses a new catalogue.

## Reproduction and observed commands

```text
go run ./cmd/contractctl lint docs/atlas-prd/03-contracts/openapi.yaml docs/atlas-prd/03-contracts/asyncapi.yaml
PASS: both canonical contracts parse, use exact versions, and resolve internal references

go test ./tests/contract -count=1
PASS

go test ./internal/architecture -count=1
PASS: includes canonical manifest and Phase 01 evidence-policy validation

pwsh -NoProfile -File ./scripts/test-p01-s02-contract-canary.ps1
PASS: missing-openapi-path rejected; allow-by-default-policy rejected

pwsh -NoProfile -File ./scripts/test-s07-contract-compatibility.ps1 -BaseRef HEAD
PASS: additive compatibility for OpenAPI and unchanged AsyncAPI

go test ./... -count=1
PASS

go build ./cmd/api ./cmd/worker ./cmd/simulator
PASS
```

The single command for revalidation is:

```text
pwsh -NoProfile -File ./scripts/verify-p01-s02.ps1
```

It also verifies the source-bound Phase 01 evidence catalogue and its artifact-tamper,
stale-source, duplicate-ID, and unsafe-path canaries.

## Requirements, threats, and ownership

- **Requirements specified, not satisfied:** `IAM-001..007`, `IAM-010..015`, `IAM-020..026`,
  `IAM-030..034`, `IAM-040..044`.
- **Principal threats routed:** `THR-005..008`, `THR-018`, `THR-020`, `THR-023..024`, `THR-028`,
  `THR-037..041`, `THR-044`, `THR-053`, `THR-056..058`, `THR-060`.
- **Owners:** Identity (principal/session/tenancy/authorization/credentials), Operations
  (approvals), Audit (records), API (transport), with Security/Platform/Data/Privacy review.

## Sanitization

This report contains no credential, cookie, CSRF value, identity token, person data, private
endpoint, database data, or provider payload. IDs and hashes are repository revision/evidence
identifiers only. Canary mutations use structural strings, not secret-shaped values.

## Limitations and revalidation

- No OpenAPI product route is served by the running API.
- No schema, migration, principal, session, organization, authorization evaluator, approval,
  credential, synchronous audit record, OIDC exchange, browser product flow, telemetry, alert, or
  recovery behavior is implemented.
- Contract examples are not live-handler proof.
- The exact synthetic Keycloak client/realm/seeded-login profile remains a deliberate S04 blocker.
- The exact OpenAPI TypeScript generator package/version remains a deliberate S09 blocker; only
  the canonical-source, Bun-runtime, reproducible-output, and no-hand-edits strategy is fixed.
- Independent human review remains unavailable under ADR 0012 and is not claimed.
- Revalidate on every contract/policy/ADR/traceability change, before P01-S03, and no later than
  2026-10-23 if the slice remains uncommitted.
