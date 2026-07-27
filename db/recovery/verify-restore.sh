#!/bin/sh
set -eu

export PGPASSWORD="$ATLAS_POSTGRES_MIGRATION_PASSWORD"
query() {
  psql -X -h 127.0.0.1 -U "$ATLAS_POSTGRES_MIGRATION_USER" -d "$ATLAS_POSTGRES_DB" -v ON_ERROR_STOP=1 -Atqc "$1"
}

[ "$(query 'SELECT count(*) FROM atlas_foundation.schema_migrations')" = '7' ]
[ "$(query "SELECT count(*) FROM atlas_foundation.recovery_probe WHERE marker_id = 's05-pitr-marker' AND marker_value = 'present-at-recovery-target'")" = '1' ]
[ "$(query 'SELECT pg_is_in_recovery()')" = 'f' ]
[ "$(query "SELECT count(*) FROM atlas_foundation.seed_applications WHERE seed_id = 'atlas-phase01-identity-v1'")" = '1' ]
[ "$(query "SELECT count(*) FROM atlas_identity.principals")" = '3' ]
[ "$(query "SELECT count(*) FROM atlas_identity.memberships")" = '2' ]
[ "$(query "SELECT count(*) FROM atlas_identity.sessions WHERE session_id = 'ses_01JAT1AS00000000000901' AND status = 'revoked' AND revoked_at IS NOT NULL")" = '1' ]
[ "$(query "SELECT count(*) FROM atlas_audit.audit_events WHERE audit_event_id = 'aud_01JAT1AS00000000000001'")" = '1' ]
[ "$(query "SELECT to_regclass('atlas_identity.oidc_transactions') IS NOT NULL")" = 't' ]
[ "$(query "SELECT to_regclass('atlas_identity.session_revocation_requests') IS NOT NULL")" = 't' ]
[ "$(query "SELECT to_regclass('atlas_identity.step_up_challenge_requests') IS NOT NULL")" = 't' ]
[ "$(query "SELECT has_table_privilege('atlas_api', 'atlas_identity.memberships', 'SELECT')")" = 't' ]
[ "$(query "SELECT has_column_privilege('atlas_api', 'atlas_identity.principals', 'authorization_version', 'UPDATE')")" = 't' ]
[ "$(query "SELECT has_column_privilege('atlas_api', 'atlas_identity.principals', 'status', 'UPDATE')")" = 'f' ]
[ "$(query "SELECT has_column_privilege('atlas_api', 'atlas_identity.principals', 'display_name', 'UPDATE')")" = 'f' ]
[ "$(query "SELECT has_table_privilege('atlas_api', 'atlas_identity.oidc_transactions', 'SELECT,INSERT,UPDATE')")" = 't' ]
[ "$(query "SELECT has_table_privilege('atlas_api', 'atlas_identity.oidc_transactions', 'DELETE')")" = 'f' ]
[ "$(query "SELECT has_table_privilege('atlas_api', 'atlas_identity.session_revocation_requests', 'SELECT,INSERT')")" = 't' ]
[ "$(query "SELECT has_table_privilege('atlas_api', 'atlas_identity.session_revocation_requests', 'UPDATE,DELETE')")" = 'f' ]
[ "$(query "SELECT has_table_privilege('atlas_api', 'atlas_identity.step_up_challenge_requests', 'SELECT,INSERT,UPDATE')")" = 't' ]
[ "$(query "SELECT has_table_privilege('atlas_api', 'atlas_identity.step_up_challenge_requests', 'DELETE')")" = 'f' ]
[ "$(query "SELECT has_table_privilege('atlas_api', 'atlas_audit.audit_events', 'INSERT')")" = 't' ]
[ "$(query "SELECT has_table_privilege('atlas_api', 'atlas_audit.audit_events', 'UPDATE')")" = 'f' ]
[ "$(query "SELECT has_table_privilege('atlas_worker', 'atlas_identity.sessions', 'SELECT')")" = 'f' ]
[ "$(query "SELECT has_table_privilege('atlas_worker', 'atlas_identity.oidc_transactions', 'SELECT')")" = 'f' ]
[ "$(query "SELECT has_table_privilege('atlas_reporting_read', 'atlas_identity.session_revocation_requests', 'SELECT')")" = 'f' ]
[ "$(query "SELECT has_table_privilege('atlas_worker', 'atlas_identity.step_up_challenge_requests', 'SELECT')")" = 'f' ]

unset PGPASSWORD
echo 'database_isolated_pitr_restore=PASS product_identity_state=verified revoked_authority=preserved'
