# Phase 00 known limitations — post-commit

Last reviewed: 2026-07-22. These are release-gate facts, not silently accepted exceptions. The earlier pre-commit limitations file remains preserved because it is hashed by EVD-P00-S08-001.

| Limitation | Current evidence | Closure condition |
|---|---|---|
| Independent code-owner approval is unavailable under solo mode | ADR 0012, EVD-P00-S08-004, and active ruleset `19577130` prove PR-only required checks, sensitive declarations, no bypass actors, and an explicit `UNAVAILABLE_NOT_CLAIMED` result. These controls do not create independent human judgment. | Obtain genuine qualified code-owner review and supersede ADR 0012 before any non-synthetic deployment, real data/money/provider, second maintainer, or production-readiness claim. |
| No hosted release, GHCR promotion, signature, or provenance | The release workflow has no run; no registry digest or keyless signature/attestation was observed. | Run the fail-closed release from committed S08 source and retain digest, Cosign identity, GitHub build/SBOM attestation, wrong-source/tamper, and signing-unavailable evidence. |
| Independent clean machine is unproven | EVD-P00-S08-002 passed an exact detached clean clone with empty clone-local dependency/build caches on this host. | Run the documented full command from a separately administered clean supported Podman or Docker host and retain revision/config/runtime identity. |
| Web teardown required forced termination | Podman sent SIGTERM, waited ten seconds, then used SIGKILL for the stateless Bun web container; the overall isolated teardown completed successfully. | Add and verify an explicit bounded Bun shutdown path before treating graceful web termination as proved. |
| Backup/WAL storage is unencrypted local volume state | S05/S08 prove physical backup, WAL archive, target-time restore, checksums, and isolation only. | Select and prove reference backup encryption, custody, retention, deletion, and key-recovery controls. |
| Product restore reconciliation is intentionally absent | No journal, outbox, inbox, idempotency, provider callback, object reference, or encryption-key state exists. | Add requirement-specific replay/reconciliation and missing-object/key tests with the first owning product slices. |
| Event/job propagation and queue/retry emission are absent | FND-040 covers the current API/readiness/database trace; FND-042 defines queue/retry metrics but emits none without a queue/job. | Extend telemetry and adversarial tests with the first event and worker flows. |
| Alert routing is not deployed | Alert definitions, ownership, mutation tests, and runbooks exist; no receiver/backend has been selected or exercised. | Select the reference alert backend and prove routing, acknowledgement, escalation, and outage behavior. |
| Staging/production-reference secret custody is not implemented | Closed configurations and unique credential references exist; local/test values are generated and separated. | Select managed custody and prove provisioning, rotation overlap, revocation, restore, access, and environment separation. |

No real money, real identity documents, cardholder data, production providers, compliance certification, availability target, security guarantee, or scale claim is supported by Phase 00.
