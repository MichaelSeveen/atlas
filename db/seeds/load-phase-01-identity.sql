\set ON_ERROR_STOP on
\set QUIET on

CREATE TEMP TABLE atlas_phase01_seed_document (
    payload jsonb NOT NULL,
    seed_checksum character(64) NOT NULL
) ON COMMIT DROP;

INSERT INTO atlas_phase01_seed_document(payload, seed_checksum)
VALUES (:'seed_document'::jsonb, :'seed_checksum');

DO $seed_validation$
DECLARE
    document jsonb;
    document_checksum text;
BEGIN
    SELECT payload, seed_checksum INTO STRICT document, document_checksum
    FROM atlas_phase01_seed_document;
    IF document->>'seed_id' <> 'atlas-phase01-identity-v1'
       OR (document->>'schema_version')::integer <> 1
       OR document->>'virtual_time' <> '2026-07-26T00:00:00Z'
       OR document->>'policy_sha256' !~ '^[0-9a-f]{64}$'
       OR document_checksum !~ '^[0-9a-f]{64}$' THEN
        RAISE EXCEPTION 'Phase 01 seed identity or checksum is invalid';
    END IF;
END
$seed_validation$;

INSERT INTO atlas_identity.permission_catalogue(permission_id, policy_checksum)
SELECT permission_id, payload->>'policy_sha256'
FROM atlas_phase01_seed_document,
     jsonb_array_elements_text(payload->'permissions') AS permission(permission_id)
ON CONFLICT (permission_id) DO NOTHING;

INSERT INTO atlas_identity.role_catalogue(role_id, population, standing_status, policy_checksum)
SELECT
    role->>'id',
    role->>'population',
    COALESCE(role->>'standing_status', 'enabled'),
    payload->>'policy_sha256'
FROM atlas_phase01_seed_document,
     jsonb_array_elements(payload->'roles') AS roles(role)
ON CONFLICT (role_id) DO NOTHING;

INSERT INTO atlas_identity.role_permissions(role_id, permission_id)
SELECT role->>'id', permission_id
FROM atlas_phase01_seed_document,
     jsonb_array_elements(payload->'roles') AS roles(role),
     jsonb_array_elements_text(role->'permissions') AS permissions(permission_id)
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO atlas_identity.role_delegations(role_id, delegable_role_id)
SELECT role->>'id', delegable_role_id
FROM atlas_phase01_seed_document,
     jsonb_array_elements(payload->'roles') AS roles(role),
     jsonb_array_elements_text(role->'delegable_roles') AS delegations(delegable_role_id)
ON CONFLICT (role_id, delegable_role_id) DO NOTHING;

INSERT INTO atlas_identity.organizations (
    tenant_id,
    organization_type,
    display_name,
    normalized_name,
    confusable_skeleton,
    status,
    authorization_version,
    version,
    created_at,
    updated_at
)
SELECT
    organization.tenant_id,
    organization.organization_type,
    organization.display_name,
    organization.normalized_name,
    organization.confusable_skeleton,
    'active',
    1,
    1,
    seed.virtual_time,
    seed.virtual_time
FROM atlas_phase01_seed_document,
     jsonb_to_recordset(payload->'organizations') AS organization(
         tenant_id text,
         organization_type text,
         display_name text,
         normalized_name text,
         confusable_skeleton text
     ),
     LATERAL (SELECT (payload->>'virtual_time')::timestamptz AS virtual_time) AS seed
ON CONFLICT (tenant_id) DO NOTHING;

INSERT INTO atlas_identity.principals (
    principal_id,
    principal_type,
    display_name,
    person_anchor,
    status,
    authorization_version,
    version,
    created_at,
    updated_at
)
SELECT
    principal.principal_id,
    principal.principal_type,
    principal.display_name,
    principal.person_anchor,
    'active',
    1,
    1,
    seed.virtual_time,
    seed.virtual_time
FROM atlas_phase01_seed_document,
     jsonb_to_recordset(payload->'principals') AS principal(
         principal_id text,
         principal_type text,
         display_name text,
         person_anchor text
     ),
     LATERAL (SELECT (payload->>'virtual_time')::timestamptz AS virtual_time) AS seed
ON CONFLICT (principal_id) DO NOTHING;

INSERT INTO atlas_identity.external_subjects (
    external_subject_id,
    principal_id,
    population,
    issuer,
    subject,
    created_at
)
SELECT
    external_subject.external_subject_id,
    external_subject.principal_id,
    external_subject.population,
    external_subject.issuer,
    external_subject.subject,
    seed.virtual_time
FROM atlas_phase01_seed_document,
     jsonb_to_recordset(payload->'external_subjects') AS external_subject(
         external_subject_id text,
         principal_id text,
         population text,
         issuer text,
         subject text
     ),
     LATERAL (SELECT (payload->>'virtual_time')::timestamptz AS virtual_time) AS seed
ON CONFLICT (external_subject_id) DO NOTHING;

INSERT INTO atlas_identity.memberships (
    membership_id,
    tenant_id,
    principal_id,
    role_id,
    population,
    status,
    authorization_version,
    version,
    created_at,
    updated_at,
    revoked_at
)
SELECT
    membership.membership_id,
    membership.tenant_id,
    membership.principal_id,
    membership.role_id,
    principal.principal_type,
    'active',
    1,
    1,
    seed.virtual_time,
    seed.virtual_time,
    NULL
FROM atlas_phase01_seed_document,
     jsonb_to_recordset(payload->'memberships') AS membership(
         membership_id text,
         tenant_id text,
         principal_id text,
         role_id text
     ),
     LATERAL (
         SELECT principal_type
         FROM atlas_identity.principals
         WHERE principal_id = membership.principal_id
     ) AS principal,
     LATERAL (SELECT (payload->>'virtual_time')::timestamptz AS virtual_time) AS seed
ON CONFLICT (membership_id) DO NOTHING;

INSERT INTO atlas_identity.principal_roles (
    principal_role_id,
    principal_id,
    role_id,
    population,
    status,
    authorization_version,
    version,
    created_at,
    updated_at,
    revoked_at
)
SELECT
    principal_role.principal_role_id,
    principal_role.principal_id,
    principal_role.role_id,
    principal.principal_type,
    'active',
    1,
    1,
    seed.virtual_time,
    seed.virtual_time,
    NULL
FROM atlas_phase01_seed_document,
     jsonb_to_recordset(payload->'principal_roles') AS principal_role(
         principal_role_id text,
         principal_id text,
         role_id text
     ),
     LATERAL (
         SELECT principal_type
         FROM atlas_identity.principals
         WHERE principal_id = principal_role.principal_id
     ) AS principal,
     LATERAL (SELECT (payload->>'virtual_time')::timestamptz AS virtual_time) AS seed
ON CONFLICT (principal_role_id) DO NOTHING;

INSERT INTO atlas_identity.sessions (
    session_id,
    principal_id,
    population,
    tenant_id,
    global_scope,
    verifier_sha256,
    assurance,
    status,
    authorization_version,
    rotation_version,
    version,
    created_at,
    last_seen_at,
    idle_expires_at,
    absolute_expires_at,
    revoked_at
)
SELECT
    session.session_id,
    session.principal_id,
    principal.principal_type,
    session.tenant_id,
    session.global_scope,
    decode(session.verifier_sha256, 'hex'),
    session.assurance,
    session.status,
    1,
    1,
    1,
    seed.virtual_time,
    seed.virtual_time,
    session.idle_expires_at,
    session.absolute_expires_at,
    session.revoked_at
FROM atlas_phase01_seed_document,
     jsonb_to_recordset(payload->'sessions') AS session(
         session_id text,
         principal_id text,
         tenant_id text,
         global_scope text,
         verifier_sha256 text,
         assurance text,
         status text,
         idle_expires_at timestamptz,
         absolute_expires_at timestamptz,
         revoked_at timestamptz
     ),
     LATERAL (
         SELECT principal_type
         FROM atlas_identity.principals
         WHERE principal_id = session.principal_id
     ) AS principal,
     LATERAL (SELECT (payload->>'virtual_time')::timestamptz AS virtual_time) AS seed
ON CONFLICT (session_id) DO NOTHING;

INSERT INTO atlas_audit.audit_events (
    audit_event_id,
    actor_id,
    actor_type,
    tenant_id,
    global_scope,
    session_assurance,
    action,
    target_type,
    target_id,
    decision_id,
    decision,
    reason_code,
    correlation_id,
    approval_id,
    occurred_at,
    safe_before_reference,
    safe_after_reference
)
SELECT
    event.audit_event_id,
    event.actor_id,
    event.actor_type,
    event.tenant_id,
    event.global_scope,
    event.session_assurance,
    event.action,
    event.target_type,
    event.target_id,
    event.decision_id,
    event.decision,
    event.reason_code,
    event.correlation_id,
    event.approval_id,
    seed.virtual_time,
    event.safe_before_reference,
    event.safe_after_reference
FROM atlas_phase01_seed_document,
     jsonb_to_recordset(payload->'audit_events') AS event(
         audit_event_id text,
         actor_id text,
         actor_type text,
         tenant_id text,
         global_scope text,
         session_assurance text,
         action text,
         target_type text,
         target_id text,
         decision_id text,
         decision text,
         reason_code text,
         correlation_id text,
         approval_id text,
         safe_before_reference text,
         safe_after_reference text
     ),
     LATERAL (SELECT (payload->>'virtual_time')::timestamptz AS virtual_time) AS seed
ON CONFLICT (audit_event_id) DO NOTHING;

DO $seed_rows$
DECLARE
    document jsonb;
    expected_policy_checksum text;
    virtual_time timestamptz;
BEGIN
    SELECT payload, payload->>'policy_sha256', (payload->>'virtual_time')::timestamptz
    INTO STRICT document, expected_policy_checksum, virtual_time
    FROM atlas_phase01_seed_document;

    IF EXISTS (
        SELECT permission_id
        FROM jsonb_array_elements_text(document->'permissions') AS permissions(permission_id)
        EXCEPT
        SELECT permission_id
        FROM atlas_identity.permission_catalogue
        WHERE atlas_identity.permission_catalogue.policy_checksum = expected_policy_checksum
    ) THEN
        RAISE EXCEPTION 'Phase 01 permission seed drift detected';
    END IF;

    IF EXISTS (
        SELECT role->>'id', role->>'population', COALESCE(role->>'standing_status', 'enabled')
        FROM jsonb_array_elements(document->'roles') AS roles(role)
        EXCEPT
        SELECT role_id, population, standing_status
        FROM atlas_identity.role_catalogue
        WHERE atlas_identity.role_catalogue.policy_checksum = expected_policy_checksum
    ) THEN
        RAISE EXCEPTION 'Phase 01 role seed drift detected';
    END IF;

    IF EXISTS (
        SELECT role->>'id', permission_id
        FROM jsonb_array_elements(document->'roles') AS roles(role),
             jsonb_array_elements_text(role->'permissions') AS permissions(permission_id)
        EXCEPT
        SELECT role_id, permission_id FROM atlas_identity.role_permissions
    ) THEN
        RAISE EXCEPTION 'Phase 01 role-permission seed drift detected';
    END IF;

    IF EXISTS (
        SELECT role->>'id', delegable_role_id
        FROM jsonb_array_elements(document->'roles') AS roles(role),
             jsonb_array_elements_text(role->'delegable_roles') AS delegations(delegable_role_id)
        EXCEPT
        SELECT role_id, delegable_role_id FROM atlas_identity.role_delegations
    ) THEN
        RAISE EXCEPTION 'Phase 01 role-delegation seed drift detected';
    END IF;

    IF EXISTS (
        SELECT organization.tenant_id, organization.organization_type, organization.display_name, organization.normalized_name, organization.confusable_skeleton
        FROM jsonb_to_recordset(document->'organizations') AS organization(
            tenant_id text,
            organization_type text,
            display_name text,
            normalized_name text,
            confusable_skeleton text
        )
        EXCEPT
        SELECT tenant_id, organization_type, display_name, normalized_name, confusable_skeleton
        FROM atlas_identity.organizations
        WHERE status = 'active' AND authorization_version = 1 AND version = 1
          AND created_at = virtual_time AND updated_at = virtual_time
    ) THEN
        RAISE EXCEPTION 'Phase 01 organization seed drift detected';
    END IF;

    IF EXISTS (
        SELECT principal.principal_id, principal.principal_type, principal.display_name, principal.person_anchor
        FROM jsonb_to_recordset(document->'principals') AS principal(
            principal_id text,
            principal_type text,
            display_name text,
            person_anchor text
        )
        EXCEPT
        SELECT principal_id, principal_type, display_name, person_anchor
        FROM atlas_identity.principals
        WHERE status = 'active' AND authorization_version = 1 AND version = 1
          AND created_at = virtual_time AND updated_at = virtual_time
    ) THEN
        RAISE EXCEPTION 'Phase 01 principal seed drift detected';
    END IF;

    IF EXISTS (
        SELECT external_subject.external_subject_id, external_subject.principal_id, external_subject.population, external_subject.issuer, external_subject.subject
        FROM jsonb_to_recordset(document->'external_subjects') AS external_subject(
            external_subject_id text,
            principal_id text,
            population text,
            issuer text,
            subject text
        )
        EXCEPT
        SELECT external_subject_id, principal_id, population, issuer, subject
        FROM atlas_identity.external_subjects
        WHERE created_at = virtual_time
    ) THEN
        RAISE EXCEPTION 'Phase 01 external-subject seed drift detected';
    END IF;

    IF EXISTS (
        SELECT membership.membership_id, membership.tenant_id, membership.principal_id, membership.role_id
        FROM jsonb_to_recordset(document->'memberships') AS membership(
            membership_id text,
            tenant_id text,
            principal_id text,
            role_id text
        )
        EXCEPT
        SELECT membership_id, tenant_id, principal_id, role_id
        FROM atlas_identity.memberships
        WHERE status = 'active' AND authorization_version = 1 AND version = 1
          AND created_at = virtual_time AND updated_at = virtual_time AND revoked_at IS NULL
    ) THEN
        RAISE EXCEPTION 'Phase 01 membership seed drift detected';
    END IF;

    IF EXISTS (
        SELECT principal_role.principal_role_id, principal_role.principal_id, principal_role.role_id
        FROM jsonb_to_recordset(document->'principal_roles') AS principal_role(
            principal_role_id text,
            principal_id text,
            role_id text
        )
        EXCEPT
        SELECT principal_role_id, principal_id, role_id
        FROM atlas_identity.principal_roles
        WHERE status = 'active' AND authorization_version = 1 AND version = 1
          AND created_at = virtual_time AND updated_at = virtual_time AND revoked_at IS NULL
    ) THEN
        RAISE EXCEPTION 'Phase 01 principal-role seed drift detected';
    END IF;

    IF EXISTS (
        SELECT session.session_id, session.principal_id, session.tenant_id, session.global_scope,
               session.verifier_sha256, session.assurance, session.status,
               session.idle_expires_at, session.absolute_expires_at, session.revoked_at
        FROM jsonb_to_recordset(document->'sessions') AS session(
            session_id text,
            principal_id text,
            tenant_id text,
            global_scope text,
            verifier_sha256 text,
            assurance text,
            status text,
            idle_expires_at timestamptz,
            absolute_expires_at timestamptz,
            revoked_at timestamptz
        )
        EXCEPT
        SELECT session_id, principal_id, tenant_id, global_scope, encode(verifier_sha256, 'hex'),
               assurance, status, idle_expires_at, absolute_expires_at, revoked_at
        FROM atlas_identity.sessions
        WHERE authorization_version = 1 AND rotation_version = 1 AND version = 1
          AND created_at = virtual_time AND last_seen_at = virtual_time
    ) THEN
        RAISE EXCEPTION 'Phase 01 session seed drift detected';
    END IF;

    IF EXISTS (
        SELECT event.audit_event_id, event.actor_id, event.actor_type, event.tenant_id, event.global_scope,
               event.session_assurance, event.action, event.target_type, event.target_id,
               event.decision_id, event.decision, event.reason_code, event.correlation_id,
               event.approval_id, event.safe_before_reference, event.safe_after_reference
        FROM jsonb_to_recordset(document->'audit_events') AS event(
            audit_event_id text,
            actor_id text,
            actor_type text,
            tenant_id text,
            global_scope text,
            session_assurance text,
            action text,
            target_type text,
            target_id text,
            decision_id text,
            decision text,
            reason_code text,
            correlation_id text,
            approval_id text,
            safe_before_reference text,
            safe_after_reference text
        )
        EXCEPT
        SELECT audit_event_id, actor_id, actor_type, tenant_id, global_scope, session_assurance,
               action, target_type, target_id, decision_id, decision, reason_code, correlation_id,
               approval_id, safe_before_reference, safe_after_reference
        FROM atlas_audit.audit_events
        WHERE occurred_at = virtual_time
    ) THEN
        RAISE EXCEPTION 'Phase 01 audit seed drift detected';
    END IF;
END
$seed_rows$;

INSERT INTO atlas_foundation.seed_applications(seed_id, seed_checksum, policy_checksum)
SELECT payload->>'seed_id', seed_checksum, payload->>'policy_sha256'
FROM atlas_phase01_seed_document
ON CONFLICT (seed_id) DO NOTHING;

DO $seed_application$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM atlas_foundation.seed_applications
        WHERE seed_id = 'atlas-phase01-identity-v1'
          AND seed_checksum = (
              SELECT seed_checksum FROM atlas_phase01_seed_document
          )
          AND policy_checksum = (
              SELECT payload->>'policy_sha256' FROM atlas_phase01_seed_document
          )
    ) THEN
        RAISE EXCEPTION 'Phase 01 seed application manifest drift detected';
    END IF;
END
$seed_application$;

\set QUIET off
