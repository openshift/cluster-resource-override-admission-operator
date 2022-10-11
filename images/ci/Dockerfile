FROM registry.ci.openshift.org/openshift/release:golang-1.16 AS builder

WORKDIR /go/src/github.com/openshift/cluster-resource-override-admission-operator

COPY . .

RUN make build

FROM registry.ci.openshift.org/ocp/4.12:base

COPY --from=builder /go/src/github.com/openshift/cluster-resource-override-admission-operator/bin/cluster-resource-override-admission-operator /usr/bin/

