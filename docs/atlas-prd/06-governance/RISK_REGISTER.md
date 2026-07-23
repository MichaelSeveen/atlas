# Atlas programme risk register

## Rating

Likelihood and impact use `Low`, `Medium`, `High`, `Critical`. Residual rating is reassessed after evidence, not presumed from planned controls.

| Risk ID | Risk | Initial | Primary treatment | Evidence required | Residual target |
|---|---|---|---|---|---|
| RSK-001 | Scope expands into five shallow products instead of one coherent platform | High | Phase gates, explicit non-goals, flagship journey | roadmap/release evidence review | Low |
| RSK-002 | Ledger is nominally double-entry but bypassable/inaccurate | Critical | controlled templates, DB grants, independent model/rebuild | property, permission, corruption, restore tests | Low |
| RSK-003 | Duplicate economic effect under retry/race | Critical | idempotency, business uniqueness, locks/state machines | concurrency/failpoint/model tests | Low |
| RSK-004 | External timeout is treated as failure and resubmitted | Critical | durable attempt and `outcome_unknown` lifecycle | provider simulator game day | Low |
| RSK-005 | Cross-tenant or privileged data/action exposure | Critical | deny-by-default policy, tenant context, negative matrix | manual/automated authorization report | Low |
| RSK-006 | Admin console becomes a direct-edit bypass | Critical | command APIs, approvals, reason, audit, DB restriction | direct-call/DB tamper tests | Low |
| RSK-007 | Real personal/card/bank data enters portfolio environment | High | synthetic-only policy, no card data, scanning and UI banners | repository/data inventory scan | Low |
| RSK-008 | Webhook feature introduces SSRF/internal network access | Critical | canonical URL/DNS/egress controls, no redirects | rebinding/private-IP hostile suite | Medium |
| RSK-009 | Secrets or sensitive data leak through logs/traces/client | Critical | classification, source redaction, canary scans, BFF | sink scan/browser inspection | Low |
| RSK-010 | Reconciliation auto-fixes discrepancies incorrectly | Critical | explicit exceptions, maker-checker adjustments, suspense governance | one-minor-unit and stale-approval tests | Low |
| RSK-011 | Restore succeeds technically but financial state is inconsistent | Critical | post-restore ledger/holds/outbox/provider/object reconciliation | DR report | Low |
| RSK-012 | Premature microservices/message complexity obscures correctness | High | modular monolith ADR and measured extraction criteria | dependency tests and ADR review | Low |
| RSK-013 | Performance claims are misleading or non-reproducible | High | workload/environment/invariant report template | signed benchmark artifacts | Low |
| RSK-014 | Standards mapping is represented as compliance | High | explicit disclaimer and claims review | release claims audit | Low |
| RSK-015 | Frontend is polished but hides ambiguous/restricted/stale states | High | state matrix, accessibility, failure-first E2E | browser critical journey tests | Low |
| RSK-016 | Event consumers depend on exactly-once/order assumptions | High | at-least-once contract, inbox, replay/out-of-order tests | event chaos report | Low |
| RSK-017 | Key rotation or restore makes encrypted evidence unusable | Critical | key inventory/versioning/rotation/restore test | rotation + isolated restore | Medium |
| RSK-018 | Dependency/toolchain compromise reaches release | High | pinned deps, SBOM, signed provenance, reviews | release supply-chain evidence | Medium |
| RSK-019 | Asynchronous jobs become permanently stuck without operator path | High | watchdog, age SLO, retry/DLQ/case workflow | stuck-job game day | Low |
| RSK-020 | Build-in-public posts disclose exploit paths or unsupported claims | High | content disclosure and claims checklist | content review log | Low |
| RSK-021 | Project takes too long and never reaches coherent demo | High | vertical phase acceptance and evidence gates | phase completion burn-up | Medium |
| RSK-022 | Simulation is so simplistic it fails to demonstrate real system judgement | High | adversarial provider/KYC/scenario engines | scenario catalogue/demo | Low |
| RSK-023 | Overengineering consumes effort without reviewer signal | High | non-goals and evidence-value prioritization | phase review: control/demo value | Medium |
| RSK-024 | Custom cryptography/auth implementation creates avoidable flaws | Critical | managed IdP/KMS abstractions and standard libraries | source/security review | Low |
| RSK-025 | Privacy retention/deletion breaks financial/audit history | High | pseudonymization/retention model and referential tests | data-rights exercise | Low |
| RSK-026 | Go concurrency/resource leak causes cascading outage | High | bounded workers, race/leak/soak tests | profile and endurance report | Low |
| RSK-027 | API docs drift from implementation | High | contract-first CI and conformance | deployed conformance report | Low |
| RSK-028 | Test suite gives confidence without independent oracle | High | model/differential/mutation requirements | mutation score and oracle review | Low |
| RSK-029 | Public demo is abused or mistaken for real financial service | High | synthetic labels, quotas, reset, no real rails, disclaimer | deployment/security review | Low |
| RSK-030 | Legal/regulatory interpretation is wrong or outdated | High | frame as design reference; qualified review required for real use | baseline version/review record | Medium |
| RSK-031 | Solo maintainer introduces or approves a sensitive defect without independent challenge | High | synthetic-only scope, protected pull requests, required hosted gates, sensitive-change declaration, fresh-context self-review, independent-review triggers | ADR 0012; policy canaries; ruleset and PR evidence; qualified review before trigger | Medium |
| RSK-032 | A Phase 00 applicability decision remains cited after product, event, job, credential, or recovery topology changes | High | closed requirement dispositions, hashed capability boundaries, guarded directory inventories, protected revalidation triggers | ADR 0013; `TestPhase00GateClosurePolicy`; same-change implementation and evidence at trigger | Medium |

## Risk review questions

At each phase gate:

- Which risk did this phase materially retire?
- Which new asset, trust boundary, privileged action, parser, outbound connection, or asynchronous state was introduced?
- What evidence changed the residual rating?
- Which risk is accepted only because this is synthetic/portfolio scope?
- Is any public claim stronger than the residual evidence?
