# Atlas verification strategy

## Objective

Tests must establish evidence for financial correctness, authorization, security, deterministic recovery, and operational safety. Coverage percentage is a diagnostic, not the acceptance criterion. The strongest tests target invariants and boundaries that could lose money, expose data, create duplicate economic effects, or leave operators unable to recover.

## Verification principles

1. **Test the economic effect, not the HTTP response.** A `201` is insufficient unless the expected hold, journal, outbox row, audit event, and lifecycle state are verified.
2. **Use independent oracles.** Ledger projection, reconciliation, statement totals, and risk decisions are checked against separately implemented models or fixtures—not by calling the production helper twice.
3. **Assume ambiguous failure.** Test process death, connection loss, timeout, retry, redelivery, and reordering at every durability boundary.
4. **Prove denial.** Authorization, database permissions, masking, egress restrictions, and immutable-table controls require negative tests.
5. **Preserve the real infrastructure semantics.** Integration tests use PostgreSQL, the chosen broker, object storage, identity provider, and browser; critical paths do not use repository mocks.
6. **Make time controllable.** Domain code receives a clock. Tests advance time across expiry, day/month/year boundaries, DST/timezone reporting boundaries, and retry schedules.
7. **Make nondeterminism reproducible.** Property/fuzz seeds, workload, environment, source revision, and failing corpus are retained.
8. **Every escaped defect gets a regression test** at the lowest useful layer plus an end-to-end reproduction for critical defects.

## Test architecture

### Layer A — Pure domain and state-machine tests

Fast Go tests for:

- amount/currency primitives;
- posting templates;
- transfer, payment, refund, hold, case, approval, and KYC state machines;
- authorization policy inputs and decisions;
- fee/FX/rounding rules;
- canonical request hashing;
- signature canonicalization;
- parser normalization.

These tests use no database and are suitable for exhaustive table tests, property tests, mutation testing, and fuzzing.

### Layer B — PostgreSQL integration tests

Run against real PostgreSQL with the production schema and role grants. Cover:

- transaction isolation and lock ordering;
- unique/exclusion/check/deferred constraints;
- application-role denial of immutable table updates;
- idempotency races;
- outbox atomicity;
- tenant predicates and row ownership;
- migrations and rollback/forward-fix behaviour;
- query plans on representative data;
- consistent-snapshot verification and report generation.

Do not replace this layer with SQLite or mocked repositories.

### Layer C — Component tests

Start a Go API/worker or frontend component with real protocol boundaries but constrained dependencies. Examples:

- provider adapter against deterministic simulator;
- webhook sender against hostile test server;
- BFF against a test identity provider;
- reconciliation importer against object storage and malformed files;
- React/Vue view against generated contract fixtures and accessibility engine.

### Layer D — Contract tests

- OpenAPI request/response conformance.
- AsyncAPI schema fixtures and compatibility.
- Provider adapter consumer-driven contracts.
- Merchant webhook signature verification fixtures in Go and TypeScript.
- Error catalogue status/code/retryability conformance.
- Generated client/server type drift.

### Layer E — End-to-end journey tests

Run through browser and public APIs with all relevant services. Critical journeys:

- customer onboarding and secure session;
- internal transfer;
- external transfer with delayed success;
- merchant payment, capture, settlement, and refund;
- risk hold and analyst decision;
- reconciliation exception and adjustment approval;
- statement generation and download grant;
- session/credential revocation;
- operator transaction investigation.

Tests assert backend evidence through authorized test-only observation APIs or database verification harness—not brittle UI text alone.

### Layer F — Adversarial, chaos, restore, and performance verification

Scheduled and release-gated suites inject failure and malicious inputs. They produce reports, traces, invariant results, and known limitations.

## Required test techniques

### Example-based tests

Use named business scenarios and exact expected postings/state. They improve reviewability and accounting explanation.

### Property-based tests

Generate valid and invalid sequences to verify invariants such as:

- per-currency debits equal credits;
- available balance never exceeds policy-defined derivation;
- total captured never exceeds hold/payment authorization;
- total refunded never exceeds settled/captured amount;
- a transfer reaches at most one successful economic completion;
- reversal creates an equal-and-opposite effect without deleting history;
- idempotent replay leaves state observationally equivalent;
- statement opening + movement = closing.

### Model-based state-machine tests

Maintain a small independent reference model. Generate commands including invalid transitions, retries, callbacks, expiry, cancellation, reversal, and operator intervention. Compare every observable state and money effect after each step.

### Metamorphic tests

Examples:

- permuting independent journal postings does not change account totals;
- duplicate event delivery does not change the final economic state;
- rerunning reconciliation on identical immutable input produces identical classifications and totals;
- formatting/locale changes do not change minor-unit values;
- adding an irrelevant authorized search result does not change previously returned cursor items;
- restart/replay produces the same terminal state as uninterrupted execution.

### Mutation tests

Deliberately mutate critical rules and require the suite to detect them:

- remove debit-credit equality check;
- invert account normal-side logic;
- omit tenant predicate;
- skip step-up or maker-checker check;
- change `>` to `>=` at a limit boundary;
- remove idempotency request-hash comparison;
- process duplicate provider event twice;
- accept expired signature/quote;
- omit release of a failed transfer hold;
- disable CSV formula sanitization.

### Fuzz tests

Use Go fuzzing and retained corpus for:

- JSON/cursor/identifier parsing;
- amount/currency/date/rate primitives;
- webhook signatures and HTTP structured fields;
- provider callbacks;
- CSV settlement import;
- metadata allowlists;
- state transition command decoders;
- encrypted envelope metadata;
- redaction and log encoding.

Fuzz failures become permanent corpus cases.

### Differential tests

Compare:

- database-derived balances versus an independent in-memory accounting model;
- Go and TypeScript signature implementations;
- CSV, JSON, and PDF statement totals;
- old/new reconciliation rulesets on frozen datasets;
- old/new API versions for promised compatibility;
- primary versus restored environment after replay.

## Test data strategy

- Synthetic-only identities and financial details.
- Deterministic named scenarios plus generated data.
- Factories construct valid domain objects through public builders—not impossible database fixtures by default.
- A separate corruption fixture layer is allowed only for verification and incident exercises.
- Data generators model multiple tenants, currencies, timezone/reporting boundaries, high-contention accounts, and deep history.
- No production database copies or real customer data.

## Isolation and cleanup

- Test schemas/databases are unique per parallel worker.
- Broker consumer groups and topics/namespaces are isolated.
- Object keys include run ID and are lifecycle-cleaned.
- Fake clock is process/test scoped.
- Tests never depend on execution order.
- Cleanup failure is reported, not silently ignored.

## Deterministic failure injection

Every critical boundary has named failpoints disabled in normal builds and safely enabled in test environments:

- before/after domain validation;
- before first SQL write;
- after journal insert;
- before transaction commit;
- after commit before response;
- after outbox claim before publish;
- after broker publish before outbox acknowledgement;
- after provider receives request before client response;
- after webhook recipient processes before sender records success;
- during statement object write;
- during restore replay.

Failpoints identify expected invariant and recovery path.

## Frontend verification

### Component and interaction

- Use accessible queries by role/name rather than implementation selectors.
- Test loading, empty, partial, stale, unauthorized, restricted, ambiguous, failed, and recovered states.
- Test exact minor-unit rendering with large values and locale switches.
- Test keyboard-only operation, focus restoration, live-region announcements, semantic tables, and non-colour status communication.
- Test permission changes while page is open; hidden controls are not the authorization control.
- Test stale ETag/approval and conflict resolution UX.
- Test session expiry and step-up return-to-flow without replaying money commands incorrectly.

### Browser security

- HttpOnly session tokens are absent from JavaScript storage.
- CSRF is required for cookie-authenticated mutations.
- CSP blocks inline/eval/untrusted frames as designed.
- Sensitive pages have safe cache headers and clear state on logout/tenant switch.
- Cross-origin requests and postMessage channels follow strict origin checks.
- CSV/HTML/error rendering resists injection.

### Visual tests

Use narrowly for high-value financial tables, timeline relationships, print/PDF previews, and responsive operational layouts. A screenshot diff does not replace semantic assertions.

## Backend verification in Go

- `go test ./...` for ordinary suites.
- `go test -race` on concurrency-capable packages and selected integration suites.
- `go test -fuzz` targets in scheduled CI with retained corpus.
- Benchmark functions use representative fixtures and report allocations.
- Static checks reject float money, domain `time.Now()`, unsafe random, ignored errors, unconstrained goroutines, and forbidden module imports.
- Goroutine leak detection wraps worker/component tests.
- Tests assert error identity, not just message text.

## Database verification

- Migration from empty database and every supported prior release.
- Expand/backfill/contract migrations tested under concurrent traffic.
- Lock timeout and statement timeout are asserted.
- Query plans captured for critical queries at scale-shaped data volumes.
- Backup restore validates grants, extensions, constraints, sequences, and encrypted fields—not only row counts.
- Periodic independent balance rebuild compares exact results.

## CI test lanes

### Pull request, blocking

- formatting, lint, type checks;
- unit/property tests with fixed seed set;
- PostgreSQL integration tests;
- frontend component/accessibility tests;
- OpenAPI/AsyncAPI validation and breaking-change analysis;
- migration tests;
- SAST, dependency, secret scan;
- selected race and fuzz corpus replay;
- requirement/evidence link validation.

### Main branch, blocking deployment

- full integration and contract suites;
- browser critical journeys;
- container/IaC scan and signed artifact checks;
- mutation test threshold on critical modules;
- representative query-plan checks;
- provider and webhook hostile-server suite.

### Scheduled/nightly

- extended fuzzing;
- randomized state-machine tests;
- race detector across broader packages;
- dependency outage/chaos suite;
- large reconciliation and statement datasets;
- restore rehearsal in isolated environment;
- performance trend run.

### Release candidate

- full adversarial catalogue;
- disaster-recovery exercise or recent valid evidence;
- security verification checklist;
- workload-defined performance run;
- manual authorization matrix review;
- known-risk sign-off and evidence manifest.

## Flaky-test policy

- No automatic infinite rerun.
- One controlled rerun may collect diagnostics but original failure remains visible.
- Critical financial/security tests cannot be quarantined to ship.
- Quarantined non-critical test has owner, issue, expiry, risk statement, and replacement evidence.
- Track flake rate and root cause; nondeterminism is a defect.

## Coverage and quality metrics

Track:

- requirements with executable evidence;
- mutation score in ledger, authorization, idempotency, signature, and state-machine packages;
- branch coverage as a diagnostic;
- fuzz corpus growth and unique crash count;
- escaped defect classes;
- flaky test rate;
- critical journey duration;
- restore verification age;
- security finding regression coverage;
- percentage of privileged actions with negative tests.

## Definition of verified

A requirement is verified only when its traceability row identifies:

- test or manual procedure ID;
- test layer and environment;
- expected invariant/control;
- captured evidence path;
- owner;
- last execution revision/time;
- result and unresolved limitation.
