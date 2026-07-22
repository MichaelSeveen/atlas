# Atlas agent operating guide

## Project purpose

Atlas is a portfolio-grade, security-first multi-currency wallet and financial-operations platform. Its value is demonstrated by financial correctness, recoverable failure handling, strict authorization and tenancy, deterministic provider simulation, reconciliation, operational tooling, and reproducible evidence—not by feature count or visual polish.

This repository is currently in **Phase 00 — Secure engineering foundation**. Do not implement wallet or money-movement features before their prerequisite phases pass.

## Stack and commands discovered

- Backend: Go only. The module is `github.com/MichaelSeveen/atlas`, derived from the configured GitHub origin, with language baseline Go 1.25.0 and security-patched toolchain Go 1.25.12. Separate feature-free `api`, `worker`, and `simulator` entry points exist. The API exposes only the three S03 operational routes; worker and simulator remain inert.
- Frontend: React 19.2.7 + TypeScript only, built/tested/served by Bun 1.3.0 with `bun.lock` as the sole lockfile. Do not add Node.js, pnpm, npm, Yarn, Vue, or a competing frontend toolchain. The three S04 route shells are synthetic foundation UI only.
- React source uses function components and hooks only. Class declarations are forbidden under `apps/web/src` and enforced by `TestFrontendUsesFunctionComponentsOnly`; do not introduce React class components for error handling or any other reason.
- Authoritative store: PostgreSQL. Redis is ephemeral only. Async fan-out uses a transactional outbox and an at-least-once broker only when required.
- Contracts: source-controlled OpenAPI 3.1.1 and AsyncAPI 3.0.0.
- Telemetry: OpenTelemetry-compatible traces, metrics, and structured logs.
- External dependencies: deterministic synthetic provider, KYC, and settlement simulators only.
- S01 verification: `pwsh -NoProfile -File ./scripts/verify-s01.ps1`.
- S02 verification: `pwsh -NoProfile -File ./scripts/verify-s02.ps1`.
- S03 verification: `pwsh -NoProfile -File ./scripts/verify-s03.ps1`.
- S04 verification: `pwsh -NoProfile -File ./scripts/verify-s04.ps1`; add `-Live` with the synthetic stack running.
- S05 verification: `pwsh -NoProfile -File ./scripts/verify-s05.ps1`; add `-Live` to run the real PostgreSQL/NATS role, migration, lock, backup, WAL, and isolated-restore checks.
- S06 verification: `pwsh -NoProfile -File ./scripts/verify-s06.ps1`; add `-Live` for the exported golden trace, metric assertions, and collector-outage readiness exercise.
- S07 verification: `pwsh -NoProfile -File ./scripts/verify-s07.ps1`; add `-History` for full-history secrets and Govulncheck, and `-SupplyChain -ContainerRuntime podman` for four SBOMs plus image CVE/license/hardening checks.
- Backend checks: `go test ./...` and `go build ./cmd/api ./cmd/worker ./cmd/simulator`.
- Boundary/policy checks: `go test ./internal/architecture -count=1`; seeded negatives cover forbidden imports, floating-point money, and direct domain `time.Now()`.
- S04 composes local PostgreSQL, Redis, NATS, MinIO, the OTel Collector, Keycloak, API, worker, simulator, and web. S05 adds repository-owned PostgreSQL migration/role/recovery controls. S06 exports bounded traces/metrics from the Go processes, emits source-redacted JSON logs, and keeps the collector outside authoritative readiness. S07 adds only CI, contract, SBOM, scanner, ownership, and release-integrity controls. These remain synthetic local/reference controls only. No product schema, broker stream, identity exchange, worker job, managed secret provider, or financial behavior exists yet.
- Git is initialized on `main` with origin `https://github.com/MichaelSeveen/atlas.git`. S06 is committed as `3342b4d`; S07 is locally verified from an uncommitted worktree based on that revision, while hosted ruleset/signature/provenance proof remains outstanding.

## Source-of-truth hierarchy

1. Product purpose, scope, and non-goals: `docs/atlas-prd/00-master/`.
2. Project-wide invariants and release gates: `00-master/03_REQUIREMENTS_AND_QUALITY_GATES.md`.
3. Architecture boundaries: `01-architecture/` and accepted ADRs in `06-governance/adrs/`.
4. Current scope and acceptance: `02-phases/PHASE-00_ENGINEERING_FOUNDATION.md`.
5. HTTP behavior: `03-contracts/openapi.yaml`.
6. Event behavior: `03-contracts/asyncapi.yaml` and `03-contracts/EVENT_CATALOG.md`.
7. Security: the trust model, threat register, and security verification plan.
8. Completion: `06-governance/DEFINITION_OF_DONE.md`.
9. Existing code is evidence of current state, not authority over the specification.

Do not silently resolve conflicts. Cite the exact files/sections, check accepted ADRs, then record unresolved conflicts as blocking decisions.

## Non-negotiable invariants

- No balance change without a balanced journal or explicitly modeled reservation.
- Money is integer minor units plus explicit currency; JSON amounts are decimal strings. Never use floating point for money.
- Posted journals/postings are immutable; corrections are linked compensating entries.
- Money-moving commands are idempotent, concurrency-safe, auditable, and recoverable after ambiguous failure.
- Provider timeout means unknown outcome until evidence establishes final state.
- Domain change and outbox record commit atomically; delivery is at least once; consumers handle duplicates, replay, and ordering failure.
- PostgreSQL owns financial, idempotency, authorization, inbox/outbox, and durable workflow truth. Redis never does.
- Browser permission checks are presentation only; server-side tenant, object, action, and field authorization is mandatory.
- Privileged actions require separate workforce identity, purpose, authorization, audit, step-up, and maker-checker where specified.
- Sensitive data is minimized, classified, masked/encrypted, and excluded from unsafe logs, events, errors, fixtures, and browser storage.
- The portfolio environment never uses real money, cardholder data, identity documents, or unsupported compliance/security/scale claims.

## Module and dependency rules

- Preserve a boundary-enforced modular monolith per ADR 0001; extraction requires measured need and an ADR.
- Domains own their packages, migrations/tables, application interfaces, errors, and events.
- Cross-module writes use explicit application services. Never write another module's authoritative tables directly.
- Ledger posting is available only through narrow typed posting templates and controlled database roles.
- Do not import another module's internal persistence package or create a shared mutable `models` package.
- Shared platform code is limited to identifiers, money, clock, transaction wrapper, actor/audit/correlation context, errors, tracing, and test fixtures.
- Pass tenant and authorization context explicitly; do not infer them from global state.
- External calls never occur inside financial database transactions.
- API contracts are not database models. Contract changes update OpenAPI/AsyncAPI first and receive compatibility checks.

## Context-loading rules

- Start with `docs/engineering/IMPLEMENTATION_STATUS.md`, this file, the current phase, and `docs/engineering/CONTEXT_INDEX.md`.
- Before implementation, name the phase/slice, requirement IDs, threat IDs, affected contexts, authorization and financial boundaries, idempotency/concurrency risks, before/after-commit failures, contract changes, and evidence.
- Load only relevant sections of large contracts, CSV registers, test catalogues, and later phases. Search by stable ID, endpoint, event, module, phase, or control family.
- Do not copy complete PRD files, contracts, registers, or catalogues into context documents.
- Use `docs/atlas-prd/` as canonical. Root-level PRD or contract copies are forbidden; link to or edit the canonical artifact directly.

## Testing and evidence rules

- A file or dependency is not proof that a requirement works. Require a reproducible test/review command and revision-bound evidence.
- Test invariants and failure modes with the appropriate mix of table/property/model/metamorphic/mutation/fuzz tests, PostgreSQL integration tests using realistic roles, concurrency/failpoint tests, contract tests, authorization matrices, and browser tests.
- Mocks are not the sole proof of transactional correctness. Critical integration tests use real PostgreSQL and broker containers.
- Every phase implements and evidences at least one named “test most agents skip.”
- Phase completion requires its acceptance journey, adversarial tests, threat/traceability updates, telemetry, alerts, runbooks, sanitized evidence, and honest known limitations.
- Evidence includes requirement/threat/test IDs, source and environment revision, seed, reproduce command, expected/observed result, sanitization statement, digest, limitation, and revalidation date.
- Never delete or overwrite material historical evidence; add a new version.

## Prohibited shortcuts

- No wallet screens or financial features before prerequisite phase gates.
- No invented product semantics, endpoints, events, statuses, or direct database operations.
- No microservices, Kafka, Kubernetes, event sourcing, CQRS, blockchain, custom IdP, custom cryptography, or cloud-specific complexity without demonstrated need and an ADR.
- No direct status edits, arbitrary journal editor, or database edit as an operations workflow.
- No production provider, real data, browser-readable access tokens, wildcard credentialed CORS, secrets in source, or sensitive logging.
- No “exactly once,” compliance, security, availability, or scale claim without reproducible evidence and limitations.
- No phase completion claim without adversarial, restore/operations, and evidence gates.

## Links

- [PRD root](docs/atlas-prd/README.md)
- [Current phase](docs/atlas-prd/02-phases/PHASE-00_ENGINEERING_FOUNDATION.md)
- [Context routing index](docs/engineering/CONTEXT_INDEX.md)
- [Implementation status](docs/engineering/IMPLEMENTATION_STATUS.md)
- [Phase 00 execution plan](docs/engineering/PHASE-00-PLAN.md)
- [Definition of Done](docs/atlas-prd/06-governance/DEFINITION_OF_DONE.md)
