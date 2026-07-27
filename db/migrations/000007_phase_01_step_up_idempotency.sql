CREATE TABLE atlas_identity.step_up_challenge_requests (
    challenge_request_id text PRIMARY KEY CHECK (
        challenge_request_id ~ '^[a-z]{2,8}_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    principal_id text NOT NULL REFERENCES atlas_identity.principals(principal_id),
    population text NOT NULL CHECK (population IN ('customer', 'merchant', 'workforce')),
    tenant_id text REFERENCES atlas_identity.organizations(tenant_id),
    global_scope text,
    idempotency_scope_sha256 bytea NOT NULL UNIQUE CHECK (
        octet_length(idempotency_scope_sha256) = 32
    ),
    request_sha256 bytea NOT NULL CHECK (octet_length(request_sha256) = 32),
    requested_action text NOT NULL CHECK (
        requested_action ~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$'
    ),
    lifecycle text NOT NULL CHECK (
        lifecycle IN ('processing', 'completed', 'failed-retryable')
    ),
    transaction_id text UNIQUE REFERENCES atlas_identity.oidc_transactions(transaction_id),
    authorization_url text CHECK (
        authorization_url IS NULL OR length(authorization_url) BETWEEN 16 AND 4096
    ),
    challenge_expires_at timestamptz,
    processing_expires_at timestamptz,
    correlation_id text NOT NULL CHECK (
        correlation_id ~ '^cor_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    retained_until timestamptz NOT NULL,
    CHECK (
        (tenant_id IS NOT NULL AND global_scope IS NULL)
        OR (tenant_id IS NULL AND global_scope = 'identity-security')
    ),
    CHECK (
        (population IN ('customer', 'merchant') AND tenant_id IS NOT NULL)
        OR (population = 'workforce' AND global_scope = 'identity-security')
    ),
    CHECK (retained_until > created_at),
    CHECK (
        (
            lifecycle = 'processing'
            AND transaction_id IS NULL
            AND authorization_url IS NULL
            AND challenge_expires_at IS NULL
            AND processing_expires_at IS NOT NULL
        )
        OR (
            lifecycle = 'completed'
            AND transaction_id IS NOT NULL
            AND authorization_url IS NOT NULL
            AND challenge_expires_at IS NOT NULL
            AND processing_expires_at IS NULL
        )
        OR (
            lifecycle = 'failed-retryable'
            AND transaction_id IS NULL
            AND authorization_url IS NULL
            AND challenge_expires_at IS NULL
            AND processing_expires_at IS NULL
        )
    )
);

CREATE INDEX step_up_challenge_requests_retention_idx
    ON atlas_identity.step_up_challenge_requests (retained_until, challenge_request_id);

CREATE INDEX step_up_challenge_requests_processing_idx
    ON atlas_identity.step_up_challenge_requests (processing_expires_at, challenge_request_id)
    WHERE lifecycle = 'processing';

INSERT INTO atlas_foundation.data_scope_registry
    (schema_name, table_name, scope_kind, tenant_column, global_scope_reason)
VALUES (
    'atlas_identity',
    'step_up_challenge_requests',
    'mixed',
    'tenant_id',
    'Workforce step-up replay state is global while customer and merchant state is tenant-scoped.'
);

REVOKE ALL ON atlas_identity.step_up_challenge_requests FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON atlas_identity.step_up_challenge_requests TO atlas_api;
