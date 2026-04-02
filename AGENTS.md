This file provides guidance when working with code in this repository.

## Repository Overview

This is the OpenShift **ClusterResourceOverride Admission Operator**. It manages the lifecycle of the [ClusterResourceOverride admission webhook server](https://github.com/openshift/cluster-resource-override-admission) (the operand), which intercepts pod creation requests and overrides CPU/memory resource requests and limits based on configurable ratios. This enables administrators to control overcommit levels and manage container density on nodes.

The operator watches a cluster-scoped singleton `ClusterResourceOverride` CR and ensures the admission webhook operand (Deployment, Service, ConfigMap, APIService, MutatingWebhookConfiguration, RBAC, etc.) is installed in the operator namespace.

Namespaces opt in to resource overrides via the label:
```
clusterresourceoverrides.admission.autoscaling.openshift.io/enabled: "true"
```

## Architecture

### Core Components
- **Operator binary**: `cmd/cluster-resource-override-admission-operator/main.go` — Cobra CLI delegating to `pkg/cmd/operator`
- **Controller**: Custom informer + workqueue controller (not a full controller-runtime Manager)
- **Reconciler**: Sequential handler-chain pattern processing the singleton CR
- **Secondary watches**: SharedInformerFactory watching operand resources and enqueuing the primary CR

### Singleton CR

The operator only reconciles a single `ClusterResourceOverride` CR named **`cluster`** (defined as `DefaultCR` in `pkg/operator/run.go`). All other CR names are silently skipped by the reconciler.

### Handler Chain

The reconciler executes a sequential chain of handlers for each reconciliation:

1. `AvailabilityHandler` (initial check)
2. `ValidationHandler`
3. `ConfigurationHandler` (ConfigMap)
4. `ServiceHandler`
5. `DeploymentHandler`
6. `DeploymentReadyHandler`
7. `APIServiceHandler`
8. `WebhookConfigurationHandler`
9. `AvailabilityHandler` (final check)

The `AvailabilityHandler` intentionally appears twice — as a bookend pattern to check availability at both the start and end of reconciliation.

### Key Directories
- `cmd/`: Operator binary entrypoint and test utilities (`yaml2json`, `json2yaml`)
- `pkg/apis/autoscaling/v1/`: CRD Go types — `ClusterResourceOverride` in group `operator.autoscaling.openshift.io`
- `pkg/generated/`: Generated clientset, informers, and listers (via `hack/update-codegen.sh`)
- `pkg/clusterresourceoverride/`: Primary controller construction and enqueuer
- `pkg/clusterresourceoverride/internal/reconciler/`: Reconcile loop, handler chain, status updates
- `pkg/clusterresourceoverride/internal/handlers/`: Individual handler implementations
- `pkg/controller/`: Generic informer runner and workqueue worker
- `pkg/operator/`: Config and runner orchestration
- `pkg/secondarywatch/`: Secondary informers that enqueue the primary CRO
- `pkg/asset/`: Operand manifest templates and values
- `pkg/deploy/`, `pkg/ensurer/`, `pkg/dynamic/`: Resource apply/update logic
- `pkg/runtime/`: Client wrappers, operand context, enqueuer types
- `test/e2e/`: End-to-end tests
- `test/helper/`: Test preconditions and helpers
- `hack/`: Code generation, deploy helpers, bundle generation scripts
- `artifacts/`: Deploy manifests, OLM subscription/registry templates, examples
- `manifests/`: Stable OLM/CSV, CRD, network policies, package manifest
- `bundle/`: OLM operator-sdk bundle
- `images/`: CI, dev, and operator-registry Dockerfiles

## Development Commands

### Building
```bash
make build                    # Build operator binary (via build-machinery-go)
make local-image              # Build dev container image
make local-push               # Push dev container image
```

### Testing
```bash
make e2e-local                # Deploy operator and run E2E tests locally
make e2e-ci                   # Deploy operator and run E2E tests in CI
```

### Code Quality
```bash
make verify                   # Full verification (gofmt, govet, golang-versions)
make codegen                  # Regenerate clients, listers, informers, deepcopy
```

### Dependency Management
```bash
make vendor                   # Run go mod vendor + go mod tidy
./hack/update-vendor.sh       # Update Kubernetes dependencies to target version
```

### Deployment
```bash
make deploy-local             # Deploy operator using kube manifests (no OLM)
make undeploy-local           # Remove operator and its resources
make deploy-olm-local         # Deploy operator via OLM (local)
make undeploy-olm             # Remove OLM-deployed operator
make create-cro-cr            # Create the ClusterResourceOverride CR
make delete-cro-cr            # Delete the ClusterResourceOverride CR
```

### OLM / Bundle
```bash
make operator-registry-generate   # Generate operator registry manifests
make operator-registry-image      # Build operator registry image
make operator-registry-deploy     # Deploy operator registry
make olm-generate                 # Generate OLM subscription resources
make olm-apply                    # Apply OLM resources
./hack/generate-bundle.sh         # Regenerate operator-sdk bundle manifests (full build + push)
SKIP_BUILD=true ./hack/generate-bundle.sh  # Regenerate bundle manifests only (no image build)
```

#### OLM Bundle Generation Workflow

`hack/generate-bundle.sh` is the script that produces the `bundle/` directory contents. It is part of the CI toolchain and must be re-run whenever `artifacts/deploy/*` or `manifests/stable/*` change, since the bundle is derived from those sources. The bundle is currently used for **OLM upgrade tests**.

The script performs these steps:
1. Builds the operator binary and image (skipped when `SKIP_BUILD=true`)
2. Concatenates `artifacts/deploy/*` and `manifests/stable/*` with `---` separators
3. Pipes the result into `operator-sdk generate bundle` (package `clusterresourceoverride-operator`, channel `stable`)
4. Replaces `CLUSTERRESOURCEOVERRIDE_OPERATOR_IMAGE` and `CLUSTERRESOURCEOVERRIDE_OPERAND_IMAGE` placeholders in the generated CSV with actual image refs
5. Validates the bundle with `operator-sdk bundle validate ./bundle`
6. Builds and pushes the bundle image (skipped when `SKIP_BUILD=true`)

Environment variables:
- `OPERATOR_IMG` — operator image reference (required for full build)
- `OPERAND_IMG` — operand image reference (required for full build)
- `BUNDLE_IMG` — bundle image reference (required for full build)
- `SKIP_BUILD` — set to `true` to skip binary/image build and push (regenerate manifests only)

For CI or release chores, use `SKIP_BUILD=true` to regenerate the bundle manifests without needing container tooling:
```bash
SKIP_BUILD=true ./hack/generate-bundle.sh
```

### Test Utilities
```bash
make build-testutil           # Build yaml2json and json2yaml utilities
```

## Testing Strategy

### Unit Tests
- There are currently no `*_test.go` files in `pkg/` or `cmd/`.

### E2E Tests
- Located in `test/e2e/`.
- Require a running OpenShift cluster with `KUBECONFIG`, `LOCAL_OPERATOR_IMAGE`, and `LOCAL_OPERAND_IMAGE` set.
- Test CRO creation, pod resource override behavior, and opt-in namespace labeling.
- Use `testify/require` for assertions.
- Run with `make e2e-local` (15m timeout).

## Key Development Patterns

### Controller Architecture
This operator does **not** use the standard controller-runtime `Manager` pattern. Instead, it uses:
- **client-go `cache.NewIndexerInformer`** + **`workqueue.RateLimitingInterface`** for the primary watch on `ClusterResourceOverride` resources.
- **`controller-runtime/pkg/reconcile.Reconciler`** interface only for the reconciler itself.
- **`SharedInformerFactory`** for secondary resource watches (Deployments, DaemonSets, Pods, ConfigMaps, Services, Secrets, ServiceAccounts, MutatingWebhookConfigurations).

### Operand Context
The `pkg/runtime.OperandContext` carries the operator name, namespace, singleton CR name, operand image, and operand version through the controller and reconciler.

### Asset Management
`pkg/asset` manages operand manifest templates. Values from the `OperandContext` and CR spec are injected into templates to produce the operand's Kubernetes resources.

### Health Check
A simple HTTP health endpoint is served on `:8080` at `/healthz`.

## Dependencies and Modules

This is a **single Go module** project. Check `go.mod` for current versions. Key dependencies:
- Kubernetes `k8s.io/*` libraries
- controller-runtime (reconcile types only, not the full manager)
- OpenShift build-machinery-go (included Makefile rules)

The project uses **vendoring** (`GOFLAGS=-mod=vendor`). Do not modify `vendor/` directly — use `make vendor` or `./hack/update-vendor.sh`.

## API Definition

- **Group**: `operator.autoscaling.openshift.io`
- **Version**: `v1`
- **Kind**: `ClusterResourceOverride` (cluster-scoped, `+genclient:nonNamespaced`)
- **Short name**: `cro`
- **Go types**: `pkg/apis/autoscaling/v1/override_types.go`
- **CRD YAML**: `manifests/stable/clusterresourceoverride.crd.yaml`

## Environment Variables

The operator binary requires:
- `OPERAND_IMAGE`: Container image of the ClusterResourceOverride admission webhook server
- `OPERAND_VERSION`: Version string for the operand

## Common Gotchas

- **Singleton CR**: Only the `ClusterResourceOverride` named `cluster` is reconciled. Creating a CR with any other name has no effect.
- **Not standard controller-runtime**: Do not assume controller-runtime Manager patterns — this uses raw client-go informers and workqueue.
- **Do not modify `vendor/` directly**: Use `make vendor` or `./hack/update-vendor.sh` to update dependencies.
- **OLM bundle generation**: `hack/generate-bundle.sh` must be re-run (with `SKIP_BUILD=true` for manifest-only regeneration) whenever `artifacts/deploy/*` or `manifests/stable/*` change, since the `bundle/` directory is derived from those sources. Requires `operator-sdk` to be installed. The bundle is currently used for OLM upgrade tests.
- **CI image**: The build root is defined in `.ci-operator.yaml` — this file is updated by the ART team, not manually.
- **E2E tests need a cluster**: There are no mock-based or local tests. E2E requires a running OpenShift cluster with proper images set.

## Code Conventions
Please see `.cursor/rules/code-formatting.mdc` for code quality, formatting, and conventions.

## Commit Messages
Please see `.cursor/rules/git-commit-format.mdc` for information on how commit messages should be generated or formatted in this project.
