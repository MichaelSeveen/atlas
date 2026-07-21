# Atlas synthetic local environment

S04 supplies a feature-free, synthetic-only environment for Phase 00. It starts PostgreSQL, Redis, NATS JetStream, MinIO, an OpenTelemetry Collector, Keycloak, the API, worker, simulator, and React web shell. S05 adds only the `atlas_foundation` migration/permission/recovery control schema and real database probes. It does not create product schemas, process financial commands, integrate a real identity/provider service, or seed balances.

## Pinned tools and services

The repository builds Go with 1.25.7 and the React application with Bun 1.3.0. React and React DOM are pinned to 19.2.7 in `apps/web/package.json`; `bun.lock` is the only frontend lockfile. No Node.js runtime or pnpm project tooling is used.

The local stack uses exact image tags for PostgreSQL 18.4 Alpine, Redis 8.2.7 Alpine, NATS 2.14.0 Alpine, MinIO `RELEASE.2025-07-23T15-54-02Z`, OpenTelemetry Collector 0.155.0, and Keycloak 26.7.0. S07 will add immutable-digest promotion, SBOMs, provenance, and vulnerability policy; S04 must not claim those controls.

## Prerequisites

- Go 1.25.7 and Bun 1.3.0.
- Podman 5.x or Docker with a Compose v2-compatible provider.
- At least 6 GiB memory available to the container runtime and ports 13000, 14222, 15432, 16379, 18080, 18081, 18222, 19000, and 19001 free on loopback. Port 25432 must also be free during the isolated recovery drill.

All published ports bind to `127.0.0.1`. The generated runtime credential file is mode-restricted, lives below ignored `.tmp/environments/local`, contains random synthetic-local values, and is never evidence. Configuration and evidence expose only secret references or fingerprints. Podman is the default; pass `-ContainerRuntime docker` to the same commands on a Docker Compose host.

## Reversible commands

```powershell
pwsh -NoProfile -File ./scripts/s04.ps1 -Action Up
pwsh -NoProfile -File ./scripts/s04.ps1 -Action Status
pwsh -NoProfile -File ./scripts/s04.ps1 -Action Smoke
pwsh -NoProfile -File ./scripts/s04.ps1 -Action Restart
pwsh -NoProfile -File ./scripts/s04.ps1 -Action Down
```

`Up` validates all four environment files and the seed manifest, prepares local-only credentials, builds the processes, starts the complete stack, waits for readiness, and runs the live smoke. Repeating `Up` reuses the contained local credential set and named volumes. `Restart` performs an ordered down/up cycle without `--volumes`; this avoids dependency-order races in provider-specific in-place restart commands. `Down` stops the stack without deleting state.

Destructive reset is limited to local/test state and requires the exact resolved phrase:

```powershell
pwsh -NoProfile -File ./scripts/s04.ps1 -Action Reset -Confirmation 'RESET ATLAS LOCAL'
```

Wrong or missing confirmation, staging, production-reference, and targets escaping the state root fail before state deletion. Reset removes only the `atlas-local` Compose volumes and `.tmp/environments/local` runtime state.

## Configuration and deterministic fixtures

`deploy/environments/` contains closed, typed local, test, staging, and production-reference documents. Unknown fields, wildcard origins, real/public endpoints, non-synthetic services, environment-shared credential references, expired/incomplete flags, mock mode outside local/test, and insecure production-reference surfaces are rejected.

`deploy/seeds/foundation.json` has a fixed virtual time, two synthetic tenants, three synthetic users, two account identity fixtures with no financial state, and eight named provider scenario identifiers. S04 validates identity, ownership, closed fields, and checksum only. Loading application tables and executing provider behavior are deliberately deferred until schemas and provider contracts exist; therefore `FND-011` remains partial.

## React route shells

The single React application owns separate `/customer`, `/merchant`, and `/workforce` route shells. Each renders the persistent environment/synthetic banner and only foundation copy. Client fixture state is in memory; logout clears it, moves to `/signed-out`, and blocks protected shells during back/forward navigation. No token, identity session, financial data, or product capability is stored or rendered.

## Verification and recovery

```powershell
pwsh -NoProfile -File ./scripts/verify-s04.ps1
pwsh -NoProfile -File ./scripts/verify-s04.ps1 -Live
```

The first command covers S01–S04 build, layout, Go/React tests, configuration, seed, reset, and seeded failure canaries. `-Live` additionally requires an already-running stack and checks every exposed foundation surface. Use the dependency-specific runbooks under `docs/runbooks/`; topology details never appear in public health responses.

S05 database and recovery commands are documented separately in [the database foundation](DATABASE_FOUNDATION.md). They preserve the S04 stack and never restore over the active PostgreSQL volume.
