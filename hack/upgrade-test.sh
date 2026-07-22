#!/usr/bin/env bash
set -euo pipefail

# Orchestrates a local OLM upgrade test for ClusterResourceOverride.
#
# Required env vars:
#   OLD_BUNDLE  - bundle image for the previous version
#   NEW_BUNDLE  - bundle image for the new version
#   KUBECONFIG  - path to cluster kubeconfig
# 
# Optional env vars:
#   SKIP_CLEANUP - skip cleanup (default: false)
# 
# Usage:
#   OLD_BUNDLE=quay.io/user/cro-bundle:v4.18 \
#   NEW_BUNDLE=quay.io/user/cro-bundle:v5.0 \
#   ./hack/upgrade-test.sh

: "${OLD_BUNDLE:?OLD_BUNDLE must be set}"
: "${NEW_BUNDLE:?NEW_BUNDLE must be set}"
: "${KUBECONFIG:?KUBECONFIG must be set}"

NAMESPACE=openshift-cluster-resource-override

cleanup() {
    if [[ "${SKIP_CLEANUP:-}" == "true" ]]; then
        echo "--- Skipping cleanup (SKIP_CLEANUP=true) ---"
        return
    fi
    echo "--- Cleaning up ---"
    oc delete clusterresourceoverride cluster --ignore-not-found 2>/dev/null || true
    sleep 5
    operator-sdk cleanup -n "$NAMESPACE" clusterresourceoverride-operator 2>/dev/null || true
    oc delete namespace "$NAMESPACE" --ignore-not-found 2>/dev/null || true
}
trap cleanup EXIT

echo "=== Step 1: Install old bundle ==="
oc create namespace "$NAMESPACE" --dry-run=client -o yaml | oc apply -f -
operator-sdk run bundle "$OLD_BUNDLE" -n "$NAMESPACE" \
    --security-context-config restricted --timeout 10m

echo "=== Step 2: Pre-upgrade check (create CR + validate old version) ==="
make e2e-upgrade-pre

echo "=== Step 3: Upgrade to new bundle ==="
operator-sdk run bundle-upgrade "$NEW_BUNDLE" -n "$NAMESPACE" \
    --security-context-config restricted --timeout 10m

echo "=== Step 4: Post-upgrade check (validate migration) ==="
make e2e-upgrade-post

echo "=== Upgrade test passed ==="
