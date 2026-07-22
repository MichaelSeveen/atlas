# S08 hosted release attempt 1

- **Evidence ID:** EVD-P00-S08-006
- **Created:** 2026-07-22T21:51:48Z
- **Release source revision:** `36fdaa630fad8686d60f5330efeb374dd0df5b6f`
- **Workflow run:** `29959302878`, attempt 1, `workflow_dispatch` from `main`
- **Correction revision:** `baf2e67777de935e985162df14d51f9a2ebaac96`
- **Requirements:** FND-010, FND-023, FND-024 remain partial
- **Threats:** THR-013; THR-014; THR-018; THR-019; THR-030; THR-042; THR-060
- **Result:** FAIL_CLOSED after substantive hosted acceptance and before registry authentication or publication
- **Revalidate by:** the next release-workflow, clean-clone, Go toolchain, image, signing, attestation, or runtime change

## Observed hosted results

The fresh GitHub Linux runner completed the substantive release preflight at the merge revision:

| Check | Observed result |
|---|---|
| Go race and Gosec | PASS; `s07_race=PASS`, `s07_gosec=PASS(full)` |
| History and supply chain | PASS; `s07_supply_chain=PASS` |
| Live observability | PASS; `s06_live_observability=PASS` |
| Database backup and isolated recovery | PASS; base backup 1 second, restore RTO 3 seconds, `s05_database_verify=PASS` |
| Constrained pool | PASS with hosted race proof |
| Bounded Bun shutdown | PASS; exit code `0` in `218ms` |
| Nested clean-clone S08 | PASS at `36fdaa630fad8686d60f5330efeb374dd0df5b6f` |

The outer preflight then failed while its `finally` block removed the disposable clone. Go had made a clone-local module-cache directory read-only, so PowerShell could not delete `gopkg.in/yaml.v3@v3.0.1/.github/workflows/go.yaml`. The workflow stopped with exit code 1.

## Fail-closed publication boundary

Every mutation step after preflight was skipped: Buildx setup, GHCR authentication, backend/web push, release SBOM generation, keyless signing, build/SBOM attestations, identity verification, and artifact retention. No GHCR digest, signature, provenance statement, or release SBOM artifact was created by this run.

## Correction

Revision `baf2e67777de935e985162df14d51f9a2ebaac96` passes Go's documented `-modcacherw` flag only to the isolated nested verifier and restores the caller's `GOFLAGS` afterward. A seeded architecture test rejects removal of that control. A local exact-revision clean-clone exercise confirmed the disposable clone was removed even when the independent evidence-staleness guard intentionally stopped the nested suite.

The release must not be rerun until the correction passes protected PR checks and is merged to `main`.

## Reproduce and inspect

```powershell
gh run view 29959302878 --json status,conclusion,headSha,jobs,url
gh run view 29959302878 --log-failed
go test ./internal/architecture -count=1
pwsh -NoProfile -File ./scripts/test-s08-clean-clone.ps1
```

## Sanitization and limitations

This record contains only public revisions, workflow/job identities, bounded durations, pass/failure markers, and a public dependency-cache path. It contains no token, credential, connection string, customer data, identity data, product payload, raw scanner database, or complete service log.

The hosted run proves the fail-before-publish boundary and the named acceptance checks, but it is not successful release evidence. FND-010 remains partial until the complete outer command exits successfully. FND-023 and FND-024 remain partial until immutable GHCR digests are published, signed, attested, and verified.
