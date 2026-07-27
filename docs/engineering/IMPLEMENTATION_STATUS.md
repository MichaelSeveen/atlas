# Atlas implementation status

- **Status date:** 2026-07-27
- **Current phase:** [Phase 01 — Identity, access, tenancy, and privileged workforce controls](../atlas-prd/02-phases/PHASE-01_IDENTITY_ACCESS_TENANCY.md)
- **Current slice:** `P01-S04` synthetic OIDC BFF and durable session lifecycle is complete at its bounded local/reference boundary. The core checkpoint is committed at `d276ad457e1ce7e3863cbd4717dfb5c432e2e29d`; the step-up checkpoint is committed at `015911fdc586b4e7b65a80d29cdf06b799e37fc4`; and `EVD-P01-S04-CLOSURE` binds the complete account-enumeration, browser navigation/storage, administrator security-revocation, additive seed, recovery, and live gate. `P01-S05` is next and has not started.
- **Implementation state:** Phase 00 remains historically complete for its bounded synthetic feature-free foundation, P01-S03 consumed the `first-product-schema`/`first-product-durable-state` triggers, and the S04 runtime exposes nine approved identity/session routes alongside the three operational routes. ADR 0015 resolves `P01-D15` additively; the owner-only revoke route remains unchanged. The browser shell now fails closed across reload/history navigation and stores only a non-sensitive tab-scoped signed-out marker. Administrator revocation is permission-, purpose-, reason-, phishing-resistant action-bound step-up-, idempotency-, authority-version-, and atomic-Audit-bound in PostgreSQL. No organization mutation, general authorization evaluator, credential, approval, event, worker job, financial workflow, managed production secret provider, or wallet UI exists.

## Repository baseline

| Area | Verified state |
|---|---|
| Version control | Valid Git repository with origin `https://github.com/MichaelSeveen/atlas.git`. PR #22 passed all five required checks and merged as `9761754709a09c96fdbb07bf1a55c39994b50e72`. Active ruleset `19577130` protects `main` with PR-only updates, five strict required contexts, conversation resolution, deletion/non-fast-forward protection, no bypass actors, and zero fabricated approvals under ADR 0012. |
| Specification | Canonical PRD is `docs/atlas-prd/`: 68 files including the manifest and 15 accepted ADRs. The corpus retains 399 requirements, 60 threats, and 154 adversarial tests. OpenAPI 3.1.1 contains 55 paths/66 operations; ADR 0015 and the additive workforce-security command close the `IAM-004` transport defect without changing the self-owned route. AsyncAPI 3.0.0 remains 9 channels/17 messages because Phase 01 deliberately adds no event. One canonical contract/spec root remains enforced by manifest and duplicate guards. |
| Application code | Go module `github.com/MichaelSeveen/atlas`; `cmd/api` serves liveness, readiness, version, and the nine approved S04 identity/session operations through Identity and Audit application boundaries, while worker/simulator remain feature-free. Identity owns typed tenant context, an immutable identity seed plus additive policy seed chain, OIDC verification, encrypted server-side login/session state, scoped step-up and revocation replay state, population lifetimes, rotation, self revocation, and workforce security revocation; Audit owns validated event/application interfaces and caller-transaction PostgreSQL recording. `cmd/dbctl` validates eight released migrations and `cmd/contractctl` lints/compares canonical contracts. No generated product client exists. |
| Tooling | Go 1.25.12 with language baseline 1.25.0, pgx/v5 5.10.0, OpenTelemetry Go 1.43.0, Bun 1.3.0, and React 19.2.7 are pinned; `bun.lock` is frozen. `verify-p01-s04.ps1` composes prior Phase 01 regression with migration/seed integrity, real PostgreSQL roles/recovery, disposable self/admin revocation concurrency and Audit-outage rollback, rebuilt synthetic Keycloak flows, a nine-sample interleaved known/absent-user timing matrix for each realm, HTTP negative matrices, browser navigation/storage proof, and bounded identity telemetry. GitHub Linux supplied the prior Phase 00 race/Gosec/CodeQL and constrained-pool evidence. |
| Local environment | Compose-compatible PostgreSQL, Redis, NATS JetStream, MinIO, OTel Collector, Keycloak, API, worker, simulator, and web run in a constrained loopback-only synthetic namespace. API, worker, and simulator export bounded OTLP traces/metrics; collector availability is explicitly non-authoritative for readiness. Local scripts use the installed WSL `podman-compose` fallback, and successful run `29964442782` independently reproduced the full one-command Docker path on a fresh GitHub-hosted runner. |
| Verified pins | Go 1.25.12/language 1.25.0; module `github.com/MichaelSeveen/atlas`; pgx/v5 5.10.0; OpenTelemetry Go/SDK/exporters 1.43.0; Bun 1.3.0; React/React DOM 19.2.7; immutable GitHub Action SHAs; hash-verified scanner archives; and tag-plus-digest external/base images. Release signatures and exact-source SLSA/SPDX attestations pass automated and independent hosted verification. |
| Sensitive/generated/binary scan | Gitleaks scans the complete history with no finding; a disposable repository proves a deleted synthetic secret is still detected. Local binaries/reports remain under ignored `.tmp/`; the hosted release retains the four sanitized SPDX surfaces for 90 days with archive and document hashes recorded in EVD-P00-S08-008. The eleven removed root PRD duplicates remain guarded against reappearance. |

## Phase 00 requirement state

| Classification | Count | Requirement IDs |
|---|---:|---|
| Satisfied | 34 | `FND-001..006`, `FND-010..013`, `FND-020..025`, `FND-027`, `FND-030..033`, `FND-041`, `FND-043`, `FND-050..055`, `FND-060..064` |
| Accepted deviation/scope decision | 3 | `FND-026` (accepted deviation); `FND-040`, `FND-042` (accepted scope decisions) |
| Partially satisfied | 0 | None. |
| Absent | 0 | None. |
| Conflicting | 0 | None identified. |
| Not yet assessed | 0 | All 37 Phase 00 requirement IDs were assessed. |

“Satisfied” is requirement- and scope-specific: S01–S08 foundation mechanics named below are verified at the stated depth. Phase 00 gate completion does not imply independent human review, product database ownership, executable product seeds, provider behavior, identity integration, managed secret custody, worker/event behavior, encrypted product-state recovery, production readiness, or any later-phase capability. ADR 0013 and the machine gate require revalidation when those surfaces appear. See the [per-requirement audit](PHASE-00-PLAN.md#requirement-by-requirement-audit).

## Phase 01 requirement state

| Classification | Count | Requirement IDs |
|---|---:|---|
| Verified | 5 | `IAM-001`, `IAM-003..005`, `IAM-007` |
| Planned | 25 | `IAM-002`, `IAM-006`, `IAM-010..015`, `IAM-020..026`, `IAM-030..034`, `IAM-040..044` |

These five rows are verified only for their explicit S04 synthetic application-session scope.
Phase 01 remains open; later organization, authorization, approval, credential, and acceptance
slices must verify their own rows without reinterpreting S04 evidence.

## Completed requirement IDs

- `FND-001` — roadmap-aligned directories, canonical-source guard, pinned Go metadata, and repository-owned verification exist.
- `FND-002` — dependency rules are documented and enforced by a clean-tree scanner plus a seeded cross-context persistence-import rejection.
- `FND-003` — API, worker, and provider-simulator Go entry points build independently; only the API has a runtime lifecycle, now serving the three operational and nine approved identity/session endpoints.
- `FND-004` — React + TypeScript is consistently selected in the PRD, with no competing frontend implementation.
- `FND-005` — bounded integer money/currency, cryptographically random opaque IDs, injectable UTC clocks, explicit actor/correlation contexts, and data-minimizing domain errors pass table/property/fuzz and mutation proof.
- `FND-006` — the architecture checker rejects seeded floating-money and direct domain wall-clock violations while permitting explicit safe controls.
- `FND-010` — successful release run `29964442782` completed the full fresh-host Docker S08 command, including the constrained topology, readiness/smoke/trace/outage, real PostgreSQL/NATS, backup/restore, hosted race, exit-zero bounded teardown, and exact clean-clone cleanup.
- `FND-011` — the original foundation catalogue and released Phase 01 identity seed v1 remain immutable; P01-S03 consumes the first-product-schema trigger, while S04 advances the canonical policy binding through checksum-bound additive seed v2 after verifying the exact v1 predecessor. Both seeds are transactional and pass upgrade, fresh, replay, backup, and restore lanes. Executable provider scenarios remain a guarded future trigger.
- `FND-012` — portfolio configuration is synthetic-only, loopback/reserved-host constrained, and rejects real/public endpoint, development-key, wildcard, and missing-synthetic canaries.
- `FND-013` — reset is limited to local/test, validates target containment, prints its resolved target, and requires the exact environment confirmation.
- `FND-020` — PR #19 run `29949126130` passed static/history/contracts, real PostgreSQL/NATS, both CodeQL languages, race/Gosec, supply-chain, and solo sensitive-declaration checks; active `main` ruleset `19577130` strictly requires the five hosted contexts with no bypass actors.
- `FND-030` — strict local, test, staging, and production-reference configurations are present and validated as one closed set.
- `FND-031` — every configured signing, encryption, database, identity, merchant, broker, and object-storage reference is environment/purpose scoped, and generated local/test credential fingerprints are distinct; managed non-local material remains trigger-bound.
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
- `FND-022` — the S07 supply-chain command generates and hashes backend source, frontend source, backend image, and web image SPDX SBOMs, verifies expected identities, rejects denied licenses, and scans every surface; run `29964442782` retained all four documents with archive and individual file hashes.
- `FND-023` — run `29964442782` published source-tagged immutable backend/web GHCR indexes after the full S08 gate; exact index and Linux/amd64 manifest digests, source revision, compressed sizes, and hardened image properties are recorded in EVD-P00-S08-008.
- `FND-024` — the successful workflow and an independent recheck verify both digest-only keyless Cosign signatures plus exact-repository/workflow/source/ref SLSA v1 and SPDX 2.3 attestations before the retained release-SBOM step.
- `FND-025` — empty and previous-version throwaway databases reach the current schema, repeated application is idempotent, cleanup is bounded, and a long-lock migration aborts safely.
- `FND-027` — the canonical OpenAPI 3.1.1 and AsyncAPI 3.0.0 receive syntax/reference lint, baseline comparison, real-process examples, and seeded removed-path/field/message/reference failures without creating a second mutable contract.
- `FND-054` — Go/Bun/application dependencies, GitHub Actions, scanner archives, and external/base images are pinned and verified; frozen installs, Govulncheck, license/CVE gates, Dependabot schedules, and documented normal/emergency updates are executable.
- `FND-060` — distinct migration, API, worker, reporting-read, and disabled break-glass identities use distinct generated credentials; the real permission matrix proves their allowed and denied paths.
- `FND-061` — API, worker, and reporting roles cannot create, alter, or drop schema objects, grant effective public access, assume the migration role, or create disallowed temporary state.
- `FND-062` — a closed released SHA-256 manifest covers SQL and risk metadata; changed and deleted released-file canaries are killed and clean post-commit verification binds the result to `5ea77fc`.
- `FND-063` — every migration has closed lock/timeout/data/plan/space/forward-fix/rollback metadata, and the representative foundation lock canary proves bounded abort and transaction recovery.
- `FND-064` — ADR 0008/0010 physical base backup, continuous WAL archive, verification, and isolated point-in-time recovery pass on the synthetic reference platform, including fresh-host S08 proof; product/deployed/encrypted recovery remains trigger-bound.

## Accepted dispositions

- `FND-026` is an accepted solo-maintainer deviation under ADR 0012. The closed policy, PR declaration, canaries, and ruleset `19577130` enforce the available compensating controls. Independent human review remains unavailable and is not claimed; it becomes blocking at any non-synthetic/data/provider/second-maintainer/production-readiness trigger.
- `FND-040` is an accepted scope decision under ADR 0013. Validated request/correlation/W3C trace context is exported across every causally reachable request boundary. The first worker/simulator input, event/consumer, or broker stream must implement and evidence propagation in the same change.
- `FND-042` is an accepted scope decision under ADR 0013. Current HTTP RED, database readiness/pool, and revision/build metrics have closed cardinality plus catalogued dashboards/alerts/runbooks. The first queue/job/retry source or deployed alert backend must add emission/routing and failure evidence in the same change.

S08 static/live/history/supply acceptance is preserved in EVD-P00-S08-001 through EVD-P00-S08-005. EVD-P00-S08-006 and EVD-P00-S08-007 retain the two material fail-closed corrections. EVD-P00-S08-008 binds the successful fresh-host release, immutable digests, automated and independent signature/SLSA/SPDX verification, and retained four-surface SBOM artifact. EVD-P00-GATE-001 binds the final six-row disposition and fail-closed topology policy. Phase 00 completion is claimed only for the synthetic feature-free foundation scope.

## Decisions and future triggers

| Decision/gap | Impact | Required resolution |
|---|---|---|
| Production broker, IdP deployment, object store, and secret manager are not selected | Local/reference products are accepted only by ADR 0008; production semantics, key rotation, backup, and promotion remain blocked. | Resolve with scoped ADRs before any reference release; do not treat local NATS/Keycloak/MinIO as a production selection. |
| Independent code-owner approval remains trigger-bound | Active ruleset `19577130` requires the passing PR gates and successful hosted release identity is recorded, but ADR 0012 does not represent owner self-review as organizational separation. | Keep the ruleset active and obtain genuine qualified approval before any real-data/provider/deployment/second-maintainer/production-readiness trigger. |
| Generated product-client strategy is deferred | S07 enforces compatibility directly from the sole canonical contracts and introduces no client or product call. | Select a deterministic generated-client path before the first product API consumer; never create a second hand-edited contract. |
| Local backup/WAL volumes are not encrypted | P01-S04 revalidates synthetic Identity/Audit rows, checksums, grants, and revoked application-session authority through isolated PITR only. | Before a reference deployment or backup encryption/key custody, select deployment/object/key controls and run the complete encrypted isolated restore/replay gate. |
| `P01-D15`: administrator session-revocation transport | Resolved by ADR 0015 and `POST /v1/security/sessions/{session_id}/revocations`; the self-owned delete operation remains unchanged. | Preserve the closed workforce permission/purpose/reason/action-bound-step-up/idempotency/decision/Audit semantics and rerun migration/recovery evidence on every change. |

The first four rows are missing implementation decisions rather than contradictory product
semantics. ADR 0015 resolves the implementation-discovered `P01-D15` defect additively and retains
the owner-only operation’s original scope.

## Known deviations

- Roadmap directories now exist, but most are intentional ownership placeholders and must not be described as implemented capability.
- The architecture decision index says `06-governance/adr/`; the real accepted-ADR directory is `06-governance/adrs/`.
- The sole mutable PRD contracts live under `docs/atlas-prd/03-contracts/`; implementation-owned publication/generation remains deferred and must not create a second hand-edited source.
- ADR 0012 permits synthetic solo-maintainer progress with required hosted checks and sensitive-change attestations. It does not represent self-review as independent approval and cannot cross its real-data/provider/deployment/production-claim triggers.
- Telemetry covers the flows that exist: API/readiness/database, process build/lifecycle, and bounded S04 identity/provider operations. No web-to-event-to-worker trace, queue lag, retry counter, authorization-decision metric, or financial telemetry is claimed.
- The local collector uses a detailed debug exporter solely for deterministic synthetic verification; no production telemetry backend, retention, alert engine, or routing is selected.
- S07 local image/SBOM proof uses the existing Podman WSL fallback. Syft completed with valid artifacts but emitted non-fatal Windows temporary-directory cleanup warnings; EVD-P00-S08-008 separately proves the full clean hosted Docker path.
- The Windows host has CGO disabled and Gosec 2.25.0 does not complete in a bounded local run. GitHub Linux run `29943586545` supplied S08 race/Gosec/CodeQL proof, including `s08_named_skipped_test_10=PASS`.
- EVD-P00-S08-006 retains attempt 1's cleanup failure and zero publication; EVD-P00-S08-007 retains attempt 2's missing-token verification failure after partial publication; EVD-P00-S08-008 records the corrected green release, including Bun exit `0` in `213ms`.
- The Phase 00 seed catalogue remains historical. P01-S03/S04 use a separate immutable synthetic Identity/Audit seed, an additive policy-binding seed, and runtime-only Keycloak users; they are not real identities or a production IdP selection.
- The S04 closure proves login/current/session-list/logout, revoke-one/all mechanics, durable exact step-up replay/conflict, stale-claim rejection, injected-provider outage behavior, live customer/merchant higher-assurance rotation with old-cookie rejection, bounded three-population known/absent-user response/timing differentials, real-browser signed-out reload/history/direct-route protection, and audit-atomic workforce administrator revocation. The browser remains a synthetic shell detached from product APIs, Keycloak evidence is not a real-provider or MFA claim, and no production recovery/rate-control posture is claimed.
- Project policy forbids React class components. Because React 19 has no function-component error-boundary API, the feature-free web shell now uses a route-aware root `onUncaughtError` fallback; strict subtree-local error containment is not claimed.
- The PRD validation report proves planning-pack consistency only; it is not implementation, security, performance, recovery, or compliance evidence.

## Evidence links

- [PRD pack validation report](../atlas-prd/PACK_VALIDATION_REPORT.md)
- [PRD integrity manifest](../atlas-prd/MANIFEST.sha256)
- [Requirements traceability matrix](../atlas-prd/06-governance/REQUIREMENTS_TRACEABILITY.csv)
- [Threat register](../atlas-prd/06-governance/THREAT_REGISTER.csv)
- [Phase 00 audit and execution plan](PHASE-00-PLAN.md)
- [Phase 01 audit and execution plan](PHASE-01-PLAN.md)
- [Phase 01 S02 contract/decision evidence](../../evidence/phase-01/architecture/S02-contract-and-security-decisions.md)
- [Phase 01 S03 persistence/recovery evidence](../../evidence/phase-01/persistence/S03-identity-audit-persistence-report.md)
- [Phase 01 S04 identity/session core evidence](../../evidence/phase-01/identity-session/S04-identity-session-core-report.md)
- [Phase 01 S04 post-commit verification](../../evidence/phase-01/identity-session/S04-post-commit-verification.md)
- [Phase 01 S02 pre-commit evidence catalogue](../../evidence/phase-01/architecture/P01-evidence-catalogue-precommit.json)
- [Phase 01 S03 pre-commit evidence catalogue](../../evidence/phase-01/persistence/P01-S03-evidence-catalogue-precommit.json)
- [Phase 01 S04 pre-commit evidence catalogue](../../evidence/phase-01/identity-session/P01-S04-evidence-catalogue-precommit.json)
- [Phase 01 S04 step-up idempotency and assurance evidence](../../evidence/phase-01/identity-session/S04-step-up-idempotency-and-assurance-precommit.md)
- [Phase 01 S04 step-up evidence catalogue](../../evidence/phase-01/identity-session/P01-S04-evidence-catalogue-step-up-precommit.json)
- [Phase 01 S04 account-enumeration evidence](../../evidence/phase-01/identity-session/S04-account-enumeration-precommit.md)
- [Phase 01 S04 closure evidence](../../evidence/phase-01/identity-session/S04-closure-precommit.md)
- [Phase 01 S04 closure catalogue](../../evidence/phase-01/identity-session/P01-S04-evidence-catalogue-closure-precommit.json)
- [Phase 01 S04 step-up post-commit verification](../../evidence/phase-01/identity-session/S04-step-up-idempotency-and-assurance-post-commit.md)
- [Phase 01 S04 step-up post-commit catalogue](../../evidence/phase-01/identity-session/P01-S04-evidence-catalogue-step-up-postcommit.json)
- [ADR 0014 identity/access contract boundary](../atlas-prd/06-governance/adrs/0014-phase-01-identity-access-contract-boundary.md)
- [ADR 0015 administrator session-revocation boundary](../atlas-prd/06-governance/adrs/0015-admin-session-revocation-boundary.md)
- [Machine-readable identity/access policy](../atlas-prd/03-contracts/identity-access-policy.json)
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
- [Successful S08 hosted release](../../evidence/phase-00/acceptance/S08-hosted-release-success.md)
- [Known limitations after hosted release](../../evidence/phase-00/acceptance/S08-known-limitations-hosted-release.md)
- [Phase 00 gate-closure evidence](../../evidence/phase-00/acceptance/Phase-00-gate-closure.md)
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

S07 PR head `747cc80f058d570851f64592c0eb3a9ca0e33adc` passed all five hosted jobs in run `29928153984` and merged to `main` as `f6ad53553e739ea44718cc1336920a37c3fd05bc`. Registry promotion, signature, and provenance were absent at that revision and are later closed by EVD-P00-S08-008.

S08 implementation revision `6b09b4abfec050d6cdceb98af01f12bf0cab03af` (tree `027b3dfe3855d89447d2a4b6bb598dfecfef8aeb`) preserves the local full/static/history/supply/live proof. Evidence revision `431821f364165055d7e7ca7d69f047e860ee66aa` (tree `aa2412782836554b76c36bfbf1a83d46b4817156`) passed exact detached same-host clean-clone verification with empty clone-local caches. Local CGO-disabled execution cannot run the required S08 race proof; the hosted Linux run below supplies it.

S08 PR #19 run `29943586545` passed all five hosted jobs against `10ed35b8d86a68d821c89f69822289f5ab655aa8`; the real PostgreSQL lane recorded `s08_constrained_pool_connections=1`, `s08_constrained_pool_race=PASS`, and `s08_named_skipped_test_10=PASS`.

ADR 0012 implementation `08762a3e1333043d021264a875b8e5e222e9c34c` and catalogue revision `8c1032333356fe2d10b91ab46328f0a187290024` pass local S07/S08 static gates. Hosted run `29949126130` passed the sensitive declaration plus all five jobs at that exact head; active `main` ruleset `19577130` returns no bypass actors and `current_user_can_bypass=never`.

Release correction PR #22 passed all five protected checks at `88ebaca6baa8f92dd3ecc584042eb723d6abe0fe` and merged as `9761754709a09c96fdbb07bf1a55c39994b50e72`. Authorized run `29964442782` then completed the full preflight and every publication, signing, attestation-verification, retention, and cleanup step successfully; EVD-P00-S08-008 records the exact identities and independent recheck.

Phase 00 gate-closure implementation revision `188578b96e5b2fe5dab27930a9e2e66f20d2ca12` adds ADR 0013 and the fail-closed topology policy. Evidence-binding revision `4859bb54a4b510f73889ab2e4442c624988940c4` passed clean static S08 on 2026-07-23, including the evidence tamper/stale-source canaries and all four new requirement/trigger/digest/directory mutations. Live/release surfaces remain bound to unchanged hosted run `29964442782`.
