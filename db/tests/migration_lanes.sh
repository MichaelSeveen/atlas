#!/bin/sh
set -eu

: "${ATLAS_POSTGRES_DB:?required}"
: "${ATLAS_POSTGRES_USER:?required}"
: "${ATLAS_POSTGRES_PASSWORD:?required}"

empty_database='atlas_s05_empty_test'
previous_database='atlas_s05_previous_test'
export PGPASSWORD="$ATLAS_POSTGRES_PASSWORD"

cleanup() {
  PGPASSWORD="$ATLAS_POSTGRES_PASSWORD" psql -X -h 127.0.0.1 -U "$ATLAS_POSTGRES_USER" -d "$ATLAS_POSTGRES_DB" -v ON_ERROR_STOP=1 \
    -c "DROP DATABASE IF EXISTS $empty_database WITH (FORCE)" \
    -c "DROP DATABASE IF EXISTS $previous_database WITH (FORCE)" >/dev/null
}
trap cleanup EXIT INT TERM
cleanup

psql -X -h 127.0.0.1 -U "$ATLAS_POSTGRES_USER" -d "$ATLAS_POSTGRES_DB" -v ON_ERROR_STOP=1 \
  -c "CREATE DATABASE $empty_database OWNER atlas_migration" \
  -c "REVOKE CONNECT, TEMPORARY ON DATABASE $empty_database FROM PUBLIC" \
  -c "CREATE DATABASE $previous_database OWNER atlas_migration" \
  -c "REVOKE CONNECT, TEMPORARY ON DATABASE $previous_database FROM PUBLIC" >/dev/null

ATLAS_MIGRATION_TARGET_DATABASE="$empty_database" /database/tools/apply-migrations.sh >/dev/null
empty_count="$(PGPASSWORD="$ATLAS_POSTGRES_MIGRATION_PASSWORD" psql -X -h 127.0.0.1 -U atlas_migration -d "$empty_database" -Atqc 'SELECT count(*) FROM atlas_foundation.schema_migrations')"
[ "$empty_count" = '2' ]

ATLAS_MIGRATION_TARGET_DATABASE="$previous_database" ATLAS_MIGRATION_MAX_VERSION=1 /database/tools/apply-migrations.sh >/dev/null
previous_count="$(PGPASSWORD="$ATLAS_POSTGRES_MIGRATION_PASSWORD" psql -X -h 127.0.0.1 -U atlas_migration -d "$previous_database" -Atqc 'SELECT count(*) FROM atlas_foundation.schema_migrations')"
[ "$previous_count" = '1' ]
ATLAS_MIGRATION_TARGET_DATABASE="$previous_database" /database/tools/apply-migrations.sh >/dev/null
upgraded_count="$(PGPASSWORD="$ATLAS_POSTGRES_MIGRATION_PASSWORD" psql -X -h 127.0.0.1 -U atlas_migration -d "$previous_database" -Atqc 'SELECT count(*) FROM atlas_foundation.schema_migrations')"
[ "$upgraded_count" = '2' ]

unset PGPASSWORD
echo 'database_empty_and_previous_lanes=PASS'
