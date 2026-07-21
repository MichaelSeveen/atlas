# Local broker unavailable

Scope: synthetic S04 NATS only. Do not infer production recovery semantics.

1. Confirm `GET /health/ready` is not ready and `GET /health/live` remains alive.
2. Inspect the `nats` container state and bounded recent logs; never print its token or runtime environment file.
3. Restore with `scripts/s04.ps1 -Action Up`, then run `-Action Smoke` and confirm the JetStream readiness marker.
4. If local state is corrupt, obtain the exact reset confirmation and recreate the contained local namespace. Do not delete broker volumes by hand.

No application stream, consumer, outbox, or financial event exists in S04. Delivery/replay recovery begins in later slices.
