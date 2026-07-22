# S08 successful hosted release

- **Evidence ID:** EVD-P00-S08-008
- **Created:** 2026-07-22T23:10:22Z
- **Release source revision:** `9761754709a09c96fdbb07bf1a55c39994b50e72`
- **Workflow run:** `29964442782`, attempt 1, release run number 3, `workflow_dispatch` from protected `main`
- **Workflow conclusion:** SUCCESS in 8 minutes 29 seconds
- **Requirements:** FND-010; FND-022; FND-023; FND-024
- **Threats:** THR-013; THR-014; THR-018; THR-019; THR-020; THR-030; THR-042; THR-060
- **Result:** PASS for full fresh-host S08, immutable publication, keyless signing, exact-source SLSA/SPDX verification, and explicit 90-day SBOM retention
- **Revalidate by:** the next release-workflow, source, dependency, image, signing, attestation, registry, or supported-runtime change

## Hosted acceptance before publication

The release ran the repository-owned full live/history/supply/clean-clone command on a fresh GitHub-hosted Docker runner before registry authentication or any publication step.

| Check | Observed result |
|---|---|
| Go race and Gosec | PASS; `s07_race=PASS`, `s07_gosec=PASS(full)` |
| History and four-surface supply chain | PASS; `s07_supply_chain=PASS` |
| Live observability and collector-outage readiness | PASS; `s06_live_observability=PASS` |
| PostgreSQL base backup and WAL archive | PASS; base backup 1 second |
| Isolated point-in-time restore | PASS; restore RTO 3 seconds |
| Constrained real PostgreSQL pool | PASS with one connection and hosted race proof |
| Bounded Bun shutdown | PASS; container exit code `0` in `213ms` |
| Evidence integrity | PASS; tamper and stale-source canaries killed |
| Exact clean clone | PASS at `9761754709a09c96fdbb07bf1a55c39994b50e72` |
| Outer acceptance | PASS; `s08_verification=PASS` |

No release mutation began until this preflight completed successfully.

## Immutable images

The source revision tag and OCI revision label identify the release source. Public GHCR index and Linux/amd64 manifest inspection produced the following identities. Compressed size is the image config plus compressed layers and excludes the OCI index envelope and its separate attestation descriptor.

| Image | OCI index digest | Linux/amd64 manifest | Compressed size | Layers | Index descriptors |
|---|---|---|---:|---:|---:|
| `ghcr.io/michaelseveen/atlas-backend` | `sha256:17026f682f58e2e400a3e94f4d37516870d7e3593f20955d93f73abd10dfd9e9` | `sha256:f72cab9a7baff4b060261b6dc165cb4c8080529293dbfa108d5130cac4f33f9c` | 21,984,327 bytes (20.97 MiB) | 3 | 2 |
| `ghcr.io/michaelseveen/atlas-web` | `sha256:a0224f362c119e1802d26d6b58fea642b4080c233a530db6d74f83d69d23c733` | `sha256:29e9d43425210c16698139541e60382ffae4920a8dc1c15470b42cdbd959f429` | 43,234,186 bytes (41.23 MiB) | 9 | 2 |

These are synthetic feature-free foundation images. Publication is not deployment, production readiness, capacity, availability, compliance, or security certification.

## Signatures and attestations

The workflow signed both exact OCI index digests with keyless Cosign and attached GitHub SLSA v1 provenance plus SPDX 2.3 SBOM attestations. Its corrected automated verification step passed before release completion and enforced:

- repository `MichaelSeveen/atlas`;
- signer workflow `MichaelSeveen/atlas/.github/workflows/release.yml`;
- source digest `9761754709a09c96fdbb07bf1a55c39994b50e72`;
- source ref `refs/heads/main`;
- SLSA predicate `https://slsa.dev/provenance/v1`;
- SPDX predicate `https://spdx.dev/Document/v2.3`.

Independent verification after the run also passed for both digests. Cosign used the stricter exact certificate identity `https://github.com/MichaelSeveen/atlas/.github/workflows/release.yml@refs/heads/main` and issuer `https://token.actions.githubusercontent.com`; separate GitHub CLI calls verified both predicate types against the exact repository, workflow, source digest, and source ref.

## Retained release SBOMs

Artifact `8547437658`, named `atlas-sboms-9761754709a09c96fdbb07bf1a55c39994b50e72`, was retained for 90 days. The GitHub artifact API reported 40,216 compressed bytes, archive digest `sha256:ab8768a345df55677d7efda1d7b50383e6783dc088eb91775cc5661865b72391`, and expiry `2026-10-20T22:54:32Z`. Download and JSON parsing verified four SPDX 2.3 documents:

| File | Bytes | SHA-256 | Document identity | Packages |
|---|---:|---|---|---:|
| `backend-image.spdx.json` | 201,613 | `a2f3bd4da43b23ed8b58179df37a2a15b31fb05a1c038f1572f439619808e154` | `ghcr.io/michaelseveen/atlas-backend` | 97 |
| `backend-source.spdx.json` | 110,734 | `a48fc7af7c0f7eda7ac9c613dc265db8aed7cd76e61cbd90b75d2c7befd1d7a8` | `.` | 60 |
| `frontend-source.spdx.json` | 3,853 | `1b93856e5528ad6106c06e311e2c2c514c5cf768251d098d258418c9f7c56538` | `atlas-frontend-9761754709a09c96fdbb07bf1a55c39994b50e72` | 8 |
| `web-image.spdx.json` | 100,175 | `472308f6d320e42f6080e9f6b035e905e01f68dff21561629742b1ac3ae3066a` | `ghcr.io/michaelseveen/atlas-web` | 20 |

The artifact is time-limited CI retention, not the sole durable proof: this record preserves its source, archive identity, document identities, and individual document hashes so it can be regenerated and compared from the exact revision.

## Reproduce and inspect

```powershell
gh run view 29964442782 --json status,conclusion,headSha,jobs,url
gh run view 29964442782 --log
gh api repos/MichaelSeveen/atlas/actions/runs/29964442782/artifacts
gh run download 29964442782 --name atlas-sboms-9761754709a09c96fdbb07bf1a55c39994b50e72
```

For each digest, run Cosign verification with the exact main-workflow certificate identity and GitHub attestation verification twice, once for SLSA v1 and once for SPDX 2.3, binding `--repo MichaelSeveen/atlas`, `--signer-workflow MichaelSeveen/atlas/.github/workflows/release.yml`, `--source-digest 9761754709a09c96fdbb07bf1a55c39994b50e72`, and `--source-ref refs/heads/main`.

## Sanitization and remaining scope

This record contains only public revisions, workflow/artifact identities, immutable OCI and file digests, compressed sizes, bounded durations, package counts, expiry, and pass/failure markers. It contains no token, credential, connection string, customer data, identity data, product payload, raw scanner database, certificate body, or complete service log.

This closes the hosted-release gaps for FND-023 and FND-024 and strengthens FND-010/FND-022 at the current synthetic Phase 00 depth. It does not satisfy the six separately recorded partial requirements, cross an ADR 0012 trigger, or support a claim that all of Phase 00 is complete.
