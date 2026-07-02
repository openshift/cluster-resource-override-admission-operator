# OLM Upgrade Testing

How to test an OLM-managed upgrade of ClusterResourceOverride from one
version to another on an OpenShift cluster.

The upgrade uses `operator-sdk run bundle` and `run bundle-upgrade`,
which inject bundles into a transient in-cluster registry. OLM
performs the actual upgrade (InstallPlan, CSV lifecycle, deployment
rollout), but the catalog serving mechanism differs from production.
The SDK constructs a synthetic FBC from bundle annotations and serves
it via an ephemeral pod rather than a real catalog image.

## Prerequisites

- An OpenShift cluster with OLM running (`oc cluster-info` succeeds).
- `podman`, `oc`, `operator-sdk` (v1.42+) installed locally.
- A container registry you can push to (e.g. `quay.io/<user>`).
  Log in with `podman login quay.io` before starting.

You need the following container image repos to be public in your
registry:

| Repo | Purpose |
|------|---------|
| `<user>/clusterresourceoverride-operator` | Operator image |
| `<user>/clusterresourceoverride` | Operand image |
| `<user>/cro-bundle` | OLM bundle image |

## Terminology

| Term | Meaning |
|------|---------|
| OLD_VERSION | The version you are upgrading *from* (e.g. `4.18`). |
| NEW_VERSION | The version you are upgrading *to* (e.g. `5.0`). |
| Operator image | The controller binary (`cluster-resource-override-admission-operator`). |
| Operand image | The admission webhook server (`clusterresourceoverride`). |
| Bundle image | An OCI image containing the CSV + CRD + manifests for one version. |

Throughout this document, substitute your own registry prefix for
`quay.io/$USER`.

---

## Building the bundles

You need two bundle images: one for the old version and one for the new.
`generate-bundle.sh` builds the operator binary, operator image, bundle
manifests, and bundle image in one shot.

### Operand images

The operand (admission webhook server) lives in a separate repo:
[openshift/cluster-resource-override-admission](https://github.com/openshift/cluster-resource-override-admission).

If you only need to test the operator upgrade mechanism, you can reuse
a single existing operand image for both bundles (e.g.
`quay.io/$USER/clusterresourceoverride:dev`).

If you need to test differences between operand binaries across
versions, clone the operand repo and build each version from its
release branch:

```bash
cd /path/to/cluster-resource-override-admission
git checkout release-${OLD_VERSION}
podman build -t quay.io/$USER/clusterresourceoverride:v${OLD_VERSION} .
podman push quay.io/$USER/clusterresourceoverride:v${OLD_VERSION}

git checkout release-${NEW_VERSION}  # or main
podman build -t quay.io/$USER/clusterresourceoverride:v${NEW_VERSION} .
podman push quay.io/$USER/clusterresourceoverride:v${NEW_VERSION}
```

Then use the matching operand tag for each bundle's `OPERAND_IMG`
below.

### Generate the old version bundle

Check out the branch or tag for the old version so the CSV already has
the correct version strings and image references:

```bash
git checkout release-${OLD_VERSION}

OPERATOR_IMG=quay.io/$USER/clusterresourceoverride-operator:v${OLD_VERSION} \
OPERAND_IMG=quay.io/$USER/clusterresourceoverride:v${OLD_VERSION} \
BUNDLE_IMG=quay.io/$USER/cro-bundle:v${OLD_VERSION} \
./hack/generate-bundle.sh
```

### Generate the new version bundle

Switch back to the branch with the new version:

```bash
git checkout main   # or your feature branch

OPERATOR_IMG=quay.io/$USER/clusterresourceoverride-operator:v${NEW_VERSION} \
OPERAND_IMG=quay.io/$USER/clusterresourceoverride:v${NEW_VERSION} \
BUNDLE_IMG=quay.io/$USER/cro-bundle:v${NEW_VERSION} \
./hack/generate-bundle.sh
```

---

## Automated upgrade test

The automated test creates a `ClusterResourceOverride` CR against the
old version, validates that it works, upgrades to the new bundle, then
verifies the CR survived the upgrade: status healthy, operand running,
and webhook still mutating pods.

### Running locally

```bash
OLD_BUNDLE=quay.io/$USER/cro-bundle:v${OLD_VERSION} \
NEW_BUNDLE=quay.io/$USER/cro-bundle:v${NEW_VERSION} \
KUBECONFIG=/path/to/kubeconfig \
make e2e-upgrade-test
```

This runs `hack/upgrade-test.sh`, which orchestrates the full flow:

1. Installs the old bundle with `operator-sdk run bundle`
2. Runs `make e2e-upgrade-pre` (Go test: creates CR, validates old version)
3. Upgrades with `operator-sdk run bundle-upgrade`
4. Runs `make e2e-upgrade-post` (Go test: validates CR migration)
5. Cleans up

### Running individual steps

You can run the pre and post checks independently against a cluster
that already has the operator installed:

```bash
# After old version is installed:
make e2e-upgrade-pre

# After upgrade:
make e2e-upgrade-post
```

### What the tests verify

**`TestUpgradePre`** (before upgrade):

- Creates the `ClusterResourceOverride` CR with the default config
  (memoryRequestToLimitPercent=50, cpuRequestToLimitPercent=25,
  limitCPUToMemoryPercent=200)
- Waits for the CR to reach Available status
- Checks the operand deployment is running
- Verifies the MutatingWebhookConfiguration exists
- Creates an opt-in namespace and pod, asserts resource mutation

**`TestUpgradePost`** (after upgrade):

- Verifies the CR still exists and is Available
- Checks operand deployment has available replicas
- Creates an opt-in namespace and pod, asserts resource mutation still
  works
- Removes the CR (cleanup for subsequent tests)

As more upgrade-sensitive areas are identified (e.g. CRD field
migrations, RBAC changes, new operand behavior), add assertions to
`TestUpgradePre` and `TestUpgradePost` accordingly.

### CI integration

The same Go tests run in CI as part of the `e2e-aws-upgrade` job.
See the [ci-operator config](https://github.com/openshift/release/blob/master/ci-operator/config/openshift/cluster-resource-override-admission-operator/openshift-cluster-resource-override-admission-operator-main.yaml).

---

## Manual upgrade procedure

If you want to run the upgrade steps manually without the tests:

### Install the old version

```bash
NAMESPACE=clusterresourceoverride-operator
oc create namespace $NAMESPACE

operator-sdk run bundle \
  quay.io/$USER/cro-bundle:v${OLD_VERSION} \
  -n $NAMESPACE \
  --security-context-config restricted \
  --timeout 10m
```

### Create the CR and verify

```bash
oc apply -f artifacts/example/clusterresourceoverride-cr.yaml
sleep 10
oc rollout status deployment/clusterresourceoverride -n $NAMESPACE --timeout=120s
```

### Upgrade

```bash
operator-sdk run bundle-upgrade \
  quay.io/$USER/cro-bundle:v${NEW_VERSION} \
  -n $NAMESPACE \
  --security-context-config restricted \
  --timeout 10m
```

### Verify

```bash
oc get csv -n $NAMESPACE
oc get deployment clusterresourceoverride-operator -n $NAMESPACE \
  -o jsonpath='{.spec.template.spec.containers[0].image}'
oc rollout status deployment/clusterresourceoverride -n $NAMESPACE --timeout=120s
```

### Cleanup

```bash
oc delete clusterresourceoverride cluster
operator-sdk cleanup -n $NAMESPACE clusterresourceoverride-operator
oc delete namespace $NAMESPACE
```

---

## Catalog-based upgrade (OCP console)

The `operator-sdk run bundle` approach above is CLI-driven and uses a
synthetic catalog. If you want to see both versions in the console's
OperatorHub UI and trigger the upgrade by approving an InstallPlan
(closer to how customers experience it), you can build a real FBC
catalog image and deploy it via a CatalogSource.

This requires one additional image repo in your registry:

| Repo | Purpose |
|------|---------|
| `<user>/cro-catalog` | FBC catalog image |

### Build the catalog

After building both bundle images (see above), use `build-fbc.sh` to
create a catalog containing both bundles with a `replaces` upgrade
edge:

```bash
CATALOG_REPO=quay.io/$USER/cro-catalog \
BUNDLE_REPO=quay.io/$USER/cro-bundle \
PREV_VERSION=${OLD_VERSION} \
./hack/build-fbc.sh
```

This generates `catalog/cro-fbc.yaml`, builds the catalog image, and
pushes it.

### Deploy the CatalogSource

```bash
CATALOG_IMG=quay.io/$USER/cro-catalog:v${NEW_VERSION} \
./hack/deploy-catalogsource.sh
```

This creates a `CatalogSource` named `cro-fbc` in
`openshift-marketplace` and waits for the PackageManifest to appear.

### Installing and upgrading from the console

Once the CatalogSource is deployed, go to OperatorHub in the OpenShift
console and search for "ClusterResourceOverride". Select the one from
the dev catalog, choose the starting version (e.g. v4.22), and install.
When ready to upgrade, go to the installed operator's page and approve
the upgrade to the newer version.

---

## Notes

### Version strings in the CSV

The CSV `metadata.name` must follow the pattern
`clusterresourceoverride-operator.v<X.Y>.0` and `spec.version` must be
`<X.Y>.0`. The `generate-bundle.sh` script substitutes image references
but does not change the version string. When generating a bundle for a
version different from the source tree, the two versions must be different.
