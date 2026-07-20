# S03 HTTP and trace foundation

S03 activates only three unauthenticated operational endpoints in the `api` process. It implements no product endpoint, authentication, authorization, tenancy, financial behavior, database access, broker, provider integration, or UI.

## Canonical contract and route inventory

The sole mutable HTTP contract remains `docs/atlas-prd/03-contracts/openapi.yaml`. The `contracts/openapi/` directory is reserved for a future generated/published artifact and must not contain a second hand-edited contract.

| Route | Contract |
|---|---|
| `GET /health/live` | Process liveness only. It never checks dependencies. |
| `GET /health/ready` | Returns `200` only when the injected dependency and migration-state checker is ready. Every failure is the same topology-free `DEPENDENCY_DEGRADED` problem. |
| `GET /version` | Returns only source revision, canonical contract version, and UTC build time. Build metadata is validated at startup. |

Unknown, debug, and test routes return a fixed problem response. The deployed route inventory test permits only these three paths. Product paths in the reference contract remain intentionally unimplemented until their owning phases.

## HTTP safety policy

- Default listener: `127.0.0.1:8080`, overridable by `ATLAS_HTTP_ADDR` without being echoed in errors.
- Read-header/read/write/idle timeouts: 5s/10s/15s/60s. Maximum request headers: 16 KiB.
- Foundation routes accept no request body or query string. The generic body guard is 1 MiB, detects both declared and streamed overflow, and rejects compressed bodies without decompression.
- Responses set `no-store`, CSP, HSTS, MIME, frame, referrer, permissions, and cross-origin-resource protections.
- CORS defaults to no allowed browser origin. Configured origins must be exact canonical HTTP(S) origins; wildcard origins are always rejected and credentials are never combined with `*`.
- Every ordinary response contains validated opaque request/correlation IDs. Duplicate, malformed, wrong-prefix, or unsafe client values are replaced and are never authorization inputs.
- All handled failures use bounded RFC 9457-style problem JSON with stable catalogued codes and no stack, database, migration, hostname, or panic detail.

The request-body rule is intentionally stricter than future product handlers. A later contract-authorized mutation endpoint must opt into its own schema/content-type/decompression/depth limits and tests; it must not weaken these operational routes.

## Trace and metric seed

The middleware validates W3C `traceparent`, preserves only a valid trace ID/parent relationship, creates a new server span ID, and replaces malformed context. The S03 recorder interface accepts a closed span structure containing only fixed route/name/outcome, hex trace IDs, opaque request/correlation IDs, status, and UTC times. The metric seed likewise accepts only fixed route, method, status, and duration.

The production default recorders discard data because no collector/exporter is selected yet. The golden synthetic test installs an in-memory recorder and proves `GET /health/ready` produces a linked server span plus readiness child span. This is a propagation/instrumentation seed, not a claim that runtime observability, alerting, or `FND-042` is complete.

## Build and readiness posture

The executable supports Go link-time injection into `main.sourceRevision`, `main.contractVersion`, and `main.buildTime`. Defaults are safe development metadata. The contract version must equal `2026-07-20`, source revision must be `development` or lowercase hexadecimal, and build time must parse as RFC 3339.

No database exists in S03. The real executable therefore starts live but deliberately not ready. S04/S05 must inject a real dependency/migration checker; bypassing readiness to make a deployment green is prohibited. See the [database unavailable runbook](../runbooks/DATABASE_UNAVAILABLE.md).

## Reproduction

```powershell
pwsh -NoProfile -File ./scripts/verify-s03.ps1
```

The verifier replays S01/S02, builds all processes, runs focused OpenAPI and live-handler conformance, security/resource/deadline suites, the named migration-lag test, a bounded metadata fuzz campaign, and a seeded contract-path mutation that must be killed.
