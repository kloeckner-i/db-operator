#!/bin/sh

TESTDATA="$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1)"
echo ${TESTDATA} > /tmp/checkdata

retry=5
interval=10

for i in `seq 1 $retry`
do
    sleep $interval
    if [ ! -f "${POSTGRES_PASSWORD_FILE}" ]; then
        echo "Password file does not exists"
        exit 1;
    else
        POSTGRES_PASSWORD=$(cat ${POSTGRES_PASSWORD_FILE})
    fi

    echo "writing data into postgres database..."
    PGPASSWORD=${POSTGRES_PASSWORD} psql -h ${POSTGRES_HOST} -U ${POSTGRES_USERNAME} ${POSTGRES_DB} \
    -c "CREATE TABLE IF NOT EXISTS test (no serial PRIMARY KEY, data VARCHAR (50) NOT NULL); INSERT INTO test (data) VALUES('${TESTDATA}');" \
    && break
done