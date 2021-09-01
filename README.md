# Overview
This operator manages OpenShift `ClusterResourceOverride` Admission Webhook Server.

Scheduling is based on resources requested, while quota and hard limits refer to resource limits, which can be set higher 
than requested resources. The difference between request and limit determines the level of overcommit; for instance, 
if a container is given a memory request of `1Gi` and a memory limit of `2Gi`, it is scheduled based on the `1Gi` request 
being available on the node, but could use up to `2Gi`; so it is `200%` overcommitted.

If OpenShift Container Platform administrators would like to control the level of `overcommit` and manage container 
density on nodes, `ClusterResourceOverride` Admission Webhook can be configured to override the ratio between request and
limit set on developer containers. In conjunction with a per-project `LimitRange` specifying `limits` and `defaults`, 
this adjusts the container `limit` and `request` to achieve the desired level of `overcommit`.

`ClusterResourceOverride` Admission Webhook Server is located at [cluster-resource-override-admission](https://github.com/openshift/cluster-resource-override-admission).

## Prerequisites
- [git][git_tool]
- [go][go_tool] version v1.12+.
- [docker][docker_tool] version 17.03+.
  - Alternatively [podman][podman_tool] `v1.2.0+` or [buildah][buildah_tool] `v1.7+`
- [kubectl][kubectl_tool] version v1.11.3+.
- Access to a OpenShift v4.x cluster.

## Getting Started
A quick way to test your changes is to build the operator binary and run it directly from the command line.
```bash
# change to the root folder of the repo

# build the operator binary
make build

# the operator owns a CRD, so register the CRD
kubectl apply -f artifacts/olm/manifests/clusterresourceoverride/1.0.0/clusterresourceoverride.crd.yaml 

# make sure you have a cluster up and running
# create a namespace where the operator binary will manage its resource(s)
kubectl create ns cro

# before you run the operator binary, make sure you have the following
# OPERAND_IMAGE: this points to the image of ClusterResourceOverride admission webhook server.
# OPERAND_VERSION: the version of the operand.

# run the operator binary
OPERAND_IMAGE=quay.io/{openshift}/clusterresourceoverride@sha256:{image digest} OPERAND_VERSION=1.0.0 \
bin/cluster-resource-override-admission-operator start --namespace=cro --kubeconfig=${KUBECONFIG} --v=4
``` 

Now, if you want to install the `ClusterResourceOverride` admission webhook server then simply create a custom resource
of `ClusterResourceOverride` type.
```yaml
apiVersion: operator.autoscaling.openshift.io/v1
kind: ClusterResourceOverride
metadata:
  name: cluster
spec:
  podResourceOverride:
    spec:
      memoryRequestToLimitPercent: 50
      cpuRequestToLimitPercent: 25
      limitCPUToMemoryPercent: 200
```

This repo ships with an example CR, you can directly apply the YAML resource as well. 
```bash
kubectl apply -f artifacts/example/clusterresourceoverride-cr.yaml
```

The operator watches for the custom resource(s) of `ClusterResourceOverride` type and will ensure that the 
`ClusterResourceOverride` admission webhook server is installed into the same namespace as the operator. You can check
the current state of the admission webhook by checking the status of the `cluster` custom resource    
```bash
kubectl get clusterresourceoverride cluster -o yaml
```
```yaml
apiVersion: operator.autoscaling.openshift.io/v1
kind: ClusterResourceOverride
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"operator.autoscaling.openshift.io/v1","kind":"ClusterResourceOverride","metadata":{"annotations":{},"name":"cluster"},"spec":{"podResourceOverride":{"spec":{"cpuRequestToLimitPercent":25,"limitCPUToMemoryPercent":200,"memoryRequestToLimitPercent":50}}}}
  creationTimestamp: "2019-12-18T22:35:02Z"
  generation: 1
  name: cluster
  resourceVersion: "127622"
  selfLink: /apis/operator.autoscaling.openshift.io/v1/clusterresourceoverrides/cluster
  uid: 978fc959-1717-4bd1-97d0-ae00ee111e8d
spec:
  podResourceOverride:
    spec:
      cpuRequestToLimitPercent: 25
      limitCPUToMemoryPercent: 200
      memoryRequestToLimitPercent: 50
status:
  certsRotateAt: "2020-12-15T22:52:46Z"
  conditions:
  - lastTransitionTime: "2019-12-18T23:27:03Z"
    status: "True"
    type: Available
  - lastTransitionTime: "2019-12-18T23:27:03Z"
    status: "False"
    type: InstallReadinessFailure
  hash:
    configuration: 577fe3d2b05619ac326571a3504857e3e7e70a275c941e3397aa9db5c1a1d3a4
    servingCert: bae6f72e906e9b054d0f4eb4b1540755f284373ad2febdec8454521ebfac5765
  image: docker.io/tohinkashem/clusterresourceoverride@sha256:f99ed0d7ecf197596bb34eed342ddc8a9169f68b68a457ee5a15070bc68e848a
  resources:
    configurationRef:
      apiVersion: v1
      kind: ConfigMap
      name: clusterresourceoverride-configuration
      namespace: cro
      resourceVersion: "78476"
      uid: e8a7f90e-90b1-4f2b-9b03-0064ffd33d4b
    deploymentRef:
      apiVersion: apps/v1
      kind: DaemonSet
      name: clusterresourceoverride
      namespace: cro
      resourceVersion: "78676"
      uid: 947ebfc5-c244-4621-acc3-c3117a10a97b
    mutatingWebhookConfigurationRef:
      apiVersion: admissionregistration.k8s.io/v1
      kind: MutatingWebhookConfiguration
      name: clusterresourceoverrides.admission.autoscaling.openshift.io
      resourceVersion: "127621"
      uid: 98b3b8ae-d5ce-462b-8ab5-a729ea8f38f3
    serviceCAConfigMapRef:
      apiVersion: v1
      kind: ConfigMap
      name: clusterresourceoverride-service-serving
      namespace: cro
      resourceVersion: "78479"
      uid: faf27927-0fb0-453b-8656-1b05ea9a3d74
    serviceCertSecretRef:
      apiVersion: v1
      kind: Secret
      name: server-serving-cert-clusterresourceoverride
      namespace: cro
      resourceVersion: "78478"
      uid: cc6efc0e-0fce-4148-a943-6045dadaa72c
  version: 1.0.0
```

### Test Pod Resource Override
The `ClusterResourceOverride` admission webhook enforces an opt-in approach, object(s) belonging to a namespace that has the following label are admitted, all other objects are ignored.
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: test
  labels:
    clusterresourceoverrides.admission.autoscaling.openshift.io/enabled: "true"
``` 

* Create a namespace with the appropriate label.
* Create a `Pod` in the above namespace. The `requests` and `limits` of the `Pod's` `resources` are overridden according to the configuration of the webhook server.
```bash
kubectl apply -f artifacts/example/test-namespace.yaml
kubectl apply -f artifacts/example/test-pod.yaml    
```

The original request of the `Pod` has the following `resources` section. 
```yaml
spec:
  containers:
    - name: hello-openshift
      image: openshift/hello-openshift
      resources:
        limits:
          memory: "512Mi"
          cpu: "2000m"
``` 

The Admission webhook intercepts the original `Pod` request and overrides the `resources` according to the configuration. 
```yaml
spec:
  containers:
  - image: openshift/hello-openshift
    name: hello-openshift
    resources:
      limits:
        cpu: "1"
        memory: 512Mi
      requests:
        cpu: 250m
        memory: 256Mi
``` 

## Deploy 
You can also deploy the operator on an OpenShift cluster:
* Build the operator binary
* Build and push operator image
* Apply kubernetes manifests

```bash
# change to the root folder of the repo

# build operator binary
make build

# build and push operator image
make dev-image DEV_IMAGE_REGISTRY=docker.io/{your org}/clusterresourceoverride-operator IMAGE_TAG=dev
make dev-push DEV_IMAGE_REGISTRY=docker.io/{your org}/clusterresourceoverride-operator IMAGE_TAG=dev

# deploy on local cluster
# CLUSTERRESOURCEOVERRIDE_OPERAND_IMAGE: operator image
# CLUSTERRESOURCEOVERRIDE_OPERATOR_IMAGE: operand image
make deploy-local CLUSTERRESOURCEOVERRIDE_OPERAND_IMAGE="{operand image}" CLUSTERRESOURCEOVERRIDE_OPERATOR_IMAGE="{operator image}
```

## Deploy via OLM
There are three steps:
* Package the OLM manifests into an [operator registry](https://github.com/operator-framework/operator-registry) bundle image and push it to an image registry
* Make the above operator `catalog source` available to your OpenShift cluster.
* Deploy the operator via `OLM`.

Before you package the OLM manifests, make sure the `CSV` file `artifacts/olm/manifests/clusterresourceoverride/1.0.0/clusterresourceoverride.v1.csv.yaml`points to the right operator and operand image.
```bash
# build and push the image
make operator-registry OLM_IMAGE_REGISTRY=docker.io/{your org}/clusterresourceoverride-registry IMAGE_TAG=dev

# make your catalog source available to the cluster
kubectl apply -f artifacts/olm/catalog-source.yaml

# wait for the CatalogSource object to be in 'READY' state.
# one way to make sure is to check the 'status' block of the CatalogSource object
kubectl -n clusterresourceoverride-operator get catalogsource clusterresourceoverride-catalog -o yaml

# or, you can query to check if your operator has been registered
kubectl -n clusterresourceoverride-operator get packagemanifests | grep clusterresourceoverride 

# at this point, you can install the operator from OperatorHub UI.
# if you want to do it from the command line, then execute the following:

# create an `OperatorGroup` object to associated with the operator namespace
# and the create a Subscription object.
kubectl apply -f artifacts/olm/operator-group.yaml
kubectl apply -f artifacts/olm/subscription.yaml

# install the ClusterResourceOverride admission webhook server by creating a custom resource
kubectl apply -f artifacts/example/clusterresourceoverride-cr.yaml 
```
