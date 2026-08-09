# P01-S07 typed approval closure — pre-commit evidence

- **Evidence ID:** `EVD-P01-S07-APPROVAL-CLOSURE`
- **Phase/slice:** `PHASE-01_IDENTITY_ACCESS_TENANCY` / `P01-S07`
- **Source:** `UNCOMMITTED_WORKTREE(base=d3b8612ff4df2bd67bc16faa322c8ca397318f6e)`
- **Environment:** isolated local Podman PostgreSQL/NATS project plus repository-local Go caches
- **Determinism:** synthetic principals and fixed UTC domain clocks; PostgreSQL concurrency uses bounded serializable retries
- **Reproduce:** `pwsh -NoProfile -File ./scripts/verify-p01-s07.ps1 -Live -ContainerRuntime podman`
- **Observed result:** `PASS`
- **Revalidate by:** 2026-11-09

## Boundary and decisions

Operations owns the approval aggregate and tables. Identity provides explicit transaction-bound
authorization, target validation, and the only executable Phase 01 target command. Audit facts,
approval transitions, membership changes, authorization-version increments, and target-session
revocation commit atomically. No external call occurs in the transaction.

The executable registry is deliberately restricted to
`identity.organization.membership.change_admin`. Its schema is version 1, its canonical bytes are
RFC 8785 JSON, and its immutable digest is SHA-256. The payload binds organization, membership,
expected membership version, requested administrator role, and purpose. A 24-hour expiry and
`dynamic-at-decision-and-execution` checker policy are stored explicitly. Additional financial,
credential, event, worker, arbitrary JSON, direct SQL, and administrator-removal actions are absent.

The immutable Atlas principal ID, not an IdP login or session ID, enforces maker/checker
separation. Create, decision, execution, and cancellation are independently idempotent. Approval
and target versions are optimistic preconditions. Execution is explicit and rechecks the maker,
checker, executor, action-bound step-up, current tenant permission, target state/version, canonical
payload bytes/digest, and action binding inside one serializable transaction.

## Scope proved

The cumulative verifier binds the S04–S06 regression chain, OpenAPI/AsyncAPI lint, architecture
guards, migration 14 manifest, real database roles and isolated recovery, approval model and HTTP
tests, bounded observability mutation canaries, evidence integrity, builds, and the real PostgreSQL
approval suite. The focused live run proved:

- exact create replay returns the original authorization decision ID;
- an alternate session for the same principal cannot approve the maker's request;
- simultaneous eligible checkers produce one durable decision and one conflict;
- maker or checker permission downgrade after approval blocks execution with no target effect;
- expired action-bound execution step-up blocks with no target effect;
- a successful execution changes the bound target role once, increments authority/version,
  revokes target sessions, records approval-linked Identity history with the canonical payload
  digest, writes execution plus target Audit facts, and replays without a second target effect;
- database payload tamper and target-version races supersede the approval without changing the
  target;
- cancelled, rejected, expired, and superseded approvals cannot execute; cancellation replay is
  stable and every target remains unchanged;
- Audit outage rolls back target, approval, execution, and Audit state, after which the same
  idempotent execution safely succeeds; and
- strict request decoding, CSRF, idempotency, ETag, tenant concealment, bounded route inventory,
  response headers, errors, metrics, grants, cross-context persistence rules, and restore checks
  pass; and
- interrupted approval and invitation integration runs are recovered by exact synthetic-fixture
  startup cleanup with transactional, ordered, error-checked teardown; the subsequent recovery
  gate requires the canonical three seeded principals and therefore detects leaked test state.

## Requirement, threat, risk, and adversarial links

- **Requirements:** `IAM-006`, `IAM-030..034`; `IAM-020..026` and `IAM-025` are revalidated at
  create, decision, and execution.
- **Threats:** `THR-007`, `THR-008`, `THR-018`, `THR-024`, `THR-040`, `THR-041`, `THR-058` at this
  bounded approval action.
- **Risks:** `RSK-006` is addressed by the closed typed action/state transition order;
  `RSK-028` is addressed by state-model, mutation, contract, database-role, recovery, real
  PostgreSQL, concurrency, tamper, and Audit-failure oracles. Programme-wide residual risks remain.
- **Adversarial tests:** `ADV-IAM-005`, `ADV-IAM-006`, `ADV-WEB-011`, `ADV-AUD-002`; plus
  same-principal alternate-login, simultaneous checker, database payload mutation, target race,
  terminal-state execution, response-loss replay, and Audit-outage cases.

## Failure posture and operations

Malformed, unknown, stale, unauthorized, cross-tenant, expired, terminal, version-mismatched,
integrity-failed, and unavailable states deny safely. Database serialization, deadlock, and bounded
lock-timeout errors retry up to three times. A failure before commit has no target effect. Response
loss after commit returns the durable result without repeating the effect. Integrity or binding
failure supersedes the approval and pages the bounded approval-integrity signal.

The runbook permits route disablement, session/role containment, evidence-preserving inspection,
idempotent replay, and restore rehearsal. It forbids direct payload/status edits and deletion of
approval or Audit records. `P01-S07-ALR-001..004` cover aging, expired/execution-failed state,
integrity failure, and conflict spikes without identifier labels. No deployed pager or SIEM is
claimed.

## Sanitization

This report includes only public source identities, closed synthetic action/status names, bounded
PASS aggregates, migration identity, requirement/threat/test IDs, and control semantics. It
excludes credentials and database URLs, cookies, CSRF/idempotency/OIDC material, opaque tenant,
principal, session, member, approval, decision and Audit identifiers, raw payloads, raw SQL errors,
WAL/data pages, network addresses, and real identity data.

## Limitations

- Phase 01 has one executable synthetic non-financial approval action. Generic future action
  registration requires its owning phase, typed target service, authorization catalogue entry,
  threat review, contract, migration if needed, and evidence.
- Technical pre-commit target or Audit failure leaves an approved request retryable rather than
  manufacturing an `execution_failed` outcome. The state model permits `execution_failed` for a
  future typed action that can durably report a failed execution without applying its target
  effect; the current all-or-nothing target has no such outcome.
- Administrator removal remains fail-closed until exact last-administrator and fresh-step-up
  semantics are approved. API credentials remain absent until P01-S08.
- Evidence is same-host synthetic proof. No real IdP, production data, independent reviewer,
  production pager, compliance certification, or financial behavior is claimed.
