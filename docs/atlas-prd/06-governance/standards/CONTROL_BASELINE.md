# Atlas standards and control baseline

## Status and use

This file pins the reference baseline used to design and verify Atlas as of July 20, 2026. It is a control-alignment aid for a synthetic portfolio project. It does not assert legal advice, regulatory authorization, PCI compliance, certification, or that every control in a source standard applies.

Before any real deployment, qualified security, privacy, financial, and legal owners must determine applicability, update versions, collect formal evidence, and approve residual risks.

## Pinned baseline

| Domain | Reference | Atlas use |
|---|---|---|
| Application security | OWASP ASVS 5.0.0 | Primary application security requirements and verification mapping |
| Web risk awareness | OWASP Top 10:2025 | Threat-review cross-check, not the complete control set |
| API security | OWASP API Security Top 10:2023 | API abuse and authorization test catalogue |
| Secure development | NIST SP 800-218 SSDF 1.1 | Secure software lifecycle, provenance, vulnerability response |
| Digital identity | NIST SP 800-63-4 suite | Identity proofing, authentication, federation, assurance vocabulary |
| OAuth security | RFC 9700 | OAuth 2.0 security best current practice |
| High-value API security | OpenID FAPI 2.0 Security Profile (Final) | Design reference for stronger financial API authorization profiles where applicable |
| Payment card data | PCI DSS 4.0.1 | Scope-awareness and explicit no-cardholder-data design; Atlas does not claim compliance |
| Incident response | NIST SP 800-61 Rev. 3 | Incident preparation, response, recovery, lessons learned |
| Cybersecurity risk | NIST Cybersecurity Framework 2.0 | Governance and risk-management organization |
| Key management | NIST SP 800-57 Part 1 Rev. 5 | Key lifecycle and cryptoperiod principles |
| Secure-by-design | CISA Secure by Design | Safe defaults, customer burden reduction, product-level security ownership |
| API description | OpenAPI 3.1.1 | Pinned contract version for tooling compatibility; review newer versions separately |
| Event description | AsyncAPI 3.0.0 | Event channel/message contract |
| API errors | RFC 9457 | Problem Details response shape |
| HTTP signatures | RFC 9421 concepts | Webhook signature profile input; Atlas documents its exact profile |
| Trace propagation | W3C Trace Context | `traceparent`/`tracestate` interoperability |
| Observability | OpenTelemetry specification and semantic conventions | Traces, metrics, logs, bounded semantic attributes |
| Database transactions | Current supported PostgreSQL documentation | Isolation, serializable retry, locking, constraints, backup/restore |
| Go robustness | Official Go fuzzing/security guidance | Fuzzing and corpus-based parser/domain verification |
| Build provenance | SLSA framework | Build integrity/provenance maturity reference |
| SBOM | CycloneDX specification | Machine-readable software component inventory |
| Nigerian privacy | Nigeria Data Protection Act 2023 and NDPC General Application and Implementation Directive 2025 | Data rights, lawful processing, security, DPIA, automated decision, transfer, breach workflow design |
| Nigerian payments context | Central Bank of Nigeria Payments System resources, open-banking guidelines, consumer-protection framework | Domain/context reference only; no claim Atlas is licensed or approved |

## Official source registry

- OWASP ASVS: https://owasp.org/www-project-application-security-verification-standard/
- OWASP Top 10 2025: https://owasp.org/Top10/2025/
- OWASP API Security Top 10 2023: https://owasp.org/API-Security/editions/2023/en/0x11-t10/
- NIST SSDF: https://csrc.nist.gov/pubs/sp/800/218/final
- NIST Digital Identity Guidelines: https://csrc.nist.gov/pubs/sp/800/63/4/final
- RFC 9700: https://www.rfc-editor.org/info/rfc9700/
- OpenID FAPI 2.0 Security Profile: https://openid.net/specs/fapi-security-profile-2_0-final.html
- PCI DSS: https://www.pcisecuritystandards.org/standards/pci-dss/
- NIST incident response: https://csrc.nist.gov/pubs/sp/800/61/r3/final
- NIST CSF 2.0: https://www.nist.gov/cyberframework
- NIST key management: https://csrc.nist.gov/pubs/sp/800/57/pt1/r5/final
- CISA Secure by Design: https://www.cisa.gov/securebydesign
- OpenAPI 3.1.1: https://spec.openapis.org/oas/v3.1.1.html
- AsyncAPI 3.0.0: https://www.asyncapi.com/docs/reference/specification/v3.0.0
- RFC 9457: https://www.rfc-editor.org/info/rfc9457/
- RFC 9421: https://www.rfc-editor.org/info/rfc9421/
- W3C Trace Context: https://www.w3.org/TR/trace-context/
- OpenTelemetry specification: https://opentelemetry.io/docs/specs/otel/
- PostgreSQL transaction isolation: https://www.postgresql.org/docs/current/transaction-iso.html
- Go fuzzing: https://go.dev/doc/security/fuzz/
- SLSA: https://slsa.dev/
- CycloneDX: https://cyclonedx.org/specification/overview/
- Nigeria Data Protection Act 2023: https://ndpc.gov.ng/wp-content/uploads/2024/03/Nigeria_Data_Protection_Act_2023.pdf
- NDPC GAID 2025: https://ndpc.gov.ng/wp-content/uploads/2025/07/NDP-ACT-GAID-2025-MARCH-20TH.pdf
- CBN Payments System: https://www.cbn.gov.ng/PaymentsSystem/

## Atlas control families

### GOV — Governance and assurance

- `GOV-01` Named owner for product, ledger, security, privacy, reliability, and evidence.
- `GOV-02` Stable requirements, threat, risk, and decision registers.
- `GOV-03` Security/privacy architecture review for every material phase/change.
- `GOV-04` Claims-to-evidence discipline and no unsupported compliance/scale claims.
- `GOV-05` Exception process with owner, expiry, compensating control, and residual risk.

### SDLC — Secure development and supply chain

- `SDLC-01` Protected changes and code-owner review for critical paths.
- `SDLC-02` Pinned dependencies, SBOM, vulnerability and secret scanning.
- `SDLC-03` Signed artifacts/provenance and digest promotion.
- `SDLC-04` Threat-led tests, fuzzing, mutation, contract compatibility.
- `SDLC-05` Vulnerability disclosure, triage, emergency update, and regression process.

### IAM — Identity, session, authorization, and privileged access

- `IAM-01` Separate customer, merchant, workforce, and machine identity populations.
- `IAM-02` Secure BFF session, CSRF, rotation, revocation, timeout.
- `IAM-03` Step-up for high-risk action and phishing-resistant workforce authentication.
- `IAM-04` Deny-by-default tenant/object/action/field authorization.
- `IAM-05` Maker-checker, payload binding, execution-time recheck.
- `IAM-06` Time-bound, alerted, reviewed break-glass.

### FIN — Financial integrity

- `FIN-01` Integer minor units and explicit currency/rate precision.
- `FIN-02` Controlled double-entry templates and per-currency balance.
- `FIN-03` Immutable posted records and compensating corrections.
- `FIN-04` Atomic journal/projection/outbox boundary.
- `FIN-05` Concurrency-safe holds and available-balance control.
- `FIN-06` Idempotency and duplicate business/provider reference controls.
- `FIN-07` Independent balance/trial-balance/statement/reconciliation verification.
- `FIN-08` Period close, suspense governance, and approved adjustments.

### APP/API — Application and integration security

- `APP-01` Contract-first APIs with input/output/error limits and compatibility.
- `APP-02` Object/property/function authorization and mass-assignment resistance.
- `APP-03` Resource consumption and sensitive business-flow controls.
- `APP-04` Third-party/provider response validation and normalization.
- `APP-05` Signed, replay-resistant callbacks/webhooks.
- `APP-06` SSRF-safe outbound networking and constrained egress.
- `APP-07` Browser CSP, CSRF, cache, output encoding, and no JS-readable session token.

### DAT/PRV — Data protection and privacy

- `DAT-01` Data inventory, classification, minimisation, and purpose.
- `DAT-02` Encryption, masking, purpose-bound reveal, and logging exclusion.
- `DAT-03` Versioned notice/consent/lawful-basis evidence.
- `DAT-04` Access, correction, portability, closure, retention, pseudonymization workflows.
- `DAT-05` DPIA/risk assessment for high-risk/automated processing.
- `DAT-06` Processor/transfer/breach records and response design.

### OPS/REL — Operations, observability, resilience, and incident response

- `OPS-01` Structured audit, traces, metrics, logs, and bounded attributes.
- `OPS-02` Actionable alerts and runbooks for financial/security states.
- `OPS-03` Operator tooling uses commands, authorization, reason, and audit—not direct edits.
- `REL-01` Timeouts, backpressure, bulkheads, retries, dead-letter/operator paths.
- `REL-02` Encrypted backup, PITR, object inventory, and isolated restore rehearsal.
- `REL-03` Post-restore reconciliation before financial writes reopen.
- `IR-01` Incident lifecycle, evidence preservation, containment, recovery, and postmortem regression.

## Mapping method

`REQUIREMENTS_TRACEABILITY.csv` maps product requirement IDs to control IDs, threats, tests, and evidence. Mapping is many-to-many and should be reviewed, not automatically generated from keywords.

## Baseline review cadence

Review:

- before each public release;
- after a material architecture or scope change;
- after a critical finding or incident exercise;
- when a pinned standard is superseded;
- at least quarterly while the project is active.

Record changes through an ADR or governance change log and re-run affected controls/tests.
