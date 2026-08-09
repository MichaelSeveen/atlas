CREATE TABLE atlas_identity.api_credentials (
    credential_id text PRIMARY KEY CHECK (
        credential_id ~ '^key_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    tenant_id text NOT NULL REFERENCES atlas_identity.organizations(tenant_id),
    name text NOT NULL CHECK (length(name) BETWEEN 1 AND 120),
    secret_verifier_sha256 bytea NOT NULL CHECK (octet_length(secret_verifier_sha256) = 32),
    secret_hint text NOT NULL CHECK (secret_hint ~ '^[A-Za-z0-9_-]{8}$'),
    verifier_algorithm text NOT NULL CHECK (verifier_algorithm = 'sha256'),
    verifier_version bigint NOT NULL CHECK (verifier_version = 1),
    scopes text[] NOT NULL CHECK (scopes = ARRAY['identity:read']::text[]),
    environment text NOT NULL CHECK (
        environment IN ('local', 'test', 'staging', 'production-reference')
    ),
    audience text NOT NULL CHECK (audience = 'atlas-api'),
    status text NOT NULL CHECK (status IN ('active', 'rotating', 'revoked')),
    version bigint NOT NULL CHECK (version > 0),
    expires_at timestamptz NOT NULL,
    overlap_ends_at timestamptz,
    last_used_at timestamptz,
    last_used_network_signal_sha256 bytea CHECK (
        last_used_network_signal_sha256 IS NULL
        OR octet_length(last_used_network_signal_sha256) = 32
    ),
    last_use_anomalous boolean NOT NULL DEFAULT false,
    authentication_count bigint NOT NULL DEFAULT 0 CHECK (authentication_count >= 0),
    created_by_principal_id text NOT NULL REFERENCES atlas_identity.principals(principal_id),
    previous_credential_id text REFERENCES atlas_identity.api_credentials(credential_id),
    replacement_credential_id text REFERENCES atlas_identity.api_credentials(credential_id),
    created_at timestamptz NOT NULL,
    revoked_at timestamptz,
    UNIQUE (credential_id, tenant_id),
    CHECK (expires_at > created_at),
    CHECK (previous_credential_id IS NULL OR previous_credential_id <> credential_id),
    CHECK (replacement_credential_id IS NULL OR replacement_credential_id <> credential_id),
    CHECK (
        (status = 'active' AND overlap_ends_at IS NULL AND revoked_at IS NULL)
        OR
        (status = 'rotating' AND overlap_ends_at IS NOT NULL AND revoked_at IS NULL
            AND replacement_credential_id IS NOT NULL)
        OR
        (status = 'revoked' AND revoked_at IS NOT NULL)
    )
);

CREATE INDEX api_credentials_tenant_created_idx
    ON atlas_identity.api_credentials (tenant_id, created_at DESC, credential_id DESC);
CREATE INDEX api_credentials_rotation_expiry_idx
    ON atlas_identity.api_credentials (overlap_ends_at, expires_at, credential_id)
    WHERE status IN ('active', 'rotating');

CREATE TABLE atlas_identity.api_credential_mutation_requests (
    tenant_id text NOT NULL REFERENCES atlas_identity.organizations(tenant_id),
    actor_principal_id text NOT NULL REFERENCES atlas_identity.principals(principal_id),
    actor_session_id text NOT NULL REFERENCES atlas_identity.sessions(session_id),
    operation text NOT NULL CHECK (operation IN ('create', 'rotate', 'revoke')),
    idempotency_key_sha256 bytea NOT NULL CHECK (octet_length(idempotency_key_sha256) = 32),
    request_sha256 bytea NOT NULL CHECK (octet_length(request_sha256) = 32),
    requested_credential_id text REFERENCES atlas_identity.api_credentials(credential_id)
        DEFERRABLE INITIALLY DEFERRED,
    result_credential_id text NOT NULL REFERENCES atlas_identity.api_credentials(credential_id)
        DEFERRABLE INITIALLY DEFERRED,
    authorization_decision_id text NOT NULL CHECK (
        authorization_decision_id ~ '^dec_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    audit_event_id text NOT NULL CHECK (
        audit_event_id ~ '^aud_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    created_at timestamptz NOT NULL,
    PRIMARY KEY (tenant_id, actor_principal_id, operation, idempotency_key_sha256),
    CHECK (
        (operation = 'create' AND requested_credential_id IS NULL)
        OR
        (operation IN ('rotate', 'revoke') AND requested_credential_id IS NOT NULL)
    ),
    CHECK (operation <> 'revoke' OR requested_credential_id = result_credential_id)
);

INSERT INTO atlas_foundation.data_scope_registry
    (schema_name, table_name, scope_kind, tenant_column, global_scope_reason)
VALUES
    ('atlas_identity', 'api_credentials', 'tenant', 'tenant_id', NULL),
    ('atlas_identity', 'api_credential_mutation_requests', 'tenant', 'tenant_id', NULL);

REVOKE ALL ON
    atlas_identity.api_credentials,
    atlas_identity.api_credential_mutation_requests
FROM PUBLIC;
GRANT SELECT, INSERT ON
    atlas_identity.api_credentials,
    atlas_identity.api_credential_mutation_requests
TO atlas_api;
GRANT UPDATE (
    status, version, overlap_ends_at, last_used_at, last_used_network_signal_sha256,
    last_use_anomalous, authentication_count, replacement_credential_id, revoked_at
) ON atlas_identity.api_credentials TO atlas_api;
