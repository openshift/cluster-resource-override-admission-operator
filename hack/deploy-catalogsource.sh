#!/usr/bin/env bash

set -euo pipefail

# Required: CATALOG_IMG environment variables
#
# This script applies a catalog source to your cluster containing
# the built catalog image.
#
# e.g., CATALOG_IMG=quay.io/macao/cro-catalog:v4.19 ./hack/deploy-catalogsource.sh

DISPLAY_NAME="Cluster Resource Override Operator dev catalog"
PKG_NAME=clusterresourceoverride-operator

# Get namespace of the installed operator if it exists (cleanup)
NAMESPACE=$(oc get subscriptions --all-namespaces -o json | \
  jq -r "[.items[] | select(.spec.name == \"$PKG_NAME\") | .metadata.namespace] | first // \"\"")
if [ -n "$NAMESPACE" ]; then
  operator-sdk cleanup -n "$NAMESPACE" "$PKG_NAME"
fi

oc apply -f - <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: cro-fbc
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: $CATALOG_IMG
  displayName: $DISPLAY_NAME
  publisher: autoscaling-team
  updateStrategy:
    registryPoll:
      interval: 10m
EOF

echo
echo -n "Waiting for package manifest to become available."
first=1
while [ $first = 1 ] || sleep 5; do
  first=0
  [ "$(oc get packagemanifest -n openshift-marketplace -o json | \
    jq -r "[ .items[] | select(.metadata.name==\"$PKG_NAME\") | select(.status.catalogSourceDisplayName == \"$DISPLAY_NAME\") ] | length")" -gt 0 ] && break
  echo -n .
done
echo " done"
