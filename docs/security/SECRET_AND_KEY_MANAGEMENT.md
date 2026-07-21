# Secret and key-management baseline

The S06 abstraction is provider-neutral and performs no signing or encryption. It defines the boundary a managed production provider must satisfy before product cryptography is introduced.

## Reference and metadata policy

References use exactly `secret://atlas/{environment}/{purpose}`. Environment and purpose cannot be inferred or substituted. Every version records owner, algorithm, monotonically increasing version, activation, expiry, grace deadline, and revocation state. Signing, encryption, and opaque credentials are distinct purposes; material reuse across environment or purpose boundaries is rejected.

Applications ask for an explicit algorithm and minimum version. Active use selects only the newest non-revoked version in its activation window. Verification may receive the current version plus an old version only during the declared grace window. Unknown, unavailable, expired, revoked, wrong-environment, wrong-purpose, wrong-algorithm, and below-floor requests fail closed with stable errors.

Raw material is copied only into a callback and wiped immediately afterward. The application stores references, never durable raw material. The local environment may use disposable generated credentials; it is not evidence of a production secret manager, HSM, or key custody control.

## Planned rotation procedure

1. Name the purpose, environment, owner, algorithm, current version, consumers, rollback floor, maximum overlap, and verification evidence.
2. Generate the next version inside the approved provider. Never copy a version across environments or purposes.
3. Make readers accept current plus previous verification versions before activating the new writer version.
4. Activate the new version and raise the consumer minimum-version floor. Observe stable version-use telemetry; never log material.
5. Complete synthetic in-flight verification during the bounded grace window.
6. Revoke the old version, verify that below-floor use fails, and remove it according to provider destruction policy.
7. Add revision-bound evidence. A failed rotation follows `docs/runbooks/SECRET_EXPOSURE.md` if exposure is possible; otherwise restore provider availability without lowering the floor.

## Recovery and restore

A restore must recover provider references, version metadata, and the required current/grace versions together. Restoring an older database/configuration must not lower the minimum accepted version. The `ADV-DR-004` abstraction facet is tested by outage/recovery, overlap, wrong-boundary, and downgrade cases; actual provider backup/HSM recovery remains a later deployment gate.
