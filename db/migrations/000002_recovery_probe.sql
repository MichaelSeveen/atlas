CREATE TABLE atlas_foundation.recovery_probe (
    marker_id text PRIMARY KEY CHECK (length(marker_id) BETWEEN 1 AND 64),
    marker_value text NOT NULL CHECK (length(marker_value) BETWEEN 1 AND 128),
    recorded_at timestamptz NOT NULL DEFAULT transaction_timestamp()
);

REVOKE INSERT, UPDATE, DELETE ON atlas_foundation.recovery_probe FROM atlas_api, atlas_worker;
