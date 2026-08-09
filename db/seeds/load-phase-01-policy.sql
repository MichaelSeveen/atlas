\set ON_ERROR_STOP on
\set QUIET on

CREATE TEMP TABLE atlas_phase01_policy_seed_document (
    payload jsonb NOT NULL,
    seed_checksum character(64) NOT NULL
) ON COMMIT DROP;

INSERT INTO atlas_phase01_policy_seed_document(payload, seed_checksum)
VALUES (:'seed_document'::jsonb, :'seed_checksum');

DO $seed_validation$
DECLARE
    document jsonb;
    document_checksum text;
BEGIN
    SELECT payload, seed_checksum INTO STRICT document, document_checksum
    FROM atlas_phase01_policy_seed_document;
    IF document->>'seed_id' <> 'atlas-phase01-identity-policy-v2'
       OR document->>'predecessor_seed_id' <> 'atlas-phase01-identity-v1'
       OR (document->>'schema_version')::integer <> 1
       OR document->>'virtual_time' <> '2026-07-27T00:00:00Z'
       OR document->>'previous_policy_sha256' !~ '^[0-9a-f]{64}$'
       OR document->>'policy_sha256' !~ '^[0-9a-f]{64}$'
       OR document->>'previous_policy_sha256' = document->>'policy_sha256'
       OR document_checksum !~ '^[0-9a-f]{64}$' THEN
        RAISE EXCEPTION 'Phase 01 policy seed identity or checksum is invalid';
    END IF;
END
$seed_validation$;

DO $predecessor_validation$
DECLARE
    document jsonb;
BEGIN
    SELECT payload INTO STRICT document
    FROM atlas_phase01_policy_seed_document;
    IF NOT EXISTS (
        SELECT 1
        FROM atlas_foundation.seed_applications
        WHERE seed_id = document->>'predecessor_seed_id'
          AND seed_checksum = 'e5a8ab37437edad69ed655e6589efffd824ca4b9151b6f9d9358632bf1f13d6c'
          AND policy_checksum = document->>'previous_policy_sha256'
    ) OR (
        SELECT count(*)
        FROM atlas_identity.permission_catalogue
        WHERE policy_checksum = document->>'previous_policy_sha256'
    ) <> 23 OR (
        SELECT count(*)
        FROM atlas_identity.role_catalogue
        WHERE policy_checksum = document->>'previous_policy_sha256'
    ) <> 13 THEN
        RAISE EXCEPTION 'Phase 01 policy seed predecessor state is invalid';
    END IF;
END
$predecessor_validation$;

UPDATE atlas_identity.permission_catalogue
SET policy_checksum = document.payload->>'policy_sha256'
FROM atlas_phase01_policy_seed_document AS document
WHERE atlas_identity.permission_catalogue.policy_checksum =
      document.payload->>'previous_policy_sha256';

UPDATE atlas_identity.role_catalogue
SET policy_checksum = document.payload->>'policy_sha256'
FROM atlas_phase01_policy_seed_document AS document
WHERE atlas_identity.role_catalogue.policy_checksum =
      document.payload->>'previous_policy_sha256';

INSERT INTO atlas_foundation.seed_applications(seed_id, seed_checksum, policy_checksum)
SELECT payload->>'seed_id', seed_checksum, payload->>'policy_sha256'
FROM atlas_phase01_policy_seed_document;

DO $seed_application$
DECLARE
    document jsonb;
BEGIN
    SELECT payload INTO STRICT document
    FROM atlas_phase01_policy_seed_document;
    IF (
        SELECT count(*)
        FROM atlas_identity.permission_catalogue
        WHERE policy_checksum = document->>'policy_sha256'
    ) <> 23 OR (
        SELECT count(*)
        FROM atlas_identity.role_catalogue
        WHERE policy_checksum = document->>'policy_sha256'
    ) <> 13 OR NOT EXISTS (
        SELECT 1
        FROM atlas_foundation.seed_applications
        WHERE seed_id = document->>'seed_id'
          AND seed_checksum = (
              SELECT seed_checksum FROM atlas_phase01_policy_seed_document
          )
          AND policy_checksum = document->>'policy_sha256'
    ) THEN
        RAISE EXCEPTION 'Phase 01 policy seed application drift detected';
    END IF;
END
$seed_application$;

\set QUIET off
