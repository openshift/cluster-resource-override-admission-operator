FROM registry.svc.ci.openshift.org/openshift/release:golang-1.15 AS builder

WORKDIR /go/src/github.com/openshift/cluster-resource-override-admission-operator

COPY . .

RUN make build

FROM registry.svc.ci.openshift.org/openshift/origin-v4.0:base

COPY --from=builder /go/src/github.com/openshift/cluster-resource-override-admission-operator/bin/cluster-resource-override-admission-operator /usr/bin/

