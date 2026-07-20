# Incident postmortem — [Title]

- **Incident ID:**
- **Severity:**
- **Start/detection/containment/recovery times:**
- **Environment/source revision/config digest:**
- **Authors/reviewers:**
- **Customer/merchant/financial/privacy/security impact:**
- **Status:** Draft | Final

## Executive summary

Plain description of what happened, affected scope, duration, and current safety.

## Impact

- Accepted commands affected.
- Incorrect, duplicated, delayed, or ambiguous financial effects.
- Data/tenant/security impact.
- Operator workload and external dependencies.
- Exact reconciliation status and residual uncertainty.

## Detection

What signal detected it, why it fired when it did, and whether a better signal existed.

## Timeline

Use UTC and distinguish event time, recorded time, and discovery time.

| Time | Event/evidence | Actor/system |
|---|---|---|
| ... | ... | ... |

## Technical narrative

Trace one representative command through API, authorization, risk, hold, provider, ledger, outbox/event, webhook, settlement/reconciliation, and audit as relevant.

## Financial integrity assessment

- Journals/postings/trial balance.
- Holds and available balances.
- Duplicate/business/provider references.
- External/provider state.
- Settlement/reconciliation exceptions.
- Statements/reports/audit evidence.
- Restore/replay effects.

## Security and privacy assessment

Authorization, secrets/keys, data exposure, audit completeness, evidence integrity, and attacker/insider possibility.

## Root cause

Use causal factors, not “human error.” Identify design, implementation, testing, observability, process, and recovery gaps.

## Contributing conditions

- ...

## What worked

- ...

## What failed or made recovery harder

- ...

## Containment and recovery

Commands/actions performed, approvals, safe retry/replay, manual adjustments, and validation before reopening writes.

## Corrective actions

| Action | Control/requirement | Owner | Priority | Due | Verification and evidence |
|---|---|---|---|---|---|
| ... | ... | ... | ... | ... | ... |

At least one action must improve prevention, one detection, and one recovery where applicable. Critical defects require regression tests.

## Lessons and residual risk

What assumption changed, what remains unproven, and whether public technical disclosure is safe/useful.
