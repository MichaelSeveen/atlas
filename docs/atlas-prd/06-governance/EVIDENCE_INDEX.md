# Atlas evidence index

## Purpose

This index is populated as implementation proceeds. It keeps the PRD honest by connecting requirements, threats, tests, claims, source revisions, and reproducible artifacts.

## Storage convention

```text
evidence/
  <phase>/
    architecture/
    contracts/
    tests/
    security/
    performance/
    operations/
    recovery/
    content/
```

Every artifact has a sidecar metadata file or embedded header containing:

- evidence ID;
- source revision and build/image digest;
- environment/configuration digest;
- creation timestamp and actor/tool;
- requirement/threat/test/claim IDs;
- synthetic fixture/scenario seed;
- command/procedure to reproduce;
- expected and observed result;
- sanitization/redaction statement;
- integrity digest/signature where appropriate;
- limitation and expiry/revalidation date.

## Evidence catalogue template

| Evidence ID | Phase | Type | Supports | Artifact | Revision | Result | Reproduce | Last verified | Limitation |
|---|---|---|---|---|---|---|---|---|---|
| EVD-0001 | Phase 03 | ledger verification | LED-012; CLM-001; THR-001 | `evidence/...` | commit | pass | command/runbook | date | ... |

## Minimum phase evidence

- accepted architecture/security review;
- requirement and threat updates;
- OpenAPI/AsyncAPI/schema examples where applicable;
- critical invariant and adversarial test report;
- one failure-injection trace;
- database/ledger/audit facts;
- dashboard/alert/runbook proof;
- acceptance demo recording or reproducible script;
- known limitations;
- sanitized X content artifact.

## Integrity rules

- Do not overwrite historical evidence; create a new version.
- A screenshot without source revision/scenario is supporting context, not primary proof.
- CI artifacts with limited retention must be copied to durable versioned storage or regenerated before public claims.
- Public evidence is sanitized derivative of internal test evidence; preserve the linkage and digest.
- Failed tests/findings are retained when they materially explain a design correction.
- Evidence expires when implementation, dependency, standard baseline, or environment materially changes.
