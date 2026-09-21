package e2e

// Tests in this file cover the cpuRequestToRequestPercent enhancement that IBM Z
// (s390x) workloads rely on (AUTOSCALE-705 / MULTIARCH-5562): on s390x, CPU limits
// aren't fabricated from memory the way amd64 profiles do, so charts typically ship
// a CPU *request* with no CPU *limit* and need that request scaled directly.
//
// The mutation math itself is architecture-agnostic Go code, so these tests run on
// any cluster. TestIBMZCPURequestToRequestPercentNoCPULimit additionally pins the
// pod to a real s390x node via nodeSelector when the cluster has one, to prove the
// same behavior holds under the actual kubelet/runtime; on amd64-only dev clusters
// that pinning is skipped and only the mutation math is verified.
//
// Idempotency of the annotation-backed original-request lookup (no compounding on
// webhook reinvocation) is intentionally NOT re-tested here — it requires a second
// interfering mutating webhook to trigger a real reinvocation and is already covered
// at the unit level in the admission repo's mutator_test.go.

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

func s390xContainer(name string, requirements corev1.ResourceRequirements) corev1.Container {
	return corev1.Container{
		Name:      name,
		Image:     "openshift/hello-openshift",
		Resources: requirements,
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: ptr.To(false),
			Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
			RunAsNonRoot:             ptr.To(true),
			SeccompProfile:           &corev1.SeccompProfile{Type: "RuntimeDefault"},
		},
	}
}

// TestIBMZCPURequestToRequestPercentNoCPULimit covers the base IBM Z scenario: a pod
// with a CPU request and no CPU limit gets its request scaled by
// cpuRequestToRequestPercent, since limitCPUToMemoryPercent never runs without a
// memory limit to derive a CPU limit from, so OverrideCPUWithLimit (step 3) is a
// no-op and OverrideCPUWithRequest (step 4) is the only thing that can act.
func TestIBMZCPURequestToRequestPercentNoCPULimit(t *testing.T) {
	client := helper.NewClient(t, options.config)

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
		Containers: []corev1.Container{
			s390xContainer("app", corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("1000m"),
				},
			}),
		},
	}

	if helper.HasNodesWithArch(t, client.Kubernetes, s390xArch) {
		t.Log("s390x nodes detected in this cluster — pinning the test pod to verify real IBM Z scheduling/runtime")
		spec.NodeSelector = map[string]string{"kubernetes.io/arch": s390xArch}
	} else {
		t.Log("no s390x nodes in this cluster — verifying mutation math only (see docs/S390X_E2E_TEST_PLAN.md Tier 2 for real-hardware coverage)")
	}

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

// TestIBMZCPURequestToRequestPercentOverwritesLimitBasedRequest covers a profile
// that configures both cpuRequestToLimitPercent (step 3) and
// cpuRequestToRequestPercent (step 4) together. Per the documented mutator
// ordering, step 4 always runs last and overwrites step 3's result, deriving
// from the pod's *original* CPU request (preserved via annotation before step 3
// runs), not from whatever step 3 already wrote into requests.cpu.
func TestIBMZCPURequestToRequestPercentOverwritesLimitBasedRequest(t *testing.T) {
	client := helper.NewClient(t, options.config)

	f := &helper.PreCondition{Client: client.Kubernetes}
	f.MustHaveAdmissionRegistrationV1(t)

	override := operatorv1.PodResourceOverride{
		Spec: operatorv1.PodResourceOverrideSpec{
			CPURequestToLimitPercent:   25, // step 3: would set requests.cpu = 25% of the 4000m limit = 1000m
			CPURequestToRequestPercent: 50, // step 4: overwrites with 50% of the *original* 800m request = 400m
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

	podGot, podDisposer := helper.NewPod(t, client.Kubernetes, ns.GetName(), spec)
	defer podDisposer.Dispose()

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
	helper.MustMatchMemoryAndCPU(t, resourceWant, &podGot.Spec)
}

// TestIBMZResourceOverrideConflictEmitsWarningEventOnLoser covers the namespace-scoped
// ResourceOverride path for an IBM Z profile: with two ResourceOverride objects in the
// same namespace both matching the pod (empty podSelector on each), the
// lexicographically-first name wins, and the losing object gets a Warning
// "OverrideConflict" event rather than blocking admission.
func TestIBMZResourceOverrideConflictEmitsWarningEventOnLoser(t *testing.T) {
	client := helper.NewClient(t, options.config)

	f := &helper.PreCondition{Client: client.Kubernetes}
	f.MustHaveAdmissionRegistrationV1(t)

	// Cluster-wide fallback config, deliberately different from either RO below so we
	// can confirm the RO path is actually what's taking effect.
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
	podGot, podDisposer := helper.NewPod(t, client.Kubernetes, ns.GetName(), spec)
	defer podDisposer.Dispose()

	// winnerName sorts before loserName lexicographically, so its 50% ratio applies —
	// not the loser's 75%, and not the cluster fallback's 90%.
	resourceWant := map[string]corev1.ResourceRequirements{
		"app": {
			Requests: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("500m"),
			},
		},
	}
	helper.MustMatchMemoryAndCPU(t, resourceWant, &podGot.Spec)

	event := helper.WaitForWarningEvent(t, client.Kubernetes, ns.GetName(), "OverrideConflict", loserName)
	require.Containsf(t, event.Message, winnerName, "expected the conflict event on the loser to name the winning ResourceOverride")
}
