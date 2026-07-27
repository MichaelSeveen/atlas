#!/bin/sh
set -eu

: "${ATLAS_POSTGRES_DB:?required}"
: "${ATLAS_POSTGRES_USER:?required}"
: "${ATLAS_POSTGRES_PASSWORD:?required}"

test_database='atlas_p01_s03_seed_test'
failure_output='/tmp/atlas-p01-s03-denial.out'

admin_sql() {
  PGPASSWORD="$ATLAS_POSTGRES_PASSWORD" psql -X -h 127.0.0.1 -U "$ATLAS_POSTGRES_USER" -d "$ATLAS_POSTGRES_DB" -v ON_ERROR_STOP=1 -Atqc "$1"
}

query() {
  PGPASSWORD="$ATLAS_POSTGRES_MIGRATION_PASSWORD" psql -X -h 127.0.0.1 -U atlas_migration -d "$test_database" -v ON_ERROR_STOP=1 -Atqc "$1"
}

expect_rejected() {
  sql="$1"
  label="$2"
  if query "$sql" >"$failure_output" 2>&1; then
    echo "Phase 01 database canary unexpectedly accepted $label" >&2
    rm -f "$failure_output"
    exit 1
  fi
  rm -f "$failure_output"
}

cleanup() {
  admin_sql "DROP DATABASE IF EXISTS $test_database WITH (FORCE)" >/dev/null 2>&1 || true
  rm -f "$failure_output"
}
trap cleanup EXIT INT TERM
cleanup

admin_sql "CREATE DATABASE $test_database OWNER atlas_migration" >/dev/null
admin_sql "REVOKE CONNECT, TEMPORARY ON DATABASE $test_database FROM PUBLIC" >/dev/null

ATLAS_MIGRATION_TARGET_DATABASE="$test_database" /database/tools/apply-migrations.sh >/dev/null
ATLAS_SEED_TARGET_DATABASE="$test_database" /database/tools/apply-phase-01-seeds.sh >/dev/null
ATLAS_SEED_TARGET_DATABASE="$test_database" /database/tools/apply-phase-01-seeds.sh >/dev/null

[ "$(query 'SELECT count(*) FROM atlas_foundation.schema_migrations')" = '8' ]
[ "$(query 'SELECT count(*) FROM atlas_foundation.seed_applications')" = '2' ]
[ "$(query "SELECT count(*) FROM atlas_foundation.data_scope_registry WHERE schema_name IN ('atlas_identity', 'atlas_audit')")" = '15' ]
[ "$(query 'SELECT count(*) FROM atlas_identity.permission_catalogue')" = '23' ]
[ "$(query 'SELECT count(*) FROM atlas_identity.role_catalogue')" = '13' ]
[ "$(query "SELECT count(*) FROM atlas_identity.permission_catalogue WHERE policy_checksum = '8c5085e94e6006b232f28974ebb6aa251452be18647f9863dd4155ce43c7f8cf'")" = '23' ]
[ "$(query "SELECT count(*) FROM atlas_identity.role_catalogue WHERE policy_checksum = '8c5085e94e6006b232f28974ebb6aa251452be18647f9863dd4155ce43c7f8cf'")" = '13' ]
[ "$(query 'SELECT count(*) FROM atlas_identity.role_permissions')" = '119' ]
[ "$(query 'SELECT count(*) FROM atlas_identity.role_delegations')" = '6' ]
[ "$(query 'SELECT count(*) FROM atlas_identity.organizations')" = '2' ]
[ "$(query 'SELECT count(*) FROM atlas_identity.principals')" = '3' ]
[ "$(query 'SELECT count(*) FROM atlas_identity.external_subjects')" = '3' ]
[ "$(query 'SELECT count(*) FROM atlas_identity.memberships')" = '2' ]
[ "$(query 'SELECT count(*) FROM atlas_identity.principal_roles')" = '1' ]
[ "$(query "SELECT count(*) FROM atlas_identity.sessions WHERE status = 'revoked' AND revoked_at IS NOT NULL")" = '1' ]
[ "$(query 'SELECT count(*) FROM atlas_audit.audit_events')" = '1' ]

registered_tables="$(query "SELECT count(*) FROM information_schema.tables t JOIN atlas_foundation.data_scope_registry r ON r.schema_name = t.table_schema AND r.table_name = t.table_name WHERE t.table_type = 'BASE TABLE' AND t.table_schema IN ('atlas_identity', 'atlas_audit')")"
product_tables="$(query "SELECT count(*) FROM information_schema.tables WHERE table_type = 'BASE TABLE' AND table_schema IN ('atlas_identity', 'atlas_audit')")"
[ "$registered_tables" = "$product_tables" ]

[ "$(query "SELECT count(*) FROM atlas_identity.memberships WHERE tenant_id = 'ten_01JAT1AS00000000000001' AND membership_id = 'mem_01JAT1AS00000000000001'")" = '1' ]
[ "$(query "SELECT count(*) FROM atlas_identity.memberships WHERE tenant_id = 'ten_01JAT1AS00000000000002' AND membership_id = 'mem_01JAT1AS00000000000001'")" = '0' ]

expect_rejected "INSERT INTO atlas_identity.external_subjects(external_subject_id, principal_id, population, issuer, subject, created_at) VALUES ('ext_01JAT1AS00000000000004', 'usr_01JAT1AS00000000000001', 'customer', 'http://keycloak:8080/realms/atlas-customer-local', '00000000-0000-4000-8000-000000000101', '2026-07-26T00:00:00Z')" duplicate-external-subject
expect_rejected "INSERT INTO atlas_identity.memberships(membership_id, tenant_id, principal_id, role_id, population, status, authorization_version, version, created_at, updated_at) VALUES ('mem_01JAT1AS00000000000003', 'ten_01JAT1AS00000000000002', 'usr_01JAT1AS00000000000001', 'merchant_viewer', 'merchant', 'active', 1, 1, '2026-07-26T00:00:00Z', '2026-07-26T00:00:00Z')" cross-population-membership
expect_rejected "INSERT INTO atlas_identity.oidc_transactions(transaction_id, transaction_kind, population, state_sha256, nonce_sha256, pkce_verifier_ciphertext, encryption_key_version, return_to, principal_id, status, created_at, expires_at) VALUES ('oid_01JAT1AS00000000000001', 'login', 'customer', decode(repeat('11', 32), 'hex'), decode(repeat('22', 32), 'hex'), decode(repeat('33', 60), 'hex'), 1, '/customer', 'usr_01JAT1AS00000000000001', 'pending', '2026-07-26T00:00:00Z', '2026-07-26T00:05:00Z')" login-transaction-principal-binding
expect_rejected "INSERT INTO atlas_identity.oidc_transactions(transaction_id, transaction_kind, population, state_sha256, nonce_sha256, pkce_verifier_ciphertext, encryption_key_version, return_to, status, created_at, expires_at) VALUES ('oid_01JAT1AS00000000000002', 'login', 'customer', decode(repeat('44', 32), 'hex'), decode(repeat('55', 32), 'hex'), decode(repeat('66', 60), 'hex'), 1, '/merchant', 'pending', '2026-07-26T00:00:00Z', '2026-07-26T00:05:00Z')" cross-population-login-return
expect_rejected "BEGIN; INSERT INTO atlas_identity.oidc_transactions(transaction_id, transaction_kind, population, state_sha256, nonce_sha256, pkce_verifier_ciphertext, encryption_key_version, return_to, status, created_at, expires_at) VALUES ('oid_01JAT1AS00000000000003', 'login', 'customer', decode(repeat('77', 32), 'hex'), decode(repeat('88', 32), 'hex'), decode(repeat('99', 60), 'hex'), 1, '/customer', 'pending', '2026-07-26T00:00:00Z', '2026-07-26T00:05:00Z'); INSERT INTO atlas_identity.oidc_transactions(transaction_id, transaction_kind, population, state_sha256, nonce_sha256, pkce_verifier_ciphertext, encryption_key_version, return_to, status, created_at, expires_at) VALUES ('oid_01JAT1AS00000000000004', 'login', 'customer', decode(repeat('77', 32), 'hex'), decode(repeat('aa', 32), 'hex'), decode(repeat('bb', 60), 'hex'), 1, '/customer', 'pending', '2026-07-26T00:00:00Z', '2026-07-26T00:05:00Z'); COMMIT;" duplicate-oidc-state
expect_rejected "INSERT INTO atlas_identity.step_up_challenge_requests(challenge_request_id, principal_id, population, global_scope, idempotency_scope_sha256, request_sha256, requested_action, lifecycle, processing_expires_at, correlation_id, created_at, updated_at, retained_until) VALUES ('idr_01JAT1AS00000000000001', 'usr_01JAT1AS00000000000001', 'customer', 'identity-security', decode(repeat('12', 32), 'hex'), decode(repeat('13', 32), 'hex'), 'identity.approval.decide', 'processing', '2026-07-26T00:00:30Z', 'cor_01JAT1AS00000000000001', '2026-07-26T00:00:00Z', '2026-07-26T00:00:00Z', '2026-07-27T00:00:00Z')" customer-step-up-without-tenant
expect_rejected "INSERT INTO atlas_identity.step_up_challenge_requests(challenge_request_id, principal_id, population, tenant_id, idempotency_scope_sha256, request_sha256, requested_action, lifecycle, correlation_id, created_at, updated_at, retained_until) VALUES ('idr_01JAT1AS00000000000002', 'usr_01JAT1AS00000000000001', 'customer', 'ten_01JAT1AS00000000000001', decode(repeat('14', 32), 'hex'), decode(repeat('15', 32), 'hex'), 'identity.approval.decide', 'completed', 'cor_01JAT1AS00000000000002', '2026-07-26T00:00:00Z', '2026-07-26T00:00:00Z', '2026-07-27T00:00:00Z')" incomplete-step-up-replay
expect_rejected "UPDATE atlas_identity.sessions SET step_up_action = 'identity.session.admin_revoke' WHERE session_id = 'ses_01JAT1AS00000000000901'" unpaired-session-step-up-binding
expect_rejected "INSERT INTO atlas_identity.admin_session_revocation_requests(revocation_request_id, actor_principal_id, actor_session_id, target_session_id, idempotency_key_sha256, request_sha256, purpose, reason_code, outcome, current_revoked, decision_id, committed_at) VALUES ('asr_01JAT1AS00000000000001', 'usr_01JAT1AS00000000000003', 'ses_01JAT1AS00000000000901', 'ses_01JAT1AS00000000000902', decode(repeat('16', 32), 'hex'), decode(repeat('17', 32), 'hex'), 'self_service', 'compromised_session', 'not_found', false, 'dec_01JAT1AS00000000000003', '2026-07-26T00:00:00Z')" invalid-admin-revocation-purpose

atomic_event='aud_01JAT1AS00000000000997'
expect_rejected "BEGIN; INSERT INTO atlas_audit.audit_events(audit_event_id, actor_id, actor_type, tenant_id, session_assurance, action, target_type, target_id, decision_id, decision, reason_code, correlation_id, occurred_at) VALUES ('$atomic_event', 'usr_01JAT1AS00000000000003', 'workforce', 'ten_01JAT1AS00000000000002', 'none', 'identity.atomic.canary', 'membership', 'mem_01JAT1AS00000000000002', 'dec_01JAT1AS00000000000997', 'executed', 'atomic_seed', 'cor_01JAT1AS00000000000997', '2026-07-26T00:00:00Z'); INSERT INTO atlas_identity.memberships(membership_id, tenant_id, principal_id, role_id, population, status, authorization_version, version, created_at, updated_at) VALUES ('mem_01JAT1AS00000000000004', 'ten_01JAT1AS00000000000002', 'usr_01JAT1AS00000000000001', 'merchant_viewer', 'merchant', 'active', 1, 1, '2026-07-26T00:00:00Z', '2026-07-26T00:00:00Z'); COMMIT;" audit-atomicity
[ "$(query "SELECT count(*) FROM atlas_audit.audit_events WHERE audit_event_id = '$atomic_event'")" = '0' ]

echo 'phase01_identity_seed_idempotence=PASS'
echo 'phase01_identity_tenant_predicate=PASS'
echo 'phase01_identity_population_constraints=PASS'
echo 'phase01_oidc_transaction_constraints=PASS'
echo 'phase01_step_up_replay_constraints=PASS'
echo 'phase01_admin_revocation_constraints=PASS'
echo 'phase01_audit_atomicity=PASS'
