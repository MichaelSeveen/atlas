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

expect_denied atlas_reporting_read "$ATLAS_POSTGRES_REPORTING_PASSWORD" "INSERT INTO atlas_foundation.permission_probe(probe_key, marker) VALUES ('reporting', 'denied')" reporting-write
expect_denied atlas_api "$ATLAS_POSTGRES_API_PASSWORD" 'CREATE SCHEMA api_bypass' api-create-schema
expect_denied atlas_api "$ATLAS_POSTGRES_API_PASSWORD" 'CREATE TABLE atlas_foundation.api_ddl_bypass(id integer)' api-create-table
expect_denied atlas_api "$ATLAS_POSTGRES_API_PASSWORD" 'ALTER TABLE atlas_foundation.permission_probe ADD COLUMN api_bypass text' api-alter-table
expect_denied atlas_api "$ATLAS_POSTGRES_API_PASSWORD" 'DROP TABLE atlas_foundation.permission_probe' api-drop-table
run_sql atlas_api "$ATLAS_POSTGRES_API_PASSWORD" 'GRANT SELECT ON atlas_foundation.permission_probe TO PUBLIC' >"$output" 2>&1 || true
rm -f "$output"
[ "$(bootstrap_sql "SELECT has_table_privilege('public', 'atlas_foundation.permission_probe', 'SELECT')")" = 'f' ]
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
