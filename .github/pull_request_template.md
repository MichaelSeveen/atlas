## Change and scope

- Requirements:
- Threats:
- Affected contexts:
- Contract or migration impact:
- Financial, authorization, idempotency, and concurrency impact:
- Before/after-commit failure and recovery impact:
- Evidence and reproduce command:

## Solo-maintainer sensitive-change declaration

Complete every item when this PR changes a path listed in `.github/solo-maintainer-policy.json`.

- [ ] requirements-and-threats-reviewed — The named requirements, threats, affected contexts, and deferred behavior are accurate.
- [ ] financial-and-authorization-boundaries-reviewed — Financial and authorization boundaries are unchanged or explicitly tested and documented.
- [ ] failure-concurrency-and-recovery-reviewed — Before/after-commit failure, idempotency, concurrency, rollback, and forward-fix risks are addressed or not applicable with a reason.
- [ ] sensitive-data-and-public-claims-reviewed — No secret, real personal data, unsafe log/event/error content, or claim beyond the retained evidence is introduced.
- [ ] tests-and-revision-evidence-reviewed — Relevant static, integration, adversarial, restore, contract, and revision-binding checks pass or an explicit blocker is recorded.
- [ ] fresh-context-self-review-completed — I reviewed the final diff again after implementation and CI feedback; this is self-review, not independent human approval.

## Solo-maintainer limitation

Atlas currently operates as a synthetic portfolio project under ADR 0012. Checking these boxes does not claim independent code-owner review. Stop and obtain qualified independent review before any trigger in the solo-maintainer policy is crossed.
