# Phase 13 — Portfolio evidence and public release

## Outcome

Package Atlas so a serious reviewer can understand the product, architecture, security posture, accounting model, hard failures, tests, and measured evidence in under thirty minutes, then dive deeply for hours.

## Why this phase matters

A complex project that cannot be evaluated is weaker than a smaller one with disciplined evidence. The release must make clear what is implemented, what is simulated, what is designed but deferred, and which claims are proven.

## Dependencies

Phase 12.

## Public release components

### Repository front page

- one-sentence positioning statement;
- synthetic/no-real-money disclaimer;
- architecture diagram;
- five standout capabilities;
- quick-start with reproducible commands;
- demo accounts/scenarios;
- evidence index;
- security reporting guidance;
- limitations and non-goals;
- license.

### Reviewer paths

#### Ten-minute path

1. Watch a five-minute narrated failure demo.
2. Inspect system diagram.
3. Open transaction inspector screenshot.
4. Review ledger invariant tests.
5. Read reconciliation close report.

#### Thirty-minute path

1. Run local stack or hosted synthetic demo.
2. Execute duplicate internal transfer.
3. Execute timeout-after-provider-acceptance payout.
4. Inspect journal, hold, trace, callback, and reconciliation.
5. Review threat model and ADRs.

#### Deep technical path

- ledger design;
- API contracts;
- adversarial test catalog;
- security control mapping;
- restore/game-day reports;
- benchmark methodology;
- privacy/retention design;
- content notebook.

## Flagship demo script

### Scenario: merchant payment through failure and reconciliation

1. Merchant creates payment intent with idempotency key.
2. Customer pays from Atlas wallet or synthetic provider.
3. Risk policy allows or sends to review.
4. Funds reserve.
5. Provider simulator accepts but times out.
6. API shows pending provider confirmation and prevents duplicate resubmission.
7. Worker recovers by provider query.
8. Hold captures and balanced journal posts exactly once.
9. Merchant webhook endpoint temporarily fails and later receives a signed retry.
10. Merchant retrieves event independently through API.
11. Provider settlement file contains a fee mismatch.
12. Reconciliation creates exception and suspense item.
13. Finance resolves through maker-checker and closes batch.
14. Customer/merchant statement reflects original transaction and adjustment correctly.
15. Reviewer inspects one distributed trace and audit timeline.

The demo must include at least one failure. A happy-path-only demo does not represent Atlas.

## Evidence requirements

- `POR-001` Every headline claim maps to a stable evidence artifact.
- `POR-002` Hosted demo resets synthetic data safely and visibly.
- `POR-003` Demo identities have least privilege and no reusable production secrets.
- `POR-004` Architecture docs match deployed repository revision.
- `POR-005` OpenAPI and AsyncAPI render publicly and validate.
- `POR-006` Tests list command, environment, duration, and expected evidence.
- `POR-007` Benchmarks include hardware, dataset, workload, commit, and invariant result.
- `POR-008` Security evidence is sanitized without becoming vague.
- `POR-009` Known limitations and unimplemented items are explicit.
- `POR-010` Content posts link back to durable docs, not the reverse.

## Portfolio pages

### Project overview

- problem and thesis;
- user groups;
- product surfaces;
- architecture and boundaries;
- core decisions;
- screenshots that show workflows, not decorative dashboards;
- results/evidence;
- reflection.

### Financial correctness

- chart of accounts;
- posting examples;
- invariants;
- concurrency and response-loss demo;
- projection verification.

### Distributed failure

- provider simulator;
- ambiguous outcomes;
- callback ordering;
- outbox/inbox;
- restore replay.

### Operations and reconciliation

- transaction inspector;
- role matrix;
- maker-checker;
- file import/matching;
- suspense and close.

### Security and privacy

- trust model;
- identity separation;
- authorization test matrix;
- webhook SSRF;
- data classification/retention;
- audit integrity;
- limitations.

### Testing and performance

- test pyramid is insufficient; show invariant, model, fuzz, concurrency, contract, chaos, and security layers;
- benchmark methodology and bottleneck;
- game-day findings.

## Public claim ledger

Maintain a table:

| Claim | Evidence | Scope | Last verified | Limitation |
|---|---|---|---|---|
| Every posted journal balances per currency | property + DB integration tests | tested commit | date | assumes controlled posting role |
| Duplicate retry creates one transfer | multi-replica integration test | internal transfers | date | idempotency retention documented |
| Restore replay does not duplicate journals | game-day report | tested scenario | date | reference topology only |

Do not publish “bank-grade,” “unhackable,” “exactly once,” “infinitely scalable,” or “fully compliant.”

## Frontend showcase requirements

Include polished recordings/screens for:

- customer wallet balance explanation;
- transfer confirmation and review/pending states;
- merchant API credentials and webhook delivery;
- operations transaction inspector;
- risk decision explanation;
- finance reconciliation exception resolution;
- audit verification;
- data-right export/closure blocker.

The UI should show dense data with hierarchy, accessible interaction, keyboard navigation, responsive layouts, and realistic failure recovery—not only a clean landing page.

## Final verification checklist

- fresh-clone setup succeeds;
- test suite and contract checks pass;
- hosted demo health and reset work;
- no real PII or secrets in repository/history/images/logs;
- demo URLs and documents are accessible;
- screenshots match current UI;
- seed scenarios are deterministic;
- release image/provenance/SBOM verify;
- all cited evidence exists;
- spelling and accounting terminology reviewed;
- limitations and license visible;
- security contact and safe-testing rules visible.

## X content pillars

### Pillar A — “The seven-minute Atlas failure demo”

A tight screen recording of the flagship scenario with trace, journal, webhook retry, and reconciliation break.

### Pillar B — “What I deliberately did not build”

- no real funds, card data, microservice sprawl, AI decisions, or fake compliance;
- explain why each exclusion improved signal.

### Pillar C — “The claims ledger for my portfolio project”

- show claim → test/evidence → limitation;
- explain why this is more credible than adjectives.

### Pillar D — “A guided code review of one transfer”

- API contract;
- domain command;
- risk and hold;
- posting transaction;
- outbox;
- adversarial tests;
- UI timeline.

### Pillar E — “What broke during the build”

Publish three real engineering postmortems: one concurrency bug, one security issue, one operational/reconciliation problem. Include regression tests.

## Do not waste time on

- a cinematic marketing video without technical evidence;
- hiding all code behind a hosted demo;
- dozens of shallow blog posts;
- fake testimonials or usage numbers;
- badges for standards not assessed;
- screenshots with only happy paths;
- calling unfinished designs implemented.
