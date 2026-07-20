# Phase 05 — Risk policy, transaction controls, and limits

## Outcome

Build a deterministic, versioned, explainable policy engine that evaluates account capability, KYC tier, transaction limits, velocity, beneficiary and device/context signals, and restrictions before money movement is accepted.

## Why this phase is high-signal

Risk is not a random score or an “AI fraud” badge. In a credible system, the exact policy version, input snapshot, factors, outcome, override path, and downstream effect must be reproducible months later. This phase demonstrates policy design, temporal queries, explainability, concurrency, privacy, and operational judgement.

## Dependencies

Phases 01, 02, and 04.

## Risk model

### Decision outcomes

- `allow` — command may continue if all non-risk conditions pass.
- `review` — command is durably accepted into a reviewable state; funds may be held according to policy.
- `deny` — command is rejected or terminated with no unmodelled financial effect.

### Policy components

1. Eligibility: wallet state, customer state, KYC tier, currency, product capability.
2. Static limits: per transaction, daily, weekly, monthly, balance and beneficiary limits.
3. Velocity: count and sum over configured windows.
4. Change risk: new device/session, contact change, newly added beneficiary, recently rotated credential.
5. Pattern rules: rapid pass-through, repeated failures followed by success, shared beneficiary concentration, amount thresholds.
6. Operational restrictions: manual blocks, compliance holds, provider constraints.

## Functional requirements

### Policy definition and versioning

- `RSK-001` Policy and rule versions are immutable once activated.
- `RSK-002` Draft, shadow, active, retired, and rolled-back lifecycle is explicit.
- `RSK-003` Activation requires validation, simulation, reviewer approval for high-impact changes, and effective time.
- `RSK-004` A transaction stores the exact policy version and input snapshot references used.
- `RSK-005` Retired policy remains queryable for historical explanation.
- `RSK-006` Rule order and conflict resolution are deterministic.

### Decision execution

- `RSK-010` Decision happens inside the command acceptance boundary using a consistent view of restrictions, limits, and relevant balances.
- `RSK-011` Decision factors are structured codes with safe customer-facing groups and richer workforce details.
- `RSK-012` Missing or stale critical input fails closed or routes to review according to explicit policy.
- `RSK-013` Decision is idempotent for the same command and input snapshot.
- `RSK-014` Risk cannot directly mutate ledger balances; it returns constraints and outcomes to the owning workflow.
- `RSK-015` A manual override creates a new decision, never edits the original.

### Limits and counters

- `LIM-001` Limits support amount and count, scoped by customer, wallet, merchant, beneficiary, credential, and transaction type.
- `LIM-002` Windows have explicit timezone and boundary semantics; default is UTC rolling windows unless product requirement says otherwise.
- `LIM-003` Accepted/reserved/completed/failed/reversed states define whether and when they consume or release limit capacity.
- `LIM-004` Concurrent commands cannot both pass a limit that only one should satisfy.
- `LIM-005` Counter projections are rebuildable from source transactions.
- `LIM-006` Policy changes do not retroactively change historical decisions.

### Rule simulator and shadow mode

- `RSK-020` Run proposed policy against synthetic historical events.
- `RSK-021` Compare allow/review/deny changes, rule hit rates, customer segments, and operational review volume.
- `RSK-022` Shadow policy evaluates live synthetic flows without affecting outcomes.
- `RSK-023` Simulator output stores dataset version, policy version, code revision, and timestamp.

### Review and appeal foundation

- `RSK-030` Review cases include factors, evidence references, transaction snapshot, deadlines, and assignment.
- `RSK-031` Customer-facing adverse outcome has a safe reason category and support/review path where applicable.
- `RSK-032` Reviewer override requires role, step-up for high-risk cases, reason code, notes, and audit.
- `RSK-033` Override execution rechecks current account and transaction state.

## API surface

Policy administration:

- `GET /v1/risk/policies`
- `POST /v1/risk/policies`
- `GET /v1/risk/policies/{policy_id}/versions/{version}`
- `POST /v1/risk/policies/{policy_id}/simulations`
- `POST /v1/risk/policies/{policy_id}/activation-requests`

Decision and case reads:

- `GET /v1/risk/decisions/{decision_id}`
- `GET /v1/risk/cases`
- `GET /v1/risk/cases/{case_id}`
- `POST /v1/risk/cases/{case_id}/decisions`

Internal command interface:

```go
type EvaluationRequest struct {
    CommandID       string
    Subject         SubjectSnapshot
    Transaction     TransactionSnapshot
    Context         ContextSnapshot
    EffectiveAt     time.Time
}

type EvaluationResult struct {
    DecisionID      string
    Outcome         Outcome
    PolicyVersion   string
    Factors         []Factor
    Constraints     Constraints
}
```

## Frontend requirements

### Risk analyst workbench

- Policy version diff with semantic changes, not raw JSON only.
- Rule builder limited to supported typed rules; no arbitrary code execution.
- Historical simulation comparison with review-volume impact.
- Decision view shows timestamped input snapshot and factors.
- Case queue supports age, severity, product, factor, and assignment filters without exposing unnecessary PII.
- Override action displays financial and customer consequence.

### Customer experience

- Known eligibility/limit denial explains what limit or requirement applies when safe.
- Fraud-sensitive reasons remain grouped and do not teach evasion.
- Review state gives realistic timing and avoids false success.
- Retry guidance distinguishes “change input,” “wait,” “complete verification,” and “contact support.”

## Tests most agents will skip

1. Two concurrent transfers both approach a daily limit; only permitted aggregate passes.
2. Reversed transfer releases limit capacity exactly once under duplicate reversal events.
3. Transaction crosses UTC day boundary while policy uses rolling 24-hour window.
4. DST boundary for a deliberately configured local-calendar limit is deterministic.
5. Policy activation occurs between idempotent retries; retry returns original decision, not new policy result.
6. Shadow policy failure cannot block live transaction.
7. Missing velocity projection is detected and does not silently undercount.
8. Counter rebuild from transaction history matches projection after random state sequences.
9. Rule conflict where one allows and another denies resolves by documented precedence.
10. Reviewer overrides after wallet becomes closed; execution is blocked.
11. Risk input contains stale KYC tier due cache; authoritative recheck prevents decision.
12. Device/contact change signal expires at exact boundary without clock race.
13. Customer-facing reason remains safe while workforce sees detailed factor.
14. Policy JSON attempts deeply nested/huge rule payload; resource limits prevent denial of service.
15. Malicious regex or expression cannot create catastrophic evaluation time because arbitrary regex/code is disallowed or bounded.
16. Historical decision remains explainable after rule and customer data change.
17. Duplicate identity/beneficiary graph signals do not leak another customer’s identity.

## Observability and alerts

Metrics:

- decisions by outcome, rule, policy version, transaction type;
- evaluation latency;
- rule error/missing input;
- limit-denial and review volume;
- policy simulation duration;
- override rate by analyst and rule;
- projection/rebuild variance;
- review case age.

Alerts:

- sudden allow/deny distribution change after activation;
- risk engine unavailable or critical input missing;
- anomalous override volume;
- policy evaluation latency breach;
- limit projection variance;
- expired cases beyond SLA.

## Acceptance gate

A reviewer can create a draft policy, run it against seeded transaction history, compare outcomes, activate through approval, race concurrent limit-consuming transfers, inspect a stored decision months-later style after policy changes, and override a review case without editing original evidence.

## X content pillars

### Pillar A — “I did not build an AI fraud score”

- Explain deterministic rules and why they are appropriate for this project.
- Show policy version, input snapshot, factors, and human review.
- Discuss where ML could later fit without decision opacity.

### Pillar B — “A daily limit is a concurrency problem”

- Demonstrate two simultaneous transfers.
- Show atomic reservation of limit capacity.
- Rebuild counters from source history.

### Pillar C — “Risk decisions must survive policy changes”

- Evaluate under version A.
- Activate version B.
- Reopen old decision and explain it from stored evidence.

### Pillar D — “Shadow mode before enforcement”

- Show simulation and expected review workload.
- Run a shadow policy.
- Explain rollback criteria.

## Do not waste time on

- black-box ML model training;
- device fingerprinting vendor complexity;
- arbitrary code/SQL rule engines;
- hundreds of rules;
- graph databases before a relational model is inadequate;
- customer-facing disclosure of evasion-sensitive rules;
- using Redis counters as the only limit truth.
