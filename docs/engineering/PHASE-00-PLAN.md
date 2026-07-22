# Phase 00 audit and execution plan

## Scope and audit method

This plan audits the current repository against all 37 requirements in [Phase 00](../atlas-prd/02-phases/PHASE-00_ENGINEERING_FOUNDATION.md). It does not treat specification text, a dependency name, or a configuration file alone as runtime proof. Status meanings:

- **Satisfied:** the requirement is a decision/definition and the authoritative definition exists, with no contradictory implementation.
- **Partial:** concrete design/evidence exists, but an enforcement, implementation, test, or operational proof required by the wording/Definition of Done is missing.
- **Absent:** no repository artifact performs the required behavior.
- **Conflicting:** two authoritative sources require incompatible behavior and no accepted ADR resolves it.
- **Unassessed:** evidence could not be inspected.

The audit uses [project-wide gates](../atlas-prd/00-master/03_REQUIREMENTS_AND_QUALITY_GATES.md), [system architecture](../atlas-prd/01-architecture/00_SYSTEM_ARCHITECTURE.md), [security model](../atlas-prd/01-architecture/01_SECURITY_AND_TRUST_MODEL.md), [reliability/observability/DR](../atlas-prd/01-architecture/04_RELIABILITY_OBSERVABILITY_DR.md), [accepted ADRs](../atlas-prd/06-governance/adrs/), [traceability matrix](../atlas-prd/06-governance/REQUIREMENTS_TRACEABILITY.csv), [threat register](../atlas-prd/06-governance/THREAT_REGISTER.csv), [adversarial catalogue](../atlas-prd/04-testing/ADVERSARIAL_TEST_CATALOG.md), [test strategy](../atlas-prd/04-testing/TEST_STRATEGY.md), [security verification plan](../atlas-prd/04-testing/SECURITY_VERIFICATION_PLAN.md), and [Definition of Done](../atlas-prd/06-governance/DEFINITION_OF_DONE.md).

## Repository audit summary

S01 through S06 are implemented as a feature-free engineering foundation. Git is valid on branch `main`; the configured GitHub origin supplies the Go module identity. In addition to the S01–S05 scaffold, API edge, synthetic topology, and database/recovery controls, S06 adds source-redacted JSON logs, bounded OTLP traces/metrics, an exported API/readiness/database golden trace, an executable 60-threat coverage index, versioned secret-provider boundaries, and security/telemetry runbooks. It still has no product behavior or schema, CI, broker stream, identity exchange, worker job, managed production secret provider, alert backend, or financial state. S05 is clean-worktree post-commit verified as `5ea77fc`; S06 is pre-commit verified on base `7a08056`. The PRD pack remains structurally valid after traceability updates, and the architecture test rejects reintroduced root duplicates.

| Status | Count |
|---|---:|
| Satisfied | 24 |
| Partially satisfied | 7 |
| Absent | 6 |
| Conflicting | 0 |
| Not yet assessed | 0 |

## Requirement-by-requirement audit

Slice references (`S01`–`S08`) identify the primary sequencing below. Threat IDs refer to the canonical [threat register](../atlas-prd/06-governance/THREAT_REGISTER.csv). Tests include the minimum proof implied by Phase 00 and the Definition of Done; evidence must also carry source/config revision, seed, reproduce command, expected/observed result, sanitization, digest, limitations, and revalidation date.

| Requirement | Status | Existing repository evidence | Missing implementation and completion proof | Applicable threats | Primary sequence |
|---|---|---|---|---|---|
| `FND-001` | Satisfied | Roadmap-aligned ownership tree, [root commands](../../README.md), pinned Go metadata, layout/canonical-source tests, and [S01 evidence](../../evidence/phase-00/architecture/S01-boundary-report-v3.md) | Reverify from a clean clone; later slices must replace placeholders with owned artifacts without creating duplicate contract/spec truth. | `THR-013`, `THR-042`, `THR-060` | S01 complete; preserve |
| `FND-002` | Satisfied | Rules in [module boundaries](MODULE_BOUNDARIES.md) are enforced by a clean import scan and seeded cross-context persistence-import rejection in `internal/architecture`; the S05 real-role matrix prevents application schema bypass. | Extend the checker and role grants as real context APIs/tables appear; no product table ownership is claimed before its owning phase. | `THR-040`, `THR-054`, `THR-060` | S01 complete; S05 DB foundation proof |
| `FND-003` | Satisfied | Separate `cmd/api`, `cmd/worker`, and `cmd/simulator` entry points pass package tests and build independently. The API now has a bounded HTTP lifecycle and only the three S03 operational routes; worker and simulator remain inert. | S04 must provide environment composition and real readiness probes. No product endpoint, identity, job, provider scenario, or external dependency is claimed. | `THR-025`, `THR-042`, `THR-060` | S01/S03 complete; extend S04 |
| `FND-004` | Satisfied | React + TypeScript is selected in the [PRD README](../atlas-prd/README.md), [product charter](../atlas-prd/00-master/00_PRODUCT_CHARTER.md), and [architecture §3](../atlas-prd/01-architecture/00_SYSTEM_ARCHITECTURE.md); no Vue implementation exists | Preserve with one frontend package/lockfile and CI rule rejecting a second framework. The later shell must prove route/identity separation and accessible failure states. | `THR-029`, `THR-060` | Preserve in S01/S04 |
| `FND-005` | Satisfied | Narrow [platform primitives](PLATFORM_PRIMITIVES.md) implement bounded integer money/currency, cryptographically random opaque IDs, injectable UTC clocks, explicit actor and correlation/causation contexts, and data-minimizing stable domain errors. Table/property tests, canonical JSON fixtures, three 100-execution fuzz campaigns, clock-boundary cases, and a killed currency-guard mutation are recorded in [S02 evidence](../../evidence/phase-00/primitives/S02-primitives-report.md). | Add a frontend consumer for the shared decimal-string fixture only when the frontend toolchain is authorized; keep product authorization, tenancy, persistence, and telemetry semantics out of these primitives. | `THR-001`, `THR-002`, `THR-009`, `THR-030`, `THR-031` | S02 complete; preserve |
| `FND-006` | Satisfied | The architecture checker rejects named `float32`/`float64` money and direct domain `time.Now()` (including aliases/dot imports), while safe-control fixtures permit non-money measurements and the platform clock adapter. Seeded canaries are captured in [S02 evidence](../../evidence/phase-00/primitives/S02-primitives-report.md). | Wire the existing command into protected CI in S07/`FND-020`; reviewers must still reject deliberately obscure aliases/names because this is a conservative syntax check. | `THR-001`, `THR-029`, `THR-031` | S02 complete; CI enforcement S07 |
| `FND-010` | Partial | S04 defines the complete constrained topology; S08 re-ran every named service, golden trace, outage, restore, bounded teardown, and then passed a detached exact-revision clone with empty clone-local dependency/build caches. | The full container wrapper still has not passed on a separate independently administered supported machine. | `THR-019`, `THR-025`, `THR-042`, `THR-045` | S08 same-host clean clone pass; independent host open |
| `FND-011` | Partial | The closed seed manifest validates two tenants, three users, two account identity fixtures without financial state, eight named provider scenario IDs, fixed virtual time, tenant ownership, and repeatable SHA-256. | Application schemas and executable simulator/provider contracts do not exist; S04 deliberately does not load database rows or claim scenario behavior. | `THR-043`, `THR-045`, `THR-060` | S04 identity/catalogue; executable contracts later |
| `FND-012` | Satisfied | Typed configuration and canaries require synthetic labels/services, fixed local Compose endpoints or reserved `atlas.invalid` reference hosts, loopback local surfaces, environment-scoped secret references, and no real service credentials. | Revalidate whenever a real deployment/provider adapter is introduced; this is portfolio-environment proof only. | `THR-020`, `THR-045` | S04 complete; preserve |
| `FND-013` | Satisfied | Guarded reset accepts only local/test, prints the resolved target, requires the exact environment phrase, validates containment, and rejects wrong-environment and production-reference canaries before deletion. | Compose-volume deletion remains local namespace-scoped; production recovery is not represented by this command. | `THR-019`, `THR-045` | S04 complete; preserve |
| `FND-020` | Partial | S07 PR run `29928153984` passed all five configured hosted jobs against the merged head, including Linux race/Gosec, real PostgreSQL/NATS, CodeQL, contracts/history, and supply chain; [hosted evidence](../../evidence/phase-00/supply-chain/S07-hosted-pr-verification.md) is revision-bound. | GitHub reports no `main` ruleset, and the newly added S08 constrained-pool race lane has not run in hosted Linux. Configure required checks and retain the S08 job/ruleset proof. | `THR-009`, `THR-013`, `THR-014`, `THR-030`, `THR-042`, `THR-060` | Hosted execution observed; enforcement/S08 race open |
| `FND-021` | Satisfied | S05 live scripts require PostgreSQL 18.4 and NATS 2.14.0 JetStream containers. Real role, migration, lock, backup, and recovery commands cannot be satisfied by repository mocks; the S07 PR integration lane calls them. | Obtain the hosted PR run. Duplicate/replay cases are inapplicable until an outbox/consumer exists and must be added with that behavior. | `THR-014`, `THR-015`, `THR-019`, `THR-040`, `THR-054` | S05 complete locally; S07 configured |
| `FND-022` | Satisfied | S07 generates and hashes backend source, frontend source, backend image, and web image SPDX SBOMs; expected identities, denied licenses, and critical vulnerabilities are checked. | Attach new SBOMs to the committed hosted release; local disposable SBOMs are not release artifacts. | `THR-013`, `THR-060` | S07 local complete; release attachment S08 |
| `FND-023` | Partial | Committed S07 backend/web images use digest-pinned bases, source labels, exact revision tags, recorded SHA-256 identity, minimal runtime files, and non-root/read-only/capability-dropped/no-new-privileges execution; the hosted PR supply-chain job passed. | No GHCR digest is published or promoted; verify one registry digest through the hosted release/promotion path. | `THR-013`, `THR-060` | S07 mechanics/hosted PR; release open |
| `FND-024` | Partial | The release workflow configures digest-only keyless Cosign signatures and GitHub build/SBOM attestations, verifies expected repository/workflow identity, and has no unsigned fallback. | The hosted release run list is empty. Run it from committed S08 source and retain signature, provenance, SBOM attestation, tamper/wrong-source, and unavailable-signing evidence. | `THR-013`, `THR-018`, `THR-020`, `THR-060` | S07 design/config; hosted release open |
| `FND-025` | Satisfied | Fixed throwaway empty and version-1 databases migrate to version 2, current application repeats idempotently, cleanup is bounded, and `ADV-REL-008` holds an exclusive lock long enough to prove the migration aborts before release and leaves no column; S07 wires the lane into PR CI. | Obtain the hosted PR run and add real product-schema compatibility fixtures only when such a schema exists. | `THR-014`, `THR-019`, `THR-060` | S05 complete locally; S07 configured |
| `FND-026` | Partial | `CODEOWNERS` covers ledger, authorization, migrations, cryptography, CI, deployment, contracts, and supply-chain locks; a static test rejects coverage drift. | The rulesets API is empty and PR #1 was self-authored. Configure and evidence independent code-owner review, stale-approval dismissal, conversation resolution, and audited bypass constraints. | `THR-007`, `THR-013`, `THR-014`, `THR-018`, `THR-020`, `THR-040` | File/policy complete; hosted enforcement open |
| `FND-027` | Satisfied | Canonical OpenAPI/AsyncAPI receive exact-version YAML/reference lint, Git-baseline removal comparison, live API examples, and seeded path/field/message/reference failures through `contractctl`. | Revalidate on every contract change and extend rules/examples with the first product contract; hosted execution is tracked under FND-020. | `THR-042`, `THR-060` | S07 local complete |
| `FND-030` | Satisfied | Four strict typed JSON configurations define local, test, staging, and production-reference with closed fields, exact origins, fixed service portfolios, surfaces, flags, banners, and credential references. | Deployment-specific secret backends and drift automation remain S07. | `THR-020`, `THR-042`, `THR-045`, `THR-060` | S04 complete; enforce S07 |
| `FND-031` | Partial | All four configurations have non-overlapping environment/purpose credential references; generated local/test password/token fingerprints are unique and never recorded as evidence values. | Staging/production-reference provisioning, signing/encryption material, rotation grace, restore, and secret-manager fingerprints remain S07/S08. | `THR-020`, `THR-023`, `THR-045` | S04 reference separation; finalize S07/S08 |
| `FND-032` | Satisfied | React customer, merchant, and workforce shells render the persistent runtime synthetic banner; live/browser proof covers all routes, no-store responses, empty browser storage, logout cache clear, and back-navigation denial. | Responsive and broader accessibility matrices expand in S07; no product UI is claimed. | `THR-045`, `THR-046`, `THR-047` | S04 complete; preserve |
| `FND-033` | Satisfied | Typed flag metadata requires owner, future expiry, default, risk, and rollback; validation rejects stale/incomplete/high-risk-default-on state, while immutable concurrent evaluation fails closed on unknown keys and defaults on source outage. | Financial-state flags remain prohibited until a later reviewed design; S04 has only a low-risk shell flag. | `THR-024`, `THR-036`, `THR-042`, `THR-059` | S04 complete; preserve |
| `FND-040` | Partial | Request/correlation/W3C context is validated at the API edge and exported through linked API, readiness, and database spans. Duplicate/malformed metadata fuzz and fixed-parent live continuity pass. | Worker/simulator have no request/event/job input, and no outbox/event/retry path exists; those propagation facets remain unimplemented rather than simulated. | `THR-009`, `THR-015`, `THR-030`, `THR-060` | Current path complete S06; future flows S08/later |
| `FND-041` | Satisfied | API/worker/simulator runtime and bootstrap logs use a closed JSON schema with source redaction. CRLF/forged-field input is rejected before sink write; raw SDK/server diagnostics are suppressed at source. | Re-run sink scans for every new field and connect deployment retention/access controls in S07/S08. | `THR-009`, `THR-030` | S06 complete; preserve |
| `FND-042` | Partial | HTTP RED, database readiness/pool, and build/revision metrics are emitted with a closed label/cardinality catalog. Dashboard, alert owner/severity/rationale/runbook/test metadata and mutation canaries pass. | Queue lag and retry metrics are definition-only because no queue/job exists. No deployed alert engine or routing receiver has been selected or tested. | `THR-010`, `THR-025`, `THR-030`, `THR-048`, `THR-049` | Current metrics S06; runtime routing S07/S08 |
| `FND-043` | Satisfied | A fixed inbound W3C trace is exported through `GET /health/ready`, the readiness child, and `database.schema_readiness`; the live collector asserts trace ID/names plus RED/pool/build metrics. Collector outage leaves readiness `200`. | This is the complete current synthetic request path, not evidence for nonexistent web/event/worker/product flows. Revalidate against the final reference deployment in S08. | `THR-009`, `THR-030`, `THR-042`, `THR-060` | S06 complete for current path; preserve |
| `FND-050` | Satisfied | [Phase 00 threat model](../security/PHASE-00-THREAT-MODEL.md) defines the current context/DFD, six trust boundaries, and initial STRIDE review. A machine-checked index links every canonical `THR-001..060` to applicability, boundary, control, evidence, owner, and residual risk. | Human review is recorded as the S06 engineering review; update the model before every new endpoint/event/job/identity/data boundary. | All register threats | S06 baseline complete; preserve |
| `FND-051` | Satisfied | Canonical classification/logging rules are preserved; the [S06 field inventory](../security/PHASE-00-DATA-LOG-INVENTORY.md) and executable `FieldPolicies` constrain purpose, classification, retention, and sink admission. | Every future field/sink must extend the inventory and tests before use. | `THR-009`, `THR-017`, `THR-026..030`, `THR-045`, `THR-050`, `THR-057` | Preserved and enforced S06 |
| `FND-052` | Satisfied | The provider-neutral secret boundary binds environment/purpose/algorithm/version, enforces minimum versions and bounded grace, revocation and provider outage, wipes callback copies, and rejects cross-boundary material reuse. Rotation/recovery is documented. | No managed production provider, HSM, custody, or product cryptography is selected or claimed; those become deployment-specific gates before use. | `THR-020`, `THR-031`, `THR-056` | S06 abstraction complete; deployment adapter later |
| `FND-053` | Satisfied | The API edge sets secure/cache headers, enforces exact-origin CORS, rejects wildcard origins, bounds headers/body/read/write/idle/readiness work, refuses compression/query/body on foundation routes, recovers safely, and emits catalogued topology-free problems. The slow-header, streamed overflow, `ADV-RES-001` facets, panic-detail, and route-inventory tests pass. | Re-run in CI and against the later reference deployment/proxy in S07/S08; future product handlers need schema-specific controls and may not weaken this edge policy. | `THR-010`, `THR-041`, `THR-042`, `THR-043` | S03 complete; preserve |
| `FND-054` | Satisfied | Exact Go/application/Bun pins, frozen lockfile, immutable action SHAs, tag-plus-digest images, hash-verified scanner archives, Govulncheck/license/CVE gates, Dependabot ecosystem coverage, and normal/emergency [update policy](TOOLCHAIN_POLICY.md) pass S07 tests. | Continue monthly/emergency review and capture hosted update PR evidence; any pin change invalidates this evidence. | `THR-013`, `THR-060` | S07 complete locally; ongoing operation |
| `FND-055` | Satisfied | `SECURITY.md`, vulnerability-disclosure, and dependency-emergency runbooks define private intake, severity, containment, evidence, patch/rebuild/revoke, communication, retrospective, and revalidation steps. The S06 report records a synthetic dependency-advisory tabletop that refuses to invent a fixed version. | Hosted private-reporting configuration and a real advisory drill remain operational environment evidence, not missing runbook behavior. | `THR-013`, `THR-020`, `THR-060` | S06 complete; exercise again S07/S08 |
| `FND-060` | Satisfied | [ADR 0010](../atlas-prd/06-governance/adrs/0010-native-postgresql-migrations-and-recovery.md) and bootstrap create distinct migration, API, worker, reporting-read, expired break-glass, and backup identities with distinct generated credentials. A real allow/deny matrix proves the boundary. | Replace the local bootstrap identity and provision environment-managed credentials before deployment; extend grants only with owned product schemas. | `THR-007`, `THR-014`, `THR-018`, `THR-040`, `THR-054` | S05 complete |
| `FND-061` | Satisfied | Real PostgreSQL attempts prove API/worker/reporting cannot create schemas/tables, alter/drop the probe, create disallowed temporary state, grant effective public access, or assume migration role. | Re-run for every future product-schema grant and database upgrade. | `THR-014`, `THR-040` | S05 complete; preserve |
| `FND-062` | Satisfied | Closed `db/migrations/MANIFEST.sha256` binds every released SQL/metadata pair; validator rejects unmanifested/reordered/malformed inventory and the changed-SQL/deleted-metadata canaries are killed. Forward correction is documented and clean verification is bound to `5ea77fc`. | Wire the same check into protected CI before treating it as a hosted release gate. | `THR-014`, `THR-019`, `THR-060` | S05 complete; enforce S07 |
| `FND-063` | Satisfied | Closed metadata requires lock/data/plan/space/forward-fix/rollback analysis and bounded timeouts. Representative foundation rows and a three-second exclusive lock prove the ALTER aborts in one second, rolls back, and leaves the database usable. | Add product-scale representative datasets, query plans, disk forecasts, and forward-fix rehearsals with each future schema. | `THR-014`, `THR-019`, `THR-051` | S05 foundation complete; preserve |
| `FND-064` | Partial | S08 re-ran native base backup, WAL archive, `pg_verifybackup`, and internal-only target-time recovery; the migration checksum and pre-deletion marker passed with observed backup duration 27 seconds and restore RTO 62 seconds. | Local volumes are unencrypted. No product schema/seed/object/key/outbox/inbox/idempotency/financial replay exists, and local timing is not a production RPO/RTO claim. | `THR-019`, `THR-020`, `THR-027`, `THR-054` | Local recovery revalidated; encryption/product replay open |

## Unnumbered Phase 00 deliverables

These are release-blocking even though the phase file does not assign them `FND-*` IDs.

| Deliverable | Current state | Planned slice |
|---|---|---|
| `GET /health/live`, `GET /health/ready`, `GET /version` | Implemented contract-first. Liveness is process-only; readiness uses bounded real dependency and application-role migration-version/checksum probes and fails closed without topology; version is limited to revision/contract/build time. | S03 API and S04/S05 real probes complete; preserve |
| Common OpenAPI components for problem details, request/correlation IDs, idempotency, cursor pagination, money, actor/session, ETags/versions | S03 added request/correlation/trace parameters and operational response headers while preserving the existing money/idempotency/cursor/ETag/Problem components. Actor/session coverage and full semantic contract tooling remain later work. | S03 partial; finish S07 |
| Customer/merchant/workforce React shells, accessible primitives, generated/verified client, safe cache clearing | Three synthetic function-component route shells, persistent banners, no-store delivery, cache clear, and logout/back proof exist; no product client or UI exists. | S04 shell complete; generated client/contract enforcement S07 |
| Ten named “tests most agents skip” | S02–S06 implement named tests #2, #3, #6, #8, and #9 plus adjacent mutation/failure cases; S08 must report all ten. | S02–S08 |
| Eight runbook topics | Database unavailable, migration failure, backup corruption, broker unavailable, IdP unavailable, and object storage unavailable exist at the applicable foundation depth. Secret, dependency, telemetry, and deployment operating procedures remain later work. | S03–S07 |
| Acceptance gate and content artifacts | No clean-clone/start/trace/sign/restore evidence; planning-only content exists. | S08 after evidence |

## Ordered execution plan

S01 is complete with evidence bound to initial scaffold revision `f72f5468c52a05a442fa0efbbe996fa16450a2bb`. S02 implementation is committed as `dc638d2`; the owner-authorized canonical-PRD cleanup is committed and post-commit verified as `240adbf`. S03 is post-commit verified as `b5fd25b`; S04 is post-commit verified as `39121a3`; S05 is post-commit verified as `5ea77fc`. No remote push is claimed until it succeeds.

| Order | Slice | Primary requirements | Hard prerequisite |
|---:|---|---|---|
| 1 | S01 — Versioned repository and process-boundary scaffold | `FND-001..004`, start `FND-054` | Confirm intended Git root/toolchain policy |
| 2 | S02 — Safe cross-cutting primitives and static bans | `FND-005..006` | S01 build/module boundaries |
| 3 | S03 — Contract-first health, HTTP safety, and trace seed | `FND-003`, `FND-040`, `FND-043`, `FND-053`; API foundation | S01/S02 and active contract-location decision |
| 4 | S04 — Reproducible synthetic local/reference environment | `FND-010..013`, `FND-030..033`; frontend foundation | S03 health endpoints; broker/IdP/config decisions |
| 5 | S05 — Database roles, migration safety, and recovery foundation | `FND-021`, `FND-025`, `FND-060..064` | S04 PostgreSQL/object/local environment |
| 6 | S06 — Observability and security operating baseline | `FND-040..043`, `FND-050..053`, `FND-055` | S03/S04 runtime; secret/key decision |
| 7 | S07 — CI, contracts, and supply-chain integrity | `FND-020..027`, `FND-054`; enforce earlier checks | S01–S06 executable commands and Git/CI host |
| 8 | S08 — Phase 00 acceptance, restore, and evidence release | All Phase 00 gates, especially `FND-010`, `FND-022..024`, `FND-040..043`, `FND-064` | S01–S07 complete |

### S01 — Versioned repository and process-boundary scaffold

- **Status:** implemented and verified 2026-07-20 against initial scaffold revision `f72f5468c52a05a442fa0efbbe996fa16450a2bb`.
- **Objective:** turn the specification directory into a versionable, buildable but feature-free modular-monolith skeleton with explicit ownership and dependency rules.
- **Requirement IDs:** `FND-001`, `FND-002`, `FND-003`; preserve `FND-004`; begin toolchain pins for `FND-054`.
- **Expected files/modules:** Git metadata after owner confirmation; `.gitignore`, `.gitattributes`, `.editorconfig`, toolchain pin/update note, `go.mod`, `cmd/api/`, `cmd/worker/`, `cmd/simulator/`, domain/platform directory skeleton, `apps/web/` ownership marker, architecture dependency checker/test, root command documentation. Do not copy mutable domain models into shared code.
- **Architecture/ADR impact:** implements ADR 0001 and does not create a new service/broker/database. Record the Go module path, Go toolchain, deferred frontend build-toolchain decision, active contract location, and CI-host assumptions; an ADR is needed only if the modular-monolith decision is changed.
- **Security/abuse cases:** keep debug/test endpoints absent; make process identities/config explicit; prevent cross-module persistence imports; canonicalize PRD ownership so duplicate root files cannot drift unnoticed.
- **Migration/rollback:** no database migration. Future scaffold changes use reviewed Git reverts or forward fixes. The root duplicates were deleted only after explicit approval and a repeated eleven-pair hash comparison; recovery remains available through Git history.
- **Automated tests:** three entry points build; package tests run; dependency graph contains only allowed edges; a fixture with a forbidden import fails; repository layout, canonical manifest, and root-duplicate absence checks pass.
- **Adversarial test:** the Phase 00 “forbidden import” test deliberately attempts a cross-module persistence import and must fail (`THR-040`, `THR-060`).
- **Observability/runbook:** no runtime telemetry claim; document build/start failure diagnostics and exact prerequisite checks.
- **Evidence:** `evidence/phase-00/architecture/S01-boundary-report-v3.*` containing the verified source revision, tool versions, dependency graph, commands, expected/observed output, and digest; earlier versions remain immutable history.
- **Content opportunity:** `CNT-00-02`, only after the failing boundary test is reproducible.
- **Completion procedure:** `go test ./...`, `go build ./cmd/api ./cmd/worker ./cmd/simulator`, and the repository-owned boundary/layout check all pass from a fresh workspace; no product endpoint, schema, or financial behavior exists.

### S02 — Safe cross-cutting primitives and static bans

- **Status:** implemented in `dc638d2` and post-commit reverified 2026-07-20 at canonical-cleanup revision `240adbf`.
- **Objective:** establish the small shared primitives needed by every later module without creating a “common models” bypass.
- **Requirement IDs:** `FND-005`, `FND-006`.
- **Expected files/modules:** narrowly scoped platform packages for money, opaque IDs, UTC clock, actor/audit context, correlation/causation context, and domain errors; TypeScript exact-money fixture/helper only if the web package is ready; custom static-check rule and violating fixtures.
- **Architecture/ADR impact:** implements ADR 0006; no new ADR unless the bounded integer/ID format or clock policy materially differs.
- **Security/abuse cases:** overflow, currency confusion, unsafe JavaScript conversion, attacker-controlled correlation fields, PII in errors, predictable/invalid IDs, and time-boundary manipulation.
- **Migration/rollback:** no persistence yet. Primitive API changes remain internal but must be settled before financial schemas/contracts depend on them.
- **Automated tests:** table/property/fuzz tests for amount bounds/currency, cross-language JSON fixtures, deterministic clock tests, error sanitization, and static canaries for `float32`/`float64` money and domain `time.Now()`.
- **Adversarial test:** maximum/overflow and locale/large-number fixtures plus a clock-skew boundary case; mutation must show the money invariant tests fail when validation is removed.
- **Observability/runbook:** define safe correlation fields and stable error codes; no PII attributes. No operational runbook required beyond developer diagnostics.
- **Evidence:** primitive test report, fuzz seed/corpus summary, mutation result, and static-rule failure transcript.
- **Content opportunity:** `CNT-00-04` after the custom rules genuinely block seeded violations.
- **Completion procedure:** platform primitive tests, bounded fuzz corpus replay, mutation target, canonical Go JSON fixture replay, and static policy checks pass. A frontend fixture consumer is deferred because S02 is explicitly Go-only and no frontend build toolchain exists.

### S03 — Contract-first health, HTTP safety, and trace seed

- **Objective:** prove one safe synchronous vertical path without product semantics: contract → API/BFF server → readiness dependencies → trace/version response.
- **Requirement IDs:** advance `FND-003`, `FND-040`, `FND-043`, `FND-053`; cover the unnumbered API foundation.
- **Expected files/modules:** active OpenAPI source under the contract location selected in S01; health/version schemas/examples; HTTP server/timeouts/limits/headers/problem middleware; build-info injection; readiness checks; tracing middleware; contract/live-server tests; database-unavailable runbook.
- **Architecture/ADR impact:** remains inside `api`; `/health/live` checks process only, `/health/ready` checks safe dependency/migration state, `/version` exposes revision/contract/build time only. No broker/provider call is introduced.
- **Security/abuse cases:** health topology disclosure, trusted inbound correlation IDs, wildcard credentialed CORS, header/cache weakness, oversized/decompression/slow-body exhaustion, stack/secret disclosure.
- **Migration/rollback:** no financial migration. Readiness schema-version check must fail closed if migration state cannot be verified. Handler rollback must not leave a contract claiming unavailable behavior.
- **Automated tests:** OpenAPI validation/examples, handler conformance, secure-header/cache/CORS matrix, body/decompression/deadline tests, liveness-vs-readiness migration-lag test, safe version metadata, trace continuity.
- **Adversarial test:** named skipped tests #3 (migration lag affects readiness only) and `ADV-RES-001`; test debug/health route inventory for `THR-042`.
- **Observability/runbook:** request/error/duration metrics seed, trace ID/correlation propagation, `database unavailable` and telemetry-degraded notes; health logging must not create noise or leak internals.
- **Evidence:** live contract test report, header scan, migration-lag demonstration, and sanitized golden trace seed.
- **Content opportunity:** early material for `CNT-00-01`; do not publish “observability complete.”
- **Completion procedure:** contract lint/examples, API tests, secure-header/resource-limit suite, and a local live/ready/version smoke procedure pass with readiness both healthy and deliberately migration-behind.

### S04 — Reproducible synthetic local/reference environment

- **Status:** implementation commit `39121a31765013ebdc51b3b0ac4e47c9bc8b1516` is post-commit verified 2026-07-21; exact clean-machine wrapper proof remains an explicit S08 limitation.
- **Objective:** provision the complete synthetic platform and frontend shells with one reversible command and deterministic configuration/data.
- **Requirement IDs:** `FND-010..013`, `FND-030..033`; preserve `FND-004`; cover the unnumbered frontend foundation.
- **Expected files/modules:** local/test/staging/production-reference config schema; Podman/Docker-compatible compose definitions; PostgreSQL, Redis, selected broker, object storage, OTel collector, selected IdP, API/worker/simulator/web services; deterministic seeds/scenarios; guarded reset; React customer/merchant/workforce shells; generated/verified request client; environment banners.
- **Architecture/ADR impact:** broker/IdP and frontend shell/client decisions are prerequisites. No Kafka/Kubernetes/custom IdP. Object storage and broker remain replaceable local/reference adapters.
- **Security/abuse cases:** real service/data connection, shared credentials, development keys in production mode, cross-tenant seed leakage, unsafe reset target, mock mode mistaken for truth, stale query cache/back-forward exposure.
- **Migration/rollback:** environment resources are disposable and namespaced; reset validates resolved target. Configuration changes have safe defaults and documented rollback. Synthetic seed migrations are separate from production/reference schema migrations.
- **Automated tests:** clean provision/restart/teardown, config negative matrix, credential fingerprint separation, seed checksum repeatability, named simulator scenarios, banner coverage, accessible shell states, logout/tenant cache and browser back-forward tests.
- **Adversarial test:** skipped tests #5 and #6 plus explicit wrong-environment reset confirmation and real-service endpoint canary.
- **Observability/runbook:** service health/dependency map; local broker backlog, IdP/database/object-store unavailable procedures; visible degraded states.
- **Evidence:** clean-machine bootstrap log, configuration digest, seed checksum/scenario catalogue, browser accessibility/security report, sanitized screenshots.
- **Content opportunity:** `CNT-00-01`, `CNT-00-05`, and the environment-banner short post after proof.
- **Completion procedure:** the documented single `up` command reaches readiness; smoke tests exercise every process/shell; repeated seed run gives the expected checksums; guarded reset rejects unsafe input and then removes only the resolved local namespace.

### S05 — Database roles, migration safety, and recovery foundation

- **Status:** requirement-scoped implementation and clean post-commit verification completed 2026-07-21 at `5ea77fcf31b349b53fcd14e14ab81a4da5da840a`; `FND-064` remains partial and clean-host live proof is outstanding.
- **Objective:** make PostgreSQL schema change, privilege separation, backup, and restore testable before financial tables exist.
- **Requirement IDs:** `FND-021`, `FND-025`, `FND-060..064`.
- **Expected files/modules:** migration/admin entry point; ordered migration/role grants; migration metadata template; released-checksum policy; empty/previous-schema fixtures; representative-data/lock tests; backup/PITR configuration; isolated restore scripts/runbooks; integration test harness with distinct role credentials.
- **Architecture/ADR impact:** implements ADR 0002. Choose migration tooling and PostgreSQL isolation/backup mechanism; no ledger isolation ADR is needed until ledger posting design starts.
- **Security/abuse cases:** application DDL/table bypass, overpowered worker/reporting roles, persistent break-glass, migration lock/data reinterpretation, incomplete restore, lost key/object/event checkpoint, replayed outbox.
- **Migration/rollback:** every migration includes lock risk and forward-fix; released files are immutable. Rollback is used only when safe; otherwise rehearse forward-fix. Restore is isolated and cannot contact real providers.
- **Automated tests:** real-PostgreSQL role matrix, DDL/DML denials, empty/previous migration paths, checksum mutation, representative dataset, lock timeout/abort, backup/restore, schema/seed/object checksum, replay/idempotency checks.
- **Adversarial test:** skipped tests #8 and #9, `ADV-REL-008`, and `ADV-DR-001..005` as applicable to the foundation dataset.
- **Observability/runbook:** migration duration/lock/timeout, backup age/failure, restore progress, DB pool metrics; migration failure, database unavailable, backup corruption, and rollback-vs-forward-fix runbooks.
- **Evidence:** migration safety report, role permission matrix, immutable migration manifest, isolated restore log with measured RPO/RTO and checksums.
- **Content opportunity:** `CNT-00-03` only after restore and invariant/checksum verification are real.
- **Completion procedure:** empty/previous migration lanes, real-role permission tests, long-lock abort, backup, isolated restore, and post-restore verification all pass from documented commands.

### S06 — Observability and security operating baseline

- **Objective:** make trust boundaries, safe telemetry, secrets, alerts, and incident handling executable across the running foundation.
- **Requirement IDs:** advance `FND-040` and `FND-042`; complete the current-scope `FND-041`, `FND-043`, `FND-050`, `FND-052`, and `FND-055`; preserve `FND-051` and `FND-053`.
- **Expected files/modules:** Phase 00 DFD/STRIDE review; threat/register link validation; data-field/logging inventory; structured logger/redaction tests; OTel config/instrumentation; metric allowlists/dashboards/alerts; secret/key abstraction and rotation procedure; security/dependency/telemetry runbooks.
- **Architecture/ADR impact:** key/secret management and deployment selection requires a scoped ADR. Threat model must cover the current browser/edge/API/worker/simulator/DB/broker/object/IdP/CI boundaries without claiming certification.
- **Security/abuse cases:** log/trace/event PII leakage, CRLF/field injection, high-cardinality exhaustion, user-controlled trace confusion, secret/key exposure/downgrade/unavailability, missing alert ownership, audit/evidence gaps.
- **Migration/rollback:** telemetry degradation cannot corrupt/block financial truth unless an explicit high-risk audit policy says fail closed. Key rotation supports grace/recovery. Config rollback retains source revision and avoids silently reducing security.
- **Automated tests:** CRLF/structured injection, sink canary scan, telemetry cardinality allowlist, context propagation/retry, alert-rule tests, threat/control/test link coverage, key purpose/version/environment/rotation/unavailable cases, secure HTTP regression.
- **Adversarial test:** skipped test #2, `ADV-RES-003`, `ADV-DR-004`, and telemetry-outage exercise.
- **Observability/runbook:** this slice owns the RED/pool/queue/retry/build dashboards, alert ownership/links, secret exposure, dependency patch, broker backlog, telemetry unavailable, and failed-deploy runbooks.
- **Evidence:** reviewed DFD/threat model, sanitized golden trace, log-injection/redaction report, metric/dashboard/alert test report, key-rotation/tabletop record.
- **Content opportunity:** trust-boundary material for `CNT-00-01` and a safe golden-trace short post.
- **Completion procedure:** security/telemetry unit/integration tests, threat-link validation, alert tests, sink scans, golden trace, and runbook tabletop all pass with revision-bound evidence.

S06 is requirement-scoped implemented and pre-commit verified on base `7a08056539de6d655086f7730d0cb8df3a9bb4c6`. The static verifier, catalog mutations, exported fixed-parent API/readiness/database trace, metric checks, and collector-outage readiness exercise pass. `FND-040` remains partial for nonexistent event/job/worker propagation; `FND-042` remains partial for definition-only queue/retry metrics and absent deployed alert routing. No later Phase 00 gate is claimed.

### S07 — CI, contracts, and supply-chain integrity

- **Status:** local implementation and verification completed 2026-07-22 from `UNCOMMITTED_WORKTREE(base=3342b4ded1cd62fab1223372cd5129f272889878)`; hosted required-check, ruleset, signature, provenance, and committed-artifact proof remain outstanding.
- **Objective:** make every earlier check a required, reproducible CI/release gate and produce verifiable immutable artifacts.
- **Requirement IDs:** `FND-020..027`, `FND-054`; enforce `FND-006`, `FND-021`, `FND-025`, and earlier security/contract checks.
- **Expected files/modules:** selected CI workflows and ownership policy; lockfiles/pinned actions/base images; PR/main/nightly/release lanes; contract lint/diff/conformance; secret/SAST/dependency/license/container/IaC scanners; SBOM generation; minimal images; signing/provenance; update/emergency policy; CI runbook.
- **Architecture/ADR impact:** choose CI host and artifact-signing/provenance mechanism; deployment/secrets decision must preserve digest promotion and environment separation. Do not claim SLSA level or certification.
- **Security/abuse cases:** poisoned runner/dependency/base image, mutable tag/action, leaked CI secret/history, unsigned/wrong-source artifact, bypassed required job/review, malicious contract/migration drift.
- **Migration/rollback:** CI change is tested in a branch and fails closed on missing tools/reports. Artifact rollback promotes a previously verified digest; database compatibility and forward-fix remain separate.
- **Automated tests:** seeded failing job matrix, deleted-history secret canary in an isolated disposable Git repo, lockfile/pin checks, contract break fixtures, image non-root/read-only scan, SBOM content, provenance/signature/tamper/source checks, CODEOWNERS/branch-policy verification.
- **Adversarial test:** skipped tests #1 and #7; poisoned/mutable build input and old/new API/event compatibility (`ADV-REL-009`).
- **Observability/runbook:** CI duration/failure metadata, scanner finding retention, build/deploy revision metrics; failed deployment, dependency emergency update, signing/key unavailable, and provenance verification runbooks.
- **Evidence:** CI reports, contract compatibility/conformance report, secret history scan, SBOMs, image digests, signatures, provenance, ownership/required-check proof.
- **Content opportunity:** canary-secret failure demo and signed-build explanation; publish limitations and avoid supply-chain maturity claims beyond evidence.
- **Completion procedure:** a local CI-equivalent command and hosted required jobs pass; seeded secret/contract/migration/static violations fail; built images verify by digest/signature/provenance and SBOM scan.

S07 passes its local CI-equivalent, history/security, Git-baseline contract, and supply-chain commands. Four hashed SPDX surfaces, zero reachable Govulncheck findings, complete-history Gitleaks, deleted-history/action/contract/reference canaries, critical-CVE/license gates, and hardened backend/web image checks pass. PR run `29928153984` subsequently passed all five hosted jobs at `747cc80f058d570851f64592c0eb3a9ca0e33adc` and merged as `f6ad53553e739ea44718cc1336920a37c3fd05bc`. No ruleset, independent review, registry promotion, signature, or provenance exists. Therefore FND-022/FND-027/FND-054 are satisfied at their stated scope, while FND-020/FND-023/FND-024/FND-026 remain partial.

### S08 — Phase 00 acceptance, restore, and evidence release

- **Objective:** run the Phase 00 gate as a reviewer would and prove the platform can be cloned, started, traced, built, restored, and independently reverified.
- **Requirement IDs:** all 37, with final proof for `FND-010`, `FND-020..024`, `FND-040..043`, and `FND-064`.
- **Expected files/modules:** one phase verification procedure, evidence manifest/catalogue, known-limitations record, acceptance demo script, final dashboards/alerts/runbooks, traceability/threat/status updates, sanitized content derivatives.
- **Architecture/ADR impact:** review every Phase 00 ADR/decision for actual implementation match; unresolved gaps block completion rather than becoming silent exceptions.
- **Security/abuse cases:** evidence tampering/drift, stale source/config, secret/PII in artifacts, restore replay duplicate, missing object/key, public claim exceeding proof, demo mistaken for real service.
- **Migration/rollback:** execute clean rebuild and isolated PITR restore; document rollback-vs-forward-fix decisions and reconcile any outbox/inbox/object/config divergence. Never restore into the active local namespace.
- **Automated tests:** full PR/main/release lanes, all ten Phase 00 skipped tests, adversarial resource/recovery subset, contract examples against live server, constrained-pool race tests, accessibility/browser security, restore verification.
- **Adversarial test:** skipped tests #4 and #10 plus `ADV-REL-001..010`/`ADV-DR-001..005` that are applicable before financial features; expected absences are documented, not faked.
- **Observability/runbook:** execute alert tests and at least one deployment/migration/secret/dependency/DB/broker/telemetry recovery tabletop or game day; prove operators need no direct DB edits.
- **Evidence:** signed/digested Phase 00 evidence catalogue tied to source/config/image revisions, clean-clone transcript, trace, build provenance, restore report, limitations, and content review.
- **Content opportunity:** `CNT-00-01..05` only where each item has its stated evidence.
- **Completion procedure:** from a clean clone, one documented command verifies prerequisites and starts the stack; all gates run; a traced request is reproduced; artifacts verify; an isolated restore passes checksums/replay; the evidence catalogue and status/traceability rows match the exact source revision.

## Completed implementation slices

**S01 — Versioned repository and process-boundary scaffold** completed its requirement-scoped acceptance conditions:

1. `.` is a valid Git worktree on `main`; origin is `https://github.com/MichaelSeveen/atlas.git`, and evidence names the verified initial scaffold revision. S01 initially retained the root duplicates; the later owner-authorized cleanup removed them only after a repeated byte-for-byte comparison.
2. Go 1.25.7 and module path `github.com/MichaelSeveen/atlas` are repository-owned and tied to the configured origin; React + TypeScript is the sole frontend decision and its build toolchain is explicitly deferred.
3. Roadmap-aligned ownership directories exist without speculative domain behavior.
4. `api`, `worker`, and `simulator` entry points compile and remain inert.
5. The clean dependency scan passes and the seeded transfer-to-ledger-persistence import is rejected.
6. Layout, canonical PRD manifest, and root-duplicate absence checks pass.
7. [Current S01 evidence](../../evidence/phase-00/architecture/S01-boundary-report-v3.md) records the verified source revision, commands, versions, outcomes, hashes, threats, limitations, and revalidation procedure; both pre-commit reports remain preserved as superseded history.

**S02 — Safe cross-cutting primitives and static bans** completed its requirement-scoped acceptance conditions:

1. Six narrow Go platform packages implement money/currency, opaque identifiers, UTC clocks, actor context, correlation/causation context, and stable data-minimizing errors without product behavior.
2. Canonical decimal-string money fixtures cover zero, values beyond JavaScript's safe integer, signed bounds, currency mismatch, locale-formatted rejection, and maximum-plus-one overflow.
3. Fixed-clock boundary tests, bounded correlation fields, opaque-ID parsing/generation, and safe error rendering pass.
4. Domain static checks reject seeded floating-money and direct wall-clock violations while accepting explicit safe controls.
5. Three fuzz targets complete their seed corpora and 100 executions each; disabling the money currency guard is killed by `TestCheckedArithmetic`.
6. [S02 evidence](../../evidence/phase-00/primitives/S02-primitives-report.md) preserves its original pre-commit limitation; [canonicalization evidence](../../evidence/phase-00/architecture/PRD-canonicalization-report.md) records the committed S02/cleanup revisions and full post-commit revalidation.

**S03 — Contract-first health, HTTP safety, and trace seed** completed its requirement-scoped acceptance conditions and is post-commit verified:

1. The canonical OpenAPI defines only the three newly implemented operational endpoints and their closed schemas, safe problems, request/correlation/trace inputs, and response headers; invalid opaque-ID examples were corrected to the existing normative alphabet.
2. `cmd/api` has a bounded graceful HTTP lifecycle. Liveness is process-only, readiness requires both dependency and migration state, version exposes only validated revision/contract/build time, and the executable defaults to not-ready while real probes are absent.
3. Secure headers, `no-store`, exact-origin CORS, body/query/decompression/header/time limits, closed route inventory, panic recovery, and stable non-disclosing problems pass handler and slow-client tests.
4. Validated request/correlation/W3C trace context reaches a readiness child span; trace/metric fields are closed and bounded, invalid metadata is replaced, and telemetry-recorder degradation does not affect the operation.
5. Named skipped test #3 proves migration lag changes readiness to a generic `503` while liveness and version remain available. `ADV-RES-001` resource facets and debug-route inventory are exercised.
6. The metadata fuzzer completes its seed corpus plus 100 executions, and deleting the copied `/health/ready` contract path is killed by the focused contract suite.
7. [S03 post-commit evidence](../../evidence/phase-00/http/S03-post-commit-verification.md) binds the result to implementation commit `b5fd25b`; the original [S03 report](../../evidence/phase-00/http/S03-http-foundation-report.md) remains preserved with its pre-commit limitation.

**S04 — Reproducible synthetic local/reference environment** completed its requirement-scoped acceptance conditions and is post-commit verified:

1. Closed local/test/staging/production-reference configuration, generated purpose-scoped credentials, contained reset, and exact synthetic dependency pins pass validation and seeded negative tests.
2. PostgreSQL, Redis, NATS JetStream, MinIO, OTel Collector, Keycloak, API, worker, simulator, and React web run as separate constrained processes; no product service or real provider is introduced.
3. Customer, merchant, and workforce shells use function components only, render persistent synthetic banners, keep browser storage empty, and clear/deny protected state across logout and back navigation.
4. [S04 post-commit evidence](../../evidence/phase-00/environment/S04-post-commit-verification.md) binds the committed implementation; the host Compose transport limitation remains explicit.

**S05 — Database roles, migration safety, and recovery foundation** completed its requirement-scoped implementation with post-commit verification:

1. [ADR 0010](../atlas-prd/06-governance/adrs/0010-native-postgresql-migrations-and-recovery.md) selects a repository-owned checksummed SQL runner, pgx readiness probe, distinct roles, and PostgreSQL-native physical recovery without a product schema.
2. The released manifest validates two ordered migration/metadata pairs; changed-SQL and deleted-metadata canaries are killed.
3. Current, empty, and previous-version migration lanes pass against real PostgreSQL. The real permission matrix proves allowed DML/read paths and denied DDL/grant/escalation paths, including an expired-by-default bounded break-glass drill.
4. Named skipped test #9 holds a three-second exclusive lock; the migration aborts in one second, rolls back, and leaves the database usable.
5. A verified six-second physical base backup plus archived WAL restores into an internal-only service at the pre-deletion target. The restored migration checksum and synthetic marker pass.
6. The original [S05 live evidence](../../evidence/phase-00/database/S05-database-report.md) preserves its pre-commit environment identity and limitations; [post-commit evidence](../../evidence/phase-00/database/S05-post-commit-verification.md) binds the clean static/build/test result to `5ea77fc`. Local volumes are unencrypted, so `FND-064` and the clean-host recovery gate remain outstanding.

**S06 — Observability and security operating baseline** completed its current requirement-scoped implementation with pre-commit evidence:

1. API/worker/simulator emit only a closed source-redacted JSON schema. Bootstrap, SDK, and HTTP-server diagnostics cannot copy raw error/connection payloads; CRLF/field injection writes no forged sink entry.
2. Official pinned OpenTelemetry trace/metric SDKs export bounded process build/lifecycle, API RED, readiness, and database pool/readiness telemetry. Collector failure is non-authoritative and bounded.
3. A fixed inbound W3C parent is observed in linked API, readiness, and database spans by the local collector; stopping the collector leaves readiness `200`.
4. The metric/dashboard/alert catalog enforces label budgets, owners, severity, rationale, runbook, and tests. Queue lag/retry remain definition-only because no queue/job exists.
5. The current DFD, six trust boundaries, STRIDE review, field inventory, and all 60 canonical threat links pass executable validation. The provider-neutral secret boundary passes rotation, overlap, revocation, outage/recovery, downgrade, purpose, algorithm, and environment tests.
6. Vulnerability disclosure, dependency emergency, secret exposure, telemetry unavailable, and failed-deployment runbooks exist; the [S06 report](../../evidence/phase-00/observability-security/S06-observability-security-report.md) records the synthetic dependency tabletop and explicit pre-commit/current-flow limits.

**S07 — CI, contracts, and supply-chain integrity** completed its local requirement-scoped implementation with pre-commit evidence:

1. [ADR 0011](../atlas-prd/06-governance/adrs/0011-github-actions-keyless-release-integrity.md) selects GitHub Actions/GHCR plus digest-only keyless Cosign and GitHub attestations; no long-lived signing key or unsigned fallback is permitted.
2. PR, CodeQL, real PostgreSQL/NATS, supply-chain, nightly, and release workflows use immutable action SHAs and repository-owned commands. `CODEOWNERS`, Dependabot, action/image/tool locks, and static drift tests exist.
3. `contractctl` lints the canonical OpenAPI/AsyncAPI, resolves internal references, compares against the Git base, rejects removed paths/fields/messages, and runs the real feature-free API examples without copying the contracts.
4. Complete-history Gitleaks and a disposable deleted-history secret canary pass. Govulncheck reports zero reachable vulnerabilities; Windows Gosec/race are explicitly unavailable and delegated to required Linux/CodeQL lanes that are not yet evidenced.
5. Backend/frontend source and backend/web image SPDX SBOMs are generated, hashed, identity/license/CVE checked, and kept disposable. Backend and web images use digest-pinned bases and pass non-root, read-only, capability-drop, and no-new-privileges checks.
6. The [S07 report](../../evidence/phase-00/supply-chain/S07-ci-contract-supply-chain-report.md) binds the local result to `UNCOMMITTED_WORKTREE(base=3342b4ded1cd62fab1223372cd5129f272889878)` and keeps hosted enforcement, release identity, and final committed revision as explicit gaps.

The later [hosted PR verification](../../evidence/phase-00/supply-chain/S07-hosted-pr-verification.md) records the five successful jobs, exact PR head, and merge commit. It closes the missing hosted-execution fact but not required rules, independent review, registry promotion, signing, or provenance.

**S08 — Phase 00 acceptance, restore, and evidence release** completed its local requirement-scoped implementation with pre-commit evidence:

1. One versioned verifier composes S07 static/history/supply-chain gates with the live synthetic stack, golden trace, collector outage, real PostgreSQL/NATS role/migration/lock/recovery lanes, and guaranteed teardown.
2. A closed S01–S08 SHA-256 catalogue rejects a changed artifact and a stale source identity in disposable seeded canaries.
3. Named skipped test #10 runs 24 concurrent readiness requests against real migrated PostgreSQL through a one-connection pool and releases the pool; local concurrency passes, while `-race` awaits the updated hosted Linux lane.
4. Named skipped test #4 and product-specific `ADV-REL`/`ADV-DR` cases are explicitly not applicable because no outbox, worker claim, provider, report/object, inbox, key, or financial state exists; no fake semantics were added.
5. The local live gate passed synthetic smoke, the API/readiness/database trace, telemetry outage readiness, long-lock abort, real NATS, 27-second backup, and 62-second isolated restore. Teardown completed, but Podman force-stopped the stateless Bun web container after ten seconds, so graceful web termination is not claimed.
6. The [S08 pre-commit report](../../evidence/phase-00/acceptance/S08-phase-00-acceptance-report.md) preserves the detailed live/history/supply results. [Post-commit verification](../../evidence/phase-00/acceptance/S08-post-commit-verification.md) binds final implementation `6b09b4a` and evidence revision `431821f`, including a passing detached same-host clean clone. Independent clean-host, hosted S08 race, ruleset/review, registry/signature/provenance, encryption, alert-routing, and product-replay gaps remain explicit.

## Specification findings requiring attention

- No authoritative documents conflict on Phase 00 behavior; accepted ADRs align with the master/architecture/phase sources.
- Local/reference broker, IdP, object store, React/Bun, contract ownership, and PostgreSQL migration/recovery decisions are accepted and deliberately do not select their production deployment models.
- GitHub Actions/GHCR and keyless release identity are accepted by ADR 0011. Hosted S07 jobs now exist and passed; ruleset, independent-review, registry, signature, and provenance evidence still does not exist.
- Production secret manager/custody, backup encryption/retention, production identity/broker/object choices, alert backend/routing, and generated product-client strategy remain unresolved or deliberately deferred before any reference release/product consumer.
- S08 traceability is requirement-specific. Remaining partial rows must be closed only when their post-commit, hosted-enforcement, independent-review, clean-host, release, encryption, routing, or later product-flow evidence exists.
