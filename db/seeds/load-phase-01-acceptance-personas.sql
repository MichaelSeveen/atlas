\set ON_ERROR_STOP on
\set QUIET on

CREATE TEMP TABLE atlas_phase01_acceptance_personas_document (
    payload jsonb NOT NULL,
    seed_checksum character(64) NOT NULL
) ON COMMIT DROP;

INSERT INTO atlas_phase01_acceptance_personas_document(payload, seed_checksum)
VALUES (:'seed_document'::jsonb, :'seed_checksum');

DO $seed_validation$
DECLARE
    document jsonb;
    document_checksum text;
BEGIN
    SELECT payload, seed_checksum INTO STRICT document, document_checksum
    FROM atlas_phase01_acceptance_personas_document;
    IF document->>'seed_id' <> 'atlas-phase01-acceptance-personas-v5'
       OR document->>'predecessor_seed_id' <> 'atlas-phase01-identity-policy-v4'
       OR (document->>'schema_version')::integer <> 1
       OR document->>'virtual_time' <> '2026-08-09T00:00:00Z'
       OR document->>'policy_sha256' <> '9b7c91b9bc78abdcfa15c9f33ba9ee40476d12da8d6c8c991e3255eaf79b3bd2'
       OR jsonb_array_length(document->'principals') <> 3
       OR document_checksum !~ '^[0-9a-f]{64}$' THEN
        RAISE EXCEPTION 'Phase 01 acceptance persona seed identity or checksum is invalid';
    END IF;
    IF NOT EXISTS (
        SELECT 1
        FROM atlas_foundation.seed_applications
        WHERE seed_id = document->>'predecessor_seed_id'
          AND seed_checksum = 'ab22049e74bf5eef30954d28429cbd0c2758b757afcb34a50e4c3f9fb7070068'
          AND policy_checksum = document->>'policy_sha256'
    ) THEN
        RAISE EXCEPTION 'Phase 01 acceptance persona predecessor state is invalid';
    END IF;
END
$seed_validation$;

CREATE TEMP TABLE atlas_phase01_acceptance_personas ON COMMIT DROP AS
SELECT
    principal.principal_id,
    principal.display_name,
    principal.person_anchor,
    principal.role_id,
    principal.username,
    principal.external_subject_id,
    principal.subject,
    (document.payload->>'virtual_time')::timestamptz AS virtual_time
FROM atlas_phase01_acceptance_personas_document AS document,
     jsonb_to_recordset(document.payload->'principals') AS principal(
         principal_id text,
         display_name text,
         person_anchor text,
         role_id text,
         username text,
         external_subject_id text,
         subject text
     );

DO $closed_persona_validation$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM atlas_phase01_acceptance_personas
        WHERE principal_id !~ '^usr_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
           OR external_subject_id !~ '^ext_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{20,32}$'
           OR person_anchor !~ '^syn_person_[a-z0-9_]{3,80}$'
           OR role_id NOT IN ('support', 'risk_analyst', 'finance_operator')
           OR username !~ '^synthetic-(support-analyst|risk-analyst|finance-operator)$'
           OR subject !~ '^[0-9a-f-]{36}$'
    ) OR (SELECT count(DISTINCT role_id) FROM atlas_phase01_acceptance_personas) <> 3
       OR (SELECT count(DISTINCT principal_id) FROM atlas_phase01_acceptance_personas) <> 3
       OR (SELECT count(DISTINCT subject) FROM atlas_phase01_acceptance_personas) <> 3 THEN
        RAISE EXCEPTION 'Phase 01 acceptance persona set is not closed and distinct';
    END IF;
END
$closed_persona_validation$;

INSERT INTO atlas_identity.principals (
    principal_id, principal_type, display_name, person_anchor, status,
    authorization_version, version, created_at, updated_at
)
SELECT principal_id, 'workforce', display_name, person_anchor, 'active', 1, 1,
       virtual_time, virtual_time
FROM atlas_phase01_acceptance_personas
ON CONFLICT (principal_id) DO NOTHING;

INSERT INTO atlas_identity.external_subjects (
    external_subject_id, principal_id, population, issuer, subject, created_at
)
SELECT external_subject_id, principal_id, 'workforce',
       'http://keycloak:8080/realms/atlas-workforce-local', subject, virtual_time
FROM atlas_phase01_acceptance_personas
ON CONFLICT (external_subject_id) DO NOTHING;

INSERT INTO atlas_identity.principal_roles (
    principal_role_id, principal_id, role_id, population, status,
    authorization_version, version, created_at, updated_at, revoked_at
)
SELECT replace(principal_id, 'usr_', 'prr_'), principal_id, role_id, 'workforce',
       'active', 1, 1, virtual_time, virtual_time, NULL
FROM atlas_phase01_acceptance_personas
ON CONFLICT (principal_role_id) DO NOTHING;

INSERT INTO atlas_foundation.seed_applications(seed_id, seed_checksum, policy_checksum)
SELECT payload->>'seed_id', seed_checksum, payload->>'policy_sha256'
FROM atlas_phase01_acceptance_personas_document;

DO $application_validation$
DECLARE
    document jsonb;
BEGIN
    SELECT payload INTO STRICT document
    FROM atlas_phase01_acceptance_personas_document;
    IF EXISTS (
        SELECT principal_id, display_name, person_anchor
        FROM atlas_phase01_acceptance_personas
        EXCEPT
        SELECT principal_id, display_name, person_anchor
        FROM atlas_identity.principals
        WHERE principal_type = 'workforce' AND status = 'active'
    ) OR EXISTS (
        SELECT external_subject_id, principal_id, subject
        FROM atlas_phase01_acceptance_personas
        EXCEPT
        SELECT external_subject_id, principal_id, subject
        FROM atlas_identity.external_subjects
        WHERE population = 'workforce'
          AND issuer = 'http://keycloak:8080/realms/atlas-workforce-local'
    ) OR EXISTS (
        SELECT replace(principal_id, 'usr_', 'prr_'), principal_id, role_id
        FROM atlas_phase01_acceptance_personas
        EXCEPT
        SELECT principal_role_id, principal_id, role_id
        FROM atlas_identity.principal_roles
        WHERE population = 'workforce' AND status = 'active'
    ) OR NOT EXISTS (
        SELECT 1
        FROM atlas_foundation.seed_applications
        WHERE seed_id = document->>'seed_id'
          AND seed_checksum = (
              SELECT seed_checksum FROM atlas_phase01_acceptance_personas_document
          )
          AND policy_checksum = document->>'policy_sha256'
    ) THEN
        RAISE EXCEPTION 'Phase 01 acceptance persona application drift detected';
    END IF;
END
$application_validation$;

\set QUIET off
