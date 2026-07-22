# Atlas implementation status

- **Status date:** 2026-07-22
- **Current phase:** [Phase 00 — Secure engineering foundation](../atlas-prd/02-phases/PHASE-00_ENGINEERING_FOUNDATION.md)
- **Current slice:** [S07 — CI, contracts, and supply-chain integrity](PHASE-00-PLAN.md#s07--ci-contracts-and-supply-chain-integrity) is locally implemented and verified from `UNCOMMITTED_WORKTREE(base=3342b4ded1cd62fab1223372cd5129f272889878)`. CI/release workflows, contract compatibility, history scanning, four SBOM surfaces, image scanning/hardening, and immutable pin policies pass locally; hosted required checks, code-owner enforcement, keyless signatures, and provenance remain unverified until GitHub runs exist.
- **Implementation state:** Feature-free engineering foundation with typed Go primitives, static policies, three operational API endpoints, a complete synthetic local dependency/process topology, strict environment configuration, deterministic fixture catalogues, three React route shells, PostgreSQL migration/role/readiness/recovery controls, closed-schema JSON logs, OTLP traces/metrics, a Phase 00 threat model, a provider-neutral secret/version boundary, and repository-owned CI/contract/supply-chain controls. No product endpoint, financial workflow, product schema, worker job, executable provider scenario, broker stream, identity exchange, managed production secret provider, or wallet UI exists.

## Repository baseline

| Area | Verified state |
|---|---|
| Version control | Valid Git repository on branch `main`; origin is `https://github.com/MichaelSeveen/atlas.git`. S05 is post-commit verified as `5ea77fcf31b349b53fcd14e14ab81a4da5da840a`; S06 is committed as `3342b4ded1cd62fab1223372cd5129f272889878`; S07 is an uncommitted worktree on that base. |
| Specification | Canonical PRD is `docs/atlas-prd/`: the 59-file validated baseline now has four accepted implementation ADRs (63 versioned files including the manifest), while the baseline report retains 399 requirements, 60 threats, 154 adversarial tests, OpenAPI 3.1.1 (33 paths/41 operations), and AsyncAPI 3.0.0 (9 channels/17 messages). S03–S07 edits preserve one canonical contract/spec root. |
| Application code | Go module `github.com/MichaelSeveen/atlas`; `cmd/api` serves only liveness, readiness, and version with typed dependency and real migration-state probes, while worker/simulator remain feature-free. `cmd/dbctl` validates the released migration inventory and `cmd/contractctl` lints/compares the canonical OpenAPI/AsyncAPI. React owns three feature-free route shells. Twelve narrow platform packages plus a feature-free contract-compatibility package, architecture/layout/toolchain policies, and focused foundation tests exist. External Go dependencies remain limited to pgx, official OpenTelemetry/OTLP modules, and the YAML parser used by the engineering contract checker. No product behavior, product schema, or generated product client exists. |
| Tooling | Go 1.25.12 with language baseline 1.25.0, pgx/v5 5.10.0, OpenTelemetry Go 1.43.0, Bun 1.3.0, and React 19.2.7 are pinned; `bun.lock` is frozen. Repository-owned S01–S07 verification now includes contract breaks, action/image/tool pin mutations, full-history secret scanning, a deleted-history canary, Govulncheck, four SPDX SBOMs, critical-CVE/license gates, and hardened image execution. GitHub Linux still must supply the hosted race, Gosec, CodeQL, required-check, signature, and provenance evidence. |
| Local environment | Compose-compatible PostgreSQL, Redis, NATS JetStream, MinIO, OTel Collector, Keycloak, API, worker, simulator, and web run in a constrained loopback-only synthetic namespace. API, worker, and simulator export bounded OTLP traces/metrics; collector availability is explicitly non-authoritative for readiness. Repository scripts now use the installed WSL `podman-compose` fallback on this host, while clean-machine one-command proof is still outstanding. |
| Verified pins | Go 1.25.12/language 1.25.0; module `github.com/MichaelSeveen/atlas`; pgx/v5 5.10.0; OpenTelemetry Go/SDK/exporters 1.43.0; Bun 1.3.0; React/React DOM 19.2.7; immutable GitHub Action SHAs; hash-verified scanner archives; and tag-plus-digest external/base images. Release signature/provenance verification is configured but not yet hosted evidence. |
| Sensitive/generated/binary scan | Gitleaks scans the complete 12-commit history with no finding; a disposable repository proves a deleted synthetic secret is still detected. No build binary, SBOM, scanner report, or OCI archive is retained outside ignored `.tmp/`; the eleven removed root PRD duplicates remain guarded against reappearance. |

## Phase 00 requirement state

| Classification | Count | Requirement IDs |
|---|---:|---|
| Satisfied | 27 | `FND-001..006`, `FND-012`, `FND-013`, `FND-021`, `FND-022`, `FND-025`, `FND-027`, `FND-030`, `FND-032`, `FND-033`, `FND-041`, `FND-043`, `FND-050..055`, `FND-060..063` |
| Partially satisfied | 10 | `FND-010`, `FND-011`, `FND-020`, `FND-023`, `FND-024`, `FND-026`, `FND-031`, `FND-040`, `FND-042`, `FND-064` |
| Absent | 0 | None. |
| Conflicting | 0 | None identified. |
| Not yet assessed | 0 | All 37 Phase 00 requirement IDs were assessed. |

”Satisfied” is requirement-scoped: S01–S07 foundation mechanics named below are verified. It does not imply clean-machine acceptance, hosted CI enforcement, signed/provenanced release publication, product database ownership, application seeds, provider behavior, identity integration, managed secret custody, complete worker/event observability, later slices, or that the Phase 00 gate passes. See the [per-requirement audit](PHASE-00-PLAN.md#requirement-by-requirement-audit).

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
- `FND-041` — API/worker/simulator runtime and bootstrap logs use a closed source-redacted JSON schema; CRLF/field injection is rejected before any sink write, and raw SDK/server diagnostics are suppressed at source.
- `FND-043` — a fixed W3C trace ID is exported through API, readiness, and database spans to the local collector; the live test also proves collector outage leaves readiness authoritative.
- `FND-050` — the Phase 00 system context, six trust boundaries, initial STRIDE analysis, and all 60 canonical threat/control/test/owner/residual links exist and pass executable coverage validation.
- `FND-051` — canonical classification rules are preserved and an executable field inventory constrains every accepted foundation log field, classification, and retention period.
- `FND-052` — a provider-neutral versioned secret boundary and documented rotation/revocation/recovery procedure reject cross-environment/purpose/algorithm use, downgrade, unavailability, and material reuse.
- `FND-053` — the API edge enforces secure headers, exact-origin CORS, fixed route/query/body/decompression limits, server deadlines, safe panic/error handling, and topology-free health responses under adversarial tests.
- `FND-055` — vulnerability-disclosure and dependency-emergency runbooks cover intake, containment, evidence, patch/rebuild/revoke, communication, and retrospective handling; the S06 report records a synthetic tabletop.
- `FND-021` — live integration commands require real PostgreSQL roles and real NATS JetStream; the database/lock/recovery scripts cannot be satisfied by mocked repositories.
- `FND-022` — the S07 supply-chain command generates and hashes backend source, frontend source, backend image, and web image SPDX SBOMs, verifies expected identities, rejects denied licenses, and scans every surface.
- `FND-025` — empty and previous-version throwaway databases reach the current schema, repeated application is idempotent, cleanup is bounded, and a long-lock migration aborts safely.
- `FND-027` — the canonical OpenAPI 3.1.1 and AsyncAPI 3.0.0 receive syntax/reference lint, baseline comparison, real-process examples, and seeded removed-path/field/message/reference failures without creating a second mutable contract.
- `FND-054` — Go/Bun/application dependencies, GitHub Actions, scanner archives, and external/base images are pinned and verified; frozen installs, Govulncheck, license/CVE gates, Dependabot schedules, and documented normal/emergency updates are executable.
- `FND-060` — distinct migration, API, worker, reporting-read, and disabled break-glass identities use distinct generated credentials; the real permission matrix proves their allowed and denied paths.
- `FND-061` — API, worker, and reporting roles cannot create, alter, or drop schema objects, grant effective public access, assume the migration role, or create disallowed temporary state.
- `FND-062` — a closed released SHA-256 manifest covers SQL and risk metadata; changed and deleted released-file canaries are killed and clean post-commit verification binds the result to `5ea77fc`.
- `FND-063` — every migration has closed lock/timeout/data/plan/space/forward-fix/rollback metadata, and the representative foundation lock canary proves bounded abort and transaction recovery.

## Active requirement IDs

- `FND-010` is partial: the complete constrained stack, readiness, restart, teardown, and smoke pass through Compose, but this host needed a Podman WSL/systemd/provider repair and the exact clean-machine wrapper command is not yet independently proven.
- `FND-011` is partial: deterministic synthetic identity/account labels and provider scenario IDs validate with fixed checksum and tenant ownership, but no application schema is loaded and no provider behavior executes.
- `FND-020` is partial: PR/static, real PostgreSQL/NATS, CodeQL, and supply-chain workflows call the repository-owned checks, but no successful hosted run or protected required-check configuration is available yet; Windows cannot supply the required Linux race/Gosec/CodeQL evidence.
- `FND-023` is partial: local backend/web images are source-labeled, digest-recorded, non-root, read-only, capability-dropped, and critical-CVE clean, while the dirty worktree prevents a final implementation revision and no registry digest has been published/promoted.
- `FND-024` is partial: the release workflow fails closed and configures digest-only keyless Cosign signatures plus GitHub build/SBOM attestations and verification, but no hosted release artifact, signature, or provenance statement exists yet.
- `FND-026` is partial: `CODEOWNERS` and static sensitive-path coverage pass, but repository files cannot prove GitHub ruleset enforcement, a code-owner approval, or bypass restrictions.
- `FND-031` is partial: four-environment credential references and generated local/test password/token fingerprints never overlap, but staging/production provisioning, rotation, restore, and secret-manager evidence do not exist.
- `FND-040` is partial: validated request/correlation/W3C trace context is exported through the API, readiness, and database spans. Worker/simulator have only build/lifecycle telemetry because no request/event/job enters them; no event propagation exists.
- `FND-042` is partial: emitted HTTP RED, database readiness/pool, and revision/build metrics have closed cardinality plus catalogued dashboards/alerts/runbooks. Queue lag and retry metrics are definition-only until a queue/job exists, and no deployed alert engine/routing proof exists.
- `FND-064` is partial: a verified physical base backup, WAL archive, isolated target-time restore, migration checksum, and pre-deletion marker pass, but the local backup volume is unencrypted and no product object/key/inbox/outbox/idempotency or financial replay state exists.

S07 is locally implemented and verified from `UNCOMMITTED_WORKTREE(base=3342b4ded1cd62fab1223372cd5129f272889878)`. The stated FND-020/FND-023/FND-024/FND-026 hosted and pre-commit limitations remain; no S08 or overall Phase 00 completion is claimed.

## Decisions and blockers

| Decision/gap | Impact | Required resolution |
|---|---|---|
| Production broker, IdP deployment, object store, and secret manager are not selected | Local/reference products are accepted only by ADR 0008; production semantics, key rotation, backup, and promotion remain blocked. | Resolve with scoped ADRs before any reference release; do not treat local NATS/Keycloak/MinIO as a production selection. |
| Hosted GitHub policy and release identity are unobserved | Versioned workflows and `CODEOWNERS` cannot prove required jobs, owner approval, ruleset bypass controls, keyless signing, or provenance publication. | Configure the `main` ruleset, run a PR and release from committed S07 source, and retain run/ruleset/signature/attestation identifiers in S08 evidence. |
| Generated product-client strategy is deferred | S07 enforces compatibility directly from the sole canonical contracts and introduces no client or product call. | Select a deterministic generated-client path before the first product API consumer; never create a second hand-edited contract. |
| Clean-host container bootstrap is unproven | Repository-owned S04–S07 commands pass on this host through explicit WSL Podman fallbacks, but a clean supported Podman/Docker machine has not independently reproduced them. | Reprove the exact documented wrapper commands from a clean supported host in S08. |
| Local backup/WAL volumes are not encrypted | S05 proves native backup/WAL/PITR mechanics but cannot satisfy the encrypted reference-environment and key-access facets of FND-064. | Select deployment/object/key controls and run the complete encrypted isolated restore/replay gate in S08. |

These are missing implementation decisions, not contradictory product semantics. No accepted ADR conflict was found.

## Known deviations

- Roadmap directories now exist, but most are intentional ownership placeholders and must not be described as implemented capability.
- The architecture decision index says `06-governance/adr/`; the real accepted-ADR directory is `06-governance/adrs/`.
- The sole mutable PRD contracts live under `docs/atlas-prd/03-contracts/`; implementation-owned publication/generation remains deferred and must not create a second hand-edited source.
- S06 telemetry covers only the flows that exist: API/readiness/database plus process build/lifecycle. No web-to-event-to-worker trace, queue lag, retry counter, or product telemetry is claimed.
- The local collector uses a detailed debug exporter solely for deterministic synthetic verification; no production telemetry backend, retention, alert engine, or routing is selected.
- S07 local image/SBOM proof uses the existing Podman WSL fallback. Syft completed with valid artifacts but emitted non-fatal Windows temporary-directory cleanup warnings; the exact clean-host lane remains S08 work.
- The Windows host has CGO disabled and Gosec 2.25.0 does not complete in a bounded local run. GitHub Linux is configured to run the race detector, full Gosec, and independent CodeQL; those lanes are not claimed until a hosted run passes.
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
- [S05 post-commit verification](../../evidence/phase-00/database/S05-post-commit-verification.md)
- [Current S06 observability/security report](../../evidence/phase-00/observability-security/S06-observability-security-report.md)
- [Current S07 CI/supply-chain report](../../evidence/phase-00/supply-chain/S07-ci-contract-supply-chain-report.md)
- [Canonical PRD cleanup report](../../evidence/phase-00/architecture/PRD-canonicalization-report.md)
- [Module boundary model](MODULE_BOUNDARIES.md)
- [Platform primitives and static policy](PLATFORM_PRIMITIVES.md)
- [HTTP foundation](HTTP_FOUNDATION.md)
- [Synthetic local environment](LOCAL_ENVIRONMENT.md)
- [Database foundation](DATABASE_FOUNDATION.md)
- [Toolchain policy](TOOLCHAIN_POLICY.md)
- [CI, contract, and supply-chain boundary](CI_SUPPLY_CHAIN.md)

## Last verified source revision

S04 implementation commit `39121a31765013ebdc51b3b0ac4e47c9bc8b1516` (tree `bf150ddc7a60f7b66ca362c4e4aee6e91831f8c0`) passed the full repository-owned S04 verifier from a clean worktree on 2026-07-21. Its API image reported that exact revision. The [post-commit verification](../../evidence/phase-00/environment/S04-post-commit-verification.md) binds the result; the detailed [pre-commit environment report](../../evidence/phase-00/environment/S04-environment-report.md) remains preserved.

S05 implementation commit `5ea77fcf31b349b53fcd14e14ab81a4da5da840a` (tree `258cd9bae960f06edf4825f527c42419753c5540`) passed the full repository-owned static S05 verifier from a clean worktree on 2026-07-21. The [post-commit verification](../../evidence/phase-00/database/S05-post-commit-verification.md) binds that result; the detailed pre-commit database/live/recovery report remains preserved with its Podman host and unencrypted-volume limitations.

S06 is committed as `3342b4ded1cd62fab1223372cd5129f272889878` (tree `36e8d4b1195ec3c8e8bf0bbfdef294f1df523005`). Its detailed live evidence remains the preserved pre-commit report based on `7a08056539de6d655086f7730d0cb8df3a9bb4c6`; no post-commit live rerun is claimed here.

S07 local/default, history/security, contract-baseline, and supply-chain verification passes from `UNCOMMITTED_WORKTREE(base=3342b4ded1cd62fab1223372cd5129f272889878)`. The result identifies the base and generated artifact digests but cannot identify a final S07 commit/tree; hosted CI/ruleset/signature/provenance evidence remains absent.
