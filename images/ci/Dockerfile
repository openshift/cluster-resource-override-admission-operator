FROM registry.ci.openshift.org/openshift/release:rhel-9-release-golang-1.22-openshift-4.18 AS builder

WORKDIR /go/src/github.com/openshift/cluster-resource-override-admission-operator

COPY . .

RUN make build

FROM registry.ci.openshift.org/ocp/4.18:base-rhel9

COPY --from=builder /go/src/github.com/openshift/cluster-resource-override-admission-operator/bin/cluster-resource-override-admission-operator /usr/bin/

