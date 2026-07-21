# Local identity provider unavailable

Scope: synthetic S04 Keycloak realm imports only; Atlas has no identity integration or session exchange yet.

1. Confirm API liveness and failing readiness without exposing the dependency name to the public response.
2. Inspect the Keycloak container state and bounded logs. Do not expose bootstrap credentials.
3. Restore with `scripts/s04.ps1 -Action Up` and verify the customer, merchant, and workforce realm discovery documents.
4. If disposable state is inconsistent, use the exact guarded local reset and recreate the stack.

Never bypass realm separation, create a custom IdP, or treat local synthetic users as authorization evidence.
