# Atlas implementation status

- **Status date:** 2026-07-21
- **Current phase:** [Phase 00 — Secure engineering foundation](../atlas-prd/02-phases/PHASE-00_ENGINEERING_FOUNDATION.md)
- **Current slice:** [S03 — contract-first health, HTTP safety, and trace seed](PHASE-00-PLAN.md#s03--contract-first-health-http-safety-and-trace-seed) is committed and post-commit verified as `b5fd25bac7844cfe929e28869d7c12f26e91b200`. S04 is now authorized and is the active implementation slice.
- **Implementation state:** Feature-free engineering foundation with typed Go primitives, static safety policies, and three operational API endpoints. No product or financial workflow, worker job, simulator scenario, database, broker, identity integration, runtime telemetry exporter, or frontend behavior exists.

## Repository baseline

| Area | Verified state |
|---|---|
| Version control | Valid Git repository on branch `main`; origin is `https://github.com/MichaelSeveen/atlas.git`. S03 implementation commit `b5fd25bac7844cfe929e28869d7c12f26e91b200` is locally post-commit verified. The local branch is ahead of origin; no push is claimed. |
| Specification | Canonical PRD is `docs/atlas-prd/`: 59 pack files reported, 399 requirements, 60 threats, 154 adversarial tests, OpenAPI 3.1.1 (33 paths/41 operations), and AsyncAPI 3.0.0 (9 channels/17 messages). S03 adds the three operational HTTP paths and corrects opaque-ID examples without creating a duplicate contract. |
| Application code | Go module `github.com/MichaelSeveen/atlas`; `cmd/api` now serves only liveness, readiness, and version while `cmd/worker` and `cmd/simulator` remain inert. Six narrow platform packages, architecture/layout/policy tests, and focused HTTP/contract tests exist. No product behavior, schemas, external Go dependencies, frontend package, or generated client. |
| Tooling | Go pin, formatting/ignore policy, repository-owned Go-only S01/S02/S03 verification commands, module/static checks, bounded fuzz campaigns, and seeded mutation/canary proof. React + TypeScript is the sole frontend framework; its runtime, dependency manifest, package manager, and build toolchain remain deferred. CI, CODEOWNERS enforcement, SBOM/provenance, scanners, and container definitions remain absent. |
| Local environment | No database, broker, Redis, object storage, IdP, collector/exporter, reset, or seed configuration. The API defaults to `127.0.0.1:8080`, is live, and deliberately reports not-ready until real probes exist. |
| Verified pins | Go 1.25.7, with Go language baseline 1.25.0. Module path `github.com/MichaelSeveen/atlas` matches the configured origin. React + TypeScript is selected, but no frontend build toolchain is pinned through S03. |
| Sensitive/generated/binary scan | Basic current-tree and initial-history key/token/secret-assignment scans found no candidate material. No build binary is retained; `.tmp/` build/module caches are ignored. The eleven verified root PRD duplicates were removed and the architecture test rejects their reappearance. Dedicated history scanning remains S07 work. |

## Phase 00 requirement state

| Classification | Count | Requirement IDs |
|---|---:|---|
| Satisfied | 8 | `FND-001`, `FND-002`, `FND-003`, `FND-004`, `FND-005`, `FND-006`, `FND-051`, `FND-053` |
| Partially satisfied | 6 | `FND-040`, `FND-043`, `FND-050`, `FND-052`, `FND-054`, `FND-060` |
| Absent | 23 | `FND-010..013`, `FND-020..027`, `FND-030..033`, `FND-041`, `FND-042`, `FND-055`, `FND-061..064` |
| Conflicting | 0 | None identified. |
| Not yet assessed | 0 | All 37 Phase 00 requirement IDs were assessed. |

”Satisfied” is requirement-scoped: S01 layout/process boundaries, S02 platform primitives/static bans, React choice, the S03 API-edge HTTP safeguards, and classification/logging definition are verified. It does not imply CI enforcement, database ownership, frontend behavior, runtime telemetry, later slices, or that the Phase 00 acceptance gate passes. See the [per-requirement audit](PHASE-00-PLAN.md#requirement-by-requirement-audit).

## Completed requirement IDs

- `FND-001` — roadmap-aligned directories, canonical-source guard, pinned Go metadata, and repository-owned verification exist.
- `FND-002` — dependency rules are documented and enforced by a clean-tree scanner plus a seeded cross-context persistence-import rejection.
- `FND-003` — API, worker, and provider-simulator Go entry points build independently; only the API has a runtime lifecycle and the three contract-defined operational endpoints.
- `FND-004` — React + TypeScript is consistently selected in the PRD, with no competing frontend implementation.
- `FND-005` — bounded integer money/currency, cryptographically random opaque IDs, injectable UTC clocks, explicit actor/correlation contexts, and data-minimizing domain errors pass table/property/fuzz and mutation proof.
- `FND-006` — the architecture checker rejects seeded floating-money and direct domain wall-clock violations while permitting explicit safe controls.
- `FND-051` — classification and logging rules are defined in the security and reliability specifications. Enforcement is separately outstanding under `FND-041`.
- `FND-053` — the API edge enforces secure headers, exact-origin CORS, fixed route/query/body/decompression limits, server deadlines, safe panic/error handling, and topology-free health responses under adversarial tests.

## Active requirement IDs

- `FND-040` is partial: validated request/correlation/W3C trace context is proven at the API edge and into its readiness check, but worker, simulator, database-span, and event propagation do not exist.
- `FND-043` is partial: a deterministic in-memory golden trace proves linked API-server and readiness spans with bounded fields, but there is no web/database/outbox/worker/simulator path or runtime exporter.
- `FND-054` remains partial because Go is pinned and verified while the frontend build toolchain and application dependency/image/CI-action verification remain future work.

S03 is requirement-scoped complete in the current worktree. S04 has not been authorized.

## Decisions and blockers

| Decision/gap | Impact | Required resolution |
|---|---|---|
| Broker and identity provider are not selected | Blocks the full local stack, integration tests, realm separation, and broker semantics. | Create decision records before S04; do not introduce Kafka or a custom IdP by default. |
| Deployment platform and secret-management implementation are not selected | Blocks production-reference configuration, signing/provenance, credential isolation, backup/PITR, and rotation procedure. | Resolve with scoped ADRs before S04/S05/S07, using abstractions and reversible local/reference choices. |
| Frontend route-shell/generated-client strategy is undecided | React is selected, but shell separation and generated/verified client approach remain open. | Decide before frontend/contract slice; preserve separate identity realms, request clients, and authorization boundaries. |

These are missing implementation decisions, not contradictory product semantics. No accepted ADR conflict was found.

## Known deviations

- Roadmap directories now exist, but most are intentional ownership placeholders and must not be described as implemented capability.
- The architecture decision index says `06-governance/adr/`; the real accepted-ADR directory is `06-governance/adrs/`.
- The sole mutable PRD contracts live under `docs/atlas-prd/03-contracts/`; implementation-owned publication/generation remains deferred and must not create a second hand-edited source.
- S03 trace and metric recorders default to discard because no runtime collector/exporter exists. The golden trace is an in-memory smoke seed, not deployed observability.
- The PRD validation report proves planning-pack consistency only; it is not implementation, security, performance, recovery, or compliance evidence.

## Evidence links

- [PRD pack validation report](../atlas-prd/PACK_VALIDATION_REPORT.md)
- [PRD integrity manifest](../atlas-prd/MANIFEST.sha256)
- [Requirements traceability matrix](../atlas-prd/06-governance/REQUIREMENTS_TRACEABILITY.csv)
- [Threat register](../atlas-prd/06-governance/THREAT_REGISTER.csv)
- [Phase 00 audit and execution plan](PHASE-00-PLAN.md)
- [Current S01 boundary report](../../evidence/phase-00/architecture/S01-boundary-report-v3.md)
- [Current S02 primitives report](../../evidence/phase-00/primitives/S02-primitives-report.md)
- [Current S03 HTTP foundation report](../../evidence/phase-00/http/S03-http-foundation-report.md)
- [Canonical PRD cleanup report](../../evidence/phase-00/architecture/PRD-canonicalization-report.md)
- [Module boundary model](MODULE_BOUNDARIES.md)
- [Platform primitives and static policy](PLATFORM_PRIMITIVES.md)
- [HTTP foundation](HTTP_FOUNDATION.md)
- [Toolchain policy](TOOLCHAIN_POLICY.md)

## Last verified source revision

S03 implementation commit `b5fd25bac7844cfe929e28869d7c12f26e91b200` (tree `dc62b1448e4d0d8499e4c3a7b31d3224915cf00b`) passed the full repository-owned S03 verifier from a clean worktree on 2026-07-21. The local commit has not been pushed.

The [S03 post-commit verification](../../evidence/phase-00/http/S03-post-commit-verification.md) binds the result to the implementation commit; the original [pre-commit S03 report](../../evidence/phase-00/http/S03-http-foundation-report.md) remains preserved. Historical S01/S02/canonicalization reports remain unchanged.
