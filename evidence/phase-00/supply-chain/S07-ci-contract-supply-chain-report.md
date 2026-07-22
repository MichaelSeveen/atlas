# S07 CI, contract, and supply-chain report

- **Evidence ID:** EVD-P00-S07-001
- **Created:** 2026-07-22T12:09:54Z
- **Last verified:** 2026-07-22T12:47:02Z
- **Source revision:** `UNCOMMITTED_WORKTREE(base=3342b4ded1cd62fab1223372cd5129f272889878)`
- **Base tree:** `36e8d4b1195ec3c8e8bf0bbfdef294f1df523005`
- **Requirements:** satisfied FND-022, FND-027, and FND-054; partial FND-020, FND-023, FND-024, and FND-026; preserves/enforces FND-006, FND-021, and FND-025
- **Threats:** THR-009, THR-013, THR-014, THR-018, THR-019, THR-020, THR-030, THR-040, THR-042, THR-054, and THR-060
- **Named skipped tests:** #1, deleted-history secret; #7, old/new API and event compatibility
- **Synthetic seeds:** mutable GitHub Action reference; removed OpenAPI path/field; removed AsyncAPI message/field; unresolved internal reference; deleted synthetic credential in a disposable Git repository
- **Result:** PASS for local S07 verification and artifact mechanics, with hosted enforcement/signature/provenance and pre-commit limitations
- **Revalidate by:** 2026-08-22 or on any workflow, contract, dependency, lockfile, action, image, scanner, signing, CODEOWNERS, migration, or toolchain change

## Scope and boundaries

S07 turns earlier repository-owned commands into versioned GitHub Actions lanes and adds contract compatibility, dependency/history scanning, SBOM, immutable image, ownership, and release-integrity controls. It adds no product endpoint, event, product schema, identity exchange, broker stream, financial behavior, worker job, generated product client, or wallet UI. The only mutable contracts remain the canonical OpenAPI and AsyncAPI under `docs/atlas-prd/03-contracts/`.

There is no new authorization, tenancy, idempotency, concurrency, money, or before/after-commit behavior. CI and artifact controls cannot authorize product operations. Release identity is digest-only and keyless; missing signing or attestation stops the configured release rather than falling back to an unsigned artifact.

## Environment and configuration identity

- Windows host; Go `1.25.12` (`windows/amd64`) with language baseline `1.25.0`; Bun `1.3.0`.
- Container proof used the existing rootful Podman WSL fallback. External and base images are locked by tag plus registry digest in `deploy/images.lock.json`; the web runtime installs exact Alpine `libgcc`/`libstdc++` `15.2.0-r2` and executes Bun `1.3.0` under the hardened runtime flags.
- Hash-verified tools: Gitleaks `8.28.0`, Gosec `2.25.0`, Syft `1.44.0`, Grype `0.112.0`, Cosign `3.0.6`; Govulncheck `v1.1.4` is an exact Go module pin.
- `.github/workflows/pr.yml`: `56b88a985dd124d7a23a451a70b035a0176179dd4d8d69b55043bb36634f96e4`
- `.github/workflows/release.yml`: `89b2f34b7bea897337f4553341dde4b7c5d0e94de4271eb56b7fbb8862892111`
- `.github/actions-lock.json`: `620bead85dcf151e841b3b54578f425c5de44389937475d4dd632a7d07b8880e`
- `deploy/images.lock.json`: `68cd96e33e26410500a7489bc456892dfeba466ec73eb05678a1249e55018850`
- `tools/supply-chain.lock.json`: `ed7cf0d4b372dfda7495ed2165ebb3972985ec135495558232ed09ba006c45df`
- `tools/go-tools.lock.json`: `60ac8f517d0f2c5ea651d088ddf597cff781a428df46d6ae8e70e6b708f45c37`
- `.gitleaks.toml`: `fdc270531ba8bd40016100269d46ff6bb626aed1e5098dc679be72f0b21a3d30`

The scanner binaries, OCI archives, SBOM bodies, Grype JSON, and generated manifest remain disposable ignored files under `.tmp/`. This durable report records only bounded public versions, digests, and results.

## Reproduction

```powershell
pwsh -NoProfile -File ./scripts/verify-s07.ps1
pwsh -NoProfile -File ./scripts/verify-s07.ps1 -History
pwsh -NoProfile -File ./scripts/test-s07-contract-compatibility.ps1 -BaseRef HEAD
pwsh -NoProfile -File ./scripts/verify-s07.ps1 -SupplyChain -ContainerRuntime podman
```

Expected: independent Go entry points and engineering commands build; vet/tests and frozen Bun lint/test/build pass; canonical contracts lint and compare against the Git baseline; real API examples conform; action/image/tool/CODEOWNERS/Dependabot policies pass; seeded action/contract/reference/history failures are killed; full Git history contains no leak; Govulncheck reports no reachable vulnerability; four SPDX SBOMs identify their expected surfaces; denied licenses and critical vulnerabilities block; backend/web images carry source metadata and immutable digests and execute non-root with read-only root, no capabilities, and no-new-privileges; the final web image loads the exact Bun 1.3.0 runtime.

Observed:

```text
s07_live_contract_examples=PASS
s07_contract_base=HEAD
s07_contract_compatibility=PASS
s07_deleted_history_secret_canary=PASS
current worktree scanned; no leaks found
12 commits scanned; no leaks found
Your code is affected by 0 vulnerabilities.
s07_verification=PASS
s07_sbom_surfaces=backend-source,frontend-source,backend-image,web-image
s07_vulnerability_threshold=critical
s07_image_runtime=non-root,read-only,cap-drop,no-new-privileges
s07_supply_chain=PASS
source_revision=UNCOMMITTED_WORKTREE(base=3342b4ded1cd62fab1223372cd5129f272889878)
s07_hosted_enforcement=UNVERIFIED
```

## Artifact and SBOM identity

| Surface | Observed identity | SHA-256 |
|---|---|---|
| Backend image | `localhost/atlas-backend:3342b4ded1cd62fab1223372cd5129f272889878`, user `10001:10001` | `66a671b055f20c0c8def53284adff5ec8a5aff9edadfd85f9cc94efc9d562168` |
| Web image | `localhost/atlas-web:3342b4ded1cd62fab1223372cd5129f272889878`, user `bun` | `0ee72f6d7dd166aee04e96d6ef0be8ffff530cad111259584ff9fc3f51aff905` |
| Backend source SPDX | source tree excluding `.git`, `.tmp`, and frontend install tree | `ec8a85225ee995c6e31a5fcf2d341e7ddd8da3dc305717e26f42223e80a753d9` |
| Frontend source SPDX | exact `package.json`/Bun/React/TypeScript pins | `7d76b555b5d0fa02a61809ceefef1be6b301fa30feed0181355601eb26977994` |
| Backend image SPDX | exported OCI image | `93ca2d8fac47175aa27a366a84f6c2b2678e38a336b590bc9283bd7f9dfb3494` |
| Web image SPDX | exported OCI image | `20ac0da4b994ef8a4ba218bf5f45b42d32baffc574f63f9d06f725e4d9d54601` |

These are local Podman image IDs/digests and disposable SBOM digests for the uncommitted worktree, not published GHCR release digests. A committed hosted release must produce new identities and evidence.

## Requirement results

| Requirement | Observed evidence | State |
|---|---|---|
| FND-020 | PR static/history/contracts, real PostgreSQL/NATS, CodeQL, and supply-chain workflows call repository-owned fail-closed commands | Partial: no successful hosted run or protected required-check evidence; local Windows race/Gosec are unavailable |
| FND-021 | PR integration lane invokes real S05 PostgreSQL/NATS/migration commands | Satisfied and enforced in configuration; hosted execution is pending under FND-020 |
| FND-022 | Four SPDX SBOMs are generated, identity-checked, hashed, license-checked, and Grype-scanned | Satisfied locally; release attachment is pending |
| FND-023 | Backend/web images use pinned bases, source labels, exact revision tags, recorded SHA-256 identity, minimal runtime files, hardened non-root/read-only execution, and an exact Bun runtime canary | Partial: pre-commit source and no published digest promotion |
| FND-024 | Release workflow uses digest-only keyless Cosign plus GitHub provenance/SBOM attestations and verifies expected workflow/repository identity with no unsigned fallback | Partial: no hosted signature, attestation, or tamper/wrong-source transcript exists |
| FND-025 | PR integration lane preserves empty/previous/current migration and real-role proof | Satisfied and enforced in configuration; hosted execution is pending under FND-020 |
| FND-026 | Sensitive paths are exhaustively covered by `CODEOWNERS` and a static policy test | Partial: GitHub ruleset and actual code-owner review enforcement are not observed |
| FND-027 | OpenAPI/AsyncAPI lint, internal-ref resolution, baseline compare, live examples, and breaking fixtures pass | Satisfied for the canonical current contracts |
| FND-054 | Dependency/action/image/tool pins, hash verification, vulnerability/license scans, Dependabot coverage, and normal/emergency update policy pass | Satisfied; monthly review remains an ongoing operational obligation |

## Sanitization, integrity, and limitations

No secret, credential, access token, customer/identity data, connection string, product payload, or raw scanner database is retained. The deleted-history test creates an invalid synthetic credential by concatenation only inside a disposable Git repository and removes that repository after observing the configured failure code. Gitleaks output is fully redacted.

This report is pre-commit. It identifies a committed base plus an uncommitted worktree, so it cannot prove the final S07 tree or a reproducible registry artifact. Rerun all lanes from a clean committed revision and add a new evidence artifact; do not overwrite this report.

GitHub authentication and hosted runs were unavailable during this local slice. Repository workflows and `CODEOWNERS` therefore do not prove branch rules, required checks, review enforcement, keyless identity, registry publication, signature, provenance, or SBOM attestation. FND-020, FND-024, and FND-026 remain partial for that reason; FND-023 also remains partial until a committed registry digest is built and promoted.

The Windows host has CGO disabled, so the local race lane is unavailable. Gosec `2.25.0` does not complete within a bounded Windows analysis; the Linux PR workflow requires full Gosec and independent Go/TypeScript CodeQL, but no hosted pass is claimed. Syft emitted non-fatal Windows temporary-directory cleanup warnings after producing readable hashed SBOMs. The Podman WSL fallback emitted systemd/XDG warnings but completed build, scan, and hardened runtime checks. A clean supported host remains an S08 gate.
