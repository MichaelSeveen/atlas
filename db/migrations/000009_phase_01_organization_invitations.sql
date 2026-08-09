CREATE TABLE atlas_identity.organization_invitations (
    invitation_id text PRIMARY KEY CHECK (
        invitation_id ~ '^inv_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    tenant_id text NOT NULL REFERENCES atlas_identity.organizations(tenant_id),
    invited_by_principal_id text NOT NULL REFERENCES atlas_identity.principals(principal_id),
    invited_by_session_id text NOT NULL REFERENCES atlas_identity.sessions(session_id),
    email_sha256 bytea NOT NULL CHECK (octet_length(email_sha256) = 32),
    email_hint text NOT NULL CHECK (length(email_hint) BETWEEN 3 AND 254),
    role_id text NOT NULL,
    population text NOT NULL CHECK (population = 'merchant'),
    token_sha256 bytea NOT NULL UNIQUE CHECK (octet_length(token_sha256) = 32),
    idempotency_key_sha256 bytea NOT NULL CHECK (octet_length(idempotency_key_sha256) = 32),
    request_sha256 bytea NOT NULL CHECK (octet_length(request_sha256) = 32),
    creation_decision_id text NOT NULL CHECK (
        creation_decision_id ~ '^dec_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    status text NOT NULL CHECK (status IN ('pending', 'revoked', 'expired')),
    terminal_at timestamptz,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL,
    version bigint NOT NULL CHECK (version > 0),
    UNIQUE (tenant_id, invited_by_principal_id, idempotency_key_sha256),
    FOREIGN KEY (role_id, population)
        REFERENCES atlas_identity.role_catalogue(role_id, population)
        DEFERRABLE INITIALLY IMMEDIATE,
    CHECK (tenant_id IS NOT NULL),
    CHECK (expires_at = created_at + interval '72 hours'),
    CHECK (
        (status = 'pending' AND terminal_at IS NULL)
        OR (status IN ('revoked', 'expired') AND terminal_at IS NOT NULL)
    ),
    CHECK (terminal_at IS NULL OR terminal_at >= created_at)
);

CREATE INDEX organization_invitations_tenant_status_idx
    ON atlas_identity.organization_invitations (tenant_id, status, expires_at, invitation_id);

CREATE INDEX organization_invitations_email_idx
    ON atlas_identity.organization_invitations (tenant_id, email_sha256, status);

INSERT INTO atlas_foundation.data_scope_registry
    (schema_name, table_name, scope_kind, tenant_column, global_scope_reason)
VALUES (
    'atlas_identity',
    'organization_invitations',
    'tenant',
    'tenant_id',
    NULL
);

REVOKE ALL ON atlas_identity.organization_invitations FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON atlas_identity.organization_invitations TO atlas_api;
