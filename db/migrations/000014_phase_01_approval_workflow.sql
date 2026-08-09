CREATE SCHEMA atlas_operations AUTHORIZATION atlas_migration;
REVOKE ALL ON SCHEMA atlas_operations FROM PUBLIC;
GRANT USAGE ON SCHEMA atlas_operations TO atlas_api;

CREATE TABLE atlas_operations.approvals (
    approval_id text PRIMARY KEY CHECK (
        approval_id ~ '^apr_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    tenant_id text NOT NULL REFERENCES atlas_identity.organizations(tenant_id),
    action_type text NOT NULL CHECK (
        action_type = 'identity.organization.membership.change_admin'
    ),
    target_type text NOT NULL CHECK (target_type = 'membership'),
    target_id text NOT NULL,
    target_version bigint NOT NULL CHECK (target_version > 0),
    payload_canonical bytea NOT NULL CHECK (
        octet_length(payload_canonical) BETWEEN 64 AND 2048
    ),
    payload_sha256 bytea NOT NULL CHECK (octet_length(payload_sha256) = 32),
    payload_schema_version bigint NOT NULL CHECK (payload_schema_version = 1),
    payload_canonicalization text NOT NULL CHECK (payload_canonicalization = 'rfc8785'),
    payload_hash_algorithm text NOT NULL CHECK (payload_hash_algorithm = 'sha256'),
    requester_principal_id text NOT NULL REFERENCES atlas_identity.principals(principal_id),
    requester_session_id text NOT NULL REFERENCES atlas_identity.sessions(session_id),
    eligible_checker_policy text NOT NULL CHECK (
        eligible_checker_policy = 'dynamic-at-decision-and-execution'
    ),
    purpose text NOT NULL CHECK (purpose = 'organization_administration'),
    status text NOT NULL CHECK (
        status IN (
            'pending', 'approved', 'rejected', 'cancelled', 'expired',
            'executed', 'execution_failed', 'superseded'
        )
    ),
    decision_reason text CHECK (decision_reason IS NULL OR length(decision_reason) <= 500),
    decider_principal_id text REFERENCES atlas_identity.principals(principal_id),
    decider_session_id text REFERENCES atlas_identity.sessions(session_id),
    version bigint NOT NULL CHECK (version > 0),
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL,
    decided_at timestamptz,
    executed_at timestamptz,
    updated_at timestamptz NOT NULL,
    UNIQUE (approval_id, tenant_id),
    FOREIGN KEY (target_id, tenant_id)
        REFERENCES atlas_identity.memberships(membership_id, tenant_id),
    CHECK (expires_at > created_at),
    CHECK (updated_at >= created_at),
    CHECK (
        (decider_principal_id IS NULL AND decider_session_id IS NULL AND decided_at IS NULL)
        OR
        (decider_principal_id IS NOT NULL AND decider_session_id IS NOT NULL AND decided_at IS NOT NULL)
    ),
    CHECK (
        status NOT IN ('approved', 'rejected', 'executed', 'execution_failed')
        OR decider_principal_id IS NOT NULL
    ),
    CHECK (status <> 'pending' OR decider_principal_id IS NULL),
    CHECK ((status = 'executed') = (executed_at IS NOT NULL)),
    CHECK (decider_principal_id IS NULL OR decider_principal_id <> requester_principal_id)
);

CREATE INDEX approvals_tenant_created_idx
    ON atlas_operations.approvals (tenant_id, created_at DESC, approval_id DESC);
CREATE INDEX approvals_pending_expiry_idx
    ON atlas_operations.approvals (expires_at, approval_id)
    WHERE status IN ('pending', 'approved', 'execution_failed');

CREATE TABLE atlas_operations.approval_requests (
    approval_id text PRIMARY KEY,
    tenant_id text NOT NULL,
    requester_principal_id text NOT NULL REFERENCES atlas_identity.principals(principal_id),
    idempotency_key_sha256 bytea NOT NULL CHECK (octet_length(idempotency_key_sha256) = 32),
    request_sha256 bytea NOT NULL CHECK (octet_length(request_sha256) = 32),
    authorization_decision_id text NOT NULL CHECK (
        authorization_decision_id ~ '^dec_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    audit_event_id text NOT NULL CHECK (
        audit_event_id ~ '^aud_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    created_at timestamptz NOT NULL,
    UNIQUE (tenant_id, requester_principal_id, idempotency_key_sha256),
    FOREIGN KEY (approval_id, tenant_id)
        REFERENCES atlas_operations.approvals(approval_id, tenant_id)
);

CREATE TABLE atlas_operations.approval_decisions (
    decision_record_id text PRIMARY KEY CHECK (
        decision_record_id ~ '^apd_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    approval_id text NOT NULL,
    tenant_id text NOT NULL,
    decider_principal_id text NOT NULL REFERENCES atlas_identity.principals(principal_id),
    decider_session_id text NOT NULL REFERENCES atlas_identity.sessions(session_id),
    idempotency_key_sha256 bytea NOT NULL CHECK (octet_length(idempotency_key_sha256) = 32),
    request_sha256 bytea NOT NULL CHECK (octet_length(request_sha256) = 32),
    decision text NOT NULL CHECK (decision IN ('approve', 'reject')),
    reason text CHECK (reason IS NULL OR length(reason) <= 500),
    result_approval_version bigint NOT NULL CHECK (result_approval_version > 1),
    authorization_decision_id text NOT NULL CHECK (
        authorization_decision_id ~ '^dec_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    audit_event_id text NOT NULL CHECK (
        audit_event_id ~ '^aud_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    created_at timestamptz NOT NULL,
    UNIQUE (approval_id),
    UNIQUE (tenant_id, decider_principal_id, idempotency_key_sha256),
    FOREIGN KEY (approval_id, tenant_id)
        REFERENCES atlas_operations.approvals(approval_id, tenant_id)
);

CREATE TABLE atlas_operations.approval_executions (
    execution_record_id text PRIMARY KEY CHECK (
        execution_record_id ~ '^aex_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    approval_id text NOT NULL,
    tenant_id text NOT NULL,
    executor_principal_id text NOT NULL REFERENCES atlas_identity.principals(principal_id),
    executor_session_id text NOT NULL REFERENCES atlas_identity.sessions(session_id),
    idempotency_key_sha256 bytea NOT NULL CHECK (octet_length(idempotency_key_sha256) = 32),
    request_sha256 bytea NOT NULL CHECK (octet_length(request_sha256) = 32),
    outcome text NOT NULL CHECK (outcome IN ('executed', 'execution_failed')),
    failure_reason_code text CHECK (
        failure_reason_code IS NULL OR failure_reason_code ~ '^[a-z][a-z0-9_]{2,63}$'
    ),
    result_approval_version bigint NOT NULL CHECK (result_approval_version > 2),
    authorization_decision_id text NOT NULL CHECK (
        authorization_decision_id ~ '^dec_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    audit_event_id text NOT NULL CHECK (
        audit_event_id ~ '^aud_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    target_decision_id text CHECK (
        target_decision_id IS NULL OR target_decision_id ~ '^dec_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    target_audit_event_id text CHECK (
        target_audit_event_id IS NULL OR target_audit_event_id ~ '^aud_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    created_at timestamptz NOT NULL,
    UNIQUE (tenant_id, executor_principal_id, idempotency_key_sha256),
    FOREIGN KEY (approval_id, tenant_id)
        REFERENCES atlas_operations.approvals(approval_id, tenant_id),
    CHECK (
        (outcome = 'executed' AND failure_reason_code IS NULL
            AND target_decision_id IS NOT NULL AND target_audit_event_id IS NOT NULL)
        OR
        (outcome = 'execution_failed' AND failure_reason_code IS NOT NULL
            AND target_decision_id IS NULL AND target_audit_event_id IS NULL)
    )
);

CREATE UNIQUE INDEX approval_executions_one_success_idx
    ON atlas_operations.approval_executions (approval_id)
    WHERE outcome = 'executed';

CREATE TABLE atlas_operations.approval_cancellations (
    cancellation_record_id text PRIMARY KEY CHECK (
        cancellation_record_id ~ '^apc_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    approval_id text NOT NULL UNIQUE,
    tenant_id text NOT NULL,
    actor_principal_id text NOT NULL REFERENCES atlas_identity.principals(principal_id),
    actor_session_id text NOT NULL REFERENCES atlas_identity.sessions(session_id),
    idempotency_key_sha256 bytea NOT NULL CHECK (octet_length(idempotency_key_sha256) = 32),
    request_sha256 bytea NOT NULL CHECK (octet_length(request_sha256) = 32),
    reason text NOT NULL CHECK (length(reason) BETWEEN 1 AND 500),
    result_approval_version bigint NOT NULL CHECK (result_approval_version > 1),
    authorization_decision_id text NOT NULL CHECK (
        authorization_decision_id ~ '^dec_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    audit_event_id text NOT NULL CHECK (
        audit_event_id ~ '^aud_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    ),
    created_at timestamptz NOT NULL,
    UNIQUE (tenant_id, actor_principal_id, idempotency_key_sha256),
    FOREIGN KEY (approval_id, tenant_id)
        REFERENCES atlas_operations.approvals(approval_id, tenant_id)
);

ALTER TABLE atlas_identity.membership_role_changes
    DROP CONSTRAINT membership_role_changes_before_role_id_check,
    DROP CONSTRAINT membership_role_changes_after_role_id_check,
    ADD COLUMN approval_id text,
    ADD CONSTRAINT membership_role_changes_before_role_id_check CHECK (
        before_role_id IN ('merchant_viewer', 'merchant_operator', 'merchant_admin')
    ),
    ADD CONSTRAINT membership_role_changes_after_role_id_check CHECK (
        after_role_id IN ('merchant_viewer', 'merchant_operator', 'merchant_admin')
    ),
    ADD CONSTRAINT membership_role_changes_approval_policy_check CHECK (
        (
            approval_id IS NULL
            AND before_role_id IN ('merchant_viewer', 'merchant_operator')
            AND after_role_id IN ('merchant_viewer', 'merchant_operator')
        )
        OR
        (
            approval_id IS NOT NULL
            AND (before_role_id = 'merchant_admin' OR after_role_id = 'merchant_admin')
        )
    ),
    ADD CONSTRAINT membership_role_changes_approval_fk FOREIGN KEY (approval_id)
        REFERENCES atlas_operations.approvals(approval_id);

CREATE UNIQUE INDEX membership_role_changes_one_approval_idx
    ON atlas_identity.membership_role_changes (approval_id)
    WHERE approval_id IS NOT NULL;

INSERT INTO atlas_foundation.data_scope_registry
    (schema_name, table_name, scope_kind, tenant_column, global_scope_reason)
VALUES
    ('atlas_operations', 'approvals', 'tenant', 'tenant_id', NULL),
    ('atlas_operations', 'approval_requests', 'tenant', 'tenant_id', NULL),
    ('atlas_operations', 'approval_decisions', 'tenant', 'tenant_id', NULL),
    ('atlas_operations', 'approval_executions', 'tenant', 'tenant_id', NULL),
    ('atlas_operations', 'approval_cancellations', 'tenant', 'tenant_id', NULL);

REVOKE ALL ON ALL TABLES IN SCHEMA atlas_operations FROM PUBLIC;
GRANT SELECT, INSERT ON
    atlas_operations.approvals,
    atlas_operations.approval_requests,
    atlas_operations.approval_decisions,
    atlas_operations.approval_executions,
    atlas_operations.approval_cancellations
TO atlas_api;
GRANT UPDATE (
    status, decision_reason, decider_principal_id, decider_session_id,
    version, decided_at, executed_at, updated_at
) ON atlas_operations.approvals TO atlas_api;

ALTER DEFAULT PRIVILEGES FOR ROLE atlas_migration IN SCHEMA atlas_operations
    REVOKE ALL ON TABLES FROM PUBLIC;
