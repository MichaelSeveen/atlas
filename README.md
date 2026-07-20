# Atlas

Atlas is a security-first, multi-currency wallet and financial-operations portfolio system. The repository is in [Phase 00 — Secure engineering foundation](docs/atlas-prd/02-phases/PHASE-00_ENGINEERING_FOUNDATION.md); it contains no product or financial behavior yet.

## Canonical specification

`docs/atlas-prd/` is the only authoritative PRD root. The eleven historical root-level copies are non-authoritative and remain untouched pending an explicit owner decision. The architecture test fails if any retained copy drifts from its canonical source.

## Toolchain policy

- Go `1.25.7`, with module path `github.com/MichaelSeveen/atlas`, is the only active build toolchain.
- React + TypeScript is the sole selected frontend framework; `apps/web/` is intentionally only an ownership placeholder in S01.
- No frontend runtime, package manager, dependency manifest, or build tool is selected or pinned yet. That decision is deferred until frontend implementation is authorized.

The Go pin and update procedure are documented in [Toolchain policy](docs/engineering/TOOLCHAIN_POLICY.md). The module path is derived from the configured GitHub origin and must change with it in one reviewed mechanical change before public packages or contracts depend on a different identity.

## S01 commands

```powershell
pwsh -NoProfile -File ./scripts/verify-s01.ps1
go test ./...
go build ./cmd/api ./cmd/worker ./cmd/simulator
go test ./internal/architecture -run TestBoundaryCheckerRejectsForbiddenImport -count=1 -v
```

The three process entry points are deliberately inert and independently buildable. They do not expose endpoints, connect to infrastructure, or implement domain behavior.

## Repository boundaries

- `cmd/` owns process composition only.
- `internal/<context>/` owns domain behavior and persistence for that bounded context.
- Cross-context imports may target only the other context's package root or `application` API.
- Cross-context imports of persistence, store, repository, database, SQL, or private internal packages are forbidden.
- `internal/platform/` and `internal/architecture/` cannot import domain contexts.
- Shared `common`, `shared`, or `models` domain packages are forbidden.
- `contracts/` is reserved for implementation publication/generation. Until S03 defines promotion, canonical HTTP/event contracts remain under `docs/atlas-prd/03-contracts/`.

See [AGENTS.md](AGENTS.md), [implementation status](docs/engineering/IMPLEMENTATION_STATUS.md), and the [Phase 00 plan](docs/engineering/PHASE-00-PLAN.md) before making changes.
