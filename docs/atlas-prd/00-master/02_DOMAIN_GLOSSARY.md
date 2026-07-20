# Domain glossary

## Accounting and ledger

**Account** — A ledger bucket with an owner, currency, account class, normal balance, and lifecycle status.

**Account class** — Asset, liability, equity, revenue, or expense.

**Available balance** — Funds that may be spent after subtracting active reservations from the posted spendable balance.

**Balance projection** — A synchronously maintained, rebuildable representation of ledger-derived balances used for fast decisions and reads.

**Compensating entry** — A new journal entry that economically corrects a prior entry without deleting or editing history.

**Credit** — A posting side whose effect depends on account class. Credits increase liabilities, equity, and revenue, and decrease assets and expenses.

**Debit** — A posting side whose effect depends on account class. Debits increase assets and expenses, and decrease liabilities, equity, and revenue.

**Journal entry** — An immutable business accounting event containing two or more postings whose debits equal credits per currency.

**Ledger account** — A single-currency account in the financial book. A wallet is represented by one or more ledger accounts, not by a mutable balance column alone.

**Posting** — A debit or credit line within a journal entry.

**Posted balance** — Balance represented by committed ledger postings. It does not automatically account for active holds unless the product uses memo accounts.

**Trial balance** — A report proving aggregate debits and credits and presenting account balances for a reporting period.

## Money movement

**Ambiguous outcome** — A request whose final provider outcome is unknown, usually because the caller timed out after the provider may have accepted it.

**Attempt** — A specific submission of a logical payment or transfer to a provider.

**Beneficiary** — A validated destination for an external payout.

**Capture** — Conversion of a reservation or authorised amount into a posted financial movement.

**Clearing** — Exchange and agreement of transaction details before settlement.

**Hold / reservation** — A temporary reduction of spendable funds without final accounting settlement.

**Idempotency key** — A caller-provided identifier that makes safely retrying the same logical mutation return the original effect and response.

**Payment intent** — Merchant-facing object representing the desired payment and its lifecycle independent of provider attempts.

**Payout** — Outbound transfer to an external destination.

**Provider** — External financial rail abstraction. In Atlas all providers are simulators.

**Refund** — Return of all or part of a completed payment. It is not the same as reversing an uncompleted authorisation.

**Reversal** — Cancellation or economic negation of a prior financial effect through an explicit state transition and compensating entries.

**Settlement** — Final exchange of funds between financial participants.

**Transfer** — Movement of value between source and destination accounts.

## Reconciliation

**Break / exception** — An unmatched, duplicate, missing, currency-mismatched, or amount-mismatched record requiring investigation.

**Provider statement** — Immutable external file or payload representing the provider’s account of transactions and settlement.

**Reconciliation run** — Versioned, deterministic comparison of internal records with a specific external dataset.

**Settlement batch** — Group of provider transactions settled together under a provider reference and settlement date.

**Suspense account** — Controlled account used temporarily when the correct accounting destination is not yet known. Suspense use must be visible, aged, and resolved.

## Risk and compliance

**Decision** — Deterministic outcome of a policy evaluation: allow, review, or deny.

**KYC tier** — Product-defined identity-verification level that controls account capabilities and limits. Atlas uses synthetic provider outcomes.

**Maker-checker** — A control requiring a different authorized person to approve a high-risk action initiated by another person.

**Risk factor** — Explainable input that contributed to a policy decision.

**Risk policy version** — Immutable set of rules and configuration used to evaluate a transaction.

**Velocity rule** — Rule that evaluates counts or sums within a time window.

## Security and platform

**BFF** — Backend for Frontend. A server-side web boundary that manages browser sessions and calls internal APIs so access tokens are not stored in browser JavaScript storage.

**Causation ID** — Identifier of the command or event that caused another event.

**Correlation ID** — Identifier tying together requests, commands, jobs, events, and audit records for one business flow.

**Data classification** — Assignment of handling requirements such as public, internal, confidential, sensitive personal, secret, or financial-control data.

**Idempotent consumer** — Event handler that produces the same business result when the same event is delivered multiple times.

**Inbox** — Consumer-side record of processed message identifiers used to prevent duplicate business effects.

**Outbox** — Database table written in the same transaction as domain changes, then relayed to a broker.

**Step-up authentication** — Fresh stronger authentication required immediately before a sensitive action.

**Tenant** — Isolated organization or merchant boundary. Customer wallets belong to the platform tenant; merchant resources belong to merchant tenants.

**Workforce identity** — Staff or operator identity managed separately from customer and merchant identities.
