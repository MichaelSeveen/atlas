# Atlas phase-by-phase content calendar

## Cadence

This is sequence-based rather than date-based. Publish only after the evidence exists. A normal phase can produce three to five posts over one or two weeks; difficult phases may take longer. Quality and reproducibility outrank consistency theatre.

## Phase 00 — Engineering foundation

| Content ID | Format | Hook | Evidence |
|---|---|---|---|
| CNT-00-01 | Thread | “Why I did not start my fintech portfolio with the wallet UI” | architecture map, risk-first roadmap |
| CNT-00-02 | Diagram | “A modular monolith can have stronger boundaries than premature microservices” | Go module dependency rules and failing forbidden-import test |
| CNT-00-03 | Failure demo | “The first feature I tested was restore” | database/object restore report and integrity checklist |
| CNT-00-04 | Single post | “My CI rejects floating-point money and domain `time.Now()`” | custom static rule/test output |
| CNT-00-05 | Decision note | “What Atlas deliberately excludes” | scope/non-goal document, no real rails/card data |

## Phase 01 — Identity, access, tenancy, approvals

| Content ID | Format | Hook | Evidence |
|---|---|---|---|
| CNT-01-01 | Thread | “RBAC is not authorization” | object/action/field/tenant policy matrix |
| CNT-01-02 | Diagram | “Why the React app never receives an OAuth access token” | BFF/session trust boundary |
| CNT-01-03 | Adversarial demo | “A hidden admin button is not a security control” | direct API call denied and audited |
| CNT-01-04 | Race demo | “Maker-checker under stale state” | approved payload changes/role revocation execution test |
| CNT-01-05 | Single post | “Tenant isolation must apply before pagination counts” | cross-tenant count/cursor test |

## Phase 02 — Customer, KYC simulation, privacy

| Content ID | Format | Hook | Evidence |
|---|---|---|---|
| CNT-02-01 | Thread | “KYC is a lifecycle, not an upload form” | KYC state machine and tier effects |
| CNT-02-02 | Security post | “My portfolio never stores a real identity document” | simulator fixture/data-minimisation model |
| CNT-02-03 | Diagram | “Consent is not a universal legal checkbox” | notice/lawful-basis evidence model |
| CNT-02-04 | Adversarial demo | “Contact change can become account takeover” | step-up, delay, session revocation test |
| CNT-02-05 | Operations UI | “A manual review must preserve factors and decision history” | reviewer case timeline |

## Phase 03 — Ledger core

| Content ID | Format | Hook | Evidence |
|---|---|---|---|
| CNT-03-01 | Visual explainer | “A wallet balance is a liability, not a number on a user row” | chart of accounts and cash-in journal |
| CNT-03-02 | Demo | “My ledger has no update endpoint” | failed application-role SQL update and compensating entry |
| CNT-03-03 | Failure demo | “The network died after commit. Did I post twice?” | failpoint trace and one journal |
| CNT-03-04 | Test thread | “Four tests that expose a fake double-entry ledger” | property, model, tamper, period-close tests |
| CNT-03-05 | Performance note | “Why ledger accounts are locked in canonical order” | contention/deadlock experiment |

## Phase 04 — Wallets, balances, holds

| Content ID | Format | Hook | Evidence |
|---|---|---|---|
| CNT-04-01 | Thread | “Available balance and ledger balance answer different questions” | exact formulas and UI states |
| CNT-04-02 | Race demo | “Two withdrawals, one balance” | concurrent reservation test |
| CNT-04-03 | State-machine visual | “A hold must end exactly once” | capture/release/expiry race model |
| CNT-04-04 | Frontend post | “How to display pending money without lying to users” | transaction/balance UI with ambiguous states |
| CNT-04-05 | Verification post | “I corrupted the balance projection on purpose” | independent rebuild and protective circuit |

## Phase 05 — Risk policy and limits

| Content ID | Format | Hook | Evidence |
|---|---|---|---|
| CNT-05-01 | Thread | “A fraud score without explainable factors is weak operations software” | versioned rule decision record |
| CNT-05-02 | Model demo | “Limits are temporal state, not one `if` statement” | rolling-window/velocity boundary tests |
| CNT-05-03 | Operations UI | “Why a risk analyst needs history, not a red badge” | case evidence timeline and comparison view |
| CNT-05-04 | Adversarial demo | “A policy changed while a transfer was waiting” | ruleset version and execution-time recheck |
| CNT-05-05 | Judgement post | “Where I refused to use an LLM in risk decisions” | bounded AI/non-goal explanation |

## Phase 06 — Internal transfers

| Content ID | Format | Hook | Evidence |
|---|---|---|---|
| CNT-06-01 | Sequence diagram | “An internal transfer is one atomic economic event” | command, hold/post, journal, outbox sequence |
| CNT-06-02 | Failure demo | “Double-clicking Send Money created one transfer” | browser + idempotency race |
| CNT-06-03 | Test thread | “I generated random transfer/reversal sequences” | model-based test report |
| CNT-06-04 | Frontend post | “The confirmation screen is a security boundary” | destination/amount/fee/step-up UX |
| CNT-06-05 | API post | “Why a transfer has a durable status resource” | OpenAPI timeout/retry contract |

## Phase 07 — External money movement

| Content ID | Format | Hook | Evidence |
|---|---|---|---|
| CNT-07-01 | Thread | “A provider timeout is an unknown transfer, not a failed transfer” | ambiguous-state model and simulator |
| CNT-07-02 | Failure demo | “Provider accepted, worker died” | durable attempt/request fingerprint recovery |
| CNT-07-03 | Security post | “Signed callbacks still need deduplication and lifecycle validation” | signature/replay/out-of-order tests |
| CNT-07-04 | Architecture post | “Provider adapters normalize errors without erasing evidence” | adapter contract and protected raw evidence |
| CNT-07-05 | Operations UI | “The transaction timeline spans two systems that disagree” | provider attempt/callback/reconciliation view |

## Phase 08 — Merchant payments and webhooks

| Content ID | Format | Hook | Evidence |
|---|---|---|---|
| CNT-08-01 | State-machine thread | “Authorization, capture, settlement, and refund are not one status” | payment lifecycle diagram |
| CNT-08-02 | Race demo | “Two captures raced; total captured stayed bounded” | concurrency test and postings |
| CNT-08-03 | Security demo | “The hardest part of webhooks is not HMAC—it is safe outbound networking” | DNS rebinding/redirect/private-IP test |
| CNT-08-04 | Contract post | “I sign the bytes, not a re-serialized JSON object” | signature fixture in Go/TypeScript |
| CNT-08-05 | Product post | “A good merchant dashboard explains gross, fees, refunds, and net” | settlement/payment dashboard |

## Phase 09 — Settlement and reconciliation

| Content ID | Format | Hook | Evidence |
|---|---|---|---|
| CNT-09-01 | Thread | “A payment is not finished when your API says success” | settlement lifecycle and control accounts |
| CNT-09-02 | Demo | “I changed one provider row by one kobo” | mismatch detection and exception case |
| CNT-09-03 | Test post | “Reconciliation must be deterministic and restartable” | crash/resume and exact rerun result |
| CNT-09-04 | Finance UI | “The rarest high-signal screen in a fintech portfolio” | exception queue, aging, evidence, approval |
| CNT-09-05 | Security post | “CSV is both a parser and spreadsheet attack surface” | malformed/import/formula corpus |

## Phase 10 — Operations, finance, support, cases

| Content ID | Format | Hook | Evidence |
|---|---|---|---|
| CNT-10-01 | Product thread | “The most important frontend in my wallet is not the customer app” | transaction inspector walkthrough |
| CNT-10-02 | Security post | “Admin panels are privileged clients, not trusted bypasses” | server authorization and field masking |
| CNT-10-03 | Demo | “There is no Edit Status button” | command/state transition/approval workflow |
| CNT-10-04 | Race demo | “Two analysts resolved the same case differently” | ETag conflict and explicit re-read UX |
| CNT-10-05 | Incident post | “How break-glass access is constrained and reviewed” | time-bound scope and audit report |

## Phase 11 — Statements, audit, reporting, data rights

| Content ID | Format | Hook | Evidence |
|---|---|---|---|
| CNT-11-01 | Thread | “A statement is a temporal systems test” | opening/movement/closing oracle and boundary cases |
| CNT-11-02 | Security demo | “CSV export can execute formulas” | inert corpus in spreadsheet |
| CNT-11-03 | Privacy post | “Deleting a customer does not mean deleting the ledger” | pseudonymization/retention model |
| CNT-11-04 | Integrity post | “Tamper-evident audit logs still need a failure model” | altered/missing/reordered manifest tests |
| CNT-11-05 | Differential test | “PDF, CSV, and JSON must agree to the minor unit” | cross-format report result |

## Phase 12 — Reliability, DR, performance, security

| Content ID | Format | Hook | Evidence |
|---|---|---|---|
| CNT-12-01 | Benchmark thread | “I load-tested the invariants, not just the endpoint” | workload, p95/p99, post-run verification |
| CNT-12-02 | DR report | “My backup restored. Then the dangerous work started.” | journals/holds/outbox/provider/object reconciliation |
| CNT-12-03 | Go engineering post | “A goroutine leak can become a payment outage” | soak/profile before and after fix |
| CNT-12-04 | Security report | “Security findings I did not hide” | sanitized findings and regression links |
| CNT-12-05 | Judgement post | “TPS without a workload model is marketing” | workload assumptions and limitations |

## Phase 13 — Portfolio release

| Content ID | Format | Hook | Evidence |
|---|---|---|---|
| CNT-13-01 | Demo video | “The seven-minute Atlas failure story” | payment -> timeout -> one posting -> webhook -> reconciliation -> refund |
| CNT-13-02 | Scope post | “What I deliberately did not build” | non-goals and rationale |
| CNT-13-03 | Evidence post | “Every portfolio claim has a receipt” | claims-to-evidence ledger |
| CNT-13-04 | Code review | “One transfer, reviewed across React, Go, PostgreSQL, events, and operations” | guided code path |
| CNT-13-05 | Retrospective | “What broke, what I changed, and what remains unproven” | postmortem and limitations |

## Reuse model

One strong phase artifact can produce:

- a technical thread;
- a 60–120 second failure clip;
- a diagram carousel;
- a longer engineering note;
- a portfolio evidence card.

Reuse the evidence, not identical wording. The durable repository artifact is the source of truth.
