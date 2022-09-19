package helper

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"

	"github.com/stretchr/testify/require"

	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/clientset/versioned"
)

var (
	WaitInterval time.Duration = 1 * time.Second
	WaitTimeout  time.Duration = 5 * time.Minute
)

type Disposer func()

func (d Disposer) Dispose() {
	d()
}

type ConditionFunc func(override *autoscalingv1.ClusterResourceOverride) bool

type Client struct {
	Operator   versioned.Interface
	Kubernetes kubernetes.Interface
}

func NewClient(t *testing.T, config *rest.Config) *Client {
	operator, err := versioned.NewForConfig(config)
	require.NoErrorf(t, err, "failed to construct client for autoscaling.openshift.io - %v", err)

	kubeclient, err := kubernetes.NewForConfig(config)
	require.NoErrorf(t, err, "failed to construct client for kubernetes - %v", err)

	return &Client{
		Operator:   operator,
		Kubernetes: kubeclient,
	}
}

func EnsureAdmissionWebhook(t *testing.T, client versioned.Interface, name string, override autoscalingv1.PodResourceOverride) (current *autoscalingv1.ClusterResourceOverride, changed bool) {
	changed = true
	cluster := autoscalingv1.ClusterResourceOverride{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: autoscalingv1.ClusterResourceOverrideSpec{
			PodResourceOverride: override,
		},
	}

	var err error
	current, err = client.AutoscalingV1().ClusterResourceOverrides().Create(context.TODO(), &cluster, metav1.CreateOptions{})
	if err == nil {
		return
	}

	if !k8serrors.IsAlreadyExists(err) {
		require.FailNowf(t, "unexpected error - %s", err.Error())
	}

	current, err = client.AutoscalingV1().ClusterResourceOverrides().Get(context.TODO(), "cluster", metav1.GetOptions{})
	require.NoErrorf(t, err, "failed to get - %v", err)
	require.NotNil(t, current)

	// if the desired spec matches current spec then no change.
	if reflect.DeepEqual(current.Spec, cluster.Spec) {
		changed = false
		return
	}

	current.Spec.PodResourceOverride = *override.DeepCopy()
	current, err = client.AutoscalingV1().ClusterResourceOverrides().Update(context.TODO(), current, metav1.UpdateOptions{})
	require.NoErrorf(t, err, "failed to update - %v", err)
	require.NotNil(t, current)
	return
}

func RemoveAdmissionWebhook(t *testing.T, client versioned.Interface, name string) {
	_, err := client.AutoscalingV1().ClusterResourceOverrides().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			require.FailNowf(t, "unexpected error - %s", err.Error())
		}

		return
	}

	err = client.AutoscalingV1().ClusterResourceOverrides().Delete(context.TODO(), name, metav1.DeleteOptions{})
	require.NoError(t, err)
}

func NewNamespace(t *testing.T, client kubernetes.Interface, name string, optIn bool) (ns *corev1.Namespace, disposer Disposer) {
	request := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", name),
		},
	}

	if optIn {
		request.ObjectMeta.Labels = map[string]string{
			"clusterresourceoverrides.admission.autoscaling.openshift.io/enabled": "true",
		}
	}

	object, err := client.CoreV1().Namespaces().Create(context.TODO(), request, metav1.CreateOptions{})
	require.NoError(t, err)
	require.NotNil(t, object)

	ns = object
	disposer = func() {
		err := client.CoreV1().Namespaces().Delete(context.TODO(), object.Name, metav1.DeleteOptions{})
		require.NoError(t, err)
	}
	return
}

func NewPod(t *testing.T, client kubernetes.Interface, namespace string, spec corev1.PodSpec) (pod *corev1.Pod, disposer Disposer) {
	request := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "croe2e-",
		},
		Spec: spec,
	}

	object, err := client.CoreV1().Pods(namespace).Create(context.TODO(), request, metav1.CreateOptions{})
	require.NoError(t, err)
	require.NotNil(t, object)

	pod = object
	disposer = func() {
		err := client.CoreV1().Pods(object.Namespace).Delete(context.TODO(), object.Name, metav1.DeleteOptions{})
		require.NoError(t, err)
	}
	return
}

func NewPodWithResourceRequirement(t *testing.T, client kubernetes.Interface, namespace string, containerName string, requirements corev1.ResourceRequirements) (pod *corev1.Pod, disposer Disposer) {
	request := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "croe2e-",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  containerName,
					Image: "openshift/hello-openshift",
					Ports: []corev1.ContainerPort{
						{
							Name:          "app",
							ContainerPort: 8080,
						},
					},
					Resources: requirements,
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
	}

	object, err := client.CoreV1().Pods(namespace).Create(context.TODO(), request, metav1.CreateOptions{})
	require.NoError(t, err)
	require.NotNil(t, object)

	pod = object
	disposer = func() {
		err := client.CoreV1().Pods(object.Namespace).Delete(context.TODO(), object.Name, metav1.DeleteOptions{})
		require.NoError(t, err)
	}
	return
}

func GetClusterResourceOverride(t *testing.T, client versioned.Interface, name string) *autoscalingv1.ClusterResourceOverride {
	current, err := client.AutoscalingV1().ClusterResourceOverrides().Get(context.TODO(), name, metav1.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, current)

	return current
}

func Wait(t *testing.T, client versioned.Interface, name string, f ConditionFunc) (override *autoscalingv1.ClusterResourceOverride) {
	err := wait.Poll(WaitInterval, WaitTimeout, func() (done bool, err error) {
		override, err = client.AutoscalingV1().ClusterResourceOverrides().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return
		}

		if override == nil || !f(override) {
			return
		}

		done = true
		return
	})

	require.NoErrorf(t, err, "wait.Poll returned error - %v", err)
	require.NotNil(t, override)
	return
}

func GetAvailableConditionFunc(original *autoscalingv1.ClusterResourceOverride, expectNewResourceVersion bool) ConditionFunc {
	return func(current *autoscalingv1.ClusterResourceOverride) bool {
		switch {
		// we expect current to have a different resource version than original
		case expectNewResourceVersion:
			return original.ResourceVersion != current.ResourceVersion && IsAvailable(current)
		default:
			return IsAvailable(current)
		}
	}
}

func GetCondition(override *autoscalingv1.ClusterResourceOverride, condType autoscalingv1.ClusterResourceOverrideConditionType) *autoscalingv1.ClusterResourceOverrideCondition {
	for i := range override.Status.Conditions {
		condition := &override.Status.Conditions[i]
		if condition.Type == condType {
			return condition
		}
	}

	return nil
}

func IsAvailable(override *autoscalingv1.ClusterResourceOverride) bool {
	available := GetCondition(override, autoscalingv1.Available)
	readinessFailure := GetCondition(override, autoscalingv1.InstallReadinessFailure)
	if available == nil || readinessFailure == nil {
		return false
	}

	if available.Status != corev1.ConditionTrue || readinessFailure.Status != corev1.ConditionFalse {
		return false
	}

	return true
}

func IsMatch(t *testing.T, requirementsWant, requirementsGot corev1.ResourceRequirements) {
	quantityWant, ok := requirementsWant.Requests[corev1.ResourceMemory]
	if ok {
		quantityGot, ok := requirementsGot.Requests[corev1.ResourceMemory]
		require.Truef(t, ok, "key=%s not found in %v", corev1.ResourceMemory, requirementsGot)
		require.Truef(t, quantityWant.Equal(quantityGot), "type=request resource=%s expected=%v actual=%v",
			corev1.ResourceMemory, quantityWant, quantityGot)
	}

	quantityWant, ok = requirementsWant.Requests[corev1.ResourceCPU]
	if ok {
		quantityGot, ok := requirementsGot.Requests[corev1.ResourceCPU]
		require.True(t, ok)
		require.True(t, quantityWant.Equal(quantityGot))
	}

	quantityWant, ok = requirementsWant.Limits[corev1.ResourceMemory]
	if ok {
		quantityGot, ok := requirementsGot.Limits[corev1.ResourceMemory]
		require.True(t, ok)
		require.True(t, quantityWant.Equal(quantityGot))
	}

	quantityWant, ok = requirementsWant.Limits[corev1.ResourceCPU]
	if ok {
		quantityGot, ok := requirementsGot.Limits[corev1.ResourceCPU]
		require.True(t, ok)
		require.True(t, quantityWant.Equal(quantityGot))
	}
}

func GetContainer(t *testing.T, name string, spec *corev1.PodSpec) corev1.ResourceRequirements {
	for i, container := range spec.InitContainers {
		if container.Name == name {
			return spec.InitContainers[i].Resources
		}
	}

	for i, container := range spec.Containers {
		if container.Name == name {
			return spec.Containers[i].Resources
		}
	}

	require.FailNowf(t, "failed to find container  in Pod spec - %s", name)
	return corev1.ResourceRequirements{}
}

func MustMatchMemoryAndCPU(t *testing.T, resourceWant map[string]corev1.ResourceRequirements, specGot *corev1.PodSpec) {
	for name, want := range resourceWant {
		got := GetContainer(t, name, specGot)
		IsMatch(t, want, got)
	}
}

func NewLimitRanges(t *testing.T, client kubernetes.Interface, namespace string, spec corev1.LimitRangeSpec) (object *corev1.LimitRange, disposer Disposer) {
	request := corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "cro-limits-",
			Namespace:    namespace,
		},
		Spec: spec,
	}

	object, err := client.CoreV1().LimitRanges(namespace).Create(context.TODO(), &request, metav1.CreateOptions{})
	require.NoError(t, err)
	require.NotNil(t, object)

	disposer = func() {
		err := client.CoreV1().LimitRanges(object.Namespace).Delete(context.TODO(), object.Name, metav1.DeleteOptions{})
		require.NoError(t, err)
	}
	return
}
