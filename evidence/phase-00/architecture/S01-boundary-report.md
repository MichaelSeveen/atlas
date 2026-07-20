# S01 repository and boundary verification

## Evidence metadata

| Field | Value |
|---|---|
| Evidence ID | `EVD-P00-S01-001` |
| Phase / slice | Phase 00 / S01 — Versioned repository and process-boundary scaffold |
| Result | PASS with pre-commit limitation |
| Created | 2026-07-20, Africa/Lagos |
| Actor/tool | Codex using repository-owned PowerShell and Go checks |
| Requirements | `FND-001`, `FND-002`, `FND-003`; preserves `FND-004`; advances `FND-054` |
| Threats | `THR-013`, `THR-025`, `THR-040`, `THR-042`, `THR-054`, `THR-060` |
| Controls | `GOV-01`, `GOV-02`, `SDLC-01..04`, `OPS-01`, `REL-02` |
| Source branch | `main` |
| Source revision | `UNBORN` — Git is initialized but no commit was authorized |
| PRD baseline | `docs/atlas-prd/MANIFEST.sha256` SHA-256 `A9AD9C85D6BE582DE3CB8765AD11962A6845A062C0829FBA1984B8A76D2414DB` |
| Fixture/seed | Isolated forbidden import from transfer to `atlas/internal/ledger/persistence` |

## Scope verified

- Roadmap-aligned ownership directories exist and persist through tracked placeholder files.
- `docs/atlas-prd/` remains canonical; all 58 PRD manifest entries match.
- Each retained root PRD duplicate is either absent or byte-identical to its canonical source; all eleven present copies matched.
- Go module path is the documented reversible local value `atlas`.
- Go 1.25.7, Node 24.13.0, and pnpm 11.10.0 match repository pins.
- pnpm accepts the empty workspace lockfile in frozen, offline, no-script mode.
- `cmd/api`, `cmd/worker`, and `cmd/simulator` are separate, inert Go entry points and build successfully.
- The clean Go source tree has zero dependency-boundary violations.
- A temporary test fixture that imports another context's persistence package produces exactly one expected violation.
- A temporary unregistered `internal/debugtools` module is rejected, preventing undeclared modules/processes from bypassing the policy registry.
- No product endpoint, job, provider scenario, frontend package/UI, schema, broker, identity, telemetry, or financial behavior was introduced.

## Reproduction

Primary command:

```powershell
pwsh -NoProfile -File ./scripts/verify-s01.ps1
```

The command verifies exact tool pins and valid Git metadata, then runs:

```powershell
go test ./...
go build ./cmd/api ./cmd/worker ./cmd/simulator
go test ./internal/architecture `
  -run 'TestArchitectureBoundaries|TestBoundaryCheckerRejectsForbiddenImport|TestBoundaryCheckerRejectsUnregisteredModule|TestImportRules|TestRepositoryLayout|TestCanonicalPRDDuplicates|TestCanonicalPRDManifest' `
  -count=1 -v
pnpm install --lockfile-only --offline --ignore-scripts --frozen-lockfile
```

Observed summary:

```text
atlas/cmd/api                no test files; build passed
atlas/cmd/simulator          no test files; build passed
atlas/cmd/worker             no test files; build passed
atlas/internal/architecture  PASS
TestArchitectureBoundaries                 PASS
TestBoundaryCheckerRejectsForbiddenImport  PASS
TestBoundaryCheckerRejectsUnregisteredModule PASS
TestImportRules                            PASS (9 cases)
TestRepositoryLayout                      PASS
TestCanonicalPRDDuplicates                PASS
TestCanonicalPRDManifest                  PASS
toolchain_go=1.25.7
toolchain_node=24.13.0
toolchain_pnpm=11.10.0
source_revision=UNBORN
s01_verification=PASS
pnpm frozen offline lockfile verification: Done using pnpm v11.10.0
```

The first direct `go test` attempt inside the restricted workspace could not write the standard user-profile Go cache. The final repository-owned command sets `GOCACHE` to ignored workspace path `.tmp/go-build`, was rerun without relying on that external cache, and passed. This is an execution-environment constraint, not a product-test failure.

## Expected and observed results

| Check | Expected | Observed |
|---|---|---|
| Git metadata | Valid worktree, branch `main`, no automatic commit | Valid; `HEAD` is unborn |
| Repository layout | Every S01 canonical/ownership path exists; forbidden shared-domain directories absent | PASS |
| Canonical PRD | 58/58 manifest entries match | PASS |
| Retained root duplicates | No retained copy differs from canonical PRD source | 11/11 matched |
| Clean import graph | Zero violations | 0 violations |
| Seeded forbidden import | Exactly one cross-context private-import violation | 1 expected violation; test PASS |
| Seeded unregistered module | Exactly one unregistered-module violation | 1 expected violation; test PASS |
| Process packages | API, worker, simulator compile independently | PASS |
| Toolchain pins | Observed tools exactly match pins | PASS |
| Empty pnpm lockfile | Frozen/offline/no-script verification succeeds without dependency resolution | PASS |
| Product behavior | None introduced | Static review: entry points contain empty `main` functions and no imports |

The normalized dependency result is in [S01-dependency-graph.txt](S01-dependency-graph.txt).

## Integrity digests

```text
90221261CDDB6274A1D6BDAD3EFFE43229E8D2A528332AF28314DC2323A51CF7  evidence/phase-00/architecture/S01-dependency-graph.txt
3C006B494961794FE7D16592532070AFFB814660CBAF78C8B9606539D8F39F0A  go.mod
DDADD0412DE5E5AD594167772B9514DFA04CC323212E5D0C865901F9C035E9EF  package.json
80BD29BA7260AA173C0F7A94C1F1E094017DE662BFBEB828B9788F8F1BDC6128  internal/architecture/checker.go
5934F027C7180450A5DD18BE70791CD1B9480F5454E361AA5FEDBD8BFA6EA2D7  internal/architecture/checker_test.go
A6AB6655FA66F26D4316501A744AD047077CCE13FDE7C3961C7B39B0ADD10A37  scripts/verify-s01.ps1
A9AD9C85D6BE582DE3CB8765AD11962A6845A062C0829FBA1984B8A76D2414DB  docs/atlas-prd/MANIFEST.sha256
```

The report digest is stored in the adjacent `S01-boundary-report.sha256` sidecar. No artifact is signed because signing/provenance belongs to `FND-024`/S07.

## Sanitization and data statement

The evidence contains no customer, merchant, workforce, financial, provider, identity, token, credential, or real-person data. The forbidden-import fixture contains source paths only. A basic repository secret-pattern scan is part of handoff validation; full history scanning cannot run before a commit/history exists.

## Limitations and revalidation

- `UNBORN` is not a source revision. This report cannot prove a clean clone, immutable history, CODEOWNERS enforcement, or revision-bound provenance.
- All files are uncommitted. The user must separately approve a first commit; S01 must then be rerun and a new evidence version created rather than overwriting this report.
- The checker enforces Go source imports. PostgreSQL write ownership remains `FND-060`/S05; CI-required enforcement remains `FND-020`/S07.
- Cross-context root/application imports are permitted interfaces, not permission to write another context's tables.
- The toolchain pins match the implementation host; no long-term-support, vulnerability-free, portability, or supply-chain claim is made.
- The entry points only prove source/process separation and buildability. They do not prove runtime lifecycle, health, configuration, identity, telemetry, or operability.
- The retained root PRD duplicates are still a drift risk despite the hash guard.
- Revalidate after any toolchain, module-path, directory-ownership, import-rule, canonical PRD, or entry-point change, and no later than the first commit.

No content item was published from this evidence.
