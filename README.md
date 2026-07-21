# Atlas

Atlas is a security-first, multi-currency wallet and financial-operations portfolio system. The repository is in [Phase 00 — Secure engineering foundation](docs/atlas-prd/02-phases/PHASE-00_ENGINEERING_FOUNDATION.md); it contains no product or financial behavior yet.

## Canonical specification

`docs/atlas-prd/` is the only authoritative PRD root. The eleven historical root-level copies were removed in an owner-authorized cleanup after all eleven were reverified byte-identical to their canonical files. The architecture test now fails if any root duplicate reappears.

## Toolchain policy

- Go `1.25.7`, with module path `github.com/MichaelSeveen/atlas`, is the backend toolchain.
- React `19.2.7` + TypeScript is the sole frontend framework; Bun `1.3.0` is its pinned package manager, test runner, bundler, and server runtime.
- `bun.lock` is the only frontend lockfile. The repository has no Node.js runtime or pnpm project tooling.
- React source uses function components and hooks only; the architecture suite rejects every class declaration under `apps/web/src`.

The Go pin and update procedure are documented in [Toolchain policy](docs/engineering/TOOLCHAIN_POLICY.md). The module path is derived from the configured GitHub origin and must change with it in one reviewed mechanical change before public packages or contracts depend on a different identity.

## S01 commands

```powershell
pwsh -NoProfile -File ./scripts/verify-s01.ps1
go test ./...
go build ./cmd/api ./cmd/worker ./cmd/simulator
go test ./internal/architecture -run TestBoundaryCheckerRejectsForbiddenImport -count=1 -v
```

S01 created three independently buildable, initially inert process entry points. S03 later activated only the API operational foundation; no product or financial behavior was added.

## S02 commands

```powershell
pwsh -NoProfile -File ./scripts/verify-s02.ps1
go test ./internal/platform/... -count=1
go test ./internal/architecture -count=1
pwsh -NoProfile -File ./scripts/test-s02-mutation.ps1
```

S02 adds only narrow Go primitives for integer money/currency, opaque IDs, UTC clocks, actor/correlation context, and stable safe errors, plus static bans on floating-point money and direct domain `time.Now()`. See [platform primitives and static policy](docs/engineering/PLATFORM_PRIMITIVES.md). It adds no Node.js/package-manager toolchain or product behavior.

## S03 commands

```powershell
pwsh -NoProfile -File ./scripts/verify-s03.ps1
go test ./cmd/api/... ./tests/contract -count=1
pwsh -NoProfile -File ./scripts/test-s03-contract-canary.ps1
```

S03 makes the API process serve only `GET /health/live`, `GET /health/ready`, and `GET /version`. The executable is live but deliberately not ready until later slices provide real dependency and migration probes. HTTP limits, exact-origin CORS, safe problems, validated request/correlation/trace context, and a closed trace/metric seed are documented in [the HTTP foundation](docs/engineering/HTTP_FOUNDATION.md). No database, product endpoint, authentication, runtime telemetry exporter, worker job, simulator scenario, or UI was added.

## S04 commands

```powershell
pwsh -NoProfile -File ./scripts/s04.ps1 -Action Up
pwsh -NoProfile -File ./scripts/s04.ps1 -Action Smoke
pwsh -NoProfile -File ./scripts/s04.ps1 -Action Restart
bun run --cwd apps/web typecheck
pwsh -NoProfile -File ./scripts/verify-s04.ps1 -Live
pwsh -NoProfile -File ./scripts/s04.ps1 -Action Down
pwsh -NoProfile -File ./scripts/s04.ps1 -Action Reset -Confirmation 'RESET ATLAS LOCAL'
```

S04 adds the complete synthetic local/reference topology, four closed environment configurations, guarded reset, deterministic identity/scenario fixtures, and customer/merchant/workforce React route shells. It remains feature-free: no product endpoint, database schema, financial behavior, broker stream, identity exchange, or wallet UI exists. See [the local environment guide](docs/engineering/LOCAL_ENVIRONMENT.md).

## S05 commands

```powershell
pwsh -NoProfile -File ./scripts/s05.ps1 -Action Up
pwsh -NoProfile -File ./scripts/s05.ps1 -Action Migrate
pwsh -NoProfile -File ./scripts/s05.ps1 -Action Verify
pwsh -NoProfile -File ./scripts/s05.ps1 -Action BackupRestore
pwsh -NoProfile -File ./scripts/verify-s05.ps1
pwsh -NoProfile -File ./scripts/verify-s05.ps1 -Live
pwsh -NoProfile -File ./scripts/s05.ps1 -Action Down
```

S05 adds only feature-free PostgreSQL migration, role, readiness, and recovery controls. Real role denials, empty/previous migration lanes, checksum canaries, bounded lock failure, physical backup, WAL archive, and an isolated point-in-time restore are reproducible. The local backup volume is not encrypted, so recovery mechanics pass while `FND-064` remains partial. See [the database foundation](docs/engineering/DATABASE_FOUNDATION.md).

## S06 commands

```powershell
pwsh -NoProfile -File ./scripts/verify-s06.ps1
pwsh -NoProfile -File ./scripts/verify-s06.ps1 -Live -ContainerRuntime podman
pwsh -NoProfile -File ./scripts/s06.ps1 -Action Down -ContainerRuntime podman
```

S06 adds only the operating baseline: closed source-redacted JSON logs, bounded OTLP traces/metrics, an exported API/readiness/database golden trace, metric/alert catalog checks, an executable Phase 00 threat model, a provider-neutral versioned-secret boundary, and incident runbooks. Telemetry failure does not determine readiness. Queue/retry metrics remain definition-only because no job or broker flow exists, and no product behavior, identity exchange, managed secret provider, or wallet UI is added. See the [S06 evidence](evidence/phase-00/observability-security/S06-observability-security-report.md).

## Repository boundaries

- `cmd/` owns process composition only.
- `internal/<context>/` owns domain behavior and persistence for that bounded context.
- Cross-context imports may target only the other context's package root or `application` API.
- Cross-context imports of persistence, store, repository, database, SQL, or private internal packages are forbidden.
- `internal/platform/` and `internal/architecture/` cannot import domain contexts.
- Shared `common`, `shared`, or `models` domain packages are forbidden.
- `contracts/` is reserved for future generated/published artifacts. The sole mutable HTTP/event contracts remain under `docs/atlas-prd/03-contracts/`; a second hand-edited copy is forbidden.

See [AGENTS.md](AGENTS.md), [implementation status](docs/engineering/IMPLEMENTATION_STATUS.md), and the [Phase 00 plan](docs/engineering/PHASE-00-PLAN.md) before making changes.
