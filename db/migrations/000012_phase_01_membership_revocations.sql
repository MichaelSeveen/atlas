CREATE TABLE atlas_identity.membership_revocations (
    membership_revocation_id text PRIMARY KEY CHECK (
        membership_revocation_id ~ '^mrv_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    tenant_id text NOT NULL,
    membership_id text NOT NULL,
    target_principal_id text NOT NULL,
    population text NOT NULL CHECK (population = 'merchant'),
    actor_principal_id text NOT NULL REFERENCES atlas_identity.principals(principal_id),
    actor_session_id text NOT NULL REFERENCES atlas_identity.sessions(session_id),
    idempotency_key_sha256 bytea NOT NULL CHECK (octet_length(idempotency_key_sha256) = 32),
    request_sha256 bytea NOT NULL CHECK (octet_length(request_sha256) = 32),
    expected_membership_version bigint NOT NULL CHECK (expected_membership_version > 0),
    target_role_id text NOT NULL CHECK (
        target_role_id IN ('merchant_viewer', 'merchant_operator')
    ),
    result_membership_version bigint NOT NULL CHECK (
        result_membership_version = expected_membership_version + 1
    ),
    membership_created_at timestamptz NOT NULL,
    revoked_at timestamptz NOT NULL,
    decision_id text NOT NULL CHECK (
        decision_id ~ '^dec_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    audit_event_id text NOT NULL CHECK (
        audit_event_id ~ '^aud_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    created_at timestamptz NOT NULL,
    UNIQUE (tenant_id, actor_principal_id, idempotency_key_sha256),
    FOREIGN KEY (membership_id, tenant_id)
        REFERENCES atlas_identity.memberships(membership_id, tenant_id),
    FOREIGN KEY (target_principal_id, population)
        REFERENCES atlas_identity.principals(principal_id, principal_type),
    FOREIGN KEY (target_role_id, population)
        REFERENCES atlas_identity.role_catalogue(role_id, population),
    CHECK (revoked_at = created_at)
);

CREATE INDEX membership_revocations_membership_created_idx
    ON atlas_identity.membership_revocations (tenant_id, membership_id, created_at);

INSERT INTO atlas_foundation.data_scope_registry
    (schema_name, table_name, scope_kind, tenant_column, global_scope_reason)
VALUES (
    'atlas_identity',
    'membership_revocations',
    'tenant',
    'tenant_id',
    NULL
);

REVOKE ALL ON atlas_identity.membership_revocations FROM PUBLIC;
GRANT SELECT, INSERT ON atlas_identity.membership_revocations TO atlas_api;
