# Privacy and regulatory alignment

## 1. Positioning

Atlas is a portfolio system and does not claim legal compliance. The design nevertheless maps product decisions to current Nigerian data-protection obligations and broadly recognized security and payment guidance so the implementation demonstrates compliance-aware engineering.

Any real deployment would require licensed financial partners, legal and compliance ownership, regulator-specific interpretation, independent security assessment, and jurisdiction-specific operating procedures.

## 2. Nigeria-focused privacy baseline

The product documentation must maintain:

- record of processing activities;
- data inventory and classification;
- documented purpose and lawful-basis placeholder per processing activity;
- privacy notice versioning and evidence of presentation/consent where relied upon;
- data privacy impact assessment for high-risk processing;
- data-subject request workflow;
- automated-decision explanation and human-review path;
- data breach assessment and notification runbook;
- cross-border transfer inventory and safeguards placeholder;
- retention and deletion schedule;
- processor/vendor register.

The implementation uses only synthetic personal data, but the controls are built as if sensitive data handling must be justified and audited.

## 3. Financial-services regulatory posture

Atlas may model concepts found in Nigerian payment, open-banking, consumer-protection, KYC, and cybersecurity guidance, but it must not imply CBN approval or operating authorization.

Design implications:

- clear transaction status and fees;
- complaint and dispute workflow;
- customer access to statements and transaction history;
- tiered capabilities and transaction limits;
- consent and third-party access boundaries;
- secure APIs, audit, and incident readiness;
- separation of customer funds, platform funds, revenue, expenses, clearing, and suspense in the ledger;
- operational evidence and reconciliation.

## 4. PCI posture

Atlas intentionally keeps cardholder data out of scope.

- no PAN, CVV, PIN, track, or real card token storage;
- no card entry form that could be confused for a production card flow;
- provider simulator uses opaque synthetic payment methods;
- architecture documents how hosted/tokenized real integrations would isolate payment-account data;
- payment-data security guidance is used as a control reference, not a certification claim.

## 5. Privacy by design requirements

### Purpose limitation

Each field belongs to a named processing purpose. “Might be useful later” is not a valid purpose.

### Data minimisation

- Store KYC provider outcome and references, not unnecessary document images.
- Use last-four or masked values for UI and operations.
- Keep high-risk evidence in restricted object storage rather than copying into notes.
- Events carry identifiers and minimum facts, not entire customer objects.

### Storage limitation

Retention categories:

- active account data;
- financial and audit records;
- KYC evidence metadata;
- risk cases;
- raw provider payloads;
- webhook attempts;
- logs and traces;
- generated exports;
- deleted-account linkage tokens.

Every category has trigger, duration placeholder, legal rationale placeholder, archive, deletion/anonymisation method, and test.

### Accuracy and correction

Personal profile correction must not alter historical financial facts. Historical records retain the identifier/reference valid at the time while current display may use corrected profile data under an explicit rendering rule.

### Confidentiality and integrity

Use least privilege, encryption, masked displays, secure exports, audit, backup, integrity checks, and breach detection.

### Accountability

Link requirements, controls, tests, incidents, approvals, and evidence in the traceability matrix.

## 6. Data-subject rights workflows

### Access / portability

- authenticated request with step-up;
- identity and scope validation;
- asynchronous export generation;
- machine-readable and human-readable formats;
- redaction of other persons and internal security data;
- short-lived download;
- access audit;
- expiry and secure deletion of export.

### Correction

- identify authoritative field owner;
- review impact on identity verification and restrictions;
- preserve historical financial records;
- audit before and after references.

### Deletion / account closure

- block new activity;
- settle or release holds;
- resolve pending transfers, disputes, and negative positions;
- preserve required financial/audit records;
- unlink or pseudonymise removable personal data;
- revoke sessions and credentials;
- document retained categories and reason.

### Objection / automated decisions

- explain risk factors at an appropriate level;
- allow human review for reviewable or materially adverse synthetic decisions;
- preserve the policy version and evidence used;
- ensure the reviewer can override only through an audited reasoned workflow.

## 7. Privacy threat cases

- support agent browsing unrelated customers;
- broad exports used for data exfiltration;
- PII copied into free-text case notes;
- logs containing request bodies;
- analytics joining identities unnecessarily;
- tenant data leakage through autocomplete or filters;
- generated statement URL shared or cached;
- deletion job removing legal ledger linkage;
- event payload retaining deleted personal data;
- cross-environment use of production-like data.

Each requires preventive, detective, and recovery controls plus an adversarial test.

## 8. Evidence disclaimer for public portfolio

Public materials must say:

> Atlas is a synthetic portfolio environment. Its controls are mapped to recognized standards and Nigerian regulatory themes for educational demonstration. It is not licensed, certified, audited, or suitable for real funds or real identity data.
