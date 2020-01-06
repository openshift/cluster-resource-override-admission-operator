#!/bin/bash

set -ex

which jq &>/dev/null || { echo "Please install jq (https://stedolan.github.io/jq/)."; exit 1; }

IMAGE_URL_FILE_PATH=$1
if [ "${IMAGE_URL_FILE_PATH}" == "" ]; then
  echo "Must specify path to the file that has the operator/operand image URL(s)"
  exit 1
fi

DEPLOYMENT_YAML=$2
if [ "${DEPLOYMENT_YAML}" == "" ]; then
  echo "Must specify a path to the yaml file for the Deployment object"
  exit 1
fi

OPERATOR_IMAGE=$(cat ${IMAGE_URL_FILE_PATH} | jq '."operator"')
OPERAND_IMAGE=$(cat ${IMAGE_URL_FILE_PATH} | jq '."operand"')

sed "s,CLUSTERRESOURCEOVERRIDE_OPERATOR_IMAGE,${OPERATOR_IMAGE},g" -i "${DEPLOYMENT_YAML}"
sed "s,CLUSTERRESOURCEOVERRIDE_OPERAND_IMAGE,${OPERAND_IMAGE},g" -i "${DEPLOYMENT_YAML}"
