# ADR 0007 — External financial and KYC providers are deterministic simulators

- **Status:** Accepted
- **Date:** 2026-07-20
- **Related requirements:** FND-011, KYC-001, EXT-001 through EXT-053

## Context

Real provider sandboxes are unstable, rate-limited, incomplete, credential-dependent, and usually do not expose the precise failures Atlas must demonstrate. Real rails or identity data would also increase risk and distract from system design.

## Decision

Implement provider interfaces plus deterministic simulators supporting named scenario IDs:

- immediate success/failure;
- validation decline;
- timeout before receipt;
- accepted then timeout;
- delayed status success/failure;
- callback before response;
- duplicate callback;
- out-of-order/conflicting callback;
- malformed/oversized response;
- reference reuse/mismatch;
- rate limit and prolonged outage;
- settlement file mismatch/duplicate/missing rows;
- KYC pass/fail/manual review/duplicate identity/replay.

Each scenario records request fingerprint, virtual time, expected provider state, callbacks, and settlement output. Simulator access is restricted to non-production/reference environments and visibly labeled.

## Consequences

- Failure tests are reproducible and can run in CI.
- Adapter architecture remains realistic without external dependency.
- Simulator correctness needs its own model/contract tests.
- Atlas does not claim integration certification with any real provider.

## Rejected alternatives

- Happy-path mock returning a boolean.
- Real money/identity providers for portfolio evidence.
- Hand-editing database states to simulate provider outcomes.

## Verification

Scenario contract tests, fixed virtual-clock tests, adapter differential tests, callback signature/replay suite, provider game days, settlement reconciliation fixtures.
