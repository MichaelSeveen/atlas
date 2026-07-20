# Atlas implementation status

- **Status date:** 2026-07-20
- **Current phase:** [Phase 00 — Secure engineering foundation](../atlas-prd/02-phases/PHASE-00_ENGINEERING_FOUNDATION.md)
- **Current slice:** [S01 — versioned repository and process-boundary scaffold](PHASE-00-PLAN.md#s01--versioned-repository-and-process-boundary-scaffold) is implemented and verified with the pre-commit limitation below. S02 is not authorized or started.
- **Implementation state:** Feature-free engineering scaffold; no product, financial, API, worker-job, simulator-scenario, database, or frontend behavior exists.

## Repository baseline

| Area | Verified state |
|---|---|
| Version control | Valid Git repository on branch `main`, initialized during S01. `HEAD` is unborn and every file is uncommitted/untracked; no commit or history scan exists because the user prohibited creating a commit without separate approval. |
| Specification | Canonical PRD is `docs/atlas-prd/`: 59 pack files reported, 399 requirements, 60 threats, 154 adversarial tests, OpenAPI 3.1.1 (30 paths/38 operations), and AsyncAPI 3.0.0 (9 channels/17 messages). S01 traceability rows and the PRD manifest were updated together. |
| Application code | Go module `github.com/MichaelSeveen/atlas`, derived from the configured GitHub origin; separate inert `cmd/api`, `cmd/worker`, and `cmd/simulator` entry points; architecture/layout tests. No product behavior, schemas, runtime dependencies, frontend package, or generated client. |
| Tooling | Go pin, formatting/ignore policy, repository-owned Go-only S01 verification command, module-boundary checker, and update policy. React + TypeScript is the sole frontend framework; its runtime, dependency manifest, package manager, and build toolchain are deliberately deferred. CI, CODEOWNERS enforcement, SBOM/provenance, scanners, and container definitions remain absent. |
| Local environment | No database, broker, Redis, object-storage, IdP, telemetry, application, reset, or seed configuration. |
| Verified pins | Go 1.25.7, with Go language baseline 1.25.0. Module path `github.com/MichaelSeveen/atlas` matches the configured origin. React + TypeScript is selected, but no frontend build toolchain is pinned in S01. |
| Sensitive/generated/binary scan | Basic key/token/secret-assignment scan found no candidate material. No build binary is retained; `.tmp/` build cache is ignored. Eleven root PRD duplicates remain and are hash-guarded against canonical sources. Git-history scanning remains impossible until a first commit/history exists. |

## Phase 00 requirement state

| Classification | Count | Requirement IDs |
|---|---:|---|
| Satisfied | 5 | `FND-001`, `FND-002`, `FND-003`, `FND-004`, `FND-051` |
| Partially satisfied | 4 | `FND-050`, `FND-052`, `FND-054`, `FND-060` |
| Absent | 28 | `FND-005`, `FND-006`, `FND-010..013`, `FND-020..027`, `FND-030..033`, `FND-040..043`, `FND-053`, `FND-055`, `FND-061..064` |
| Conflicting | 0 | None identified. |
| Not yet assessed | 0 | All 37 Phase 00 requirement IDs were assessed. |

“Satisfied” is requirement-scoped: S01 layout, entry-point builds, clean import scan, seeded forbidden-import rejection, React choice, and classification/logging definition are verified. It does not imply CI, database ownership, frontend behavior, later slices, or the Phase 00 acceptance gate passes. See the [per-requirement audit](PHASE-00-PLAN.md#requirement-by-requirement-audit).

## Completed requirement IDs

- `FND-001` — roadmap-aligned directories, canonical-source guard, pinned Go metadata, and repository-owned verification exist.
- `FND-002` — dependency rules are documented and enforced by a clean-tree scanner plus a seeded cross-context persistence-import rejection.
- `FND-003` — API, worker, and provider-simulator Go entry points build independently and contain no runtime/product behavior.
- `FND-004` — React + TypeScript is consistently selected in the PRD, with no competing frontend implementation.
- `FND-051` — classification and logging rules are defined in the security and reliability specifications. Enforcement is separately outstanding under `FND-041`.

## Active requirement IDs

None. S01 is complete; S02 (`FND-005`, `FND-006`) is the next planned slice but has not been authorized. `FND-054` is partial because S01 pins and verifies Go and documents updates, while the frontend build toolchain and all application dependency/image/CI-action verification and emergency workflow remain future work.

## Decisions and blockers

| Decision/gap | Impact | Required resolution |
|---|---|---|
| Git `HEAD` is unborn | S01 evidence can identify branch/configuration/digests but not a commit; history secret checks, provenance, and clean-clone proof remain impossible. | Owner reviews the uncommitted scaffold and separately authorizes the first commit. Rerun S01 afterward and supersede the pre-commit evidence. |
| Root-level duplicate PRD files exist | Eleven files currently hash-identically to canonical files in `docs/atlas-prd/`, but two mutable locations can drift. | Treat `docs/atlas-prd/` as authoritative; decide whether to remove the verified duplicates before the first source baseline. |
| Broker and identity provider are not selected | Blocks the full local stack, integration tests, realm separation, and broker semantics. | Create decision records before S04; do not introduce Kafka or a custom IdP by default. |
| Deployment platform and secret-management implementation are not selected | Blocks production-reference configuration, signing/provenance, credential isolation, backup/PITR, and rotation procedure. | Resolve with scoped ADRs before S04/S05/S07, using abstractions and reversible local/reference choices. |
| Frontend route-shell/generated-client strategy is undecided | React is selected, but shell separation and generated/verified client approach remain open. | Decide before frontend/contract slice; preserve separate identity realms, request clients, and authorization boundaries. |
| Phase 00 health/version paths are absent from OpenAPI | Implementing handlers first would violate contract ownership. | Add `/health/live`, `/health/ready`, and `/version` contract definitions before or with their first implementation. |

These are missing implementation decisions, not contradictory product semantics. No accepted ADR conflict was found.

## Known deviations

- Roadmap directories now exist, but most are intentional ownership placeholders and must not be described as implemented capability.
- Eleven non-authoritative root files duplicate canonical PRD artifacts byte-for-byte.
- The architecture decision index says `06-governance/adr/`; the real accepted-ADR directory is `06-governance/adrs/`.
- The PRD contracts live under `docs/atlas-prd/03-contracts/`; no implementation-owned contract publication/generation location exists yet.
- The PRD validation report proves planning-pack consistency only; it is not implementation, security, performance, recovery, or compliance evidence.

## Evidence links

- [PRD pack validation report](../atlas-prd/PACK_VALIDATION_REPORT.md)
- [PRD integrity manifest](../atlas-prd/MANIFEST.sha256)
- [Requirements traceability matrix](../atlas-prd/06-governance/REQUIREMENTS_TRACEABILITY.csv)
- [Threat register](../atlas-prd/06-governance/THREAT_REGISTER.csv)
- [Phase 00 audit and execution plan](PHASE-00-PLAN.md)
- [Current S01 boundary report](../../evidence/phase-00/architecture/S01-boundary-report-v2.md)
- [Module boundary model](MODULE_BOUNDARIES.md)
- [Toolchain policy](TOOLCHAIN_POLICY.md)

## Last verified source revision

No Git commit/source revision exists; `HEAD` is `UNBORN`. S01 evidence is therefore pre-commit and must be superseded after the first owner-approved commit. The specification baseline was verified against PRD version/date `2026-07-20`; all 58 entries in `docs/atlas-prd/MANIFEST.sha256` matched after the traceability update, and the manifest file SHA-256 was:

```text
48E77F2217D177444FF16A939F48BE1247335659705FC8ADE5FF22F5642B84D8
```

Replace this manifest-only identifier with a Git commit plus build/configuration digests after the first commit is explicitly authorized.
