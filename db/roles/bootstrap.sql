\set ON_ERROR_STOP on
\set QUIET on
\getenv database_name ATLAS_POSTGRES_DB
\getenv migration_password ATLAS_POSTGRES_MIGRATION_PASSWORD
\getenv api_password ATLAS_POSTGRES_API_PASSWORD
\getenv worker_password ATLAS_POSTGRES_WORKER_PASSWORD
\getenv reporting_password ATLAS_POSTGRES_REPORTING_PASSWORD
\getenv break_glass_password ATLAS_POSTGRES_BREAK_GLASS_PASSWORD
\getenv backup_password ATLAS_POSTGRES_BACKUP_PASSWORD

SELECT format('CREATE ROLE atlas_migration LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD %L', :'migration_password')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'atlas_migration') \gexec
SELECT format('CREATE ROLE atlas_api LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD %L', :'api_password')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'atlas_api') \gexec
SELECT format('CREATE ROLE atlas_worker LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD %L', :'worker_password')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'atlas_worker') \gexec
SELECT format('CREATE ROLE atlas_reporting_read LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD %L', :'reporting_password')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'atlas_reporting_read') \gexec
SELECT format('CREATE ROLE atlas_break_glass LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD %L VALID UNTIL %L', :'break_glass_password', '1970-01-01T00:00:00Z')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'atlas_break_glass') \gexec
SELECT format('CREATE ROLE atlas_backup LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE REPLICATION NOBYPASSRLS PASSWORD %L', :'backup_password')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'atlas_backup') \gexec

SELECT format('ALTER ROLE atlas_migration WITH LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD %L VALID UNTIL %L', :'migration_password', 'infinity') \gexec
SELECT format('ALTER ROLE atlas_api WITH LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD %L VALID UNTIL %L', :'api_password', 'infinity') \gexec
SELECT format('ALTER ROLE atlas_worker WITH LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD %L VALID UNTIL %L', :'worker_password', 'infinity') \gexec
SELECT format('ALTER ROLE atlas_reporting_read WITH LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD %L VALID UNTIL %L', :'reporting_password', 'infinity') \gexec
SELECT format('ALTER ROLE atlas_break_glass WITH LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD %L VALID UNTIL %L', :'break_glass_password', '1970-01-01T00:00:00Z') \gexec
SELECT format('ALTER ROLE atlas_backup WITH LOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE REPLICATION NOBYPASSRLS PASSWORD %L VALID UNTIL %L', :'backup_password', 'infinity') \gexec

GRANT atlas_migration TO atlas_break_glass;

SELECT format('REVOKE CONNECT, TEMPORARY ON DATABASE %I FROM PUBLIC', :'database_name') \gexec
SELECT format('GRANT CONNECT, CREATE ON DATABASE %I TO atlas_migration', :'database_name') \gexec
SELECT format('GRANT CONNECT ON DATABASE %I TO atlas_api, atlas_worker, atlas_reporting_read, atlas_break_glass', :'database_name') \gexec
SELECT format('REVOKE TEMPORARY, CREATE ON DATABASE %I FROM atlas_api, atlas_worker, atlas_reporting_read, atlas_break_glass', :'database_name') \gexec
REVOKE CREATE ON SCHEMA public FROM PUBLIC;

ALTER ROLE atlas_migration SET lock_timeout = '500ms';
ALTER ROLE atlas_migration SET statement_timeout = '5s';
ALTER ROLE atlas_migration SET idle_in_transaction_session_timeout = '5s';
ALTER ROLE atlas_api SET lock_timeout = '250ms';
ALTER ROLE atlas_api SET statement_timeout = '2s';
ALTER ROLE atlas_api SET idle_in_transaction_session_timeout = '5s';
ALTER ROLE atlas_worker SET lock_timeout = '250ms';
ALTER ROLE atlas_worker SET statement_timeout = '5s';
ALTER ROLE atlas_worker SET idle_in_transaction_session_timeout = '5s';
ALTER ROLE atlas_reporting_read SET default_transaction_read_only = on;
ALTER ROLE atlas_reporting_read SET statement_timeout = '5s';
ALTER ROLE atlas_reporting_read SET idle_in_transaction_session_timeout = '5s';

SELECT format('ALTER ROLE atlas_migration IN DATABASE %I SET search_path = atlas_foundation, pg_catalog', :'database_name') \gexec
SELECT format('ALTER ROLE atlas_api IN DATABASE %I SET search_path = atlas_foundation, pg_catalog', :'database_name') \gexec
SELECT format('ALTER ROLE atlas_worker IN DATABASE %I SET search_path = atlas_foundation, pg_catalog', :'database_name') \gexec
SELECT format('ALTER ROLE atlas_reporting_read IN DATABASE %I SET search_path = atlas_foundation, pg_catalog', :'database_name') \gexec
\set QUIET off
