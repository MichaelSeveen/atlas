# CI, contract, and supply-chain boundary

## Scope

S07 is a feature-free Phase 00 control slice. It adds no endpoint, event, schema, identity exchange, broker stream, financial behavior, worker job, or wallet UI. The only mutable contracts remain `docs/atlas-prd/03-contracts/openapi.yaml` and `asyncapi.yaml`.

## Lanes

| Lane | Trigger | Required behavior |
|---|---|---|
| PR static | pull request and manual | build, vet, tests, targeted race, Bun frozen install/lint/test/build, contract lint/conformance/examples, migration manifest, complete-history secrets, Gosec, dependency vulnerability checks |
| PR CodeQL | pull request | independent Go and TypeScript taint/static analysis with retained SARIF |
| PR integration | pull request | real PostgreSQL/NATS and empty/previous migration lanes through S05 |
| PR supply chain | pull request | four SPDX SBOMs, critical-vulnerability gate, denied-license gate, source-revision image tags/digests, non-root/read-only runtime proof |
| Nightly | schedule/manual | S07 supply-chain plus S06 live trace/metric/collector-outage proof |
| Release | protected `main`/version tag manual or tag push | full live/history/supply/clean-clone S08 preflight on the fresh hosted runner before registry authentication; GHCR images tagged by the full source revision; digest-only signing/attestation; SPDX attestation; signature and GitHub provenance verification |

The versioned workflows are not proof that GitHub enforces them. Under [ADR 0012](../atlas-prd/06-governance/adrs/0012-solo-maintainer-sensitive-change-governance.md), `main` must have a branch ruleset requiring pull requests, all PR jobs, conversation resolution, deletion/non-fast-forward protection, and no unrecorded bypass. While the closed synthetic solo policy is active, required human approvals remain zero because the only owner cannot independently approve their own work; sensitive PRs instead require the machine-checked declaration and fresh-context self-review. This is an accepted deviation, not independent-review evidence. Capture the ruleset identifier and successful PR run before marking hosted enforcement verified. A genuine code-owner approval becomes mandatory before any policy revalidation trigger.

## Contract policy

`contractctl lint` checks exact OpenAPI/AsyncAPI versions, YAML parsing, non-empty roots, internal-only references, and reference resolution. `contractctl compare` rejects removed OpenAPI paths/methods/responses/schema fields and removed AsyncAPI channels/operations/messages/schema fields. PR comparison reads the base revision directly from Git and never stores another editable contract. Additive changes still require compatibility and owner review. Product operations described by the canonical planning contract are not implemented by this slice.

## Artifact policy

External images use tag-plus-digest references from `deploy/images.lock.json`. Local/CI builds use full Git SHA tags and OCI revision labels; dirty local proof is marked `UNCOMMITTED_WORKTREE(base=...)`. Release publication, SBOMs, signatures, provenance, and verification address `name@sha256:digest`, never a mutable tag. Grype blocks critical findings; other findings remain visible in the retained JSON report and require normal triage. AGPL/SSPL licenses fail the automated gate; all other new or unknown licenses remain review items.

Keyless OIDC identity fails closed. Signing or attestation outage stops the release. There is no unsigned fallback and no long-lived project signing key. An attestation establishes artifact origin and build metadata, not a claim that the artifact is secure, vulnerability-free, or compliant.

The release job is ref-guarded to `main` or `v*` tags. Its S08 preflight includes `-Live -History -SupplyChain -CleanClone -ContainerRuntime docker` and must finish before Buildx setup, GHCR authentication, image push, signing, or attestation. A workflow file containing those steps is not publication evidence; retain the exact successful hosted run and immutable digests separately.

## Reproduce

```powershell
pwsh -NoProfile -File ./scripts/verify-s07.ps1
pwsh -NoProfile -File ./scripts/test-solo-maintainer-governance.ps1
pwsh -NoProfile -File ./scripts/verify-s07.ps1 -History
pwsh -NoProfile -File ./scripts/verify-s07.ps1 -History -SupplyChain -ContainerRuntime podman
pwsh -NoProfile -File ./scripts/test-s07-contract-compatibility.ps1 -BaseRef HEAD
```

`-History` downloads hash-verified scanners and runs the disposable deleted-history secret canary. `-SupplyChain` builds local images and writes disposable reports under `.tmp/s07-reports`; it does not publish or sign them.
