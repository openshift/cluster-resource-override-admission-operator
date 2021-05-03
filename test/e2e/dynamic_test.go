package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/cluster-resource-override-admission-operator/pkg/dynamic"
	"github.com/openshift/cluster-resource-override-admission-operator/test/helper"
)

func TestDynamicClient(t *testing.T) {
	client := helper.NewClient(t, options.config)

	namespace := options.namespace
	name := "test-dynamic-client"
	var replicas int32 = 1
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Annotations: map[string]string{
				"annotation1": "value1",
			},
			Labels: map[string]string{
				"label1": "value1",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"selector1": "true",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
					Annotations: map[string]string{
						"annotation1": "value1",
					},
					Labels: map[string]string{
						"selector1": "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            name,
							Image:           "openshift/hello-openshift",
							ImagePullPolicy: corev1.PullAlways,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		},
	}

	dynamic, err := dynamic.NewForConfig(options.config)
	require.NoError(t, err)
	require.NotNil(t, dynamic)

	// ensure that this deployment does not exist.
	_, err = client.Kubernetes.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	require.Truef(t, k8serrors.IsNotFound(err), "precondition failed - Deployment=%s/%s already exists", namespace, name)

	// scenario 1: does not exist, create only.
	objectGot, errGot := dynamic.Ensure("deployments", deployment)
	require.NoError(t, errGot)
	require.NotNil(t, objectGot)

	defer func() {
		err := client.Kubernetes.AppsV1().Deployments(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
		require.NoError(t, err)
	}()

	// update/patch
	deployment.Annotations["annotation2"] = "value2"
	deployment.Labels["label2"] = "value2"
	deployment.Spec.Template.Labels["label2"] = "value2"
	deployment.Spec.Template.Annotations["annotation2"] = "value2"

	objectGot, errGot = dynamic.Ensure("deployments", deployment)
	require.NoError(t, errGot)
	require.NotNil(t, objectGot)

	current, err := client.Kubernetes.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	require.NoError(t, err)
	require.True(t, current.Annotations["annotation1"] == "value1", "original annotation not preserved")
	require.True(t, current.Annotations["annotation2"] == "value2", "new annotation not applied")
	require.True(t, current.Spec.Template.Annotations["annotation1"] == "value1", "original annotation in Spec.Template not preserved")
	require.True(t, current.Spec.Template.Annotations["annotation2"] == "value2", "new annotation in Spec.Template not preserved")

	require.True(t, current.Labels["label1"] == "value1", "original label not preserved")
	require.True(t, current.Labels["label2"] == "value2", "new label not applied")
	require.True(t, current.Spec.Template.Labels["selector1"] == "true", "original label in Spec.Template not preserved")
	require.True(t, current.Spec.Template.Labels["label2"] == "value2", "new label in Spec.Template not applied")
}
