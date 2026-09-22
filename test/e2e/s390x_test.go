package e2e

// s390x_test.go contains tests that are specific to IBM Z (s390x) hardware.
// Every test in this file calls t.Skip when no schedulable s390x node is present,
// so the suite remains runnable on any cluster but only exercises real-hardware
// paths when the target architecture is actually available.
//
// Architecture-agnostic behaviour (cpuRequestToRequestPercent mutation math,
// ResourceOverride conflict resolution, field precedence) is tested in e2e_test.go.

import (
	"testing"

	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"

	operatorv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/operator/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/test/helper"
)

const s390xArch = "s390x"

// s390xTestImage is the container image used by IBM Z e2e test pods.
// registry.access.redhat.com/ubi9/httpd-24:latest — no auth required, s390x
// multi-arch manifest available on s390x OCP cluster worker nodes.
const s390xTestImage = "registry.access.redhat.com/ubi9/httpd-24:latest"

func s390xContainer(name string, requirements corev1.ResourceRequirements) corev1.Container {
	return corev1.Container{
		Name:      name,
		Image:     s390xTestImage,
		Resources: requirements,
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: ptr.To(false),
			Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
			RunAsNonRoot:             ptr.To(true),
			SeccompProfile:           &corev1.SeccompProfile{Type: "RuntimeDefault"},
		},
	}
}

// TestIBMZCPURequestToRequestPercentNoCPULimit verifies that on a real s390x node a
// pod with a CPU request and no CPU limit has its request scaled by
// cpuRequestToRequestPercent, and that the original request is preserved via
// annotation so a later webhook reinvocation cannot compound the scaling.
//
// Skipped on clusters without schedulable s390x nodes: the general
// cpuRequestToRequestPercent mutation math is already covered by
// TestClusterResourceOverrideAdmissionWithCPURequestToRequestPercent in e2e_test.go.
// The unique value here is real IBM Z hardware pinning and the annotation assertion.
func TestIBMZCPURequestToRequestPercentNoCPULimit(t *testing.T) {
	client := helper.NewClient(t, options.config)

	if !helper.HasNodesWithArch(t, client.Kubernetes, s390xArch) {
		t.Skipf("no schedulable s390x nodes in this cluster — skipping IBM Z real-hardware test " +
			"(mutation math covered by TestClusterResourceOverrideAdmissionWithCPURequestToRequestPercent)")
	}

	f := &helper.PreCondition{Client: client.Kubernetes}
	f.MustHaveAdmissionRegistrationV1(t)

	override := operatorv1.PodResourceOverride{
		Spec: operatorv1.PodResourceOverrideSpec{
			CPURequestToRequestPercent: 50,
		},
	}
	current, changed := helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override, nil)
	defer helper.RemoveAdmissionWebhook(t, client.Operator, current.GetName())
	helper.Wait(t, client.Operator, "cluster", helper.GetAvailableConditionFunc(current, changed))

	ns, disposer := helper.NewNamespace(t, client.Kubernetes, "ibmz-e2e", true)
	defer disposer.Dispose()

	spec := corev1.PodSpec{
		NodeSelector: map[string]string{"kubernetes.io/arch": s390xArch},
		Containers: []corev1.Container{
			s390xContainer("app", corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("1000m"),
				},
			}),
		},
	}

	t.Log("pinning test pod to s390x node to verify real IBM Z scheduling/runtime")
	podGot, podDisposer := helper.NewPod(t, client.Kubernetes, ns.GetName(), spec)
	defer podDisposer.Dispose()

	resourceWant := map[string]corev1.ResourceRequirements{
		"app": {
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("500m"),
			},
		},
	}
	helper.MustMatchMemoryAndCPU(t, resourceWant, &podGot.Spec)

	require.Containsf(t, podGot.Annotations,
		"clusterresourceoverrides.admission.autoscaling.openshift.io/original-cpu-request-app",
		"expected the original CPU request to be recorded so a later webhook reinvocation can't compound the scaling")
}
