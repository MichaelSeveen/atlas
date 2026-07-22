# Atlas evidence index

## Purpose

This index is populated as implementation proceeds. It keeps the PRD honest by connecting requirements, threats, tests, claims, source revisions, and reproducible artifacts.

## Storage convention

```text
evidence/
  <phase>/
    architecture/
    contracts/
    tests/
    security/
    performance/
    operations/
    recovery/
    content/
```

Every artifact has a sidecar metadata file or embedded header containing:

- evidence ID;
- source revision and build/image digest;
- environment/configuration digest;
- creation timestamp and actor/tool;
- requirement/threat/test/claim IDs;
- synthetic fixture/scenario seed;
- command/procedure to reproduce;
- expected and observed result;
- sanitization/redaction statement;
- integrity digest/signature where appropriate;
- limitation and expiry/revalidation date.

## Evidence catalogue template

| Evidence ID | Phase | Type | Supports | Artifact | Revision | Result | Reproduce | Last verified | Limitation |
|---|---|---|---|---|---|---|---|---|---|
| EVD-0001 | Phase 03 | ledger verification | LED-012; CLM-001; THR-001 | `evidence/...` | commit | pass | command/runbook | date | ... |
| EVD-P00-S04-001 | Phase 00 | synthetic environment | FND-004; FND-010..013; FND-030..033; FND-054 | `evidence/phase-00/environment/S04-environment-report.md`; `S04-post-commit-verification.md` | `39121a31765013ebdc51b3b0ac4e47c9bc8b1516` | pass with explicit partials | `scripts/verify-s04.ps1 -Live` | 2026-07-21 | exact clean-machine wrapper, executable seeds, and staging/production credentials remain outstanding |
| EVD-P00-S05-001 | Phase 00 | database migration, roles, and recovery | FND-021; FND-025; FND-060..064 | `evidence/phase-00/database/S05-database-report.md`; `S05-static-verification.txt`; `S05-live-database.txt`; `S05-recovery.txt` | `UNCOMMITTED_WORKTREE(base=199b86113a9f0fcda323ae2775acf026b521067e)` | pass for S05 mechanics; FND-064 partial | `scripts/verify-s05.ps1 -Live` | 2026-07-21 | pre-commit; Windows host Compose transport workaround; backup/WAL volumes unencrypted; no product replay state |
| EVD-P00-S05-002 | Phase 00 | clean post-commit verification | FND-021; FND-025; FND-060..064 | `evidence/phase-00/database/S05-post-commit-verification.md` | `5ea77fcf31b349b53fcd14e14ab81a4da5da840a` | pass; FND-064 remains partial | `scripts/verify-s05.ps1` | 2026-07-21 | static/build/test proof; live evidence remains EVD-P00-S05-001 with its host/encryption limitations |
| EVD-P00-S06-001 | Phase 00 | observability and security operating baseline | FND-040..043; FND-050..053; FND-055 | `evidence/phase-00/observability-security/S06-observability-security-report.md` | `UNCOMMITTED_WORKTREE(base=7a08056539de6d655086f7730d0cb8df3a9bb4c6)` | pass for current S06 path; FND-040/FND-042 partial | `scripts/verify-s06.ps1 -Live -ContainerRuntime podman` | 2026-07-21 | pre-commit; no event/job propagation, queue/retry emission, deployed alert routing, managed secret provider, or clean-host proof |
| EVD-P00-S07-001 | Phase 00 | CI, contract, and supply-chain integrity | FND-020..027; FND-054 | `evidence/phase-00/supply-chain/S07-ci-contract-supply-chain-report.md` | `UNCOMMITTED_WORKTREE(base=3342b4ded1cd62fab1223372cd5129f272889878)` | local pass; FND-020/FND-023/FND-024/FND-026 partial | `scripts/verify-s07.ps1 -History`; `scripts/verify-s07.ps1 -SupplyChain -ContainerRuntime podman` | 2026-07-22 | pre-commit; hosted required checks/ruleset/review/signature/provenance absent; Windows race/Gosec unavailable; clean-host proof outstanding |

## Minimum phase evidence

- accepted architecture/security review;
- requirement and threat updates;
- OpenAPI/AsyncAPI/schema examples where applicable;
- critical invariant and adversarial test report;
- one failure-injection trace;
- database/ledger/audit facts;
- dashboard/alert/runbook proof;
- acceptance demo recording or reproducible script;
- known limitations;
- sanitized X content artifact.

## Integrity rules

- Do not overwrite historical evidence; create a new version.
- A screenshot without source revision/scenario is supporting context, not primary proof.
- CI artifacts with limited retention must be copied to durable versioned storage or regenerated before public claims.
- Public evidence is sanitized derivative of internal test evidence; preserve the linkage and digest.
- Failed tests/findings are retained when they materially explain a design correction.
- Evidence expires when implementation, dependency, standard baseline, or environment materially changes.
