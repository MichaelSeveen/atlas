ALTER TABLE atlas_identity.oidc_transactions
    ADD COLUMN invitation_id text REFERENCES atlas_identity.organization_invitations(invitation_id),
    ADD COLUMN invitation_token_sha256 bytea CHECK (
        invitation_token_sha256 IS NULL OR octet_length(invitation_token_sha256) = 32
    );

ALTER TABLE atlas_identity.oidc_transactions
    DROP CONSTRAINT oidc_transactions_transaction_kind_check,
    DROP CONSTRAINT oidc_transactions_check2;

ALTER TABLE atlas_identity.oidc_transactions
    ADD CONSTRAINT oidc_transactions_transaction_kind_check
    CHECK (transaction_kind IN ('login', 'step-up', 'invitation-acceptance')),
    ADD CONSTRAINT oidc_transactions_kind_binding_check
    CHECK (
        (
            transaction_kind = 'login'
            AND principal_id IS NULL
            AND requested_action IS NULL
            AND invitation_id IS NULL
            AND invitation_token_sha256 IS NULL
        )
        OR (
            transaction_kind = 'step-up'
            AND principal_id IS NOT NULL
            AND replaced_session_id IS NOT NULL
            AND requested_action IS NOT NULL
            AND invitation_id IS NULL
            AND invitation_token_sha256 IS NULL
        )
        OR (
            transaction_kind = 'invitation-acceptance'
            AND population = 'merchant'
            AND principal_id IS NULL
            AND requested_action IS NULL
            AND invitation_id IS NOT NULL
            AND invitation_token_sha256 IS NOT NULL
        )
    );

ALTER TABLE atlas_identity.sessions
    ADD COLUMN invitation_id text REFERENCES atlas_identity.organization_invitations(invitation_id),
    ADD COLUMN verified_email_sha256 bytea CHECK (
        verified_email_sha256 IS NULL OR octet_length(verified_email_sha256) = 32
    );

ALTER TABLE atlas_identity.sessions
    DROP CONSTRAINT sessions_global_scope_check,
    DROP CONSTRAINT sessions_check;

ALTER TABLE atlas_identity.sessions
    ADD CONSTRAINT sessions_global_scope_check
    CHECK (global_scope IN ('workforce', 'machine', 'invitation-acceptance')),
    ADD CONSTRAINT sessions_scope_binding_check
    CHECK (
        (
            population IN ('customer', 'merchant')
            AND tenant_id IS NOT NULL
            AND global_scope IS NULL
            AND invitation_id IS NULL
        )
        OR (
            population = 'merchant'
            AND tenant_id IS NULL
            AND global_scope = 'invitation-acceptance'
            AND invitation_id IS NOT NULL
            AND verified_email_sha256 IS NOT NULL
        )
        OR (
            population = 'workforce'
            AND tenant_id IS NULL
            AND global_scope = 'workforce'
            AND invitation_id IS NULL
        )
        OR (
            population = 'machine'
            AND tenant_id IS NULL
            AND global_scope = 'machine'
            AND invitation_id IS NULL
        )
    );

ALTER TABLE atlas_identity.organization_invitations
    ADD COLUMN accepted_by_principal_id text REFERENCES atlas_identity.principals(principal_id),
    ADD COLUMN accepted_membership_id text REFERENCES atlas_identity.memberships(membership_id),
    ADD COLUMN acceptance_idempotency_key_sha256 bytea CHECK (
        acceptance_idempotency_key_sha256 IS NULL
        OR octet_length(acceptance_idempotency_key_sha256) = 32
    ),
    ADD COLUMN acceptance_request_sha256 bytea CHECK (
        acceptance_request_sha256 IS NULL OR octet_length(acceptance_request_sha256) = 32
    ),
    ADD COLUMN acceptance_decision_id text CHECK (
        acceptance_decision_id IS NULL
        OR acceptance_decision_id ~ '^dec_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
    );

ALTER TABLE atlas_identity.organization_invitations
    DROP CONSTRAINT organization_invitations_status_check,
    DROP CONSTRAINT organization_invitations_check1;

ALTER TABLE atlas_identity.organization_invitations
    ADD CONSTRAINT organization_invitations_status_check
    CHECK (status IN ('pending', 'accepted', 'revoked', 'expired')),
    ADD CONSTRAINT organization_invitations_terminal_binding_check
    CHECK (
        (
            status = 'pending'
            AND terminal_at IS NULL
            AND accepted_by_principal_id IS NULL
            AND accepted_membership_id IS NULL
            AND acceptance_idempotency_key_sha256 IS NULL
            AND acceptance_request_sha256 IS NULL
            AND acceptance_decision_id IS NULL
        )
        OR (
            status = 'accepted'
            AND terminal_at IS NOT NULL
            AND accepted_by_principal_id IS NOT NULL
            AND accepted_membership_id IS NOT NULL
            AND acceptance_idempotency_key_sha256 IS NOT NULL
            AND acceptance_request_sha256 IS NOT NULL
            AND acceptance_decision_id IS NOT NULL
        )
        OR (
            status IN ('revoked', 'expired')
            AND terminal_at IS NOT NULL
            AND accepted_by_principal_id IS NULL
            AND accepted_membership_id IS NULL
            AND acceptance_idempotency_key_sha256 IS NULL
            AND acceptance_request_sha256 IS NULL
            AND acceptance_decision_id IS NULL
        )
    );

UPDATE atlas_foundation.data_scope_registry
SET global_scope_reason =
    'Workforce and machine sessions are globally scoped; invitation-acceptance sessions grant no tenant authority until atomic membership creation and session rotation.'
WHERE schema_name = 'atlas_identity' AND table_name = 'sessions';
