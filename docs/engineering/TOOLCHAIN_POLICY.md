# Atlas toolchain policy

## Active pins

| Tool | Repository pin | Source |
|---|---|---|
| Go language baseline | `1.25.0` | `go.mod` |
| Go toolchain | `1.25.12` | `go.mod`, `.go-version`; security update from `1.25.7` after S07 Govulncheck findings |
| PostgreSQL Go driver | `github.com/jackc/pgx/v5` `5.10.0` | `go.mod`, `go.sum` |
| Telemetry API/SDK/exporters | `go.opentelemetry.io/otel` family `1.43.0` | `go.mod`, `go.sum` |
| Frontend package/runtime tool | Bun `1.3.0` | `apps/web/package.json`, `apps/web/Containerfile` |
| Frontend framework | React and React DOM `19.2.7` | `apps/web/package.json`, `apps/web/bun.lock` |
| CI actions | reviewed versions plus immutable commit SHAs | `.github/actions-lock.json`, `.github/workflows/` |
| External images | exact tags plus registry manifest digests | `deploy/images.lock.json`, Containerfiles, `deploy/local/compose.yaml` |
| Security/supply-chain tools | Gitleaks `8.28.0`, Gosec `2.25.0`, Syft `1.44.0`, Grype `0.112.0`, Cosign `3.0.6` | `tools/supply-chain.lock.json` with per-platform SHA-256 |
| Go advisory tool | Govulncheck `1.1.4` | `tools/go-tools.lock.json`; Go checksum database/module verification applies |

The pins match the reviewed S01–S07 implementation environments; they are reproducibility choices, not compatibility or support-lifetime claims. Local policy and generation are reproducible; hosted required-check, signature, and provenance proof remains pending until the GitHub workflows execute under the selected OIDC identity.

## Frontend framework decision

React + TypeScript is the sole frontend choice under `FND-004`. ADR 0009 selects Bun as the only package manager, test runner, bundler, and frontend server runtime. `bun.lock` is versioned; competing lockfiles and script runtimes are rejected by `TestFrontendToolchainPolicy`. Do not add Node.js, pnpm, npm, Yarn, Vue, or a second frontend application toolchain without an owner-approved superseding ADR.

All project React components must be function components using hooks where state or lifecycle behavior is needed. Class declarations are forbidden in `apps/web/src`; `TestFrontendUsesFunctionComponentsOnly` scans the tree and a seeded negative proves the policy fails closed. React 19 root error callbacks provide the current feature-free shell fallback without a class component.

## Go module identity

The module path is `github.com/MichaelSeveen/atlas`, derived from configured origin `https://github.com/MichaelSeveen/atlas.git`. If repository ownership or location changes, update the module directive and internal imports together in one reviewed mechanical change, rerun the full boundary/build suite, and supersede the affected evidence. Do not guess a replacement identity before its remote exists.

## Dependency policy

- Pin Go dependencies when introduced; `go.mod`/`go.sum` changes receive dependency review. S05 pins pgx/v5 for bounded PostgreSQL readiness; S06 pins the official stable OpenTelemetry trace/metric SDK and OTLP gRPC exporters for the feature-free observability path.
- Pin all frontend dependencies exactly and update `package.json` with `bun.lock`; do not add a competing package manager or unbounded range.
- Run `bun run --cwd apps/web typecheck`; the pinned TypeScript and React declarations must pass before frontend tests or builds.
- Pin container base images by exact tag plus registry digest and CI actions by immutable commit SHA. The adjacent human-readable version is review metadata, never the executable reference.
- Downloaded release tools must come from the reviewed HTTPS URL and match the platform SHA-256 in `tools/supply-chain.lock.json`; raw remote install scripts are forbidden.
- Dependabot opens bounded weekly changes for Go modules, Bun, GitHub Actions, and Containerfiles. It proposes changes; it does not bypass owner review, compatibility, security, or evidence gates.
- React + TypeScript remains the sole frontend choice. S04 implements only actor route shells and synthetic environment state.

## Update procedure

1. Open a dedicated change naming affected requirements and `THR-013`/`THR-060`.
2. Review release notes, compatibility, vulnerability, license, and provenance implications.
3. Change the repository pins and lockfiles together; do not mutate generated lockfiles by hand once dependencies exist.
4. Run `pwsh -NoProfile -File ./scripts/verify-s07.ps1 -History -SupplyChain` plus relevant live lanes. Verify tool/archive hashes, SBOM contents, license findings, vulnerability results, image identities, and contract compatibility.
5. Record old/new versions, commands, evidence, known regressions, and rollback/forward-fix plan.
6. Roll back by restoring the prior reviewed pins/lockfiles if no schema/contract incompatibility exists; otherwise use an explicit forward-fix.

Review normal updates at least monthly. The lock `reviewed_at` value records review, not an automatic trust extension. Security emergency updates bypass cadence, not review, testing, evidence, compatibility analysis, signing, or provenance. Use the dependency-emergency runbook, rebuild from reviewed source, revoke/retire affected digests, and publish a new digest; never overwrite prior evidence.
