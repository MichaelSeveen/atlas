CREATE TABLE atlas_foundation.data_scope_registry (
    schema_name text NOT NULL CHECK (schema_name ~ '^atlas_[a-z][a-z0-9_]{2,63}$'),
    table_name text NOT NULL CHECK (table_name ~ '^[a-z][a-z0-9_]{2,63}$'),
    scope_kind text NOT NULL CHECK (scope_kind IN ('global', 'tenant', 'mixed')),
    tenant_column text CHECK (tenant_column IS NULL OR tenant_column ~ '^[a-z][a-z0-9_]{2,63}$'),
    global_scope_reason text CHECK (global_scope_reason IS NULL OR length(global_scope_reason) BETWEEN 16 AND 512),
    PRIMARY KEY (schema_name, table_name),
    CHECK (
        (scope_kind = 'global' AND tenant_column IS NULL AND global_scope_reason IS NOT NULL)
        OR (scope_kind = 'tenant' AND tenant_column IS NOT NULL AND global_scope_reason IS NULL)
        OR (scope_kind = 'mixed' AND tenant_column IS NOT NULL AND global_scope_reason IS NOT NULL)
    )
);

CREATE TABLE atlas_foundation.seed_applications (
    seed_id text PRIMARY KEY CHECK (seed_id ~ '^atlas-phase[0-9]{2}-[a-z0-9-]{3,80}-v[0-9]+$'),
    seed_checksum character(64) NOT NULL CHECK (seed_checksum ~ '^[0-9a-f]{64}$'),
    policy_checksum character(64) NOT NULL CHECK (policy_checksum ~ '^[0-9a-f]{64}$'),
    applied_at timestamptz NOT NULL DEFAULT transaction_timestamp()
);

REVOKE INSERT, UPDATE, DELETE ON atlas_foundation.data_scope_registry FROM atlas_api, atlas_worker;
REVOKE INSERT, UPDATE, DELETE ON atlas_foundation.seed_applications FROM atlas_api, atlas_worker;

CREATE SCHEMA atlas_identity AUTHORIZATION atlas_migration;
REVOKE ALL ON SCHEMA atlas_identity FROM PUBLIC;
GRANT USAGE ON SCHEMA atlas_identity TO atlas_api;

CREATE TABLE atlas_identity.principals (
    principal_id text PRIMARY KEY CHECK (principal_id ~ '^[a-z]{2,8}_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'),
    principal_type text NOT NULL CHECK (principal_type IN ('customer', 'merchant', 'workforce', 'machine')),
    display_name text NOT NULL CHECK (length(display_name) BETWEEN 1 AND 160),
    person_anchor text UNIQUE CHECK (person_anchor IS NULL OR person_anchor ~ '^syn_person_[a-z0-9_]{3,80}$'),
    status text NOT NULL CHECK (status IN ('active', 'disabled')),
    authorization_version bigint NOT NULL CHECK (authorization_version > 0),
    version bigint NOT NULL CHECK (version > 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (principal_id, principal_type),
    CHECK (
        (principal_type IN ('merchant', 'workforce') AND person_anchor IS NOT NULL)
        OR (principal_type IN ('customer', 'machine') AND person_anchor IS NULL)
    ),
    CHECK (updated_at >= created_at)
);

CREATE TABLE atlas_identity.external_subjects (
    external_subject_id text PRIMARY KEY CHECK (external_subject_id ~ '^[a-z]{2,8}_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'),
    principal_id text NOT NULL,
    population text NOT NULL CHECK (population IN ('customer', 'merchant', 'workforce', 'machine')),
    issuer text NOT NULL CHECK (length(issuer) BETWEEN 8 AND 512),
    subject text NOT NULL CHECK (length(subject) BETWEEN 1 AND 255),
    created_at timestamptz NOT NULL,
    UNIQUE (population, issuer, subject),
    FOREIGN KEY (principal_id, population)
        REFERENCES atlas_identity.principals(principal_id, principal_type)
);

CREATE TABLE atlas_identity.permission_catalogue (
    permission_id text PRIMARY KEY CHECK (permission_id ~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$'),
    policy_checksum character(64) NOT NULL CHECK (policy_checksum ~ '^[0-9a-f]{64}$')
);

CREATE TABLE atlas_identity.role_catalogue (
    role_id text PRIMARY KEY CHECK (role_id ~ '^[a-z][a-z0-9_]{2,63}$'),
    population text NOT NULL CHECK (population IN ('customer', 'merchant', 'workforce', 'machine')),
    standing_status text NOT NULL CHECK (standing_status IN ('enabled', 'disabled')),
    policy_checksum character(64) NOT NULL CHECK (policy_checksum ~ '^[0-9a-f]{64}$'),
    UNIQUE (role_id, population)
);

CREATE TABLE atlas_identity.role_permissions (
    role_id text NOT NULL REFERENCES atlas_identity.role_catalogue(role_id),
    permission_id text NOT NULL REFERENCES atlas_identity.permission_catalogue(permission_id),
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE atlas_identity.role_delegations (
    role_id text NOT NULL REFERENCES atlas_identity.role_catalogue(role_id),
    delegable_role_id text NOT NULL REFERENCES atlas_identity.role_catalogue(role_id),
    PRIMARY KEY (role_id, delegable_role_id),
    CHECK (role_id <> delegable_role_id)
);

CREATE TABLE atlas_identity.organizations (
    tenant_id text PRIMARY KEY CHECK (tenant_id ~ '^[a-z]{2,8}_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'),
    organization_type text NOT NULL CHECK (organization_type IN ('customer', 'merchant')),
    display_name text NOT NULL CHECK (length(display_name) BETWEEN 1 AND 160),
    normalized_name text NOT NULL CHECK (length(normalized_name) BETWEEN 1 AND 160),
    confusable_skeleton text NOT NULL CHECK (length(confusable_skeleton) BETWEEN 1 AND 160),
    status text NOT NULL CHECK (status IN ('active', 'disabled')),
    authorization_version bigint NOT NULL CHECK (authorization_version > 0),
    version bigint NOT NULL CHECK (version > 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (normalized_name),
    UNIQUE (confusable_skeleton),
    UNIQUE (tenant_id, organization_type),
    CHECK (updated_at >= created_at)
);

CREATE TABLE atlas_identity.memberships (
    membership_id text PRIMARY KEY CHECK (membership_id ~ '^[a-z]{2,8}_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'),
    tenant_id text NOT NULL,
    principal_id text NOT NULL,
    role_id text NOT NULL,
    population text NOT NULL CHECK (population IN ('customer', 'merchant')),
    status text NOT NULL CHECK (status IN ('active', 'revoked')),
    authorization_version bigint NOT NULL CHECK (authorization_version > 0),
    version bigint NOT NULL CHECK (version > 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    revoked_at timestamptz,
    UNIQUE (tenant_id, principal_id),
    FOREIGN KEY (tenant_id, population)
        REFERENCES atlas_identity.organizations(tenant_id, organization_type),
    FOREIGN KEY (principal_id, population)
        REFERENCES atlas_identity.principals(principal_id, principal_type),
    FOREIGN KEY (role_id, population)
        REFERENCES atlas_identity.role_catalogue(role_id, population),
    CHECK (
        (status = 'active' AND revoked_at IS NULL)
        OR (status = 'revoked' AND revoked_at IS NOT NULL)
    ),
    CHECK (updated_at >= created_at),
    CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);

CREATE INDEX memberships_tenant_principal_idx
    ON atlas_identity.memberships (tenant_id, principal_id, status);

CREATE TABLE atlas_identity.principal_roles (
    principal_role_id text PRIMARY KEY CHECK (principal_role_id ~ '^[a-z]{2,8}_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'),
    principal_id text NOT NULL,
    role_id text NOT NULL,
    population text NOT NULL CHECK (population IN ('workforce', 'machine')),
    status text NOT NULL CHECK (status IN ('active', 'revoked')),
    authorization_version bigint NOT NULL CHECK (authorization_version > 0),
    version bigint NOT NULL CHECK (version > 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    revoked_at timestamptz,
    UNIQUE (principal_id, role_id),
    FOREIGN KEY (principal_id, population)
        REFERENCES atlas_identity.principals(principal_id, principal_type),
    FOREIGN KEY (role_id, population)
        REFERENCES atlas_identity.role_catalogue(role_id, population),
    CHECK (
        (status = 'active' AND revoked_at IS NULL)
        OR (status = 'revoked' AND revoked_at IS NOT NULL)
    ),
    CHECK (updated_at >= created_at),
    CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);

CREATE TABLE atlas_identity.sessions (
    session_id text PRIMARY KEY CHECK (session_id ~ '^[a-z]{2,8}_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'),
    principal_id text NOT NULL,
    population text NOT NULL CHECK (population IN ('customer', 'merchant', 'workforce', 'machine')),
    tenant_id text REFERENCES atlas_identity.organizations(tenant_id),
    global_scope text CHECK (global_scope IN ('workforce', 'machine')),
    verifier_sha256 bytea NOT NULL CHECK (octet_length(verifier_sha256) = 32),
    assurance text NOT NULL CHECK (assurance IN ('baseline', 'phishing-resistant')),
    status text NOT NULL CHECK (status IN ('active', 'revoked', 'expired')),
    authorization_version bigint NOT NULL CHECK (authorization_version > 0),
    rotation_version bigint NOT NULL CHECK (rotation_version > 0),
    version bigint NOT NULL CHECK (version > 0),
    created_at timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL,
    idle_expires_at timestamptz NOT NULL,
    absolute_expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    UNIQUE (verifier_sha256),
    FOREIGN KEY (principal_id, population)
        REFERENCES atlas_identity.principals(principal_id, principal_type),
    FOREIGN KEY (tenant_id, population)
        REFERENCES atlas_identity.organizations(tenant_id, organization_type),
    CHECK (
        (population IN ('customer', 'merchant') AND tenant_id IS NOT NULL AND global_scope IS NULL)
        OR (population = 'workforce' AND tenant_id IS NULL AND global_scope = 'workforce')
        OR (population = 'machine' AND tenant_id IS NULL AND global_scope = 'machine')
    ),
    CHECK (last_seen_at >= created_at),
    CHECK (idle_expires_at >= last_seen_at),
    CHECK (absolute_expires_at >= idle_expires_at),
    CHECK (
        (status = 'revoked' AND revoked_at IS NOT NULL)
        OR (status IN ('active', 'expired') AND revoked_at IS NULL)
    )
);

CREATE INDEX sessions_principal_status_idx
    ON atlas_identity.sessions (principal_id, status, last_seen_at);

INSERT INTO atlas_foundation.data_scope_registry
    (schema_name, table_name, scope_kind, tenant_column, global_scope_reason)
VALUES
    ('atlas_identity', 'principals', 'global', NULL, 'Atlas principals span identity populations and tenant memberships.'),
    ('atlas_identity', 'external_subjects', 'global', NULL, 'External subjects bind one identity population to one Atlas principal.'),
    ('atlas_identity', 'permission_catalogue', 'global', NULL, 'The source-controlled permission vocabulary applies across tenants.'),
    ('atlas_identity', 'role_catalogue', 'global', NULL, 'The source-controlled role vocabulary applies across tenants.'),
    ('atlas_identity', 'role_permissions', 'global', NULL, 'Role-to-permission policy bindings apply across tenants.'),
    ('atlas_identity', 'role_delegations', 'global', NULL, 'Role delegation policy bindings apply across tenants.'),
    ('atlas_identity', 'organizations', 'tenant', 'tenant_id', NULL),
    ('atlas_identity', 'memberships', 'tenant', 'tenant_id', NULL),
    ('atlas_identity', 'principal_roles', 'global', NULL, 'Workforce and machine authorization assignments are outside customer and merchant tenancy.'),
    ('atlas_identity', 'sessions', 'mixed', 'tenant_id', 'Workforce and machine sessions are globally scoped; customer and merchant sessions require tenant_id.');

REVOKE ALL ON ALL TABLES IN SCHEMA atlas_identity FROM PUBLIC;
GRANT SELECT ON
    atlas_identity.principals,
    atlas_identity.external_subjects,
    atlas_identity.permission_catalogue,
    atlas_identity.role_catalogue,
    atlas_identity.role_permissions,
    atlas_identity.role_delegations,
    atlas_identity.organizations,
    atlas_identity.memberships,
    atlas_identity.principal_roles,
    atlas_identity.sessions
TO atlas_api;
GRANT INSERT ON atlas_identity.principals, atlas_identity.external_subjects TO atlas_api;
GRANT INSERT, UPDATE ON atlas_identity.organizations, atlas_identity.memberships, atlas_identity.principal_roles, atlas_identity.sessions TO atlas_api;

ALTER DEFAULT PRIVILEGES FOR ROLE atlas_migration IN SCHEMA atlas_identity REVOKE ALL ON TABLES FROM PUBLIC;
