#!/bin/sh

retry=3
interval=5

for i in `seq 1 $retry`
do
    sleep $interval
    if [ ! -f "${POSTGRES_PASSWORD_FILE}" ]; then
        echo "Password file does not exists"
        exit 1;
    else
        POSTGRES_PASSWORD=$(cat ${POSTGRES_PASSWORD_FILE})
    fi

    TESTDATA="$(cat /tmp/checkdata)"

    echo "reading data from postgres..."
    FOUNDDATA=$(PGPASSWORD=${POSTGRES_PASSWORD} psql \
            -h ${POSTGRES_HOST} \
            -U ${POSTGRES_USERNAME} \
            ${POSTGRES_DB} \
            -c "SELECT data FROM test WHERE data = '${TESTDATA}';")

    echo "$FOUNDDATA" | grep "$TESTDATA" && break;
done