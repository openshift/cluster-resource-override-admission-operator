#!/usr/bin/env bash

set -euo pipefail

# Required: OPERATOR_IMG, OPERAND_IMG, BUNDLE_IMG environment variables
#
# This script does a number of steps.
# 1. Builds the binary, builds the operator image, and pushes it to your specified repo.
# 2. Generates an OLM bundle with the specified operator and operand img.
# 3. Builds and pushes a bundle image.
# 
# e.g.,
# 
#   OPERATOR_IMG=quay.io/macao/clusterresourceoverride-operator:VERSION \
#   OPERAND_IMG=quay.io/macao/clusterresourceoverride:VERSION \
#   BUNDLE_IMG=quay.io/macao/cro-bundle:VERSION \
#   ./hack/build-bundle.sh

REQUIRED_ENV_VARS=("OPERATOR_IMG" "OPERAND_IMG" "BUNDLE_IMG")
check_env_vars() {
    local REQUIRED_VARS=("$@")
    for VAR_NAME in "${REQUIRED_VARS[@]}"; do
        if [ -z "${!VAR_NAME:-}" ]; then
            echo "Error: $VAR_NAME environment variable is not set." ; exit 1
        fi
    done
}

check_env_vars "${REQUIRED_ENV_VARS[@]}"

SCRIPT_DIR=$(dirname "$(realpath "$0")")
ROOT_DIR=$SCRIPT_DIR/..

# Build and push the operator image
make build
podman build -t "$OPERATOR_IMG" -f ./images/dev/Dockerfile.dev .
podman push "$OPERATOR_IMG"

rm -rf bundle

# Aggregate files by --- separators
awk 'FNR==1 && NR!=1 {print "---"} {print}' artifacts/deploy/* manifests/stable/* | operator-sdk generate bundle \
    --package clusterresourceoverride-operator \
    --default-channel stable \
    --channels stable

trap "rm -rf bundle" EXIT

CSV_BUNDLE_PATH=$ROOT_DIR/bundle/manifests/clusterresourceoverride-operator.clusterserviceversion.yaml

if ! [ -e "$CSV_BUNDLE_PATH" ]; then
    echo "File $CSV_BUNDLE_PATH doesn't exist." ; exit 1
fi

# Update the CSV bundle with correct image references
sed -e "s|CLUSTERRESOURCEOVERRIDE_OPERATOR_IMAGE|$OPERATOR_IMG|g" \
    -e "s|CLUSTERRESOURCEOVERRIDE_OPERAND_IMAGE|$OPERAND_IMG|g" \
    "$CSV_BUNDLE_PATH" -i

# Validate the bundle
operator-sdk bundle validate ./bundle

# Build and push the bundle image
podman build -t "$BUNDLE_IMG" -f bundle.Dockerfile .
podman push "$BUNDLE_IMG"
