#! /bin/bash
set -e
# ---------------------------------------------------------------------
# -- Setting permissions. The "postgres" user should be the one
# --  that has the permissions like a user provided by AWS
# ---------------------------------------------------------------------
export PGDATABASE=postgres
export PGUSER=aws_superuser

export ADMIN_USERNAME=${DB_USERNAME:-postgres}
export ADMIN_PASSWORD=${DB_PASSWORD:-test1234}

psql -c "CREATE ROLE postgres;"
psql -c "ALTER ROLE postgres WITH CREATEDB CREATEROLE REPLICATION;"
psql -c "CREATE ROLE \"${ADMIN_USERNAME}\" WITH LOGIN PASSWORD '${ADMIN_USERNAME}' CREATEDB CREATEROLE;"
psql -c "GRANT postgres TO \"${ADMIN_USERNAME}\";"
psql -c "REVOKE ALL ON pg_database FROM public;"
psql -c "REVOKE ALL ON pg_authid FROM public;"
psql -c "ALTER ROLE aws_superuser WITH NOLOGIN;"
