package e2e

// ibmz_test.go contains IBM Z coverage for the CROO e2e suite.
//
// TestIBMZCPURequestToRequestPercentNoCPULimit is s390x hardware-gated: it calls
// t.Skip when no schedulable s390x node is present, so it only runs on real IBM Z
// clusters. The mutation math it exercises is already covered by
// TestClusterResourceOverrideAdmissionWithCPURequestToRequestPercent in e2e_test.go;
// the unique value here is real IFL hardware pinning and the annotation assertion.
//
// TestIBMZCPURequestToRequestPercentOverwritesLimitBasedRequest and
// TestIBMZResourceOverrideConflictEmitsWarningEventOnLoser cover scenarios not
// present in e2e_test.go and run on all architectures — they were motivated by
// IBM Z workload patterns but the behaviour they test is architecture-agnostic.

import (
	"testing"

	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"

	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
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
	resourceWant := map[string]corev1.ResourceRequirements{
		"app": {
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("500m"),
			},
		},
	}
	// Use EventuallyMustMatchPodSpec rather than a single NewPod: the webhook
	// configuration may not have propagated to all operand replicas yet even after
	// Available=True. Retrying avoids spurious failures after a config-change test.
	podGot, podDisposer := helper.EventuallyMustMatchPodSpec(t, client.Kubernetes, ns.GetName(), spec, resourceWant)
	defer podDisposer.Dispose()

	require.Containsf(t, podGot.Annotations,
		"clusterresourceoverrides.admission.autoscaling.openshift.io/original-cpu-request-app",
		"expected the original CPU request to be recorded so a later webhook reinvocation can't compound the scaling")
}

// TestIBMZCPURequestToRequestPercentOverwritesLimitBasedRequest covers a profile
// that configures both cpuRequestToLimitPercent and cpuRequestToRequestPercent
// together. cpuRequestToRequestPercent always runs last and overwrites the result
// of cpuRequestToLimitPercent, deriving from the pod's original CPU request
// (preserved via annotation) rather than the intermediate value already written.
func TestIBMZCPURequestToRequestPercentOverwritesLimitBasedRequest(t *testing.T) {
	client := helper.NewClient(t, options.config)

	f := &helper.PreCondition{Client: client.Kubernetes}
	f.MustHaveAdmissionRegistrationV1(t)

	override := operatorv1.PodResourceOverride{
		Spec: operatorv1.PodResourceOverrideSpec{
			CPURequestToLimitPercent:   25, // would set requests.cpu = 25% of the 4000m limit = 1000m
			CPURequestToRequestPercent: 50, // overwrites with 50% of the *original* 800m request = 400m
		},
	}
	current, changed := helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override, nil)
	defer helper.RemoveAdmissionWebhook(t, client.Operator, current.GetName())
	helper.Wait(t, client.Operator, "cluster", helper.GetAvailableConditionFunc(current, changed))

	ns, disposer := helper.NewNamespace(t, client.Kubernetes, "ibmz-e2e", true)
	defer disposer.Dispose()

	spec := corev1.PodSpec{
		Containers: []corev1.Container{
			s390xContainer("app", corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("800m"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("4000m"),
				},
			}),
		},
	}

	resourceWant := map[string]corev1.ResourceRequirements{
		"app": {
			Limits: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("4000m"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("400m"),
			},
		},
	}
	podGot, podDisposer := helper.EventuallyMustMatchPodSpec(t, client.Kubernetes, ns.GetName(), spec, resourceWant)
	defer podDisposer.Dispose()
	_ = podGot
}

// TestIBMZResourceOverrideConflictEmitsWarningEventOnLoser covers the namespace-scoped
// ResourceOverride path: with two ResourceOverride objects in the same namespace both
// matching the pod (empty podSelector on each), the lexicographically-first name wins,
// and the losing object gets a Warning "OverrideConflict" event rather than blocking
// admission.
func TestIBMZResourceOverrideConflictEmitsWarningEventOnLoser(t *testing.T) {
	client := helper.NewClient(t, options.config)

	f := &helper.PreCondition{Client: client.Kubernetes}
	f.MustHaveAdmissionRegistrationV1(t)

	// Cluster-wide fallback deliberately different from either RO below, to confirm
	// the namespace-scoped RO path is what takes effect.
	override := operatorv1.PodResourceOverride{
		Spec: operatorv1.PodResourceOverrideSpec{
			CPURequestToRequestPercent: 90,
		},
	}
	current, changed := helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override, nil)
	defer helper.RemoveAdmissionWebhook(t, client.Operator, current.GetName())
	helper.Wait(t, client.Operator, "cluster", helper.GetAvailableConditionFunc(current, changed))

	ns, nsDisposer := helper.NewNamespace(t, client.Kubernetes, "ibmz-e2e", true)
	defer nsDisposer.Dispose()

	const winnerName = "a-ibmz-ro"
	const loserName = "b-ibmz-ro"

	winnerSpec := autoscalingv1.ResourceOverrideSpec{
		PodResourceOverride: autoscalingv1.PodResourceOverrideSpec{
			CPURequestToRequestPercent: 50,
		},
	}
	_, winnerDisposer := helper.CreateResourceOverride(t, client.Operator, ns.GetName(), winnerName, winnerSpec)
	defer winnerDisposer.Dispose()
	helper.WaitForResourceOverrideCondition(t, client.Operator, ns.GetName(), winnerName, helper.IsResourceOverrideValidationPassing)

	loserSpec := autoscalingv1.ResourceOverrideSpec{
		PodResourceOverride: autoscalingv1.PodResourceOverrideSpec{
			CPURequestToRequestPercent: 75,
		},
	}
	_, loserDisposer := helper.CreateResourceOverride(t, client.Operator, ns.GetName(), loserName, loserSpec)
	defer loserDisposer.Dispose()
	helper.WaitForResourceOverrideCondition(t, client.Operator, ns.GetName(), loserName, helper.IsResourceOverrideValidationPassing)

	spec := corev1.PodSpec{
		Containers: []corev1.Container{
			s390xContainer("app", corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("1000m"),
				},
			}),
		},
	}

	// winnerName sorts before loserName lexicographically, so its 50% ratio applies —
	// not the loser's 75%, and not the cluster fallback's 90%.
	resourceWant := map[string]corev1.ResourceRequirements{
		"app": {
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("500m"),
			},
		},
	}
	// Use EventuallyMustMatchPodSpec: the ResourceOverride informer may not have
	// synced to all webhook replicas yet. Retry until the expected mutation lands
	// before asserting the event.
	_, podDisposer := helper.EventuallyMustMatchPodSpec(t, client.Kubernetes, ns.GetName(), spec, resourceWant)
	defer podDisposer.Dispose()

	event := helper.WaitForWarningEvent(t, client.Kubernetes, ns.GetName(), "OverrideConflict", loserName)
	require.Containsf(t, event.Message, winnerName, "expected the conflict event on the loser to name the winning ResourceOverride")
}
