package e2e

// ibmz_test.go contains IBM Z coverage for the CROO e2e suite.
//
// TestIBMZCPURequestToRequestPercentNoCPULimit is s390x hardware-gated: it calls
// t.Skip when no schedulable s390x node is present. It verifies admission mutation
// AND that the pod actually reaches Running on a real s390x node. The general
// cpuRequestToRequestPercent mutation math is already covered by
// TestClusterResourceOverrideAdmissionWithCPURequestToRequestPercent in e2e_test.go;
// the unique value here is real IFL hardware execution and the full annotation contract.
//
// TestIBMZCPURequestToRequestPercentOverwritesLimitBasedRequest and
// TestIBMZResourceOverrideConflictEmitsWarningEventOnLoser cover scenarios not
// present in e2e_test.go. They verify admission mutation only (no nodeSelector) and
// run on all architectures — the behaviour they test is architecture-agnostic.

import (
	"testing"

	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// TestIBMZCPURequestToRequestPercentNoCPULimit verifies the full contract for the
// standard IBM Z workload pattern: a pod with a CPU request and no CPU limit.
//
//  1. Admission mutation: cpuRequestToRequestPercent=50 scales 1000m → 500m.
//  2. No CPU limit is fabricated by the webhook (IBM Z workloads must not have
//     limits injected — see docs on IFL scheduling).
//  3. The original 1000m request is preserved in the per-container annotation so
//     a later webhook reinvocation cannot compound the scaling.
//  4. The pod is scheduled and reaches Running on a real s390x node, confirming
//     the mutated spec is accepted by the s390x kubelet/runtime.
//
// Skipped on clusters without schedulable s390x nodes.
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

	t.Log("submitting pod pinned to s390x node — verifying admission mutation and real IBM Z execution")
	resourceWant := map[string]corev1.ResourceRequirements{
		"app": {
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("500m"),
			},
		},
	}
	// Retry until the webhook config has propagated to all operand replicas.
	podGot, podDisposer := helper.EventuallyMustMatchPodSpec(t, client.Kubernetes, ns.GetName(), spec, resourceWant)
	defer podDisposer.Dispose()

	// 1. No CPU limit must have been injected — cpuRequestToRequestPercent must
	//    not fabricate a limit when none was submitted.
	require.NotContains(t, podGot.Spec.Containers[0].Resources.Limits, corev1.ResourceCPU,
		"webhook must not inject a CPU limit when the submitted pod had none")

	// 2. Original request annotation must exist and carry the pre-mutation value.
	const annotationKey = "clusterresourceoverrides.admission.autoscaling.openshift.io/original-cpu-request-app"
	require.Contains(t, podGot.Annotations, annotationKey,
		"original CPU request annotation must be present to prevent compounding on webhook reinvocation")
	originalCPU := resource.MustParse("1000m")
	require.Equal(t, originalCPU.String(), podGot.Annotations[annotationKey],
		"annotation must record the original 1000m request, not the mutated 500m value")

	// 3. Pod must reach Running on an s390x node — confirms the mutated spec is
	//    accepted by the real s390x kubelet and container runtime.
	nodeName := helper.WaitForPodRunningOnNode(t, client.Kubernetes, ns.GetName(), podGot.Name)
	node, err := client.Kubernetes.CoreV1().Nodes().Get(t.Context(), nodeName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, s390xArch, node.Labels["kubernetes.io/arch"],
		"pod must have run on an s390x node, not %s", nodeName)
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
	_, podDisposer := helper.EventuallyMustMatchPodSpec(t, client.Kubernetes, ns.GetName(), spec, resourceWant)
	defer podDisposer.Dispose()
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
