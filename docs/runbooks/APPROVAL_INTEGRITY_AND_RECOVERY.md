# Approval integrity and recovery

## Scope

This runbook covers the Phase 01 Operations-owned approval workflow and its sole executable action,
`identity.organization.membership.change_admin`. It applies to stale or expired approvals,
unexpected conflict spikes, canonical-payload integrity failures, maker-checker separation failures,
Audit persistence failure, and ambiguous HTTP response loss. It does not authorize direct database
state repair, financial actions, or revival of a terminal approval.

## Immediate safety posture

1. Keep approval execution fail-closed. If an integrity alert fires, disable the approval execution
   route at the application boundary before considering broader service shutdown.
2. Do not edit `atlas_operations` approval status, payload bytes, digest metadata, maker/checker IDs,
   target version, decision, execution, cancellation, or Audit rows.
3. Preserve the source revision, environment revision, alert window, sanitized trace/correlation and
   decision IDs, database logs, and a protected recovery copy. Never place session cookies, CSRF
   values, idempotency keys, raw payloads, or local runtime credentials in incident evidence.
4. Escalate any payload or maker-checker integrity failure to the Operations and security owners.
   These failures are not expected during ordinary stale-client behavior.

## Triage

1. Confirm that readiness and Audit persistence are healthy. Follow
   [Database unavailable](DATABASE_UNAVAILABLE.md) or
   [Audit persistence unavailable](AUDIT_PERSISTENCE_UNAVAILABLE.md) first when either dependency is
   failing.
2. Use the authorized approval read API and append-only Audit chain to identify the bounded action,
   current status, version, maker, checker, target reference, and decision IDs. Do not query or expose
   another tenant's approval.
3. Classify the signal:
   - `state`, `precondition`, or `idempotency` conflicts normally indicate a stale client, racing
     checker, or changed request and require a fresh read before any retry;
   - `superseded` means the canonical payload, target version/state, or separation binding no longer
     authorizes execution;
   - `payload_*` or `maker_checker_integrity_failure` requires security investigation and continued
     execution disablement;
   - old `pending`, `expired`, or `execution_failed` observations require review of reviewer capacity,
     session step-up, target state, and Audit availability.
4. Correlate the approval decision and target-domain Audit facts. A successful execution must have
   exactly one Operations execution record, one approval-linked Identity role-change record, the
   target authorization/session invalidation, and both Audit facts from one commit.

## Safe recovery

- For HTTP response loss, retry the exact request with the same idempotency key and request body, or
  read the approval. Atlas returns the committed result and does not repeat the target effect.
- For a stale ETag, changed target, revoked maker/checker permission, or expired action-bound step-up,
  re-read authoritative state. Reauthenticate or step up only through the normal Identity flow.
- For an expired, rejected, cancelled, executed, or superseded approval, create a new approval from
  the current typed target state if policy still permits it. Never reopen or clone stored payload
  bytes by database edit.
- After an Audit outage, confirm the failed attempt left approval and target unchanged, restore Audit,
  then retry through the API with current permissions, step-up, target version, and policy.
- If storage corruption is confirmed, keep writes disabled and use the repository backup/PITR
  procedure. Verify migration 14, the Operations scope registry and grants, immutable payload/hash
  metadata, Identity target links, and the Audit chain before reopening execution.

## Exit criteria

- The triggering metric has returned to its normal bounded baseline.
- No approval or Audit row was directly edited or deleted.
- Current permission, maker/checker separation, action-bound step-up, target version/state, canonical
  payload digest, and one-effect idempotent replay are revalidated.
- Any route disablement is removed only after static tests, real PostgreSQL approval integration, and
  isolated restore verification pass at the incident source revision.
- The incident record states the synthetic/reference limitations and links sanitized, digest-bound
  evidence.
