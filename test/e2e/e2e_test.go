package e2e

import (
	"context"
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

const (
	operatorNamespace = "openshift-cluster-resource-override"
	configMapName     = "clusterresourceoverride-configuration"
)

func TestClusterResourceOverrideAdmissionWithOptIn(t *testing.T) {
	tests := []struct {
		name           string
		request        *corev1.PodSpec
		limitRangeSpec *corev1.LimitRangeSpec
		resourceWant   map[string]corev1.ResourceRequirements
	}{
		{
			name: "WithMultipleContainers",
			request: &corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "db",
						Image: "openshift/hello-openshift",
						Ports: []corev1.ContainerPort{
							{
								Name:          "db",
								ContainerPort: 60000,
							},
						},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("1024Mi"),
								corev1.ResourceCPU:    resource.MustParse("1000m"),
							},
						},
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: ptr.To(false),
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{"ALL"},
							},
							RunAsNonRoot: ptr.To(true),
							SeccompProfile: &corev1.SeccompProfile{
								Type: "RuntimeDefault",
							},
						},
					},
					{
						Name:  "app",
						Image: "openshift/hello-openshift",
						Ports: []corev1.ContainerPort{
							{
								Name:          "app",
								ContainerPort: 60100,
							},
						},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("512Mi"),
								corev1.ResourceCPU:    resource.MustParse("500m"),
							},
						},
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: ptr.To(false),
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{"ALL"},
							},
							RunAsNonRoot: ptr.To(true),
							SeccompProfile: &corev1.SeccompProfile{
								Type: "RuntimeDefault",
							},
						},
					},
				},
			},
			resourceWant: map[string]corev1.ResourceRequirements{
				"db": {
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1024Mi"),
						corev1.ResourceCPU:    resource.MustParse("2000m"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("512Mi"),
						corev1.ResourceCPU:    resource.MustParse("500m"),
					},
				},
				"app": {
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("512Mi"),
						corev1.ResourceCPU:    resource.MustParse("1000m"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("256Mi"),
						corev1.ResourceCPU:    resource.MustParse("250m"),
					},
				},
			},
		},
		{
			name: "WithInitContainer",
			request: &corev1.PodSpec{
				InitContainers: []corev1.Container{
					{
						Name:  "init",
						Image: "busybox:latest",
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("1024Mi"),
								corev1.ResourceCPU:    resource.MustParse("1000m"),
							},
						},
						Command: []string{
							"sh",
							"-c",
							"echo The app is running! && sleep 1",
						},
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: ptr.To(false),
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{"ALL"},
							},
							RunAsNonRoot: ptr.To(true),
							SeccompProfile: &corev1.SeccompProfile{
								Type: "RuntimeDefault",
							},
						},
					},
				},
				Containers: []corev1.Container{
					{
						Name:  "app",
						Image: "openshift/hello-openshift",
						Ports: []corev1.ContainerPort{
							{
								Name:          "app",
								ContainerPort: 60100,
							},
						},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("512Mi"),
								corev1.ResourceCPU:    resource.MustParse("500m")},
						},
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: ptr.To(false),
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{"ALL"},
							},
							RunAsNonRoot: ptr.To(true),
							SeccompProfile: &corev1.SeccompProfile{
								Type: "RuntimeDefault",
							},
						},
					},
				},
			},
			resourceWant: map[string]corev1.ResourceRequirements{
				"init": {
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1024Mi"),
						corev1.ResourceCPU:    resource.MustParse("2000m"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("512Mi"),
						corev1.ResourceCPU:    resource.MustParse("500m"),
					},
				},
				"app": {
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("512Mi"),
						corev1.ResourceCPU:    resource.MustParse("1000m"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("256Mi"),
						corev1.ResourceCPU:    resource.MustParse("250m"),
					},
				},
			},
		},

		{
			name: "WithLimitRangeWithDefaultLimitForCPUAndMemory",
			limitRangeSpec: &corev1.LimitRangeSpec{
				Limits: []corev1.LimitRangeItem{
					{
						Type: corev1.LimitTypeContainer,
						Default: corev1.ResourceList{
							corev1.ResourceMemory: resource.MustParse("512Mi"),
							corev1.ResourceCPU:    resource.MustParse("2000m"),
						},
					},
				},
			},
			request: &corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "app",
						Image: "openshift/hello-openshift",
						Ports: []corev1.ContainerPort{
							{
								Name:          "app",
								ContainerPort: 60100,
							},
						},
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: ptr.To(false),
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{"ALL"},
							},
							RunAsNonRoot: ptr.To(true),
							SeccompProfile: &corev1.SeccompProfile{
								Type: "RuntimeDefault",
							},
						},
					},
				},
			},
			resourceWant: map[string]corev1.ResourceRequirements{
				"app": {
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("512Mi"),
						corev1.ResourceCPU:    resource.MustParse("1000m"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("256Mi"),
						corev1.ResourceCPU:    resource.MustParse("250m"),
					},
				},
			},
		},

		// LimitRange Maximum for CPU is 1000m, the operator, as expected is going to
		// override the CPU limit of the Pod to 2000m (since LimitCPUToMemoryPercent=200).
		// But then it should clamp it to the namespace Limit Maximum.
		{
			name: "WithLimitRangeWithMaximumForCPU",
			limitRangeSpec: &corev1.LimitRangeSpec{
				Limits: []corev1.LimitRangeItem{
					{
						Type: corev1.LimitTypeContainer,
						Max: corev1.ResourceList{
							corev1.ResourceMemory: resource.MustParse("1024Mi"),
							corev1.ResourceCPU:    resource.MustParse("1000m"),
						},
					},
				},
			},
			request: &corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "app",
						Image: "openshift/hello-openshift",
						Ports: []corev1.ContainerPort{
							{
								Name:          "app",
								ContainerPort: 60100,
							},
						},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("1024Mi"),
								corev1.ResourceCPU:    resource.MustParse("1000m")},
						},
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: ptr.To(false),
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{"ALL"},
							},
							RunAsNonRoot: ptr.To(true),
							SeccompProfile: &corev1.SeccompProfile{
								Type: "RuntimeDefault",
							},
						},
					},
				},
			},
			resourceWant: map[string]corev1.ResourceRequirements{
				"app": {
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1024Mi"),
						corev1.ResourceCPU:    resource.MustParse("1000m"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("512Mi"),
						corev1.ResourceCPU:    resource.MustParse("250m"),
					},
				},
			},
		},
	}

	client := helper.NewClient(t, options.config)

	f := &helper.PreCondition{Client: client.Kubernetes}
	f.MustHaveAdmissionRegistrationV1(t)

	// ensure we have the webhook up and running with the desired config
	configuration := operatorv1.PodResourceOverrideSpec{
		LimitCPUToMemoryPercent:     200,
		CPURequestToLimitPercent:    25,
		MemoryRequestToLimitPercent: 50,
	}
	override := operatorv1.PodResourceOverride{
		Spec: configuration,
	}

	t.Logf("setting webhook configuration - %s", configuration.String())
	current, changed := helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override, nil)
	defer helper.RemoveAdmissionWebhook(t, client.Operator, current.GetName())

	t.Log("waiting for webhook configuration to take effect")
	current = helper.Wait(t, client.Operator, "cluster", helper.GetAvailableConditionFunc(current, changed))

	f.MustHaveClusterResourceOverrideAdmissionConfiguration(t)
	t.Log("webhook configuration has been set successfully")

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			func() {
				// ensure a namespace that is properly labeled
				ns, disposer := helper.NewNamespace(t, client.Kubernetes, "croe2e", true)
				defer disposer.Dispose()
				namespace := ns.GetName()

				// make sure we add limit range for the namespace.
				if test.limitRangeSpec != nil {
					_, disposer := helper.NewLimitRanges(t, client.Kubernetes, namespace, *test.limitRangeSpec)
					defer disposer.Dispose()
				}

				podGot, disposer := helper.NewPod(t, client.Kubernetes, namespace, *test.request)
				defer disposer.Dispose()

				helper.MustMatchMemoryAndCPU(t, test.resourceWant, &podGot.Spec)
			}()
		})
	}
}

func TestClusterResourceOverrideAdmissionWithConfigurationChange(t *testing.T) {
	client := helper.NewClient(t, options.config)

	f := &helper.PreCondition{Client: client.Kubernetes}
	f.MustHaveAdmissionRegistrationV1(t)

	before := operatorv1.PodResourceOverrideSpec{
		LimitCPUToMemoryPercent:     100,
		CPURequestToLimitPercent:    10,
		MemoryRequestToLimitPercent: 75,
		ForceSelinuxRelabel:         false,
	}
	override := operatorv1.PodResourceOverride{
		Spec: before,
	}
	croSpec := operatorv1.ClusterResourceOverrideSpec{
		PodResourceOverride: override,
	}

	t.Logf("initial configuration - %s", before.String())

	current, changed := helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override, nil)
	defer helper.RemoveAdmissionWebhook(t, client.Operator, current.GetName())

	current = helper.Wait(t, client.Operator, "cluster", helper.GetAvailableConditionFunc(current, changed))
	require.Equal(t, croSpec.Hash(), current.Status.Hash.Configuration)

	after := operatorv1.PodResourceOverrideSpec{
		LimitCPUToMemoryPercent:     50,
		CPURequestToLimitPercent:    50,
		MemoryRequestToLimitPercent: 50,
		ForceSelinuxRelabel:         false,
	}
	override = operatorv1.PodResourceOverride{
		Spec: after,
	}
	croSpec.PodResourceOverride = override

	t.Logf("second configuration - %s", after.String())

	current, changed = helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override, nil)
	current = helper.Wait(t, client.Operator, "cluster", helper.GetAvailableConditionFunc(current, changed))
	require.Equal(t, croSpec.Hash(), current.Status.Hash.Configuration)

	// create a new Pod, we expect the Pod resources to be overridden based of the new configuration.
	ns, disposer := helper.NewNamespace(t, client.Kubernetes, "croe2e", true)
	defer disposer.Dispose()

	requirements := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("1024Mi"),
			corev1.ResourceCPU:    resource.MustParse("1000m"),
		},
	}

	resourceWant := map[string]corev1.ResourceRequirements{
		"test": {
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("1024Mi"),
				corev1.ResourceCPU:    resource.MustParse("500m"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("250m"),
			},
		},
	}

	podGot, disposer := helper.NewPodWithResourceRequirement(t, client.Kubernetes, ns.GetName(), "test", requirements)
	defer disposer.Dispose()

	helper.MustMatchMemoryAndCPU(t, resourceWant, &podGot.Spec)

	// test changing ForceSelinuxRelabel changes the configuration hash and reconciles the configMap
	after.ForceSelinuxRelabel = true
	override = operatorv1.PodResourceOverride{
		Spec: after,
	}
	croSpec.PodResourceOverride = override

	t.Logf("final configuration: forceSelinuxRelabel - %s", after.String())

	originalCm := helper.GetConfigMap(t, client.Kubernetes, operatorNamespace, configMapName)

	current, changed = helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override, nil)
	current = helper.Wait(t, client.Operator, "cluster", helper.GetAvailableConditionFunc(current, changed))
	require.Equal(t, croSpec.Hash(), current.Status.Hash.Configuration)

	cm := helper.WaitForConfigMap(t, client.Kubernetes, operatorNamespace, configMapName, originalCm)
	rawData := cm.Data["configuration.yaml"]
	require.Contains(t, rawData, "forceSelinuxRelabel: true")
	require.NotContains(t, rawData, "forceSelinuxRelabel: false")
}

func TestClusterResourceOverrideAdmissionWithNoOptIn(t *testing.T) {
	client := helper.NewClient(t, options.config)

	f := &helper.PreCondition{Client: client.Kubernetes}
	f.MustHaveAdmissionRegistrationV1(t)

	configuration := operatorv1.PodResourceOverrideSpec{
		LimitCPUToMemoryPercent:     200,
		CPURequestToLimitPercent:    50,
		MemoryRequestToLimitPercent: 50,
	}
	override := operatorv1.PodResourceOverride{
		Spec: configuration,
	}

	current, changed := helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override, nil)
	defer helper.RemoveAdmissionWebhook(t, client.Operator, current.GetName())

	current = helper.Wait(t, client.Operator, "cluster", helper.GetAvailableConditionFunc(current, changed))

	// make sure everything works after cert is regenerated
	ns, disposer := helper.NewNamespace(t, client.Kubernetes, "croe2e", false)
	defer disposer.Dispose()

	requirements := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
	}

	resourceWant := map[string]corev1.ResourceRequirements{
		"test": {
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
		},
	}

	podGot, disposer := helper.NewPodWithResourceRequirement(t, client.Kubernetes, ns.GetName(), "test", requirements)
	defer disposer.Dispose()

	helper.MustMatchMemoryAndCPU(t, resourceWant, &podGot.Spec)
}

func TestClusterResourceOverrideDeploymentOverrides(t *testing.T) {
	client := helper.NewClient(t, options.config)

	f := &helper.PreCondition{Client: client.Kubernetes}
	f.MustHaveAdmissionRegistrationV1(t)

	// Set up the CRD object with the desired configurations
	configuration := operatorv1.PodResourceOverrideSpec{
		LimitCPUToMemoryPercent:     200,
		CPURequestToLimitPercent:    25,
		MemoryRequestToLimitPercent: 50,
	}
	deploymentOverrides := operatorv1.DeploymentOverrides{
		Replicas: ptr.To[int32](1),
		NodeSelector: map[string]string{
			"node-role.kubernetes.io/worker": "",
		},
		Tolerations: []corev1.Toleration{
			{
				Key:      "key",
				Operator: corev1.TolerationOpEqual,
				Value:    "value",
				Effect:   corev1.TaintEffectNoSchedule,
			},
		},
	}
	override := operatorv1.PodResourceOverride{
		Spec: configuration,
	}

	t.Logf("setting webhook configuration - %s", configuration.String())
	current, changed := helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override, &deploymentOverrides)
	defer helper.RemoveAdmissionWebhook(t, client.Operator, current.GetName())

	t.Log("waiting for webhook configuration to take effect")
	current = helper.Wait(t, client.Operator, "cluster", helper.GetAvailableConditionFunc(current, changed))

	f.MustHaveClusterResourceOverrideAdmissionConfiguration(t)
	t.Log("webhook configuration has been set successfully")

	// Verify the deployment created by the operator matches the deployment overrides
	deployment := helper.GetDeployment(t, client.Kubernetes, operatorNamespace, "clusterresourceoverride")
	require.Equal(t, *deployment.Spec.Replicas, *deploymentOverrides.Replicas)
	require.Equal(t, deployment.Spec.Template.Spec.NodeSelector, deploymentOverrides.NodeSelector)
	require.Equal(t, deployment.Spec.Template.Spec.Tolerations, deploymentOverrides.Tolerations)
}

func TestClusterResourceOverrideAdmissionWithCPURequestToRequestPercent(t *testing.T) {
	client := helper.NewClient(t, options.config)

	f := &helper.PreCondition{Client: client.Kubernetes}
	f.MustHaveAdmissionRegistrationV1(t)

	configuration := operatorv1.PodResourceOverrideSpec{
		CPURequestToRequestPercent:  50,
		MemoryRequestToLimitPercent: 50,
	}
	override := operatorv1.PodResourceOverride{
		Spec: configuration,
	}

	t.Logf("setting webhook configuration - %s", configuration.String())
	current, changed := helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override, nil)
	defer helper.RemoveAdmissionWebhook(t, client.Operator, current.GetName())

	t.Log("waiting for webhook configuration to take effect")
	current = helper.Wait(t, client.Operator, "cluster", helper.GetAvailableConditionFunc(current, changed))

	f.MustHaveClusterResourceOverrideAdmissionConfiguration(t)
	t.Log("webhook configuration has been set successfully")

	ns, disposer := helper.NewNamespace(t, client.Kubernetes, "croe2e", true)
	defer disposer.Dispose()

	requirements := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse("200m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
	}

	resourceWant := map[string]corev1.ResourceRequirements{
		"app": {
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("256Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
		},
	}

	podGot, disposer := helper.NewPodWithResourceRequirement(t, client.Kubernetes, ns.GetName(), "app", requirements)
	defer disposer.Dispose()

	helper.MustMatchMemoryAndCPU(t, resourceWant, &podGot.Spec)
}

func TestResourceOverrideAdmissionOverridesPod(t *testing.T) {
	client := helper.NewClient(t, options.config)

	f := &helper.PreCondition{Client: client.Kubernetes}
	f.MustHaveAdmissionRegistrationV1(t)

	croConfig := operatorv1.PodResourceOverrideSpec{
		LimitCPUToMemoryPercent:     200,
		CPURequestToLimitPercent:    25,
		MemoryRequestToLimitPercent: 50,
	}
	override := operatorv1.PodResourceOverride{
		Spec: croConfig,
	}

	t.Logf("setting CRO webhook configuration - %s", croConfig.String())
	current, changed := helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override, nil)
	defer helper.RemoveAdmissionWebhook(t, client.Operator, current.GetName())

	t.Log("waiting for CRO webhook to become available")
	current = helper.Wait(t, client.Operator, "cluster", helper.GetAvailableConditionFunc(current, changed))

	ns, nsDisposer := helper.NewNamespace(t, client.Kubernetes, "roe2e", true)
	defer nsDisposer.Dispose()

	roSpec := autoscalingv1.ResourceOverrideSpec{
		PodResourceOverride: autoscalingv1.PodResourceOverrideSpec{
			LimitCPUToMemoryPercent:     100,
			CPURequestToLimitPercent:    50,
			MemoryRequestToLimitPercent: 75,
		},
	}
	t.Log("creating ResourceOverride with different ratios than CRO")
	_, roDisposer := helper.CreateResourceOverride(t, client.Operator, ns.GetName(), "test-ro", roSpec)
	defer roDisposer.Dispose()

	t.Log("waiting for ResourceOverride validation to pass")
	helper.WaitForResourceOverrideCondition(t, client.Operator, ns.GetName(), "test-ro", helper.IsResourceOverrideValidationPassing)

	t.Log("creating pod and verifying RO ratios take precedence over CRO")
	requirements := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("1024Mi"),
			corev1.ResourceCPU:    resource.MustParse("1000m"),
		},
	}

	resourceWant := map[string]corev1.ResourceRequirements{
		"test": {
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("1024Mi"),
				corev1.ResourceCPU:    resource.MustParse("1000m"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("768Mi"),
				corev1.ResourceCPU:    resource.MustParse("500m"),
			},
		},
	}

	podGot, podDisposer := helper.NewPodWithResourceRequirement(t, client.Kubernetes, ns.GetName(), "test", requirements)
	defer podDisposer.Dispose()

	helper.MustMatchMemoryAndCPU(t, resourceWant, &podGot.Spec)
}

func TestResourceOverrideAdmissionWithPodSelector(t *testing.T) {
	client := helper.NewClient(t, options.config)

	f := &helper.PreCondition{Client: client.Kubernetes}
	f.MustHaveAdmissionRegistrationV1(t)

	croConfig := operatorv1.PodResourceOverrideSpec{
		LimitCPUToMemoryPercent:     200,
		CPURequestToLimitPercent:    25,
		MemoryRequestToLimitPercent: 50,
	}
	override := operatorv1.PodResourceOverride{
		Spec: croConfig,
	}

	t.Logf("setting CRO webhook configuration - %s", croConfig.String())
	current, changed := helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override, nil)
	defer helper.RemoveAdmissionWebhook(t, client.Operator, current.GetName())

	t.Log("waiting for CRO webhook to become available")
	current = helper.Wait(t, client.Operator, "cluster", helper.GetAvailableConditionFunc(current, changed))

	ns, nsDisposer := helper.NewNamespace(t, client.Kubernetes, "roe2e", true)
	defer nsDisposer.Dispose()

	roSpec := autoscalingv1.ResourceOverrideSpec{
		PodResourceOverride: autoscalingv1.PodResourceOverrideSpec{
			CPURequestToLimitPercent:    75,
			MemoryRequestToLimitPercent: 90,
		},
		PodSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"override": "custom",
			},
		},
	}
	t.Log("creating ResourceOverride with podSelector")
	_, roDisposer := helper.CreateResourceOverride(t, client.Operator, ns.GetName(), "test-ro-selector", roSpec)
	defer roDisposer.Dispose()

	t.Log("waiting for ResourceOverride validation to pass")
	helper.WaitForResourceOverrideCondition(t, client.Operator, ns.GetName(), "test-ro-selector", helper.IsResourceOverrideValidationPassing)

	requirements := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("1000m"),
		},
	}

	t.Log("creating pod with matching label - expecting RO ratios")
	resourceWantRO := map[string]corev1.ResourceRequirements{
		"test": {
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("1000m"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("460Mi"),
				corev1.ResourceCPU:    resource.MustParse("750m"),
			},
		},
	}

	podGot1, podDisposer1 := helper.NewPodWithLabels(t, client.Kubernetes, ns.GetName(), "test", requirements, map[string]string{"override": "custom"})
	defer podDisposer1.Dispose()

	helper.MustMatchMemoryAndCPU(t, resourceWantRO, &podGot1.Spec)

	t.Log("creating pod without matching label - expecting CRO ratios")
	resourceWantCRO := map[string]corev1.ResourceRequirements{
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

	podGot2, podDisposer2 := helper.NewPodWithResourceRequirement(t, client.Kubernetes, ns.GetName(), "test", requirements)
	defer podDisposer2.Dispose()

	helper.MustMatchMemoryAndCPU(t, resourceWantCRO, &podGot2.Spec)
}

func TestResourceOverrideFallbackToCRO(t *testing.T) {
	client := helper.NewClient(t, options.config)

	f := &helper.PreCondition{Client: client.Kubernetes}
	f.MustHaveAdmissionRegistrationV1(t)

	croConfig := operatorv1.PodResourceOverrideSpec{
		LimitCPUToMemoryPercent:     200,
		CPURequestToLimitPercent:    25,
		MemoryRequestToLimitPercent: 50,
	}
	override := operatorv1.PodResourceOverride{
		Spec: croConfig,
	}

	t.Logf("setting CRO webhook configuration - %s", croConfig.String())
	current, changed := helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override, nil)
	defer helper.RemoveAdmissionWebhook(t, client.Operator, current.GetName())

	t.Log("waiting for CRO webhook to become available")
	current = helper.Wait(t, client.Operator, "cluster", helper.GetAvailableConditionFunc(current, changed))

	ns, nsDisposer := helper.NewNamespace(t, client.Kubernetes, "roe2e", true)
	defer nsDisposer.Dispose()

	roSpec := autoscalingv1.ResourceOverrideSpec{
		PodResourceOverride: autoscalingv1.PodResourceOverrideSpec{
			LimitCPUToMemoryPercent:     100,
			CPURequestToLimitPercent:    50,
			MemoryRequestToLimitPercent: 75,
		},
	}
	t.Log("creating ResourceOverride with different ratios than CRO")
	_, roDisposer := helper.CreateResourceOverride(t, client.Operator, ns.GetName(), "test-ro-fallback", roSpec)

	t.Log("waiting for ResourceOverride validation to pass")
	helper.WaitForResourceOverrideCondition(t, client.Operator, ns.GetName(), "test-ro-fallback", helper.IsResourceOverrideValidationPassing)

	requirements := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("1024Mi"),
			corev1.ResourceCPU:    resource.MustParse("1000m"),
		},
	}

	t.Log("creating pod and verifying RO ratios are applied")
	resourceWantRO := map[string]corev1.ResourceRequirements{
		"test": {
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("1024Mi"),
				corev1.ResourceCPU:    resource.MustParse("1000m"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("768Mi"),
				corev1.ResourceCPU:    resource.MustParse("500m"),
			},
		},
	}

	podGot1, podDisposer1 := helper.NewPodWithResourceRequirement(t, client.Kubernetes, ns.GetName(), "test", requirements)
	defer podDisposer1.Dispose()

	helper.MustMatchMemoryAndCPU(t, resourceWantRO, &podGot1.Spec)

	t.Log("deleting ResourceOverride and verifying fallback to CRO ratios")
	roDisposer.Dispose()

	resourceWantCRO := map[string]corev1.ResourceRequirements{
		"test": {
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("1024Mi"),
				corev1.ResourceCPU:    resource.MustParse("2000m"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("500m"),
			},
		},
	}

	podGot2, podDisposer2 := helper.NewPodWithResourceRequirement(t, client.Kubernetes, ns.GetName(), "test", requirements)
	defer podDisposer2.Dispose()

	helper.MustMatchMemoryAndCPU(t, resourceWantCRO, &podGot2.Spec)
}

func TestResourceOverrideConfigurationChange(t *testing.T) {
	client := helper.NewClient(t, options.config)

	f := &helper.PreCondition{Client: client.Kubernetes}
	f.MustHaveAdmissionRegistrationV1(t)

	croConfig := operatorv1.PodResourceOverrideSpec{
		LimitCPUToMemoryPercent:     200,
		CPURequestToLimitPercent:    25,
		MemoryRequestToLimitPercent: 50,
	}
	override := operatorv1.PodResourceOverride{
		Spec: croConfig,
	}

	t.Logf("setting CRO webhook configuration - %s", croConfig.String())
	current, changed := helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override, nil)
	defer helper.RemoveAdmissionWebhook(t, client.Operator, current.GetName())

	t.Log("waiting for CRO webhook to become available")
	current = helper.Wait(t, client.Operator, "cluster", helper.GetAvailableConditionFunc(current, changed))

	ns, nsDisposer := helper.NewNamespace(t, client.Kubernetes, "roe2e", true)
	defer nsDisposer.Dispose()

	roSpec := autoscalingv1.ResourceOverrideSpec{
		PodResourceOverride: autoscalingv1.PodResourceOverrideSpec{
			LimitCPUToMemoryPercent:     100,
			CPURequestToLimitPercent:    50,
			MemoryRequestToLimitPercent: 50,
		},
	}
	t.Log("creating ResourceOverride with initial ratios")
	ro, roDisposer := helper.CreateResourceOverride(t, client.Operator, ns.GetName(), "test-ro-update", roSpec)
	defer roDisposer.Dispose()

	t.Log("waiting for ResourceOverride validation to pass")
	ro = helper.WaitForResourceOverrideCondition(t, client.Operator, ns.GetName(), "test-ro-update", helper.IsResourceOverrideValidationPassing)

	t.Log("creating pod and verifying initial RO ratios")
	requirements := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("1024Mi"),
			corev1.ResourceCPU:    resource.MustParse("1000m"),
		},
	}

	resourceWantBefore := map[string]corev1.ResourceRequirements{
		"test": {
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("1024Mi"),
				corev1.ResourceCPU:    resource.MustParse("1000m"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("500m"),
			},
		},
	}

	podGot1, podDisposer1 := helper.NewPodWithResourceRequirement(t, client.Kubernetes, ns.GetName(), "test", requirements)
	defer podDisposer1.Dispose()

	helper.MustMatchMemoryAndCPU(t, resourceWantBefore, &podGot1.Spec)

	t.Log("updating ResourceOverride to new ratios")
	ro.Spec.PodResourceOverride.CPURequestToLimitPercent = 75
	ro.Spec.PodResourceOverride.MemoryRequestToLimitPercent = 75
	_, err := client.Operator.AutoscalingV1().ResourceOverrides(ns.GetName()).Update(context.TODO(), ro, metav1.UpdateOptions{})
	require.NoError(t, err)

	t.Log("waiting for updated ResourceOverride validation to pass")
	helper.WaitForResourceOverrideCondition(t, client.Operator, ns.GetName(), "test-ro-update", helper.IsResourceOverrideValidationPassing)

	t.Log("creating pod and verifying updated RO ratios")
	resourceWantAfter := map[string]corev1.ResourceRequirements{
		"test": {
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("1024Mi"),
				corev1.ResourceCPU:    resource.MustParse("1000m"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("768Mi"),
				corev1.ResourceCPU:    resource.MustParse("750m"),
			},
		},
	}

	podGot2, podDisposer2 := helper.NewPodWithResourceRequirement(t, client.Kubernetes, ns.GetName(), "test", requirements)
	defer podDisposer2.Dispose()

	helper.MustMatchMemoryAndCPU(t, resourceWantAfter, &podGot2.Spec)
}

func TestResourceOverrideAdmissionWithCPURequestToRequestPercent(t *testing.T) {
	client := helper.NewClient(t, options.config)

	f := &helper.PreCondition{Client: client.Kubernetes}
	f.MustHaveAdmissionRegistrationV1(t)

	croConfig := operatorv1.PodResourceOverrideSpec{
		MemoryRequestToLimitPercent: 50,
	}
	override := operatorv1.PodResourceOverride{
		Spec: croConfig,
	}

	t.Logf("setting CRO webhook configuration - %s", croConfig.String())
	current, changed := helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override, nil)
	defer helper.RemoveAdmissionWebhook(t, client.Operator, current.GetName())

	t.Log("waiting for CRO webhook to become available")
	current = helper.Wait(t, client.Operator, "cluster", helper.GetAvailableConditionFunc(current, changed))

	ns, nsDisposer := helper.NewNamespace(t, client.Kubernetes, "roe2e", true)
	defer nsDisposer.Dispose()

	roSpec := autoscalingv1.ResourceOverrideSpec{
		PodResourceOverride: autoscalingv1.PodResourceOverrideSpec{
			CPURequestToRequestPercent:  50,
			MemoryRequestToLimitPercent: 50,
		},
	}
	t.Log("creating ResourceOverride with initial ratios")
	_, roDisposer := helper.CreateResourceOverride(t, client.Operator, ns.GetName(), "test-ro-cpurequest", roSpec)
	defer roDisposer.Dispose()

	t.Log("waiting for ResourceOverride validation to pass")
	helper.WaitForResourceOverrideCondition(t, client.Operator, ns.GetName(), "test-ro-cpurequest", helper.IsResourceOverrideValidationPassing)

	t.Log("creating pod and verifying CPU request is scale")
	requirements := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse("200m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
	}

	resourceWant := map[string]corev1.ResourceRequirements{
		"app": {
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("256Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
		},
	}

	podGot, podDisposer := helper.NewPodWithResourceRequirement(t, client.Kubernetes, ns.GetName(), "app", requirements)
	defer podDisposer.Dispose()

	helper.MustMatchMemoryAndCPU(t, resourceWant, &podGot.Spec)
}
