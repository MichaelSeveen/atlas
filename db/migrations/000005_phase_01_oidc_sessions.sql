ALTER TABLE atlas_identity.sessions
    ADD COLUMN client_label text CHECK (client_label IS NULL OR length(client_label) BETWEEN 1 AND 120);

ALTER TABLE atlas_identity.sessions
    DROP CONSTRAINT sessions_assurance_check;

ALTER TABLE atlas_identity.sessions
    ADD CONSTRAINT sessions_assurance_check
    CHECK (assurance IN ('baseline', 'stepped-up', 'phishing-resistant'));

ALTER TABLE atlas_audit.audit_events
    DROP CONSTRAINT audit_events_session_assurance_check;

ALTER TABLE atlas_audit.audit_events
    ADD CONSTRAINT audit_events_session_assurance_check
    CHECK (session_assurance IN ('none', 'baseline', 'stepped-up', 'phishing-resistant'));

CREATE TABLE atlas_identity.oidc_transactions (
    transaction_id text PRIMARY KEY CHECK (transaction_id ~ '^[a-z]{2,8}_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'),
    transaction_kind text NOT NULL CHECK (transaction_kind IN ('login', 'step-up')),
    population text NOT NULL CHECK (population IN ('customer', 'merchant', 'workforce')),
    state_sha256 bytea NOT NULL CHECK (octet_length(state_sha256) = 32),
    nonce_sha256 bytea NOT NULL CHECK (octet_length(nonce_sha256) = 32),
    pkce_verifier_ciphertext bytea NOT NULL CHECK (
        octet_length(pkce_verifier_ciphertext) BETWEEN 60 AND 2048
    ),
    encryption_key_version bigint NOT NULL CHECK (encryption_key_version > 0),
    return_to text NOT NULL CHECK (return_to IN ('/customer', '/merchant', '/workforce')),
    principal_id text REFERENCES atlas_identity.principals(principal_id),
    replaced_session_id text REFERENCES atlas_identity.sessions(session_id),
    requested_action text CHECK (
        requested_action IS NULL
        OR requested_action ~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$'
    ),
    status text NOT NULL CHECK (status IN ('pending', 'consumed')),
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    UNIQUE (state_sha256),
    CHECK (expires_at > created_at),
    CHECK (
        (status = 'pending' AND consumed_at IS NULL)
        OR (status = 'consumed' AND consumed_at IS NOT NULL)
    ),
    CHECK (
        (transaction_kind = 'login' AND principal_id IS NULL AND requested_action IS NULL)
        OR (
            transaction_kind = 'step-up'
            AND principal_id IS NOT NULL
            AND replaced_session_id IS NOT NULL
            AND requested_action IS NOT NULL
        )
    ),
    CHECK (
        (population = 'customer' AND return_to = '/customer')
        OR (population = 'merchant' AND return_to = '/merchant')
        OR (population = 'workforce' AND return_to = '/workforce')
    )
);

CREATE INDEX oidc_transactions_pending_expiry_idx
    ON atlas_identity.oidc_transactions (expires_at, transaction_id)
    WHERE status = 'pending';

CREATE TABLE atlas_identity.session_revocation_requests (
    revocation_request_id text PRIMARY KEY CHECK (revocation_request_id ~ '^[a-z]{2,8}_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'),
    principal_id text NOT NULL REFERENCES atlas_identity.principals(principal_id),
    idempotency_key_sha256 bytea NOT NULL CHECK (octet_length(idempotency_key_sha256) = 32),
    request_sha256 bytea NOT NULL CHECK (octet_length(request_sha256) = 32),
    include_current boolean NOT NULL,
    current_revoked boolean NOT NULL,
    committed_at timestamptz NOT NULL,
    UNIQUE (principal_id, idempotency_key_sha256)
);

INSERT INTO atlas_foundation.data_scope_registry
    (schema_name, table_name, scope_kind, tenant_column, global_scope_reason)
VALUES
    (
        'atlas_identity',
        'oidc_transactions',
        'global',
        NULL,
        'Short-lived authentication protocol state is population-scoped before tenant authority is established.'
    ),
    (
        'atlas_identity',
        'session_revocation_requests',
        'global',
        NULL,
        'Self-session revocation replay records are principal-scoped and may cover every tenant context.'
    );

REVOKE ALL ON atlas_identity.oidc_transactions, atlas_identity.session_revocation_requests FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON atlas_identity.oidc_transactions TO atlas_api;
GRANT SELECT, INSERT ON atlas_identity.session_revocation_requests TO atlas_api;
