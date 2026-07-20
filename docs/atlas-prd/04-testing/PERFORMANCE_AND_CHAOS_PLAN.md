# Performance, resilience, chaos, and recovery plan

## Purpose

Prove that Atlas preserves correctness and remains diagnosable under representative contention, dependency degradation, backlog, and recovery. The objective is not a vanity TPS number.

## Workload model

Every report identifies:

- source revision and image digests;
- infrastructure limits and versions;
- schema/data volume and history depth;
- tenants, wallets, currencies, and account distribution;
- request mix and concurrency;
- hot-account/contention profile;
- provider latency/error distribution;
- event and webhook fan-out;
- warm/cold cache condition;
- test duration and ramp pattern;
- invariant verification after run.

### Baseline synthetic distribution

A scale-shaped reference dataset:

- 100 tenants: many small, several medium, one intentionally noisy;
- 250,000 customer identities;
- 400,000 wallets across NGN and USD;
- 25 million journal postings;
- 5 million transactions;
- 2 million provider attempts/events;
- 1 million webhook deliveries;
- 250,000 settlement lines and 25,000 reconciliation exceptions;
- account-history skew so a small percentage of wallets are very active;
- synthetic-only data.

Scale down locally while preserving ratios and contention patterns.

### Traffic mix

Representative steady state, adjusted with measured results:

- 35% wallet/transaction reads;
- 12% identity/session/authorization reads;
- 10% internal transfer commands/status;
- 8% external transfer commands/status;
- 8% merchant payment/capture/refund commands;
- 7% operational search/case work;
- 5% webhook API reads/replays;
- 5% report/statement requests;
- 5% settlement/reconciliation work;
- 5% other.

Background workers simultaneously publish outbox events, query ambiguous provider attempts, expire holds, deliver webhooks, generate statements, and reconcile files.

## Performance experiments

### PERF-EXP-01 — Ledger posting baseline

- Controlled posting templates with 2, 4, and 10 lines.
- Low contention and hot-account contention.
- Capture commit latency, lock wait, serialization retries, WAL volume, CPU, allocations, and exact invariant results.
- Compare synchronous projection strategies only after correctness is fixed.

### PERF-EXP-02 — Double-spend contention

- Many concurrent withdrawals from one/few wallets.
- Prove accepted reservations never exceed available funds.
- Report rejected commands separately from system errors.
- Inspect fairness/starvation and tail latency.

### PERF-EXP-03 — Transaction history and cursor stability

- Deep histories, concurrent inserts, multiple filters.
- Measure p50/p95/p99 and query plans.
- Verify no cross-tenant leakage, duplicate page items, or invalid cursor reuse.

### PERF-EXP-04 — Outbox and consumer backlog

- Broker outage then recovery.
- Measure publication age, throughput, duplicate deliveries, consumer lag, database pressure, and time to drain.
- Invariants and online command latency remain within defined safety limits.

### PERF-EXP-05 — Provider ambiguity storm

- Elevated timeouts with mixed eventual success/failure.
- Measure query scheduler, attempt aging, hold duration, operator queue, and provider rate controls.
- No alternate blind resubmission.

### PERF-EXP-06 — Webhook noisy tenant

- One merchant returns slow `5xx`; others healthy.
- Prove per-endpoint/tenant bulkheads and queue fairness.
- Measure retry age, worker utilization, connection pool, and healthy-tenant latency.

### PERF-EXP-07 — Reconciliation import and matching

- Files from 10k to 5m rows; duplicates, malformed records, and high exception rates.
- Measure streaming memory, parse/match throughput, database temp/spill, restart/resume, and deterministic rerun.

### PERF-EXP-08 — Statements and exports

- Concurrent long-period statements on deep histories.
- Verify queue/backpressure, object storage throughput, exact totals, cross-format consistency, and API isolation.

### PERF-EXP-09 — Operations search

- Authorized filters against representative indexes and masked joins.
- Test abusive broad queries and cancellation.
- Validate access controls remain before pagination/counting.

### PERF-EXP-10 — Endurance/soak

- 12–24 hour mixed workload in reference environment.
- Detect goroutine/file/socket leaks, pool drift, queue lag, cache growth, idempotency retention growth, and latency degradation.
- Run balance/outbox/inbox/reconciliation verification at end.

## SLO candidates for the portfolio reference environment

These are targets to validate and adjust, not production promises:

- 99.9% of accepted internal-transfer API commands return durable status within 750 ms under defined normal load.
- 99.9% wallet balance reads under 300 ms under defined normal load.
- 99% outbox events published within 30 seconds; no accepted command lost.
- 99% provider callback processing within 15 seconds excluding provider delay.
- 99% healthy merchant webhook first attempts begin within 30 seconds.
- reconciliation run completion based on file-size classes, with explicit queue/start and throughput objectives.
- critical financial invariant violations: zero tolerated.

Report achieved values and environment constraints honestly.

## Backpressure and overload policy

- Bound HTTP body, concurrency, queue, database pool, outbound connection, and worker leases.
- Reject/defer before exhausting shared resources.
- Prioritize state finalization, provider callbacks, hold expiry, and integrity verification over reports/analytics.
- Separate worker pools/queues for financial lifecycle, webhooks, reports, and bulk reconciliation.
- Rate limits combine identity, tenant, credential, endpoint, IP/network risk, and sensitive-flow dimensions.
- Circuit breakers do not turn ambiguous financial outcomes into failures.

## Chaos experiments

### CHAOS-01 — API process termination at commit boundary

Use failpoints before commit and immediately after commit. Expected: rollback before commit; stable idempotent status after commit.

### CHAOS-02 — Worker death around outbox publication

Terminate after claim and after broker send. Expected: lease recovery, possible duplicate delivery, one consumer effect.

### CHAOS-03 — Broker partition/outage

Continue money commands only while durable outbox capacity/age policy permits. Recover and drain without database collapse.

### CHAOS-04 — PostgreSQL primary loss

Exercise failover/reference recovery policy. No dual-primary writes. Verify committed journal boundary and client ambiguity handling.

### CHAOS-05 — Connection pool exhaustion

Inject slow queries/held connections. Expected bounded waits, load shedding, diagnostics, no goroutine explosion.

### CHAOS-06 — Redis loss

Financial truth unaffected. Rate limiting/cache follows explicit degraded policy; no balance or authorization source switches to Redis.

### CHAOS-07 — Object storage degradation

Statements/imports pause/retry; resource never marked ready before complete digest/object. Critical transfer processing stays isolated.

### CHAOS-08 — Identity provider outage

Existing low-risk session policy only; new authentication/step-up/high-risk action fails safely.

### CHAOS-09 — Provider timeout/duplicate/out-of-order burst

Use simulator scenario controls. Expected deterministic state machine, bounded queries, one economic effect, operator visibility.

### CHAOS-10 — Key-management dependency outage

New encrypt/decrypt/sign operations behave per data/control criticality. No plaintext fallback. Alerts and runbook triggered.

### CHAOS-11 — Clock skew

Skew one process within/outside tolerated bounds. Test signatures, quotes, leases, expiry, and audit ordering. Durable database sequence remains authoritative where required.

### CHAOS-12 — Slow/bad deployment

Roll old/new versions with event/API compatibility fixtures and live synthetic traffic. Abort on invariant/compatibility indicators.

## Disaster recovery exercises

### Restore target

Restore PostgreSQL to an isolated network at a selected timestamp, restore/inventory object storage, deploy exact compatible application revision/config, then reconcile:

- journal/posting totals and period trial balances;
- account balance projections;
- active holds;
- transfer/payment/refund states;
- idempotency records;
- outbox and consumer inbox/checkpoints;
- provider attempts/callback receipts;
- settlement files and reconciliation outcomes;
- statement/report objects and digests;
- audit manifests/signatures;
- key-version availability.

### Recovery scenarios

1. Point just before API acknowledged an internal transfer.
2. Point after provider accepted an external transfer but before callback.
3. Point after journal commit but before event publication.
4. Point during a settlement import/reconciliation run.
5. Point during signing-key rotation.
6. Restore from backup containing an older schema/config version.
7. Bad migration recovery through forward fix or restore according to runbook.

### RPO/RTO evidence

Measure separately:

- infrastructure available;
- database restored;
- integrity verification complete;
- read-only operational access available;
- financial writes safely reopened;
- asynchronous backlogs reconciled.

Do not call the service recovered merely because PostgreSQL started.

## Observability required during experiments

- request rate, errors, duration by operation and status class;
- queue depth/oldest age, retries, dead letters;
- database pool, locks, deadlocks, serialization retries, query duration, WAL/storage;
- Go heap, GC, goroutines, scheduler, file descriptors, sockets;
- provider attempt states/ages;
- holds by age and reason;
- webhook health by tenant/endpoint;
- balance/reconciliation/inbox/outbox invariant results;
- deployment/config/build digests;
- trace samples for success, rejection, ambiguity, duplicate, and recovery.

Telemetry attributes must be bounded and free of sensitive/high-cardinality raw values.

## Report format

Every performance/chaos report contains:

1. question and acceptance hypothesis;
2. architecture path under test;
3. environment and reproducibility;
4. workload/fault timeline;
5. results and confidence/variance;
6. financial/security invariant checks;
7. traces/query plans/profiles;
8. bottleneck and causal analysis;
9. changes made and before/after result;
10. limitations and next experiment.
