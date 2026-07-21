# ADR 0009 — React route shells use the pinned Bun toolchain

- **Status:** Accepted
- **Date:** 2026-07-21
- **Owners:** Platform owner
- **Related requirements/threats:** FND-004, FND-032, FND-054; THR-029, THR-045, THR-046, THR-047, THR-060

## Context

Atlas selected React and TypeScript once, while the owner requires the repository to avoid Node.js and pnpm project tooling. Phase 00 needs customer, merchant, and workforce shells, persistent synthetic labels, safe browser state clearing, and a reversible toolchain before product UI exists.

## Decision

Use one React 19.2.7 application with three explicit route shells and Bun 1.3.0 as its package manager, test runner, bundler, and server runtime. Version `bun.lock`; reject competing lockfiles and script runtimes. Keep identity realm, request-client, cache, and authorization boundaries explicit even though the route shells share one deployable application.

The shells contain only foundation state. Use no browser token storage, clear in-memory query state on logout, fail closed on protected back/forward navigation, serve no-store responses with strict security headers, and render the non-production synthetic banner persistently.

## Consequences

- The repository remains React + TypeScript and Go, with one pinned frontend toolchain.
- One application minimizes premature deployment complexity while route boundaries remain testable.
- A later split into separately deployed actor applications remains possible if measured security or delivery needs justify it.
- OpenAPI client generation and compatibility enforcement remain S07 work; S04 must not hand-invent product calls.

## Migration and rollback/exit strategy

Restore the prior reviewed `package.json` and `bun.lock` together to roll back a toolchain/dependency change. Route shells can move behind separate entry points without changing product contracts because S04 has none. A replacement package/runtime requires a superseding ADR and owner review.

## Verification and evidence

Run frozen dependency install, React unit/build tests, route/banner live smoke, browser logout/back-forward proof, security-header scan, and the static toolchain policy. Revisit versions monthly once S07 dependency automation exists.
