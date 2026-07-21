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

S01 through S03 are implemented as a feature-free engineering foundation. Git is valid on branch `main`; the configured GitHub origin supplies the Go module identity. The repository has a pinned Go policy, React + TypeScript as the sole frontend decision with its build toolchain deferred, three independent process entry points, roadmap ownership directories, typed cross-cutting Go primitives, static import/money/clock checks, and a hardened API edge exposing only liveness, readiness, and version. Seeded negative tests, bounded fuzz campaigns, the killed money mutation, and the killed S03 contract-path canary are revision-aware evidence. It still has no product/frontend behavior, database/migration implementation, CI, container/local environment, broker, IdP, or runtime telemetry exporter. The PRD pack remains structurally valid after traceability updates, and the architecture test rejects reintroduced root duplicates.

| Status | Count |
|---|---:|
| Satisfied | 8 |
| Partially satisfied | 6 |
| Absent | 23 |
| Conflicting | 0 |
| Not yet assessed | 0 |

## Requirement-by-requirement audit

Slice references (`S01`–`S08`) identify the primary sequencing below. Threat IDs refer to the canonical [threat register](../atlas-prd/06-governance/THREAT_REGISTER.csv). Tests include the minimum proof implied by Phase 00 and the Definition of Done; evidence must also carry source/config revision, seed, reproduce command, expected/observed result, sanitization, digest, limitations, and revalidation date.

| Requirement | Status | Existing repository evidence | Missing implementation and completion proof | Applicable threats | Primary sequence |
|---|---|---|---|---|---|
| `FND-001` | Satisfied | Roadmap-aligned ownership tree, [root commands](../../README.md), pinned Go metadata, layout/canonical-source tests, and [S01 evidence](../../evidence/phase-00/architecture/S01-boundary-report-v3.md) | Reverify from a clean clone; later slices must replace placeholders with owned artifacts without creating duplicate contract/spec truth. | `THR-013`, `THR-042`, `THR-060` | S01 complete; preserve |
| `FND-002` | Satisfied | Rules in [module boundaries](MODULE_BOUNDARIES.md) are enforced by a clean import scan and seeded cross-context persistence-import rejection in `internal/architecture` | Database write ownership is intentionally not claimed and remains S05/`FND-060`; extend the checker as real context APIs appear. | `THR-040`, `THR-054`, `THR-060` | S01 complete; DB proof S05 |
| `FND-003` | Satisfied | Separate `cmd/api`, `cmd/worker`, and `cmd/simulator` entry points pass package tests and build independently. The API now has a bounded HTTP lifecycle and only the three S03 operational routes; worker and simulator remain inert. | S04 must provide environment composition and real readiness probes. No product endpoint, identity, job, provider scenario, or external dependency is claimed. | `THR-025`, `THR-042`, `THR-060` | S01/S03 complete; extend S04 |
| `FND-004` | Satisfied | React + TypeScript is selected in the [PRD README](../atlas-prd/README.md), [product charter](../atlas-prd/00-master/00_PRODUCT_CHARTER.md), and [architecture §3](../atlas-prd/01-architecture/00_SYSTEM_ARCHITECTURE.md); no Vue implementation exists | Preserve with one frontend package/lockfile and CI rule rejecting a second framework. The later shell must prove route/identity separation and accessible failure states. | `THR-029`, `THR-060` | Preserve in S01/S04 |
| `FND-005` | Satisfied | Narrow [platform primitives](PLATFORM_PRIMITIVES.md) implement bounded integer money/currency, cryptographically random opaque IDs, injectable UTC clocks, explicit actor and correlation/causation contexts, and data-minimizing stable domain errors. Table/property tests, canonical JSON fixtures, three 100-execution fuzz campaigns, clock-boundary cases, and a killed currency-guard mutation are recorded in [S02 evidence](../../evidence/phase-00/primitives/S02-primitives-report.md). | Add a frontend consumer for the shared decimal-string fixture only when the frontend toolchain is authorized; keep product authorization, tenancy, persistence, and telemetry semantics out of these primitives. | `THR-001`, `THR-002`, `THR-009`, `THR-030`, `THR-031` | S02 complete; preserve |
| `FND-006` | Satisfied | The architecture checker rejects named `float32`/`float64` money and direct domain `time.Now()` (including aliases/dot imports), while safe-control fixtures permit non-money measurements and the platform clock adapter. Seeded canaries are captured in [S02 evidence](../../evidence/phase-00/primitives/S02-primitives-report.md). | Wire the existing command into protected CI in S07/`FND-020`; reviewers must still reject deliberately obscure aliases/names because this is a conservative syntax check. | `THR-001`, `THR-029`, `THR-031` | S02 complete; CI enforcement S07 |
| `FND-010` | Partial | S04 defines and live-verifies the complete constrained Compose topology, readiness, process restart, smoke, teardown, and a documented reversible command. | The current Windows Podman machine required a host-only WSL/systemd and Compose-provider workaround, so the exact repository `Up` wrapper has not yet been proven from a clean machine with no host repair. Revalidate in S08. | `THR-019`, `THR-025`, `THR-042`, `THR-045` | S04 implementation; clean-machine proof S08 |
| `FND-011` | Partial | The closed seed manifest validates two tenants, three users, two account identity fixtures without financial state, eight named provider scenario IDs, fixed virtual time, tenant ownership, and repeatable SHA-256. | Application schemas and executable simulator/provider contracts do not exist; S04 deliberately does not load database rows or claim scenario behavior. | `THR-043`, `THR-045`, `THR-060` | S04 identity/catalogue; executable contracts later |
| `FND-012` | Satisfied | Typed configuration and canaries require synthetic labels/services, fixed local Compose endpoints or reserved `atlas.invalid` reference hosts, loopback local surfaces, environment-scoped secret references, and no real service credentials. | Revalidate whenever a real deployment/provider adapter is introduced; this is portfolio-environment proof only. | `THR-020`, `THR-045` | S04 complete; preserve |
| `FND-013` | Satisfied | Guarded reset accepts only local/test, prints the resolved target, requires the exact environment phrase, validates containment, and rejects wrong-environment and production-reference canaries before deletion. | Compose-volume deletion remains local namespace-scoped; production recovery is not represented by this command. | `THR-019`, `THR-045` | S04 complete; preserve |
| `FND-020` | Absent | Required lanes are specified in [test strategy §CI test lanes](../atlas-prd/04-testing/TEST_STRATEGY.md); no CI exists | Blocking PR lane must execute backend/frontend/type/lint/contract/migration/security/secret/race checks. Prove required jobs fail closed using seeded canaries and retain reports. | `THR-009`, `THR-013`, `THR-014`, `THR-030`, `THR-042`, `THR-060` | S07 after S01–S06 |
| `FND-021` | Absent | Real-container rule exists only in Phase 00/test strategy | Run integration tests against real PostgreSQL and selected broker containers with realistic roles and constrained pools. Prove mocks cannot satisfy the integration lane; include rollback/cleanup and duplicate/replay cases. | `THR-014`, `THR-015`, `THR-019`, `THR-040`, `THR-054` | S05, enforced S07 |
| `FND-022` | Absent | Supply-chain requirements only; no build artifacts | Generate frontend/backend/image SBOMs, verify expected packages/images and digests, scan them, and attach revision-bound artifacts. | `THR-013`, `THR-060` | S07 |
| `FND-023` | Absent | No images/build metadata | Build immutable minimal images labeled/tagged by source revision and record digest; prove same digest is promoted, mutable tag is not evidence, runtime is non-root/read-only where viable. | `THR-013`, `THR-060` | S07 |
| `FND-024` | Absent | No signing/provenance configuration | Sign artifacts and generate build provenance using environment-separated keys/identity. Verify signature, subject digest, source revision, tamper rejection, and key-unavailable failure. | `THR-013`, `THR-018`, `THR-020`, `THR-060` | S07 |
| `FND-025` | Absent | Migration checks are specified only | Test up/verification on empty and previous-release schema snapshots, including repeatability and incompatible-contract detection. Include `ADV-REL-008` long-lock timeout/abort and safe recovery. | `THR-014`, `THR-019`, `THR-060` | S05, enforced S07 |
| `FND-026` | Absent | Code-owner areas are listed in security/quality gates; no VCS/owner policy exists | Add owner rules for ledger, authorization, migrations, crypto, CI, and deploy paths; prove matching and protected-review enforcement. Evidence must distinguish local policy file from host branch-protection proof. | `THR-007`, `THR-013`, `THR-014`, `THR-018`, `THR-020`, `THR-040` | S07; requires Git/host decision |
| `FND-027` | Absent | Reference [OpenAPI](../atlas-prd/03-contracts/openapi.yaml) and [AsyncAPI](../atlas-prd/03-contracts/asyncapi.yaml) exist; no baseline comparison runs | Add syntax/reference lint, breaking-change comparison, example execution, and implementation conformance. Seed a removed field/path/message and prove CI rejection. | `THR-042`, `THR-060` | S03 contract source, S07 enforcement |
| `FND-030` | Satisfied | Four strict typed JSON configurations define local, test, staging, and production-reference with closed fields, exact origins, fixed service portfolios, surfaces, flags, banners, and credential references. | Deployment-specific secret backends and drift automation remain S07. | `THR-020`, `THR-042`, `THR-045`, `THR-060` | S04 complete; enforce S07 |
| `FND-031` | Partial | All four configurations have non-overlapping environment/purpose credential references; generated local/test password/token fingerprints are unique and never recorded as evidence values. | Staging/production-reference provisioning, signing/encryption material, rotation grace, restore, and secret-manager fingerprints remain S07/S08. | `THR-020`, `THR-023`, `THR-045` | S04 reference separation; finalize S07/S08 |
| `FND-032` | Satisfied | React customer, merchant, and workforce shells render the persistent runtime synthetic banner; live/browser proof covers all routes, no-store responses, empty browser storage, logout cache clear, and back-navigation denial. | Responsive and broader accessibility matrices expand in S07; no product UI is claimed. | `THR-045`, `THR-046`, `THR-047` | S04 complete; preserve |
| `FND-033` | Satisfied | Typed flag metadata requires owner, future expiry, default, risk, and rollback; validation rejects stale/incomplete/high-risk-default-on state, while immutable concurrent evaluation fails closed on unknown keys and defaults on source outage. | Financial-state flags remain prohibited until a later reviewed design; S04 has only a low-risk shell flag. | `THR-024`, `THR-036`, `THR-042`, `THR-059` | S04 complete; preserve |
| `FND-040` | Partial | The S03 API edge generates or validates opaque request/correlation IDs, validates/replaces W3C trace context, returns safe response context, and links the readiness span. Duplicate/malformed metadata fuzz and rejection tests pass. | Propagation through database spans, outbox/events, worker, simulator, retries, and causation cannot be proven until those components exist; context remains explicitly non-authoritative. | `THR-009`, `THR-015`, `THR-030`, `THR-060` | S03 API facet complete; finish S06/S08 |
| `FND-041` | Absent | Logging rules are defined, but no logger/sink/test exists | Implement structured injection-safe source redaction. Prove CRLF/newline and structured-field input cannot forge entries and canary PII/secrets never reach log/trace/event sinks. | `THR-009`, `THR-030` | S06 |
| `FND-042` | Absent | Metric names are specified in reliability §6; no instrumentation/dashboard/alert exists | Instrument RED, DB pool, queue lag, retries, and build/deploy metadata with bounded labels. Test metric semantics/cardinality, alert thresholds/routing, and runbook links. | `THR-010`, `THR-025`, `THR-030`, `THR-048`, `THR-049` | S06 |
| `FND-043` | Partial | The S03 golden test records a deterministic `GET /health/ready` server span and linked readiness child with closed safe fields, bounded metric labels, and continuity from valid inbound trace context. Telemetry-recorder failure cannot change the HTTP result. | No runtime exporter/collector or web, database, outbox, worker, and simulator spans exist; the required full synthetic path remains S06/S08. | `THR-009`, `THR-030`, `THR-042`, `THR-060` | S03 seed complete; finish S06/S08 |
| `FND-050` | Partial | System-context and trust-boundary diagrams exist in architecture/security docs; the 60-row threat register exists | Add a Phase 00 DFD with per-boundary controls and an explicit initial STRIDE review linking assets, trust boundaries, `THR-001..060`, controls, tests, owners, and residual risk. Run link/coverage validation and human review. | All register threats; immediate focus `THR-005`, `THR-009`, `THR-013`, `THR-014`, `THR-019`, `THR-020`, `THR-030`, `THR-040`, `THR-042`, `THR-045`, `THR-054`, `THR-060` | S06, revisited each slice |
| `FND-051` | Satisfied | Classification is defined in [security model §9](../atlas-prd/01-architecture/01_SECURITY_AND_TRUST_MODEL.md); log field/redaction/retention rules in [reliability §6](../atlas-prd/01-architecture/04_RELIABILITY_OBSERVABILITY_DR.md) | Preserve via field inventory review and enforce under `FND-041`; every new field/sink gets purpose, retention, masking, logging/event decision, and tests. | `THR-009`, `THR-017`, `THR-026..030`, `THR-045`, `THR-050`, `THR-057` | Preserve in all; enforcement S06 |
| `FND-052` | Partial | Key versioning/rotation/separation rules exist in [security model §10](../atlas-prd/01-architecture/01_SECURITY_AND_TRUST_MODEL.md) | Implement a secret/key abstraction plus operational rotation/revocation/recovery procedure. Test old/in-flight verification, wrong purpose/environment, unavailable key, restored key version, and downgrade rejection. | `THR-020`, `THR-031`, `THR-056` | S06; provider/deploy decision first |
| `FND-053` | Satisfied | The API edge sets secure/cache headers, enforces exact-origin CORS, rejects wildcard origins, bounds headers/body/read/write/idle/readiness work, refuses compression/query/body on foundation routes, recovers safely, and emits catalogued topology-free problems. The slow-header, streamed overflow, `ADV-RES-001` facets, panic-detail, and route-inventory tests pass. | Re-run in CI and against the later reference deployment/proxy in S07/S08; future product handlers need schema-specific controls and may not weaken this edge policy. | `THR-010`, `THR-041`, `THR-042`, `THR-043` | S03 complete; preserve |
| `FND-054` | Partial | Exact Go pin, verification, and [update policy](TOOLCHAIN_POLICY.md) exist; React + TypeScript is the sole frontend framework | The frontend build toolchain is deferred and no third-party application dependency is installed; dependency/checksum/license/scanner, base-image, CI-action, reproducible install, and emergency-update proof remain S07. | `THR-013`, `THR-060` | S01 partial; complete S07 |
| `FND-055` | Absent | Required incident topics exist in security model; no runbooks | Write vulnerability disclosure and dependency emergency-update runbooks with intake, severity, containment, evidence, patch/rebuild/revoke, communication, and retrospective tests. Run a tabletop against a synthetic vulnerable dependency. | `THR-013`, `THR-020`, `THR-060` | S06/S07 |
| `FND-060` | Partial | Exact role separation is declared in [architecture §7](../atlas-prd/01-architecture/00_SYSTEM_ARCHITECTURE.md) and [ADR 0002](../atlas-prd/06-governance/adrs/0002-postgresql-financial-ledger.md) | Create roles/grants/default privileges and distinct credentials. Integration matrix must prove permitted and denied DDL/DML/table paths for migration, API, worker, reporting-read, and time-bounded break-glass. | `THR-007`, `THR-014`, `THR-018`, `THR-040`, `THR-054` | S05 |
| `FND-061` | Absent | Policy only; no roles/schema | Attempt `CREATE/ALTER/DROP` and ownership/grant escalation as application roles against real PostgreSQL; all must fail and produce safe detection without revealing credentials. | `THR-014`, `THR-040` | S05 |
| `FND-062` | Absent | No migration files or released baseline | Establish append-only released migration checksums and forward-only correction policy. Test changed/deleted/reordered released migration detection against prior source/database history. | `THR-014`, `THR-019`, `THR-060` | S05; requires Git baseline |
| `FND-063` | Absent | Lock/forward-fix requirement only | Require per-migration risk metadata, representative dataset test, statement/lock timeout, cancel/abort recovery, query-plan/space review, and forward-fix rehearsal. Include `ADV-REL-008`. | `THR-014`, `THR-019`, `THR-051` | S05 |
| `FND-064` | Absent | Backup/PITR design and targets exist in [reliability §8](../atlas-prd/01-architecture/04_RELIABILITY_OBSERVABILITY_DR.md); no environment/evidence exists | Configure encrypted backup/PITR and isolated restore. Execute schema, seed/object checksum, key access, outbox/inbox/idempotency replay, and synthetic-flow verification; record measured RPO/RTO and limitations. Cover `ADV-DR-001..005`. | `THR-019`, `THR-020`, `THR-027`, `THR-054` | S05 implementation; S08 gate |

## Unnumbered Phase 00 deliverables

These are release-blocking even though the phase file does not assign them `FND-*` IDs.

| Deliverable | Current state | Planned slice |
|---|---|---|
| `GET /health/live`, `GET /health/ready`, `GET /version` | Implemented contract-first. Liveness is process-only; readiness fails closed on dependency/migration state without topology; version is limited to revision/contract/build time. The real executable deliberately defaults to not-ready until real probes exist. | S03 complete; real probes S04/S05 |
| Common OpenAPI components for problem details, request/correlation IDs, idempotency, cursor pagination, money, actor/session, ETags/versions | S03 added request/correlation/trace parameters and operational response headers while preserving the existing money/idempotency/cursor/ETag/Problem components. Actor/session coverage and full semantic contract tooling remain later work. | S03 partial; finish S07 |
| Customer/merchant/workforce React shells, accessible primitives, generated/verified client, safe cache clearing | No frontend exists. | S04, contract enforcement S07 |
| Ten named “tests most agents skip” | None implemented. Each is assigned across S02–S08; S08 must report all ten. | S02–S08 |
| Eight runbook topics | S03 adds a database-unavailable/readiness runbook without claiming a database exists. Migration, backup, secret, dependency, broker, telemetry, and deployment procedures remain later work. | S03/S05/S06/S07 |
| Acceptance gate and content artifacts | No clean-clone/start/trace/sign/restore evidence; planning-only content exists. | S08 after evidence |

## Ordered execution plan

S01 is complete with evidence bound to initial scaffold revision `f72f5468c52a05a442fa0efbbe996fa16450a2bb`. S02 implementation is committed as `dc638d2`; the owner-authorized canonical-PRD cleanup is committed and post-commit verified as `240adbf`. S03 implementation is committed and post-commit verified as `b5fd25bac7844cfe929e28869d7c12f26e91b200`; no push is claimed. S04 is now authorized and active.

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

- **Status:** implemented and pre-commit verified 2026-07-21 against `UNCOMMITTED_WORKTREE(base=c327135)`; exact clean-machine wrapper proof remains an explicit S08 limitation.
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
- **Requirement IDs:** complete `FND-040..043`, `FND-050..053`, `FND-055`; preserve `FND-051`.
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

### S07 — CI, contracts, and supply-chain integrity

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

## Specification findings requiring attention

- No authoritative documents conflict on Phase 00 behavior; accepted ADRs align with the master/architecture/phase sources.
- The accepted-ADR index contains a path typo (`06-governance/adr/` versus actual `06-governance/adrs/`).
- The Phase 00 health/version endpoints are required but absent from the reference OpenAPI; this is required contract work, not permission to implement undocumented handlers.
- Broker, IdP, migration tool, CI host, deployment/secret manager, frontend shell/client strategy, and active implementation-contract location are undecided. Resolve each before its affected slice; only material choices require ADRs.
- The Phase 00 traceability rows currently map every `FND-*` requirement to the same broad threat/test/evidence set. Implementation should refine those rows using the requirement-specific mappings above rather than claiming one generic CI report proves all 37 requirements.
