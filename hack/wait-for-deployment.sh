#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

util::await_operator_deployment_create() {
    local namespace="$1"
    local name="$2"
    local retries="${3:-50}"
    local output

    until [[ "${retries}" -le "0" ]]; do
        output=$(kubectl get deployment -n "${namespace}" "${name}" -o jsonpath='{.metadata.name}' 2>/dev/null || echo "waiting for olm to deploy the operator")

        if [ "${output}" = "${name}" ] ; then
            echo "${namespace}/${name} has been created" >&2
            return 0
        fi

        retries=$((retries - 1))
        echo "${output} - remaining attempts: ${retries}" >&2

        sleep 3
    done

    echo "error - olm has not created the deployment yet ${namespace}/${name}" >&2
    return 1
}

NAMESPACE=$1
DEPLOYMENT_NAME=$2
WAIT_TIME=$3

exitcode=$(util::await_operator_deployment_create "${NAMESPACE}" "${DEPLOYMENT_NAME}" "${WAIT_TIME}")
exit $exitcode
