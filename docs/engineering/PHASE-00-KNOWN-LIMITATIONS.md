# Phase 00 known limitations

Last reviewed: 2026-07-23 for ADR 0013 Phase 00 closure after successful hosted release run `29964442782`. These are explicit residual limits and trigger-bound future obligations, not claims of present capability.

| Limitation | Current evidence | Closure condition |
|---|---|---|
| Independent code-owner approval remains unavailable | ADR 0012, the closed solo-maintainer policy, sensitive declarations, self-review, and active no-bypass ruleset are verified. They do not manufacture organizational separation. | Obtain qualified independent approval before any real data, provider, money movement, second maintainer, deployment, or production-readiness trigger. |
| Staging/production-reference secret custody is not implemented | FND-031 is satisfied for the four closed environment configurations and distinct local/test material; no managed non-local material exists. | Revalidate before the first staging/production credential, managed secret provider, or deployable environment; prove provisioning, rotation, revocation, restore, access, and separation. |
| Backup/WAL storage is unencrypted synthetic volume state | FND-064 is satisfied for the ADR 0008 local/reference platform; the hosted release proves physical backup, WAL archive, and isolated three-second restore mechanics only. | Revalidate before the first product durable state, reference deployment, or backup encryption/key-custody change. |
| Product restore reconciliation is intentionally absent | No journal, outbox, inbox, idempotency, provider callback, object reference, encryption-key state, or financial state exists. | Add requirement-specific replay/reconciliation and missing-object/key tests with the first owning product state. |
| Event/job propagation and queue/retry emission are absent | ADR 0013 accepts the absent facets of FND-040/FND-042 only while worker/simulator inputs, events, queues, jobs, and retries do not exist. | Revalidate in the same change as the first such path and add propagation, retry/duplicate, bounded-label, and exported telemetry tests. |
| Alert routing is not deployed | Alert definitions, ownership, mutation tests, and runbooks exist; no receiver/backend has been selected or exercised. | Revalidate before the first deployed alert backend and prove routing, acknowledgement, escalation, and outage behavior. |

Successful run `29964442782` closes the earlier ruleset, fresh-host, bounded-web-shutdown, GHCR publication, signature/provenance, automated identity-verification, and retained-SBOM limitations. EVD-P00-S08-006 and EVD-P00-S08-007 remain preserved because their failures materially explain the cleanup and token-binding corrections.

No real money, real identity documents, cardholder data, production providers, deployment, compliance certification, availability target, security guarantee, or scale claim is supported by this synthetic feature-free release. Phase 00 completion applies only to that closed scope, with accepted decisions for `FND-026`, `FND-040`, and `FND-042`; every trigger in `phase-00-gate-policy.json` must be revalidated before the corresponding capability is introduced.
