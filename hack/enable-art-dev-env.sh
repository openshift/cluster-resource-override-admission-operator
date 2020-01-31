#!/usr/bin/env bash

set -e

#
# This script disables the default operator source(s) shipped with OperatorHub in OpenShift
# and enables 'redhat-operators-art' operator source ( dev env for ART, where you can
# validate and check your changes to the operator manifests ).
# For more information, check https://docs.google.com/document/d/1t81RSsZbUoGO4r5OgJ1bqAESKt2fM25MvV6pcgQUPSk/edit#heading=h.tma2itt39x73
#

if [ "${QUAY_ROBOT_NAME}" == "" ]; then
  echo "QUAY_ROBOT_NAME env must be set"
  exit 1
fi

if [ "${QUAY_ROBOT_TOKEN}" == "" ]; then
  echo "QUAY_ROBOT_TOKEN env must be set"
  exit 1
fi

RESPONSE=$(curl -sH "Content-Type: application/json" -XPOST https://quay.io/cnr/api/v1/users/login -d '{ "user": { "username": "'"${QUAY_ROBOT_NAME}"'","password": "'"${QUAY_ROBOT_TOKEN}"'"}}')
QUAY_TOKEN=$(jq -e -r '.token' <<< "${RESPONSE}" || echo "")

if [ "${QUAY_TOKEN}" == "" ]; then
  echo "failed to retrieve token response=${RESPONSE}"
  exit 1
fi

cat <<EOF | kubectl apply -f -
apiVersion: config.openshift.io/v1
kind: OperatorHub
metadata:
  name: cluster
spec:
  disableAllDefaultSources: true
EOF

cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Secret
metadata:
  name: marketplacesecret
  namespace: openshift-marketplace
type: Opaque
stringData:
  token: "${QUAY_TOKEN}"
EOF

cat <<EOF | kubectl apply -f -
apiVersion: operators.coreos.com/v1
kind: OperatorSource
metadata:
  name: art-applications
  namespace: openshift-marketplace
spec:
  type: appregistry
  endpoint: https://quay.io/cnr
  registryNamespace: redhat-operators-art
  authorizationToken:
    secretName: marketplacesecret
EOF
