package helper

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"

	"github.com/stretchr/testify/require"

	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/clientset/versioned"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/tlsprofile"
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

func EnsureAdmissionWebhook(t *testing.T, client versioned.Interface, name string, override autoscalingv1.PodResourceOverride, deploymentOverrides *autoscalingv1.DeploymentOverrides) (current *autoscalingv1.ClusterResourceOverride, changed bool) {
	changed = true
	cluster := autoscalingv1.ClusterResourceOverride{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: autoscalingv1.ClusterResourceOverrideSpec{
			PodResourceOverride: override,
		},
	}

	if deploymentOverrides != nil {
		cluster.Spec.DeploymentOverrides = *deploymentOverrides
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
	err := wait.PollUntilContextTimeout(context.TODO(), WaitInterval, WaitTimeout, true, func(ctx context.Context) (done bool, err error) {
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

func GetConfigMap(t *testing.T, client kubernetes.Interface, namespace, name string) (cm *corev1.ConfigMap) {
	cm, err := client.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, cm)
	return
}

func WaitForConfigMap(t *testing.T, client kubernetes.Interface, namespace, name string, original *corev1.ConfigMap) (cm *corev1.ConfigMap) {
	err := wait.PollUntilContextTimeout(context.TODO(), WaitInterval, WaitTimeout, true, func(ctx context.Context) (done bool, err error) {
		cm, err = client.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return false, nil
			}

			return false, err
		}
		if cm == nil || cm.ResourceVersion == original.ResourceVersion {
			return
		}

		done = true
		return
	})
	require.NoErrorf(t, err, "wait.PollUntilContextTimeout returned error - %v", err)
	require.NotNil(t, cm)
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

func GetDeployment(t *testing.T, client kubernetes.Interface, namespace, name string) *appsv1.Deployment {
	deployment, err := client.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, deployment)
	return deployment
}

var apiServerGVR = schema.GroupVersionResource{
	Group:    "config.openshift.io",
	Version:  "v1",
	Resource: "apiservers",
}

// GetAPIServerTLSProfile returns the raw JSON bytes of the current
// spec.tlsSecurityProfile field of the cluster APIServer object, or nil if
// the field is not set.
func GetAPIServerTLSProfile(t *testing.T, config *rest.Config) []byte {
	t.Helper()

	dynClient, err := dynamic.NewForConfig(config)
	require.NoError(t, err)

	apiServer, err := dynClient.Resource(apiServerGVR).Get(context.TODO(), "cluster", metav1.GetOptions{})
	require.NoError(t, err)

	profile, found, err := unstructured.NestedFieldNoCopy(apiServer.Object, "spec", "tlsSecurityProfile")
	require.NoError(t, err)
	if !found || profile == nil {
		return nil
	}

	raw, err := json.Marshal(profile)
	require.NoError(t, err)
	return raw
}

// SetAPIServerTLSProfile patches spec.tlsSecurityProfile on the cluster
// APIServer object to the value encoded in profileJSON. Pass nil to clear the
// field (restoring the cluster default).
func SetAPIServerTLSProfile(t *testing.T, config *rest.Config, profileJSON []byte) {
	t.Helper()

	dynClient, err := dynamic.NewForConfig(config)
	require.NoError(t, err)

	var profileValue interface{}
	if profileJSON != nil {
		require.NoError(t, json.Unmarshal(profileJSON, &profileValue))
	}

	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"tlsSecurityProfile": profileValue,
		},
	}
	patchBytes, err := json.Marshal(patch)
	require.NoError(t, err)

	_, err = dynClient.Resource(apiServerGVR).Patch(
		context.TODO(), "cluster",
		types.MergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	require.NoError(t, err)
}

// WaitForOperandTLSArgs polls the named deployment until the named container's
// args contain exactly the TLS flags implied by want. It fails the test if the
// deployment does not converge within WaitTimeout.
func WaitForOperandTLSArgs(t *testing.T, client kubernetes.Interface, namespace, deploymentName, containerName string, want tlsprofile.Args) {
	t.Helper()

	err := wait.PollUntilContextTimeout(context.TODO(), WaitInterval, WaitTimeout, true, func(ctx context.Context) (bool, error) {
		deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}

		var args []string
		for _, c := range deployment.Spec.Template.Spec.Containers {
			if c.Name == containerName {
				args = c.Args
				break
			}
		}

		return operandTLSArgsMatch(args, want), nil
	})

	require.NoErrorf(t, err,
		"timed out waiting for operand deployment %s/%s container %q to have TLS args (minVersion=%q ciphers=%q)",
		namespace, deploymentName, containerName, want.MinVersion, want.CipherSuites)
}

// operandTLSArgsMatch returns true when args contains exactly the TLS flags
// implied by want: the right --tls-min-version and --tls-cipher-suites values
// when non-empty, and neither flag when empty.
func operandTLSArgsMatch(args []string, want tlsprofile.Args) bool {
	var gotMinVersion, gotCipherSuites string
	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--tls-min-version="):
			gotMinVersion = strings.TrimPrefix(arg, "--tls-min-version=")
		case strings.HasPrefix(arg, "--tls-cipher-suites="):
			gotCipherSuites = strings.TrimPrefix(arg, "--tls-cipher-suites=")
		}
	}
	return gotMinVersion == want.MinVersion && gotCipherSuites == want.CipherSuites
}
