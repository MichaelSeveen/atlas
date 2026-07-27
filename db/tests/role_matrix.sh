#!/bin/sh
set -eu

output='/tmp/atlas-s05-role-denial.out'

run_sql() {
  role="$1"
  password="$2"
  sql="$3"
  PGPASSWORD="$password" psql -X -h 127.0.0.1 -U "$role" -d "$ATLAS_POSTGRES_DB" -v ON_ERROR_STOP=1 -Atqc "$sql"
}

expect_denied() {
  role="$1"
  password="$2"
  sql="$3"
  label="$4"
  if run_sql "$role" "$password" "$sql" >"$output" 2>&1; then
    echo "database privilege canary $label unexpectedly succeeded for $role" >&2
    rm -f "$output"
    exit 1
  fi
  rm -f "$output"
}

bootstrap_sql() {
  PGPASSWORD="$ATLAS_POSTGRES_PASSWORD" psql -X -h 127.0.0.1 -U "$ATLAS_POSTGRES_USER" -d "$ATLAS_POSTGRES_DB" -v ON_ERROR_STOP=1 -Atqc "$1"
}

deactivate_break_glass() {
  bootstrap_sql "ALTER ROLE atlas_break_glass VALID UNTIL '1970-01-01T00:00:00Z'" >/dev/null 2>&1 || true
  rm -f "$output"
}
trap deactivate_break_glass EXIT INT TERM

unsafe_roles="$(bootstrap_sql "SELECT count(*) FROM pg_roles WHERE rolname IN ('atlas_migration','atlas_api','atlas_worker','atlas_reporting_read','atlas_break_glass') AND (rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls)")"
[ "$unsafe_roles" = '0' ]

run_sql atlas_migration "$ATLAS_POSTGRES_MIGRATION_PASSWORD" 'CREATE TABLE atlas_foundation.migration_role_canary(id integer); DROP TABLE atlas_foundation.migration_role_canary;' >/dev/null

run_sql atlas_api "$ATLAS_POSTGRES_API_PASSWORD" "INSERT INTO atlas_foundation.permission_probe(probe_key, marker) VALUES ('api', 'synthetic') ON CONFLICT (probe_key) DO UPDATE SET marker = EXCLUDED.marker; SELECT marker FROM atlas_foundation.permission_probe WHERE probe_key = 'api'; DELETE FROM atlas_foundation.permission_probe WHERE probe_key = 'api';" >/dev/null
run_sql atlas_worker "$ATLAS_POSTGRES_WORKER_PASSWORD" "INSERT INTO atlas_foundation.permission_probe(probe_key, marker) VALUES ('worker', 'synthetic') ON CONFLICT (probe_key) DO UPDATE SET marker = EXCLUDED.marker; DELETE FROM atlas_foundation.permission_probe WHERE probe_key = 'worker';" >/dev/null
run_sql atlas_reporting_read "$ATLAS_POSTGRES_REPORTING_PASSWORD" 'SELECT count(*) FROM atlas_foundation.permission_probe' >/dev/null
run_sql atlas_api "$ATLAS_POSTGRES_API_PASSWORD" 'SELECT count(*) FROM atlas_identity.memberships' >/dev/null
run_sql atlas_api "$ATLAS_POSTGRES_API_PASSWORD" "BEGIN; SELECT principal_id FROM atlas_identity.principals WHERE principal_id = 'usr_01JAT1AS00000000000001' FOR SHARE; UPDATE atlas_identity.principals SET authorization_version = authorization_version WHERE principal_id = 'usr_01JAT1AS00000000000001'; ROLLBACK;" >/dev/null
run_sql atlas_api "$ATLAS_POSTGRES_API_PASSWORD" "BEGIN; INSERT INTO atlas_identity.oidc_transactions(transaction_id, transaction_kind, population, state_sha256, nonce_sha256, pkce_verifier_ciphertext, encryption_key_version, return_to, status, created_at, expires_at) VALUES ('oid_01JAT1AS00000000000991', 'login', 'customer', decode(repeat('11', 32), 'hex'), decode(repeat('22', 32), 'hex'), decode(repeat('33', 60), 'hex'), 1, '/customer', 'pending', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP + INTERVAL '5 minutes'); UPDATE atlas_identity.oidc_transactions SET status = 'consumed', consumed_at = CURRENT_TIMESTAMP WHERE transaction_id = 'oid_01JAT1AS00000000000991'; ROLLBACK;" >/dev/null
run_sql atlas_api "$ATLAS_POSTGRES_API_PASSWORD" "BEGIN; INSERT INTO atlas_identity.session_revocation_requests(revocation_request_id, principal_id, idempotency_key_sha256, request_sha256, include_current, current_revoked, committed_at) VALUES ('rev_01JAT1AS00000000000991', 'usr_01JAT1AS00000000000001', decode(repeat('44', 32), 'hex'), decode(repeat('55', 32), 'hex'), true, false, CURRENT_TIMESTAMP); ROLLBACK;" >/dev/null
run_sql atlas_api "$ATLAS_POSTGRES_API_PASSWORD" "BEGIN; INSERT INTO atlas_identity.step_up_challenge_requests(challenge_request_id, principal_id, population, tenant_id, idempotency_scope_sha256, request_sha256, requested_action, lifecycle, processing_expires_at, correlation_id, created_at, updated_at, retained_until) VALUES ('idr_01JAT1AS00000000000991', 'usr_01JAT1AS00000000000001', 'customer', 'ten_01JAT1AS00000000000001', decode(repeat('66', 32), 'hex'), decode(repeat('77', 32), 'hex'), 'identity.approval.decide', 'processing', CURRENT_TIMESTAMP + INTERVAL '30 seconds', 'cor_01JAT1AS00000000000991', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP + INTERVAL '24 hours'); UPDATE atlas_identity.step_up_challenge_requests SET lifecycle = 'failed-retryable', processing_expires_at = NULL WHERE challenge_request_id = 'idr_01JAT1AS00000000000991'; ROLLBACK;" >/dev/null
run_sql atlas_api "$ATLAS_POSTGRES_API_PASSWORD" "BEGIN; INSERT INTO atlas_audit.audit_events(audit_event_id, actor_id, actor_type, global_scope, session_assurance, action, target_type, target_id, decision_id, decision, reason_code, correlation_id, occurred_at) VALUES ('aud_01JAT1AS00000000000999', 'usr_01JAT1AS00000000000003', 'workforce', 'platform-security', 'none', 'identity.audit.canary', 'database_role', 'atlas_api', 'dec_01JAT1AS00000000000999', 'executed', 'role_matrix', 'cor_01JAT1AS00000000000999', '2026-07-26T00:00:00Z'); ROLLBACK;" >/dev/null

expect_denied atlas_reporting_read "$ATLAS_POSTGRES_REPORTING_PASSWORD" "INSERT INTO atlas_foundation.permission_probe(probe_key, marker) VALUES ('reporting', 'denied')" reporting-write
expect_denied atlas_reporting_read "$ATLAS_POSTGRES_REPORTING_PASSWORD" 'SELECT count(*) FROM atlas_identity.memberships' reporting-identity-read
expect_denied atlas_reporting_read "$ATLAS_POSTGRES_REPORTING_PASSWORD" 'SELECT count(*) FROM atlas_identity.oidc_transactions' reporting-oidc-state-read
expect_denied atlas_worker "$ATLAS_POSTGRES_WORKER_PASSWORD" 'SELECT count(*) FROM atlas_identity.sessions' worker-identity-read
expect_denied atlas_worker "$ATLAS_POSTGRES_WORKER_PASSWORD" 'SELECT count(*) FROM atlas_identity.oidc_transactions' worker-oidc-state-read
expect_denied atlas_worker "$ATLAS_POSTGRES_WORKER_PASSWORD" 'SELECT count(*) FROM atlas_identity.session_revocation_requests' worker-revocation-replay-read
expect_denied atlas_worker "$ATLAS_POSTGRES_WORKER_PASSWORD" 'SELECT count(*) FROM atlas_identity.step_up_challenge_requests' worker-step-up-replay-read
expect_denied atlas_reporting_read "$ATLAS_POSTGRES_REPORTING_PASSWORD" 'SELECT count(*) FROM atlas_identity.step_up_challenge_requests' reporting-step-up-replay-read
expect_denied atlas_worker "$ATLAS_POSTGRES_WORKER_PASSWORD" "INSERT INTO atlas_audit.audit_events(audit_event_id, actor_id, actor_type, global_scope, session_assurance, action, target_type, target_id, decision_id, decision, reason_code, correlation_id, occurred_at) VALUES ('aud_01JAT1AS00000000000998', 'usr_01JAT1AS00000000000003', 'workforce', 'platform-security', 'none', 'identity.audit.canary', 'database_role', 'atlas_worker', 'dec_01JAT1AS00000000000998', 'denied', 'role_matrix', 'cor_01JAT1AS00000000000998', '2026-07-26T00:00:00Z')" worker-audit-write
expect_denied atlas_api "$ATLAS_POSTGRES_API_PASSWORD" 'SELECT count(*) FROM atlas_audit.audit_events' api-audit-read
expect_denied atlas_api "$ATLAS_POSTGRES_API_PASSWORD" "UPDATE atlas_audit.audit_events SET reason_code = 'tampered' WHERE audit_event_id = 'aud_01JAT1AS00000000000001'" api-audit-update
expect_denied atlas_api "$ATLAS_POSTGRES_API_PASSWORD" "DELETE FROM atlas_audit.audit_events WHERE audit_event_id = 'aud_01JAT1AS00000000000001'" api-audit-delete
expect_denied atlas_api "$ATLAS_POSTGRES_API_PASSWORD" "INSERT INTO atlas_identity.role_catalogue(role_id, population, standing_status, policy_checksum) VALUES ('api_bypass', 'workforce', 'enabled', repeat('0', 64))" api-policy-write
expect_denied atlas_api "$ATLAS_POSTGRES_API_PASSWORD" "UPDATE atlas_identity.principals SET status = 'disabled' WHERE principal_id = 'usr_01JAT1AS00000000000001'" api-principal-status-write
expect_denied atlas_api "$ATLAS_POSTGRES_API_PASSWORD" "UPDATE atlas_identity.principals SET display_name = 'tampered' WHERE principal_id = 'usr_01JAT1AS00000000000001'" api-principal-profile-write
expect_denied atlas_api "$ATLAS_POSTGRES_API_PASSWORD" "UPDATE atlas_foundation.data_scope_registry SET scope_kind = 'global' WHERE schema_name = 'atlas_identity' AND table_name = 'memberships'" api-scope-registry-write
expect_denied atlas_api "$ATLAS_POSTGRES_API_PASSWORD" "DELETE FROM atlas_identity.oidc_transactions WHERE transaction_id = 'oid_01JAT1AS00000000000991'" api-oidc-state-delete
expect_denied atlas_api "$ATLAS_POSTGRES_API_PASSWORD" "DELETE FROM atlas_identity.session_revocation_requests WHERE revocation_request_id = 'rev_01JAT1AS00000000000991'" api-revocation-replay-delete
expect_denied atlas_api "$ATLAS_POSTGRES_API_PASSWORD" "DELETE FROM atlas_identity.step_up_challenge_requests WHERE challenge_request_id = 'idr_01JAT1AS00000000000991'" api-step-up-replay-delete
expect_denied atlas_api "$ATLAS_POSTGRES_API_PASSWORD" 'CREATE SCHEMA api_bypass' api-create-schema
expect_denied atlas_api "$ATLAS_POSTGRES_API_PASSWORD" 'CREATE TABLE atlas_foundation.api_ddl_bypass(id integer)' api-create-table
expect_denied atlas_api "$ATLAS_POSTGRES_API_PASSWORD" 'ALTER TABLE atlas_foundation.permission_probe ADD COLUMN api_bypass text' api-alter-table
expect_denied atlas_api "$ATLAS_POSTGRES_API_PASSWORD" 'DROP TABLE atlas_foundation.permission_probe' api-drop-table
run_sql atlas_api "$ATLAS_POSTGRES_API_PASSWORD" 'GRANT SELECT ON atlas_foundation.permission_probe TO PUBLIC' >"$output" 2>&1 || true
rm -f "$output"
[ "$(bootstrap_sql "SELECT has_table_privilege('public', 'atlas_foundation.permission_probe', 'SELECT')")" = 'f' ]
[ "$(bootstrap_sql "SELECT count(*) FROM information_schema.role_table_grants WHERE grantee = 'PUBLIC' AND table_schema IN ('atlas_identity', 'atlas_audit')")" = '0' ]
expect_denied atlas_api "$ATLAS_POSTGRES_API_PASSWORD" 'SET ROLE atlas_migration' api-set-migration-role
expect_denied atlas_worker "$ATLAS_POSTGRES_WORKER_PASSWORD" 'ALTER TABLE atlas_foundation.permission_probe ADD COLUMN worker_bypass text' worker-alter-table
expect_denied atlas_reporting_read "$ATLAS_POSTGRES_REPORTING_PASSWORD" 'CREATE TEMP TABLE reporting_temp(id integer)' reporting-create-temp

expect_denied atlas_break_glass "$ATLAS_POSTGRES_BREAK_GLASS_PASSWORD" 'SELECT 1' break-glass-disabled
break_glass_expiry="$(bootstrap_sql "SELECT CURRENT_TIMESTAMP + INTERVAL '5 minutes'")"
bootstrap_sql "ALTER ROLE atlas_break_glass VALID UNTIL '$break_glass_expiry'" >/dev/null
run_sql atlas_break_glass "$ATLAS_POSTGRES_BREAK_GLASS_PASSWORD" 'SET ROLE atlas_migration; CREATE TABLE atlas_foundation.break_glass_canary(id integer); DROP TABLE atlas_foundation.break_glass_canary;' >/dev/null
deactivate_break_glass
expect_denied atlas_break_glass "$ATLAS_POSTGRES_BREAK_GLASS_PASSWORD" 'SELECT 1' break-glass-expired

echo 'database_role_matrix=PASS'
