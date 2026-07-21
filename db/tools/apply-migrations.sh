#!/bin/sh
set -eu

: "${ATLAS_POSTGRES_DB:?required}"
: "${ATLAS_POSTGRES_MIGRATION_USER:?required}"
: "${ATLAS_POSTGRES_MIGRATION_PASSWORD:?required}"

database_name="${ATLAS_MIGRATION_TARGET_DATABASE:-$ATLAS_POSTGRES_DB}"
maximum_version="${ATLAS_MIGRATION_MAX_VERSION:-999999}"
case "$database_name" in
  atlas_local|atlas_s05_empty_test|atlas_s05_previous_test) ;;
  *) echo 'migration target is outside the contained S05 database set' >&2; exit 1 ;;
esac
case "$maximum_version" in
  *[!0-9]*|'') echo 'migration maximum version is invalid' >&2; exit 1 ;;
esac

export PGPASSWORD="$ATLAS_POSTGRES_MIGRATION_PASSWORD"
export PGOPTIONS='-c lock_timeout=500ms -c statement_timeout=5s -c idle_in_transaction_session_timeout=5s'

for sql_path in /database/migrations/[0-9][0-9][0-9][0-9][0-9][0-9]_*.sql; do
  filename="$(basename "$sql_path")"
  version_text="${filename%%_*}"
  version="$(printf '%s' "$version_text" | sed 's/^0*//')"
  [ -n "$version" ] || version=0
  [ "$version" -le "$maximum_version" ] || continue
  name="${filename#*_}"
  name="${name%.sql}"
  checksum="$(sha256sum "$sql_path" | awk '{print $1}')"

  history_exists="$(psql -X -h 127.0.0.1 -U "$ATLAS_POSTGRES_MIGRATION_USER" -d "$database_name" -Atqc "SELECT to_regclass('atlas_foundation.schema_migrations') IS NOT NULL")"
  if [ "$history_exists" = "t" ]; then
    applied_checksum="$(psql -X -h 127.0.0.1 -U "$ATLAS_POSTGRES_MIGRATION_USER" -d "$database_name" -Atqc "SELECT checksum FROM atlas_foundation.schema_migrations WHERE version = $version")"
    if [ -n "$applied_checksum" ]; then
      [ "$applied_checksum" = "$checksum" ] || { echo "released migration checksum mismatch at version $version" >&2; exit 1; }
      continue
    fi
  fi

  psql -X -h 127.0.0.1 -U "$ATLAS_POSTGRES_MIGRATION_USER" -d "$database_name" \
    -v ON_ERROR_STOP=1 \
    -1 -f "$sql_path" \
    -c "INSERT INTO atlas_foundation.schema_migrations(version, name, checksum) VALUES ($version, '$name', '$checksum')"
done

unset PGPASSWORD PGOPTIONS
echo "database_migrations=PASS target=$database_name"
