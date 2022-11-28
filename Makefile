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

KUBECTL = kubectl
VERSION := 4.13

OPERATOR_NAMESPACE 			:= clusterresourceoverride-operator
OPERATOR_DEPLOYMENT_NAME 	:= clusterresourceoverride-operator

export OLD_OPERATOR_IMAGE_URL_IN_CSV 	= quay.io/openshift/clusterresourceoverride-rhel8-operator:$(VERSION)
export OLD_OPERAND_IMAGE_URL_IN_CSV 	= quay.io/openshift/clusterresourceoverride-rhel8:$(VERSION)
export CSV_FILE_PATH_IN_REGISTRY_IMAGE 	= /manifests/stable/clusterresourceoverride-operator.clusterserviceversion.yaml

LOCAL_OPERATOR_IMAGE	?= quay.io/redhat/clusterresourceoverride-operator:latest
LOCAL_OPERAND_IMAGE 	?= quay.io/redhat/clusterresourceoverride:latest
export LOCAL_OPERATOR_IMAGE
export LOCAL_OPERAND_IMAGE
export LOCAL_OPERATOR_REGISTRY_IMAGE

# Include the library makefile
include $(addprefix ./vendor/github.com/openshift/library-go/alpha-build-machinery/make/, \
	golang.mk \
	targets/openshift/images.mk \
)

# build image for ci
CI_IMAGE_REGISTRY ?=registry.ci.openshift.org
$(call build-image,clusterresourceoverride-operator,$(CI_IMAGE_REGISTRY)/autoscaling/clusterresourceoverride-operator,./images/ci/Dockerfile,.)

REGISTRY_SETUP_BINARY := bin/registry-setup

$(REGISTRY_SETUP_BINARY): GO_BUILD_PACKAGES =./test/registry-setup/...
$(REGISTRY_SETUP_BINARY): build

# build image for dev use.
dev-image:
	docker build -t $(DEV_IMAGE_REGISTRY):$(IMAGE_TAG) -f ./images/dev/Dockerfile.dev .

dev-push:
	docker push $(DEV_IMAGE_REGISTRY):$(IMAGE_TAG)

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

deploy-olm-local: operator-registry-deploy-local olm-generate olm-apply
deploy-olm-ci: operator-registry-deploy-ci olm-generate olm-apply

operator-registry-deploy-local: operator-registry-generate operator-registry-image-ci operator-registry-deploy
operator-registry-deploy-ci: operator-registry-generate operator-registry-deploy

# TODO: Use alpha-build-machinery for codegen
PKG=github.com/openshift/cluster-resource-override-admission-operator
CODEGEN_INTERNAL:=./vendor/k8s.io/code-generator/generate-internal-groups.sh

codegen:
	docker build -t cro:codegen -f Dockerfile.codegen .
	docker run --name cro-codegen cro:codegen /bin/true
	docker cp cro-codegen:/go/src/github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/. ./pkg/generated
	docker cp cro-codegen:/go/src/github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/. ./pkg/apis
	docker rm cro-codegen

codegen-internal: export GO111MODULE := off
codegen-internal:
	mkdir -p vendor/k8s.io/code-generator/hack
	cp boilerplate.go.txt vendor/k8s.io/code-generator/hack/boilerplate.go.txt
	$(CODEGEN_INTERNAL) deepcopy,conversion,client,lister,informer $(PKG)/pkg/generated $(PKG)/pkg/apis $(PKG)/pkg/apis "autoscaling:v1"

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
# local e2e test is done exactly the same way as ci with the exception that
# in ci the operator registry image is built by ci agent.
# on the other hand, in local mode, we need to build the image.
operator-registry-image-ci:
	docker build --build-arg VERSION=$(VERSION) -t $(LOCAL_OPERATOR_REGISTRY_IMAGE) -f images/operator-registry/Dockerfile.registry.ci .
	docker push $(LOCAL_OPERATOR_REGISTRY_IMAGE)

# build and push the OLM manifests for this operator into an operator-registry image.
# this builds an image with the generated database, (unlike image used for ci)
operator-registry-image: MANIFESTS_DIR := "$(OUTPUT_DIR)/manifests"
operator-registry-image: CSV_FILE := "$(MANIFESTS_DIR)/stable/clusterresourceoverride-operator.clusterserviceversion.yaml"
operator-registry-image:
	rm -rf $(MANIFESTS_DIR)
	mkdir -p $(MANIFESTS_DIR)
	cp -r manifests/* $(MANIFESTS_DIR)/

	sed "s,$(OLD_OPERATOR_IMAGE_URL_IN_CSV),$(LOCAL_OPERATOR_IMAGE),g" -i "$(CSV_FILE)"
	sed "s,$(OLD_OPERAND_IMAGE_URL_IN_CSV),$(LOCAL_OPERAND_IMAGE),g" -i "$(CSV_FILE)"

	docker build --build-arg MANIFEST_LOCATION=$(MANIFESTS_DIR) -t $(LOCAL_OPERATOR_REGISTRY_IMAGE) -f images/operator-registry/Dockerfile.registry .
	docker push $(LOCAL_OPERATOR_REGISTRY_IMAGE)
