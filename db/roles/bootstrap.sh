#!/bin/sh
set -eu

: "${ATLAS_POSTGRES_DB:?required}"
: "${ATLAS_POSTGRES_USER:?required}"
: "${ATLAS_POSTGRES_PASSWORD:?required}"
: "${ATLAS_POSTGRES_MIGRATION_USER:?required}"
: "${ATLAS_POSTGRES_MIGRATION_PASSWORD:?required}"
: "${ATLAS_POSTGRES_API_USER:?required}"
: "${ATLAS_POSTGRES_API_PASSWORD:?required}"
: "${ATLAS_POSTGRES_WORKER_USER:?required}"
: "${ATLAS_POSTGRES_WORKER_PASSWORD:?required}"
: "${ATLAS_POSTGRES_REPORTING_USER:?required}"
: "${ATLAS_POSTGRES_REPORTING_PASSWORD:?required}"
: "${ATLAS_POSTGRES_BREAK_GLASS_USER:?required}"
: "${ATLAS_POSTGRES_BREAK_GLASS_PASSWORD:?required}"
: "${ATLAS_POSTGRES_BACKUP_USER:?required}"
: "${ATLAS_POSTGRES_BACKUP_PASSWORD:?required}"

[ "$ATLAS_POSTGRES_MIGRATION_USER" = "atlas_migration" ]
[ "$ATLAS_POSTGRES_API_USER" = "atlas_api" ]
[ "$ATLAS_POSTGRES_WORKER_USER" = "atlas_worker" ]
[ "$ATLAS_POSTGRES_REPORTING_USER" = "atlas_reporting_read" ]
[ "$ATLAS_POSTGRES_BREAK_GLASS_USER" = "atlas_break_glass" ]
[ "$ATLAS_POSTGRES_BACKUP_USER" = "atlas_backup" ]

export PGPASSWORD="$ATLAS_POSTGRES_PASSWORD"
psql -X -h 127.0.0.1 -U "$ATLAS_POSTGRES_USER" -d "$ATLAS_POSTGRES_DB" -v ON_ERROR_STOP=1 -f /database/roles/bootstrap.sql
unset PGPASSWORD
echo 'database_role_bootstrap=PASS'
