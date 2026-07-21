# S04 synthetic environment report

- **Evidence ID:** EVD-P00-S04-001
- **Created:** 2026-07-21
- **Source revision:** `UNCOMMITTED_WORKTREE(base=c327135)`
- **Requirements:** FND-004, FND-010 through FND-013, FND-030 through FND-033, FND-054
- **Threats:** THR-019, THR-020, THR-023 through THR-025, THR-029, THR-036, THR-042, THR-043, THR-045 through THR-047, THR-059, THR-060
- **Named tests:** most-agents-skip #5 hostile production-reference configuration; #6 logout/back-forward cache exposure
- **Seed:** `atlas-phase00-foundation-v1`; SHA-256 `d350303e1f1129c41ba96af24ea33315fb1739f721b6bc7b7bcf96c774debe4d`
- **Revalidate by:** 2026-08-21 or any source, lockfile, configuration, image-tag, browser, or runtime-provider change

## Result

S04 is implemented and pre-commit verified as a feature-free synthetic foundation. The complete Compose topology reached ready with PostgreSQL, Redis, bounded NATS JetStream, MinIO, the OpenTelemetry Collector, three imported Keycloak realms, API, worker, simulator, and React web. All exposed ports were loopback-bound. Memory/PID limits applied to all containers; API/worker/simulator/web ran non-root with read-only roots.

The API exposed only `/health/live`, `/health/ready`, and `/version`. Worker and simulator performed configuration validation and process lifecycle only. No product endpoint, schema, broker stream, identity exchange, provider behavior, financial state, or wallet UI was introduced.

## Reproduction

```powershell
pwsh -NoProfile -File ./scripts/verify-s04.ps1
pwsh -NoProfile -File ./scripts/s04.ps1 -Action Up
pwsh -NoProfile -File ./scripts/verify-s04.ps1 -Live
pwsh -NoProfile -File ./scripts/s04.ps1 -Action Restart
pwsh -NoProfile -File ./scripts/s04.ps1 -Action Reset -Confirmation 'RESET ATLAS LOCAL'
```

The verifier replays S01–S03, checks formatting/build/vet/layout and every Go package, performs frozen Bun install/test/build, validates all four environments and the seed checksum, and kills wildcard production-reference, wrong-target reset, production-reference reset, and unknown-tenant seed canaries. The live smoke checks every exposed dependency/process surface without returning topology through public health responses.

## Observed proof

- Configuration set: 4; strict decode/set validation passed.
- Frozen frontend: Bun 1.3.0; React/React DOM 19.2.7; two tests passed; browser bundle built.
- Named test #5 rejected wildcard origin, local signing reference, mock mode, real endpoint, non-synthetic service, and plaintext production-reference origin.
- Credential references were unique across four environments. Five generated password/token fingerprints per local/test environment were all nonmatching; values were never retained.
- Seed inventory: two tenants, three users, two feature-free account identity fixtures, eight scenario IDs, fixed time/checksum; the unknown-tenant mutation was killed.
- Reset rejected wrong environment and production-reference while preserving the target, then exact contained local reset passed.
- Live smoke passed before and after a controlled volume-preserving restart.
- Real Edge browser proof found the persistent synthetic banner, zero console errors, no browser storage/cookies, and kept Back on `/signed-out` after logout.

Supporting artifacts: [live smoke](S04-live-smoke.txt), [browser report](S04-browser-report.txt), [configuration digests](S04-config-digests.txt), [restart failure/recovery](S04-restart-failure-and-recovery.txt), and [sanitized screenshot](S04-customer-shell.png).

## Failures retained and controls changed

The first Go run caught a container recipe incorrectly named with a `.go` suffix; renaming it restored clean package scanning. Initial browser proof exposed a favicon 404; the foundation server now handles it with a no-content route and the final console is clean. The provider's in-place Compose restart raced dependency ordering; the supported `Restart` action now performs ordered, volume-preserving down/up and recovered all ten services.

## Sanitization and limitations

Only synthetic labels, hashes, references, statuses, and observed image digests were handled as evidence. A diagnostic Compose render displayed the first generated synthetic-local credential set in transient tool output; that entire set was immediately rotated before any container start and is absent from repository evidence. No runtime environment file or credential value is versioned.

This report is pre-commit and must be followed by revision-bound verification. The current Windows Podman machine had never started, lacked systemd configuration, and had no host Compose provider; verification repaired the disposable VM and ran the equivalent Compose provider inside it. Therefore `FND-010` remains partial pending exact clean-machine execution of the repository wrapper in S08. `FND-011` remains partial because fixtures are validated catalogues, not loaded schemas or executable provider contracts. `FND-031` remains partial because staging/production provisioning, rotation, restore, and secret-manager evidence are absent. Image tags/digests observed here are not S07 SBOM, signing, provenance, vulnerability, or immutable-promotion proof.
