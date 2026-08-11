#! /bin/bash
set -e
# ---------------------------------------------------------------------
# -- Setting permissions. The "postgres" user should be the one
# --  that has the permissions like a user provided by Azure
# ---------------------------------------------------------------------
export PGDATABASE=postgres
export PGUSER=az_admin
export ADMIN_USERNAME=${DB_USERNAME:-postgres}
export ADMIN_PASSWORD=${DB_PASSWORD:-test1234}

psql -c "CREATE USER \"${ADMIN_USERNAME}\" WITH PASSWORD '${ADMIN_PASSWORD}' CREATEDB CREATEROLE;"
psql -c "GRANT pg_monitor TO \"az_admin\";"
psql -c "GRANT az_admin TO \"${ADMIN_USERNAME}\";"
psql -c "GRANT pg_read_all_settings TO \"${ADMIN_USERNAME}\";"
psql -c "GRANT pg_read_all_stats TO \"${ADMIN_USERNAME}\";"
psql -c "GRANT pg_stat_scan_tables to \"${ADMIN_USERNAME}\";"
# ---------------------------------------------------------------------
# -- Azure has a complicated extensions setup,
# --  so it doesn't make sense to test them here
# -- Only "uuid-ossd" is tested properly by marking it as a trusted
# ---------------------------------------------------------------------
export PGDATABASE=template1
psql -c "CREATE EXTENSION IF NOT EXISTS pgcrypto;"
