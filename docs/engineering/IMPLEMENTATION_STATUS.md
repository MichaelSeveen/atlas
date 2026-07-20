# Atlas implementation status

- **Status date:** 2026-07-20
- **Current phase:** [Phase 00 — Secure engineering foundation](../atlas-prd/02-phases/PHASE-00_ENGINEERING_FOUNDATION.md)
- **Current slice:** [S02 — safe cross-cutting primitives and static bans](PHASE-00-PLAN.md#s02--safe-cross-cutting-primitives-and-static-bans) is implemented and committed. The owner-authorized canonical-PRD cleanup is complete as an S03 prerequisite; S03 is not authorized or started.
- **Implementation state:** Feature-free engineering foundation with typed Go primitives and static safety policies; no product, financial workflow, endpoint, worker-job, simulator-scenario, database, broker, identity integration, or frontend behavior exists.

## Repository baseline

| Area | Verified state |
|---|---|
| Version control | Valid Git repository on branch `main`; origin is `https://github.com/MichaelSeveen/atlas.git`. S02 implementation commit is `dc638d2`; owner-authorized canonical-PRD cleanup commit `240adbf` removes only the eleven reverified root duplicates and makes their absence enforceable. |
| Specification | Canonical PRD is `docs/atlas-prd/`: 59 pack files reported, 399 requirements, 60 threats, 154 adversarial tests, OpenAPI 3.1.1 (30 paths/38 operations), and AsyncAPI 3.0.0 (9 channels/17 messages). S01/S02 traceability rows and the PRD manifest were updated together. |
| Application code | Go module `github.com/MichaelSeveen/atlas`, derived from the configured GitHub origin; separate inert `cmd/api`, `cmd/worker`, and `cmd/simulator` entry points; six narrow platform packages for money, IDs, clocks, actor/correlation context, and errors; architecture/layout/policy tests. No product behavior, schemas, external runtime dependencies, frontend package, or generated client. |
| Tooling | Go pin, formatting/ignore policy, repository-owned Go-only S01/S02 verification commands, module-boundary checker, float-money/wall-clock static bans, fuzz campaigns, mutation proof, and update policy. React + TypeScript is the sole frontend framework; its runtime, dependency manifest, package manager, and build toolchain are deliberately deferred. CI, CODEOWNERS enforcement, SBOM/provenance, scanners, and container definitions remain absent. |
| Local environment | No database, broker, Redis, object-storage, IdP, telemetry, application, reset, or seed configuration. |
| Verified pins | Go 1.25.7, with Go language baseline 1.25.0. Module path `github.com/MichaelSeveen/atlas` matches the configured origin. React + TypeScript is selected, but no frontend build toolchain is pinned through S02. |
| Sensitive/generated/binary scan | Basic current-tree and initial-history key/token/secret-assignment scans found no candidate material. No build binary is retained; `.tmp/` build/module caches are ignored. The eleven verified root PRD duplicates were removed and the architecture test rejects their reappearance. Dedicated history scanning remains S07 work. |

## Phase 00 requirement state

| Classification | Count | Requirement IDs |
|---|---:|---|
| Satisfied | 7 | `FND-001`, `FND-002`, `FND-003`, `FND-004`, `FND-005`, `FND-006`, `FND-051` |
| Partially satisfied | 4 | `FND-050`, `FND-052`, `FND-054`, `FND-060` |
| Absent | 26 | `FND-010..013`, `FND-020..027`, `FND-030..033`, `FND-040..043`, `FND-053`, `FND-055`, `FND-061..064` |
| Conflicting | 0 | None identified. |
| Not yet assessed | 0 | All 37 Phase 00 requirement IDs were assessed. |

“Satisfied” is requirement-scoped: S01 layout/process boundaries, S02 platform primitives/static bans, React choice, and classification/logging definition are verified. It does not imply CI enforcement, database ownership, frontend behavior, later slices, or that the Phase 00 acceptance gate passes. See the [per-requirement audit](PHASE-00-PLAN.md#requirement-by-requirement-audit).

## Completed requirement IDs

- `FND-001` — roadmap-aligned directories, canonical-source guard, pinned Go metadata, and repository-owned verification exist.
- `FND-002` — dependency rules are documented and enforced by a clean-tree scanner plus a seeded cross-context persistence-import rejection.
- `FND-003` — API, worker, and provider-simulator Go entry points build independently and contain no runtime/product behavior.
- `FND-004` — React + TypeScript is consistently selected in the PRD, with no competing frontend implementation.
- `FND-005` — bounded integer money/currency, cryptographically random opaque IDs, injectable UTC clocks, explicit actor/correlation contexts, and data-minimizing domain errors pass table/property/fuzz and mutation proof.
- `FND-006` — the architecture checker rejects seeded floating-money and direct domain wall-clock violations while permitting explicit safe controls.
- `FND-051` — classification and logging rules are defined in the security and reliability specifications. Enforcement is separately outstanding under `FND-041`.

## Active requirement IDs

None. S01 and S02 are requirement-scoped complete; S03 is the next planned slice but has not been authorized. `FND-054` is partial because S01 pins and verifies Go and documents updates, while the frontend build toolchain and all application dependency/image/CI-action verification and emergency workflow remain future work.

## Decisions and blockers

| Decision/gap | Impact | Required resolution |
|---|---|---|
| Broker and identity provider are not selected | Blocks the full local stack, integration tests, realm separation, and broker semantics. | Create decision records before S04; do not introduce Kafka or a custom IdP by default. |
| Deployment platform and secret-management implementation are not selected | Blocks production-reference configuration, signing/provenance, credential isolation, backup/PITR, and rotation procedure. | Resolve with scoped ADRs before S04/S05/S07, using abstractions and reversible local/reference choices. |
| Frontend route-shell/generated-client strategy is undecided | React is selected, but shell separation and generated/verified client approach remain open. | Decide before frontend/contract slice; preserve separate identity realms, request clients, and authorization boundaries. |
| Phase 00 health/version paths are absent from OpenAPI | Implementing handlers first would violate contract ownership. | Add `/health/live`, `/health/ready`, and `/version` contract definitions before or with their first implementation. |
| Opaque-ID contract examples violate the normative regex | Several examples contain `L` in an `ATLAS` mnemonic, but the canonical Crockford pattern excludes `I`, `L`, `O`, and `U`. S02 correctly rejects those examples. | Correct examples contract-first in S03 or record an accepted contract decision before changing the invariant. |

These are missing implementation decisions, not contradictory product semantics. No accepted ADR conflict was found.

## Known deviations

- Roadmap directories now exist, but most are intentional ownership placeholders and must not be described as implemented capability.
- The architecture decision index says `06-governance/adr/`; the real accepted-ADR directory is `06-governance/adrs/`.
- The PRD contracts live under `docs/atlas-prd/03-contracts/`; no implementation-owned contract publication/generation location exists yet.
- The PRD validation report proves planning-pack consistency only; it is not implementation, security, performance, recovery, or compliance evidence.

## Evidence links

- [PRD pack validation report](../atlas-prd/PACK_VALIDATION_REPORT.md)
- [PRD integrity manifest](../atlas-prd/MANIFEST.sha256)
- [Requirements traceability matrix](../atlas-prd/06-governance/REQUIREMENTS_TRACEABILITY.csv)
- [Threat register](../atlas-prd/06-governance/THREAT_REGISTER.csv)
- [Phase 00 audit and execution plan](PHASE-00-PLAN.md)
- [Current S01 boundary report](../../evidence/phase-00/architecture/S01-boundary-report-v3.md)
- [Current S02 primitives report](../../evidence/phase-00/primitives/S02-primitives-report.md)
- [Canonical PRD cleanup report](../../evidence/phase-00/architecture/PRD-canonicalization-report.md)
- [Module boundary model](MODULE_BOUNDARIES.md)
- [Platform primitives and static policy](PLATFORM_PRIMITIVES.md)
- [Toolchain policy](TOOLCHAIN_POLICY.md)

## Last verified source revision

S02 implementation is committed as `dc638d2949335fc5808aea39906618406cd5c042`. Owner-authorized canonical-PRD cleanup revision `240adbf32b73951062b9e2233e1aa6257b4d386d` was verified after commit: all eleven root/canonical pairs had been byte-identical immediately before deletion, every canonical artifact remains, the hard absence check passes, and all 58 entries in `docs/atlas-prd/MANIFEST.sha256` match.

The [canonicalization report](../../evidence/phase-00/architecture/PRD-canonicalization-report.md) records the committed cleanup revision and post-commit verification. Historical S01/S02 reports remain unchanged.
