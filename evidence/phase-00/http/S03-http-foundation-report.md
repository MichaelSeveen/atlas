# S03 HTTP-foundation verification report

## Evidence identity

| Field | Value |
|---|---|
| Evidence version | `S03-http-foundation-report` v1 |
| Verification date / revalidation date | `2026-07-20` / revalidate after every source, contract, toolchain, edge-policy, or readiness change |
| Requirements | `FND-003`, `FND-040` (API-edge facet), `FND-043` (in-memory seed), `FND-053` |
| Threats | `THR-009`, `THR-010`, `THR-015`, `THR-025`, `THR-030`, `THR-041`, `THR-042`, `THR-043`, `THR-060` |
| Adversarial cases | named Phase 00 skipped test #3; `ADV-RES-001` applicable body-size facet; slow partial header; hostile metadata; wildcard credentialed CORS; debug-route inventory |
| Base revision | `710bca0a3c5dd44fb009512e2200a65b5da59dcd` |
| Source revision | `UNCOMMITTED_WORKTREE(base=710bca0a3c5dd44fb009512e2200a65b5da59dcd)` |
| Go toolchain | language baseline `1.25.0`; toolchain `1.25.7` |
| Toolchain scope | Go only; no Node.js runtime or package manager added |
| Canonical contract | `docs/atlas-prd/03-contracts/openapi.yaml` |

The source-revision value is intentionally not a commit claim. S03 was implemented and verified after committed baseline `710bca0a3c5dd44fb009512e2200a65b5da59dcd`, and this turn did not authorize creating or pushing an S03 commit. Re-run the full verifier and add a new post-commit evidence version if the owner later authorizes a commit; do not rewrite this report as though it had been commit-bound at creation.

## Scope and results

S03 adds one feature-free synchronous foundation path: canonical OpenAPI → API HTTP server → injected dependency/migration readiness state → safe response and linked in-memory trace seed. It activates only `GET /health/live`, `GET /health/ready`, and `GET /version`. It adds no product endpoint, authentication, authorization, tenancy, financial behavior, schema, database, broker, provider call, worker job, simulator scenario, UI, Node.js toolchain, or runtime telemetry exporter.

| Proof | Expected | Observed |
|---|---|---|
| `go test ./...` | All repository, architecture, platform, HTTP, and focused contract tests pass; canonical manifest remains valid. | PASS |
| `go build ./cmd/api ./cmd/worker ./cmd/simulator` | All three feature-free processes build independently. | PASS |
| `TestFoundationEndpointContract` | The three handlers match closed safe response bodies and headers. | PASS |
| `TestLiveServerSmokeHealthyAndMigrationBehind` | A real loopback HTTP server serves live/ready/version; readiness is `200` when healthy and `503` when migration state is behind while live/version remain `200`. | PASS |
| Actual `cmd/api` loopback smoke | The real process wiring serves live/version, defaults readiness to a generic `503`, applies `no-store`, and stops cleanly after the smoke check. | PASS; see `S03-live-smoke.txt`. |
| `TestMigrationLagFailsReadinessOnly` | Phase 00 skipped test #3 fails readiness without topology disclosure and preserves liveness. | PASS; see `S03-migration-lag.txt`. |
| HTTP security/resource suite | Exact CORS, cache/security headers, declared and streamed body limits, compression/query/body rejection, fixed problems, route inventory, panic safety, bounded readiness context, and slow partial headers behave as specified. | PASS; see `S03-header-scan.txt`. |
| `TestGoldenSyntheticTraceAndBoundedMetrics` | Valid inbound context links unique server/readiness spans; fields and metric labels are closed and safe. | PASS; see `S03-golden-trace.txt`. |
| `FuzzUntrustedRequestMetadata`, `-fuzztime=100x` | Seed corpus and 100 bounded executions retain valid response IDs/trace context without reflection or panic. | PASS; see `S03-fuzz-summary.txt`. |
| `scripts/test-s03-contract-canary.ps1` | Removing `/health/ready` from a disposable contract copy makes the focused contract test fail. | PASS; mutant KILLED; see `S03-contract-canary.txt`. |
| `pwsh -NoProfile -File ./scripts/verify-s03.ps1` | Replay S01/S02, builds, S03 suites, fuzz, and canary from one repository-owned command. | PASS. |

## Contract, security, and failure-boundary proof

- The sole mutable HTTP contract remains canonical under `docs/atlas-prd/`. S03 added the three operational paths before/with their implementation and corrected invalid opaque-ID examples to the pre-existing Crockford alphabet. The contract test rejects a mutable root/`contracts/` duplicate and kills a missing-readiness-path mutation.
- Liveness checks only that the process can answer. Readiness requires both dependency and migration-state booleans, receives a bounded context, fails closed, and returns one generic `DEPENDENCY_DEGRADED` problem. The real executable deliberately supplies an unready checker until S04/S05 can supply real probes.
- Version exposes exactly source revision, contract version, and UTC build time. Startup rejects malformed source revision, mismatched contract version, missing build time, unsafe body limits, invalid CORS origins, and missing readiness configuration.
- Every response is `no-store` and receives CSP, HSTS, MIME, frame, referrer, permissions, and cross-origin-resource protections. Foundation routes reject any query, non-empty or oversized body, and non-identity content encoding without attempting decompression.
- Client request/correlation IDs are accepted only as one valid opaque value with the required prefix. W3C trace context is accepted only in the supported lowercase version-00 form. Duplicate/malformed values are replaced, never reflected, and never used for authorization.
- The route inventory contains only the three operational paths. Unknown/debug paths, wrong methods, unsafe CORS preflights, resource violations, identifier degradation, readiness failure, and panics return fixed catalogued problems without stack, database, migration, hostname, secret, or attacker detail.

## Limitations and follow-up

1. Evidence is based on an uncommitted worktree. No S03 commit or remote push is claimed.
2. No database or migration system exists. The readiness interface and migration-behind state are deterministic test seams, not evidence of a real PostgreSQL probe, connection pool, migration table, or production deployment.
3. A readiness checker receives a timeout context and future real probes must honor it. S03 does not claim that arbitrary injected code which ignores cancellation can be forcibly terminated.
4. The trace/metric recorders default to discard because no collector/exporter, structured logger, dashboard, alert, or runtime telemetry configuration exists. `FND-040` and `FND-043` remain partial; `FND-041` and `FND-042` remain absent.
5. The golden trace spans only API server → readiness check in memory. Web, database, transaction, outbox, event, worker, simulator, retry, causation, collector, and exported-trace proof remain S04/S06/S08.
6. `ADV-RES-001` coverage is limited to rejecting declared/streamed oversized bodies and all non-empty operational-route bodies before JSON parsing. No product JSON schema or depth parser exists in this feature-free slice.
7. The OpenAPI tests are focused source/contract checks, not a full standards-compliant OpenAPI semantic linter or breaking-change baseline. Full contract tooling and protected CI remain `FND-027`/S07.
8. HSTS and the other handler headers must be revalidated through the eventual TLS terminator/reverse proxy and reference deployment. Future product handlers require their own schema, media-type, depth, decompression, authorization, and abuse tests.
9. S03 is requirement-scoped only. It does not complete Phase 00 or authorize S04.

## Reproduction, integrity, and sanitization

From repository root:

```powershell
pwsh -NoProfile -File ./scripts/verify-s03.ps1
```

Integrity digests are recorded below after the final traceability and manifest update:

```text
e64701a88e0321e18e3cc9e6dbb5c647f156fbc80a08f974f7750705a0d44fe6  docs/atlas-prd/MANIFEST.sha256
f7fb4fe38f9183724a156a922200f23db9802fa08ef54f091076413cc09419eb  evidence/phase-00/http/S03-header-scan.txt
a3d127675b90c81f71ba552fa495cdfa7c62a72f11b13cbb261b21a69b152dd9  evidence/phase-00/http/S03-migration-lag.txt
9e01799aa89b2ffe44fb01ab7021e14ff97fd7ebfb37b135d261acf8e1e3579e  evidence/phase-00/http/S03-golden-trace.txt
37f2a52d747c2bdf485efebce225755d5cfaf30f729f13b565584aadcaa5745d  evidence/phase-00/http/S03-contract-canary.txt
b0501ede6cc9eeaafe428ecb1f9343025302ecb4160bfdfb7b4f406dfda23f55  evidence/phase-00/http/S03-fuzz-summary.txt
33b7ddf6f3851e5147108203132b78ab579c68a9ed662c7791c1d2ea7fb1267f  evidence/phase-00/http/S03-live-smoke.txt
```

Evidence contains only synthetic opaque identifiers, standard trace examples, route names, fixed statuses, public repository metadata, and tool versions. The verification used no secrets, credentials, tokens, customer records, real service endpoints, production payloads, or personal data. Panic/detail canaries use reserved synthetic example values and are asserted absent from responses and trace fields.
