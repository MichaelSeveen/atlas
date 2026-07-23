# ADR 0013 — Phase 00 closes against the implemented topology with mandatory revalidation triggers

- **Status:** Accepted
- **Date:** 2026-07-23
- **Owners:** Product, platform, security, and reliability owner
- **Related requirements/threats:** FND-011; FND-031; FND-040; FND-042; FND-064; THR-009; THR-015; THR-019; THR-020; THR-025; THR-030; THR-042; THR-043; THR-045; THR-048; THR-049; THR-054; THR-060; RSK-032
- **Supersedes/superseded by:** Clarifies the Phase 00 completion interpretation of ADR 0008 and amends the outstanding Phase 00 recovery statement in ADR 0010; it does not supersede any future deployment, encryption, product-replay, event, job, or provider requirement

## Context

Phase 00 deliberately produced a feature-free engineering foundation. It has no product schema, identity exchange, provider adapter, broker stream, outbox, consumer, worker job, financial state, product object/key state, or deployed staging/production environment. Requiring executable behavior across those absent surfaces would force fake product semantics into the foundation and conflict with the Phase 00 non-goals. Treating every absent future surface as silently satisfied would create the opposite failure: later code could cross a boundary without activating its required controls.

The Definition of Done permits a mandatory phase requirement to be implemented or explicitly rejected through a documented risk/scope decision. This ADR applies that rule only to facets that have no causal or durable state path in the implemented Phase 00 topology. It does not weaken the canonical requirement when the relevant path begins to exist.

## Decision

Phase 00 is evaluated as a closed, synthetic, feature-free topology. The machine-readable policy at `docs/engineering/phase-00-gate-policy.json` records the evidence, deferred surfaces, and triggers below.

### FND-011 — deterministic synthetic seeds

The closed, checksummed foundation seed manifest is the Phase 00 seed. It contains fixed virtual time, two tenants, three users, two account identity fixtures without financial state, and eight deterministic provider scenario identities/seeds. Schema loading and provider execution are not part of the wording needed to establish the seed catalogue and are deferred until an owning product schema or simulator contract exists. FND-011 is satisfied at the Phase 00 scope.

### FND-031 — environment credential separation

All four required environment configurations contain complete, environment-and-purpose-scoped references for signing, encryption, database, identity, merchant, broker, and object-storage credentials. The validator rejects a shared reference across any environment, and generated local/test material has distinct fingerprints. Staging and production-reference material does not exist, so no deployment or managed custody is claimed. FND-031 is satisfied for the configured environments; the first provisioned non-local environment must add actual provider/fingerprint, rotation, revocation, and recovery evidence.

### FND-040 — context propagation

Request ID, correlation ID, and W3C trace context propagate across every causally reachable Phase 00 request boundary: edge, API, readiness child, and database span. Worker and simulator processes have lifecycle telemetry but receive no request, message, callback, event, or job, so there is no causal context to propagate through them. The worker/event/simulator-input facets are explicitly deferred by scope decision. The first such input or broker stream must implement propagation, duplicate/retry behavior, and an exported continuity test in the same change.

### FND-042 — baseline metrics

The implemented HTTP RED, database readiness/pool, and build/revision metrics are emitted with bounded labels, dashboards, owned alerts, runbooks, and mutation tests. Queue lag and worker retry metrics are registered as `definition-only` and are statically forbidden from emission while no queue/job exists. Their runtime facets, and deployment of an alert receiver, are explicitly deferred by scope decision. The first queue, job, retry path, or deployed alert backend must add emission, bounded labels, routing/outage tests, and runbook evidence.

### FND-064 — reference recovery

ADR 0008 defines the current local/reference platform. ADR 0010 configures native physical base backup, continuous WAL archiving, verification, and isolated point-in-time recovery for that platform. The same path passed on a fresh hosted Docker runner, including a one-second base backup and three-second isolated restore. FND-064 is satisfied for the synthetic reference environment. Unencrypted developer volumes, production retention/key custody, and replay of nonexistent product/object/key/financial state remain explicit limitations and become mandatory when those surfaces are introduced.

### FND-026 — separate accepted deviation

ADR 0012 remains the sole decision for independent review. Phase progression may use that bounded synthetic-only deviation because the protected ruleset, required gates, declaration, and revalidation triggers are evidenced. FND-026 is not represented as independently satisfied.

## Revalidation and enforcement

The closure policy hashes the current seed, environment, worker/simulator, metric, migration, and solo-governance boundaries and closes the relevant directory inventories. `TestPhase00GateClosurePolicy` fails when a guarded artifact changes, a guarded directory gains or loses a file, a disposition/trigger is removed, or the independent-review triggers diverge from ADR 0012's policy.

Revalidation is mandatory with the first:

- product schema or executable provider scenario;
- worker/simulator input, event/consumer, broker stream, queue, job, or retry path;
- provisioned staging/production-reference credential or managed secret provider;
- deployed alert backend;
- product durable state, object/key state, or reference deployment recovery change;
- ADR 0012 independent-review trigger.

Revalidation means implementing the newly applicable requirement facet and its tests/evidence, then deliberately updating the policy guards in the same protected pull request. Deleting a trigger or merely refreshing a hash without the owning evidence is prohibited.

## Consequences and residual risk

- Phase 01 may begin without fake Phase 00 identity tables or an invented message path.
- Later phases cannot rely on this decision after they activate one of the guarded surfaces.
- Hash guards are deliberate tripwires, not proof that changed behavior is correct; review must inspect why a guard changed.
- A contributor could attempt to introduce a novel path outside the guarded inventory. Architecture/module policy, sensitive-path review, traceability review, and THR-060 evidence-drift checks remain defense in depth.
- Local backup encryption, independent human review, managed credential custody, product replay, event/job telemetry, and deployed alert routing remain limitations until their triggers occur.

## Financial, authorization, and failure boundaries

This decision adds no endpoint, identity exchange, authorization rule, product schema, transaction, event, provider call, worker job, money representation, or financial state. It creates no idempotency/concurrency or before/after-commit product behavior. Its failure mode is stale applicability: future capability could be added while still citing feature-free evidence. The closed policy, guarded inventories, required PR checks, and revision-bound evidence are the controls for that risk.

## Migration and rollback/exit strategy

Each triggered surface exits only its applicable deferral. Add the real implementation, tests, telemetry, recovery proof, and traceability first; then update the policy guard and disposition. A later deployment ADR must supersede the local/reference credential and recovery decisions before deployment. Rollback restores the prior guarded feature-free topology or preserves the stronger newly applicable control; it must not silently reinstate a deferral after product state exists.

## Verification and evidence

- `go test ./internal/architecture -run TestPhase00GateClosurePolicy -count=1`
- `pwsh -NoProfile -File ./scripts/verify-s08.ps1`
- Existing S04 seed/config canaries, S06 telemetry/catalog canaries, S08 live recovery, protected PR/ruleset evidence, and EVD-P00-S08-008 hosted release evidence
- Seeded policy failures for a removed requirement, missing trigger, drifted guarded digest, and expanded guarded directory
