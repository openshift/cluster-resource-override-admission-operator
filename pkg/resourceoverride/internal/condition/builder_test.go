package condition

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
)

func TestWithValidationFailure(t *testing.T) {
	status := &autoscalingv1.ResourceOverrideStatus{}
	builder := NewBuilderWithStatus(status)

	builder.WithValidationFailure(autoscalingv1.InvalidParameters, "invalid spec")

	require.Len(t, status.Conditions, 1)
	cond := &status.Conditions[0]
	require.Equal(t, autoscalingv1.ValidationFailure, cond.Type)
	require.Equal(t, corev1.ConditionTrue, cond.Status)
	require.Equal(t, autoscalingv1.InvalidParameters, cond.Reason)
	require.Equal(t, "invalid spec", cond.Message)
	require.False(t, cond.LastTransitionTime.IsZero())
}

func TestWithValidationCleared(t *testing.T) {
	status := &autoscalingv1.ResourceOverrideStatus{}
	builder := NewBuilderWithStatus(status)

	builder.WithValidationCleared()

	require.Len(t, status.Conditions, 1)
	cond := &status.Conditions[0]
	require.Equal(t, autoscalingv1.ValidationFailure, cond.Type)
	require.Equal(t, corev1.ConditionFalse, cond.Status)
	require.Empty(t, cond.Reason)
	require.Empty(t, cond.Message)
}

func TestWithConditionAppends(t *testing.T) {
	status := &autoscalingv1.ResourceOverrideStatus{}
	builder := NewBuilderWithStatus(status)
	builder.init()

	desired := &autoscalingv1.ResourceOverrideCondition{
		Type:   autoscalingv1.ValidationFailure,
		Status: corev1.ConditionTrue,
		Reason: autoscalingv1.InvalidParameters,
	}
	builder.WithCondition(desired)

	require.Len(t, status.Conditions, 1)
	require.Equal(t, autoscalingv1.InvalidParameters, status.Conditions[0].Reason)
}

func TestWithConditionUpdatesInPlace(t *testing.T) {
	status := &autoscalingv1.ResourceOverrideStatus{
		Conditions: []autoscalingv1.ResourceOverrideCondition{
			{
				Type:   autoscalingv1.ValidationFailure,
				Status: corev1.ConditionTrue,
				Reason: autoscalingv1.InvalidParameters,
			},
		},
	}
	builder := NewBuilderWithStatus(status)

	updated := &autoscalingv1.ResourceOverrideCondition{
		Type:   autoscalingv1.ValidationFailure,
		Status: corev1.ConditionFalse,
		Reason: "",
	}
	builder.WithCondition(updated)

	require.Len(t, status.Conditions, 1)
	require.Equal(t, corev1.ConditionFalse, status.Conditions[0].Status)
	require.Empty(t, status.Conditions[0].Reason)
}

func TestWithConditionNoOpWhenEqual(t *testing.T) {
	status := &autoscalingv1.ResourceOverrideStatus{
		Conditions: []autoscalingv1.ResourceOverrideCondition{
			{
				Type:    autoscalingv1.ValidationFailure,
				Status:  corev1.ConditionTrue,
				Reason:  autoscalingv1.InvalidParameters,
				Message: "bad spec",
			},
		},
	}
	builder := NewBuilderWithStatus(status)

	same := &autoscalingv1.ResourceOverrideCondition{
		Type:    autoscalingv1.ValidationFailure,
		Status:  corev1.ConditionTrue,
		Reason:  autoscalingv1.InvalidParameters,
		Message: "bad spec",
	}
	builder.WithCondition(same)

	require.Len(t, status.Conditions, 1)
}

func TestWithConditionNilIsNoOp(t *testing.T) {
	status := &autoscalingv1.ResourceOverrideStatus{}
	builder := NewBuilderWithStatus(status)

	builder.WithCondition(nil)

	require.Empty(t, status.Conditions)
}

func TestFind(t *testing.T) {
	t.Run("returns nil when not found", func(t *testing.T) {
		status := &autoscalingv1.ResourceOverrideStatus{}
		result := Find(status, autoscalingv1.ValidationFailure)
		require.Nil(t, result)
	})

	t.Run("returns correct pointer from multiple conditions", func(t *testing.T) {
		status := &autoscalingv1.ResourceOverrideStatus{
			Conditions: []autoscalingv1.ResourceOverrideCondition{
				{
					Type:   "Other",
					Status: corev1.ConditionFalse,
					Reason: "should-not-match",
				},
				{
					Type:   autoscalingv1.ValidationFailure,
					Status: corev1.ConditionTrue,
					Reason: autoscalingv1.ExemptNamespace,
				},
			},
		}
		result := Find(status, autoscalingv1.ValidationFailure)
		require.NotNil(t, result)
		require.Equal(t, autoscalingv1.ExemptNamespace, result.Reason)
	})
}

func TestEqual(t *testing.T) {
	now := metav1.Now()
	earlier := metav1.NewTime(now.Add(-1 * time.Hour))

	a := &autoscalingv1.ResourceOverrideCondition{
		Type:               autoscalingv1.ValidationFailure,
		Status:             corev1.ConditionTrue,
		Reason:             autoscalingv1.InvalidParameters,
		Message:            "bad",
		LastTransitionTime: now,
	}
	b := &autoscalingv1.ResourceOverrideCondition{
		Type:               autoscalingv1.ValidationFailure,
		Status:             corev1.ConditionTrue,
		Reason:             autoscalingv1.InvalidParameters,
		Message:            "bad",
		LastTransitionTime: earlier,
	}
	require.True(t, Equal(a, b), "should be equal despite different LastTransitionTime")

	b.Message = "different"
	require.False(t, Equal(a, b))
}
