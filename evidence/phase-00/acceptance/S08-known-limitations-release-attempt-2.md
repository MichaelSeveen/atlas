# Phase 00 known limitations after hosted release attempt 2

Recorded 2026-07-22 after release run `29962201077` partially published source `0ebf1b8ac5b5a72d28a8c911154664475756c3a9`.

| Open limitation | Current evidence | Required closure |
|---|---|---|
| Automated release verification and explicit SBOM retention are not green | EVD-P00-S08-007 records complete hosted S08, two published immutable images, signatures, SLSA/SPDX attestations, and independent verification. The in-workflow GitHub CLI lacked `GH_TOKEN`; the retained-SBOM step was skipped. | Merge correction `4b34edd`, rerun from protected `main`, require exact-source SLSA/SPDX verification for both digests, and retain the named 90-day release SBOM artifact. |
| Independent human review remains unavailable | ADR 0012 permits compensating solo-maintainer controls only while the project remains synthetic and trigger-free. | Obtain qualified independent review before any real data, provider, money movement, second maintainer, or production-readiness trigger. |
| Local backup storage is unencrypted | Hosted/local synthetic recovery validates mechanics, not production custody. | Select and verify encrypted backup/key custody before production-reference readiness. |
| No product replay state exists | No product schema, outbox/inbox, idempotency, object, key, or financial state exists in Phase 00. | Add and test those replay classes only in their prerequisite product phases. |
| Alert routing and event/job telemetry are absent | No broker stream or worker job exists; no deployed alert backend is selected. | Implement and verify those controls when their owning phases introduce the behavior. |

Fresh-host full S08, hosted bounded Bun shutdown, immutable digest publication, and independent signature/SLSA/SPDX verification are no longer absent. They do not replace the required successful automated release transcript and retained SBOM artifact.
