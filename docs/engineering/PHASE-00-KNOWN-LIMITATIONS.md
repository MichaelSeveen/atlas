# Phase 00 known limitations

Last reviewed: 2026-07-22. These are release-gate facts, not silently accepted exceptions.

| Limitation | Current evidence | Closure condition |
|---|---|---|
| No GitHub ruleset protects `main` | The repository rulesets API returned an empty list after PR #1 merged. | Require the named successful PR checks, dismiss stale approvals, require conversation resolution and code-owner review, and constrain/audit bypass. |
| No independent code-owner approval | The sole configured owner authored and merged PR #1; a self-authored PR cannot prove independent owner review. | Add an authorized collaborator/team and retain an actual required approval on a protected change. |
| No hosted release, GHCR promotion, signature, or provenance | The release workflow has no run; no registry digest or keyless signature/attestation was observed. | Run the fail-closed release from committed S08 source and retain digest, Cosign identity, GitHub build/SBOM attestation, wrong-source/tamper, and signing-unavailable evidence. |
| Clean supported machine is unproven | Local Podman/WSL and hosted GitHub jobs passed their scoped lanes; the S08 clean-clone helper is same-host only. | Run the documented full command from a separate clean supported Podman or Docker host and retain revision/config/runtime identity. |
| S08 evidence is pre-commit | The current catalogue names `UNCOMMITTED_WORKTREE(base=f6ad53553e739ea44718cc1336920a37c3fd05bc)`. | After commit, add a new post-commit catalogue/report and rerun static, live, restore, clean-clone, history, and supply-chain checks; preserve this pre-commit record. |
| S08 race proof has not run in hosted Linux | The real one-connection/24-call test passed locally, but this Windows host has CGO disabled. The updated PR lane requires `-race` and has not run for S08 yet. | Obtain a successful hosted S08 PR integration job and bind its exact job/revision to post-commit evidence. |
| Web teardown required forced termination | Podman sent SIGTERM, waited ten seconds, then used SIGKILL for the stateless Bun web container; the overall isolated teardown completed successfully. | Add and verify an explicit bounded Bun shutdown path before treating graceful web termination as proved. |
| Backup/WAL storage is unencrypted local volume state | S05 proves physical backup, WAL archive, target-time restore, checksums, and isolation only. | Select and prove reference backup encryption, custody, retention, deletion, and key-recovery controls. |
| Product restore reconciliation is intentionally absent | No journal, outbox, inbox, idempotency, provider callback, object reference, or encryption-key state exists. | Add requirement-specific replay/reconciliation and missing-object/key tests with the first owning product slices. |
| Event/job propagation and queue/retry emission are absent | FND-040 covers the current API/readiness/database trace; FND-042 defines queue/retry metrics but emits none without a queue/job. | Extend telemetry and adversarial tests with the first event and worker flows. |
| Alert routing is not deployed | Alert definitions, ownership, mutation tests, and runbooks exist; no receiver/backend has been selected or exercised. | Select the reference alert backend and prove routing, acknowledgement, escalation, and outage behavior. |
| Staging/production-reference secret custody is not implemented | Closed configurations and unique credential references exist; local/test values are generated and separated. | Select managed custody and prove provisioning, rotation overlap, revocation, restore, access, and environment separation. |

No real money, real identity documents, cardholder data, production providers, compliance certification, availability target, security guarantee, or scale claim is supported by Phase 00.
