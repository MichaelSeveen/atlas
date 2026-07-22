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
| EVD-P00-S07-002 | Phase 00 | hosted PR verification | FND-020..023; FND-025..027; FND-054 | `evidence/phase-00/supply-chain/S07-hosted-pr-verification.md` | PR head `747cc80f058d570851f64592c0eb3a9ca0e33adc`; merge `f6ad53553e739ea44718cc1336920a37c3fd05bc` | five hosted jobs pass; enforcement/release partials remain | read-only `gh pr view`, `gh run view 29928153984`, ruleset and release-run queries | 2026-07-22 | no ruleset, independent owner approval, registry promotion, signature, or provenance |
| EVD-P00-S08-001 | Phase 00 | acceptance, restore, and evidence integrity | all Phase 00; focus FND-010; FND-020..024; FND-040..043; FND-064 | `evidence/phase-00/acceptance/S08-phase-00-acceptance-report.md`; `S08-evidence-catalogue-precommit.json` | `UNCOMMITTED_WORKTREE(base=f6ad53553e739ea44718cc1336920a37c3fd05bc)` | static/live/restore/constrained-pool pass; Phase 00 not complete | `scripts/verify-s08.ps1`; add `-Live -History -SupplyChain`; add `-CleanClone` after commit | 2026-07-22 | pre-commit; S08 race/clean-host/ruleset/review/release/encryption/routing gaps; web required forced stop |
| EVD-P00-S08-002 | Phase 00 | committed static and clean-clone acceptance | all Phase 00; focus FND-010 and evidence integrity | `evidence/phase-00/acceptance/S08-post-commit-verification.md`; `S08-evidence-catalogue-postcommit.json` | implementation `6b09b4abfec050d6cdceb98af01f12bf0cab03af`; verified `431821f364165055d7e7ca7d69f047e860ee66aa` | committed static and detached same-host clean clone pass; Phase 00 not complete | `scripts/verify-s08.ps1 -CleanClone` | 2026-07-22 | same host, not independent machine; race later closed by EVD-P00-S08-003 and ruleset by EVD-P00-S08-004; review/release/encryption/routing gaps remain |
| EVD-P00-S08-003 | Phase 00 | hosted PR verification | FND-020..023; FND-025; FND-027; FND-054 | `evidence/phase-00/acceptance/S08-hosted-pr-verification.md` | PR head `10ed35b8d86a68d821c89f69822289f5ab655aa8` | five hosted jobs and constrained-pool race test pass | read-only `gh pr view 19`; `gh run view 29943586545`; job-log, ruleset, and release-run queries | 2026-07-22 | required-check ruleset later closed by EVD-P00-S08-004; independent review/release/clean-host gaps remain |
| EVD-P00-S08-004 | Phase 00 | solo-maintainer governance and active ruleset | FND-020; FND-026; FND-054 | `evidence/phase-00/acceptance/S08-solo-maintainer-governance.md` | implementation `08762a3e1333043d021264a875b8e5e222e9c34c`; hosted head `8c1032333356fe2d10b91ab46328f0a187290024` | sensitive declaration, five hosted jobs, and active no-bypass `main` rules pass; FND-026 remains an accepted deviation | policy/architecture/S08 commands; `gh run view 29949126130`; ruleset and applicable-branch-rule queries | 2026-07-22 | independent human review unavailable/not claimed; mandatory before ADR 0012 trigger; release and independent clean host remain open |
| EVD-P00-S08-005 | Phase 00 | hosted-release closure preflight | FND-010; FND-023; FND-024 | `evidence/phase-00/acceptance/S08-hosted-release-closure-preflight.md` | implementation `eae43b62f3e0e3f95a09bff46f1ac73217dde5c3`; verified `7b1e28ed8e52bc44d593b6114372c28782c8468a` | exact committed full local S08 and exit-zero bounded web shutdown pass; full S08 precedes every release mutation; release not run | `scripts/verify-s08.ps1 -Live -History -SupplyChain -CleanClone -ContainerRuntime podman` | 2026-07-22 | pre-PR local Podman host; hosted clean-machine, GHCR, signature, and provenance proof remain open |
| EVD-P00-S08-006 | Phase 00 | fail-closed hosted release attempt 1 | FND-010; FND-023; FND-024 | `evidence/phase-00/acceptance/S08-hosted-release-attempt-1.md`; `S08-known-limitations-release-attempt-1.md` | release source `36fdaa630fad8686d60f5330efeb374dd0df5b6f`; correction `baf2e67777de935e985162df14d51f9a2ebaac96` | hosted acceptance checks pass; outer cleanup fails; every publication step skipped | `gh run view 29959302878 --log-failed`; focused architecture and clean-clone checks | 2026-07-22 | successful outer S08, GHCR digests, signatures, provenance, and release SBOMs remain open |
| EVD-P00-S08-007 | Phase 00 | hosted release attempt 2 and partial publication | FND-010; FND-023; FND-024 | `evidence/phase-00/acceptance/S08-hosted-release-attempt-2.md`; `S08-known-limitations-release-attempt-2.md` | release source `0ebf1b8ac5b5a72d28a8c911154664475756c3a9`; correction `4b34edda1ac89f64840841a1f82e0ddd4159882f` | full hosted S08 and publication/signing/attestation pass; independent exact-source verification passes; automated verification lacks token and retained SBOM step is skipped | run/log queries; Cosign; exact-source GitHub SLSA/SPDX verification; public OCI manifest inspection | 2026-07-22 | FND-010 satisfied; successful automated release tail and retained SBOMs remain open for FND-023/FND-024 |
| EVD-P00-S08-008 | Phase 00 | successful hosted release closure | FND-010; FND-022; FND-023; FND-024 | `evidence/phase-00/acceptance/S08-hosted-release-success.md`; `S08-known-limitations-hosted-release.md` | release source `9761754709a09c96fdbb07bf1a55c39994b50e72`; run `29964442782` | full fresh-host S08, immutable publication, keyless signing, automated and independent exact-source SLSA/SPDX verification, and four-surface SBOM retention pass | run/log/artifact queries; strict Cosign identity; exact-source GitHub predicates; public OCI manifest and downloaded SPDX inspection | 2026-07-22 | hosted-release closure complete at synthetic foundation depth; final requirement dispositions are recorded separately by EVD-P00-GATE-001 |
| EVD-P00-GATE-001 | Phase 00 | final requirement disposition and topology guard | FND-011; FND-026; FND-031; FND-040; FND-042; FND-064 | `evidence/phase-00/acceptance/Phase-00-gate-closure.md`; ADR 0013; `docs/engineering/phase-00-gate-policy.json` | pre-commit until catalogue-binding commit | 34 requirements satisfied; FND-026 accepted deviation; FND-040/FND-042 accepted scope decisions; four policy mutations rejected | `go test ./internal/architecture -run TestPhase00GateClosurePolicy -count=1`; `scripts/verify-s08.ps1` | 2026-07-23 | complete only for the synthetic feature-free foundation; every recorded topology/independent-review trigger remains mandatory |

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
