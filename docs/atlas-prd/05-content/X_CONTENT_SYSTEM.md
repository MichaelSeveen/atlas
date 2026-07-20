# Atlas evidence-led X content system

## Objective

Document the build in public without producing generic “day N of coding” posts. Every public claim must point to a concrete artifact: diagram, invariant, test, trace, benchmark, failure reproduction, migration, threat model, or design decision.

The audience should learn two things simultaneously:

1. how a serious financial system works; and
2. how you reason, verify, communicate trade-offs, and correct mistakes.

## Positioning

Core identity:

> Frontend-leaning full-stack engineer building trustworthy financial interfaces and correctness-critical Go systems.

Do not position Atlas as a licensed bank, payment provider, production-ready product, or compliance-certified platform. State that it is a synthetic portfolio system designed to demonstrate architecture, financial correctness, security, and operations.

## The seven content pillars

### 1. Financial correctness

Topics:

- wallet balances as liabilities;
- chart of accounts;
- debit/credit flows;
- available versus ledger balance;
- holds, capture, release, reversals, refunds;
- settlement and reconciliation;
- statements and period boundaries.

Evidence:

- posting diagrams;
- trial-balance output;
- property tests;
- independent balance rebuild;
- one-minor-unit mismatch demonstration.

### 2. Distributed failure and recovery

Topics:

- idempotency;
- ambiguous provider timeouts;
- transactional outbox;
- duplicate/out-of-order events;
- process crash after commit;
- retry budgets and dead letters;
- restore and replay.

Evidence:

- trace timeline;
- failure-injection video;
- before/after database facts;
- replay report;
- game-day postmortem.

### 3. Security and abuse resistance

Topics:

- BFF session model;
- tenant/object authorization;
- step-up and maker-checker;
- immutable financial tables;
- webhook signing and SSRF prevention;
- data classification and redaction;
- key rotation and supply chain.

Evidence:

- denied SQL/API attempt;
- authorization matrix;
- threat diagram;
- hostile webhook test;
- canary secret/redaction test;
- signed build artifact.

### 4. Operations and internal product design

Topics:

- transaction inspector;
- risk case workbench;
- finance reconciliation console;
- audit search;
- stale-state conflict UX;
- reasoned and reversible operator actions.

Evidence:

- annotated UI walkthrough;
- accessibility test;
- operator timeline;
- approval state race;
- incident workflow.

### 5. API and event design

Topics:

- money as string minor units;
- RFC-style errors;
- idempotency contract;
- ETag/preconditions;
- stable cursor pagination;
- event versioning and replay;
- webhook contract.

Evidence:

- OpenAPI/AsyncAPI excerpts;
- client retry demonstration;
- compatibility diff;
- contract test output.

### 6. Performance with integrity

Topics:

- workload models;
- hot-account contention;
- lock ordering;
- query plans;
- queue fairness;
- reconciliation streaming;
- Go allocation/goroutine analysis.

Evidence:

- reproducible benchmark report;
- p50/p95/p99 plus hardware/data shape;
- invariant result after load;
- profile/query-plan comparison.

### 7. Engineering judgement

Topics:

- modular monolith choice;
- what was deliberately not built;
- ADRs;
- migration strategy;
- finding and fixing flawed assumptions;
- known limitations.

Evidence:

- ADR excerpt;
- rejected-design comparison;
- honest postmortem;
- claims-to-evidence ledger.

## Post formats

### Single high-signal post

Structure:

1. concrete surprising statement;
2. the failure/constraint;
3. design decision;
4. evidence/result;
5. durable artifact link.

Example pattern:

> A provider timeout is not a failed transfer. It is an unknown transfer.
>
> In Atlas, the command keeps its original ID and funds reservation while a worker queries the rail. Retrying with a new key is blocked.
>
> I killed the API after the provider accepted the request. One transfer completed, one journal posted, and the trace shows exactly why.

### Short thread

Use 5–8 posts:

1. problem;
2. naive implementation and failure;
3. domain model;
4. transaction/security boundary;
5. adversarial test;
6. result;
7. limitation/trade-off;
8. artifact.

### Visual explainer

- one system diagram or state machine;
- maximum one core idea;
- labels readable on mobile;
- alt text;
- no meaningless architecture-cloud collage.

### Failure demonstration

- name the failpoint;
- show expected invariant before executing;
- inject fault;
- show user/operator state;
- show ledger/idempotency/outbox/audit evidence;
- explain recovery;
- state what remains unproven.

### Build report

Weekly or phase-close:

- completed controls/behaviours;
- strongest evidence;
- defect or assumption discovered;
- measured result;
- next risk to retire.

Avoid lists of commits or hours worked.

## Evidence hierarchy

Strongest to weakest:

1. reproducible adversarial test with financial/security invariant;
2. restore/game-day report;
3. benchmark with workload and environment;
4. contract plus conformance test;
5. code/design review with trade-offs;
6. annotated UI showing real operational workflow;
7. screenshot of ordinary happy path;
8. unsupported assertion.

Most posts should use levels 1–5.

## Phase content contract

A phase is not publicly documented by saying it is “done.” Publish at least:

- one domain/system explainer;
- one adversarial test or failure demo;
- one architecture/security decision;
- one product/frontend/operations artifact where applicable;
- one honest limitation or rejected alternative.

The phase files contain exact pillar ideas. `CONTENT_CALENDAR.md` sequences them.

## Writing rules

- Lead with the engineering truth, not “I’m excited to announce.”
- Use precise nouns and verbs: “posted one journal,” not “handled transactions robustly.”
- Distinguish accepted, reserved, posted, settled, reconciled, reversed, and failed.
- Never equate retries with exactly-once delivery.
- Never call a hash chain “immutable.” State its tamper-evidence boundary.
- Never claim scale without workload, environment, and invariant evidence.
- Never call standards mapping “compliance.”
- Explain why a choice exists and what failure it prevents.
- Include numbers only when reproducible.
- Show corrections publicly when a previous model was wrong.

## Security and disclosure guardrails

Do not publish:

- real secrets, tokens, internal IPs, unrestricted endpoint URLs, exploitable cloud/account details;
- raw vulnerability steps against a live public deployment before remediation;
- synthetic data that resembles a real person’s full identity;
- exact fraud thresholds that would weaken a real system;
- private employer code/designs;
- unredacted logs, traces, provider payloads, or object storage links.

Use sanitized fixtures and clearly label synthetic environments.

## Content production workflow

For each implementation pull request:

1. Add requirement and threat IDs.
2. Identify the strongest non-obvious engineering lesson.
3. Capture evidence during test execution—not by reconstructing a story later.
4. Store sanitized diagram/test/trace/report under versioned evidence.
5. Draft the post from the evidence.
6. Verify every claim and number against the artifact.
7. Link to a durable document/commit/release, not only a transient demo.
8. Record post URL against the phase/content ID after publishing.

## Content quality checklist

Before posting:

- Does this teach a specific financial/security/system concept?
- Is there an artifact proving the claim?
- Could a reviewer reproduce it?
- Did I state the failure mode and trade-off?
- Did I avoid inflated enterprise/compliance/scale language?
- Is the screenshot/diagram readable and sanitized?
- Does the post distinguish application state from financial truth?
- Is the opening sentence useful without the rest of the thread?
- Is there one clear takeaway rather than six shallow points?

## Portfolio connection

Each post should eventually map into one of six reviewer paths:

- financial correctness;
- distributed systems;
- security/privacy;
- APIs/integrations;
- operations/frontend;
- performance/recovery.

At release, curate the best evidence; do not make reviewers read the full chronological feed.
