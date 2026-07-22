# S08 post-commit verification

- **Evidence ID:** EVD-P00-S08-002
- **Created:** 2026-07-22T17:29:53Z
- **Implementation revision:** `6b09b4abfec050d6cdceb98af01f12bf0cab03af`
- **Implementation tree:** `027b3dfe3855d89447d2a4b6bb598dfecfef8aeb`
- **Verification revision:** `431821f364165055d7e7ca7d69f047e860ee66aa`
- **Verification tree:** `aa2412782836554b76c36bfbf1a83d46b4817156`
- **Requirements:** all Phase 00 requirements reviewed; post-commit/clean-clone proof for FND-010 and the S08 evidence gate
- **Threats:** THR-013, THR-014, THR-019, THR-025, THR-042, THR-045, and THR-060
- **Result:** PASS for committed static S08 acceptance and same-host isolated clean clone; Phase 00 completion is not claimed
- **Revalidate by:** 2026-08-22 or on any implementation, evidence, workflow, contract, dependency, migration, configuration, or toolchain change

## Revision relationship

The post-commit catalogue names implementation revision `6b09b4a`. Verification revision `431821f` is its descendant and changes only the versioned S08 catalogue plus its sidecar. The evidence verifier checks this ancestry and rejects any intervening path outside the closed evidence/status allowlist. This prevents an old catalogue from silently validating changed application, workflow, contract, configuration, migration, or test code.

The pre-commit report and catalogue remain preserved. This report does not rewrite them or pretend their earlier source identity was a commit.

## Reproduction and observed result

```powershell
pwsh -NoProfile -File ./scripts/verify-s08.ps1 -CleanClone
```

The sandboxed attempt correctly failed before clone creation because Git's bundled `sh.exe` could not create its Windows signal pipe. Repeating the identical repository command with normal host process permissions passed:

```text
source_revision=431821f364165055d7e7ca7d69f047e860ee66aa
s07_verification=PASS
s08_evidence_tamper_canary=PASS
s08_evidence_stale-source_canary=PASS
s08_evidence_integrity=PASS
s08_clean_clone_revision=431821f364165055d7e7ca7d69f047e860ee66aa
s08_clean_clone=PASS
s08_phase_00_completion=NOT_CLAIMED
s08_verification=PASS
```

The isolated clone used a `file:///` upload-pack source, detached the exact verification revision, created clone-local Go build/module caches, downloaded Go `1.25.12` plus the locked modules into those empty caches, performed a frozen Bun `1.3.0` install, and passed all Go tests, architecture policies, contract/migration checks, React tests/typecheck/build, and catalogue mutation checks. The clone was removed through the bounded repository-confined cleanup path.

## Relationship to live and supply-chain evidence

EVD-P00-S08-001 records the complete-history/security, four-SBOM/hardened-image, synthetic-stack, golden-trace, outage, real PostgreSQL/NATS, long-lock, backup, restore, and constrained-pool results from the worktree that became the committed implementation. The only later implementation change was the Windows clean-clone transport correction in `6b09b4a`; post-commit static and clean-clone verification cover that final code.

## Sanitization and limitations

No credential, token, connection string, customer/identity data, product payload, dependency cache, runtime state, or raw scanner output is retained. This is a clean clone on the same Windows host, not a separate independently administered machine. Hosted S08 race, protected rules, independent review, registry promotion, keyless signature/provenance, backup encryption, deployed alert routing, and later product replay cases remain open.
