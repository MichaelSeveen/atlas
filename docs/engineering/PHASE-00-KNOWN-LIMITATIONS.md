# Phase 00 known limitations

Last reviewed: 2026-07-22 after successful hosted release run `29964442782`. These are scope and release-gate facts, not silently accepted exceptions.

| Limitation | Current evidence | Closure condition |
|---|---|---|
| Independent code-owner approval remains unavailable | ADR 0012, the closed solo-maintainer policy, sensitive declarations, self-review, and active no-bypass ruleset are verified. They do not manufacture organizational separation. | Obtain qualified independent approval before any real data, provider, money movement, second maintainer, deployment, or production-readiness trigger. |
| Staging/production-reference secret custody is not implemented | Closed configurations and unique credential references exist; local/test values are generated and separated. No managed provider or signing/encryption custody exists. | Select managed custody and prove provisioning, rotation overlap, revocation, restore, access, and environment separation. |
| Backup/WAL storage is unencrypted synthetic volume state | The hosted release proves physical backup, WAL archive, and isolated three-second restore mechanics only. | Select and prove reference backup encryption, key custody, retention, deletion, and key recovery. |
| Product restore reconciliation is intentionally absent | No journal, outbox, inbox, idempotency, provider callback, object reference, encryption-key state, or financial state exists. | Add requirement-specific replay/reconciliation and missing-object/key tests with the first owning product slices. |
| Event/job propagation and queue/retry emission are absent | FND-040 covers the current API/readiness/database trace; FND-042 defines queue/retry metrics but emits none without a queue/job. | Extend telemetry and adversarial tests with the first event and worker flows. |
| Alert routing is not deployed | Alert definitions, ownership, mutation tests, and runbooks exist; no receiver/backend has been selected or exercised. | Select the reference alert backend and prove routing, acknowledgement, escalation, and outage behavior. |

Successful run `29964442782` closes the earlier ruleset, fresh-host, bounded-web-shutdown, GHCR publication, signature/provenance, automated identity-verification, and retained-SBOM limitations. EVD-P00-S08-006 and EVD-P00-S08-007 remain preserved because their failures materially explain the cleanup and token-binding corrections.

No real money, real identity documents, cardholder data, production providers, deployment, compliance certification, availability target, security guarantee, or scale claim is supported by this synthetic feature-free release. Overall Phase 00 completion is not claimed while six requirements remain partial.
