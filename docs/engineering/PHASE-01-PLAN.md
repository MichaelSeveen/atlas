# Phase 01 identity, access, and tenancy audit and execution plan

## Status and scope

- **Completed slices:** `P01-S01 — identity/access/tenancy audit and execution plan`; `P01-S02 — canonical contract, ownership, and security-decision closure`; `P01-S03 — identity/audit persistence, deterministic seeds, roles, and recovery revalidation`
- **Current slice:** `P01-S04 — synthetic OIDC BFF and durable session lifecycle` is underway; the
  core session checkpoint plus idempotent step-up execution and live higher-assurance completion
  now pass, while admin security revocation and remaining browser/differential evidence are open.
- **Audit date:** 2026-07-23
- **Audited base revision:** `2884484a99eeb2b846a56c90177163e37e419d11`
- **Base tree:** `b921b93cb8341e28344b97e1202d37f0376dff19`
- **Current evidence posture:** the S01-S04 core checkpoint is committed at `d276ad4`. S01 is
  planning evidence, S02 is contract/decision evidence, S03 is real PostgreSQL
  persistence/recovery evidence, and S04 core has revision-bound synthetic
  OIDC/application-session static/live evidence. No IAM row is closed phase-wide.
- **Allowed environment:** synthetic local/reference identities and data only under ADR 0008 and ADR 0012.

S04 composes the ratified `atlas_identity`/`atlas_audit` persistence foundations into the approved
OIDC BFF and durable application-session routes only. Synthetic customer/merchant login and
session lifecycle work; workforce baseline login fails closed. It adds no event, worker job,
authorization evaluator, credential, approval, or financial behavior. All 30 Phase 01
requirement rows remain `Planned`; exact checkpoint evidence is attached without overstating
phase-wide completion.

The financial boundary is closed for the whole phase: Phase 01 may define authorization and approval foundations that later financial phases call, but it must not create a wallet, balance, hold, journal, payment, refund, payout, transfer, beneficiary, or other money-moving state or operation.

Pre-commit verification of this planning worktree passed `go test ./...`, all three entry-point builds, and `go test ./internal/architecture -count=1`. Static `verify-s08.ps1` passed its solo-governance and complete S07 sub-gates, then correctly stopped at the Phase 00 evidence-freshness check: the live Phase 00 catalogue is bound to implementation revision `188578b96e5b2fe5dab27930a9e2e66f20d2ca12`, not this dirty Phase 01 planning worktree. No historical catalogue was rebound merely to turn the check green. `P01-D14` and S02 own the versioned cross-phase evidence-continuity decision.

## Canonical basis and audit method

The audit used the source hierarchy in `AGENTS.md` and read the following controlling sources:

- `docs/engineering/IMPLEMENTATION_STATUS.md`
- `docs/engineering/CONTEXT_INDEX.md`
- `docs/atlas-prd/02-phases/PHASE-01_IDENTITY_ACCESS_TENANCY.md`
- `docs/atlas-prd/00-master/03_REQUIREMENTS_AND_QUALITY_GATES.md`
- `docs/atlas-prd/00-master/04_ROADMAP_AND_DEPENDENCIES.md`
- `docs/atlas-prd/01-architecture/00_SYSTEM_ARCHITECTURE.md`
- `docs/atlas-prd/01-architecture/01_SECURITY_AND_TRUST_MODEL.md`
- `docs/atlas-prd/01-architecture/03_API_AND_EVENT_STANDARDS.md`
- `docs/atlas-prd/03-contracts/openapi.yaml`
- `docs/atlas-prd/03-contracts/asyncapi.yaml`
- `docs/atlas-prd/03-contracts/EVENT_CATALOG.md`
- `docs/atlas-prd/06-governance/REQUIREMENTS_TRACEABILITY.csv`
- `docs/atlas-prd/06-governance/THREAT_REGISTER.csv`
- `docs/atlas-prd/06-governance/RISK_REGISTER.md`
- `docs/atlas-prd/06-governance/DEFINITION_OF_DONE.md`
- ADR 0001, ADR 0002, ADR 0004, ADR 0008, ADR 0010, ADR 0012, and ADR 0013
- `docs/engineering/phase-00-gate-policy.json`

The implementation audit covered the three synthetic Keycloak realm imports, the Phase 00 seed catalogue, the API composition and HTTP foundation, the frontend session-shell helper, the module checker, the migration manifest/validator, PostgreSQL bootstrap roles and permission tests, the current recovery procedure, the identity-provider runbook, and the observability catalogue.

No handoff assertion was used to override repository state. Git confirms a clean protected-main base at merge `2884484`, whose first parent history includes the Phase 00 gate-closure revisions. The configured remote is the expected repository, with the owner name embedded in the HTTPS URL.

## Audit result

### Current implementation

| Surface | Audited result |
|---|---|
| Phase 00 gate | Closed for the bounded synthetic feature-free topology by ADR 0013. `FND-026` remains an accepted deviation, and `FND-040`/`FND-042` remain accepted scope decisions. |
| Identity contexts | `internal/identity`, `internal/operations`, and `internal/audit` are ownership placeholders only. There is no product Go package, store, schema, or application service. |
| API | The running API exposes only `/health/live`, `/health/ready`, and `/version`. It has no BFF, session, authorization, tenant, approval, credential, or product handler. |
| Keycloak | Three isolated synthetic realms and one user per realm exist. They prove discovery and population separation only. They define no clients, redirect URIs, PKCE policy, session policy, MFA flow, role catalogue, seeded login credential, or Atlas identity exchange. |
| PostgreSQL | Only `atlas_foundation` exists. Runtime roles are schema-inert; the migration validator deliberately rejects the product term `identity`; the released migration directory and manifest are closed by ADR 0013. |
| Seeds | The checksummed Phase 00 catalogue names two tenants and three synthetic users, but no loader inserts product data or binds Keycloak subjects to Atlas principals. |
| Frontend | Customer, merchant, and workforce route shells plus a synthetic in-memory cache clear test exist. There is no real session, organization switch, step-up, approval, member, or credential UI. |
| Authorization | No permission catalogue, role matrix, policy evaluator, tenant-query interface, decision ID, or cache invalidation mechanism exists. |
| Audit | No durable application audit record exists. The AsyncAPI contains a future `audit.event.recorded.v1` integration event, but no outbox, stream, relay, or consumer implements it. |
| Telemetry | The bounded catalogue covers operational routes and database readiness. It has no authentication, authorization, session, membership, approval, or credential metrics/alerts. |

### Traceability quality

`REQUIREMENTS_TRACEABILITY.csv` contains exactly 30 Phase 01 rows and all are correctly still `Planned`. However, all 30 rows currently repeat the same eight-threat bundle and the same broad verification text. That is planning-pack traceability, not an implementable per-requirement test map. Before a row can leave `Planned`, its threats, exact tests, evidence ID, owner, and limitations must be narrowed to the implemented behavior.

The relevant threat set is wider and more specific than the repeated bundle:

- canonical row bundle: `THR-005..008`, `THR-023`, `THR-037..039`;
- additional directly applicable threats: `THR-018`, `THR-020`, `THR-024`, `THR-028`, `THR-040..041`, `THR-044`, `THR-053`, `THR-056..058`, and `THR-060`;
- principal programme risks: `RSK-005`, `RSK-006`, `RSK-009`, `RSK-024`, `RSK-027`, `RSK-028`, `RSK-031`, and `RSK-032`.

Threat rows remain `Open`. This planning audit does not lower residual risk.

## S01 baseline requirement-by-requirement audit (historical)

The findings below preserve the pre-S02 audit baseline and are not the current implementation
status. “Planned” meant no implementation or evidence existed at that audit revision. Use
`IMPLEMENTATION_STATUS.md`, canonical traceability, and the exact-next-checkpoint section below
for current progress; phase closure remains `P01-S09`.

| Requirement | Audit finding | Earliest slice | Primary threats and proof |
|---|---|---|---|
| `IAM-001` | OIDC validation is specified, but no clients, callback, verifier, token policy, or exchange exists. | S04 | `THR-006`, `THR-023`, `THR-056`; callback state/nonce/PKCE, wrong issuer/audience, clock boundary, stale/algorithm-confusion tests |
| `IAM-002` | No durable application session or rotation path exists. | S04 | `THR-006`, `THR-037`, `THR-038`; `ADV-IAM-003`, `ADV-IAM-008..009`, old-cookie rejection |
| `IAM-003` | Population-specific idle and absolute values and enforcement semantics are unspecified. | S02/S04 | `THR-006`, `THR-037`; injected-clock boundary and concurrent-request expiry tests |
| `IAM-004` | `/v1/sessions*` contract and durable revocation/audit are absent. | S02/S04 | `THR-006`, `THR-037`, `THR-058`; revoke-one/all/admin, response-loss, multi-replica, logout/back-navigation tests |
| `IAM-005` | Atlas recovery behavior is intentionally delegated to the IdP, but generic BFF/IdP error and timing policy is not defined. | S02/S04 | `THR-039`, `THR-057`; `ADV-IAM-010`, differential status/body/timing/rate tests |
| `IAM-006` | Current high-risk actions and future payout/beneficiary/contact/refund/export actions lack a versioned action/freshness catalogue. | S02/S04/S07/S08 | `THR-007`, `THR-037`, `THR-053`; step-up expiry immediately before commit and absent-future-action fail-closed tests |
| `IAM-007` | The existing IdP runbook explicitly says no integration exists; no degraded-session policy is defined. | S02/S04 | `THR-006`, `THR-044`; IdP outage with existing low-risk session and high-risk step-up denial |
| `IAM-010` | No product row or documented global-scope catalogue exists. | S03 | `THR-005`, `THR-040..041`; schema inventory and tenant-not-null/global-scope canaries |
| `IAM-011` | No tenant repository exists; explicit tenant context must become a typed application/repository parameter. | S03/S05 | `THR-005`, `THR-044`; architecture/static signatures plus real PostgreSQL integration |
| `IAM-012` | No product query or tenant-predicate test exists. | S03/S05 | `THR-005`, `THR-040`; seeded missing-predicate mutation and two-tenant role tests |
| `IAM-013` | `NotFoundOrConcealed` exists as a generic OpenAPI response, but no Phase 01 object operation uses a defined enumeration policy. | S02/S05/S06 | `THR-005`, `THR-057`; `ADV-IAM-001`, timing/count/body differential tests |
| `IAM-014` | Invitation contract, role-delegation lattice, expiry, consumption, and acceptance semantics are absent. | S02/S05 | `THR-038`, `THR-041`; `ADV-IAM-007`, double-accept/race/expiry tests |
| `IAM-015` | No membership or revocation bound exists. Mutation serialization against removal is undecided. | S02/S05 | `THR-006`, `THR-038`, `THR-044`; stale-tab mutation and removal/commit race tests |
| `IAM-020` | Roles are named in the phase, but permissions, delegation rules, purposes, and forbidden combinations are not catalogued. | S02/S06 | `THR-005`, `THR-007`, `THR-038`; closed source catalogue and complete negative matrix |
| `IAM-021` | No policy engine or deny-by-default fallback exists. | S06 | `THR-005`, `THR-007`; unknown action/resource/field/role mutation canaries |
| `IAM-022` | Object/action/field/tenant enforcement is absent. | S05/S06 | `THR-005`, `THR-041`, `THR-057`; BOLA, mass-assignment, field-mask and direct-service tests |
| `IAM-023` | No sensitive list/search implementation exists. | S06 | `THR-005`, `THR-028`, `THR-057`; `ADV-IAM-002`, autocomplete/count/cursor differential tests |
| `IAM-024` | No field classification, purpose, reveal, or masking policy exists for Phase 01 resources. | S02/S06 | `THR-007`, `THR-028`, `THR-057`; permission/purpose matrix and safe-log/browser assertions |
| `IAM-025` | No authorization decision record or stable decision ID exists. | S03/S06 | `THR-007`, `THR-018`, `THR-058`; denied/high-risk audit completeness and audit-outage policy tests |
| `IAM-026` | No policy cache exists. Phase 01 should prefer authoritative PostgreSQL version checks rather than introduce Redis authorization truth. | S02/S06 | `THR-006`, `THR-038`, `THR-044`; multi-replica role/session-assurance invalidation; future restriction trigger |
| `IAM-030` | The phase defines an approval state model, but no aggregate owner, canonical payload hash, action registry, or execution protocol exists. | S02/S07 | `THR-008`, `THR-024`, `THR-041`; state-model/property/concurrency tests |
| `IAM-031` | Principal separation is required, but alternate IdP accounts mapping to one human/principal is unspecified. | S02/S07 | `THR-008`; `ADV-IAM-005`, same-principal second-login and role-change tests |
| `IAM-032` | Payload canonicalization and database-tamper response are unspecified. | S02/S07 | `THR-008`, `THR-040..041`; fixture mutation and hash mismatch tests |
| `IAM-033` | Target command binding and execution-time transaction/reauthorization semantics are absent. | S02/S07 | `THR-006`, `THR-008`, `THR-024`; `ADV-IAM-006`, state/role/step-up races |
| `IAM-034` | Terminal/superseded retry semantics and optimistic concurrency are absent. | S02/S07 | `THR-008`, `THR-024`; replay, expiry, cancellation, supersession, response-loss tests |
| `IAM-040` | OpenAPI defines `merchantOAuth`, while the phase requires a one-time-display API credential resource; the supported machine-credential modes are not reconciled. | S02/S08 | `THR-020`, `THR-023`, `THR-041`, `THR-056`; entropy/hash/no-log/unknown-field/audience tests |
| `IAM-041` | Credential create contract, step-up binding, and atomic audit do not exist. | S02/S08 | `THR-007`, `THR-023`, `THR-058`; expired step-up and audit-failure tests |
| `IAM-042` | Overlap duration, activation order, in-flight behavior, idempotency, and old-key revocation are unspecified. | S02/S08 | `THR-023`, `THR-024`; old/new concurrent requests and linearizable revocation tests |
| `IAM-043` | No one-time secret response or non-recovery control exists. | S02/S08 | `THR-020`, `THR-023`; response-loss, read/list, database/log/browser inspection tests |
| `IAM-044` | Rate-limit/anomaly dimensions and authoritative/ephemeral split are unspecified. | S02/S08 | `THR-023`, `THR-044`, `THR-057`; per-credential/tenant isolation, Redis-loss, bounded-cardinality tests |

## S01 baseline contract and event audit (historical)

### OpenAPI coverage

The Phase 01 file lists 15 distinct paths and 17 operations. The canonical OpenAPI contains only `GET /v1/me`.

| Route group | Required paths / operations | Present | Blocking gap |
|---|---:|---:|---|
| Current principal, sessions, revocation, step-up | 5 / 5 | 1 / 1 | Session list/delete/revoke-all and step-up challenge absent |
| Organizations, members, invitations | 4 / 5 | 0 / 0 | All paths absent; tenant switch and invitation acceptance are not listed even in the phase API surface |
| API credentials | 3 / 4 | 0 / 0 | All paths absent; authentication scheme conflicts with the existing `merchantOAuth`-only contract |
| Approvals | 3 / 3 | 0 / 0 | All paths absent; no execution semantics |
| **Total** | **15 / 17** | **1 / 1** | **14 paths and 16 operations absent** |

Additional BFF protocol routes or an explicitly documented non-OpenAPI browser redirect boundary are required for login initiation, OIDC callback, logout, and step-up return. The CSRF request/response mechanism is absent from OpenAPI. Exact cookies, redirect allowlists, idempotency, ETag/precondition behavior, pagination, masking, errors, rate limits, audit effect, and cache headers are also missing for the Phase 01 operations.

`Principal` is present with identity type, active tenant, assurance, and permissions. It does not settle whether permissions are effective point-in-time values, how tenant switch occurs, or how stale clients detect a policy/session version.

No product handler may be added until S02 updates the canonical OpenAPI first and contract lint/compatibility tests pass.

### AsyncAPI and event catalogue

Neither AsyncAPI nor the event catalogue defines an identity, session, membership, organization, role, invitation, approval, or API-credential event. AsyncAPI does define future `audit.event.recorded.v1`, but no Phase 00 outbox or audit pipeline implements it.

Phase 01 does not need an asynchronous path to satisfy its current requirements. Durable audit recording will be synchronous and transactionally coupled through the Audit application service. Publication of `audit.event.recorded.v1`, or any new event/consumer/stream, is out of scope and would activate `FND-040` plus outbox, inbox, worker, telemetry, replay, ordering, and recovery obligations. S02 must record this deliberate no-event decision; AsyncAPI remains unchanged unless a concrete consumer and causal need are approved.

## Chosen module and data ownership

The accepted architecture resolves the package boundary as follows:

| Owner | Phase 01 ownership |
|---|---|
| `internal/identity` | Atlas principals and external-subject links; durable application sessions and revocation/assurance metadata; merchant organizations, memberships, invitations, and active-tenant eligibility; permission catalogue, role bindings, delegation policy, and authorization version; merchant API-credential metadata and verifier state. |
| `internal/operations` | Generic approval aggregate, immutable requested-payload binding, checker policy, decision lifecycle, and target-command execution coordination. The system architecture explicitly assigns approvals to Operations. |
| `internal/audit` | Append-only security/access audit facts and the application API used atomically by Identity and Operations. It does not activate the AsyncAPI audit pipeline. |
| `cmd/api` | Composition, BFF/HTTP adapters, middleware, secure-cookie/CSRF boundary, and routing only. It does not own domain state or authorization truth. |
| Keycloak | Synthetic authentication, factor, and recovery behavior only. It does not own Atlas tenant membership, authorization, approval, or API-credential truth. |
| PostgreSQL | Authoritative sessions/revocation, principals, membership/tenant, authorization versions/bindings, approvals, API credentials, and audit metadata. Redis may accelerate rate limits or non-authoritative caches only. |

There will be no new `internal/authorization` bounded context. It is not in the canonical bounded-context list or architecture checker, and `internal/identity` is already a sensitive path. The currently nonexistent `internal/authorization/` entry in the solo-maintainer sensitive-path list is not ownership authority and must not be used to bypass the registered-module rule.

Each context owns its tables and migrations within the single ordered repository migration stream. Cross-context writes call root/application APIs. Identity or Operations code must never issue SQL against Audit-owned tables, and API adapters must not import a context’s store package.

The intended PostgreSQL namespaces are `atlas_identity`, `atlas_operations`, and `atlas_audit`; S02 must ratify names and transaction participation before the first migration. Application authorization remains mandatory even if later row-level security is added as defense in depth.

## S02 decision register and owner-slice blocks

These were specification gaps, not implementation discretion. S02 resolved `P01-D01..D09`,
`P01-D11`, `P01-D12`, and `P01-D14` in ADR 0014, OpenAPI, and
`identity-access-policy.json`. `P01-D10` remains an explicit block on S04’s synthetic OIDC
configuration, and the exact package/version portion of `P01-D13` remains an explicit block on
S09’s first web product consumer; the Bun/OpenAPI/no-hand-edits strategy is resolved. Neither
deferred owner-slice decision authorizes implementation in S03. The table preserves the original
question each owning slice must continue to satisfy.

| Decision | Required resolution |
|---|---|
| `P01-D01` HTTP operation semantics | Add the missing Phase 01 operations, including exact request/response schemas, safe errors, pagination, masking, rate class, audit effect, ETag/idempotency posture, and compatibility notes. |
| `P01-D02` BFF/CSRF boundary | Define login/callback/logout/step-up return routing, approved return destinations, cookie attributes/path/domain, CSRF token transport and rotation, and CORS behavior on every success/error/preflight path. |
| `P01-D03` session/assurance policy | Choose per-population idle/absolute timeouts, step-up freshness, concurrent session policy, rotation grace (normally none), clock skew, and the exact IdP-outage degraded policy. |
| `P01-D04` principal identity and separation | Define immutable external-subject-to-Atlas-principal mapping, account-linking uniqueness, and how multiple IdP accounts for one human map to one maker identity so an alternate login cannot defeat separation. |
| `P01-D05` organization semantics | Define organization-name normalization/confusable handling, active-tenant switch, invitation issue/accept/expiry/single-use flow, delegation lattice, membership version, and revocation serialization bound. |
| `P01-D06` authorization model | Publish closed permission/role/purpose/masking/delegation catalogues, decision inputs/outputs, decision-ID format, deny/conceal rules, list-count policy, and the decision to avoid an authorization cache in Phase 01. |
| `P01-D07` approval execution | Define action registry, canonical payload bytes/hash algorithm/version, target version binding, eligible-checker snapshot/dynamic rules, expiry, ETag/idempotency, execution trigger, response-loss retry, and atomic target/audit behavior. |
| `P01-D08` merchant credential mode | Reconcile OAuth client credentials with one-time API keys; define key format/entropy/verifier, scope/audience, create/rotate idempotency, overlap duration, in-flight and revocation semantics, rate-limit response, and anomaly inputs. |
| `P01-D09` minimum audit foundation | Define Phase 01 audit schema and append-only grants, required fields, safe reason vocabulary, atomicity, read authorization, retention/classification, and fail-closed/fail-open policy under audit persistence failure. |
| `P01-D10` synthetic Keycloak profile | Define realm clients, public/confidential posture, redirect URIs, PKCE, MFA/step-up test flow, seeded synthetic login mechanism, realm/session settings, and client-secret generation without committing secrets. |
| `P01-D11` token/session secret protection | Decide whether upstream tokens must be retained, their encryption/key version and revocation behavior, opaque-cookie verifier storage, API-key hashing, key outage behavior, and backup classification. Use platform cryptography; do not build an IdP or custom algorithm. |
| `P01-D12` future-surface requirements | Record how `IAM-006` future financial/customer actions and `IAM-026` future restriction changes remain fail-closed and trigger revalidation without fabricating those later-phase endpoints. |
| `P01-D13` generated client | Select a deterministic OpenAPI-to-TypeScript client/docs path compatible with Bun before the web becomes the first product API consumer. Do not create a second hand-edited contract. |
| `P01-D14` cross-phase evidence continuity | Preserve the final Phase 00 catalogue and define a versioned Phase 01 catalogue/integrity policy that supports dirty/pre-commit and committed descendant verification without treating unrelated code/config drift as evidence-only. Retain tamper/stale-source canaries; do not refresh a source revision without owning evidence. |

## Phase 00 revalidation obligations

ADR 0013 is a tripwire, not an obstacle to work around. Revalidation means implementing the now-applicable control and evidence first, then updating the guarded digest/inventory and requirement basis in the same protected pull request.

| First changed surface | Consequence |
|---|---|
| Canonical Phase 01 OpenAPI | No ADR 0013 topology trigger by itself, but it is a sensitive contract change. Update the canonical PRD manifest, contract lint/compatibility evidence, and traceability plan. Do not implement a handler first. |
| First product migration / `atlas_identity` or `atlas_audit` state | Activates `first-product-schema` for `FND-011` and `first-product-durable-state` for `FND-064`. Remove only the now-invalid schema-seed/recovery deferrals; preserve unrelated provider/deployment/encryption limitations. |
| `db/migrations` inventory and manifest | The guarded directory and manifest digest will fail closed. Extend the migration validator intentionally, update current-version/checksum metadata, run empty/current/previous lanes and lock tests, and bind the guard change to product-schema evidence. Merely deleting the product-term ban or refreshing hashes is prohibited. |
| Executable Phase 01 seeds | Extend the deterministic seed catalogue and add an idempotent schema loader that binds synthetic Keycloak subjects, principals, tenants, memberships, roles, and test sessions/credentials only as explicitly designed. Update `FND-011` basis and seed digest. |
| Product durable recovery | A physical backup/WAL/PITR drill must restore identity, audit, session/revocation, membership, approval, and credential state introduced at that point, validate checksums/constraints, and exercise post-restore authorization/revocation checks. Local unencrypted-volume limitations remain explicit; encrypted deployment custody is not claimed. |
| Environment identity configuration | Changes to guarded environment files require typed validation, unique purpose/environment references, safe local/test generated material, and deliberate digest updates. Synthetic Keycloak does not activate the `FND-026` real-provider trigger. Any real IdP or real identity data is blocked. |
| Observability catalogue | Product routes and auth metrics require bounded allowlists, dashboards, alerts, runbooks, mutation tests, and a deliberate guarded-artifact update. This does not activate the queue/job/deployed-alert `FND-042` triggers. |
| First event/consumer/worker/simulator input or broker stream | Not planned. If introduced, stop and satisfy `FND-040` propagation, duplicate/replay/ordering, outbox/inbox, exported trace continuity, and recovery in the same change. |
| First queue/job/retry or deployed alert backend | Not planned. If introduced, stop and satisfy `FND-042` runtime emission/routing/outage evidence rather than citing definition-only metrics. |
| Real identity provider, identity data, deployment, or production claim | Activates ADR 0012/`FND-026`; independent qualified review becomes blocking. It is forbidden within this plan. |

Once product durable state exists, rollback cannot silently restore the feature-free `FND-064` deferral while leaving product state behind. Either remove the entire uncommitted product topology or preserve the stronger recovery control and forward-fix.

## Ordered execution plan

The handoff’s proposed eight-slice decomposition is adjusted to nine slices. A dedicated contract/decision slice is inserted before schema because 14 of 15 Phase 01 paths and material session, tenant, approval, and credential semantics are absent from the canonical contract. The proposed schema slice therefore becomes S03. No implementation slice may leapfrog S02.

### P01-S01 — identity/access/tenancy audit and execution plan

- **Requirements:** audits `IAM-001..007`, `IAM-010..015`, `IAM-020..026`, `IAM-030..034`, and `IAM-040..044`; completes none.
- **Threats/risks:** maps `THR-005..008`, `THR-018`, `THR-020`, `THR-023..024`, `THR-028`, `THR-037..041`, `THR-044`, `THR-053`, `THR-056..058`, `THR-060`; `RSK-005`, `RSK-006`, `RSK-009`, `RSK-024`, `RSK-027`, `RSK-028`, `RSK-031`, `RSK-032`.
- **Contexts/owners:** planning across Identity, Operations, Audit, API/BFF, web, security, data, and platform; principal owner is Identity with Security/Platform review under ADR 0012.
- **Authorization/financial boundary:** documents the future boundary; adds no authorization decision or financial capability.
- **Idempotency/concurrency:** identifies the required session, invitation, membership, approval, and key-rotation races; implements none.
- **Failure boundaries:** failure is an incomplete or stale plan. No database commit or external call exists.
- **Schema/contracts:** none changed.
- **Tests:** repository/Git audit, exact 30-row IAM count, OpenAPI path inventory, AsyncAPI/event search, Keycloak/config/schema/module inspection.
- **Telemetry/runbooks:** inventories gaps only.
- **Rollback/forward fix:** revert or amend this document; do not rewrite historical evidence.
- **Evidence/reproduce:** this document at `UNCOMMITTED_WORKTREE(base=2884484a99eeb2b846a56c90177163e37e419d11)` until committed; Go tests/builds and architecture policy pass. `pwsh -NoProfile -File ./scripts/verify-s08.ps1` reaches S07 PASS and then fails closed on the expected Phase 00 catalogue source mismatch pending `P01-D14`.
- **Phase 00 triggers:** none.

### P01-S02 — canonical contract, ownership, and security-decision closure

**Status:** Complete in the pre-commit worktree. ADR 0014 and
`03-contracts/identity-access-policy.json` resolve or explicitly route every `P01-D01..D14`
decision to its owning slice; no runtime capability is claimed.

- **Requirements:** specifies all 30 IAM rows without claiming runtime satisfaction.
- **Threats/risks:** all Phase 01 threats above, especially `THR-005..008`, `THR-023`, `THR-041`, `THR-056..058`; `RSK-024` and `RSK-027`.
- **Contexts/owners:** Identity owns identity/tenant/authorization/credential semantics; Operations owns approvals; Audit owns records; API owns transport. Identity, Security, Platform, Data, and Privacy owners sign the decisions.
- **Authorization/financial boundary:** publish the permission/delegation/masking/purpose catalogue and high-risk action registry. Future financial action names are policy identifiers only and cannot execute financial behavior.
- **Idempotency/concurrency:** define mutation idempotency, ETags/preconditions, invitation consumption, membership revocation serialization, approval execution, secret-producing retry behavior, and rotation overlap before code.
- **Before/after commit failures:** specify safe error and retry behavior for every operation, including IdP success before Atlas commit, Atlas commit before cookie/secret response, audit failure, response loss, and stale target state.
- **Schema/migration:** ratify context schema ownership and transaction participation; add no migration.
- **OpenAPI/AsyncAPI:** the original 14 missing paths/16 operations are now present. Seven support
  paths/eight operations close login/callback/logout, tenant switch, invitation acceptance, and
  approval creation/execution/cancellation. CSRF, BFF and least-privilege `AtlasKey` security
  schemes are explicit. Phase 01 creates no event/consumer and leaves AsyncAPI runtime inactive.
- **Tests:** OpenAPI examples/reference lint, additive compatibility, unknown-field and mass-assignment schemas, error-path CORS/CSRF contract cases, permission/action catalogue closed-inventory canaries.
- **Telemetry/alerts/runbooks:** define bounded metric and audit field catalogues and runbook ownership without claiming emission.
- **Rollback/forward fix:** before implementation, revert an incorrect additive draft; after publication, use compatible additive correction or a versioned breaking change. Accepted material trade-offs require a superseding ADR.
- **Evidence/reproduce:** `pwsh -NoProfile -File ./scripts/verify-p01-s02.ps1`; focused contract and
  architecture tests, contract lint, missing-path/allow-default mutation canaries, canonical
  manifest validation, and a source-bound Phase 01 evidence catalogue. See `EVD-P01-S02-*`.
- **Phase 00 triggers:** no capability trigger; canonical-manifest and sensitive-contract review are mandatory.

### P01-S03 — identity/audit persistence, deterministic seeds, roles, and recovery revalidation

- **Requirements:** foundation for `IAM-004`, `IAM-010..012`, `IAM-020..021`, and `IAM-025`; no session/org/approval/credential endpoint yet.
- **Threats/risks:** `THR-005..006`, `THR-018`, `THR-020`, `THR-040..041`, `THR-044`, `THR-058`, `THR-060`; `RSK-005`, `RSK-006`, `RSK-009`, `RSK-028`, `RSK-032`.
- **Contexts/owners:** Identity owns principal/subject/tenant/role/session metadata foundations; Audit owns insert-only audit facts; Data/Platform own migration/recovery mechanics.
- **Authorization/financial boundary:** tenant context is explicit; API/worker/reporting roles cannot bypass ownership; no financial tables or permissions.
- **Idempotency/concurrency:** unique external-subject and tenant/member identities; idempotent deterministic seed application; ordered locks and version columns where revocation/role state will race; no external call in a transaction.
- **Before/after commit failures:** migration/seed/audit writes roll back atomically; a failed seed is rerunnable; post-commit response loss exposes no endpoint yet; restore must preserve checksums and deny stale/invalid authority.
- **Schema/migration:** first ordered product migrations for ratified `atlas_identity` and `atlas_audit` namespaces, ownership/grants, global-scope registry, migration metadata, manifest/constants, and realistic role tests. Audit records are insert-only to application roles.
- **OpenAPI/AsyncAPI:** no additional change beyond S02; no outbox/event.
- **Tests:** empty/current/previous migration lanes, idempotent seed load, duplicate/cross-tenant fixtures, missing-tenant-predicate mutation, API/worker/reporting/break-glass permission matrix, long-lock abort, backup/WAL/PITR restore with product rows and post-restore authorization/revocation checks.
- **Telemetry/alerts/runbooks:** migration/seed/audit-persistence signals with bounded fields; update migration, database, backup, and audit-persistence outage runbooks.
- **Rollback/forward fix:** forward-only migration correction. Before release, full removal is allowed only with all uncommitted product state; after release, preserve history and add a forward fix.
- **Evidence/reproduce:** proposed `pwsh -NoProfile -File ./scripts/verify-p01-s03.ps1 -Live`; `go run ./cmd/dbctl verify --migration-dir db/migrations`; `pwsh -NoProfile -File ./scripts/verify-s05.ps1 -Live`; isolated restore report and `EVD-P01-S03-*`.
- **Phase 00 triggers:** exits `FND-011:first-product-schema` and `FND-064:first-product-durable-state`; updates seed, migration manifest/directory, environment if needed, gate policy, architecture tests, traceability, and recovery evidence in the same protected PR.

### P01-S04 — synthetic OIDC BFF and durable session lifecycle

- **Requirements:** `IAM-001..007`, plus `IAM-020..021` and `IAM-025` at the session boundary.
- **Threats/risks:** `THR-006..007`, `THR-020`, `THR-037`, `THR-039`, `THR-044`, `THR-056`, `THR-058`; `RSK-009`, `RSK-024`.
- **Contexts/owners:** Identity domain and `cmd/api` BFF adapters; three synthetic Keycloak realms; web consumes only the opaque application cookie.
- **Authorization/financial boundary:** identity population, active tenant eligibility, assurance, and policy version are server-validated on every protected request. No JWT claim is durable authorization truth. No financial action exists.
- **Idempotency/concurrency:** one-time state/nonce/PKCE transaction, callback replay rejection, session rotation invalidating the old ID, concurrent revoke/use, idle/absolute expiry, step-up expiry, privilege/tenant-version races, and bounded clock skew.
- **Before/after commit failures:** perform discovery/token calls outside database transactions; if IdP succeeds but Atlas commit fails, create no session; if commit succeeds but cookie response is lost, do not resurrect/redisplay a session and require safe re-entry; logout/revoke is idempotent.
- **Schema/migration:** durable sessions, opaque-cookie verifier/digest, external subject links, revocation and assurance versions, OIDC transaction state, and encrypted token metadata only if S02 proves retention is necessary.
- **OpenAPI/AsyncAPI:** implement only S02-defined BFF, `/v1/me`, session, revoke-all, and step-up operations; AsyncAPI unchanged.
- **Tests:** issuer/audience/state/nonce/PKCE/redirect/time/algorithm/key rotation; fixation before/after callback; rotation at login/step-up/privilege/tenant change; revoke-one/all/admin; CSRF; CORS errors; IdP outage; account-enumeration differential; multi-replica revocation; browser storage/cookie/cache inspection.
- **Telemetry/alerts/runbooks:** safe auth outcome categories, step-up, session rotation/revocation, IdP latency/outage, high-risk denial decision IDs; update `IDENTITY_PROVIDER_UNAVAILABLE.md` and add session compromise/revocation runbooks.
- **Rollback/forward fix:** disable protected product routes and reject sessions; preserve revocation/audit facts. Never fall back to browser tokens or fail-open step-up.
- **Evidence/reproduce:** proposed `pwsh -NoProfile -File ./scripts/verify-p01-s04.ps1 -Live`; Go integration/fuzz tests; Bun/browser session suite; exported trace and redacted cookie/storage report; `EVD-P01-S04-*`.
- **Phase 00 triggers:** product migrations/recovery remain active; guarded environment and observability artifacts require deliberate evidence updates. Synthetic Keycloak does not trigger `FND-026`; no worker/event trigger.

### P01-S05 — merchant organizations, memberships, invitations, and active-tenant switching

- **Requirements:** `IAM-010..015`, with `IAM-002`, `IAM-020..026` enforced at organization boundaries.
- **Threats/risks:** `THR-005..006`, `THR-038`, `THR-041`, `THR-044`, `THR-057..058`; `RSK-005`, `RSK-009`.
- **Contexts/owners:** Identity owns organizations/memberships/invitations and active-tenant eligibility; API transports commands; Audit records changes.
- **Authorization/financial boundary:** delegation cannot exceed the inviter; merchant roles never access workforce routes; tenant switch rotates session context and clears client state. No wallet/customer financial resource is introduced.
- **Idempotency/concurrency:** invitation token single use, duplicate issue/accept, role-change ETag, membership removal versus in-flight mutation, tenant switch across tabs, Unicode-normalized/confusable name collisions, and immediate authority-version bump.
- **Before/after commit failures:** invitation/membership/audit changes commit atomically; no email or external call occurs in the transaction; response loss returns the current durable state; removed membership cannot authorize a commit that began from stale browser data.
- **Schema/migration:** organizations, memberships, invitations, delegation/version state, tenant/global scope constraints and indexes; secret invitation verifier stored non-recoverably.
- **OpenAPI/AsyncAPI:** implement only the S02-defined organization/member/invitation/switch/accept operations; no event.
- **Tests:** real PostgreSQL two-tenant negative matrix, valid foreign IDs with timing/count differential, invitation expiry/double acceptance/delegation, `ADV-IAM-007..008`, removed-member stale tab and removal/commit race, Unicode confusable property corpus.
- **Telemetry/alerts/runbooks:** role/membership/invitation/switch/revocation metrics with bounded role/action labels; mass-role-change and cross-tenant-denial alerts; membership compromise runbook.
- **Rollback/forward fix:** disable invitation/role mutation routes while preserving membership truth; forward-fix schema/state. Never restore removed access from Redis or UI state.
- **Evidence/reproduce:** proposed `pwsh -NoProfile -File ./scripts/verify-p01-s05.ps1 -Live`; PostgreSQL concurrency/authorization suite; browser cross-tab test; `EVD-P01-S05-*`.
- **Phase 00 triggers:** migration and product-restore guard updates; no event/job trigger.

### P01-S06 — deny-by-default authorization, purpose, masking, and decision audit

- **Requirements:** `IAM-020..026`, with `IAM-011..013` as repository/query preconditions.
- **Threats/risks:** `THR-005`, `THR-007`, `THR-028`, `THR-038`, `THR-041`, `THR-044`, `THR-057..058`; `RSK-005`, `RSK-006`, `RSK-009`, `RSK-028`.
- **Contexts/owners:** Identity owns the catalogue/evaluator and authorization version; each domain supplies typed resource facts through its application API; Audit owns decision facts.
- **Authorization/financial boundary:** evaluate tenant, principal, action, object, field, purpose, assurance, resource state, and separation. Browser permission lists are presentation hints. Phase 01 uses authoritative PostgreSQL checks and no Redis authorization cache.
- **Idempotency/concurrency:** repeated decisions are side-effect free apart from bounded audit/metric recording; authorization and sensitive mutation must share a transaction or lock/version protocol so revocation cannot race between check and commit.
- **Before/after commit failures:** unknown catalogue entry, missing purpose, database timeout, stale version, or policy-evaluator error denies safely. Audit persistence follows the S02 fail policy; response errors reveal no policy internals.
- **Schema/migration:** source-controlled catalogue plus role bindings/policy versions already owned by Identity; only additive migration if S03/S05 did not supply a required constraint.
- **OpenAPI/AsyncAPI:** enforce S02-defined masks, conceal/deny responses, decision ID, totals/cursors, and permission version; no event.
- **Tests:** complete role/permission/action/object/field/tenant/purpose matrix; direct application-service bypass; unknown permission mutation; list filter before totals/cursors/suggestions; autocomplete enumeration; cross-tenant timing; field masking; multi-replica invalidation without Redis truth.
- **Telemetry/alerts/runbooks:** high-risk/denied decision IDs, latency, safe denial categories, cross-tenant spike and mass-role-change alerts; authorization incident/invalidation runbook.
- **Rollback/forward fix:** route disable or narrower policy only; never fail open or re-enable a removed grant. Catalogue corrections are versioned and audited.
- **Evidence/reproduce:** proposed `pwsh -NoProfile -File ./scripts/verify-p01-s06.ps1 -Live`; authorization matrix artifact, mutation report, query-plan/count differential report, exported traces; `EVD-P01-S06-*`.
- **Phase 00 triggers:** observability guarded-artifact revalidation and any additive migration/recovery update; no `FND-040`/queue trigger.

### P01-S07 — maker-checker approval foundation and execution-time reauthorization

- **Requirements:** `IAM-006`, `IAM-030..034`, with `IAM-020..026` and `IAM-025` mandatory at create/decision/execute.
- **Threats/risks:** `THR-007..008`, `THR-018`, `THR-024`, `THR-040..041`, `THR-058`; `RSK-006`, `RSK-028`.
- **Contexts/owners:** Operations owns approval state; Identity supplies principal/permission/assurance decisions; target context owns the approved command; Audit owns facts.
- **Authorization/financial boundary:** only S02-approved synthetic non-financial action types are executable in Phase 01. Generic approval does not grant ledger/database authority and cannot mutate another context’s tables directly.
- **Idempotency/concurrency:** canonical payload hash/version, unique logical request where required, optimistic version/ETag, one terminal decision, maker/checker identity across role/account changes, expiry/cancel/supersede races, and safe retry after `ExecutionFailed`.
- **Before/after commit failures:** create/decision/audit commit atomically; execution rechecks checker/maker separation, permission, step-up, target state/version, and payload in the target transaction. A failure before commit has no target effect; response loss after commit returns the existing execution result and never repeats the effect.
- **Schema/migration:** Operations-owned approval/request/decision/execution records with immutable payload bytes/hash metadata, target version, expiry, reason codes, and constraints; no arbitrary JSON-to-domain write path.
- **OpenAPI/AsyncAPI:** implement S02-defined approval list/get/decision/execution semantics; no approval event or worker.
- **Tests:** state-machine/property tests; maker role change and alternate login; database payload tamper; role/target/step-up expiry between creation, decision, confirmation, and commit; stale ETag; simultaneous checkers; terminal replay; audit failure.
- **Telemetry/alerts/runbooks:** approval age/status/decision/execution conflict and integrity-failure metrics; stale/expired approval alerts; approval integrity/recovery runbook without direct database edits.
- **Rollback/forward fix:** disable new approval/execution routes; preserve immutable decisions and audit. Forward-fix state machines; never edit approved payload/status directly.
- **Evidence/reproduce:** proposed `pwsh -NoProfile -File ./scripts/verify-p01-s07.ps1 -Live`; model/concurrency/mutation/PostgreSQL tests; `EVD-P01-S07-*`.
- **Phase 00 triggers:** migration/product-restore and observability guard updates; no event/job trigger.

### P01-S08 — merchant API credentials, one-time secret, rotation, and scoped anomaly controls

- **Requirements:** `IAM-006`, `IAM-020..026`, `IAM-040..044`.
- **Threats/risks:** `THR-020`, `THR-023..024`, `THR-041`, `THR-044`, `THR-056..058`; `RSK-009`, `RSK-017`, `RSK-024`.
- **Contexts/owners:** Identity owns credential metadata/verifier/rotation; API authenticates and authorizes; Audit records lifecycle; Redis may hold only reconstructible rate counters.
- **Authorization/financial boundary:** tenant/environment/audience/scope/status are checked against PostgreSQL on every request. A credential cannot authorize any money operation absent from Phase 01.
- **Idempotency/concurrency:** secret-producing create/rotate response-loss policy, one-time display, unique key ID, overlap activation, old/new in-flight requests, explicit old-key revocation, concurrent rotate/delete, expiry, last-used update contention, and bounded rate counters.
- **Before/after commit failures:** generate secrets with platform randomness before the bounded transaction; persist only the S02-approved verifier/encrypted form plus atomic audit; after commit/response loss never reveal the secret again. Redis loss may reset throttling only within an explicit conservative policy and never re-enable a revoked key.
- **Schema/migration:** credential, scope, rotation-link/version, verifier/key-version, lifecycle, expiry, last-used and anomaly metadata with tenant/environment constraints and no recoverable plaintext.
- **OpenAPI/AsyncAPI:** implement S02-defined create/list/rotate/revoke and machine authentication scheme; secrets are write-only and one-time; no event.
- **Tests:** entropy/format, secret scan/log/browser/database inspection, unknown field/mass assignment, wrong tenant/environment/audience/scope, step-up, response loss, old/new concurrent traffic, revocation boundary, Redis outage, per-key/tenant rate isolation, algorithm/key downgrade fuzzing.
- **Telemetry/alerts/runbooks:** creation/rotation/revocation, auth outcomes, rate rejection and deterministic new-network anomaly with bounded identifiers excluded from labels; credential compromise/rotation runbook.
- **Rollback/forward fix:** disable creation/rotation and reject affected credentials; preserve revocation/audit; rotate rather than recover. Never restore plaintext or fail open to an older key.
- **Evidence/reproduce:** proposed `pwsh -NoProfile -File ./scripts/verify-p01-s08.ps1 -Live`; concurrent auth/rotation suite, secret-sink scan, rate-limit/anomaly report; `EVD-P01-S08-*`.
- **Phase 00 triggers:** migration/product-restore, secret-boundary, environment, and observability guard updates as applicable; no real provider/managed secret claim and no worker/event trigger.

### P01-S09 — frontend flows, operational proof, adversarial acceptance, and phase closure

- **Requirements:** integrates and evidences all 30 IAM requirements; no row changes status without its own test/evidence link.
- **Threats/risks:** closes or explicitly dispositions the complete Phase 01 threat/risk set without lowering residual risk beyond evidence.
- **Contexts/owners:** web plus Identity, Operations, Audit, API, Security, Privacy, Platform, and Data owners.
- **Authorization/financial boundary:** separate customer, merchant, and workforce routes; no staff impersonation; browser state is non-authoritative; no financial behavior.
- **Idempotency/concurrency:** browser double-submit, stale tab, cross-tab tenant switch/logout, back-forward cache, step-up return without replay, approval/credential response loss, and API key in-flight rotation.
- **Before/after commit failures:** every UI flow renders loading, empty, denied, stale, failed, committed-but-response-lost, and recovery states from durable server truth. Client cache clears on logout, tenant switch, session expiry, and revocation.
- **Schema/migration:** no new schema unless an evidenced closure defect requires a forward migration; any such change reruns product recovery.
- **OpenAPI/AsyncAPI:** final conformance against the sole canonical OpenAPI; AsyncAPI remains unimplemented for Phase 01 unless separately approved with full trigger obligations.
- **Tests:** all 15 phase “tests most agents skip,” `ADV-IAM-001..015`, PostgreSQL authorization matrix, real synthetic Keycloak browser journey, accessibility, CORS/CSRF, cache/storage, failpoints, mutation, bounded-resource and restore checks.
- **Telemetry/alerts/runbooks:** emit/test the phase metric catalogue; dashboard authentication/authorization/session/member/approval/credential signals; alert tests for break-glass, denial spike, credential anomaly, mass role change, and repeated step-up failure; all runbooks exercised synthetically.
- **Rollback/forward fix:** disable affected route/action narrowly, preserve audit/revocation/approval/credential truth, forward-fix migrations, and keep synthetic warnings. No direct database edits.
- **Evidence/reproduce:** proposed `pwsh -NoProfile -File ./scripts/verify-p01.ps1 -Live`; `go test ./...`; `go build ./cmd/api ./cmd/worker ./cmd/simulator`; Bun tests/build; browser acceptance; Phase 00 static/live/history/supply regressions when runtime/config/image/dependency changes justify them; `EVD-P01-S09-*`.
- **Phase 00 triggers:** all prior product-schema/recovery/guard changes must be closed. A full live S08 regression is required because runtime, identity configuration, migrations, recovery, web, and images changed. Hosted release/merge remains separately authorized.

## Tests most agents skip coverage

| Required test | Planned slice and minimum proof |
|---|---|
| 1. Valid cross-tenant ID without timing/count leakage | S05/S06: two-tenant PostgreSQL fixtures and differential body/status/count/timing bounds |
| 2. Authorization before list totals | S06: seeded unauthorized rows plus query/mutation proof for totals/cursors |
| 3. Removed member using stale open tab | S05/S09: removal/version race across API replicas and browser tab |
| 4. Role downgrade between approval create and execute | S07: execution-time authorization in target transaction |
| 5. Checker gains maker role or uses alternate account | S02 identity-link decision plus S07 immutable principal separation tests |
| 6. Database fixture changes approved payload | S07: canonical hash/version mismatch blocks execution and audits integrity failure |
| 7. Step-up expires between confirmation and commit | S04/S07/S08: injected clock and transaction-bound freshness recheck |
| 8. Session fixation before/after callback | S04: old cookie/transaction IDs rejected across login and step-up |
| 9. Back-forward cache after logout | S04/S09: real browser `pageshow`/BFCache/storage/cache assertions |
| 10. Old/new API keys in flight during rotation | S08: concurrent barrier test across overlap and explicit revocation |
| 11. Unicode-confusable organization names | S05: normalization/confusable corpus and authorization-ambiguity rejection |
| 12. Search autocomplete enumeration | S06: authorized filtering before suggestion/rank/count/timing |
| 13. Permission invalidation across API replicas | S06: PostgreSQL authority/version checks with Redis absent/stale |
| 14. CSRF against cookie mutation | S04/S09: valid session with missing/wrong/cross-origin token fails |
| 15. Credentialed CORS on error paths | S02/S04/S09: preflight and 4xx/5xx header matrix with exact origins |

The phase also runs `ADV-IAM-001..015`, including recovery enumeration, workforce purpose/reveal, wrong audience, and token algorithm/key confusion. A named test is not evidence until it runs against the real boundary it claims.

## Evidence, status, and closure policy

Planned evidence is additive under:

```text
evidence/phase-01/
  architecture/
  database/
  identity-session/
  tenancy-authorization/
  approvals/
  credentials/
  acceptance/
```

Use stable IDs such as `EVD-P01-S04-001`. Every report records requirement/threat/test IDs, exact source and environment revisions, seed/time, command, expected/observed result, sanitization statement, SHA-256, limitations, and revalidation date. Pre-commit evidence uses `UNCOMMITTED_WORKTREE(base=<40-hex>)`; post-commit evidence is added rather than overwriting history.

At each implementation slice:

1. update only the IAM traceability rows actually evidenced;
2. update applicable threat/risk status and residual rationale without bulk-closing threats;
3. add evidence and evidence-index entries;
4. update implementation status and known limitations;
5. update canonical `MANIFEST.sha256` for canonical PRD edits;
6. deliberately update Phase 00 guards only after newly applicable controls pass;
7. include all six ADR 0012 sensitive-change attestations in the protected pull request.

Phase 01 cannot close until the acceptance journey runs for customer, merchant member, support, risk, and finance identities; cross-tenant attempts fail; maker permission change blocks approval execution; credential rotation succeeds under in-flight traffic; and the complete synchronous audit chain is inspectable. PostgreSQL integration, authorization matrix, adversarial/browser tests, telemetry, alert tests, runbooks, restore, sanitized evidence, content claims, and known limitations must all pass.

No hosted release, merge, production/reference deployment, real identity provider, or real identity data is authorized by this plan.

## Exact next implementation checkpoint

The next checkpoint remains within **P01-S04 — synthetic OIDC BFF and durable session
lifecycle**: implement administrator security revocation with audit and concurrency proof before
moving to S05. Contracted step-up `Idempotency-Key` replay/conflict semantics and live
higher-assurance rotation are committed at `015911fdc586b4e7b65a80d29cdf06b799e37fc4`.

S03 completed the first Go/database implementation slice. S04 core now composes that boundary
through the ADR 0014-owned OpenAPI surface. The current checkpoint:

- preserves and revalidates the `FND-011:first-product-schema` and
  `FND-064:first-product-durable-state` controls;
- uses real PostgreSQL roles for session/revocation concurrency, migration, lock, permission, and
  backup/WAL/PITR checks;
- validates issuer/audience/state/nonce/PKCE/redirect/timing and rotates durable encrypted
  application sessions through customer/merchant flows;
- persists scoped hash-only step-up replay state, returns the exact stored response for matching
  retries, rejects changed requests, and proves live LoA 2 session/CSRF rotation with old-cookie
  rejection;
- keeps workforce baseline authentication fail-closed and existing low-risk sessions available
  during an injected provider outage;
- preserves historical catalogues and adds source-bound `EVD-P01-S04-*`;
- adds no Redis authorization truth, event/outbox, worker job, authorization/approval/credential
  behavior, frontend product behavior, or financial state.

S04 remains open until administrator security revocation, enumeration timing, and complete browser
cache/storage/BFCache evidence are implemented. The checkpoint makes no phase-wide IAM completion
claim.
