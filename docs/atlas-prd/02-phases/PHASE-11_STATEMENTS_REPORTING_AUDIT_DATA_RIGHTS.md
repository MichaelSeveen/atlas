# Phase 11 — Statements, reporting, audit evidence, and data rights

## Outcome

Generate accurate customer and merchant statements, finance reports, trial balances, safe exports, tamper-evident audit evidence, data-access/portability packages, retention execution, and account closure/pseudonymisation workflows.

## Why this phase is high-signal

Reporting exposes temporal, numeric, privacy, and audit weaknesses. A statement must remain reproducible after profile corrections, reversals, period close, timezone boundaries, key rotation, and software changes. Data rights must coexist with financial-record retention rather than pretending all rows can be deleted.

## Dependencies

Phases 09 and 10.

## Reporting principles

- Reports are derived from immutable or versioned source data with an explicit cutoff.
- Generated artifacts carry report type/version, input watermark, code revision, timezone, locale, checksum, and generation time.
- A regenerated historical report can explain differences from an earlier version.
- Exports never become an authorization bypass.

## Functional requirements

### Customer statements

- `RPT-001` Statement period has explicit inclusive/exclusive boundaries and timezone.
- `RPT-002` Opening balance + period movements = closing balance using ledger-derived values.
- `RPT-003` Holds appear only according to statement definition; pending and posted are clearly separated.
- `RPT-004` Reversals and refunds are separate entries linked to originals.
- `RPT-005` Transaction descriptions use versioned rendering rules and safe historical counterparty snapshots.
- `RPT-006` Statement generation is asynchronous, idempotent, and resumable.
- `RPT-007` PDF/CSV/JSON outputs agree on totals and identifiers.
- `RPT-008` Download requires current authorization and short-lived access; object URL is not durable public access.

### Merchant reports

- `RPT-010` Payment, refund, fee, settlement, and reconciliation reports share consistent identifiers.
- `RPT-011` Gross - refunds - fees +/- adjustments = net settlement under documented formula.
- `RPT-012` Reports distinguish transaction date, effective date, posted date, and settlement date.
- `RPT-013` Export fields and masking respect merchant role/scopes.

### Finance and audit reports

- `RPT-020` Trial balance by book/currency/period.
- `RPT-021` Journal export with posting lines and business references.
- `RPT-022` Settlement close and reconciliation exception aging.
- `RPT-023` Suspense balance and movement.
- `RPT-024` Privileged access and action report.
- `RPT-025` Control evidence report maps requirements to tests/artifacts.

### Audit log

- `AUD-010` Audit events are append-only to application roles.
- `AUD-011` Event contains actor, actor type, tenant, session assurance, action, target, decision, reason, source IP/network metadata under privacy policy, correlation, case/approval, timestamp, and safe before/after references.
- `AUD-012` Audit search is permissioned, paginated, and export controlled.
- `AUD-013` Daily canonical event batches produce hash manifest and managed-key signature.
- `AUD-014` Verification detects missing, reordered, modified, or substituted events within the covered design limits.
- `AUD-015` Audit pipeline failure policy is explicit for high-risk mutations.

### Data access and portability

- `DSR-001` User requests access/export under authentication and step-up.
- `DSR-002` Export job scopes data, excludes security-sensitive and third-party data, and records redactions.
- `DSR-003` Package includes structured machine-readable data and plain-language index.
- `DSR-004` Package is encrypted or delivered through short-lived authenticated channel.
- `DSR-005` Generation, download, expiry, and deletion are audited.

### Correction, closure, and deletion

- `DSR-010` Correctable current profile data changes without rewriting historical financial facts.
- `DSR-011` Closure checks balances, holds, pending transfers, disputes, cases, merchant obligations, and retention.
- `DSR-012` Sessions, API credentials, webhooks, and active beneficiaries are revoked/disabled.
- `DSR-013` Removable personal data is deleted or pseudonymised by policy while ledger/audit linkage is preserved through non-public internal references.
- `DSR-014` Retention job has dry run, approval for high-impact deletion, immutable execution report, and restartability.
- `DSR-015` Backups and derived stores have documented expiry or tombstone propagation strategy.

## API surface

Statements/reports:

- `POST /v1/statements`
- `GET /v1/statements`
- `GET /v1/statements/{statement_id}`
- `POST /v1/reports`
- `GET /v1/reports/{report_id}`
- `POST /v1/reports/{report_id}/download-grants`

Audit:

- `GET /v1/audit-events`
- `GET /v1/audit-manifests`
- `POST /v1/audit-manifests/{manifest_id}/verification-runs`

Data rights:

- `POST /v1/data-rights/requests`
- `GET /v1/data-rights/requests/{request_id}`
- `POST /v1/data-rights/requests/{request_id}/verification`
- `POST /v1/data-rights/requests/{request_id}/download-grants`
- `POST /v1/account-closure-requests`

## Frontend requirements

### Customer

- Statement period selector with timezone/availability explanation.
- Generation progress and expiry.
- Accessible transaction table and download options.
- Data request page that explains categories, expected timeline, exclusions, and secure delivery.
- Closure checklist with unresolved blockers and retained-record explanation.

### Merchant

- Report builder limited to safe indexed filters.
- Column dictionary and accounting formula.
- Export job state and expiration.

### Audit/security

- Audit timeline with actor, action, target, decision, reason, assurance, correlation, and linked case/approval.
- Manifest verification view and integrity failure workflow.
- Control evidence matrix linking requirements to test/artifact URLs.

## Tests most agents will skip

### Statement correctness

1. Period boundary exactly at midnight UTC and configured local timezone.
2. Leap year, month-end, year-end, and DST boundary.
3. Transaction effective in period but posted after period; statement policy is consistent.
4. Reversal in later period links original and balances both periods correctly.
5. Profile/counterparty display name changes after transaction; historical rendering rule is deterministic.
6. Very large amount preserves integer precision in JSON, CSV, React, and PDF.
7. Statement generated twice with same version/cutoff is checksum-identical except permitted metadata.
8. PDF, CSV, and JSON totals match independent ledger query.
9. Active holds at period end are represented according to documented definition, not double-counted.
10. Closed wallet statement remains available under retention policy.

### Export security

11. CSV cells beginning `=`, `+`, `-`, `@`, tab, CR are neutralized.
12. Spreadsheet/CSV delimiter and locale do not change numeric meaning.
13. Download URL cannot be used after role revocation or expiry.
14. Report generation request cannot inject SQL through filters or choose unauthorized columns.
15. Huge export is paged/streamed and cannot exhaust memory.
16. Shared transaction redacts another customer’s private data.

### Audit integrity

17. Modify one event, remove one event, reorder events, substitute manifest, or rotate key; verifier identifies expected failure/success.
18. Two workers build same daily manifest; one canonical result.
19. Late event after manifest cutoff goes to correct next/adjustment batch under policy.
20. Audit event write outage during privileged action exercises fail-closed/durable-spool policy.
21. Audit search cannot reveal denied tenant data through counts.

### Data rights and retention

22. Export initiated before user contact change and downloaded after; package scope/cutoff is explicit and access reauthorized.
23. Closure races with incoming cash-in or delayed provider callback; no orphan funds.
24. Retention job crashes mid-batch and resumes idempotently.
25. Deletion preserves financial references and verification jobs still pass.
26. Tombstoned identity does not reappear from analytics/search index rebuild.
27. Backup restore reintroduces data past deletion date; documented post-restore retention reconciliation removes it.
28. Data package excludes secrets, password/authenticator material, internal fraud rules, and unrelated persons.

## Observability and alerts

Metrics:

- statement/report generation time, queue age, size, failure;
- download grants and expiry;
- export volume by actor/tenant;
- audit event ingest/manifest lag;
- integrity verification failures;
- data-right request age;
- retention dry-run/delete counts;
- closure blocker categories.

Alerts:

- audit pipeline gap or signature failure;
- unusually large/sensitive export;
- expired report still accessible;
- data-right SLA threshold;
- retention job variance/failure;
- statement balance formula failure;
- closure with unresolved financial state attempt.

## Acceptance gate

A reviewer can generate a period statement, independently reconcile opening/movements/closing, test month-end and reversal cases, produce safe CSV/JSON/PDF, verify and tamper with an audit manifest, submit a data-export request, close a synthetic account with blockers, run retention dry-run/execution, and prove ledger verification remains intact.

## X content pillars

### Pillar A — “A bank statement is a temporal systems test”

- Show effective/posting/settlement dates.
- Demonstrate a reversal across month boundary.
- Reconcile opening plus movements to closing.

### Pillar B — “CSV export is an injection surface”

- Demonstrate formula payload.
- Open safe export.
- Explain field allowlists and async jobs.

### Pillar C — “The right to deletion does not mean deleting the ledger”

- Explain pseudonymisation and retained financial references.
- Demonstrate closure blockers and retention report.
- Avoid legal overclaim.

### Pillar D — “Tamper-evident audit logs need a failure model”

- Show what the hash/signature design can detect.
- Tamper, delete, reorder, and verify.
- Explain what it cannot guarantee alone.

## Do not waste time on

- a general BI product;
- dozens of chart formats;
- rendering pixel-perfect branded statements before totals are proved;
- long-lived public download links;
- pretending pseudonymisation is always deletion;
- audit hash chains without access control, backups, and external key protection.
