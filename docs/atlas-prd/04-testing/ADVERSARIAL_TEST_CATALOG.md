# Atlas adversarial test catalogue

## Use

These are release-relevant scenarios, not aspirational examples. Each test must have an automated harness or a step-by-step controlled procedure, explicit expected invariants, and retained evidence. IDs remain stable.

## A. Ledger and accounting integrity

| ID | Adversarial scenario | Required proof |
|---|---|---|
| ADV-LED-001 | One posting removed from otherwise valid journal | Entire journal rejected; no balance/outbox/audit success effect |
| ADV-LED-002 | Duplicate posting line inserted | Template/aggregate constraints reject or totals/model reveal duplicate |
| ADV-LED-003 | Mixed currencies forced into one balancing group | Reject; no implicit FX equality |
| ADV-LED-004 | Maximum supported amount plus one minor unit | Safe domain error; no overflow/wrap |
| ADV-LED-005 | Negative/zero posting amount | Reject before persistence |
| ADV-LED-006 | Same business reference under concurrent requests | One economic journal; deterministic replay/conflict |
| ADV-LED-007 | App DB role issues `UPDATE`/`DELETE` on posted journal | Database permission denies and security telemetry records attempt |
| ADV-LED-008 | Direct insert bypassing posting service | Grants/constraints prevent a valid-looking unbalanced effect |
| ADV-LED-009 | Process dies after journal rows before commit | Rollback leaves no partial state |
| ADV-LED-010 | Commit succeeds, response connection dies | Same-key retry returns one journal and original durable result |
| ADV-LED-011 | Two account sets supplied in reverse order | Canonical lock ordering avoids application-created deadlock |
| ADV-LED-012 | Serializable transaction abort under contention | Bounded whole-command retry; no duplicate side effects |
| ADV-LED-013 | Concurrent hard-close and backdated posting | Exactly one coherent result; period evidence remains valid |
| ADV-LED-014 | Reversal requested twice concurrently | At most one authorized full reversal journal |
| ADV-LED-015 | Reversal payload changed after approval | Payload-hash mismatch blocks execution |
| ADV-LED-016 | Projection row corrupted in test harness | Independent rebuild detects exact variance and activates protection |
| ADV-LED-017 | Verification reads while postings continue | Defined consistent snapshot and reproducible cutoff |
| ADV-LED-018 | Closed-period correction attempted by ordinary template | Reject; current-period adjustment path required |
| ADV-LED-019 | Rounding remainder silently discarded | Property test fails; explicit rounding account required |
| ADV-LED-020 | Restore missing one journal/outbox page | Manifest/count/hash/reconciliation identifies discrepancy |

## B. Wallets, holds, and double-spend resistance

| ID | Scenario | Required proof |
|---|---|---|
| ADV-WAL-001 | Two withdrawals race for the same available balance | At most eligible amount is reserved; no negative balance unless policy permits |
| ADV-WAL-002 | Hold expires while capture races | One terminal disposition; capture cannot exceed remaining hold |
| ADV-WAL-003 | Partial capture repeated with same key | Same captured amount once; stable replay response |
| ADV-WAL-004 | Partial captures with different keys exceed original hold | Final command rejected atomically |
| ADV-WAL-005 | Release and capture arrive concurrently | Legal state transition wins once; no stranded/duplicated funds |
| ADV-WAL-006 | Wallet freezes after command validation before posting | Execution-time policy/state recheck behaves as specified |
| ADV-WAL-007 | Balance cache is stale during transfer | Authoritative transactional check prevents overspend |
| ADV-WAL-008 | Projection update omitted through mutation | Independent balance/model test fails |
| ADV-WAL-009 | Pending credit is incorrectly treated as available | Risk/available-balance invariant catches it |
| ADV-WAL-010 | Wallet closure races new hold | Closure or hold succeeds coherently; no closed wallet with active hidden obligation |

## C. Idempotency and request ambiguity

| ID | Scenario | Required proof |
|---|---|---|
| ADV-IDM-001 | Same key, byte-identical request, concurrent first use | One lease/command; all callers observe compatible result |
| ADV-IDM-002 | Same key, semantically same JSON with field reordering | Canonical hash policy produces documented equivalence |
| ADV-IDM-003 | Same key, different amount/beneficiary | `409`; original command unchanged |
| ADV-IDM-004 | Same key reused by another tenant/principal | Scope isolation; no cross-tenant replay or existence leak |
| ADV-IDM-005 | Handler crashes while idempotency record is `processing` | Lease recovery cannot execute effect twice |
| ADV-IDM-006 | Idempotency response is larger/new header set after upgrade | Stored allowlisted response remains contract-compatible |
| ADV-IDM-007 | Key expires while an external outcome is unresolved | Retention policy prevents unsafe new economic command |
| ADV-IDM-008 | Client retries 500 times with backoff bug | One effect; rate control prevents resource exhaustion |
| ADV-IDM-009 | Gateway retries POST without client awareness | Required key and server semantics protect command |
| ADV-IDM-010 | Timeout occurs before server receives request versus after commit | Client recovery path safely handles both indistinguishable cases |

## D. Transfer/provider lifecycle

| ID | Scenario | Required proof |
|---|---|---|
| ADV-TRF-001 | Provider accepts request then connection times out | Transfer enters `outcome_unknown`; no blind alternate submission |
| ADV-TRF-002 | Delayed provider status query confirms success | Reserved funds captured and ledger posted once |
| ADV-TRF-003 | Delayed callback confirms failure after ambiguous state | Hold released once; failure reason normalized |
| ADV-TRF-004 | Callback arrives before synchronous provider response | State machine accepts fact once and later response cannot regress it |
| ADV-TRF-005 | Success callback delivered five times | One provider event consumption and one financial completion |
| ADV-TRF-006 | Failure event arrives after success | No terminal regression; inconsistency becomes operations evidence |
| ADV-TRF-007 | Provider reuses reference for a different request | Uniqueness/correlation conflict escalates; no wrong transfer completion |
| ADV-TRF-008 | Provider response has valid HTTP but malformed amount/currency | Adapter rejects and marks ambiguous/error without unsafe posting |
| ADV-TRF-009 | Provider clock is skewed | Local ordering/security policy does not trust provider time blindly |
| ADV-TRF-010 | Worker dies after provider submission before attempt state update | Durable attempt/request fingerprint enables recovery without duplicate submission |
| ADV-TRF-011 | Worker dies after status success before journal posting | Reconciliation/resume posts once from durable state |
| ADV-TRF-012 | Cancellation races provider success | State machine/funds outcome coherent; operator sees actual rail result |
| ADV-TRF-013 | Beneficiary is changed after quote creation | Destination-bound quote fails verification |
| ADV-TRF-014 | Quote expires at exact confirmation boundary | Single defined clock comparison; no timezone ambiguity |
| ADV-TRF-015 | Provider unavailable for hours | Backlog/age alerts, bounded retries, customer status, operator path |

## E. Merchant payments, capture, refund, and webhooks

| ID | Scenario | Required proof |
|---|---|---|
| ADV-PAY-001 | Two captures race against one authorization | Total captured never exceeds authorized amount |
| ADV-PAY-002 | Automatic capture and manual capture race | State machine prevents duplicate capture |
| ADV-PAY-003 | Refunds concurrently exceed captured amount | At most refundable amount accepted/reserved |
| ADV-PAY-004 | Refund succeeds externally but local response is lost | Recovery posts/refers to one refund effect |
| ADV-PAY-005 | Merchant credential revoked during in-flight request | Defined point-of-authorization semantics and audit |
| ADV-PAY-006 | Old and new credential overlap during rotation | Both work only in documented window; old revocation immediate after cutoff |
| ADV-WHK-001 | Endpoint URL resolves to cloud metadata address after registration | Delivery blocked before connection |
| ADV-WHK-002 | Redirect points to internal/private IP | Redirect denied and endpoint health records failure |
| ADV-WHK-003 | IPv4-mapped IPv6/private-network encoding bypass attempt | Canonical address validation denies |
| ADV-WHK-004 | JSON whitespace/order changes after signature | Verification fails because raw transmitted bytes are covered |
| ADV-WHK-005 | Same event redelivered under new delivery ID | Merchant fixture processes business effect once by event ID |
| ADV-WHK-006 | Merchant processes then socket closes before Atlas sees response | Retry occurs; delivery history remains honest; merchant effect once |
| ADV-WHK-007 | Key rotates between scheduling and transmission | Deterministic key selection and valid key ID/signature |
| ADV-WHK-008 | Key revoked during retry schedule | Subsequent attempt follows revocation policy; audit identifies key version |
| ADV-WHK-009 | Recipient slow-drips response headers/body | Deadline and byte limits release worker resources |
| ADV-WHK-010 | Recipient returns malicious headers/newlines/huge body | Logs/UI safely encode, redact, and truncate |
| ADV-WHK-011 | Manual replay races automatic retry | Separate delivery IDs, bounded scheduling, no event mutation |
| ADV-WHK-012 | One merchant creates massive failing backlog | Per-tenant bulkhead protects other tenants |

## F. Settlement and reconciliation

| ID | Scenario | Required proof |
|---|---|---|
| ADV-REC-001 | Exact settlement file uploaded twice under different filename | Content digest/provider identity deduplicates import |
| ADV-REC-002 | Same provider reference appears twice with different amounts | Duplicate/mismatch exception; no arbitrary match |
| ADV-REC-003 | Internal transaction missing from provider file | `missing_provider` exception and aging workflow |
| ADV-REC-004 | Provider record has no internal transaction | `missing_internal` exception; no synthetic journal without approval |
| ADV-REC-005 | Currency differs while numeric amount matches | Currency mismatch, never false match |
| ADV-REC-006 | Fee/net relationship is inconsistent | Explicit exception using documented formula |
| ADV-REC-007 | CSV contains duplicate headers, NUL, invalid UTF-8, huge field | Import quarantined with bounded resources |
| ADV-REC-008 | CSV cell begins `=`, `+`, `-`, `@` | Export/import handling prevents spreadsheet formula execution |
| ADV-REC-009 | Reconciliation job crashes halfway | Resume/idempotent rerun yields same classifications once |
| ADV-REC-010 | Two runs attempt same batch/ruleset | One scope owner or deterministic duplicate response |
| ADV-REC-011 | Ruleset changes after initial run | Historical run remains reproducible; comparison is new run |
| ADV-REC-012 | Exception is resolved using stale source data | ETag/version and execution-time checks reject |
| ADV-REC-013 | Adjustment approval exists but amount/reference changed | Payload hash blocks execution |
| ADV-REC-014 | Batch close races new imported file | Scope cutoff/close policy produces coherent result |
| ADV-REC-015 | Suspense account is used as generic auto-fix | Policy/tests require reason, owner, aging, and approved clearing |

## G. Identity, authorization, tenant isolation, and workforce abuse

| ID | Scenario | Required proof |
|---|---|---|
| ADV-IAM-001 | Change object ID to another tenant’s resource | No data/count/timing leakage beyond policy; access denied server-side |
| ADV-IAM-002 | List endpoint applies tenant filter after pagination/count | Test catches count/cursor leakage |
| ADV-IAM-003 | User loses role while privileged page remains open | Next server action denied; UI updates safely |
| ADV-IAM-004 | Policy cache retains old permission after revocation | Invalidation/maximum staleness meets documented bound |
| ADV-IAM-005 | Maker gains checker role after creating request | Still cannot approve own request |
| ADV-IAM-006 | Checker approves, then target state/permissions change | Execution rechecks and may reject stale approval |
| ADV-IAM-007 | Invitation grants role above inviter’s delegation | Rejected before invite creation/acceptance |
| ADV-IAM-008 | Tenant switch occurs in another tab | Sessions/UI cannot submit command under unintended tenant |
| ADV-IAM-009 | Session fixation across login/step-up | Session identifier rotates and old session is invalid |
| ADV-IAM-010 | Recovery endpoint probes existing/non-existing users | Response/status/timing/rate behaviour resists enumeration |
| ADV-IAM-011 | Workforce search uses full email/phone without purpose | Permission/purpose/reveal workflow enforced and audited |
| ADV-IAM-012 | Operator exports data after access removed | Job execution/download re-authorizes, not only creation time |
| ADV-IAM-013 | Break-glass token used outside incident scope/expiry | Denied and paged/audited |
| ADV-IAM-014 | API client sends valid token for wrong audience | Rejected |
| ADV-IAM-015 | Algorithm/key confusion or stale signing key | Token verification fail-closed with controlled rotation |

## H. Browser and frontend security/correctness

| ID | Scenario | Required proof |
|---|---|---|
| ADV-WEB-001 | XSS attempts read session credentials | Tokens absent from JS-readable storage; CSP/encoding controls |
| ADV-WEB-002 | Cross-site form submits money command | CSRF control rejects before economic effect |
| ADV-WEB-003 | Browser back cache reveals data after logout/tenant switch | Cache policy/state purge prevents exposure |
| ADV-WEB-004 | Step-up expires after confirmation screen before submit | Server rejects; UI does not silently replay with new command |
| ADV-WEB-005 | Double-click/keyboard repeat on submit | One idempotency key per intended command and disabled/traceable state |
| ADV-WEB-006 | User refreshes during ambiguous outcome | Durable status view resumes; no “failed” lie or duplicate command |
| ADV-WEB-007 | Large minor-unit value exceeds JS safe integer | String/BigInt-safe formatting preserves exact value |
| ADV-WEB-008 | Locale uses comma/period/grouping variants | Display/parser cannot alter submitted minor units |
| ADV-WEB-009 | Status conveyed only through colour | Accessibility test fails; semantic text/icon relation required |
| ADV-WEB-010 | Modal closes on error and loses idempotency context | Recovery preserves intended command/status safely |
| ADV-WEB-011 | Stale approval/case ETag | Conflict UX forces re-read and explicit decision |
| ADV-WEB-012 | Malicious provider/operator text rendered | Contextual encoding prevents script/HTML injection |
| ADV-WEB-013 | CSV export opened in spreadsheet | Formula injection corpus remains inert |
| ADV-WEB-014 | Print/PDF statement splits totals/header incorrectly | Semantic and visual/print tests catch unusable evidence |

## I. Privacy, audit, reporting, and data rights

| ID | Scenario | Required proof |
|---|---|---|
| ADV-PRV-001 | Sensitive value included in error/log/trace/breadcrumb | Redaction test and sink scan fail build |
| ADV-PRV-002 | User requests export while account investigation restricts fields | Purpose/authorization/redaction policy applied and recorded |
| ADV-PRV-003 | Download URL copied after logout/expiry | Short-lived bound grant denied |
| ADV-PRV-004 | Deletion job removes ledger/audit referential evidence | Retention/pseudonymization controls prevent integrity damage |
| ADV-PRV-005 | Retention job crashes midway | Restartable manifest and exact item disposition |
| ADV-PRV-006 | Backup still contains expired/deleted data | Backup expiry/tombstone strategy evidenced, limitation explicit |
| ADV-AUD-001 | Audit event edited/deleted in test corruption harness | Daily manifest/signature verification detects change/missing/reorder within model |
| ADV-AUD-002 | Audit pipeline unavailable during high-risk action | Fail-closed/degraded policy behaves per action class |
| ADV-AUD-003 | Operator searches own actions to conceal activity | Read/search itself audited and cannot alter source evidence |
| ADV-RPT-001 | Opening balance at boundary uses wrong timezone/inclusivity | Independent statement oracle catches mismatch |
| ADV-RPT-002 | Reversal/refund collapsed into original line | Statement contract test requires separate linked entry |
| ADV-RPT-003 | PDF, CSV, and JSON differ by one minor unit | Cross-format differential test fails |

## J. Resilience, deployment, and recovery

| ID | Scenario | Required proof |
|---|---|---|
| ADV-REL-001 | API dies after DB commit before outbox publication | Outbox worker later publishes once-or-more; consumers effect once |
| ADV-REL-002 | Publisher sends event then dies before marking outbox sent | Duplicate delivery handled; observability explains replay |
| ADV-REL-003 | Broker unavailable during command | Financial state/outbox remains durable; backlog age alerts |
| ADV-REL-004 | Redis unavailable | Financial correctness remains; explicit degraded rate/cache behaviour |
| ADV-REL-005 | Object storage unavailable during report generation | Job resumable; no ready status without complete digest/object |
| ADV-REL-006 | Identity provider unavailable | No new high-risk bypass; existing-session degraded policy explicit |
| ADV-REL-007 | Database connection pool exhaustion | Backpressure and timeouts avoid cascading goroutine/resource leak |
| ADV-REL-008 | Bad migration holds table lock | Migration safety/timeout/abort; service recovery without corruption |
| ADV-REL-009 | Deployment mixes old/new event/API versions | Compatibility suite and rolling-upgrade scenario pass |
| ADV-REL-010 | Clock jumps/skews across nodes | Expiry/signature/order logic uses documented tolerance and durable sequence |
| ADV-DR-001 | Point-in-time restore before provider success callback | Replay/reconciliation reaches one correct terminal result |
| ADV-DR-002 | Restore includes journals but not later object files | Integrity inventory identifies missing artifacts and blocks “complete” claim |
| ADV-DR-003 | Restore produces duplicate consumer checkpoints | Inbox/event replay remains idempotent |
| ADV-DR-004 | Encryption key version is unavailable in restore | Recovery runbook identifies dependency; no silent data loss |
| ADV-DR-005 | Primary returns after failover with divergent writes | Design prevents dual-primary or has explicit conflict/stop policy |

## K. Resource exhaustion and abuse

| ID | Scenario | Required proof |
|---|---|---|
| ADV-RES-001 | Deeply nested/huge JSON | Body/depth/field limits reject without memory spike |
| ADV-RES-002 | Unbounded search wildcard/export | Query limits, async export, quotas, and cancellation |
| ADV-RES-003 | High-cardinality telemetry values from user input | Attribute allowlist prevents telemetry backend exhaustion |
| ADV-RES-004 | Tenant generates millions of tiny idempotency keys | Quota/retention/partition controls preserve system |
| ADV-RES-005 | Expensive risk rule expression | Static validation and runtime budget prevent worker starvation |
| ADV-RES-006 | Zip/compression bomb in attachment/import | Size-after-decompression and content limits quarantine |
| ADV-RES-007 | Regex denial of service in filters/validation | Safe regex or bounded patterns and corpus test |
| ADV-RES-008 | Infinite provider retry caused by classification bug | Max attempts/age and operator terminal path |
| ADV-RES-009 | Malicious cursor creates expensive query or tenant change | Integrity-protected cursor and query budget reject |
| ADV-RES-010 | Slow client upload/download consumes all workers | Server deadlines, streaming limits, connection quotas |

## Evidence template per test

```text
Test ID:
Source revision:
Environment and configuration digest:
Synthetic fixture/scenario seed:
Injected fault or attack:
Expected business invariant:
Expected security/authorization result:
Expected ledger/hold/outbox/audit effects:
Observed result:
Trace/dashboard/log evidence (redacted):
Database/manifest verification:
Residual limitation:
Owner and execution date:
```
