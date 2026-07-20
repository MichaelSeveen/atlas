# Phase 10 — Operations, finance, support, and case management

## Outcome

Deliver role-specific workforce consoles and controlled operational workflows for investigation, customer support, risk review, reconciliation, restrictions, compensating actions, approvals, notes, work queues, and evidence—without giving staff a hidden database editor.

## Why this phase is high-signal

Real fintech companies employ people who investigate incidents, explain transactions, resolve exceptions, and protect customers. Internal tools are often where authorization, privacy, and financial-control mistakes become most dangerous. A polished, tightly controlled operations product is a stronger frontend signal than another consumer dashboard.

## Dependencies

Phases 05 through 09 and the approval foundation from Phase 01.

## Workforce role boundaries

| Role | Primary capabilities | Explicitly forbidden |
|---|---|---|
| Support | search scoped customer context, explain transactions, open/escalate cases | raw KYC evidence, ledger correction, restriction removal, settlement close |
| Operations | investigate transfers/provider attempts, initiate safe resolution requests | approve own request, change journal rows |
| Risk analyst | review risk/KYC cases, apply risk restrictions, override through policy | settlement close, arbitrary payout completion |
| Finance | reconciliation, suspense, settlement, reports, adjustment requests | customer-contact change, risk-rule override |
| Security auditor | read audit/security evidence and access decisions | business mutations |
| Platform admin | platform config and role administration under approval | customer financial mutation |
| Break-glass | narrowly scoped emergency action | standing everyday access |

## Case model

### Case types

- transaction dispute;
- ambiguous provider outcome;
- reconciliation exception;
- KYC review;
- risk review;
- account restriction review;
- suspected account takeover;
- privacy/data-right request;
- ledger integrity incident;
- merchant webhook incident;
- security incident.

### Case states

`open -> triaged -> assigned -> investigating -> awaiting_customer | awaiting_provider | awaiting_approval -> resolved -> closed`; may reopen with reason.

### Core case properties

- case type and severity;
- subject/customer/merchant/transaction references;
- source and creation reason;
- owner/team and SLA;
- structured evidence links;
- timeline;
- notes with visibility classification;
- tasks;
- decisions and approvals;
- resolution code;
- related cases/incidents;
- audit references.

## Functional requirements

### Search and customer context

- `OPS-001` Search supports exact high-confidence identifiers and constrained fuzzy display-name search.
- `OPS-002` Search results are permission filtered before counts and suggestions.
- `OPS-003` Every customer-context access is audited with purpose or case reference for sensitive roles.
- `OPS-004` PII is masked by default; reveal requires permission, reason, and short-lived state.
- `OPS-005` No staff impersonation. Customer context viewer is read-oriented and clearly indicates workforce mode.
- `OPS-006` Search rate and breadth are monitored to detect browsing/exfiltration.

### Transaction investigation

- `OPS-010` Unified timeline links request, session assurance, authorization, risk, hold, journal, provider attempts, callbacks, events, webhooks, settlement, reconciliation, disputes, and approvals.
- `OPS-011` Timeline distinguishes durable facts from derived interpretation and raw evidence.
- `OPS-012` Every action shows eligibility and why unavailable.
- `OPS-013` Operations can query provider, retry safe jobs, replay webhooks, and request corrections through explicit commands.
- `OPS-014` There is no generic status editor or SQL-like admin tool.

### Cases and notes

- `CAS-001` Case creation is idempotent for same source event where one logical case is intended.
- `CAS-002` Notes are append-only or versioned; edits preserve prior version and actor.
- `CAS-003` Notes have visibility: internal team, restricted security/legal, customer-safe draft.
- `CAS-004` Notes reject active content, detect likely secrets/PII, and render safely.
- `CAS-005` Attachments use restricted object storage, scanning, checksums, retention, and download audit.
- `CAS-006` Assignment and state transitions are optimistic-concurrency protected.
- `CAS-007` Resolution requires structured code and required evidence.

### Operational commands

- `OPS-020` Commands are narrow: release eligible hold, resubmit provider query, request transfer resolution, request restriction change, request ledger reversal, request suspense adjustment, replay webhook.
- `OPS-021` Command handler rechecks current state, authorization, step-up, approval, and idempotency.
- `OPS-022` Command creates actor/reason/case-linked audit event.
- `OPS-023` High-value financial actions require maker-checker.
- `OPS-024` Bulk actions are disabled unless purpose-built with preview, scope, dry run, limit, approval, and rollback/compensation.

### Work queues and SLAs

- `OPS-030` Queue filters are stable, indexed, role scoped, and shareable without exposing unauthorized data.
- `OPS-031` Ownership changes and SLA pauses/resumes are audited.
- `OPS-032` Aging uses explicit business-calendar or elapsed-time rules.
- `OPS-033` No case disappears because a filter or pagination cursor changed; counts and backlog metrics reconcile.

### Break-glass

- `OPS-040` Break-glass activation requires separate strong authentication, reason, ticket/incident, narrow scope, and expiry.
- `OPS-041` Activation pages security owner immediately.
- `OPS-042` Every action is highlighted, separately reviewed, and included in incident closeout.
- `OPS-043` Break-glass cannot disable audit or access raw secrets.

## API surface

Search/context:

- `GET /v1/operations/search`
- `GET /v1/operations/customers/{customer_id}/context`
- `POST /v1/operations/pii-reveal-grants`

Cases:

- `POST /v1/cases`
- `GET /v1/cases`
- `GET /v1/cases/{case_id}`
- `PATCH /v1/cases/{case_id}` with ETag
- `POST /v1/cases/{case_id}/notes`
- `POST /v1/cases/{case_id}/assignments`
- `POST /v1/cases/{case_id}/transitions`
- `POST /v1/cases/{case_id}/attachments`

Operational actions:

- explicit endpoints under the owning resource; no generic `/actions` with arbitrary command text.

## Frontend information architecture

### Global workforce shell

- queue switcher;
- global exact-reference search;
- role and active privilege indicator;
- environment/synthetic-data banner;
- notification center for approvals and escalations;
- keyboard command palette limited to authorized navigation, never hidden mutation.

### Customer context page

- identity and KYC summary with masking;
- account/wallet restrictions;
- balances and active holds;
- recent transaction timeline;
- open cases;
- access reason indicator;
- data classification and reveal state.

### Transaction inspector

Use a vertical timeline plus linked panels:

- business state;
- financial effect;
- risk decision;
- provider evidence;
- settlement/reconciliation;
- customer communications;
- controls/actions.

### Case workspace

- summary and severity;
- structured evidence;
- notes timeline;
- tasks and approvals;
- recommended next safe actions based on state;
- resolution checklist;
- audit pane.

## Tests most agents will skip

1. Support search for broad common name cannot enumerate huge result set or unauthorized tenant.
2. Search result counts do not reveal restricted rows.
3. PII reveal grant expires while page remains open; next data fetch remasks and UI clears cached value.
4. Screenshot/clipboard cannot be fully prevented, but watermark and access audit evidence are present; no false prevention claim.
5. Two analysts edit/assign same case from stale tabs; ETag prevents lost update.
6. Note containing secret pattern, stored XSS, Markdown link attack, ANSI escape, and CSV formula remains safe everywhere.
7. Attachment filename traversal, polyglot, content-type mismatch, oversized file, and malware test file are rejected/quarantined.
8. Maker loses permission before checker approves; execution recheck blocks.
9. Checker approves after resource state changed; payload hash/state precondition blocks.
10. Case resolution and incoming contradictory provider callback race; final state and evidence remain coherent.
11. Support attempts direct operations endpoint despite hidden UI; server denies.
12. Bulk export request exceeds purpose/row limit; requires approval or rejects.
13. Break-glass session expires during long-running operation; operation follows explicit commit boundary and no silent extension.
14. Privileged audit event write fails; high-risk mutation fails closed or follows documented durable fallback.
15. Queue pagination under concurrent assignments does not lose cases; stable cursor and reconciliation metric catch discrepancies.
16. Deleted/closed customer remains accessible only through permitted historical case path with masking.
17. Cross-case attachment reference cannot bypass authorization.
18. Raw provider payload view has separate permission and is never embedded in list response.

## Observability and alerts

Metrics:

- search volume/breadth by role;
- PII reveal requests and denials;
- case backlog, age, SLA breach, reopen rate;
- privileged commands and approvals;
- unsafe action rejections;
- note/attachment security detections;
- break-glass usage;
- operator error and retry rates.

Alerts:

- unusual customer browsing;
- mass PII reveals/exports;
- break-glass activation;
- high-value action without expected approval/audit;
- case backlog/SLA spike;
- repeated denied attempts by workforce identity;
- security attachment detection.

## Acceptance gate

A reviewer can sign in under each role, search with masking, inspect a transaction end-to-end, open and assign a case, add safe notes/evidence, attempt forbidden actions, request and approve a compensating workflow, see stale-state protection, activate a synthetic break-glass exercise, and audit every access/action.

## X content pillars

### Pillar A — “The most important frontend in my wallet is not the customer app”

- Show the operations transaction inspector.
- Explain role-specific decisions and masked data.
- Demonstrate one ambiguous payout investigation.

### Pillar B — “Admin panels are security boundaries”

- Show why hidden buttons are not authorization.
- Cross-role API tests.
- PII reveal expiry and browsing detection.

### Pillar C — “There is no Edit Status button”

- Compare generic admin mutation with narrow commands.
- Show state preconditions, approval, and compensating accounting.

### Pillar D — “Maker-checker under stale state”

- Request action.
- Change resource or permission.
- Attempt approval and show safe rejection.

### Short-form posts

- A redacted transaction timeline screenshot.
- “What a support agent can see versus finance versus risk.”
- A break-glass game-day clip and review checklist.

## Do not waste time on

- a generic low-code admin framework as the final product;
- staff impersonation;
- unrestricted global search;
- editable audit or transaction history;
- chat-like case UI without structured evidence;
- dozens of roles with overlapping unclear permissions;
- bulk actions without dry run and approval.
