#!/bin/sh -e

OPERATOR_NAMESPACE="operator"
TEST_NAMESPACE="test"

retry=10
interval=10

case $TEST_K8S in
    "microk8s")
        export HELM_CMD="sudo microk8s.helm"
        export KUBECTL_CMD="sudo microk8s.kubectl"
        ;;
    *)
        export HELM_CMD="helm"
        export KUBECTL_CMD="kubectl"
esac

check_instance_status() {
    echo "[DbInstance] checking"
    for i in `seq 1 $retry`
    do
        count=$($KUBECTL_CMD get dbin -o json | jq '.items | length')
        if [ $count -eq 0 ]; then
            echo "DbInstance resource doesn't exists"
            continue;
        fi

        for n in `seq 1 $count`
        do
            item=$($KUBECTL_CMD get dbin -o json | jq -c .items[$n-1])
            if [ ! $(echo $item | jq '.status.status') ]; then
                echo "[DbInstance] $(echo $item | jq -r '.metadata.name') status false"
                break; # re check with interval
            fi

            echo "[DbInstance] $(echo $item | jq -r '.metadata.name') status true"
            if [ $n -eq $count ]; then
                echo "[DbInstance] Status OK!"
                return 0 # finish check
            fi
        done
        echo "Retrying after $interval seconds..."
        sleep $interval; # retry with interval
    done # end retry

    echo "DbInstance not healthy"
    exit 1 # return false
}

create_test_resources() {
    echo "[Test] creating"
    $HELM_CMD upgrade --install --namespace ${TEST_NAMESPACE} test-mysql integration/mysql \
    && $HELM_CMD upgrade --install --namespace ${TEST_NAMESPACE} test-pg integration/postgres \
    && echo "[Test] created"
}

check_databases_status() {
    echo "[Database] checking"
    for i in `seq 1 $retry`
    do
        count=$($KUBECTL_CMD get db -n ${TEST_NAMESPACE} -o json | jq '.items | length')
        if [ $count -eq 0 ]; then
            echo "Database resource doesn't exists"
            continue;
        fi

        for n in `seq 1 $count`
        do
            item=$($KUBECTL_CMD get db -n ${TEST_NAMESPACE} -o json | jq -c .items[$n-1])
            if [ "$(echo $item | jq -r '.status.phase')" != "Ready" ]; then
                echo "[Database] $(echo $item | jq -r '.metadata.name') status false"
                break; # re check with interval
            fi
            echo "[Database] $(echo $item | jq -r '.metadata.name') status true"
            if [ $n -eq $count ]; then
                echo "[Database] Status OK!"
                return 0 # finish check
            fi
        done
        echo "Retrying after $interval seconds..."
        sleep $interval; # retry with interval
    done
    echo "Database not healthy"
    exit 1 # return false
}

run_test() {
    echo "[Test] testing read write to database"
    $HELM_CMD test test-mysql && $HELM_CMD test test-pg \
    && echo "[Test] OK!"
}

delete_databases() {
    echo "[Database] deleting"
    $KUBECTL_CMD delete db my-db-test -n ${TEST_NAMESPACE} && $KUBECTL_CMD delete db pg-db-test -n ${TEST_NAMESPACE} \
    && echo "[Database] deleted!"
}

check_databases_deleted() {
    echo "[Database] checking deleted"
    for i in `seq 1 $retry`
    do
        count=$($KUBECTL_CMD get db -n ${TEST_NAMESPACE} -o json | jq '.items | length')
        if [ $count -ne 0 ]; then
            echo "[Database] $(echo $item | jq -r '.metadata.name') not deleted"
            continue;
        fi
        echo "[Database] All deleted!"
        return 0 # all good
    done
    echo "[Database] not deleted"
    exit 1 # return false
}

create_test_resources
check_instance_status
check_databases_status
run_test
delete_databases
check_databases_deleted