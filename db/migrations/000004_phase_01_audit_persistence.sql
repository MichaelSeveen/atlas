CREATE SCHEMA atlas_audit AUTHORIZATION atlas_migration;
REVOKE ALL ON SCHEMA atlas_audit FROM PUBLIC;
GRANT USAGE ON SCHEMA atlas_audit TO atlas_api;

CREATE TABLE atlas_audit.audit_events (
    audit_event_id text PRIMARY KEY CHECK (audit_event_id ~ '^[a-z]{2,8}_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'),
    actor_id text NOT NULL CHECK (actor_id ~ '^[a-z]{2,8}_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'),
    actor_type text NOT NULL CHECK (actor_type IN ('customer', 'merchant', 'workforce', 'machine', 'system')),
    tenant_id text REFERENCES atlas_identity.organizations(tenant_id),
    global_scope text CHECK (global_scope IN ('identity-security', 'platform-security')),
    session_assurance text NOT NULL CHECK (session_assurance IN ('none', 'baseline', 'phishing-resistant')),
    action text NOT NULL CHECK (action ~ '^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$'),
    target_type text NOT NULL CHECK (target_type ~ '^[a-z][a-z0-9_]{1,63}$'),
    target_id text NOT NULL CHECK (length(target_id) BETWEEN 1 AND 96),
    decision_id text NOT NULL CHECK (decision_id ~ '^[a-z]{2,8}_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'),
    decision text NOT NULL CHECK (decision IN ('allowed', 'denied', 'executed')),
    reason_code text NOT NULL CHECK (reason_code ~ '^[a-z][a-z0-9_]{2,63}$'),
    correlation_id text NOT NULL CHECK (correlation_id ~ '^[a-z]{2,8}_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'),
    approval_id text CHECK (approval_id IS NULL OR approval_id ~ '^[a-z]{2,8}_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'),
    occurred_at timestamptz NOT NULL,
    safe_before_reference text CHECK (safe_before_reference IS NULL OR length(safe_before_reference) BETWEEN 1 AND 128),
    safe_after_reference text CHECK (safe_after_reference IS NULL OR length(safe_after_reference) BETWEEN 1 AND 128),
    CHECK (
        (tenant_id IS NOT NULL AND global_scope IS NULL)
        OR (tenant_id IS NULL AND global_scope IS NOT NULL)
    )
);

CREATE INDEX audit_events_tenant_occurred_idx
    ON atlas_audit.audit_events (tenant_id, occurred_at, audit_event_id)
    WHERE tenant_id IS NOT NULL;

CREATE INDEX audit_events_global_occurred_idx
    ON atlas_audit.audit_events (global_scope, occurred_at, audit_event_id)
    WHERE global_scope IS NOT NULL;

INSERT INTO atlas_foundation.data_scope_registry
    (schema_name, table_name, scope_kind, tenant_column, global_scope_reason)
VALUES
    ('atlas_audit', 'audit_events', 'mixed', 'tenant_id', 'Security facts may concern a tenant or a closed platform and identity security scope.');

REVOKE ALL ON ALL TABLES IN SCHEMA atlas_audit FROM PUBLIC;
GRANT INSERT ON atlas_audit.audit_events TO atlas_api;

ALTER DEFAULT PRIVILEGES FOR ROLE atlas_migration IN SCHEMA atlas_audit REVOKE ALL ON TABLES FROM PUBLIC;
