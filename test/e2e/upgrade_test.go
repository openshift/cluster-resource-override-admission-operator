package e2e

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	operatorv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/operator/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/test/helper"
)

// upgradeTestRequirements are the resource limits injected into test pods.
// The CR config (50/25/200) mutates these to upgradeTestExpected.
var upgradeTestRequirements = corev1.ResourceRequirements{
	Limits: corev1.ResourceList{
		corev1.ResourceMemory: resource.MustParse("512Mi"),
		corev1.ResourceCPU:    resource.MustParse("2000m"),
	},
}

// upgradeTestExpected is what the webhook should produce given
// upgradeTestRequirements and the CR config (50/25/200):
//
//	limitCPUToMemoryPercent=200  -> CPU limit  = 200% * 0.5Gi = 1000m
//	memoryRequestToLimitPercent=50 -> memory request = 256Mi
//	cpuRequestToLimitPercent=25    -> CPU request    = 250m
var upgradeTestExpected = map[string]corev1.ResourceRequirements{
	"test": {
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("1000m"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("256Mi"),
			corev1.ResourceCPU:    resource.MustParse("250m"),
		},
	},
}

// TestUpgradePre creates the ClusterResourceOverride CR against the old
// operator version and validates that the operand and webhook are functioning.
// Run this after the old bundle is installed but before the upgrade.
// The CR is intentionally not cleaned up so TestUpgradePost can verify it
// survived the upgrade.
func TestUpgradePre(t *testing.T) {
	client := helper.NewClient(t, options.config)

	override := operatorv1.PodResourceOverride{
		Spec: operatorv1.PodResourceOverrideSpec{
			LimitCPUToMemoryPercent:     200,
			CPURequestToLimitPercent:    25,
			MemoryRequestToLimitPercent: 50,
		},
	}
	current, changed := helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override, nil)
	// Do not defer RemoveAdmissionWebhook: the CR must survive for the post-upgrade check.

	t.Log("waiting for CR to become Available")
	helper.Wait(t, client.Operator, "cluster", helper.GetAvailableConditionFunc(current, changed))

	t.Log("waiting for operand deployment rollout")
	helper.WaitForDeploymentRollout(t, client.Kubernetes, operatorNamespace, "clusterresourceoverride")

	f := &helper.PreCondition{Client: client.Kubernetes}
	f.MustHaveClusterResourceOverrideAdmissionConfiguration(t)

	ns, nsDisposer := helper.NewNamespace(t, client.Kubernetes, "upgrade-verify", true)
	defer nsDisposer.Dispose()

	t.Log("verifying webhook mutates pods correctly")
	_, podDisposer := helper.EventuallyMustMatchPodMutation(t, client.Kubernetes, ns.Name, "test", upgradeTestRequirements, upgradeTestExpected)
	defer podDisposer.Dispose()
}

// TestUpgradePost validates that a pre-existing ClusterResourceOverride CR
// survived the bundle upgrade with its spec intact, status healthy, operand
// running, and webhook still mutating pods.
func TestUpgradePost(t *testing.T) {
	client := helper.NewClient(t, options.config)

	t.Log("verifying CR still exists after upgrade")
	helper.GetClusterResourceOverride(t, client.Operator, "cluster")
	defer helper.RemoveAdmissionWebhook(t, client.Operator, "cluster")

	t.Log("waiting for CR to become Available after upgrade")
	helper.Wait(t, client.Operator, "cluster", helper.IsAvailable)

	t.Log("waiting for operand deployment rollout to complete")
	// We have to wait for a full rollout because the operand has a pod anti-affinity that prevents pods from running on the same node.
	// To prevent pods from getting stuck in rollout if there are not enough nodes, the Deployment rolloutStrategy has a maxSurge of 0.
	// Therefore, old operand pods get removed before new ones start and the webhook can be briefly unreachable, and flake the test.
	helper.WaitForDeploymentRollout(t, client.Kubernetes, operatorNamespace, "clusterresourceoverride")

	f := &helper.PreCondition{Client: client.Kubernetes}
	f.MustHaveClusterResourceOverrideAdmissionConfiguration(t)

	ns, nsDisposer := helper.NewNamespace(t, client.Kubernetes, "upgrade-verify", true)
	defer nsDisposer.Dispose()

	t.Log("verifying webhook still mutates pods after upgrade")
	_, podDisposer := helper.EventuallyMustMatchPodMutation(t, client.Kubernetes, ns.Name, "test", upgradeTestRequirements, upgradeTestExpected)
	defer podDisposer.Dispose()
}
