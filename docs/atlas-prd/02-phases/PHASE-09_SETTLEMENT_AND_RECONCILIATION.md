# Phase 09 — Settlement and reconciliation

## Outcome

Build immutable provider-file ingestion, settlement batches, deterministic matching, exception classification, suspense management, manual resolution, period close, reruns, and finance evidence that prove Atlas and provider books agree.

## Why this phase is exceptionally high-signal

Reconciliation is where real financial systems prove that their internal story matches external reality. It exposes missing events, duplicates, amount mismatches, fee errors, settlement timing, and operational weaknesses that happy-path transaction systems hide. Very few portfolio projects build it well.

## Dependencies

Phases 07 and 08.

## Reconciliation model

### Input layers

1. Immutable raw file/object with checksum, provider, schema/version, received time, and uploader/source.
2. Parsed records with line number and canonical normalized fields.
3. Reconciliation run tied to input checksum, matcher version, configuration, and code revision.
4. Matches and exceptions as immutable run outputs.
5. Resolutions as new records; rerun does not edit prior results.

### Exception types

- missing internal;
- missing external;
- duplicate internal;
- duplicate external;
- amount mismatch;
- currency mismatch;
- fee mismatch;
- status mismatch;
- settlement-date mismatch;
- unknown reference;
- late presentment;
- parse/schema failure;
- already settled in another batch.

## Functional requirements

### File ingestion

- `REC-001` Accept only supported provider/schema versions and bounded file sizes.
- `REC-002` Stream upload and parsing; do not load unbounded files into memory.
- `REC-003` Store raw object immutably with cryptographic checksum before processing.
- `REC-004` Detect duplicate file by provider, checksum, and settlement identity.
- `REC-005` Preserve original line and safe parse error without executing formulas or active content.
- `REC-006` Malware/content-type scan where file format permits; CSV handled as untrusted text.

### Settlement batch

- `SET-001` Batch records provider, currency, settlement date, gross amount, fees, net amount, record count, external references, and status.
- `SET-002` Totals are derived from records and compared with provider control totals.
- `SET-003` Batch cannot close with unexplained control-total variance.
- `SET-004` Batch lifecycle: received, validated, reconciling, exceptions, ready_to_close, closed, reopened.
- `SET-005` Close and reopen require maker-checker and signed report.

### Matching

- `REC-010` Exact match uses provider reference, amount, currency, and expected internal object.
- `REC-011` Secondary deterministic rules may use merchant reference and bounded date window.
- `REC-012` Fuzzy candidates never auto-close; they are suggestions with confidence/explanation.
- `REC-013` A record participates in at most one accepted match within the run.
- `REC-014` Matcher is deterministic for same input, database snapshot, configuration, and code version.
- `REC-015` Reconciliation runs use a consistent snapshot or explicit cutoff watermark.

### Financial settlement posting

- `SET-010` Settlement journal moves provider receivable/clearing to settlement-bank asset and recognizes provider expense/fee differences according to accounting template.
- `SET-011` Posting is idempotent per batch and settlement action.
- `SET-012` Settlement cannot post twice across reruns or reopen.
- `SET-013` Differences route to explicit suspense or expense/revenue adjustment through approved template.
- `SET-014` Suspense item has owner, amount, currency, source, age, expected resolution, and audit.

### Exception resolution

- `REC-020` Resolution types are structured: accept timing difference, link correct record, create internal recovery case, provider dispute, approved adjustment, duplicate confirmation, ignore with policy reason.
- `REC-021` Resolution never edits raw input or original run output.
- `REC-022` High-value or financial adjustments require checker approval.
- `REC-023` Reconciliation can be rerun after new internal data while preserving prior evidence.
- `REC-024` Closed period changes require reopen/adjustment process.

### Finance reports

- `REC-030` Produce batch summary, match-rate, exception aging, gross/fee/net comparison, settlement journal, suspense movement, and signed close report.
- `REC-031` Reports identify dataset checksum, run ID, matcher version, cutoff, and approvers.
- `REC-032` Export is safe from CSV injection and follows field masking.

## API surface

- `POST /v1/settlement-files`
- `GET /v1/settlement-files/{file_id}`
- `GET /v1/settlement-batches`
- `GET /v1/settlement-batches/{batch_id}`
- `POST /v1/settlement-batches/{batch_id}/reconciliation-runs`
- `GET /v1/reconciliation-runs/{run_id}`
- `GET /v1/reconciliation-runs/{run_id}/matches`
- `GET /v1/reconciliation-runs/{run_id}/exceptions`
- `POST /v1/reconciliation-exceptions/{exception_id}/resolution-requests`
- `POST /v1/settlement-batches/{batch_id}/close-requests`
- `POST /v1/settlement-batches/{batch_id}/reopen-requests`
- `GET /v1/suspense-items`

## Frontend requirements

### Finance console

- Drag/drop or secure object selection with schema and checksum result.
- Batch overview: provider control totals versus parsed and internal totals.
- Reconciliation summary with match rate and exception categories.
- Exception work queue with source line, normalized fields, internal candidates, timeline, and financial effect.
- Side-by-side external/internal record comparison.
- Resolution wizard that shows accounting consequence and approval requirement.
- Suspense aging dashboard with owner and target date.
- Close checklist and signed report download.
- Rerun comparison: new matches, resolved, new exceptions, unchanged.

### Safety

- No bulk “mark all matched” without deterministic criteria.
- No editable provider amount/reference.
- Files and exports are treated as untrusted.
- Large result sets use cursor pagination or virtualized tables without hiding totals.

## Tests most agents will skip

### Ingestion

1. Same file uploaded with different filename; checksum duplicate detected.
2. CSV with BOM, CRLF, embedded newline, quoted delimiter, Unicode, empty final line, and duplicate headers.
3. CSV formula injection fields remain inert in exports and UI.
4. Gigantic field, zip bomb, slow upload, wrong content type, and malformed encoding are bounded.
5. Parser version changes; historical run remains tied to original normalized records/version.

### Matching

6. Same input and cutoff rerun produces byte-equivalent canonical result ordering.
7. Two external records compete for one internal record; one match and one duplicate/exception.
8. Internal record appears after initial run but before rerun; prior run stays unchanged.
9. Date-window boundary and timezone are deterministic.
10. Currency mismatch never auto-matches despite same numeric amount/reference.
11. Amount with leading zeros, decimal representation, or exponent confusion normalizes safely.
12. Fuzzy suggestion cannot be accepted without explicit analyst action/approval.

### Settlement and accounting

13. Two finance operators attempt close concurrently; one signed close and one idempotent/conflict response.
14. Close races with late reconciliation exception creation; close rechecks snapshot.
15. Settlement post commits but close response is lost; retry does not duplicate journal.
16. Reopen does not reverse settlement automatically; requires explicit adjustment policy.
17. Suspense resolution cannot exceed suspense item amount or wrong currency.
18. Partial provider settlement across multiple batches maintains receivable correctly.
19. Provider fee differs from quoted fee; explicit expense/exception, no balance hiding.
20. Restore before batch close and replay jobs; no duplicate settlement.
21. Closed report checksum/signature fails after tampering.

### Authorization

22. Support role can view customer transaction summary but not raw provider file or finance controls.
23. Finance user cannot approve own adjustment.
24. Search/export cannot leak other tenant’s merchant settlement rows.

## Observability and alerts

Metrics:

- files received/duplicates/parse errors;
- reconciliation duration and throughput;
- match rate by provider/version;
- exceptions by type/value/age;
- suspense amount/value age;
- settlement close lag;
- matcher version distribution;
- file-to-close end-to-end time.

Alerts:

- control-total mismatch;
- match-rate sudden drop;
- high-value missing external/internal;
- suspense age/value threshold;
- duplicate settlement attempt;
- closed-report integrity failure;
- settlement batch past expected date.

## Acceptance gate

A reviewer can generate a hostile provider settlement file, upload it, verify checksum and duplicate detection, run deterministic reconciliation, inspect mismatches, resolve an exception through approval, post settlement, close the batch, tamper with the report and see verification fail, then rerun without losing prior evidence.

## X content pillars

### Pillar A — “A payment is not finished when the API says success”

- Trace transaction to settlement.
- Show provider file and internal record.
- Explain why reconciliation closes the loop.

### Pillar B — “I ran the same reconciliation twice and compared the bytes”

- Explain deterministic inputs/cutoff/version.
- Demonstrate identical output.
- Then add a late internal record and compare a new run without mutating history.

### Pillar C — “Fuzzy matching should not move money automatically”

- Show exact versus candidate rules.
- Present analyst decision and approval.
- Explain false-match financial risk.

### Pillar D — “Suspense is a controlled debt, not a trash bin”

- Show suspense account and aging.
- Assign owner and resolution.
- Post approved correction.

### Short-form posts

- Provider control totals versus Atlas totals screenshot.
- “Five reconciliation breaks a transaction table cannot reveal.”
- Signed settlement close report and tamper demonstration.

## Do not waste time on

- machine-learning matching;
- PDF bank-statement OCR;
- many provider schemas;
- editable imported files;
- a single mutable `matched=true` flag;
- fuzzy auto-posting;
- dashboard match percentage without exception value and aging.
