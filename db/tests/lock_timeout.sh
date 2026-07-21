#!/bin/sh
set -eu

holder_output='/tmp/atlas-s05-lock-holder.out'
blocked_output='/tmp/atlas-s05-lock-blocked.out'
export PGPASSWORD="$ATLAS_POSTGRES_MIGRATION_PASSWORD"

cleanup() {
  rm -f "$holder_output" "$blocked_output"
}
trap cleanup EXIT INT TERM

psql -X -h 127.0.0.1 -U atlas_migration -d "$ATLAS_POSTGRES_DB" -v ON_ERROR_STOP=1 \
  -c "BEGIN; LOCK TABLE atlas_foundation.permission_probe IN ACCESS EXCLUSIVE MODE; SELECT pg_sleep(3); COMMIT;" >"$holder_output" 2>&1 &
holder_pid=$!
sleep 1

started="$(date +%s)"
if PGOPTIONS='-c lock_timeout=500ms -c statement_timeout=5s' psql -X -h 127.0.0.1 -U atlas_migration -d "$ATLAS_POSTGRES_DB" -v ON_ERROR_STOP=1 \
  -1 -c 'ALTER TABLE atlas_foundation.permission_probe ADD COLUMN lock_canary text' >"$blocked_output" 2>&1; then
  echo 'long-lock migration canary unexpectedly succeeded' >&2
  kill "$holder_pid" >/dev/null 2>&1 || true
  wait "$holder_pid" >/dev/null 2>&1 || true
  exit 1
fi
elapsed="$(( $(date +%s) - started ))"
[ "$elapsed" -lt 3 ]
wait "$holder_pid"

column_count="$(psql -X -h 127.0.0.1 -U atlas_migration -d "$ATLAS_POSTGRES_DB" -Atqc "SELECT count(*) FROM information_schema.columns WHERE table_schema='atlas_foundation' AND table_name='permission_probe' AND column_name='lock_canary'")"
[ "$column_count" = '0' ]
psql -X -h 127.0.0.1 -U atlas_migration -d "$ATLAS_POSTGRES_DB" -Atqc 'SELECT count(*) FROM atlas_foundation.schema_migrations' >/dev/null

unset PGPASSWORD
echo "database_long_lock_abort=PASS elapsed_seconds=$elapsed"
