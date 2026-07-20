# Reliability, observability, and disaster recovery

## 1. Reliability objective

Atlas must make partial failure visible, bounded, and recoverable. “The request failed” is not a sufficient model when a provider may still move money.

## 2. State-machine requirements

Every long-running object defines:

- states;
- allowed transitions;
- transition owner;
- preconditions;
- financial effect;
- timeout or expiry;
- retry policy;
- terminal states;
- customer-facing explanation;
- operational resolution;
- emitted event;
- audit record.

No general-purpose `status` string may be introduced without a state diagram.

## 3. Failure taxonomy

### Before durable acceptance

No business effect exists. Client may retry with the same idempotency key.

### After durable acceptance, before provider submission

Command and reservation exist. Worker resumes from outbox/job state.

### Provider rejected definitively

Release reservation, move to final failure, record normalized reason, notify caller.

### Provider accepted definitively

Capture/post according to lifecycle and await settlement if applicable.

### Provider outcome ambiguous

Do not submit blindly to another provider. Enter `pending_provider_confirmation`, poll or query by provider idempotency/reference, and expose an operational timeline.

### Callback delayed, duplicated, or out of order

Persist and normalize callback, compare transition version, apply only valid transitions, record duplicate or stale callback safely.

### Internal event duplicated

Inbox prevents duplicate effect. Handler remains safe if inbox record is lost and unique business constraints are the final defense.

## 4. Retry policy

- Bounded exponential backoff with jitter.
- Retry only classified transient errors.
- No retry loop inside an open database transaction.
- Serialization failures restart the whole transaction.
- Provider non-idempotent calls require an external idempotency key or query-before-retry protocol.
- Dead-lettered work has operator ownership, reason, age, and safe replay.

## 5. SLO framework

Targets are design and test targets, not production claims.

Suggested service indicators:

- successful authenticated API availability;
- p95/p99 command acceptance latency;
- percentage of accepted internal transfers posted within target time;
- percentage of external transfers with known terminal/provider-pending status within target time;
- outbox oldest age;
- webhook delivery success within retry window;
- reconciliation batch completion time;
- statement generation completion time;
- ledger projection variance count;
- unresolved reconciliation exception age.

Error budgets must exclude expected business denials but include server errors, timeouts, incorrect states, and missed processing deadlines.

## 6. Observability requirements

### Traces

A financial flow trace links:

- incoming request;
- authentication and authorization decision ID;
- risk decision;
- idempotency lookup;
- database transaction;
- journal/hold identifiers as non-sensitive attributes;
- outbox event;
- worker job;
- provider attempt;
- callback;
- webhook delivery;
- reconciliation result.

Do not put names, emails, phone numbers, identity numbers, tokens, full request bodies, or raw provider payloads into spans.

### Metrics

Technical:

- request rate, errors, duration;
- database pool saturation, lock waits, deadlocks, serialization retries;
- queue depth, lag, redelivery, dead-letter counts;
- provider latency and normalized outcomes;
- webhook delivery attempts and endpoint disablement;
- report job duration and failure;
- cache hit and rate-limit decisions.

Business/control:

- journal imbalance attempts;
- projection variance;
- negative available balance prevention count;
- idempotency conflicts;
- ambiguous provider outcomes by age;
- stuck state count;
- reconciliation exceptions by type and age;
- suspense balance and age;
- privileged action and approval counts;
- denied cross-tenant access attempts;
- sensitive export volume.

### Logs

- structured JSON;
- stable event name and severity;
- request, correlation, causation, actor pseudonymous ID, tenant, module, and outcome;
- redaction at source;
- log injection resistance;
- no secret or raw PII;
- retention by data classification.

## 7. Alert design

Alerts require an owner, severity, threshold rationale, runbook, and test.

Page-worthy examples:

- non-zero ledger imbalance or projection variance;
- unexpected negative available balance;
- sustained inability to post accepted transfers;
- database recovery or replication failure;
- widespread callback signature failures;
- leaked-secret detection;
- privileged break-glass use;
- settlement close inconsistency.

Ticket-worthy examples:

- growing reconciliation exception age;
- repeated endpoint webhook failures;
- elevated serialization retries;
- dormant API keys;
- nearing retention or key-rotation deadline.

## 8. Backup and restore

### Required capabilities

- encrypted full and incremental backups;
- point-in-time recovery for PostgreSQL;
- versioned infrastructure and migration definitions;
- object-storage versioning/immutability for evidence files;
- key backup and recovery procedures consistent with key policy;
- isolated restore environment;
- checksum and invariant verification after restore.

### Design targets

- financial database RPO target: no more than five minutes in the production reference design;
- service restoration RTO target: sixty minutes for the production reference design;
- portfolio environment must perform at least one documented point-in-time restore and one full environment rebuild.

### Restore validation

After restore:

1. run schema and migration consistency checks;
2. recompute ledger balances;
3. compare object checksums;
4. identify outbox records that may republish;
5. verify inbox and idempotency behaviour under replay;
6. confirm key access and signature validation;
7. execute synthetic critical flows;
8. record recovery time and gaps.

## 9. Disaster scenarios

- API process loss;
- worker loss mid-provider call;
- broker outage;
- Redis loss;
- database failover or restore;
- object storage unavailable;
- identity provider unavailable;
- provider unavailable;
- region-level outage in reference design;
- signing key unavailable;
- corrupted settlement file;
- bad migration;
- compromised deployment credential.

For each, document customer impact, financial safety, degraded mode, recovery, and data reconciliation.

## 10. Chaos and game days

Chaos is executed in controlled environments with synthetic data.

Required game days:

- kill worker after provider accepts but before local state update;
- duplicate every event and callback;
- delay callbacks by hours of simulated time;
- restore database to a point before outbox delivery;
- inject provider statement duplicates and omissions;
- rotate signing keys during delivery retries;
- exhaust database pool;
- force serialization conflicts;
- revoke a workforce role during an open session;
- expire step-up between approval initiation and completion.

The game-day report must include hypothesis, expected control, observed result, telemetry, invariant checks, and corrective action.
