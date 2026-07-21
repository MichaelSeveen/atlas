# Secret exposure or key-version failure

Owner: security owner with the affected service owner. This runbook is for synthetic/reference environments until a managed production provider is approved.

1. Stop further exposure without deleting evidence. Identify environment, purpose, algorithm, version, activation window, consumers, and likely sinks; never paste material into tickets or chat.
2. Revoke or disable the affected version in its provider. Create a new version for the same environment and purpose; never reuse material or change algorithm to bypass failure.
3. Configure verification overlap only if the old version is not exposed. Exposed versions are revoked immediately, even if that invalidates in-flight synthetic work.
4. Raise the application minimum-version floor and redeploy revision-bound references. Confirm wrong-purpose/environment and old-version use fail closed.
5. Search source, Git history, build artifacts, logs, traces, metrics, browser storage, evidence, backups, and provider audit records using fingerprints—not raw material.
6. Rotate dependent credentials and assess whether encrypted/signed evidence needs reprocessing or explicit unverifiable status.
7. Run the S06 secret tests, document the affected window, and add sanitized incident evidence. Never claim deletion from immutable history without proof.
