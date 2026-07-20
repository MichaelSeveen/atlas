# Atlas definition of done

## Feature-level done

A feature or requirement is done only when all applicable items are satisfied.

### Product and domain

- Requirement ID, actor, problem, preconditions, success, rejection, ambiguous, and recovery states are explicit.
- Domain terms match the glossary and state machine.
- Financial effects specify holds, journals, fees, settlement, reversal, and reconciliation implications.
- Out-of-scope behaviour is explicit; no accidental “temporary” bypass.

### Architecture and data

- Owning module and dependency direction are clear.
- Schema, indexes, constraints, migration, and retention/classification are reviewed.
- Transaction boundary, lock/order/concurrency strategy, idempotency, and event boundary are documented.
- No new distributed component is added without measured need and operational ownership.
- ADR exists for material/irreversible trade-off.

### Security and privacy

- Threat register updated with abuse cases and residual risk.
- Authentication, tenant/object/action/field authorization, step-up, and approval requirements are implemented server-side.
- Sensitive fields are minimised, encrypted/masked, and absent from logs/events/errors unless explicitly safe.
- Rate/resource limits and anti-automation controls cover sensitive flows.
- Privileged actions have reason, actor, decision, correlation, audit, and safe operational UI.
- Security negative/adversarial tests pass.

### API and events

- OpenAPI/AsyncAPI contract, examples, errors, retryability, limits, and compatibility assessment are complete.
- Money, timestamps, pagination, ETag, idempotency, and `202` semantics follow standards.
- Provider and merchant contracts have signed/replay/deduplication behaviour where applicable.
- Generated types/docs are current and contract conformance passes.

### Frontend

- Loading, empty, happy, rejection, restricted, stale, ambiguous, failure, and recovery states are designed and tested.
- UI never implies finality before the underlying lifecycle is final.
- Permissions are reflected but never treated as the authorization control.
- Keyboard, focus, semantic structure, screen-reader messaging, contrast/non-colour cues, responsive/print behaviour are verified.
- Exact monetary rendering avoids JavaScript numeric loss.
- Sensitive state clears correctly on logout/tenant switch/session expiry.

### Tests

- Domain unit/table tests.
- Property/model/metamorphic tests for critical invariants.
- PostgreSQL integration tests with production roles/constraints.
- Concurrency and failpoint tests around every durability boundary.
- Authorization/tenant negative matrix.
- Contract and browser/component tests.
- Fuzz corpus for parsers/signatures/high-risk primitives where applicable.
- Mutation test demonstrates critical test sensitivity.
- A “test most agents skip” from the phase is implemented and evidenced.

### Observability and operations

- Structured logs, metrics, traces, audit events, and safe correlation exist.
- Dashboard/alert/runbook covers stuck, duplicate, ambiguous, or integrity-failure conditions.
- Operators can investigate and recover without direct database edits.
- Retry/dead-letter/manual resolution is permissioned, idempotent, reasoned, and audited.

### Delivery and evidence

- CI, migrations, scans, SBOM/provenance, and deployment checks pass.
- Requirement traceability row points to tests and evidence.
- Screenshots/traces/reports are sanitized and tied to source revision.
- Known limitations and deferred risks are documented.
- At least one evidence-led content artifact is captured when the phase calls for it.

## Phase-level done

A phase is complete only when:

1. every mandatory phase requirement is implemented or explicitly rejected through documented risk/scope decision;
2. phase acceptance scenario runs end-to-end;
3. critical adversarial tests pass;
4. architecture/security/privacy reviews close required findings;
5. API/event docs and operational runbooks are current;
6. dashboards and alert tests exist;
7. performance/resource behaviour is measured where introduced;
8. threat/risk/traceability registers are updated;
9. content pillars are backed by real evidence;
10. no headline claim exceeds what was proven.

## Release-level done

- Complete synthetic critical journey and failure demo.
- Independent ledger/statement/reconciliation verification passes.
- Restore/game-day evidence is current.
- Security release gates pass and findings are dispositioned.
- OpenAPI/AsyncAPI validate and are publicly renderable.
- Hosted demo is isolated, synthetic, rate-limited, resettable, and visibly non-production.
- Claims ledger links architecture, tests, benchmarks, security evidence, and known limitations.
- Repository contains no real personal data, secrets, proprietary employer material, or misleading certification/licensing claims.
