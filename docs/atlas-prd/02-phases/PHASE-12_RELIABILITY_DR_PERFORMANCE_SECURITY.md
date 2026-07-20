# Phase 12 — Reliability, disaster recovery, performance, and security verification

## Outcome

Prove that Atlas remains financially correct and operationally recoverable under load, dependency failure, process death, message duplication, database restore, key rotation, malicious inputs, and authorized penetration attempts.

## Why this phase is high-signal

Anyone can list “Redis, Docker, OpenTelemetry, rate limiting, and encryption.” This phase replaces architecture adjectives with reproducible evidence, measured constraints, runbooks, game days, and honest residual risks.

## Dependencies

All prior product phases.

## Verification program

### Reliability verification

- `REL-001` Define critical user journeys and business invariants.
- `REL-002` Establish SLO targets and error-budget policy for the reference environment.
- `REL-003` Every asynchronous state has watchdog, retry, dead-letter, and operator path.
- `REL-004` Execute dependency outage and process-crash scenarios.
- `REL-005` Verify no accepted financial command is silently lost.
- `REL-006` Verify duplicate processing produces no duplicate business effect.

### Backup and disaster recovery

- `DR-001` Automated encrypted PostgreSQL backups and point-in-time logs.
- `DR-002` Object storage versioning/immutability and inventory.
- `DR-003` Infrastructure and configuration rebuild from source.
- `DR-004` Restore into isolated environment at a chosen point in time.
- `DR-005` Reconcile journals, balances, holds, idempotency, inbox/outbox, objects, manifests, and reports after restore.
- `DR-006` Document measured RPO/RTO in test conditions.
- `DR-007` Run bad-migration recovery exercise.

### Performance

- `PERF-001` Define workload model using realistic read/write ratios, currencies, tenants, accounts, history depth, and contention.
- `PERF-002` Benchmark command acceptance, ledger posting, wallet reads, transaction lists, webhook throughput, reconciliation import/matching, and report generation.
- `PERF-003` Report p50/p95/p99, throughput, errors, CPU, memory, database locks, pool saturation, queue lag, and invariant results.
- `PERF-004` Run cold/warm cache and small/representative dataset comparisons.
- `PERF-005` Load test never bypasses authentication, authorization, idempotency, ledger, or audit to improve numbers.
- `PERF-006` Publish hardware/container limits and configuration.

### Security verification

- `SEV-001` Map applicable ASVS controls to implementation and tests.
- `SEV-002` Test OWASP API risks including object/property/function authorization, resource consumption, sensitive business flow abuse, SSRF, inventory, and unsafe API consumption.
- `SEV-003` Verify authentication, session, CSRF, CORS, CSP, cache, and step-up controls manually and automatically.
- `SEV-004` Conduct threat-led review of every money-moving and privileged flow.
- `SEV-005` Run SAST, dependency, secret, container, IaC, SBOM, and provenance checks.
- `SEV-006` Fuzz parsers, amount/currency primitives, signatures, provider callbacks, CSV imports, and state transition inputs.
- `SEV-007` Perform manual authorization matrix testing across identities/tenants.
- `SEV-008` Track findings with severity, exploit path, financial impact, fix, regression test, and residual risk.

## Required game days

### Game Day 1 — Provider accepted, worker died

1. Create payout.
2. Simulator accepts.
3. Kill worker before local attempt update.
4. Restart.
5. Watchdog queries by provider reference.
6. Complete once.
7. Verify hold, journal, events, customer timeline, and reconciliation.

### Game Day 2 — Database restore and event replay

1. Create a set of transactions.
2. Allow some outbox messages to publish.
3. Restore database to point before relay marker updates.
4. Replay outbox.
5. Verify idempotent consumers, webhooks, statements, and no duplicate journals.

### Game Day 3 — Key rotation during delayed delivery

Rotate provider callback, merchant webhook, and field-encryption keys while old data/events remain in flight. Verify grace, key IDs, revocation, audit, and old-data decryptability.

### Game Day 4 — Insider misuse

Use a synthetic support identity to enumerate/search/reveal/export unrelated customers, then attempt a forbidden financial action. Verify prevention, detection, alert, and investigation evidence.

### Game Day 5 — Reconciliation break surge

Generate provider file with missing, duplicate, mismatch, late, and fee exceptions. Verify capacity, queues, prioritisation, close blocking, suspense aging, and finance reporting.

### Game Day 6 — Bad deployment/migration

Deploy a backward-incompatible application or lock-heavy migration in staging, detect it through readiness/contract/migration guards, and demonstrate forward-fix or rollback policy.

## Performance workload model

Minimum datasets:

- 100k customers;
- 200k wallets;
- 5 million postings;
- 1 million transfers/payment attempts;
- 100k webhook deliveries;
- 250k settlement lines;
- skewed “hot” wallets for contention tests;
- multiple merchant tenants and roles.

These are test data sizes, not claims about production customers.

Workloads:

- 70% balance/activity reads, 15% internal transfers, 5% external payout acceptance, 5% merchant payment/refund commands, 5% workforce/search/report operations—or document a better model.
- burst duplicate retries;
- same-source wallet contention;
- merchant credential rate-limit abuse;
- delayed provider callbacks;
- reconciliation while transaction ingestion continues.

## Performance acceptance posture

Do not hard-code vanity TPS as the primary gate. Gate on:

- zero financial invariant violations;
- defined error rate;
- no unbounded queue/lock growth after load stops;
- latency targets justified by user journey;
- stable memory and goroutine counts;
- successful recovery from saturation;
- documented bottleneck and next capacity trigger.

Suggested initial reference targets may be set in the benchmark plan, then revised using measured hardware.

## Security attack catalog

- BOLA/BFLA/BOPLA across every resource and field;
- CSRF, session fixation, stale/revoked session, step-up bypass;
- request smuggling/proxy header confusion within deployment constraints;
- SSRF and DNS rebinding in webhooks/provider configuration;
- SQL/NoSQL/command/template/log/CSV injection;
- stored/reflected DOM XSS in notes, metadata, narration, provider messages;
- mass assignment and unknown-field attacks;
- decompression bombs, oversized bodies, slow clients, pagination/resource exhaustion;
- replay and signature canonicalization confusion;
- tenant switch/cache leakage;
- ID enumeration and timing side channels;
- race conditions in holds, limits, captures, refunds, approvals, close, and deletion;
- dependency/supply-chain compromise simulation;
- secret committed to repo/image/log;
- backup/object/export authorization;
- privilege escalation through invitation/role change;
- audit suppression/tampering;
- unsafe provider response/callback consumption.

## Tests most agents will skip

1. Load test followed by full ledger and projection recomputation.
2. Goroutine leak test under repeated provider timeout/cancellation.
3. Connection-pool exhaustion while high-priority ledger command competes with reports.
4. Slowloris/body read timeout and response write timeout.
5. Broker unavailable for hours; outbox grows, API remains bounded, recovery drains safely.
6. Redis loss under active rate limits; financial truth unaffected and fail-open/closed policy documented per endpoint.
7. IdP outage during active sessions and step-up-required command.
8. Clock skew across API/worker/simulator affects signatures and expiry; tolerances are bounded.
9. Restore with rotated key version references.
10. Backup includes malicious historical payload; restore and rendering remain safe.
11. Fuzz test seed from every prior parsing/security bug becomes permanent regression corpus.
12. Query-plan regression with representative cardinality, not empty tables.
13. Deadlock detector exercise and retry storms under contention.
14. Cancellation propagates through Go contexts without abandoning an unknown provider outcome incorrectly.
15. SLO metric excludes business denials but not server-side incorrect failures.
16. Feature flag changes mid-transaction cannot alter accounting template after acceptance.
17. Security scanner failure blocks release; exception requires expiry, owner, and compensating controls.
18. CI artifact digest differs from deployed digest; deployment verification catches it.

## Deliverable evidence

- load-test report;
- flame/profile analysis for at least one bottleneck;
- chaos/game-day reports;
- point-in-time restore report;
- security control mapping;
- finding and remediation register;
- SBOM and signed provenance;
- signed release manifests;
- SLO dashboard and alert tests;
- residual risk register;
- updated architecture capacity thresholds.

## Acceptance gate

Phase 12 passes only when the project can be rebuilt and restored, hostile scenarios are demonstrated, security controls are independently exercised, measured performance is reproducible, all financial invariants pass after stress, and known residual risks are stated plainly.

## X content pillars

### Pillar A — “I load-tested the invariants, not just the endpoint”

- Publish hardware/workload.
- Show p95/p99 and lock/pool data.
- Recompute the ledger after load.
- Discuss bottleneck honestly.

### Pillar B — “My backup worked. Then I tested the dangerous part.”

- Restore before outbox marker.
- Replay duplicated events.
- Prove one financial effect.

### Pillar C — “A goroutine leak can become a payment outage”

- Show provider-timeout profile.
- Fix cancellation/resource ownership.
- Compare before/after under load.

### Pillar D — “Security test findings I did not hide”

- Publish a sanitized real finding.
- Threat, exploit, fix, regression test, residual risk.
- Demonstrate engineering maturity over perfection theatre.

### Pillar E — “Why TPS without a workload model is marketing”

- Explain dataset, contention, auth, audit, and percentiles.
- Show a misleading empty-table benchmark versus representative run.

## Do not waste time on

- enormous synthetic TPS on a laptop with correctness disabled;
- active-active multi-region implementation;
- buying every scanner;
- chasing zero low-value findings while high-risk logic lacks manual tests;
- chaos in production-like shared environments without guardrails;
- publishing secrets, real exploit paths against third parties, or unredacted security evidence.
