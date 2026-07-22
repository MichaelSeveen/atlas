# S07 hosted PR verification

- **Evidence ID:** EVD-P00-S07-002
- **Created:** 2026-07-22T15:37:21Z
- **Hosted source revision:** `747cc80f058d570851f64592c0eb3a9ca0e33adc`
- **Merged revision:** `f6ad53553e739ea44718cc1336920a37c3fd05bc`
- **Pull request:** [MichaelSeveen/atlas#1](https://github.com/MichaelSeveen/atlas/pull/1)
- **Workflow run:** [atlas-pr run 29928153984](https://github.com/MichaelSeveen/atlas/actions/runs/29928153984)
- **Requirements:** closes the hosted-execution facet of FND-020; supports FND-021, FND-022, FND-025, FND-027, and FND-054; FND-020, FND-023, FND-024, and FND-026 remain partial
- **Threats:** THR-009, THR-013, THR-014, THR-019, THR-030, THR-042, and THR-060
- **Result:** PASS for the five configured hosted jobs; protected rules, review enforcement, registry promotion, signature, and provenance remain absent
- **Revalidate by:** 2026-08-22 or on any workflow, action, dependency, contract, migration, scanner, image, ownership, or release change

## Observed hosted jobs

The `pull_request` run completed successfully against the exact PR head:

| Job | Job ID | Result |
|---|---:|---|
| `static-contracts-secret-history` | 88950550585 | PASS |
| `postgres-nats-migration-lanes` | 88950550594 | PASS |
| `codeql-go-typescript-go` | 88950550850 | PASS |
| `codeql-go-typescript-javascript-typescript` | 88950550490 | PASS |
| `sbom-vulnerability-license-container` | 88950550444 | PASS |

The run started on 2026-07-22 and completed before PR #1 merged at `2026-07-22T14:36:32Z`. The merge commit is present on `main` and contains the verified PR head.

## Remaining hosted gates

After the merge, the GitHub repository rulesets API returned `[]`, so successful jobs are execution evidence but are not required-check enforcement evidence. The release workflow run list was empty, so no GHCR digest promotion, keyless Cosign signature, GitHub build provenance, or SBOM attestation is claimed. The PR author and repository owner were both `MichaelSeveen`; this does not prove independent code-owner approval.

## Sanitization and integrity

This derivative contains public repository, revision, workflow, job, and timestamp identifiers only. It contains no GitHub token, runtime credential, scanner database, connection string, customer data, or product payload. Reproduce with read-only `gh pr view`, `gh run view`, `gh api repos/MichaelSeveen/atlas/rulesets`, and `gh run list --workflow release.yml` queries while authenticated to the repository.
