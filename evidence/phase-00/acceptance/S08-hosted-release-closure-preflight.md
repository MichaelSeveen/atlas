# S08 hosted-release closure preflight

- **Evidence ID:** EVD-P00-S08-005
- **Created:** 2026-07-22T20:18:37Z
- **Last verified:** 2026-07-22T20:59:17Z
- **Implementation revision:** `eae43b62f3e0e3f95a09bff46f1ac73217dde5c3`
- **Exact combined verification revision:** `7b1e28ed8e52bc44d593b6114372c28782c8468a`
- **Earlier focused source:** `UNCOMMITTED_WORKTREE(base=7b2f67053ab2af5ccd51713de4e7935fe2bfad81)`; preserved only as discovery history and superseded by the exact committed combined run above
- **Requirements:** advances FND-010, FND-023, and FND-024 without claiming hosted publication
- **Threats:** THR-013; THR-014; THR-018; THR-019; THR-030; THR-042; THR-060
- **Result:** PASS for local full-stack, recovery, constrained-pool, and bounded web shutdown behavior; PASS for fail-before-publish workflow policy; hosted release remains NOT RUN
- **Revalidate by:** 2026-08-22 or on any web lifecycle, Compose, S08, release-workflow, image, signing, attestation, or runtime change

## Implemented closure controls

- The release job accepts only `main` or `v*` refs and runs `verify-s08.ps1 -Live -History -SupplyChain -CleanClone -ContainerRuntime docker` before Buildx, GHCR authentication, image push, signing, or attestation.
- The static architecture policy rejects a seeded missing-`-Live` preflight and a seeded registry action placed before preflight.
- The Bun server coalesces SIGINT/SIGTERM, stops accepting new work, permits a four-second graceful drain, and then closes active connections.
- S08 teardown now requires the web container to exit zero within the eight-second Compose stop deadline.
- The final web image explicitly includes the lifecycle module; the cold container build caught and corrected the initial missing runtime module before evidence was recorded.

## Local observed results

| Check | Observed result |
|---|---|
| Bun typecheck and tests | PASS; five tests, including graceful, forced, and repeated-signal shutdown cases |
| Architecture policy and seeded release negatives | PASS |
| Exact full S08 command | PASS from clean committed revision `7b1e28ed8e52bc44d593b6114372c28782c8468a`, including history, supply chain, live, and clean-clone modes |
| S07 history and supply chain | PASS; complete-history secret scan, reachable-vulnerability analysis, four SPDX SBOM surfaces, critical-CVE/license policy, and hardened image execution |
| Complete synthetic stack smoke | PASS for API, web shells, Keycloak realm, NATS JetStream, and MinIO |
| Golden trace and collector outage | PASS; fixed API/readiness/database trace and readiness `200` during collector outage |
| Real database/broker lanes | PASS for roles, migrations, empty/previous lanes, long-lock abort, and real NATS |
| Backup/WAL/isolated restore | PASS; base backup 16 seconds and isolated restore 46 seconds |
| Constrained database pool | PASS with one real PostgreSQL connection; local CGO-disabled host cannot supply race proof |
| Web container shutdown | PASS; exit code `0`, observed Compose stop duration `6349ms`, eight-second deadline |
| Clean-clone replay | PASS from exact committed revision `7b1e28ed8e52bc44d593b6114372c28782c8468a` with isolated clone-local Go/Bun caches |
| Full hosted release | NOT RUN; no registry authentication, GHCR artifact, signature, or attestation was created |

## Reproduce

```powershell
bun run --cwd apps/web typecheck
bun test --cwd apps/web
go test ./internal/architecture -count=1
pwsh -NoProfile -File ./scripts/verify-s07.ps1
pwsh -NoProfile -File ./scripts/verify-s08.ps1 -Live -History -SupplyChain -CleanClone -ContainerRuntime podman
```

The final combined S08 command passed from committed revision `7b1e28ed8e52bc44d593b6114372c28782c8468a` before this branch was pushed. Hosted Linux PR checks must now provide race/Gosec/CodeQL and protected-workflow evidence. The release workflow remains untriggered and requires separate authorization.

## Sanitization and limitations

This evidence contains only public source revisions, bounded durations, fixed synthetic identifiers, and pass/absence markers. It contains no token, credential, connection string, customer data, identity document, product payload, raw scanner database, or full service log.

The local Podman WSL host is not an independently administered clean machine. Workflow wiring is not proof that GitHub executed it. FND-010 remains partial pending the fresh hosted live run, while FND-023/FND-024 remain partial pending an explicitly authorized GHCR publication with verified digests, signatures, provenance, and SBOM attestations. Phase 00 completion is not claimed.
