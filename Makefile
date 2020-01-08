all: build
.PHONY: all manifests

OUTPUT_DIR := "./_output"
ARTIFACTS := "./artifacts"

GO=GO111MODULE=on GOFLAGS=-mod=vendor go
GO_BUILD_BINDIR := bin

GO_TEST_PACKAGES :=./pkg/... ./cmd/...

KUBECTL = kubectl

# Include the library makefile
include $(addprefix ./vendor/github.com/openshift/library-go/alpha-build-machinery/make/, \
	golang.mk \
	targets/openshift/images.mk \
)

# build image for ci
CI_IMAGE_REGISTRY ?=registry.svc.ci.openshift.org
$(call build-image,clusterresourceoverride-operator,$(CI_IMAGE_REGISTRY)/autoscaling/clusterresourceoverride-operator,./images/ci/Dockerfile,.)

# build image for dev use.
dev-image:
	docker build -t $(DEV_IMAGE_REGISTRY):$(IMAGE_TAG) -f ./images/dev/Dockerfile.dev .

dev-push:
	docker push $(DEV_IMAGE_REGISTRY):$(IMAGE_TAG)

# build and push the OLM manifests for this operator into an operator-registry image
registry:
	docker build -t $(OLM_IMAGE_REGISTRY):$(IMAGE_TAG) -f images/operator-registry/Dockerfile.registry .
	docker push $(OLM_IMAGE_REGISTRY):$(IMAGE_TAG)

.PHONY: vendor
vendor:
	go mod vendor
	go mod tidy


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


deploy-local: DEPLOY_MODE := local
deploy-local: deploy

# oc binary should be in test pod's /tmp/shared dir
deploy-ci: KUBECTL=/tmp/shared/oc
deploy-ci: DEPLOY_MODE := ci
deploy-ci: deploy

deploy: DEPLOY_MANIFESTS_SOURCE := "$(ARTIFACTS)/deploy"
deploy: DEPLOY_DIR := "$(OUTPUT_DIR)/deploy"
deploy: DEPLOYMENT_YAML := "$(DEPLOY_DIR)/300_deployment.yaml"
deploy: IMAGE_URL_FILE_PATH := "$(OUTPUT_DIR)/image.json"
deploy:
	rm -rf $(OUTPUT_DIR)
	mkdir -p $(DEPLOY_DIR)
	cp -r $(DEPLOY_MANIFESTS_SOURCE)/* $(DEPLOY_DIR)/
	cp manifests/4.4/clusterresourceoverride.crd.yaml $(DEPLOY_DIR)/

	# write image URL(s) into a json file and
	# update the Deployment YAML with the image URL(s)
	$(GO) run ./test/image/main.go --mode=$(DEPLOY_MODE) --output=$(IMAGE_URL_FILE_PATH)
	./hack/update-image-url.sh "$(IMAGE_URL_FILE_PATH)" "$(DEPLOYMENT_YAML)"

	$(KUBECTL) apply -f $(DEPLOY_DIR)


e2e-ci: deploy-ci e2e
e2e-local: deploy-local e2e

e2e: OPERATOR_NAMESPACE := clusterresourceoverride-operator
e2e:
	$(KUBECTL) -n $(OPERATOR_NAMESPACE) rollout status -w deployment/clusterresourceoverride-operator

	export GO111MODULE=on
	$(GO) test -v -count=1 -timeout=15m ./test/e2e/... --kubeconfig=${KUBECONFIG} --namespace=$(OPERATOR_NAMESPACE)




clean:
	rm -rf $(OUTPUT_DIR)