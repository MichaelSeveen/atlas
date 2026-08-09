#!/bin/sh
set -eu

action="${1:-setup}"
export PGPASSWORD="$ATLAS_POSTGRES_MIGRATION_PASSWORD"

run_sql() {
  psql -X -h 127.0.0.1 -U "$ATLAS_POSTGRES_MIGRATION_USER" -d "$ATLAS_POSTGRES_DB" \
    -v ON_ERROR_STOP=1 -Atqc "$1"
}

credential_id='key_01JAT1AS00000000000901'
case "$action" in
  setup)
    run_sql "DELETE FROM atlas_identity.api_credential_mutation_requests WHERE result_credential_id = '$credential_id'; DELETE FROM atlas_identity.api_credentials WHERE credential_id = '$credential_id'; INSERT INTO atlas_identity.api_credentials (credential_id, tenant_id, name, secret_verifier_sha256, secret_hint, verifier_algorithm, verifier_version, scopes, environment, audience, status, version, expires_at, created_by_principal_id, created_at, revoked_at) VALUES ('$credential_id', 'ten_01JAT1AS00000000000002', 'PITR revoked credential', decode(repeat('ab', 32), 'hex'), 'deadbeef', 'sha256', 1, ARRAY['identity:read']::text[], 'local', 'atlas-api', 'revoked', 2, '2026-10-24T00:00:00Z', 'usr_01JAT1AS00000000000002', '2026-07-26T00:00:00Z', '2026-08-09T00:00:00Z');" >/dev/null
    [ "$(run_sql "SELECT count(*) FROM atlas_identity.api_credentials WHERE credential_id = '$credential_id' AND status = 'revoked' AND revoked_at IS NOT NULL AND octet_length(secret_verifier_sha256) = 32")" = '1' ]
    echo 'phase01_credential_recovery_fixture=PASS action=setup'
    ;;
  cleanup)
    run_sql "DELETE FROM atlas_identity.api_credential_mutation_requests WHERE result_credential_id = '$credential_id'; DELETE FROM atlas_identity.api_credentials WHERE credential_id = '$credential_id';" >/dev/null
    echo 'phase01_credential_recovery_fixture=PASS action=cleanup'
    ;;
  *)
    echo 'phase01 credential recovery fixture action must be setup or cleanup' >&2
    exit 1
    ;;
esac

unset PGPASSWORD
