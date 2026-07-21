#!/bin/sh
set -eu

export PGPASSWORD="$ATLAS_POSTGRES_MIGRATION_PASSWORD"
query() {
  psql -X -h 127.0.0.1 -U "$ATLAS_POSTGRES_MIGRATION_USER" -d "$ATLAS_POSTGRES_DB" -v ON_ERROR_STOP=1 -Atqc "$1"
}

[ "$(query 'SELECT count(*) FROM atlas_foundation.schema_migrations')" = '2' ]
[ "$(query "SELECT count(*) FROM atlas_foundation.recovery_probe WHERE marker_id = 's05-pitr-marker' AND marker_value = 'present-at-recovery-target'")" = '1' ]
[ "$(query 'SELECT pg_is_in_recovery()')" = 'f' ]
[ "$(query "SELECT checksum FROM atlas_foundation.schema_migrations WHERE version = 2")" = '94fdc5112a045e595ee0a6300b8e7cc50b64e09a60a562336011f176283c1dc6' ]

unset PGPASSWORD
echo 'database_isolated_pitr_restore=PASS'
