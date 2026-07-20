# RFC — [Change title]

- **Status:** Draft | Review | Accepted | Implemented | Withdrawn
- **Authors/owners:**
- **Reviewers:** Product, ledger, security, privacy, reliability, frontend, operations as applicable
- **Target phase/release:**
- **Related requirements/threats/ADRs:**

## Executive summary

What capability changes, why now, and what is the strongest constraint?

## Problem and evidence

Current behaviour, measured pain/failure, user/operator impact, and why existing design is insufficient.

## Goals and non-goals

### Goals

- ...

### Non-goals

- ...

## Actors and journeys

Include customer/merchant/workforce/machine paths, adverse/ambiguous/recovery paths, and accessibility considerations.

## Domain and financial model

State machines, invariants, accounts/journals/holds/fees/settlement/reconciliation/correction.

## Proposed architecture

Components, ownership, trust boundaries, data flow, transaction boundaries, asynchronous states, contracts, deployment.

## Security and privacy analysis

Threats, authentication/authorization, abuse, data classification, crypto/key, audit, privacy rights, residual risks.

## API/event/data changes

OpenAPI/AsyncAPI, schemas, compatibility, migrations, idempotency, ETags, error/retry semantics.

## Failure and recovery analysis

At each boundary: what can fail, what is durable, what the caller sees, how retry/replay works, and operator action.

## Alternatives and rejected shortcuts

Include “do nothing” and simpler options. Explain why complexity is justified.

## Verification plan

Independent oracles, property/model/mutation/fuzz/concurrency/failpoint/security/performance/restore tests.

## Observability and operations

SLIs/SLOs, metrics, alerts, traces/logs/audit, dashboards, runbooks, on-call/operator load.

## Rollout and migration

Expand/backfill/switch/contract, feature flags, shadow/differential validation, rollback/forward fix.

## Open questions and decisions required

- ...

## Evidence and content opportunity

What durable artifact and engineering lesson will be publishable after verification?
