# Phase 03 — Immutable double-entry ledger core

## Outcome

Implement a single-currency-per-account, multi-currency chart of accounts; immutable journal and postings; synchronous balance projection; controlled posting templates; reversals; period controls; and independent verification.

## Why this phase is the project’s strongest signal

A wallet without a rigorous ledger is a balance-changing CRUD application. This phase proves accounting knowledge, database transaction judgement, concurrency control, invariants, auditability, and recovery from financial corruption attempts.

## Dependencies

Phase 00. Identity actor context from Phase 01 should be available for audit, but customer features are not required.

## Chart of accounts

Initial platform accounts per currency:

- settlement bank asset;
- provider receivable asset;
- payout clearing/payable;
- customer wallet liability control;
- merchant payable liability;
- fee revenue;
- provider fee expense;
- FX bridge/clearing;
- rounding gain/loss;
- suspense;
- chargeback/dispute reserve if used later.

Customer and merchant subaccounts are individual ledger accounts or a documented subledger model. Reference implementation uses individual accounts for clarity.

## Functional requirements

### Account lifecycle

- `LED-001` Create accounts only through approved account templates.
- `LED-002` Account class, currency, normal side, owner, and negative-balance policy become immutable after first posting.
- `LED-003` Frozen accounts reject new postings except specifically authorized correction templates.
- `LED-004` Closing requires zero balance and no active dependencies; closed accounts remain queryable.

### Journal posting

- `LED-010` Posting request includes journal type, business reference, entries, actor, correlation, effective time, and idempotency context.
- `LED-011` Every journal has at least two positive postings.
- `LED-012` Debits equal credits per currency.
- `LED-013` Posting currency equals account currency.
- `LED-014` Posted journals and postings are immutable to application roles.
- `LED-015` Journal and balance projection commit atomically.
- `LED-016` Business reference and template rules prevent duplicate economic effects.
- `LED-017` Arbitrary callers cannot provide unrestricted account IDs and posting sides; application modules use typed templates.
- `LED-018` Metadata is allowlisted, size-limited, and free of sensitive data.

### Reversals and adjustments

- `LED-020` Reversal creates a new journal with inverted postings and link to original.
- `LED-021` A journal can be fully reversed at most once unless a documented partial-adjustment template applies.
- `LED-022` Reversal validates period state, actor permission, approval, and current business lifecycle.
- `LED-023` Closed-period corrections use a current-period adjustment with original effective-date reference, never row mutation.

### Projection and verification

- `LED-030` `account_balances` updates in the posting transaction.
- `LED-031` Projection includes last applied sequence/version.
- `LED-032` Rebuild job independently derives balances from postings.
- `LED-033` Any unexplained variance creates a critical incident and blocks affected money movement according to runbook.
- `LED-034` Verification report is checksummed, signed, and stored immutably.

### Period management

- `LED-040` Define accounting periods by tenant/book and currency policy.
- `LED-041` Soft close blocks ordinary backdated posting and reports pending dependencies.
- `LED-042` Hard close requires maker-checker and signed trial balance evidence.
- `LED-043` Reopen is exceptional, approved, reasoned, alerted, and audited.

### Multi-currency and FX foundation

- `LED-050` One journal never relies on cross-currency debit-credit equality.
- `LED-051` FX operation contains a quote and linked per-currency journals.
- `LED-052` Rounding differences post explicitly.
- `LED-053` Exchange-rate values use decimal/rational representation with defined precision, never binary float.

## Controlled posting interface

Example Go-facing conceptual API:

```go
type PostingTemplate interface {
    Validate(ctx Context, input any) error
    Build(ctx Context, input any) (JournalDraft, error)
}

type Ledger interface {
    Post(ctx context.Context, command PostCommand) (PostedJournal, error)
    Reverse(ctx context.Context, command ReverseCommand) (PostedJournal, error)
    Balance(ctx context.Context, accountID AccountID) (Balance, error)
}
```

Modules do not insert directly into `journal_entries`, `postings`, or `account_balances`.

## API surface

Workforce/read-only initially:

- `GET /v1/ledger/accounts/{account_id}`
- `GET /v1/ledger/accounts/{account_id}/balance`
- `GET /v1/ledger/journals/{journal_id}`
- `GET /v1/ledger/journals/{journal_id}/postings`
- `GET /v1/ledger/trial-balances`
- `POST /v1/ledger/reversal-requests` privileged, approval-gated
- `GET /v1/ledger/verification-runs`

Do not expose a public “create arbitrary journal” endpoint.

## Frontend requirements

### Ledger explorer

- Search by journal ID, business reference, account code, date, and safe owner identifier.
- Journal view displays debit/credit table, totals, currency, template, actor, effective/posted dates, reversal link, correlation, and associated business object.
- Account view explains account class and normal balance.
- Trial balance view clearly proves totals.
- Projection verification view highlights variance without offering silent repair.
- Reversal request UI displays economic consequence and requires reason/approval.

### Accessibility

Tables have semantic headers and summaries. Debit/credit and variance are not communicated by colour alone. Large integer amounts format correctly without losing precision.

## Database design requirements

- Use explicit transaction function and deterministic lock order.
- Application role lacks update/delete privileges on posted financial tables.
- Consider deferred constraint trigger or controlled database function for aggregate balance validation.
- Index by business reference, account and sequence, posted time, reversal relation, and tenant.
- Partition only after measured need; partitioning cannot weaken uniqueness or verification.

## Tests most agents will skip

### Property and model tests

1. Generate thousands of valid random journals and prove per-currency balance.
2. Generate invalid journals with one missing/duplicated/negative/overflow posting and prove rejection.
3. Apply random journal/reversal sequences and compare projection to independent model.
4. Prove reversal of reversal policy and duplicate reversal constraints.
5. Mutation test: intentionally remove a sign/account-class rule and prove tests fail.

### Concurrency and failure tests

6. Two postings lock the same accounts in opposite input order; implementation avoids deadlock through canonical ordering.
7. Serialization failure after journal draft causes full safe retry without duplicate journal.
8. Process crashes after insert but before commit; no partial rows or projection change.
9. Process commits but connection drops before response; retry returns same journal through idempotency/business reference.
10. Verification job runs while postings continue and uses a defined consistent snapshot.
11. Database role attempts update/delete on posted rows and is denied.
12. Direct insertion bypass attempt fails due permissions/constraints.
13. Concurrent close-period and backdated posting produce one coherent result.

### Numeric and temporal tests

14. Maximum supported amount does not overflow Go or PostgreSQL arithmetic.
15. Zero and negative posting amounts are rejected.
16. NGN/USD exponent and formatting are correct.
17. Effective date differs from posted date near month/year boundary.
18. Leap second-like or malformed timestamp input is rejected safely.
19. Sequence ordering remains stable for equal timestamps.
20. FX rounding accumulates only in explicit rounding account.

### Audit/tamper tests

21. Modify a verification fixture and signed manifest validation fails.
22. Recompute a closed period and compare exact account totals.
23. Restore to a point before outbox publication and prove journal does not duplicate when events replay.

## Observability and alerts

Metrics:

- journals posted by template;
- posting duration and lock wait;
- deadlocks and serialization retries;
- rejected unbalanced drafts;
- projection variance;
- suspense balance and age;
- reversal requests and approvals;
- closed-period adjustment count.

Critical alerts:

- any committed imbalance;
- any projection variance;
- unauthorized write attempt on immutable tables;
- unexpected negative account where policy forbids it;
- failed trial balance close.

## Acceptance gate

A reviewer can inspect the chart of accounts, post controlled synthetic journal templates, trigger invalid journals, run concurrent posting tests, simulate commit-response loss, reverse an entry through approval, close a period, rebuild balances, and validate a signed verification report.

## X content pillars

### Pillar A — “A wallet balance is a liability, not a number on the user table”

- Explain asset versus customer liability.
- Show the cash-in and internal-transfer journal.
- Demonstrate trial balance.

### Pillar B — “My ledger has no update endpoint”

- Show database permissions and append-only model.
- Demonstrate a failed SQL update under application role.
- Explain compensating entries.

### Pillar C — “The network timed out after commit. Did I charge twice?”

- Demonstrate commit-then-response-loss.
- Retry with same idempotency key.
- Trace the single journal and original response.

### Pillar D — “Tests that would catch a fake double-entry ledger”

- Property-based random journals.
- Independent projection rebuild.
- Role-level tamper test.
- Concurrent period-close race.

### Short-form posts

- Debit/credit visual explainer using one real Atlas flow.
- “Why I lock ledger accounts in sorted order.”
- A signed daily balance-verification report.

## Do not waste time on

- blockchain immutability claims;
- arbitrary journal creation UI;
- Kafka before local transaction invariants work;
- complex chart-of-accounts customization;
- floating-point exchange rates;
- storing only a current balance without postings;
- claiming the hash chain makes database controls unnecessary.
