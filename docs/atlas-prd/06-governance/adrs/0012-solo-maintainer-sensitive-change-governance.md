# ADR 0012 — Solo-maintainer sensitive-change governance

- **Status:** Accepted
- **Date:** 2026-07-22
- **Owners:** MichaelSeveen as product, platform, and security owner
- **Related requirements/threats:** FND-020; FND-026; FND-054; RSK-018; RSK-028; RSK-031; THR-007; THR-013; THR-014; THR-018; THR-020; THR-040; THR-060
- **Supersedes/superseded by:** Amends the review-enforcement portion of ADR 0011 while solo synthetic mode is active; supersede before non-synthetic use

## Context

Atlas is currently built and operated by one person as a synthetic portfolio system. Requiring a second qualified code owner for every sensitive pull request assumes a staffed team that does not exist. Using a second GitHub account controlled by the same person would satisfy an account-level mechanism without adding independent judgment and would create misleading evidence.

The constraint affects source-change review only. It does not weaken future maker-checker, workforce separation, approval, custody, or audit requirements inside Atlas. No product endpoint, financial state, real identity, real provider, or production environment is authorized by this decision.

## Decision drivers

- Preserve honest evidence and avoid manufactured independence.
- Keep synthetic portfolio development moving without weakening financial invariants.
- Make sensitive changes more deliberate and mechanically reviewable.
- Retain a clear trigger for qualified independent review before real-world use or production-readiness claims.
- Avoid granting a work account or sock-puppet identity access merely to turn a hosted control green.

## Options considered

### Require a qualified second human for every sensitive pull request

This supplies real separation of duties but is not presently available to the project. Treating it as an immediate prerequisite would stop synthetic portfolio development without retiring a proportionate current exposure.

### Use another account controlled by the owner

GitHub may treat the account as a separate reviewer, but the same person controls authorship and approval. This is rejected because it provides no independent review and would make the evidence misleading.

### Remove review controls entirely

This reduces friction but leaves sensitive changes without a versioned risk declaration, fresh-context review, or enforced hosted checks. It is rejected.

### Scoped solo-maintainer mode with compensating controls

This preserves pull requests, required hosted tests, sensitive-path detection, a closed review checklist, synthetic-only restrictions, revision-bound evidence, and explicit revalidation triggers. It does not claim equivalence to independent human review. This option is selected.

## Decision

While `.github/solo-maintainer-policy.json` is active, Atlas uses a scoped `solo-maintainer-synthetic-portfolio` mode:

1. `main` changes use pull requests and the configured hosted status checks. Direct pushes, force pushes, branch deletion, and unrecorded bypass are prohibited by repository rules.
2. A repository-owned gate identifies changes to ledger, authorization, identity, secrets, migrations, contracts, CI, deployment, supply-chain, and phase-verification paths.
3. A sensitive pull request must contain the six checked attestations defined by the closed policy and pull-request template. The final attestation records fresh-context self-review and expressly does not claim independent human approval.
4. A 24-hour cooling-off period is recommended before merging changes that introduce or alter sensitive financial semantics. Phase 00 remains feature-free, so the current governance-only change does not introduce such semantics.
5. Required automated evidence includes architecture/static policy, race where supported, real PostgreSQL/NATS integration, migration, contract, secret-history, CodeQL, dependency, SBOM, license, container-hardening, and revision-integrity gates.
6. Real money, real customer or identity data, production credentials/providers, and production-ready or independently-reviewed claims remain forbidden.
7. Independent review becomes a release-blocking requirement when any policy trigger occurs: a non-synthetic deployment, real money/personal data, a real financial or identity provider, a second qualified maintainer, or a production-readiness claim.

`FND-026` is therefore an accepted, bounded solo-maintainer deviation rather than a satisfied independent-review control. `FND-020` required-check enforcement remains independently verifiable through GitHub ruleset and workflow evidence.

## Consequences

### Positive

- The project does not invent a reviewer or misrepresent a self-approval.
- Sensitive changes receive a stable risk checklist and hosted failure gates.
- Solo development may continue inside the explicitly synthetic boundary.
- The point at which independent review becomes mandatory is machine-readable and testable.

### Negative and residual risks

- The author may share the same blind spot during fresh-context self-review.
- The repository owner can administratively change the rules and policy; Git history and hosted audit evidence detect but cannot prevent that authority.
- Automated tools cannot judge all financial, regulatory, authorization, or operational semantics.
- `FND-026` cannot be reported as independently satisfied while this mode is active.

## Financial, authorization, and failure boundaries

This decision changes repository governance only. It adds no financial state, endpoint, event, database schema, identity exchange, credential, provider, worker job, or browser capability. It creates no idempotency/concurrency or before/after-commit product failure. The principal failure is governance bypass or a false independence claim; closed policy tests, hosted status checks, Git history, and claims restrictions detect or limit that failure.

## Migration and rollback/exit strategy

To exit solo mode, invite a genuinely independent qualified maintainer, grant only the required repository access, update `CODEOWNERS`, require their approval on sensitive paths, replace the solo ruleset with independent-review enforcement, retain a passing protected pull request, and supersede this ADR. Do not use an alternate account controlled by the same person.

Rollback removes the solo policy, verifier, and template only together with a superseding ADR that restores a stronger review control. Historical self-review evidence remains labelled and is never rewritten as independent review.

## Verification and evidence

- `go test ./internal/architecture -count=1` validates the closed policy, synthetic restrictions, sensitive paths, attestations, PR workflow wiring, and revalidation triggers.
- `pwsh -NoProfile -File ./scripts/test-solo-maintainer-governance.ps1` runs sensitive-path and incomplete-attestation seeded canaries.
- The PR workflow passes base/head identities and the PR body to the verifier without executing body content.
- GitHub ruleset evidence must record the rule identifier, exact required status contexts, pull-request requirement, deletion/non-fast-forward controls, and bypass actors.
- Every evidence report states `independent review unavailable—not claimed` while solo mode is active.
