package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/test/helper"
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
							AllowPrivilegeEscalation: pointer.BoolPtr(false),
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{"ALL"},
							},
							RunAsNonRoot: pointer.BoolPtr(true),
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
							AllowPrivilegeEscalation: pointer.BoolPtr(false),
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{"ALL"},
							},
							RunAsNonRoot: pointer.BoolPtr(true),
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
							AllowPrivilegeEscalation: pointer.BoolPtr(false),
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{"ALL"},
							},
							RunAsNonRoot: pointer.BoolPtr(true),
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
							AllowPrivilegeEscalation: pointer.BoolPtr(false),
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{"ALL"},
							},
							RunAsNonRoot: pointer.BoolPtr(true),
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
							AllowPrivilegeEscalation: pointer.BoolPtr(false),
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{"ALL"},
							},
							RunAsNonRoot: pointer.BoolPtr(true),
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
							AllowPrivilegeEscalation: pointer.BoolPtr(false),
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{"ALL"},
							},
							RunAsNonRoot: pointer.BoolPtr(true),
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
	configuration := autoscalingv1.PodResourceOverrideSpec{
		LimitCPUToMemoryPercent:     200,
		CPURequestToLimitPercent:    25,
		MemoryRequestToLimitPercent: 50,
	}
	override := autoscalingv1.PodResourceOverride{
		Spec: configuration,
	}

	t.Logf("setting webhook configuration - %s", configuration.String())
	current, changed := helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override)
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

	before := autoscalingv1.PodResourceOverrideSpec{
		LimitCPUToMemoryPercent:     100,
		CPURequestToLimitPercent:    10,
		MemoryRequestToLimitPercent: 75,
	}
	override := autoscalingv1.PodResourceOverride{
		Spec: before,
	}

	t.Logf("initial configuration - %s", before.String())

	current, changed := helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override)
	defer helper.RemoveAdmissionWebhook(t, client.Operator, current.GetName())

	current = helper.Wait(t, client.Operator, "cluster", helper.GetAvailableConditionFunc(current, changed))
	require.Equal(t, override.Spec.Hash(), current.Status.Hash.Configuration)

	after := autoscalingv1.PodResourceOverrideSpec{
		LimitCPUToMemoryPercent:     50,
		CPURequestToLimitPercent:    50,
		MemoryRequestToLimitPercent: 50,
	}
	override = autoscalingv1.PodResourceOverride{
		Spec: after,
	}

	t.Logf("final configuration - %s", after.String())

	current, changed = helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override)
	current = helper.Wait(t, client.Operator, "cluster", helper.GetAvailableConditionFunc(current, changed))
	require.Equal(t, override.Spec.Hash(), current.Status.Hash.Configuration)

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
}

func TestClusterResourceOverrideAdmissionWithCertRotation(t *testing.T) {
	client := helper.NewClient(t, options.config)

	f := &helper.PreCondition{Client: client.Kubernetes}
	f.MustHaveAdmissionRegistrationV1(t)

	configuration := autoscalingv1.PodResourceOverrideSpec{
		LimitCPUToMemoryPercent:     50,
		CPURequestToLimitPercent:    50,
		MemoryRequestToLimitPercent: 50,
	}
	override := autoscalingv1.PodResourceOverride{
		Spec: configuration,
	}

	current, changed := helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override)
	defer helper.RemoveAdmissionWebhook(t, client.Operator, current.GetName())

	current = helper.Wait(t, client.Operator, "cluster", helper.GetAvailableConditionFunc(current, changed))

	originalCertHash := current.Status.Hash.ServingCert
	require.NotEmpty(t, originalCertHash)

	current.Status.CertsRotateAt = metav1.NewTime(time.Now())
	current, err := client.Operator.AutoscalingV1().ClusterResourceOverrides().UpdateStatus(context.TODO(), current, metav1.UpdateOptions{})
	require.NoError(t, err)

	current = helper.Wait(t, client.Operator, "cluster", helper.GetAvailableConditionFunc(current, true))
	newCertHash := current.Status.Hash.ServingCert
	require.NotEmpty(t, originalCertHash)
	require.NotEqual(t, originalCertHash, newCertHash)

	// make sure everything works after cert is regenerated
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
}

func TestClusterResourceOverrideAdmissionWithNoOptIn(t *testing.T) {
	client := helper.NewClient(t, options.config)

	f := &helper.PreCondition{Client: client.Kubernetes}
	f.MustHaveAdmissionRegistrationV1(t)

	configuration := autoscalingv1.PodResourceOverrideSpec{
		LimitCPUToMemoryPercent:     200,
		CPURequestToLimitPercent:    50,
		MemoryRequestToLimitPercent: 50,
	}
	override := autoscalingv1.PodResourceOverride{
		Spec: configuration,
	}

	current, changed := helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override)
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
