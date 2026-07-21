# Atlas implementation status

- **Status date:** 2026-07-21
- **Current phase:** [Phase 00 — Secure engineering foundation](../atlas-prd/02-phases/PHASE-00_ENGINEERING_FOUNDATION.md)
- **Current slice:** [S05 — database roles, migration safety, and recovery foundation](PHASE-00-PLAN.md#s05--database-roles-migration-safety-and-recovery-foundation) is implemented and pre-commit verified against `UNCOMMITTED_WORKTREE(base=199b86113a9f0fcda323ae2775acf026b521067e)`. Static and equivalent in-VM live commands pass; post-commit and clean-host wrapper proof remain outstanding.
- **Implementation state:** Feature-free engineering foundation with typed Go primitives, static policies, three operational API endpoints, a complete synthetic local dependency/process topology, strict environment configuration, deterministic fixture identities/scenario catalogue, three React route shells, and PostgreSQL migration/role/readiness/recovery controls. No product endpoint, financial workflow, product schema, worker job, executable provider scenario, broker stream, identity exchange, runtime telemetry export path, or wallet UI exists.

## Repository baseline

| Area | Verified state |
|---|---|
| Version control | Valid Git repository on branch `main`; origin is `https://github.com/MichaelSeveen/atlas.git`. S04 implementation commit `39121a31765013ebdc51b3b0ac4e47c9bc8b1516` is locally post-commit verified. The local branch is ahead of origin; no push is claimed. |
| Specification | Canonical PRD is `docs/atlas-prd/`: the 59-file validated baseline now has three accepted implementation ADRs (62 versioned files including the manifest), while the baseline report retains 399 requirements, 60 threats, 154 adversarial tests, OpenAPI 3.1.1 (33 paths/41 operations), and AsyncAPI 3.0.0 (9 channels/17 messages). S03–S05 edits preserve one canonical contract/spec root. |
| Application code | Go module `github.com/MichaelSeveen/atlas`; `cmd/api` serves only liveness, readiness, and version with typed dependency and real migration-state probes, while worker/simulator remain feature-free. `cmd/dbctl` validates the released migration inventory. React owns three feature-free route shells. Nine narrow platform packages, architecture/layout/toolchain policies, and focused HTTP/environment/database/migration/contract tests exist. `pgx/v5` is the sole external Go application dependency. No product behavior, product schema, or generated product client exists. |
| Tooling | Go 1.25.7, pgx/v5 5.10.0, Bun 1.3.0, and React 19.2.7 are pinned; `bun.lock` is frozen. Repository-owned S01–S05 verification covers module/static/toolchain checks, bounded fuzz, mutation, configuration, seed, reset, live/browser, migration, role, lock, and recovery canaries. CI, CODEOWNERS enforcement, SBOM/provenance, scanners, signing, and immutable image-digest promotion remain absent. |
| Local environment | Compose-compatible PostgreSQL, Redis, NATS JetStream, MinIO, OTel Collector, Keycloak, API, worker, simulator, and web run in a constrained loopback-only synthetic namespace. S05 adds distinct database roles, a feature-free foundation schema, WAL/base-backup volumes, and an internal-only restore service. Reset is exact-confirmation and contained. The current host required a Podman WSL/systemd/provider workaround, so clean-machine one-command proof is still outstanding. |
| Verified pins | Go 1.25.7/language 1.25.0; module `github.com/MichaelSeveen/atlas`; pgx/v5 5.10.0; Bun 1.3.0; React/React DOM 19.2.7; exact S04 dependency/web and S05 backend image tags. S07 must add digest locking, SBOM, provenance, scanning, and update automation. |
| Sensitive/generated/binary scan | Basic current-tree and initial-history key/token/secret-assignment scans found no candidate material. No build binary is retained; `.tmp/` build/module caches are ignored. The eleven verified root PRD duplicates were removed and the architecture test rejects their reappearance. Dedicated history scanning remains S07 work. |

## Phase 00 requirement state

| Classification | Count | Requirement IDs |
|---|---:|---|
| Satisfied | 19 | `FND-001..006`, `FND-012`, `FND-013`, `FND-021`, `FND-025`, `FND-030`, `FND-032`, `FND-033`, `FND-051`, `FND-053`, `FND-060..063` |
| Partially satisfied | 9 | `FND-010`, `FND-011`, `FND-031`, `FND-040`, `FND-043`, `FND-050`, `FND-052`, `FND-054`, `FND-064` |
| Absent | 9 | `FND-020`, `FND-022..024`, `FND-026`, `FND-027`, `FND-041`, `FND-042`, `FND-055` |
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
- `FND-021` — live integration commands require real PostgreSQL roles and real NATS JetStream; the database/lock/recovery scripts cannot be satisfied by mocked repositories.
- `FND-025` — empty and previous-version throwaway databases reach the current schema, repeated application is idempotent, cleanup is bounded, and a long-lock migration aborts safely.
- `FND-060` — distinct migration, API, worker, reporting-read, and disabled break-glass identities use distinct generated credentials; the real permission matrix proves their allowed and denied paths.
- `FND-061` — API, worker, and reporting roles cannot create, alter, or drop schema objects, grant effective public access, assume the migration role, or create disallowed temporary state.
- `FND-062` — a closed released SHA-256 manifest covers SQL and risk metadata; changed and deleted released-file canaries are killed. Final post-commit evidence is still required before release claims.
- `FND-063` — every migration has closed lock/timeout/data/plan/space/forward-fix/rollback metadata, and the representative foundation lock canary proves bounded abort and transaction recovery.

## Active requirement IDs

- `FND-010` is partial: the complete constrained stack, readiness, restart, teardown, and smoke pass through Compose, but this host needed a Podman WSL/systemd/provider repair and the exact clean-machine wrapper command is not yet independently proven.
- `FND-011` is partial: deterministic synthetic identity/account labels and provider scenario IDs validate with fixed checksum and tenant ownership, but no application schema is loaded and no provider behavior executes.
- `FND-031` is partial: four-environment credential references and generated local/test password/token fingerprints never overlap, but staging/production provisioning, rotation, restore, and secret-manager evidence do not exist.
- `FND-040` is partial: validated request/correlation/W3C trace context is proven at the API edge and into its readiness check, but worker, simulator, database-span, and event propagation do not exist.
- `FND-043` is partial: a deterministic in-memory golden trace proves linked API-server and readiness spans with bounded fields, but there is no web/database/outbox/worker/simulator path or runtime exporter.
- `FND-054` remains partial because Go, pgx, Bun, and React are pinned and reproducibly installed, while dependency/license/scanner, base-image-digest, CI-action, SBOM, and emergency-update proof remain S07 work.
- `FND-064` is partial: a verified physical base backup, WAL archive, isolated target-time restore, migration checksum, and pre-deletion marker pass, but the local backup volume is unencrypted and no product object/key/inbox/outbox/idempotency or financial replay state exists.

S05 is requirement-scoped implemented and pre-commit verified against base `199b86113a9f0fcda323ae2775acf026b521067e`, with the stated FND-064 and host limitations. No S06 requirement is claimed complete.

## Decisions and blockers

| Decision/gap | Impact | Required resolution |
|---|---|---|
| Production broker, IdP deployment, object store, and secret manager are not selected | Local/reference products are accepted only by ADR 0008; production semantics, key rotation, backup, and promotion remain blocked. | Resolve with scoped ADRs before S07/S08; do not treat local NATS/Keycloak/MinIO as a production selection. |
| Generated product-client strategy is undecided | S04 has only a typed runtime-config fetch and must not invent product calls. | Select and enforce generation/compatibility from the canonical OpenAPI in S07. |
| Current Podman host bootstrap is unhealthy | The Windows host transport cannot reach the repaired VM/provider; S04/S05 live proof used equivalent in-VM `podman-compose` against the repository Compose file and scripts. | Reprove the exact documented `scripts/s04.ps1 -Action Up` and `scripts/verify-s05.ps1 -Live` from a clean supported Podman/Docker host in S08. |
| Local backup/WAL volumes are not encrypted | S05 proves native backup/WAL/PITR mechanics but cannot satisfy the encrypted reference-environment and key-access facets of FND-064. | Select deployment/object/key controls in S06/S07 and run the complete encrypted isolated restore/replay gate in S08. |

These are missing implementation decisions, not contradictory product semantics. No accepted ADR conflict was found.

## Known deviations

- Roadmap directories now exist, but most are intentional ownership placeholders and must not be described as implemented capability.
- The architecture decision index says `06-governance/adr/`; the real accepted-ADR directory is `06-governance/adrs/`.
- The sole mutable PRD contracts live under `docs/atlas-prd/03-contracts/`; implementation-owned publication/generation remains deferred and must not create a second hand-edited source.
- S03 trace and metric recorders default to discard because no runtime collector/exporter exists. The golden trace is an in-memory smoke seed, not deployed observability.
- The S04 collector is reachable for topology/readiness only; application exporters and golden full-stack telemetry remain S06/S08.
- S04 seed artifacts remain validated catalogues, not inserted application data. S05 creates only the feature-free `atlas_foundation` control schema; no product schema exists.
- Project policy forbids React class components. Because React 19 has no function-component error-boundary API, the feature-free web shell now uses a route-aware root `onUncaughtError` fallback; strict subtree-local error containment is not claimed.
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
- [Current S05 database report](../../evidence/phase-00/database/S05-database-report.md)
- [Canonical PRD cleanup report](../../evidence/phase-00/architecture/PRD-canonicalization-report.md)
- [Module boundary model](MODULE_BOUNDARIES.md)
- [Platform primitives and static policy](PLATFORM_PRIMITIVES.md)
- [HTTP foundation](HTTP_FOUNDATION.md)
- [Synthetic local environment](LOCAL_ENVIRONMENT.md)
- [Database foundation](DATABASE_FOUNDATION.md)
- [Toolchain policy](TOOLCHAIN_POLICY.md)

## Last verified source revision

S04 implementation commit `39121a31765013ebdc51b3b0ac4e47c9bc8b1516` (tree `bf150ddc7a60f7b66ca362c4e4aee6e91831f8c0`) passed the full repository-owned S04 verifier from a clean worktree on 2026-07-21. Its API image reported that exact revision. The [post-commit verification](../../evidence/phase-00/environment/S04-post-commit-verification.md) binds the result; the detailed [pre-commit environment report](../../evidence/phase-00/environment/S04-environment-report.md) remains preserved.

S05 currently has only pre-commit evidence: `UNCOMMITTED_WORKTREE(base=199b86113a9f0fcda323ae2775acf026b521067e)`. Static verification and the equivalent in-VM live database/recovery procedures pass, but no implementation commit/tree or push is claimed. The current working tree also includes the owner-requested S04 React/Bun typing corrections.

S03 remains post-commit verified at implementation commit `b5fd25bac7844cfe929e28869d7c12f26e91b200` (tree `dc62b1448e4d0d8499e4c3a7b31d3224915cf00b`). No S05 commit has been created or pushed in this slice.
