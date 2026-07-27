ALTER TABLE atlas_identity.sessions
    ADD COLUMN step_up_action text,
    ADD COLUMN step_up_verified_at timestamptz,
    ADD CONSTRAINT sessions_step_up_binding_check CHECK (
        (
            step_up_action IS NULL
            AND step_up_verified_at IS NULL
        )
        OR (
            step_up_action ~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$'
            AND step_up_verified_at IS NOT NULL
            AND step_up_verified_at <= created_at + interval '1 minute'
        )
    );

CREATE TABLE atlas_identity.admin_session_revocation_requests (
    revocation_request_id text PRIMARY KEY CHECK (
        revocation_request_id ~ '^[a-z]{2,8}_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    actor_principal_id text NOT NULL REFERENCES atlas_identity.principals(principal_id),
    actor_session_id text NOT NULL REFERENCES atlas_identity.sessions(session_id),
    target_session_id text NOT NULL CHECK (
        target_session_id ~ '^ses_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    idempotency_key_sha256 bytea NOT NULL CHECK (octet_length(idempotency_key_sha256) = 32),
    request_sha256 bytea NOT NULL CHECK (octet_length(request_sha256) = 32),
    purpose text NOT NULL CHECK (purpose = 'security_review'),
    reason_code text NOT NULL CHECK (
        reason_code IN (
            'compromised_session',
            'suspected_account_takeover',
            'workforce_security_response'
        )
    ),
    outcome text NOT NULL CHECK (
        outcome IN ('executed', 'denied', 'step_up_required', 'not_found')
    ),
    current_revoked boolean NOT NULL,
    decision_id text NOT NULL CHECK (
        decision_id ~ '^dec_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    committed_at timestamptz NOT NULL,
    UNIQUE (actor_principal_id, idempotency_key_sha256)
);

CREATE INDEX admin_session_revocation_target_idx
    ON atlas_identity.admin_session_revocation_requests (target_session_id, committed_at);

INSERT INTO atlas_foundation.data_scope_registry
    (schema_name, table_name, scope_kind, tenant_column, global_scope_reason)
VALUES (
    'atlas_identity',
    'admin_session_revocation_requests',
    'global',
    NULL,
    'Workforce security revocation replay facts may target sessions from any identity population or tenant.'
);

REVOKE ALL ON atlas_identity.admin_session_revocation_requests FROM PUBLIC;
GRANT SELECT, INSERT ON atlas_identity.admin_session_revocation_requests TO atlas_api;
