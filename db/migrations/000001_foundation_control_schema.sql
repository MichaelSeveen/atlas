CREATE SCHEMA atlas_foundation AUTHORIZATION atlas_migration;
REVOKE ALL ON SCHEMA atlas_foundation FROM PUBLIC;
GRANT USAGE ON SCHEMA atlas_foundation TO atlas_api, atlas_worker, atlas_reporting_read;

CREATE TABLE atlas_foundation.schema_migrations (
    version bigint PRIMARY KEY CHECK (version > 0),
    name text NOT NULL CHECK (name ~ '^[a-z][a-z0-9_]{2,63}$'),
    checksum character(64) NOT NULL CHECK (checksum ~ '^[0-9a-f]{64}$'),
    applied_at timestamptz NOT NULL DEFAULT transaction_timestamp()
);

CREATE TABLE atlas_foundation.permission_probe (
    probe_key text PRIMARY KEY CHECK (length(probe_key) BETWEEN 1 AND 64),
    marker text NOT NULL CHECK (length(marker) BETWEEN 1 AND 128)
);

REVOKE ALL ON ALL TABLES IN SCHEMA atlas_foundation FROM PUBLIC;
GRANT SELECT ON atlas_foundation.schema_migrations TO atlas_api, atlas_worker, atlas_reporting_read;
GRANT SELECT, INSERT, UPDATE, DELETE ON atlas_foundation.permission_probe TO atlas_api, atlas_worker;
GRANT SELECT ON atlas_foundation.permission_probe TO atlas_reporting_read;

ALTER DEFAULT PRIVILEGES FOR ROLE atlas_migration IN SCHEMA atlas_foundation REVOKE ALL ON TABLES FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE atlas_migration IN SCHEMA atlas_foundation GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO atlas_api, atlas_worker;
ALTER DEFAULT PRIVILEGES FOR ROLE atlas_migration IN SCHEMA atlas_foundation GRANT SELECT ON TABLES TO atlas_reporting_read;
