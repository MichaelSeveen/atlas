# Phase XX — [Name]

## Outcome

One sentence describing the durable system capability and user/operational result—not a list of screens.

## Why this phase matters

Explain the financial, security, architectural, or product judgement this phase demonstrates.

## Dependencies

List prerequisite phases, contracts, data, environment, and unresolved decisions.

## Scope

### In scope

- ...

### Explicitly out of scope

- ...

## Actors and authorization

| Actor | Allowed actions | Field/data scope | Required assurance | Approval/purpose |
|---|---|---|---|---|
| ... | ... | ... | ... | ... |

## Domain model and state machines

Define resources, ownership, identifiers, statuses, legal transitions, terminal states, expiry, cancellation, correction, and history.

## Financial effects

For every command:

- holds/reservations;
- journal template and accounts;
- fees/FX/rounding;
- settlement implication;
- reversal/refund/adjustment path;
- reconciliation identity;
- ambiguity behaviour.

## Functional requirements

Use stable IDs:

- `XXX-001` ...
- `XXX-002` ...

## Security and privacy requirements

- Authentication/session/step-up.
- Tenant/object/action/field authorization.
- Maker-checker and execution-time recheck.
- Abuse/rate/resource limits.
- Sensitive data classification, minimization, masking, encryption, retention, logging exclusion.
- SSRF/file/parser/signature/key controls where applicable.
- Audit and non-repudiation boundaries.

## API surface

For each operation document:

- actor and scope;
- request/response schema;
- idempotency and ETag;
- synchronous durable boundary;
- asynchronous/ambiguous states;
- errors/retryability;
- rate/resource limits;
- audit and events.

## Event surface

| Event | Producer transaction | Partition/ordering | Data classification | Consumers | Duplicate/replay behaviour |
|---|---|---|---|---|---|
| ... | ... | ... | ... | ... | ... |

## Frontend requirements

Cover customer, merchant, operations, finance, risk, or support surfaces as applicable. Include loading, empty, stale, restricted, ambiguous, failure, and recovery states; exact money rendering; accessibility; responsive/print behaviour; and sensitive-field masking/reveal.

## Data and database design

- Tables and ownership.
- Constraints and indexes.
- Transaction/isolation/lock ordering.
- Migration and backfill.
- Retention/classification/encryption.
- Query plans/scale-shaped fixtures.

## Asynchronous processing and recovery

- Job state machine.
- Lease/idempotency.
- Retry/backoff/dead-letter.
- Watchdog/age SLO.
- Operator/manual resolution.
- Restore/replay semantics.

## Observability and operations

### Metrics

- ...

### Alerts

- ...

### Traces/logs/audit

- ...

### Runbooks

- ...

## Tests most agents will skip

### Invariants and model tests

1. ...

### Concurrency and failpoints

1. ...

### Authorization and abuse

1. ...

### Numeric, temporal, parser, and compatibility boundaries

1. ...

### Restore/replay/operations

1. ...

## Performance and capacity

Workload, dataset, contention, resource limits, SLO hypothesis, and post-run invariant checks.

## Acceptance gate

A reviewer can perform a named end-to-end scenario, inject at least one meaningful failure/attack, inspect the financial/security evidence, and reproduce the result.

## Evidence artifacts

- requirement/test mapping;
- diagrams/ADRs;
- contract fixtures;
- test/benchmark/security reports;
- dashboards/traces;
- runbooks/game-day/postmortem;
- sanitized demo assets.

## X content pillars

### Pillar A — [Specific engineering truth]

- Hook.
- Artifact.
- Test/failure result.
- Limitation/trade-off.

### Pillar B — ...

## Do not waste time on

List plausible but low-signal, premature, unsafe, or out-of-scope work.

## Exit checklist

- [ ] Requirements implemented/traced.
- [ ] Threat/risk register updated.
- [ ] Contracts/conformance complete.
- [ ] Critical tests and security gates pass.
- [ ] Operational/recovery path exists.
- [ ] Evidence/content captured.
- [ ] Limitations explicit.
