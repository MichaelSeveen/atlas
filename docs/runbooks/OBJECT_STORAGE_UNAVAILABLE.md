# Local object storage unavailable

Scope: synthetic S04 MinIO only. No application objects or retention controls exist yet.

1. Confirm API liveness and failing readiness.
2. Check the MinIO container health and bounded logs without printing its credentials.
3. Restore with `scripts/s04.ps1 -Action Up`, then verify `/minio/health/live` through the smoke command.
4. Reset only with the exact local confirmation if disposable object state must be removed.

S04 does not prove production retention, immutability, backup, restore, malware scanning, or signed object access.
