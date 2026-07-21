# Atlas implementation status

- **Status date:** 2026-07-21
- **Current phase:** [Phase 00 — Secure engineering foundation](../atlas-prd/02-phases/PHASE-00_ENGINEERING_FOUNDATION.md)
- **Current slice:** [S04 — reproducible synthetic local/reference environment](PHASE-00-PLAN.md#s04--reproducible-synthetic-localreference-environment) is committed and post-commit verified as `39121a31765013ebdc51b3b0ac4e47c9bc8b1516`. Clean-machine execution of the exact wrapper remains a recorded limitation; S05 is not started.
- **Implementation state:** Feature-free engineering foundation with typed Go primitives, static policies, three operational API endpoints, a complete synthetic local dependency/process topology, strict environment configuration, deterministic fixture identities/scenario catalogue, and three React route shells. No product endpoint, financial workflow, schema, worker job, executable provider scenario, broker stream, identity exchange, runtime telemetry export path, or wallet UI exists.

## Repository baseline

| Area | Verified state |
|---|---|
| Version control | Valid Git repository on branch `main`; origin is `https://github.com/MichaelSeveen/atlas.git`. S04 implementation commit `39121a31765013ebdc51b3b0ac4e47c9bc8b1516` is locally post-commit verified. The local branch is ahead of origin; no push is claimed. |
| Specification | Canonical PRD is `docs/atlas-prd/`: the 59-file validated baseline now has two accepted S04 ADRs (61 versioned files including the manifest), while the baseline report retains 399 requirements, 60 threats, 154 adversarial tests, OpenAPI 3.1.1 (33 paths/41 operations), and AsyncAPI 3.0.0 (9 channels/17 messages). S03/S04 edits preserve one canonical contract/spec root. |
| Application code | Go module `github.com/MichaelSeveen/atlas`; `cmd/api` serves only liveness, readiness, and version with typed dependency probes, while worker/simulator implement only config validation and bounded process lifecycle. React owns three feature-free route shells. Seven narrow platform packages, architecture/layout/toolchain policies, and focused HTTP/environment/contract tests exist. No product behavior, schemas, external Go dependency, or generated product client. |
| Tooling | Go 1.25.7 and Bun 1.3.0/React 19.2.7 pins, frozen `bun.lock`, repository-owned S01–S04 verification, module/static/toolchain checks, bounded fuzz, mutation, configuration, seed, reset, live, and browser canaries. CI, CODEOWNERS enforcement, SBOM/provenance, scanners, signing, and immutable image-digest promotion remain absent. |
| Local environment | Compose-compatible PostgreSQL, Redis, NATS JetStream, MinIO, OTel Collector, Keycloak, API, worker, simulator, and web run in a constrained loopback-only synthetic namespace. Reset is exact-confirmation and contained. The current host required a Podman WSL/systemd/provider workaround, so clean-machine one-command proof is still outstanding. |
| Verified pins | Go 1.25.7/language 1.25.0; module `github.com/MichaelSeveen/atlas`; Bun 1.3.0; React/React DOM 19.2.7; exact S04 service image tags. S07 must add digest locking, SBOM, provenance, scanning, and update automation. |
| Sensitive/generated/binary scan | Basic current-tree and initial-history key/token/secret-assignment scans found no candidate material. No build binary is retained; `.tmp/` build/module caches are ignored. The eleven verified root PRD duplicates were removed and the architecture test rejects their reappearance. Dedicated history scanning remains S07 work. |

## Phase 00 requirement state

| Classification | Count | Requirement IDs |
|---|---:|---|
| Satisfied | 13 | `FND-001..006`, `FND-012`, `FND-013`, `FND-030`, `FND-032`, `FND-033`, `FND-051`, `FND-053` |
| Partially satisfied | 9 | `FND-010`, `FND-011`, `FND-031`, `FND-040`, `FND-043`, `FND-050`, `FND-052`, `FND-054`, `FND-060` |
| Absent | 15 | `FND-020..027`, `FND-041`, `FND-042`, `FND-055`, `FND-061..064` |
| Conflicting | 0 | None identified. |
| Not yet assessed | 0 | All 37 Phase 00 requirement IDs were assessed. |

”Satisfied” is requirement-scoped: S01 layout/process boundaries, S02 primitives/static bans, the S03 API edge, and the specified S04 synthetic/config/reset/banner/flag facets are verified. It does not imply clean-machine acceptance, CI enforcement, database ownership, application seeds, provider behavior, identity integration, runtime telemetry, later slices, or that the Phase 00 gate passes. See the [per-requirement audit](PHASE-00-PLAN.md#requirement-by-requirement-audit).

## Completed requirement IDs

- `FND-001` — roadmap-aligned directories, canonical-source guard, pinned Go metadata, and repository-owned verification exist.
- `FND-002` — dependency rules are documented and enforced by a clean-tree scanner plus a seeded cross-context persistence-import rejection.
- `FND-003` — API, worker, and provider-simulator Go entry points build independently; only the API has a runtime lifecycle and the three contract-defined operational endpoints.
- `FND-004` — React + TypeScript is consistently selected in the PRD, with no competing frontend implementation.
- `FND-005` — bounded integer money/currency, cryptographically random opaque IDs, injectable UTC clocks, explicit actor/correlation contexts, and data-minimizing domain errors pass table/property/fuzz and mutation proof.
- `FND-006` — the architecture checker rejects seeded floating-money and direct domain wall-clock violations while permitting explicit safe controls.
- `FND-012` — portfolio configuration is synthetic-only, loopback/reserved-host constrained, and rejects real/public endpoint, development-key, wildcard, and missing-synthetic canaries.
- `FND-013` — reset is limited to local/test, validates target containment, prints its resolved target, and requires the exact environment confirmation.
- `FND-030` — strict local, test, staging, and production-reference configurations are present and validated as one closed set.
- `FND-032` — all three React actor shells render the persistent synthetic banner and pass live/browser no-store, empty-storage, logout, and back-navigation proof.
- `FND-033` — flags require complete owner/expiry/default/risk/rollback metadata and have immutable fail-closed/default-on-outage evaluation tests.
- `FND-051` — classification and logging rules are defined in the security and reliability specifications. Enforcement is separately outstanding under `FND-041`.
- `FND-053` — the API edge enforces secure headers, exact-origin CORS, fixed route/query/body/decompression limits, server deadlines, safe panic/error handling, and topology-free health responses under adversarial tests.

## Active requirement IDs

- `FND-010` is partial: the complete constrained stack, readiness, restart, teardown, and smoke pass through Compose, but this host needed a Podman WSL/systemd/provider repair and the exact clean-machine wrapper command is not yet independently proven.
- `FND-011` is partial: deterministic synthetic identity/account labels and provider scenario IDs validate with fixed checksum and tenant ownership, but no application schema is loaded and no provider behavior executes.
- `FND-031` is partial: four-environment credential references and generated local/test password/token fingerprints never overlap, but staging/production provisioning, rotation, restore, and secret-manager evidence do not exist.
- `FND-040` is partial: validated request/correlation/W3C trace context is proven at the API edge and into its readiness check, but worker, simulator, database-span, and event propagation do not exist.
- `FND-043` is partial: a deterministic in-memory golden trace proves linked API-server and readiness spans with bounded fields, but there is no web/database/outbox/worker/simulator path or runtime exporter.
- `FND-054` remains partial because Go is pinned and verified while the frontend build toolchain and application dependency/image/CI-action verification remain future work.

S04 is requirement-scoped implemented at `39121a31765013ebdc51b3b0ac4e47c9bc8b1516` with the stated partials and host limitation. S05 has not started.

## Decisions and blockers

| Decision/gap | Impact | Required resolution |
|---|---|---|
| Production broker, IdP deployment, object store, and secret manager are not selected | Local/reference products are accepted only by ADR 0008; production semantics, key rotation, backup, and promotion remain blocked. | Resolve with scoped ADRs before S07/S08; do not treat local NATS/Keycloak/MinIO as a production selection. |
| Generated product-client strategy is undecided | S04 has only a typed runtime-config fetch and must not invent product calls. | Select and enforce generation/compatibility from the canonical OpenAPI in S07. |
| Current Podman host bootstrap is unhealthy | The existing never-started WSL machine lacked systemd and a host Compose provider; live proof used a repaired VM and equivalent in-VM `podman-compose`. | Reprove the exact documented `scripts/s04.ps1 -Action Up` from a clean supported Podman/Docker host in S08. |

These are missing implementation decisions, not contradictory product semantics. No accepted ADR conflict was found.

## Known deviations

- Roadmap directories now exist, but most are intentional ownership placeholders and must not be described as implemented capability.
- The architecture decision index says `06-governance/adr/`; the real accepted-ADR directory is `06-governance/adrs/`.
- The sole mutable PRD contracts live under `docs/atlas-prd/03-contracts/`; implementation-owned publication/generation remains deferred and must not create a second hand-edited source.
- S03 trace and metric recorders default to discard because no runtime collector/exporter exists. The golden trace is an in-memory smoke seed, not deployed observability.
- The S04 collector is reachable for topology/readiness only; application exporters and golden full-stack telemetry remain S06/S08.
- S04 seed artifacts are validated catalogues, not inserted application data; no schema exists before S05.
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
- [Current S04 synthetic environment report](../../evidence/phase-00/environment/S04-environment-report.md)
- [S04 post-commit verification](../../evidence/phase-00/environment/S04-post-commit-verification.md)
- [Canonical PRD cleanup report](../../evidence/phase-00/architecture/PRD-canonicalization-report.md)
- [Module boundary model](MODULE_BOUNDARIES.md)
- [Platform primitives and static policy](PLATFORM_PRIMITIVES.md)
- [HTTP foundation](HTTP_FOUNDATION.md)
- [Synthetic local environment](LOCAL_ENVIRONMENT.md)
- [Toolchain policy](TOOLCHAIN_POLICY.md)

## Last verified source revision

S04 implementation commit `39121a31765013ebdc51b3b0ac4e47c9bc8b1516` (tree `bf150ddc7a60f7b66ca362c4e4aee6e91831f8c0`) passed the full repository-owned S04 verifier from a clean worktree on 2026-07-21. Its API image reported that exact revision. The [post-commit verification](../../evidence/phase-00/environment/S04-post-commit-verification.md) binds the result; the detailed [pre-commit environment report](../../evidence/phase-00/environment/S04-environment-report.md) remains preserved.

S03 remains post-commit verified at implementation commit `b5fd25bac7844cfe929e28869d7c12f26e91b200` (tree `dc62b1448e4d0d8499e4c3a7b31d3224915cf00b`). No S03/S04 commit has been pushed in this task.
