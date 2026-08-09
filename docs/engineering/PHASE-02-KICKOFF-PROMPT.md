# Phase 02 clean-start prompt

Copy the complete prompt below into a fresh Codex task opened on the Atlas repository.

```text
You are taking over the Atlas repository as principal engineer, security engineer, and privacy-aware technical lead.

Your immediate mission is to begin Phase 02 cleanly by completing only `P02-S01 — canonical audit, decision inventory, and execution planning`. Do not implement Customer/KYC/privacy runtime behavior in this first slice. Do not add wallet, balance, hold, journal, transfer, payout, beneficiary, refund, payment, or other financial behavior.

Known predecessor checkpoint—verify it from Git and repository evidence before relying on it:

- Phase 01 implementation closure: `067a31607dc173f02cb628db1f380f88ab2d411f`.
- Phase 01 post-commit evidence binding: `ea901dba10821ef8ce9ff3b8f04a34f908d6261a`.
- Phase 01 closes all 30 IAM rows only at bounded synthetic local/reference depth.
- Administrator membership removal and break-glass remain fail-closed. Future privileged and financial actions remain reserved fail-closed.
- The focused history/security gate passed with no worktree/history leaks and zero called Go vulnerabilities.
- The focused supply-chain gate passed at `ea901dba10821ef8ce9ff3b8f04a34f908d6261a` through the repository-supported Podman WSL fallback: backend/web images, four SPDX surfaces, critical-CVE/license checks, revision labels, and hardened non-root/read-only/cap-drop/no-new-privileges runtime assertions.
- No real identity data, real KYC provider, production deployment, compliance, scale, availability, or financial-readiness claim exists.

Treat the current worktree, Git state, and canonical repository sources as authority. First confirm the branch, HEAD, upstream divergence, worktree cleanliness, and that the two predecessor commits are ancestors. Preserve unrelated user changes. Do not pull, push, rewrite history, or create a PR unless explicitly requested.

Read these sources before acting, in this order:

1. `AGENTS.md`
2. `docs/engineering/IMPLEMENTATION_STATUS.md`
3. `docs/engineering/CONTEXT_INDEX.md`
4. `docs/atlas-prd/02-phases/PHASE-02_CUSTOMER_KYC_PRIVACY.md` in full
5. `docs/atlas-prd/00-master/03_REQUIREMENTS_AND_QUALITY_GATES.md`
6. `docs/atlas-prd/00-master/04_ROADMAP_AND_DEPENDENCIES.md`
7. `docs/atlas-prd/01-architecture/00_SYSTEM_ARCHITECTURE.md`
8. `docs/atlas-prd/01-architecture/01_SECURITY_AND_TRUST_MODEL.md`
9. `docs/atlas-prd/01-architecture/03_API_AND_EVENT_STANDARDS.md`
10. `docs/atlas-prd/01-architecture/04_RELIABILITY_OBSERVABILITY_DR.md`
11. `docs/atlas-prd/01-architecture/05_PRIVACY_AND_REGULATORY_ALIGNMENT.md` in full
12. `docs/atlas-prd/03-contracts/openapi.yaml`, `asyncapi.yaml`, and `EVENT_CATALOG.md`, loading only relevant sections after searching by stable ID/path
13. `docs/atlas-prd/06-governance/REQUIREMENTS_TRACEABILITY.csv` for the exact Phase 02 rows
14. `docs/atlas-prd/06-governance/THREAT_REGISTER.csv`, `RISK_REGISTER.md`, accepted ADRs, `DEFINITION_OF_DONE.md`, and the current evidence index/policies

Audit facts to verify, not merely repeat:

- Phase 02 has exactly 26 requirement IDs: `CUS-001`, `CUS-002`, `CUS-003`, `CUS-004`, `CUS-005`, `CUS-030`, `CUS-031`, `CUS-032`, `CUS-033`, `KYC-001`, `KYC-002`, `KYC-003`, `KYC-004`, `KYC-005`, `KYC-006`, `KYC-007`, `KYC-008`, `PRV-010`, `PRV-011`, `PRV-012`, `PRV-013`, `PRV-014`, `PRV-020`, `PRV-021`, `PRV-022`, and `PRV-023`.
- All 26 traceability rows remain `Planned`; do not mark any Verified from plans, contracts, files, or Phase 01 evidence.
- The Phase 02 document proposes 12 HTTP operations across 11 unique paths, all currently absent from the canonical OpenAPI:
  - `GET /v1/customers/{customer_id}`
  - `PATCH /v1/customers/{customer_id}` with ETag/version
  - `GET /v1/customers/{customer_id}/contact-points`
  - `POST /v1/customers/{customer_id}/contact-change-requests`
  - `GET /v1/kyc/cases/{case_id}`
  - `POST /v1/kyc/cases`
  - `POST /v1/kyc/cases/{case_id}/submissions`
  - `POST /v1/kyc/cases/{case_id}/decisions`
  - `GET /v1/privacy/notices/current`
  - `POST /v1/privacy/acknowledgements`
  - `GET /v1/customers/{customer_id}/restrictions`
  - `POST /v1/customers/{customer_id}/restriction-requests`
- Absence is a contract/decision gap to close in P02-S02, not permission to invent endpoints now.
- Phase 02 depends on Phase 01. PostgreSQL remains authoritative; Redis is never privacy, restriction, KYC, or authorization truth. Browser checks never authorize. Audit must remain caller-transaction atomic where a synchronous state change is claimed.
- Only deterministic synthetic identity data and provider scenarios are allowed. Never store or request a real identity document, biometric, PAN, bank credential, or unrestricted raw KYC payload.

Before planning implementation, produce an explicit conflict/gap register. At minimum investigate—without silently deciding—the following:

1. Customer ownership and identity linkage: one-to-one/one-to-many relationship to Phase 01 principals, tenant scope, self versus workforce access, concealment, and lifecycle authority.
2. Exact customer and KYC transition tables, invalid/late transition recording, optimistic concurrency, resubmission, closure prerequisites, and immutable history.
3. Profile PATCH field allowlist, mass-assignment rejection, ETag/version behavior, normalization/display preservation, masking, history, and correction effects.
4. Contact verification and contact-change transport: step-up freshness, delayed high-risk-action policy, enumeration resistance, session/action re-evaluation, cancellation/expiry, and missing verification operations.
5. Deterministic KYC simulator scenarios and the missing signed callback contract: algorithm/key/version, canonical versus exact raw-byte signing rule, event identity, replay ledger, callback-before-response, duplicates, out-of-order/contradictory results, provider outage, and customer/case binding. Do not invent a callback path or event.
6. Minimal KYC evidence: submitted-field hashes, provider references, reason/factor vocabulary, assurance/tier vocabulary, checksums, restricted object references, access grants/logs, content-type controls, retention, and the deliberately absent real document/biometric surfaces.
7. Manual review authorization: support/risk roles, purpose, masking/reveal, decision reasons/notes, stored-XSS and note-redaction controls, execution-time reauthorization, and whether any restriction/review action must reuse typed maker-checker approval.
8. KYC tier semantics that reference future currencies, balances, transfers, beneficiaries, payouts, merchant access, holds, or closure. Model only ratified capability-policy facts or reserved fail-closed integration contracts; do not implement later-phase financial state.
9. Privacy notice applicability/versioning, presentation versus acceptance evidence, locale/channel, mandatory-processing lawful-basis placeholders, stale-tab behavior, granular optional preferences, withdrawal, and the missing preference operations.
10. Complete field-to-purpose/classification/source/owner/retention/masking/export inventory plus record-of-processing, DPIA, processor, cross-border, breach, and claims/disclaimer artifacts appropriate to a synthetic portfolio—not legal conclusions.
11. Retention dry-run semantics, deterministic manifests, pseudonymisation/linkage preservation, and whether a durable job/worker/outbox is required. The current worker and simulator boundaries are intentionally inert; activating a worker input, broker/event, object store authority, or new telemetry family triggers same-change Phase 00 revalidation and possibly AsyncAPI work.
12. Data-right access/correction/deletion/objection foundations and short-lived evidence/export access are required by architecture/testing but are not fully represented in the Phase 02 API list. Record the gap; do not invent the surface.
13. Restriction type/scope/source/reason/effective/expiry/status/review vocabulary, deterministic composition, removal/correction semantics, approval policy, command-time checks, concurrency, and future financial consumers.
14. Async behavior and event scope. Phase 02 currently has no ratified event addition; no event, outbox relay, worker job, or “exactly once” claim may be introduced without contract-first decision closure and at-least-once/replay/out-of-order evidence.

Create `docs/engineering/PHASE-02-PLAN.md` as the P02-S01 primary artifact. It must include:

- source hierarchy and exact audited revision/tree;
- current implementation inventory versus each of the 26 requirements;
- a contract operation/gap matrix for all 12 proposed operations;
- applicable threats including at least `THR-009`, `THR-022`, `THR-026..028`, `THR-037`, `THR-043`, `THR-045`, `THR-050`, plus any additional threats found by exact scope;
- principal risks including at least `RSK-005..007`, `RSK-009`, `RSK-014..015`, `RSK-017`, `RSK-022`, `RSK-024..025`, `RSK-027..032`, with evidence-based applicability rather than bulk copying;
- authorization, privacy/financial, data-classification, tenancy, idempotency, concurrency, before/after-commit, provider-ambiguity, recovery, observability, runbook, and evidence boundaries;
- exact unresolved decision IDs (`P02-Dxx`) with owner, options, trade-offs, source conflict, blocking scope, and required acceptance proof;
- a proposed focused-slice sequence from decision closure through persistence, product behavior, synthetic provider, privacy/retention, restrictions, frontend/operations, and final acceptance—derived from the audit rather than assumed from this prompt;
- per-slice requirements, threats, contexts, contract/schema/event changes, failure modes, test mix, Phase 00 revalidation triggers, evidence IDs, rollback/forward-fix boundary, and explicit non-goals;
- all 15 Phase 02 “tests most agents will skip,” including which are executable now and which depend on absent later-phase financial surfaces and therefore remain reserved fail-closed integration tests;
- the exact Phase 02 acceptance journey and an honest public-evidence/claims boundary.

Also add a sanitized P02-S01 audit report under `evidence/phase-02/architecture/` and an integrity-checked pre-commit evidence catalogue. Update the context index, implementation status, and evidence index only to say Phase 02 planning has begun; keep all 26 Phase 02 requirements Planned. Update the canonical PRD manifest for any canonical governance edits. Add a repository-owned `scripts/verify-p02-s01.ps1` that proves the plan/audit/catalogue integrity and kills seeded mutations such as a missing requirement, invented OpenAPI implementation claim, financial-scope leak, unsafe evidence path, stale source identity, or a falsely Verified traceability row.

Verification for P02-S01 must include at least:

- `pwsh -NoProfile -File ./scripts/verify-p01.ps1` to preserve the committed Phase 01 gate;
- `go test ./...` and all three Go entry-point builds;
- canonical contract lint and PRD manifest verification;
- Bun generated-client drift, tests, typecheck, and build;
- the new P02-S01 verifier and evidence-integrity canaries;
- `git diff --check`, focused secret scanning, and a clean staged-diff review.

Do not run live provider/database migrations merely to prove a planning-only slice unless the audit changes runtime sources. Do not claim independent review, production readiness, Nigerian legal compliance, or a security/privacy guarantee. Preserve the exact portfolio disclaimer from the privacy architecture.

When P02-S01 is complete and its evidence passes, commit it as a focused planning slice. Do not push unless explicitly asked. Report the commit, verified commands, exact unresolved blocking decisions, and the next safe slice. Do not begin P02-S02 or product implementation in the same slice.
```
