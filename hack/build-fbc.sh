#!/usr/bin/env bash

set -euo pipefail

# Required: CATALOG_REPO, BUNDLE_REPO, PREV_VERSION environment variables
#
# This script builds an OLM file-based-catalog and outputs it to
# catalog/cro-fbc.yaml.
#
# # Assumes BUNDLE_REPO refers to an image repository that contains
# a CRO bundle that points to the current branch version, and assumes
# there exists a bundle that points to the PREV_VERSION.
# 
# Note that there should be no tag attached to the environment vars.
#
# This script is intended to be used alongside ./hack/generate-bundle.sh 
# for building catalogs with multiple bundles for testing operator upgrades.
# If needed in the future, it can be allowed to specify a custom catalog template.
# 
# e.g.,
# 
#   CATALOG_REPO=quay.io/macao/cro-catalog \
#   BUNDLE_REPO=quay.io/macao/cro-bundle \
#   PREV_VERSION=4.16
#   ./hack/build-fbc.sh

REQUIRED_ENV_VARS=("CATALOG_REPO" "BUNDLE_REPO" "PREV_VERSION")
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

# Get version from Makefile
VERSION=$(awk -F ':=' '/^IMAGE_VERSION/ {print $2}' "$ROOT_DIR/Makefile" | xargs)

TEMPLATE="
schema: olm.template.basic
entries:
  - schema: olm.package
    name: clusterresourceoverride-operator
    defaultChannel: stable
  - schema: olm.channel
    package: clusterresourceoverride-operator
    name: stable
    entries:
      - name: clusterresourceoverride-operator.v$PREV_VERSION.0
      - name: clusterresourceoverride-operator.v$VERSION.0
        replaces: clusterresourceoverride-operator.v$PREV_VERSION.0
  - schema: olm.bundle
    image: $BUNDLE_REPO:v$VERSION
  - schema: olm.bundle
    image: $BUNDLE_REPO:v$PREV_VERSION
"

# Generate the fbc
mkdir -p catalog
echo "$TEMPLATE" | opm alpha render-template basic -oyaml >| catalog/cro-fbc.yaml

# Build and push the catalog image
podman build -t "$CATALOG_REPO:v$VERSION" -f "$ROOT_DIR"/catalog.Dockerfile "$ROOT_DIR"
podman push "$CATALOG_REPO:v$VERSION"
