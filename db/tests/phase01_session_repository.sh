#!/bin/sh
set -eu

: "${ATLAS_P01_S04_SESSION_TEST_ACTION:?required}"
: "${ATLAS_POSTGRES_DB:?required}"
: "${ATLAS_POSTGRES_USER:?required}"
: "${ATLAS_POSTGRES_PASSWORD:?required}"
: "${ATLAS_POSTGRES_API_USER:?required}"
: "${ATLAS_POSTGRES_MIGRATION_USER:?required}"

test_database='atlas_p01_s04_session_test'

admin_sql() {
  PGPASSWORD="$ATLAS_POSTGRES_PASSWORD" psql \
    -X \
    -h 127.0.0.1 \
    -U "$ATLAS_POSTGRES_USER" \
    -d "$ATLAS_POSTGRES_DB" \
    -v ON_ERROR_STOP=1 \
    -Atqc "$1"
}

drop_test_database() {
  admin_sql "DROP DATABASE IF EXISTS $test_database WITH (FORCE)" >/dev/null
}

case "$ATLAS_P01_S04_SESSION_TEST_ACTION" in
  create)
    drop_test_database
    admin_sql "CREATE DATABASE $test_database OWNER $ATLAS_POSTGRES_MIGRATION_USER TEMPLATE template0 ENCODING 'UTF8'" >/dev/null
    admin_sql "REVOKE ALL ON DATABASE $test_database FROM PUBLIC" >/dev/null
    admin_sql "GRANT CONNECT ON DATABASE $test_database TO $ATLAS_POSTGRES_API_USER" >/dev/null
    ATLAS_MIGRATION_TARGET_DATABASE="$test_database" sh /database/tools/apply-migrations.sh
    ATLAS_SEED_TARGET_DATABASE="$test_database" sh /database/tools/apply-phase-01-seeds.sh
    PGPASSWORD="$ATLAS_POSTGRES_MIGRATION_PASSWORD" psql \
      -X \
      -h 127.0.0.1 \
      -U "$ATLAS_POSTGRES_MIGRATION_USER" \
      -d "$test_database" \
      -v ON_ERROR_STOP=1 \
      -Atqc "
        SELECT CASE
          WHEN (SELECT count(*) FROM atlas_foundation.schema_migrations) = 8
           AND (SELECT count(*) FROM atlas_identity.principals) = 3
          THEN true
          ELSE false
        END
      " | grep -qx 't'
    echo 'p01_s04_session_test_database=create_PASS'
    ;;
  drop)
    drop_test_database
    echo 'p01_s04_session_test_database=drop_PASS'
    ;;
  *)
    echo 'unsupported Phase 01 S04 session test database action' >&2
    exit 1
    ;;
esac
