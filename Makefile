all: build
.PHONY: all

GO=GO111MODULE=on GOFLAGS=-mod=vendor go
GO_BUILD_BINDIR := bin

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

registry:
	docker build -t docker.io/tohinkashem/clusterresourceoverride-registry:latest -f images/operator-registry/Dockerfile.registry .
	docker push docker.io/tohinkashem/clusterresourceoverride-registry:latest

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

