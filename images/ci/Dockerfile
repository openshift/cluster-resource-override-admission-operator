FROM registry.ci.openshift.org/ocp/builder:rhel-8-golang-1.20-openshift-4.14 AS builder

WORKDIR /go/src/github.com/openshift/cluster-resource-override-admission-operator

COPY . .

RUN make build

FROM registry.ci.openshift.org/ocp/4.14:base

COPY --from=builder /go/src/github.com/openshift/cluster-resource-override-admission-operator/bin/cluster-resource-override-admission-operator /usr/bin/

