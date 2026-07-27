#!/bin/sh
set -eu

: "${ATLAS_POSTGRES_DB:?required}"
: "${ATLAS_POSTGRES_MIGRATION_USER:?required}"
: "${ATLAS_POSTGRES_MIGRATION_PASSWORD:?required}"

database_name="${ATLAS_SEED_TARGET_DATABASE:-$ATLAS_POSTGRES_DB}"
case "$database_name" in
  atlas_local|atlas_s05_empty_test|atlas_s05_previous_test|atlas_p01_s03_seed_test|atlas_p01_s04_session_test) ;;
  *) echo 'seed target is outside the contained Phase 01 database set' >&2; exit 1 ;;
esac

seed_directory='/database/seeds'
(
  cd "$seed_directory"
  sha256sum -c MANIFEST.sha256 >/dev/null
)

export PGPASSWORD="$ATLAS_POSTGRES_MIGRATION_PASSWORD"
export PGOPTIONS='-c lock_timeout=500ms -c statement_timeout=10s -c idle_in_transaction_session_timeout=10s'
seed_history_exists="$(psql -X -h 127.0.0.1 -U "$ATLAS_POSTGRES_MIGRATION_USER" -d "$database_name" -Atqc "SELECT to_regclass('atlas_foundation.seed_applications') IS NOT NULL")"

apply_seed() {
  seed_id="$1"
  seed_name="$2"
  loader_name="$3"
  seed_path="$seed_directory/$seed_name"
  seed_checksum="$(sha256sum "$seed_path" | awk '{print $1}')"

  if [ "$seed_history_exists" = 't' ]; then
    applied_checksum="$(psql -X -h 127.0.0.1 -U "$ATLAS_POSTGRES_MIGRATION_USER" -d "$database_name" -Atqc "SELECT seed_checksum FROM atlas_foundation.seed_applications WHERE seed_id = '$seed_id'")"
    if [ -n "$applied_checksum" ]; then
      [ "$applied_checksum" = "$seed_checksum" ] || {
        echo "released Phase 01 seed checksum mismatch for $seed_id" >&2
        exit 1
      }
      return
    fi
  fi

  seed_document="$(tr -d '\r\n' <"$seed_path")"
  psql -X -h 127.0.0.1 -U "$ATLAS_POSTGRES_MIGRATION_USER" -d "$database_name" \
    -v ON_ERROR_STOP=1 \
    -v seed_checksum="$seed_checksum" \
    -v seed_document="$seed_document" \
    -1 -f "$seed_directory/$loader_name" >/dev/null
}

apply_seed 'atlas-phase01-identity-v1' '000001_phase_01_identity.json' 'load-phase-01-identity.sql'
apply_seed 'atlas-phase01-identity-policy-v2' '000002_phase_01_policy.json' 'load-phase-01-policy.sql'

unset PGPASSWORD PGOPTIONS

echo "phase01_identity_seed=PASS target=$database_name"
