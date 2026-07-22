# Atlas implementation status

- **Status date:** 2026-07-22
- **Current phase:** [Phase 00 — Secure engineering foundation](../atlas-prd/02-phases/PHASE-00_ENGINEERING_FOUNDATION.md)
- **Current slice:** [S08 — Phase 00 acceptance, restore, and evidence release](PHASE-00-PLAN.md#s08--phase-00-acceptance-restore-and-evidence-release) is implemented as `6b09b4abfec050d6cdceb98af01f12bf0cab03af`, post-commit/clean-clone verified at `431821f364165055d7e7ca7d69f047e860ee66aa`, and extended by ADR 0012 solo-governance implementation `08762a3e1333043d021264a875b8e5e222e9c34c`. PR #19 run `29949126130` passed the sensitive declaration and all five hosted jobs at `8c1032333356fe2d10b91ab46328f0a187290024`; active `main` ruleset `19577130` requires those checks with no bypass actors. Independent review is unavailable and not claimed under the synthetic-only deviation. Independent clean machine, registry promotion, signing, and provenance remain open, so Phase 00 completion is not claimed.
- **Implementation state:** Feature-free engineering foundation with typed Go primitives, static policies, three operational API endpoints, a complete synthetic local dependency/process topology, strict environment configuration, deterministic fixture catalogues, three React route shells, PostgreSQL migration/role/readiness/recovery controls, closed-schema JSON logs, OTLP traces/metrics, a Phase 00 threat model, a provider-neutral secret/version boundary, and repository-owned CI/contract/supply-chain controls. No product endpoint, financial workflow, product schema, worker job, executable provider scenario, broker stream, identity exchange, managed production secret provider, or wallet UI exists.

## Repository baseline

| Area | Verified state |
|---|---|
| Version control | Valid Git repository with origin `https://github.com/MichaelSeveen/atlas.git`. S07 PR #1 merged as `f6ad53553e739ea44718cc1336920a37c3fd05bc`; PR #19 is green at `8c1032333356fe2d10b91ab46328f0a187290024`. Active ruleset `19577130` protects `main` with PR-only updates, five strict required contexts, conversation resolution, deletion/non-fast-forward protection, no bypass actors, and zero fabricated approvals under ADR 0012. |
| Specification | Canonical PRD is `docs/atlas-prd/`: the 59-file validated baseline now has five accepted implementation ADRs (64 versioned files including the manifest), while the baseline report retains 399 requirements, 60 threats, 154 adversarial tests, OpenAPI 3.1.1 (33 paths/41 operations), and AsyncAPI 3.0.0 (9 channels/17 messages). S03–S08 edits preserve one canonical contract/spec root. |
| Application code | Go module `github.com/MichaelSeveen/atlas`; `cmd/api` serves only liveness, readiness, and version with typed dependency and real migration-state probes, while worker/simulator remain feature-free. `cmd/dbctl` validates the released migration inventory and `cmd/contractctl` lints/compares the canonical OpenAPI/AsyncAPI. React owns three feature-free route shells. Twelve narrow platform packages plus a feature-free contract-compatibility package, architecture/layout/toolchain policies, and focused foundation tests exist. External Go dependencies remain limited to pgx, official OpenTelemetry/OTLP modules, and the YAML parser used by the engineering contract checker. No product behavior, product schema, or generated product client exists. |
| Tooling | Go 1.25.12 with language baseline 1.25.0, pgx/v5 5.10.0, OpenTelemetry Go 1.43.0, Bun 1.3.0, and React 19.2.7 are pinned; `bun.lock` is frozen. Repository-owned S01–S08 verification includes contract/action/image/tool/evidence mutations, solo sensitive-path/declaration canaries, complete-history scanning, Govulncheck, four SPDX SBOMs, critical-CVE/license gates, hardened image execution, and constrained-pool integration. GitHub Linux supplied race/Gosec/CodeQL evidence, including S08 skipped test #10 with a one-connection real PostgreSQL pool. |
| Local environment | Compose-compatible PostgreSQL, Redis, NATS JetStream, MinIO, OTel Collector, Keycloak, API, worker, simulator, and web run in a constrained loopback-only synthetic namespace. API, worker, and simulator export bounded OTLP traces/metrics; collector availability is explicitly non-authoritative for readiness. Repository scripts now use the installed WSL `podman-compose` fallback on this host, while clean-machine one-command proof is still outstanding. |
| Verified pins | Go 1.25.12/language 1.25.0; module `github.com/MichaelSeveen/atlas`; pgx/v5 5.10.0; OpenTelemetry Go/SDK/exporters 1.43.0; Bun 1.3.0; React/React DOM 19.2.7; immutable GitHub Action SHAs; hash-verified scanner archives; and tag-plus-digest external/base images. Release signature/provenance verification is configured but not yet hosted evidence. |
| Sensitive/generated/binary scan | Gitleaks scans the complete history with no finding; a disposable repository proves a deleted synthetic secret is still detected. No build binary, SBOM, scanner report, or OCI archive is retained outside ignored `.tmp/`; the eleven removed root PRD duplicates remain guarded against reappearance. |

## Phase 00 requirement state

| Classification | Count | Requirement IDs |
|---|---:|---|
| Satisfied | 28 | `FND-001..006`, `FND-012`, `FND-013`, `FND-020..022`, `FND-025`, `FND-027`, `FND-030`, `FND-032`, `FND-033`, `FND-041`, `FND-043`, `FND-050..055`, `FND-060..063` |
| Partially satisfied | 9 | `FND-010`, `FND-011`, `FND-023`, `FND-024`, `FND-026`, `FND-031`, `FND-040`, `FND-042`, `FND-064` |
| Absent | 0 | None. |
| Conflicting | 0 | None identified. |
| Not yet assessed | 0 | All 37 Phase 00 requirement IDs were assessed. |

”Satisfied” is requirement-scoped: S01–S08 foundation mechanics named below are verified at the stated depth. It does not imply independent human review, clean-machine acceptance, signed/provenanced release publication, product database ownership, application seeds, provider behavior, identity integration, managed secret custody, complete worker/event observability, later phases, or that the Phase 00 gate passes. See the [per-requirement audit](PHASE-00-PLAN.md#requirement-by-requirement-audit).

## Completed requirement IDs

- `FND-001` — roadmap-aligned directories, canonical-source guard, pinned Go metadata, and repository-owned verification exist.
- `FND-002` — dependency rules are documented and enforced by a clean-tree scanner plus a seeded cross-context persistence-import rejection.
- `FND-003` — API, worker, and provider-simulator Go entry points build independently; only the API has a runtime lifecycle and the three contract-defined operational endpoints.
- `FND-004` — React + TypeScript is consistently selected in the PRD, with no competing frontend implementation.
- `FND-005` — bounded integer money/currency, cryptographically random opaque IDs, injectable UTC clocks, explicit actor/correlation contexts, and data-minimizing domain errors pass table/property/fuzz and mutation proof.
- `FND-006` — the architecture checker rejects seeded floating-money and direct domain wall-clock violations while permitting explicit safe controls.
- `FND-012` — portfolio configuration is synthetic-only, loopback/reserved-host constrained, and rejects real/public endpoint, development-key, wildcard, and missing-synthetic canaries.
- `FND-013` — reset is limited to local/test, validates target containment, prints its resolved target, and requires the exact environment confirmation.
- `FND-020` — PR #19 run `29949126130` passed static/history/contracts, real PostgreSQL/NATS, both CodeQL languages, race/Gosec, supply-chain, and solo sensitive-declaration checks; active `main` ruleset `19577130` strictly requires the five hosted contexts with no bypass actors.
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
- `FND-023` is partial: committed S07 backend/web image mechanics are source-labeled, digest-recorded, non-root, read-only, capability-dropped, and critical-CVE clean, but no GHCR digest has been published and promoted through a release.
- `FND-024` is partial: the release workflow fails closed and configures digest-only keyless Cosign signatures plus GitHub build/SBOM attestations and verification, but no hosted release artifact, signature, or provenance statement exists yet.
- `FND-026` is partial under an accepted solo-maintainer deviation: ADR 0012, the closed policy, PR declaration, canaries, and ruleset `19577130` enforce the available compensating controls. Independent human review remains unavailable and is not claimed; it becomes blocking at any non-synthetic/data/provider/second-maintainer/production-readiness trigger.
- `FND-031` is partial: four-environment credential references and generated local/test password/token fingerprints never overlap, but staging/production provisioning, rotation, restore, and secret-manager evidence do not exist.
- `FND-040` is partial: validated request/correlation/W3C trace context is exported through the API, readiness, and database spans. Worker/simulator have only build/lifecycle telemetry because no request/event/job enters them; no event propagation exists.
- `FND-042` is partial: emitted HTTP RED, database readiness/pool, and revision/build metrics have closed cardinality plus catalogued dashboards/alerts/runbooks. Queue lag and retry metrics are definition-only until a queue/job exists, and no deployed alert engine/routing proof exists.
- `FND-064` is partial: a verified physical base backup, WAL archive, isolated target-time restore, migration checksum, and pre-deletion marker pass, but the local backup volume is unencrypted and no product object/key/inbox/outbox/idempotency or financial replay state exists.

S08 static/live/history/supply acceptance is preserved in EVD-P00-S08-001, committed static/clean-clone acceptance passes in EVD-P00-S08-002, hosted race execution passes in EVD-P00-S08-003, and solo-governance/ruleset proof passes in EVD-P00-S08-004. The catalogue rejects content, stale source, and descendant code/config drift. Independent-machine, independent-review, registry, signature, and provenance proof remain absent; overall Phase 00 completion is not claimed.

## Decisions and blockers

| Decision/gap | Impact | Required resolution |
|---|---|---|
| Production broker, IdP deployment, object store, and secret manager are not selected | Local/reference products are accepted only by ADR 0008; production semantics, key rotation, backup, and promotion remain blocked. | Resolve with scoped ADRs before any reference release; do not treat local NATS/Keycloak/MinIO as a production selection. |
| Hosted release identity remains incomplete | Active ruleset `19577130` now requires the passing PR gates. Independent review is an accepted synthetic-only deviation under ADR 0012, while no release has published registry/signature/provenance evidence. | Keep the ruleset active; obtain genuine independent review before any ADR 0012 trigger; separately authorize and run the fail-closed release, retaining registry/signature/attestation identifiers. |
| Generated product-client strategy is deferred | S07 enforces compatibility directly from the sole canonical contracts and introduces no client or product call. | Select a deterministic generated-client path before the first product API consumer; never create a second hand-edited contract. |
| Independent clean-host container bootstrap is unproven | The exact committed revision passes a detached same-host clone with empty clone-local dependency/build caches, but no separate administered host has run the full container gate. | Reprove the exact full S08 command from a separate clean supported host. |
| Local backup/WAL volumes are not encrypted | S05 proves native backup/WAL/PITR mechanics but cannot satisfy the encrypted reference-environment and key-access facets of FND-064. | Select deployment/object/key controls and run the complete encrypted isolated restore/replay gate in S08. |

These are missing implementation decisions, not contradictory product semantics. No accepted ADR conflict was found.

## Known deviations

- Roadmap directories now exist, but most are intentional ownership placeholders and must not be described as implemented capability.
- The architecture decision index says `06-governance/adr/`; the real accepted-ADR directory is `06-governance/adrs/`.
- The sole mutable PRD contracts live under `docs/atlas-prd/03-contracts/`; implementation-owned publication/generation remains deferred and must not create a second hand-edited source.
- ADR 0012 permits synthetic solo-maintainer progress with required hosted checks and sensitive-change attestations. It does not represent self-review as independent approval and cannot cross its real-data/provider/deployment/production-claim triggers.
- S06 telemetry covers only the flows that exist: API/readiness/database plus process build/lifecycle. No web-to-event-to-worker trace, queue lag, retry counter, or product telemetry is claimed.
- The local collector uses a detailed debug exporter solely for deterministic synthetic verification; no production telemetry backend, retention, alert engine, or routing is selected.
- S07 local image/SBOM proof uses the existing Podman WSL fallback. Syft completed with valid artifacts but emitted non-fatal Windows temporary-directory cleanup warnings; independent clean-host proof remains open.
- The Windows host has CGO disabled and Gosec 2.25.0 does not complete in a bounded local run. GitHub Linux run `29943586545` supplied S08 race/Gosec/CodeQL proof, including `s08_named_skipped_test_10=PASS`.
- S08 teardown completed and removed the isolated services/networks; Podman waited ten seconds for the stateless Bun web container and then used SIGKILL, so graceful web termination is not claimed.
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
- [S07 hosted PR verification](../../evidence/phase-00/supply-chain/S07-hosted-pr-verification.md)
- [S08 acceptance report](../../evidence/phase-00/acceptance/S08-phase-00-acceptance-report.md)
- [Phase 00 acceptance procedure](PHASE-00-ACCEPTANCE.md)
- [Phase 00 known limitations](PHASE-00-KNOWN-LIMITATIONS.md)
- [Post-commit Phase 00 known limitations](../../evidence/phase-00/acceptance/S08-known-limitations-postcommit.md)
- [S08 post-commit verification](../../evidence/phase-00/acceptance/S08-post-commit-verification.md)
- [S08 hosted PR verification](../../evidence/phase-00/acceptance/S08-hosted-pr-verification.md)
- [S08 solo-maintainer governance and ruleset verification](../../evidence/phase-00/acceptance/S08-solo-maintainer-governance.md)
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

S07 PR head `747cc80f058d570851f64592c0eb3a9ca0e33adc` passed all five hosted jobs in run `29928153984` and merged to `main` as `f6ad53553e739ea44718cc1336920a37c3fd05bc`. Registry promotion, signature, and provenance remain absent.

S08 implementation revision `6b09b4abfec050d6cdceb98af01f12bf0cab03af` (tree `027b3dfe3855d89447d2a4b6bb598dfecfef8aeb`) preserves the local full/static/history/supply/live proof. Evidence revision `431821f364165055d7e7ca7d69f047e860ee66aa` (tree `aa2412782836554b76c36bfbf1a83d46b4817156`) passed exact detached same-host clean-clone verification with empty clone-local caches. Local CGO-disabled execution cannot run the required S08 race proof; the hosted Linux run below supplies it.

S08 PR #19 run `29943586545` passed all five hosted jobs against `10ed35b8d86a68d821c89f69822289f5ab655aa8`; the real PostgreSQL lane recorded `s08_constrained_pool_connections=1`, `s08_constrained_pool_race=PASS`, and `s08_named_skipped_test_10=PASS`.

ADR 0012 implementation `08762a3e1333043d021264a875b8e5e222e9c34c` and catalogue revision `8c1032333356fe2d10b91ab46328f0a187290024` pass local S07/S08 static gates. Hosted run `29949126130` passed the sensitive declaration plus all five jobs at that exact head; active `main` ruleset `19577130` returns no bypass actors and `current_user_can_bypass=never`.
