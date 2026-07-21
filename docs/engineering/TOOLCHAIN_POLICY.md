# Atlas toolchain policy

## Active pins

| Tool | Repository pin | Source |
|---|---|---|
| Go language baseline | `1.25.0` | `go.mod` |
| Go toolchain | `1.25.7` | `go.mod`, `.go-version` |
| PostgreSQL Go driver | `github.com/jackc/pgx/v5` `5.10.0` | `go.mod`, `go.sum` |
| Telemetry API/SDK/exporters | `go.opentelemetry.io/otel` family `1.43.0` | `go.mod`, `go.sum` |
| Frontend package/runtime tool | Bun `1.3.0` | `apps/web/package.json`, `apps/web/Containerfile` |
| Frontend framework | React and React DOM `19.2.7` | `apps/web/package.json`, `apps/web/bun.lock` |

The pins match the reviewed S01/S04/S05/S06 implementation environments; they are reproducibility choices, not compatibility or support-lifetime claims. CI, SBOM, provenance, immutable image digest, and automated update verification remain outstanding under `FND-020..024` and `FND-054`.

## Frontend framework decision

React + TypeScript is the sole frontend choice under `FND-004`. ADR 0009 selects Bun as the only package manager, test runner, bundler, and frontend server runtime. `bun.lock` is versioned; competing lockfiles and script runtimes are rejected by `TestFrontendToolchainPolicy`. Do not add Node.js, pnpm, npm, Yarn, Vue, or a second frontend application toolchain without an owner-approved superseding ADR.

All project React components must be function components using hooks where state or lifecycle behavior is needed. Class declarations are forbidden in `apps/web/src`; `TestFrontendUsesFunctionComponentsOnly` scans the tree and a seeded negative proves the policy fails closed. React 19 root error callbacks provide the current feature-free shell fallback without a class component.

## Go module identity

The module path is `github.com/MichaelSeveen/atlas`, derived from configured origin `https://github.com/MichaelSeveen/atlas.git`. If repository ownership or location changes, update the module directive and internal imports together in one reviewed mechanical change, rerun the full boundary/build suite, and supersede the affected evidence. Do not guess a replacement identity before its remote exists.

## Dependency policy

- Pin Go dependencies when introduced; `go.mod`/`go.sum` changes receive dependency review. S05 pins pgx/v5 for bounded PostgreSQL readiness; S06 pins the official stable OpenTelemetry trace/metric SDK and OTLP gRPC exporters for the feature-free observability path.
- Pin all frontend dependencies exactly and update `package.json` with `bun.lock`; do not add a competing package manager or unbounded range.
- Run `bun run --cwd apps/web typecheck`; the pinned TypeScript and React declarations must pass before frontend tests or builds.
- Pin container base images and CI actions by immutable version/digest when those artifacts are introduced in S07.
- React + TypeScript remains the sole frontend choice. S04 implements only actor route shells and synthetic environment state.

## Update procedure

1. Open a dedicated change naming affected requirements and `THR-013`/`THR-060`.
2. Review release notes, compatibility, vulnerability, license, and provenance implications.
3. Change the repository pins and lockfiles together; do not mutate generated lockfiles by hand once dependencies exist.
4. Run `pwsh -NoProfile -File ./scripts/verify-s01.ps1` plus all later-phase CI lanes that exist.
5. Record old/new versions, commands, evidence, known regressions, and rollback/forward-fix plan.
6. Roll back by restoring the prior reviewed pins/lockfiles if no schema/contract incompatibility exists; otherwise use an explicit forward-fix.

Review normal updates at least monthly once dependency-bearing code exists. Security emergency updates bypass cadence, not review, testing, evidence, or compatibility analysis. The vulnerability and dependency emergency-update runbooks are established under `FND-055`; S07 must connect them to dependency/scanner/SBOM/provenance gates.
