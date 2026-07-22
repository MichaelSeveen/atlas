#!/bin/sh
set -eu

backup_root='/backups/current'
target_file='/backups/recovery-target.txt'
marker_id='s05-pitr-marker'

[ -d /backups ]
[ -d /wal-archive ]
mkdir -p "$backup_root"
find "$backup_root" -mindepth 1 -depth -delete
rm -f "$target_file"

started="$(date +%s)"
PGPASSWORD="$ATLAS_POSTGRES_BACKUP_PASSWORD" pg_basebackup -X stream -c fast -D "$backup_root" -h 127.0.0.1 -U atlas_backup --manifest-checksums=SHA256 >/dev/null
pg_verifybackup "$backup_root" >/dev/null
backup_seconds="$(( $(date +%s) - started ))"

run_migration_sql() {
  PGPASSWORD="$ATLAS_POSTGRES_MIGRATION_PASSWORD" psql -X -h 127.0.0.1 -U atlas_migration -d "$ATLAS_POSTGRES_DB" -v ON_ERROR_STOP=1 -Atqc "$1"
}
run_bootstrap_sql() {
  PGPASSWORD="$ATLAS_POSTGRES_PASSWORD" psql -X -h 127.0.0.1 -U "$ATLAS_POSTGRES_USER" -d "$ATLAS_POSTGRES_DB" -v ON_ERROR_STOP=1 -Atqc "$1"
}
wait_for_archive() {
  wal_name="$1"
  deadline="$(( $(date +%s) + 30 ))"
  while [ ! -f "/wal-archive/$wal_name" ]; do
    [ "$(date +%s)" -lt "$deadline" ] || { echo 'WAL archive deadline exceeded' >&2; exit 1; }
    sleep 1
  done
}

run_migration_sql "INSERT INTO atlas_foundation.recovery_probe(marker_id, marker_value) VALUES ('$marker_id', 'present-at-recovery-target') ON CONFLICT (marker_id) DO UPDATE SET marker_value = EXCLUDED.marker_value, recorded_at = transaction_timestamp()" >/dev/null
insert_wal="$(run_bootstrap_sql "SELECT pg_walfile_name(pg_switch_wal())")"
wait_for_archive "$insert_wal"
recovery_target="$(run_bootstrap_sql "SELECT clock_timestamp()")"
printf '%s\n' "$recovery_target" >"$target_file"
sleep 1
run_migration_sql "DELETE FROM atlas_foundation.recovery_probe WHERE marker_id = '$marker_id'" >/dev/null
delete_wal="$(run_bootstrap_sql "SELECT pg_walfile_name(pg_switch_wal())")"
wait_for_archive "$delete_wal"

active_marker_count="$(run_migration_sql "SELECT count(*) FROM atlas_foundation.recovery_probe WHERE marker_id = '$marker_id'")"
[ "$active_marker_count" = '0' ]
echo "database_base_backup=PASS duration_seconds=$backup_seconds"
echo 'database_wal_archive=PASS'
