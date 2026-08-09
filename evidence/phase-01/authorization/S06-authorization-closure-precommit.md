# P01-S06 authorization closure — pre-commit evidence

- **Evidence ID:** `EVD-P01-S06-AUTHORIZATION-CLOSURE`
- **Phase/slice:** `PHASE-01_IDENTITY_ACCESS_TENANCY` / `P01-S06`
- **Source:** `UNCOMMITTED_WORKTREE(base=a2248839905f86b67faae5076f8115e6e924e39f)`
- **Environment:** isolated local Podman PostgreSQL/NATS project plus repository-local Go toolchain cache
- **Determinism:** repository synthetic identities and fixed UTC domain clocks; timing samples use the
  same local PostgreSQL path with interleaved ordering
- **Reproduce:** `pwsh -NoProfile -File ./scripts/verify-p01-s06.ps1 -Live -ContainerRuntime podman`
- **Observed result:** `PASS`
- **Revalidate by:** 2026-11-09

## Scope proved

The cumulative verifier binds the prior S04/S05 regression chain, Go tests, OpenAPI/AsyncAPI lint,
the closed migration manifest, architecture/mutation guards, observability alert canaries, evidence
integrity, and the real PostgreSQL authorization suite. The live database is at migration version
13 with released checksum
`84e84ecd572b50ab15f8e080f3431c04246f433c97ebecdfb9fde1719d351179`.

The bounded evidence supports:

- an immutable source-controlled runtime evaluator matching all 23 permissions, 13 roles, five
  purposes, and three field rules in the canonical identity-access policy;
- default denial for unknown or stale principal population, role, permission set, action, object,
  field, purpose, assurance, authorization version, resource version, and resource state;
- server-side session, tenant, role/permission, action, object, field, purpose, assurance, current
  authorization-version, and resource-state checks before page-size or cursor evaluation;
- a uniform concealed `404` problem shape for known-foreign and absent tenant identifiers with no
  data, count, page, cursor, total, or suggestion metadata;
- 16 interleaved real-PostgreSQL observations per known-foreign/absent arm; the recorded focused
  run observed a 7.4492 ms median delta, 1.15 median ratio, and 40.3751 ms p95 delta, all within the
  explicit 150 ms / 2.5 / 500 ms local regression bounds;
- purpose- and permission-bound `email_hint` handling: ordinary reads project SQL `NULL`; only the
  exact sensitive permission plus `organization_administration` may return an already-masked
  invitation hint, with no canonical/raw email column or response;
- atomic append-only Audit linkage for privileged masked-hint disclosure and concealed denial,
  including stable decision IDs and rollback to service-unavailable when Audit recording fails;
- two independent API pools invalidating stale role, disabled-resource, and session-assurance
  facts from PostgreSQL without Redis or process-local decision truth; after a role downgrade, the
  replacement session retains member-list access but loses the sensitive field projection; and
- PostgreSQL `55P03` lock-timeout classification as a bounded retryable membership-mutation race,
  alongside the existing serialization/deadlock retry classes and fail-closed unknown SQL states.

## Requirement, threat, and adversarial links

- **Requirements:** `IAM-013`, `IAM-020..026`; `IAM-011..012` remain verified repository/query
  preconditions.
- **Threats:** `THR-005`, `THR-007`, `THR-028`, `THR-038`, `THR-041`, `THR-044`, `THR-057`,
  `THR-058` at the implemented Identity boundary.
- **Adversarial tests:** `ADV-IAM-001..004`, `ADV-IAM-011` at the bounded masked-member field, and
  phase tests-most-agents-skip covering authorization before malformed cursor work, a valid
  foreign-ID timing differential, multi-instance invalidation without a cache, and Audit-outage
  rollback of a sensitive read.

## Failure posture

Unknown catalogue data, partial permission bindings, stale authority/session facts, disabled
resources, dependency errors, and Audit failure deny safely. Cross-tenant evaluation happens before
resource lookup and cursor work. No external call occurs inside the authorization transaction.
Repeated ordinary decisions have no product side effect; privileged disclosure and denied
high-risk paths add only their required Audit fact. Rollback disables or narrows the route/policy;
it never restores a removed grant or introduces cache truth.

## Observability and response procedure

Existing emitted Identity operation count/duration metrics provide bounded operation, outcome, and
HTTP-status categories without identifier labels. `P01-S06-ALR-001` detects aggregate concealed
member-list probing; the pre-existing `P01-S05-ALR-001` detects mass membership mutations. Both
link to source-controlled response procedures, and seeded canaries reject identity labels,
ownerless alerts, and removal of the S06 authorization alert. No deployed alert backend is claimed.

## Sanitization

This report contains only public source identities, closed synthetic policy/action names, bounded
PASS results, migration identity, timing aggregates, requirement/threat/test identifiers, and
control semantics. It excludes credentials and database URLs, cookies, CSRF/idempotency/invitation/
OIDC material, opaque tenant/principal/session/member/decision identifiers, raw or canonical email,
raw SQL errors, WAL/data pages, network addresses, and real identity data.

## Limitations

- The implemented Phase 01 authorization surface has one member-list action. No search,
  suggestion, autocomplete, workforce customer-search, credential, approval, wallet, or financial
  operation exists; absence is explicitly verified rather than represented as implemented proof.
- The persisted `email_hint` is an optional already-masked invitation display value. It is not an
  email lookup key, canonical address, reversible token, contact workflow, or identity assertion.
- The timing result is a same-host regression bound, not a constant-time proof or production
  performance claim.
- Alerts and runbooks are repository-owned synthetic controls; no deployed pager/SIEM, real IdP,
  production incident response, independent review, or compliance certification is claimed.
