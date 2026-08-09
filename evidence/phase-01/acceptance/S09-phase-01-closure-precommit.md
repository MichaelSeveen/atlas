# P01-S09 Phase 01 closure evidence

- Evidence ID: `EVD-P01-S09-CLOSURE`
- Phase/slice: `PHASE-01_IDENTITY_ACCESS_TENANCY` / `P01-S09`
- Source identity: `UNCOMMITTED_WORKTREE(base=b4a84a6b2fea5d01f267ee082ca978f161636f7d)`
- Evidence time: `2026-08-09T00:00:00Z`
- Environment: source-controlled synthetic `local`; Podman/WSL; PostgreSQL, Redis, NATS, MinIO, OTel Collector, Keycloak, API, worker, simulator, web; installed Microsoft Edge used headlessly through pinned `@playwright/cli` `0.1.18`
- Seed: `atlas-phase01-acceptance-personas-v5`; virtual time `2026-08-09T00:00:00Z`
- Requirements: `IAM-001..007`, `IAM-010..015`, `IAM-020..026`, `IAM-030..034`, `IAM-040..044`
- Threats: `THR-005..008`, `THR-018`, `THR-020`, `THR-023..024`, `THR-028`, `THR-037..041`, `THR-044`, `THR-053`, `THR-056..060`
- Adversarial tests: `ADV-IAM-001..015` and all fifteen Phase 01 tests-most-agents-skip

## Expected result

All thirty Phase 01 IAM rows have exact bounded evidence. Customer, merchant, support, risk, finance, and merchant-developer journeys preserve population isolation, server-authoritative authorization, opaque sessions, action-bound step-up, idempotency, synchronous Audit atomicity, deterministic credential lifecycle, recovery, and fail-closed absent surfaces. No financial behavior is introduced. Administrator membership removal and break-glass remain disabled because their policies are not ratified.

## Observed result

- `scripts/verify-p01.ps1` static acceptance passed: cumulative S04-S08 regressions, generated OpenAPI TypeScript drift check, 13 Bun tests/41 expectations, typecheck/build, all Go tests/builds, contract lint, operational catalogue mutation canaries, four runbook exercises, and the exact thirty-row closure policy.
- `scripts/verify-s08.ps1 -Live -HistoricalEvidence -ContainerRuntime podman` passed the inherited full local stack, real PostgreSQL/NATS, backup/WAL/isolated PITR restore, bounded telemetry outage, constrained-pool, and clean shutdown checks while verifying Phase 00 evidence as historical read-only.
- `scripts/p01-s04.ps1 -ContainerRuntime podman` passed from an empty explicitly named local synthetic provider/data state: real PostgreSQL recovery/session repositories, six deterministic Keycloak subjects, three-realm account-enumeration bounds, customer and merchant higher-assurance rotation/logout, and workforce baseline denial.
- `scripts/test-p01-s09-personas.ps1` passed separate support, risk, and finance workforce identities; baseline assurance was denied and no session was issued.
- `scripts/test-p01-s09-keycloak-browser.ps1 -Browser msedge` passed real synthetic Keycloak login, absence of browser-readable tokens, logout, and denied back-navigation restoration. The final diagnostic returned to the parameter-free signed-out page.
- `scripts/test-p01-s09-browser.ps1 -Browser msedge` passed generated-contract rendering, inert hostile server text, route/population isolation, visible fail-closed administrator removal, one-time credential secret memory-only display and clearing, CSRF/purpose/idempotency headers, cross-tab logout, and post-logout history restoration.
- `docs/engineering/phase-01-operational-exercises.json` records passing synthetic IdP-outage, authorization-incident, Audit-outage, and credential-compromise exercises. `deploy/observability/catalog.json` retains closed-cardinality signals; break-glass telemetry is definition-only and activation-gated because the capability is disabled.

The optional combined `-Live -History -SupplyChain` aggregate did not return a verdict within its one-hour wrapper bound. Inspection showed it was still inside the inherited S07 Podman backend-image build; the exact orphaned verifier/build process tree was stopped after the timeout. This is recorded as `INCONCLUSIVE(timeout-during-image-build)`, not PASS or a product failure. The mandatory closure claim relies on the successful static aggregate plus the separately successful full-stack/PITR, empty-provider identity, persona, and browser gates above. A subsequent focused `scripts/verify-s07.ps1 -History` passed worktree and 78-commit Gitleaks scans, the deleted-history-secret canary, and Govulncheck with zero called vulnerabilities. The supply-chain/image lane remains required after push.

## Material failure retained and correction

The first S09 live identity run correctly failed because `p01-s04.ps1` stopped the stack without deleting the persistent Keycloak data volume. Keycloak therefore skipped the updated realm import and generated a non-deterministic subject for a newly configured workforce user. The verifier now invokes the existing exact-confirmation `Reset` action for only the named Atlas `local` synthetic environment before rebuilding. The corrected empty-provider run proved all six checked-in subjects and passed the complete journey. No production or user data was in scope.

## Decision and claim boundary

- `IAM-002` is Verified for the exact login, step-up, privilege-change, and tenant-switch rotation boundaries implemented in Phase 01.
- `IAM-006` is Verified for implemented merchant-secret creation and approval-decision actions with 300-second freshness and commit-time reauthorization. Future payout, beneficiary, contact-change, refund, restriction-removal, and export actions remain reserved and fail closed.
- `P01-D13` selects generated immutable TypeScript shapes from the sole canonical OpenAPI document; the handwritten client owns transport mechanics only.
- The phase is complete only at synthetic local/reference evidence depth. No real IdP/data, production, compliance, scale, availability, independent-review, wallet, money-movement, or financial-readiness claim is made.

## Reproduction

```powershell
pwsh -NoProfile -File ./scripts/verify-p01.ps1
pwsh -NoProfile -File ./scripts/verify-p01.ps1 -Live -ContainerRuntime podman -Browser msedge
```

Use `-History` and `-SupplyChain` for the hosted-equivalent inherited security and image gates when the required network/container tooling is available.

## Sanitization and retention

This report contains only public source identities, closed synthetic persona/action/status names, bounded aggregate results, stable requirement/threat/test IDs, and limitation statements. Passwords, authorization codes, cookies, CSRF/idempotency values, credential secrets/verifiers, database/Redis URLs or credentials, network identifiers, raw traces, screenshots, console logs, WAL/data pages, and real identity data are excluded. Raw Playwright output under ignored `output/playwright/` is diagnostic only and is not evidence or committed content.

Revalidate by `2026-11-09`, or immediately after any identity contract, realm, seed, session/authorization/approval/credential policy, dependency, browser, migration/recovery, observability, or environment change.
