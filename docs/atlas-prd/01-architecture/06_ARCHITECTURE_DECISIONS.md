# Architecture decisions index

Architecture decisions are not hidden in code or social posts. Each major choice records context, alternatives, trade-offs, security impact, operational impact, and reversal strategy.

## Accepted baseline ADRs

| ADR | Decision | Core rationale |
|---|---|---|
| 0001 | Modular monolith first | preserve local financial transactions and reduce distributed failure surface |
| 0002 | PostgreSQL as financial source of truth | mature transactions, constraints, locking, recovery, and queryability |
| 0003 | Transactional outbox and idempotent consumers | durable state/event coupling with at-least-once delivery |
| 0004 | BFF and server-side browser sessions | keep tokens out of JavaScript storage and centralize web security |
| 0005 | No cardholder data | preserve project focus and avoid false PCI posture |
| 0006 | Integer minor-unit money | prevent floating-point financial errors and clarify currency semantics |
| 0007 | Deterministic provider simulators | reproduce edge cases without real financial integration |
| 0008 | Reversible synthetic local/reference platform | exercise complete boundaries without selecting production dependencies |
| 0009 | React route shells with Bun | preserve one frontend choice and safe actor-shell boundaries without Node.js/pnpm project tooling |
| 0010 | Native PostgreSQL migration and recovery controls | keep released SQL reviewable, runtime roles schema-inert, and PITR drills isolated before product data exists |
| 0011 | GitHub Actions and keyless release integrity | bind reviewed source, immutable image digests, SBOMs, signatures, and provenance without a long-lived signing key |
| 0012 | Solo-maintainer sensitive-change governance | preserve honest synthetic-only progress with protected automated gates and explicit triggers for future independent review |

See `06-governance/adrs/`.

## ADRs required before implementation decisions

- transaction isolation and lock order for ledger posting;
- production broker selection and delivery semantics;
- production identity-provider deployment and realm operating model;
- field-level encryption and search strategy;
- audit tamper-evidence design;
- object storage retention/immutability;
- row-level security as defense in depth;
- deployment platform and secret management;
- generated API client strategy;
- reconciliation rule versioning;
- database partitioning threshold;
- analytics/read-replica boundary;
- key rotation and webhook signature algorithm;
- feature-flag safety for financial state transitions.

## Decision quality test

An ADR is incomplete if it says only “we chose X because it is scalable.” It must answer:

1. What concrete problem exists now?
2. What invariants or constraints dominate the choice?
3. What alternatives were considered?
4. What new failure modes does the choice introduce?
5. How will the decision be verified?
6. What metrics would cause reconsideration?
7. How can the system migrate away safely?
