#!/bin/sh
set -eu

expected_pgdata='/var/lib/postgresql/restore-data'
[ "${PGDATA:-}" = "$expected_pgdata" ] || { echo 'restore PGDATA is outside its dedicated volume' >&2; exit 1; }
[ -f /backups/current/PG_VERSION ]
[ -f /backups/current/backup_manifest ]
[ -f /backups/recovery-target.txt ]
[ -d /wal-archive ]

recovery_target="$(cat /backups/recovery-target.txt)"
case "$recovery_target" in
  [0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]*[+]*) ;;
  *) echo 'recovery target is malformed' >&2; exit 1 ;;
esac

mkdir -p "$PGDATA"
find "$PGDATA" -mindepth 1 -depth -delete
cp -a /backups/current/. "$PGDATA/"
chown -R postgres:postgres "$PGDATA"
chmod 0700 "$PGDATA"
cat >>"$PGDATA/postgresql.auto.conf" <<EOF
restore_command = 'cp /wal-archive/%f %p'
recovery_target_time = '$recovery_target'
recovery_target_action = 'promote'
EOF
touch "$PGDATA/recovery.signal"
chown postgres:postgres "$PGDATA/postgresql.auto.conf" "$PGDATA/recovery.signal"

exec docker-entrypoint.sh postgres -c port=5432 -c archive_mode=off -c listen_addresses='*'
