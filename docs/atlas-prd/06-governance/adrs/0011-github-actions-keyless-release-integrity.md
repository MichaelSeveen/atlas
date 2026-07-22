# ADR 0011 — GitHub Actions and keyless release integrity

- **Status:** Accepted
- **Date:** 2026-07-22
- **Owners:** Platform and security owner
- **Related requirements/threats:** FND-020..027; FND-054; THR-013; THR-014; THR-018; THR-019; THR-030; THR-042; THR-060
- **Supersedes/superseded by:** None

## Context

Atlas needs a reproducible pull-request and release boundary before product behavior is added. The configured origin is GitHub, while the repository currently has no hosted CI, code-owner enforcement, artifact identity, SBOM, signature, or provenance mechanism. A signing design must not introduce a long-lived private key into repository or CI secrets.

## Decision drivers

- Fail-closed review and build controls for sensitive paths.
- Immutable, source-revision-bound artifact identity.
- Short-lived identity with independently verifiable evidence.
- Reversible hosting and registry coupling.
- No unsupported SLSA, security, or compliance claim.

## Options considered

### GitHub Actions, GHCR, and keyless Sigstore identity

This aligns with the configured GitHub origin. GitHub OIDC supplies a short-lived workflow identity; image signatures and attestations bind registry digests to workflow/source metadata. The main risks are host policy drift, workflow compromise, public-log disclosure, and GitHub availability.

### Long-lived signing key stored as a repository secret

This is portable but creates key generation, storage, rotation, recovery, environment-separation, and theft risks before Atlas has selected production secret custody. It is rejected for this slice.

### Local-only CI and unsigned artifacts

This is easy to reproduce but cannot enforce pull-request review or establish portable release identity. It is rejected as the release boundary.

## Decision

GitHub Actions is the selected CI host and GHCR is the reference release registry. Workflow actions are pinned to immutable commit SHAs and separately recorded with reviewed versions. Base images are pinned by tag and registry digest. Release images are tagged by the complete source revision, pushed once, and all subsequent SBOM, signature, attestation, and promotion operations use `name@sha256:digest`.

Release workflows use GitHub OIDC with keyless Sigstore signing. `actions/attest` creates digest-bound build-provenance and SPDX SBOM attestations; Cosign creates a keyless image signature. No long-lived signing key is accepted. Verification constrains the expected repository/workflow identity and OIDC issuer and includes tamper/wrong-source negative checks.

Pull-request workflows run the repository-owned static, test, race, frontend, contract, migration, secret-history, dependency, license, container, and infrastructure-policy lanes. Main/nightly/release workflows reuse the same entry points. `CODEOWNERS` declares sensitive boundaries; actual approval enforcement requires a protected GitHub branch or ruleset and must be evidenced separately.

## Consequences

### Positive

- Builds and release evidence are bound to reviewed source and immutable digests.
- No reusable signing secret is created.
- Local verification remains repository-owned and can move to another CI host.

### Negative and risks

- Hosted proof depends on GitHub, GHCR, OIDC, and public transparency services.
- Repository files alone cannot prove required-check or code-owner enforcement.
- Keyless identity proves who/what signed an artifact, not that the artifact is secure.
- Cross-platform archive hashes and action SHAs require deliberate reviewed updates.

### Operational/security implications

- Workflow permissions remain minimal and are elevated only by the release job.
- Forked pull requests never receive package-write, attestation, or OIDC permissions.
- Any unavailable signing or attestation step blocks release; unsigned fallback is prohibited.
- SBOMs and scan reports are treated as release artifacts and must contain no credentials.

## Migration and rollback/exit strategy

CI logic stays in versioned PowerShell/Go commands. A future CI host or registry can reuse those checks and publish the same SPDX and digest manifest. Replace the OIDC trust identity and supersede this ADR before migration. Rollback restores previously reviewed action/image/tool locks; released digests and historical attestations are never rewritten.

## Verification and evidence

- `pwsh -NoProfile -File ./scripts/verify-s07.ps1` validates local policy, contracts, canaries, history scanning, and CI-equivalent checks.
- `pwsh -NoProfile -File ./scripts/verify-s07.ps1 -SupplyChain -ContainerRuntime podman` generates/scans SBOMs and verifies non-root, read-only images.
- A successful hosted PR and release run must be attached before FND-020/FND-024/FND-026 are called fully satisfied.
- Review the decision and all pins monthly or after a dependency emergency.
