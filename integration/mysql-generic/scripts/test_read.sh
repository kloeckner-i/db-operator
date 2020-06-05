#!/bin/sh

retry=3
interval=5

for i in `seq 1 $retry`
do
    sleep $interval
    if [ ! -f "${MYSQL_PASSWORD_FILE}" ]; then
        echo "Password file does not exists"
        exit 1;
    else
        MYSQL_PASSWORD=$(cat ${MYSQL_PASSWORD_FILE})
    fi

    TESTDATA="$(cat /tmp/checkdata)"

    echo "reading data from mysql..."
    FOUNDDATA=$(mysql \
            -h ${MYSQL_HOST} \
            -u ${MYSQL_USERNAME} \
            -p${MYSQL_PASSWORD} ${MYSQL_DB} \
            -e "SELECT data FROM test WHERE data = '${TESTDATA}';")

    echo "$FOUNDDATA" | grep "$TESTDATA" && break;
done