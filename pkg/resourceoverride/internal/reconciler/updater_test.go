package reconciler

import (
	"testing"

	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stesting "k8s.io/client-go/testing"

	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/clientset/versioned/fake"
)

func TestStatusUpdater(t *testing.T) {
	t.Run("identical status makes no API call", func(t *testing.T) {
		status := autoscalingv1.ResourceOverrideStatus{
			Conditions: []autoscalingv1.ResourceOverrideCondition{
				{
					Type:   autoscalingv1.ValidationFailure,
					Status: corev1.ConditionFalse,
				},
			},
		}

		observed := &autoscalingv1.ResourceOverride{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Status:     status,
		}
		desired := &autoscalingv1.ResourceOverride{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Status:     status,
		}

		fakeClient := fake.NewSimpleClientset(observed)
		updater := &StatusUpdater{client: fakeClient}

		err := updater.Update(observed, desired)
		require.NoError(t, err)

		for _, action := range fakeClient.Actions() {
			require.NotEqual(t, "update", action.GetVerb(), "expected no update call for identical status")
		}
	})

	t.Run("different status calls UpdateStatus", func(t *testing.T) {
		observed := &autoscalingv1.ResourceOverride{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Status: autoscalingv1.ResourceOverrideStatus{
				Conditions: []autoscalingv1.ResourceOverrideCondition{
					{
						Type:   autoscalingv1.ValidationFailure,
						Status: corev1.ConditionFalse,
					},
				},
			},
		}
		desired := &autoscalingv1.ResourceOverride{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Status: autoscalingv1.ResourceOverrideStatus{
				Conditions: []autoscalingv1.ResourceOverrideCondition{
					{
						Type:   autoscalingv1.ValidationFailure,
						Status: corev1.ConditionTrue,
						Reason: autoscalingv1.InvalidParameters,
					},
				},
			},
		}

		fakeClient := fake.NewSimpleClientset(observed)
		updater := &StatusUpdater{client: fakeClient}

		err := updater.Update(observed, desired)
		require.NoError(t, err)

		var found bool
		for _, action := range fakeClient.Actions() {
			if action.GetVerb() == "update" {
				updateAction := action.(k8stesting.UpdateAction)
				require.Equal(t, "resourceoverrides", updateAction.GetResource().Resource)
				require.Equal(t, "status", updateAction.GetSubresource())
				found = true
			}
		}
		require.True(t, found, "expected UpdateStatus call for different status")
	})
}
