# Phase 00 known limitations after successful hosted release

Recorded 2026-07-22 after successful release run `29964442782` from source `9761754709a09c96fdbb07bf1a55c39994b50e72`.

| Open limitation | Current evidence | Required closure |
|---|---|---|
| Independent human review remains unavailable | ADR 0012 permits compensating solo-maintainer controls only while the project remains synthetic and trigger-free. Active protected checks and self-review do not manufacture organizational independence. | Obtain qualified independent review before any real data, provider, money movement, second maintainer, deployment, or production-readiness trigger. |
| Staging/production-reference secret custody is not implemented | Closed references and local/test separation exist; no managed provider, signing/encryption custody, rotation overlap, or restore proof exists. | Select provider-neutral managed custody and verify provisioning, rotation, revocation, recovery, access, and environment separation before reference deployment. |
| Local backup/WAL storage is unencrypted | Hosted/local synthetic recovery validates one-second backup, WAL archive, and three-second isolated restore mechanics, not production custody. | Select encrypted backup/object/key custody and verify key loss, rotation, retention, deletion, and recovery before reference deployment. |
| No product replay state exists | No journal, outbox, inbox, idempotency, provider callback, object reference, encryption-key state, or financial state exists in Phase 00. | Add requirement-specific replay, reconciliation, missing-object, and missing-key tests with the first owning product slices. |
| Event/job propagation and queue/retry telemetry are absent | No broker stream or worker job exists; current trace/metric proof covers the feature-free API/readiness/database path and process lifecycle only. | Extend propagation, duplicate/replay, lag, retry, and alert evidence when the owning phases introduce events and jobs. |
| Alert routing is not deployed | Definitions, ownership, mutation tests, and runbooks exist; no receiver/backend has been selected or exercised. | Select the reference alert backend and prove routing, acknowledgement, escalation, and receiver-outage behavior. |

The following earlier gaps are closed by EVD-P00-S08-008: fresh independent hosted Docker execution, full outer S08, bounded Bun shutdown, immutable GHCR publication, keyless signing, exact-source SLSA/SPDX verification, and explicit four-surface release-SBOM retention.

The release remains a synthetic feature-free foundation release. It contains no product endpoint, financial behavior, product schema, broker stream, identity exchange, managed secret provider, or wallet UI. No production, compliance, availability, security, scale, real-money, cardholder-data, or identity-document claim is supported. Overall Phase 00 completion is not claimed while six requirements remain partial.
