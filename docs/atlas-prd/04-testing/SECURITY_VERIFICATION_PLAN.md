# Security verification plan

## Purpose and posture

Atlas treats security as a system property spanning product design, identity, authorization, financial controls, data handling, software supply chain, operations, and recovery. This plan maps a risk-led verification programme to the project’s control baseline. It is not a compliance certification or penetration-test replacement.

## Verification methods

- architecture and threat-model review;
- source review of critical paths;
- automated unit/property/mutation/fuzz tests;
- database permission and constraint tests;
- API/browser dynamic testing;
- manual authorization matrix review;
- dependency/container/IaC/secret analysis;
- egress/SSRF and hostile-server tests;
- incident and recovery exercises;
- evidence and control traceability review.

## Security release gates

A release candidate cannot pass with:

- known exploitable cross-tenant/object authorization failure;
- ability to duplicate or alter an economic effect through retry/race;
- bypass of ledger immutability/balance invariants;
- plaintext secret/token/identity document leakage;
- unaudited privileged money/data action;
- high-risk action that bypasses step-up/maker-checker policy;
- unbounded SSRF path to internal/private networks;
- critical dependency vulnerability without documented non-exploitability/mitigation and owner;
- untested restore/key dependency for critical encrypted data;
- missing regression test for a fixed critical finding.

## 1. Architecture and threat model verification

For every phase and material design change:

- update context/data-flow/trust-boundary diagrams;
- identify assets, actors, entry points, data stores, external dependencies, and administrative paths;
- perform STRIDE-like threat discovery plus financial abuse cases;
- record threats in `THREAT_REGISTER.csv` with control and verification IDs;
- model insider/workforce misuse, compromised merchant, malicious customer, provider compromise, dependency compromise, and operator error;
- review failure/recovery paths as attack surfaces;
- document residual risk and consciously rejected complexity.

Threat-model review asks “how can money/data/control be abused?” rather than only “which OWASP category applies?”

## 2. Identity and session verification

Test:

- issuer, audience, signature algorithm/key, expiry/not-before, nonce/state, PKCE, and redirect allowlist;
- session fixation and rotation at login, step-up, privilege/tenant change;
- idle/absolute timeouts and concurrent-session revocation;
- secure/HttpOnly/SameSite cookie attributes and narrow paths/domains;
- CSRF on every cookie-authenticated mutation including uncommon content types;
- account/recovery enumeration through status, message, timing, rate, and side channel;
- phishing-resistant/step-up policy for configured high-risk actions;
- recovery and contact-change delay/risk interactions;
- machine credential scope/audience, rotation overlap, revocation, and storage;
- no browser access token in local/session storage or JS-visible cookie.

Manual browser inspection and proxy-based tests supplement automation.

## 3. Authorization and tenancy verification

Build an executable matrix across:

- customer, merchant roles, risk analyst, support, finance, security admin, auditor, break-glass, service identities;
- tenant A, tenant B, global resources;
- object ownership, action, field, lifecycle state, session assurance, case/approval purpose;
- list/search/count/export and nested resources;
- API and asynchronous worker execution.

Required techniques:

- swap every object ID across tenants/owners;
- modify hidden/read-only fields and mass-assignment candidates;
- call administrative functions directly;
- verify authorization before pagination/count/autocomplete;
- revoke role/session/restriction while action is in progress;
- test policy-cache invalidation;
- inspect generated SQL/repository contracts for tenant scope;
- verify workers re-establish authority from durable command context rather than trusting event payload.

## 4. Financial integrity verification

Security review covers:

- integer money and overflow bounds;
- controlled posting templates;
- per-currency debit-credit equality;
- immutable tables and database grants;
- independent projection/trial-balance verification;
- idempotency canonicalization and races;
- hold/capture/release concurrency;
- state-machine terminality and no regression;
- maker-checker payload binding and execution-time rechecks;
- provider ambiguity and duplicate callbacks;
- reconciliation exception resolution and adjustment controls;
- restore/replay with one economic effect.

This is tested as both correctness and fraud-resistance.

## 5. API security verification

Test the applicable OWASP API risk classes:

- object-level authorization;
- broken authentication;
- object-property authorization/mass assignment;
- resource consumption;
- function-level authorization;
- sensitive business-flow abuse;
- SSRF;
- configuration/inventory/version exposure;
- unsafe consumption of third-party APIs.

Additional Atlas checks:

- duplicate JSON keys and parser differentials;
- content-type confusion and method override;
- header/request smuggling at deployed edge where feasible;
- ambiguous path normalization and encoded separators;
- cursor/filter tampering;
- error disclosure and provider normalization;
- idempotency and ETag/precondition contracts;
- safe timeout/retry behaviour;
- OpenAPI conformance and undocumented endpoint discovery.

## 6. Frontend and browser verification

- contextual output encoding for all server/provider/operator content;
- strict CSP and reporting, no unsafe inline/eval without reviewed exception;
- frame-ancestors/clickjacking policy;
- cache control on authenticated/sensitive pages;
- CSRF and same-origin controls;
- postMessage target/source origin validation;
- URL/open redirect/navigation allowlists;
- file upload/download and object URL safety;
- DOM clobbering/prototype pollution exposure in dependencies;
- sensitive values absent from client telemetry, browser storage, source maps, and error UI;
- logout/tenant switch clears in-memory cached data;
- UI cannot create deceptive double-submit or hide ambiguous outcomes;
- accessibility remains intact under security/restriction states.

## 7. Cryptography, key, and secret verification

- Use mature standard libraries and managed-key abstraction; no custom primitives.
- Inventory key purposes: session/token verification, field encryption, webhook signing, audit manifest signing, artifact signing.
- Verify key IDs/versions, rotation, overlap, revocation, envelope metadata, and restore availability.
- CSPRNG for secrets/IDs/nonces; deterministic randomness only in explicit tests.
- Constant-time comparison for signatures/tokens where applicable.
- Prevent algorithm/key confusion and downgrade.
- Secrets are absent from repository, images, CI logs, telemetry, crash dumps, and support exports.
- Break-glass access to secret infrastructure is narrow, time-bound, alerted, and audited.

## 8. Data protection and privacy verification

- Data inventory and classification match actual schema/events/logs.
- Collection fields have purpose and minimisation rationale.
- Sensitive fields encrypted/masked and separated from broad queries.
- Access/reveal/export is permissioned, purpose-bound, and audited.
- Logs/traces/metrics are scanned with canary sensitive values.
- Retention/pseudonymization jobs are restartable, evidenced, and do not break financial history.
- Data access/portability packages are scoped, protected, expiring, and consistent.
- Automated risk decisions preserve factors/version/review path without exposing exploitable rules.
- Cross-border/processor dependencies are documented; portfolio uses synthetic data.

## 9. Webhook, outbound request, and SSRF verification

- URL parsing/canonicalization corpus across IPv4/IPv6/Unicode/alternate forms.
- DNS rebinding between validation/connect and across redirects.
- Private/link-local/metadata/internal network denial.
- TLS hostname verification and certificate failure.
- Egress proxy/network ACL enforcement independent of application validation.
- Request deadline, response byte, redirect, concurrency, and port limits.
- Raw-body content digest/signature, replay window, durable event dedupe, rotation.
- Logs/UI safely encode hostile response metadata/body.

## 10. File, report, and parser verification

For KYC synthetic evidence, settlement files, attachments, and reports:

- declared/actual content type and magic bytes;
- size, decompressed size, row/field/depth limits;
- malformed Unicode, NUL, duplicate headers/keys, delimiter ambiguity;
- archive traversal/symlink/bomb if archives supported;
- CSV formula injection and spreadsheet-safe output;
- PDF/HTML active content and filename/header injection;
- antivirus/sandbox only as defense-in-depth, not sole validation;
- quarantine, immutable digest, parser version, and deterministic retry.

## 11. Software supply chain verification

CI/release controls:

- pinned dependencies and lockfiles;
- dependency review and vulnerability scanning;
- secret scanning and protected branches;
- SAST plus targeted custom rules for money/time/tenant/import boundaries;
- SBOM for frontend, backend, and images;
- minimal non-root images, read-only filesystem where viable, dropped capabilities;
- IaC and Kubernetes/container configuration scan where used;
- signed artifacts and provenance;
- build from clean source in controlled runner;
- base-image and toolchain update policy;
- artifact digest promoted between environments, not rebuilt;
- dependency compromise/emergency update runbook.

## 12. Infrastructure and deployment verification

- network segmentation and default-deny egress for workloads needing it;
- database roles, TLS, backups, PITR, audit, timeout, and no public exposure;
- secret injection without image/source persistence;
- least-privilege service identities and environment separation;
- safe health endpoints without sensitive topology;
- secure headers and edge body/time limits;
- admin/observability interfaces not public;
- environment banners and synthetic data labels;
- production-like configuration drift detection;
- migration/deployment rollback and forward-fix exercises.

## 13. Logging, monitoring, and incident verification

- audit event completeness for privileged/high-risk actions and denials;
- structured logging with injection-safe encoding and redaction;
- request/correlation/trace propagation without trusting them for auth;
- alerts for financial variance, immutable-table write attempt, suspicious admin use, key failure, signature failures, provider ambiguity age, webhook SSRF blocks, credential anomalies;
- alert routing and runbook links;
- evidence preservation and clock/source revision context;
- incident exercise includes containment without deleting financial history;
- postmortem actions map to tests/controls.

## 14. Manual source-review targets

Mandatory focused review for:

- ledger transaction and posting templates;
- balance/hold concurrency;
- idempotency repository and canonicalization;
- authorization middleware/policy and tenant repositories;
- approval execution;
- provider callback signature/dedupe/state machine;
- webhook destination validation and signing;
- reconciliation matching/adjustments;
- report/download authorization;
- encryption/key wrappers;
- migrations affecting financial/security fields;
- CI/release permissions.

## Finding management

Each finding records:

- ID, title, severity, likelihood, exploit prerequisites;
- affected asset/tenant/financial impact;
- exact reproduction and evidence;
- root control failure, not only symptom;
- remediation and regression test;
- owner/due date/status;
- residual risk and approval where accepted;
- disclosure-safe public summary for portfolio evidence when appropriate.

Severity accounts for financial duplication/loss, cross-tenant exposure, privileged abuse, recoverability, detection delay, and blast radius—not CVSS alone.

## Final security evidence pack

- threat model and register;
- standards/control mapping;
- authorization matrix results;
- adversarial test report;
- fuzz/mutation summary;
- dependency/SBOM/provenance evidence;
- database role/immutability proof;
- key rotation and restore exercise;
- browser/API dynamic assessment;
- incident/game-day report;
- findings with regression links and explicit residual limitations.
