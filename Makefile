all: build
.PHONY: all manifests

OUTPUT_DIR 						:= "./_output"
ARTIFACTS 						:= "./artifacts"
OLM_ARTIFACTS 					:= "$(ARTIFACTS)/olm"
KUBE_MANIFESTS_DIR 				:= "$(OUTPUT_DIR)/deployment"
OPERATOR_REGISTRY_MANIFESTS_DIR := "$(OUTPUT_DIR)/olm/registry"
OLM_MANIFESTS_DIR 				:= "$(OUTPUT_DIR)/olm/subscription"

GO=GO111MODULE=on GOFLAGS=-mod=vendor go
GO_BUILD_BINDIR := bin
GO_TEST_PACKAGES :=./pkg/... ./cmd/...

KUBECTL ?= kubectl
CONTAINER_ENGINE ?= podman
# can support podman, docker, and buildah
IMAGE_BUILDER ?= $(CONTAINER_ENGINE)
# this is the version of the operator that should change with each release
IMAGE_VERSION := 4.22
# this is the image tag of your custom dev image
IMAGE_TAG := dev

OPERATOR_NAMESPACE 			:= clusterresourceoverride-operator
OPERATOR_DEPLOYMENT_NAME 	:= clusterresourceoverride-operator

export OLD_OPERATOR_IMAGE_URL_IN_CSV 	= quay.io/openshift/clusterresourceoverride-rhel8-operator:$(IMAGE_VERSION)
export OLD_OPERAND_IMAGE_URL_IN_CSV 	= quay.io/openshift/clusterresourceoverride-rhel8:$(IMAGE_VERSION)
export CSV_FILE_PATH_IN_REGISTRY_IMAGE 	= /manifests/stable/clusterresourceoverride-operator.clusterserviceversion.yaml

OPERATOR_IMAGE_TAG_BASE ?= quay.io/redhat/clusterresourceoverride-operator
OPERAND_IMAGE_TAG_BASE ?= quay.io/redhat/clusterresourceoverride

LOCAL_OPERATOR_IMAGE ?= $(OPERATOR_IMAGE_TAG_BASE):$(IMAGE_TAG)
LOCAL_OPERAND_IMAGE ?= $(OPERAND_IMAGE_TAG_BASE):$(IMAGE_TAG)
LOCAL_OPERATOR_REGISTRY_IMAGE ?= $(OPERATOR_IMAGE_TAG_BASE)-registry:$(IMAGE_TAG)

export LOCAL_OPERATOR_IMAGE
export LOCAL_OPERAND_IMAGE
export LOCAL_OPERATOR_REGISTRY_IMAGE

# Include the library makefile
include $(addprefix ./vendor/github.com/openshift/build-machinery-go/make/, \
	golang.mk \
	targets/openshift/images.mk \
)

# build image for ci
CI_IMAGE_REGISTRY ?=registry.ci.openshift.org
$(call build-image,clusterresourceoverride-operator,$(CI_IMAGE_REGISTRY)/autoscaling/clusterresourceoverride-operator,./images/ci/Dockerfile,.)

REGISTRY_SETUP_BINARY := bin/registry-setup

$(REGISTRY_SETUP_BINARY): GO_BUILD_PACKAGES =./test/registry-setup/...
$(REGISTRY_SETUP_BINARY): build

# build local image for dev use.
local-image:
	$(IMAGE_BUILDER) build -t $(LOCAL_OPERATOR_IMAGE) -f ./images/dev/Dockerfile.dev .

local-push:
	$(IMAGE_BUILDER) push $(LOCAL_OPERATOR_IMAGE)

.PHONY: vendor
vendor:
	go mod vendor
	go mod tidy

clean:
	rm -rf $(OUTPUT_DIR)

e2e-olm-local: DEPLOY_MODE := local
e2e-olm-local: deploy-olm-local e2e

e2e-olm-ci: DEPLOY_MODE := ci
e2e-olm-ci: KUBECTL=$(shell which oc)
e2e-olm-ci: deploy-olm-ci e2e

# oc binary should be in test pod's /tmp/shared dir
e2e-ci: DEPLOY_MODE := ci
e2e-ci: KUBECTL=$(shell which oc)
e2e-ci: deploy e2e

e2e-local: DEPLOY_MODE=local
e2e-local: deploy e2e

deploy-olm-local: DEPLOY_MODE := local
deploy-olm-local: operator-registry-deploy-local olm-generate olm-apply
deploy-olm-ci: operator-registry-deploy-ci olm-generate olm-apply

operator-registry-deploy-local: operator-registry-generate operator-registry-image operator-registry-deploy
operator-registry-deploy-ci: operator-registry-generate operator-registry-deploy

PKG=github.com/openshift/cluster-resource-override-admission-operator

# similar to make generate in operator-sdk/kubebuilder projects
codegen:
	./hack/update-codegen.sh

deploy-local: DEPLOY_MODE := local
deploy-local: deploy

# deploy the operator using kube manifests (no OLM)
deploy: KUBE_MANIFESTS_SOURCE := "$(ARTIFACTS)/deploy"
deploy: DEPLOYMENT_YAML := "$(KUBE_MANIFESTS_DIR)/300_deployment.yaml"
deploy: CONFIGMAP_ENV_FILE := "$(KUBE_MANIFESTS_DIR)/registry-env.yaml"
deploy: $(REGISTRY_SETUP_BINARY)
deploy:
	rm -rf $(KUBE_MANIFESTS_DIR)
	mkdir -p $(KUBE_MANIFESTS_DIR)
	cp -r $(KUBE_MANIFESTS_SOURCE)/* $(KUBE_MANIFESTS_DIR)/
	cp manifests/stable/clusterresourceoverride.crd.yaml $(KUBE_MANIFESTS_DIR)/
	cp $(ARTIFACTS)/registry-env.yaml $(KUBE_MANIFESTS_DIR)/

	$(REGISTRY_SETUP_BINARY) --mode=$(DEPLOY_MODE) --olm=false --configmap=$(CONFIGMAP_ENV_FILE)
	./hack/update-image-url.sh "$(CONFIGMAP_ENV_FILE)" "$(DEPLOYMENT_YAML)"

	$(KUBECTL) apply -n $(OPERATOR_NAMESPACE) -f $(KUBE_MANIFESTS_DIR)

# Alias for undeploy-local
undeploy: undeploy-local
undeploy-local: delete-test-pod delete-cro-cr
	$(KUBECTL) delete -f $(KUBE_MANIFESTS_DIR) --ignore-not-found

undeploy-olm: delete-test-pod delete-cro-cr
	$(KUBECTL) delete -n $(OPERATOR_NAMESPACE) -f $(OPERATOR_REGISTRY_MANIFESTS_DIR) --ignore-not-found
	$(KUBECTL) delete -n $(OPERATOR_NAMESPACE) -f $(OLM_MANIFESTS_DIR) --ignore-not-found
	$(KUBECTL) delete -n $(OPERATOR_NAMESPACE) -f $(KUBE_MANIFESTS_DIR) --ignore-not-found

# run e2e test(s)
e2e:
	$(KUBECTL) -n $(OPERATOR_NAMESPACE) rollout status -w deployment/clusterresourceoverride-operator
	export GO111MODULE=on
	$(GO) test -v -count=1 -timeout=15m ./test/e2e/... --kubeconfig=${KUBECONFIG} --namespace=$(OPERATOR_NAMESPACE)

# apply OLM resources to deploy the operator.
olm-apply:
	$(KUBECTL) apply -n $(OPERATOR_NAMESPACE) -f $(OLM_MANIFESTS_DIR)
	./hack/wait-for-deployment.sh $(KUBECTL) $(OPERATOR_NAMESPACE) $(OPERATOR_DEPLOYMENT_NAME) 500

# generate OLM resources (Subscription and OperatorGroup etc.) to install the operator via olm.
olm-generate: OPERATOR_GROUP_FILE := "$(OLM_MANIFESTS_DIR)/operator-group.yaml"
olm-generate: SUBSCRIPTION_FILE := "$(OLM_MANIFESTS_DIR)/subscription.yaml"
olm-generate:
	rm -rf $(OLM_MANIFESTS_DIR)
	mkdir -p $(OLM_MANIFESTS_DIR)
	cp -r $(OLM_ARTIFACTS)/subscription/* $(OLM_MANIFESTS_DIR)/

	sed "s/OPERATOR_NAMESPACE_PLACEHOLDER/$(OPERATOR_NAMESPACE)/g" -i "$(OPERATOR_GROUP_FILE)"
	sed "s/OPERATOR_NAMESPACE_PLACEHOLDER/$(OPERATOR_NAMESPACE)/g" -i "$(SUBSCRIPTION_FILE)"
	sed "s/OPERATOR_PACKAGE_CHANNEL/\"stable\"/g" -i "$(SUBSCRIPTION_FILE)"

# generate kube resources to deploy operator registry image using an init container.
operator-registry-generate: OPERATOR_REGISTRY_DEPLOYMENT_YAML := "$(OPERATOR_REGISTRY_MANIFESTS_DIR)/registry-deployment.yaml"
operator-registry-generate: CONFIGMAP_ENV_FILE := "$(OPERATOR_REGISTRY_MANIFESTS_DIR)/registry-env.yaml"
operator-registry-generate: $(REGISTRY_SETUP_BINARY)
operator-registry-generate:
	rm -rf $(OPERATOR_REGISTRY_MANIFESTS_DIR)
	mkdir -p $(OPERATOR_REGISTRY_MANIFESTS_DIR)
	cp -r $(OLM_ARTIFACTS)/registry/* $(OPERATOR_REGISTRY_MANIFESTS_DIR)/
	cp $(ARTIFACTS)/registry-env.yaml $(OPERATOR_REGISTRY_MANIFESTS_DIR)/

	# write image URL(s) into a json file and
	#   IMAGE_FORMAT='registry.svc.ci.openshift.org/ci-op-9o8bacu/stable:${component}'
	$(REGISTRY_SETUP_BINARY) --mode=$(DEPLOY_MODE) --olm=true --configmap=$(CONFIGMAP_ENV_FILE)
	./hack/update-image-url.sh "$(CONFIGMAP_ENV_FILE)" "$(OPERATOR_REGISTRY_DEPLOYMENT_YAML)"

.PHONY: build-testutil
build-testutil: bin/yaml2json bin/json2yaml ## Build utilities needed by tests

# utilities needed by tests
bin/yaml2json: cmd/testutil/yaml2json/yaml2json.go
	mkdir -p bin
	go build $(GOGCFLAGS) -ldflags "$(LD_FLAGS)" -o bin/ "$(PKG)/cmd/testutil/yaml2json"
bin/json2yaml: cmd/testutil/json2yaml/json2yaml.go
	mkdir -p bin
	go build $(GOGCFLAGS) -ldflags "$(LD_FLAGS)" -o bin/ "$(PKG)/cmd/testutil/json2yaml"

# deploy the operator registry image
operator-registry-deploy: CATALOG_SOURCE_TYPE := address
operator-registry-deploy: bin/yaml2json
	./hack/deploy-operator-registry.sh $(OPERATOR_NAMESPACE) $(KUBECTL) $(OPERATOR_REGISTRY_MANIFESTS_DIR) ./bin/yaml2json


# build operator registry image for ci locally (used for local e2e test only)
# local e2e test is done exactly the same way as ci withoperator-registry-image the exception that
# in ci the operator registry image is built by ci agent.
# on the other hand, in local mode, we need to build the image.
operator-registry-image-ci:
	$(IMAGE_BUILDER) build --build-arg VERSION=$(IMAGE_VERSION) -t $(LOCAL_OPERATOR_REGISTRY_IMAGE) -f images/operator-registry/Dockerfile.registry.ci .
	$(IMAGE_BUILDER) push $(LOCAL_OPERATOR_REGISTRY_IMAGE)

# same as operator-registry-image-ci but use a dev tag instead of a VERSION which should stay consistent
operator-registry-image:
	$(IMAGE_BUILDER) build --build-arg VERSION=$(IMAGE_TAG) -t $(LOCAL_OPERATOR_REGISTRY_IMAGE) -f images/operator-registry/Dockerfile.registry.ci .
	$(IMAGE_BUILDER) push $(LOCAL_OPERATOR_REGISTRY_IMAGE)

create-cro-cr:
	$(KUBECTL) apply -f ./artifacts/example/clusterresourceoverride-cr.yaml

delete-cro-cr: delete-api-resources
delete-cro-cr:
	$(KUBECTL) delete -f ./artifacts/example/clusterresourceoverride-cr.yaml --ignore-not-found

create-test-pod:
	$(KUBECTL) apply -f ./artifacts/example/test-namespace.yaml
	$(KUBECTL) apply -f ./artifacts/example/test-pod.yaml

delete-test-pod:
	$(KUBECTL) delete -f ./artifacts/example/test-pod.yaml --ignore-not-found
	$(KUBECTL) delete -f ./artifacts/example/test-namespace.yaml --ignore-not-found

delete-api-resources:
	$(KUBECTL) delete apiservice v1.admission.autoscaling.openshift.io --ignore-not-found
