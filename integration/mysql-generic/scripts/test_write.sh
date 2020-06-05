#!/bin/sh

TESTDATA="$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1)"
echo ${TESTDATA} > /tmp/checkdata

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

    echo "writing data into mysql database..."
    mysql -h ${MYSQL_HOST} -u ${MYSQL_USERNAME} -p${MYSQL_PASSWORD} ${MYSQL_DB} \
    -e "CREATE TABLE IF NOT EXISTS test (no INT NOT NULL AUTO_INCREMENT PRIMARY KEY, data VARCHAR(100)); INSERT INTO test (data) VALUES('${TESTDATA}');"\
    && break
done