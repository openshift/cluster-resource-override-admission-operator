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
