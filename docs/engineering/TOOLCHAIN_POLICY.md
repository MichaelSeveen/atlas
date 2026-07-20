# Atlas toolchain policy

## Active S01 pin

| Tool | Repository pin | Source |
|---|---|---|
| Go language baseline | `1.25.0` | `go.mod` |
| Go toolchain | `1.25.7` | `go.mod`, `.go-version` |

The Go pin matches the toolchain available when S01 was implemented; that is a reproducibility choice, not a compatibility or support-lifetime claim. CI/container verification remains outstanding under `FND-020` and `FND-054`.

## Frontend framework decision

React + TypeScript is the sole frontend choice under `FND-004`. No frontend runtime, package manager, dependency manifest, or build tool is active through S02. `apps/web/` is an ownership placeholder only. Select the smallest suitable frontend build toolchain when frontend implementation is separately authorized; update this policy, the verification command, dependency controls, and evidence in that slice.

## Go module identity

The module path is `github.com/MichaelSeveen/atlas`, derived from configured origin `https://github.com/MichaelSeveen/atlas.git`. If repository ownership or location changes, update the module directive and internal imports together in one reviewed mechanical change, rerun the full boundary/build suite, and supersede the affected evidence. Do not guess a replacement identity before its remote exists.

## Dependency policy

- Pin Go tool dependencies when they are introduced; `go.mod`/`go.sum` changes receive dependency review.
- Pin frontend dependencies exactly once the frontend build toolchain is deliberately selected; do not add competing package managers or unbounded ranges for build/security-critical tools.
- Pin container base images and CI actions by immutable version/digest when those artifacts are introduced in S07.
- React + TypeScript remains the sole frontend choice. S01 implements no UI.

## Update procedure

1. Open a dedicated change naming affected requirements and `THR-013`/`THR-060`.
2. Review release notes, compatibility, vulnerability, license, and provenance implications.
3. Change the repository pins and lockfiles together; do not mutate generated lockfiles by hand once dependencies exist.
4. Run `pwsh -NoProfile -File ./scripts/verify-s01.ps1` plus all later-phase CI lanes that exist.
5. Record old/new versions, commands, evidence, known regressions, and rollback/forward-fix plan.
6. Roll back by restoring the prior reviewed pins/lockfiles if no schema/contract incompatibility exists; otherwise use an explicit forward-fix.

Review normal updates at least monthly once dependency-bearing code exists. Security emergency updates bypass cadence, not review, testing, evidence, or compatibility analysis. The vulnerability and dependency emergency-update runbooks remain S06/S07 work under `FND-055`.
