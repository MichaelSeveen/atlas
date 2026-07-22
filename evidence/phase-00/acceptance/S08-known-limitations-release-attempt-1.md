# Phase 00 known limitations after hosted release attempt 1

Recorded 2026-07-22 after release run `29959302878` failed closed from source `36fdaa630fad8686d60f5330efeb374dd0df5b6f`.

| Open limitation | Current evidence | Required closure |
|---|---|---|
| No successful hosted release, GHCR promotion, signature, or provenance | EVD-P00-S08-006 records that attempt 1 completed substantive hosted acceptance but failed during disposable clone cleanup; every publication step was skipped. | Merge correction `baf2e67`, rerun the authorized release from protected `main`, and retain digests, signatures, build/SBOM attestations, identity verification, and retained SBOM identities. |
| Complete independent clean-machine command is not yet green | The fresh GitHub Linux runner passed history, supply chain, live stack, recovery, race, bounded shutdown, and nested clean-clone acceptance, but the outer command exited 1 during cleanup. | Observe `s08_clean_clone=PASS` and outer `s08_verification=PASS` with a successful job exit on the hosted runner. |
| Independent human review remains unavailable | ADR 0012 permits compensating solo-maintainer controls only while the project remains synthetic and trigger-free. | Obtain qualified independent review before any real data, provider, money movement, second maintainer, or production-readiness trigger. |
| Local backup storage is unencrypted | Local/hosted synthetic recovery validates mechanics, not production custody. | Select and verify encrypted backup/key custody before production-reference readiness. |
| No product replay state exists | No product schema, outbox/inbox, idempotency, object, key, or financial state exists in Phase 00. | Add and test those replay classes only in their prerequisite product phases. |
| Alert routing and event/job telemetry are absent | No broker stream or worker job exists; no deployed alert backend is selected. | Implement and verify those controls when their owning phases introduce the behavior. |

Hosted bounded Bun shutdown is no longer an open limitation: attempt 1 observed exit code `0` in `218ms` on Docker. This does not convert the failed release attempt into publication evidence.
