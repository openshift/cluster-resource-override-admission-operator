#!/usr/bin/env bash

set -euo pipefail

# TODO(maxcao13): This is hack script to generate an OLM bundle
# We should remove it if we migrate to operator-sdk project and use `make bundle` instead

# Required: OPERATOR_IMG, OPERAND_IMG, BUNDLE_IMG environment variables
# Optional: SKIP_BUILD
#
# This script does a number of steps.
# 1. Builds the binary, builds the operator image (if enabled), and pushes it to your specified repo.
# 2. Generates an OLM bundle with the specified operator and operand img.
# 3. Builds and pushes a bundle image, if SKIP_BUILD is not true.
# 
# e.g.,
# 
#   OPERATOR_IMG=quay.io/macao/clusterresourceoverride-operator:VERSION \
#   OPERAND_IMG=quay.io/macao/clusterresourceoverride:VERSION \
#   BUNDLE_IMG=quay.io/macao/cro-bundle:VERSION \
#   ./hack/generate-bundle.sh

operator_img=${OPERATOR_IMG:-"quay.io/placeholder/operator:1.0"}
operand_img=${OPERAND_IMG:-"quay.io/placeholder/operand:1.0"}
bundle_img=${BUNDLE_IMG:-"quay.io/placeholder/bundle:1.0"}
skip_build=${SKIP_BUILD:-"false"}

SCRIPT_DIR=$(dirname "$(realpath "$0")")
ROOT_DIR=$SCRIPT_DIR/..

# Build and push the operator image
if [ "$skip_build" == "false" ]; then
    make build
    podman build -t "$operator_img" -f ./images/dev/Dockerfile.dev .
    podman push "$operator_img"
fi

rm -rf bundle

# Aggregate files by --- separators
awk 'FNR==1 && NR!=1 {print "---"} {print}' artifacts/deploy/* manifests/stable/* | operator-sdk generate bundle \
    --package clusterresourceoverride-operator \
    --default-channel stable \
    --channels stable

CSV_BUNDLE_PATH=$ROOT_DIR/bundle/manifests/clusterresourceoverride-operator.clusterserviceversion.yaml

if ! [ -e "$CSV_BUNDLE_PATH" ]; then
    echo "File $CSV_BUNDLE_PATH doesn't exist." ; exit 1
fi

# Update the CSV bundle with correct image references
sed -e "s|CLUSTERRESOURCEOVERRIDE_OPERATOR_IMAGE|$operator_img|g" \
    -e "s|CLUSTERRESOURCEOVERRIDE_OPERAND_IMAGE|$operand_img|g" \
    "$CSV_BUNDLE_PATH" -i

# Validate the bundle
operator-sdk bundle validate ./bundle

# Build and push the bundle image
if [ "$skip_build" == "false" ]; then
    podman build -t "$bundle_img" -f bundle.Dockerfile .
    podman push "$bundle_img"
fi
