# S08 hosted release attempt 2

- **Evidence ID:** EVD-P00-S08-007
- **Created:** 2026-07-22T22:31:32Z
- **Release source revision:** `0ebf1b8ac5b5a72d28a8c911154664475756c3a9`
- **Workflow run:** `29962201077`, attempt 1, release run number 2, `workflow_dispatch` from `main`
- **Correction revision:** `4b34edda1ac89f64840841a1f82e0ddd4159882f`
- **Requirements:** satisfies FND-010 at its scoped Phase 00 depth; advances FND-023 and FND-024 without claiming completed automated release verification
- **Threats:** THR-013; THR-014; THR-018; THR-019; THR-030; THR-042; THR-060
- **Result:** PARTIAL_PUBLISH; full hosted S08, publication, signing, and attestation passed; workflow failed closed at automated GitHub attestation verification because `GH_TOKEN` was absent
- **Revalidate by:** the next release-workflow, image, signing, attestation, source, or runtime change

## Hosted acceptance

The corrected fresh GitHub Linux runner completed the entire outer release preflight successfully:

| Check | Observed result |
|---|---|
| Go race and Gosec | PASS; `s07_race=PASS`, `s07_gosec=PASS(full)` |
| History and four-surface supply chain | PASS; `s07_supply_chain=PASS` |
| Live observability | PASS; `s06_live_observability=PASS` |
| Database backup and isolated recovery | PASS; base backup 1 second, restore RTO 3 seconds |
| Constrained pool | PASS with hosted race proof |
| Bounded Bun shutdown | PASS; exit code `0` in `232ms` |
| Exact clean clone and cleanup | PASS; `s08_clean_clone_revision=0ebf1b8ac5b5a72d28a8c911154664475756c3a9` |
| Outer acceptance | PASS; `s08_verification=PASS` |

This is the first complete fresh-host execution of the full live/history/supply/clean-clone S08 command and closes FND-010 at the current feature-free Phase 00 depth.

## Published immutable images

Public OCI indexes were inspected directly through GHCR's pull-only registry API. Compressed size is the Linux/amd64 image config plus compressed layers; it excludes the OCI index envelope and its separate attestation descriptor.

| Image | Source tag | OCI index digest | Linux/amd64 manifest | Compressed size | Layers |
|---|---|---|---|---:|---:|
| Backend | `0ebf1b8ac5b5a72d28a8c911154664475756c3a9` | `sha256:87e215bfd603dd2762bcb8fb93e71447202f74f93e4762cb0f8b6f5921a4d756` | `sha256:5ca195b794e7cc045a0b027f15a0cdc99f29df234f51d87de6e786da8e258abe` | 21,984,328 bytes (20.97 MiB) | 3 |
| Web | `0ebf1b8ac5b5a72d28a8c911154664475756c3a9` | `sha256:2e3ab8bf0e8bce206f8db13d12ddfc9e4d76cf22954b67a288cd809df51fd1bc` | `sha256:20c53ee43dbd3f142e6b038e32a13e3697170bec8d89475163bbd74953f45dff` | 43,234,172 bytes (41.23 MiB) | 9 |

## Signatures and attestations

The workflow's keyless-signing step and all four GitHub attestation steps completed: backend/web Cosign signatures, backend/web SLSA v1 provenance, and backend/web SPDX 2.3 attestations.

Independent read-only verification after the run passed for both digests:

- Cosign certificate identity: `https://github.com/MichaelSeveen/atlas/.github/workflows/release.yml@refs/heads/main`;
- OIDC issuer: `https://token.actions.githubusercontent.com`;
- signer workflow: `MichaelSeveen/atlas/.github/workflows/release.yml`;
- source digest: `0ebf1b8ac5b5a72d28a8c911154664475756c3a9`;
- source ref: `refs/heads/main`;
- one SLSA v1 and one SPDX 2.3 attestation verified for each image.

## Fail-closed verification boundary

The workflow's Cosign verification of the backend succeeded and enumerated its Cosign, SLSA, and SPDX signed entries. The next command, `gh attestation verify`, refused to run because GitHub CLI requires `GH_TOKEN` in Actions. The job exited 1 before verifying the web digest in-workflow and before the explicit retained-SBOM upload. Two Buildx metadata artifacts were retained automatically, but they are not substitutes for the repository's release SBOM artifact.

Correction `4b34edda1ac89f64840841a1f82e0ddd4159882f` exposes only `${{ github.token }}` to the verification step, enforces the exact signer workflow/source digest/source ref, verifies both SLSA and SPDX predicates for both images, and adds missing-token and missing-SPDX seeded negatives.

## Reproduce and inspect

```powershell
gh run view 29962201077 --json status,conclusion,headSha,jobs,url
gh run view 29962201077 --log
gh attestation verify "oci://ghcr.io/michaelseveen/atlas-backend@sha256:87e215bfd603dd2762bcb8fb93e71447202f74f93e4762cb0f8b6f5921a4d756" --repo MichaelSeveen/atlas --signer-workflow MichaelSeveen/atlas/.github/workflows/release.yml --source-digest 0ebf1b8ac5b5a72d28a8c911154664475756c3a9 --source-ref refs/heads/main
gh attestation verify "oci://ghcr.io/michaelseveen/atlas-web@sha256:2e3ab8bf0e8bce206f8db13d12ddfc9e4d76cf22954b67a288cd809df51fd1bc" --repo MichaelSeveen/atlas --signer-workflow MichaelSeveen/atlas/.github/workflows/release.yml --source-digest 0ebf1b8ac5b5a72d28a8c911154664475756c3a9 --source-ref refs/heads/main
```

Repeat each attestation command with `--predicate-type https://spdx.dev/Document/v2.3` for the SBOM predicate.

## Sanitization and limitations

This record contains only public revisions, workflow identities, immutable OCI digests, compressed sizes, bounded durations, and pass/failure markers. It contains no token, credential, connection string, customer data, identity data, product payload, raw scanner database, certificate body, or complete service log.

FND-023 and FND-024 remain partial until a protected corrected run performs the same exact verification automatically and retains the named release SBOM artifact. The images are synthetic feature-free foundation artifacts; publication is not deployment, production readiness, or a compliance/security/availability claim. Phase 00 completion is not claimed.
