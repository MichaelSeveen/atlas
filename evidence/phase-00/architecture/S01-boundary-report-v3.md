# S01 repository and boundary verification — v3

## Evidence metadata

| Field | Value |
|---|---|
| Evidence ID | `EVD-P00-S01-003` |
| Phase / slice | Phase 00 / S01 — Versioned repository and process-boundary scaffold |
| Result | PASS |
| Created | 2026-07-20, Africa/Lagos |
| Actor/tool | Codex using repository-owned PowerShell and Go checks |
| Requirements | `FND-001`, `FND-002`, `FND-003`; preserves `FND-004`; advances `FND-054` |
| Threats | `THR-013`, `THR-025`, `THR-040`, `THR-042`, `THR-054`, `THR-060` |
| Controls | `GOV-01`, `GOV-02`, `SDLC-01..04`, `OPS-01`, `REL-02` |
| Source branch | `main` |
| Verified source revision | `f72f5468c52a05a442fa0efbbe996fa16450a2bb` |
| Remote identity | `https://github.com/MichaelSeveen/atlas.git` |
| PRD baseline | `docs/atlas-prd/MANIFEST.sha256` SHA-256 `42838F6F25F9DA9305DAA18340340342AC1CE211F7E67A783F5F393D35C3DC62` |
| Fixture/seed | Isolated forbidden import from transfer to `github.com/MichaelSeveen/atlas/internal/ledger/persistence` |
| Supersedes | `EVD-P00-S01-002`; the remote now supplies the Go module identity and a source commit exists |

The v1 and v2 reports and their digests remain unchanged as historical pre-commit evidence. This report verifies the immutable initial scaffold revision. It is stored in the subsequent evidence-only commit because a commit cannot contain a report that already knows that same commit's identifier.

## Scope verified

- `.` is the Git worktree root, branch is `main`, and the verified source revision was clean.
- Origin is `https://github.com/MichaelSeveen/atlas.git`; Go module path is `github.com/MichaelSeveen/atlas`.
- Roadmap-aligned ownership directories exist and persist through tracked placeholder files.
- `docs/atlas-prd/` remains canonical; all 58 PRD manifest entries match.
- All eleven retained root PRD duplicates are byte-identical to their canonical sources.
- Go 1.25.7 matches the repository pin; the Go language baseline is 1.25.0.
- React + TypeScript is the sole frontend framework decision. No frontend runtime, dependency manifest, package manager, dependency lock, or build tool is active in S01.
- `cmd/api`, `cmd/worker`, and `cmd/simulator` are separate, inert Go entry points and build successfully.
- The clean Go source tree has zero dependency-boundary violations.
- A temporary fixture importing another context's persistence package produces exactly one expected violation.
- A temporary unregistered `internal/debugtools` module is rejected.
- No product endpoint, job, provider scenario, frontend package/UI, schema, broker, identity, telemetry, or financial behavior was introduced.

## Reproduction

```powershell
pwsh -NoProfile -File ./scripts/verify-s01.ps1
```

The command verifies the exact Go pin and Git metadata, uses ignored workspace-local Go build/module caches, then runs:

```powershell
go test ./...
go build ./cmd/api ./cmd/worker ./cmd/simulator
go test ./internal/architecture `
  -run 'TestArchitectureBoundaries|TestBoundaryCheckerRejectsForbiddenImport|TestBoundaryCheckerRejectsUnregisteredModule|TestImportRules|TestRepositoryLayout|TestCanonicalPRDDuplicates|TestCanonicalPRDManifest' `
  -count=1 -v
```

Observed summary:

```text
github.com/MichaelSeveen/atlas/cmd/api                no test files; build passed
github.com/MichaelSeveen/atlas/cmd/simulator          no test files; build passed
github.com/MichaelSeveen/atlas/cmd/worker             no test files; build passed
github.com/MichaelSeveen/atlas/internal/architecture  PASS
TestArchitectureBoundaries                   PASS
TestBoundaryCheckerRejectsForbiddenImport    PASS
TestBoundaryCheckerRejectsUnregisteredModule PASS
TestImportRules                              PASS (9 cases)
TestRepositoryLayout                         PASS
TestCanonicalPRDDuplicates                   PASS
TestCanonicalPRDManifest                     PASS
toolchain_go=1.25.7
frontend_framework=React+TypeScript
frontend_build_toolchain=DEFERRED
source_revision=f72f5468c52a05a442fa0efbbe996fa16450a2bb
s01_verification=PASS
```

## Expected and observed results

| Check | Expected | Observed |
|---|---|---|
| Git metadata | Valid worktree, branch `main`, immutable source commit | PASS; `f72f5468c52a05a442fa0efbbe996fa16450a2bb` |
| Remote/module identity | Origin and Go module path agree | PASS |
| Repository layout | Every S01 canonical/ownership path exists; forbidden shared-domain directories absent | PASS |
| Canonical PRD | 58/58 manifest entries match | PASS |
| Retained root duplicates | No retained copy differs from canonical PRD source | 11/11 matched |
| Clean import graph | Zero violations | 0 violations |
| Seeded forbidden import | Exactly one cross-context private-import violation | 1 expected violation; test PASS |
| Seeded unregistered module | Exactly one unregistered-module violation | 1 expected violation; test PASS |
| Process packages | API, worker, simulator compile independently | PASS |
| Go toolchain pin | Observed tool exactly matches the repository pin | PASS |
| Frontend policy | React + TypeScript only; build toolchain deferred | PASS |
| Product behavior | None introduced | Static review: entry points contain empty `main` functions and no imports |

The normalized dependency result is in [S01-dependency-graph-v3.txt](S01-dependency-graph-v3.txt).

## Integrity digests

```text
9CDF582C5E42794B966582B67517EE49CC8864784BFB01E3C4011B6808BD2BA2  evidence/phase-00/architecture/S01-dependency-graph-v3.txt
33689B28A4E2DC7D95566E7D631D5DFD7FFA124149B5AA45369A5F2743352BCF  go.mod
2BEB5CC8C3EE44C563EE5DB610056331F7DDF81B1484426738B1D0DF476BDE61  .go-version
D710DA273A61C5821911922B8C5AAFAF31E1A06983B07C683746E8BAC9526C78  internal/architecture/checker.go
3DCD433161D07B46579BFBDF08BAB16FE236D031A9617568A9CAC79E2138C2F1  internal/architecture/checker_test.go
253BAFDE55BD48244E1090DB3BABD47AF866512DFA5BA07DD4BA82612CACF990  scripts/verify-s01.ps1
42838F6F25F9DA9305DAA18340340342AC1CE211F7E67A783F5F393D35C3DC62  docs/atlas-prd/MANIFEST.sha256
```

The report digest is stored in the adjacent `S01-boundary-report-v3.sha256` sidecar. No artifact is signed because signing/provenance belongs to `FND-024`/S07.

## Sanitization and data statement

The evidence contains no customer, merchant, workforce, financial, provider, identity, token, credential, or real-person data. The forbidden-import fixture contains source paths only. Basic current-tree and initial-history secret-pattern scans found no candidates; a dedicated history scanner remains S07 work.

## Limitations and revalidation

- This report verifies source revision `f72f5468c52a05a442fa0efbbe996fa16450a2bb`; the subsequent evidence-only commit stores the report and traceability updates.
- A remote clean-clone run and hosted branch-protection/CODEOWNERS proof remain future work.
- React itself is not installed. Its runtime, dependency manifest, exact versions, lockfile, package manager, build system, and UI implementation require a separately authorized frontend slice.
- The checker enforces Go source imports. PostgreSQL write ownership remains `FND-060`/S05; CI-required enforcement remains `FND-020`/S07.
- Cross-context root/application imports are permitted interfaces, not permission to write another context's tables.
- The Go pin matches the implementation host; no long-term-support, vulnerability-free, portability, or supply-chain claim is made.
- The entry points only prove source/process separation and buildability. They do not prove runtime lifecycle, health, configuration, identity, telemetry, or operability.
- The retained root PRD duplicates are still a drift risk despite the hash guard.
- Revalidate after any toolchain, module-path, directory-ownership, import-rule, canonical PRD, or entry-point change.

No content item was published from this evidence.
