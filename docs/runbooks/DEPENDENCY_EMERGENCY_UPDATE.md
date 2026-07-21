# Dependency emergency update

Owner: security owner and module owner. Trigger: confirmed exploited vulnerability, malicious package/release, revoked publisher, or toolchain compromise affecting an Atlas dependency.

1. Record advisory identifier, affected package/version/path, discovery time, severity and exploitability evidence, affected revisions/images, and decision owner. Use a synthetic reproduction; do not execute untrusted proof-of-concept code on a trusted workstation.
2. Contain affected builds and deployments. Preserve lockfiles, module graph, image digests, logs, and provenance. If exploitation is plausible, follow incident handling and rotate exposed secrets.
3. Prefer the smallest supported fixed version. Verify the upstream source, signed release/checksum where available, changelog, transitive graph, license, and Go/Bun baseline compatibility.
4. Update the canonical manifest/lockfile in a focused branch. Do not disable static bans, tests, checksum verification, TLS, or vulnerability policy to force an update.
5. Run all phase verification plus the affected package's focused tests, seeded negatives, local build, container build, and relevant live smoke. Compare API/contracts, migration manifests, images, and telemetry.
6. Obtain security and module-owner review. Deploy with a rollback revision; validate build markers and health before widening.
7. Coordinate private internal notification and any external advisory through the vulnerability-disclosure procedure. Do not announce an unverified vulnerability or fixed version.
8. Add revision-aware, sanitized evidence and update the threat/risk/claim registers. Record any accepted residual exposure with an owner and expiry.
9. Hold a security/module-owner retrospective and add the prevention test, scanner/pin policy change, or dependency-removal decision needed to prevent recurrence.

If no safe fixed version exists, remove/disable the affected capability if one exists. Phase 00 has no product capability to preserve at the cost of an unsafe dependency.
