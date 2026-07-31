package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/test/helper"
)

// This file tests the operator alone

func TestResourceOverrideValidSpec(t *testing.T) {
	tests := []struct {
		name       string
		namespace  string
		spec       autoscalingv1.ResourceOverrideSpec
		wantStatus corev1.ConditionStatus
		wantReason string
	}{
		{
			name:      "test-resourceoverride-valid-spec",
			namespace: "test-namespace",
			spec: autoscalingv1.ResourceOverrideSpec{
				PodResourceOverride: autoscalingv1.PodResourceOverrideSpec{
					MemoryRequestToLimitPercent: 50,
					CPURequestToRequestPercent:  50,
				},
			},
			wantStatus: corev1.ConditionFalse,
			wantReason: "",
		},
		{
			name:      "test-resourceoverride-valid-spec-with-podselector",
			namespace: "test-namespace",
			spec: autoscalingv1.ResourceOverrideSpec{
				PodResourceOverride: autoscalingv1.PodResourceOverrideSpec{
					MemoryRequestToLimitPercent: 50,
					CPURequestToRequestPercent:  50,
				},
				PodSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "test",
					},
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "environment",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{"production", "staging"},
						},
					},
				},
			},
			wantStatus: corev1.ConditionFalse,
			wantReason: "",
		},
		{
			name:      "test-resourceoverride-exempt-namespace",
			namespace: "openshift-monitoring",
			spec: autoscalingv1.ResourceOverrideSpec{
				PodResourceOverride: autoscalingv1.PodResourceOverrideSpec{
					MemoryRequestToLimitPercent: 50,
					CPURequestToRequestPercent:  50,
				},
			},
			wantStatus: corev1.ConditionTrue,
			wantReason: autoscalingv1.ExemptNamespace,
		},
	}

	client := helper.NewClient(t, options.config)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ns, nsDisposer := helper.NewNamespace(t, client.Kubernetes, test.namespace, false)
			defer nsDisposer.Dispose()

			_, roDisposer := helper.CreateResourceOverride(t, client.Operator, ns.Name, test.name, test.spec)
			defer roDisposer.Dispose()

			ro := helper.WaitForResourceOverrideCondition(t, client.Operator, ns.Name, test.name, func(override *autoscalingv1.ResourceOverride) bool {
				condition := helper.GetResourceOverrideCondition(override, autoscalingv1.ValidationFailure)
				return condition != nil
			})

			condition := helper.GetResourceOverrideCondition(ro, autoscalingv1.ValidationFailure)
			require.Equal(t, test.wantStatus, condition.Status)
			if test.wantReason != "" {
				require.Equal(t, test.wantReason, condition.Reason)
			}
		})
	}
}

func TestResourceOverrideInvalidCRDSpec(t *testing.T) {
	tests := []struct {
		name string
		spec autoscalingv1.ResourceOverrideSpec
	}{
		{
			name: "test-resourceoverride-invalid-override-values",
			spec: autoscalingv1.ResourceOverrideSpec{
				PodResourceOverride: autoscalingv1.PodResourceOverrideSpec{
					MemoryRequestToLimitPercent: 200,
					CPURequestToLimitPercent:    -1,
				},
			},
		},
		{
			name: "test-resourceoverride-invalid-podselector-operator",
			spec: autoscalingv1.ResourceOverrideSpec{
				PodResourceOverride: autoscalingv1.PodResourceOverrideSpec{
					MemoryRequestToLimitPercent: 50,
				},
				PodSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "app",
							Operator: metav1.LabelSelectorOperator("InvalidOperator"),
						},
					},
				},
			},
		},
	}

	client := helper.NewClient(t, options.config)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ns, nsDisposer := helper.NewNamespace(t, client.Kubernetes, "test-namespace", false)
			defer nsDisposer.Dispose()

			request := &autoscalingv1.ResourceOverride{
				ObjectMeta: metav1.ObjectMeta{
					Name:      test.name,
					Namespace: ns.Name,
				},
				Spec: test.spec,
			}

			_, err := client.Operator.AutoscalingV1().ResourceOverrides(ns.Name).Create(t.Context(), request, metav1.CreateOptions{})
			require.True(t, k8serrors.IsInvalid(err), "expected CRD validation to reject invalid spec, got: %v", err)
		})
	}
}

func TestResourceOverrideInvalidReconcilerSpec(t *testing.T) {
	tests := []struct {
		name string
		spec autoscalingv1.ResourceOverrideSpec
	}{
		{
			name: "test-resourceoverride-in-empty-values",
			spec: autoscalingv1.ResourceOverrideSpec{
				PodResourceOverride: autoscalingv1.PodResourceOverrideSpec{
					MemoryRequestToLimitPercent: 50,
				},
				PodSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "app",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{},
						},
					},
				},
			},
		},
		{
			name: "test-resourceoverride-exists-with-values",
			spec: autoscalingv1.ResourceOverrideSpec{
				PodResourceOverride: autoscalingv1.PodResourceOverrideSpec{
					MemoryRequestToLimitPercent: 50,
				},
				PodSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "app",
							Operator: metav1.LabelSelectorOpExists,
							Values:   []string{"foo"},
						},
					},
				},
			},
		},
		{
			name: "test-resourceoverride-invalid-matchlabels-key",
			spec: autoscalingv1.ResourceOverrideSpec{
				PodResourceOverride: autoscalingv1.PodResourceOverrideSpec{
					MemoryRequestToLimitPercent: 50,
				},
				PodSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"invalid key": "value",
					},
				},
			},
		},
	}

	client := helper.NewClient(t, options.config)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ns, nsDisposer := helper.NewNamespace(t, client.Kubernetes, "test-namespace", false)
			defer nsDisposer.Dispose()

			_, roDisposer := helper.CreateResourceOverride(t, client.Operator, ns.Name, test.name, test.spec)
			defer roDisposer.Dispose()

			ro := helper.WaitForResourceOverrideCondition(t, client.Operator, ns.Name, test.name, func(override *autoscalingv1.ResourceOverride) bool {
				condition := helper.GetResourceOverrideCondition(override, autoscalingv1.ValidationFailure)
				return condition != nil
			})

			condition := helper.GetResourceOverrideCondition(ro, autoscalingv1.ValidationFailure)
			require.Equal(t, corev1.ConditionTrue, condition.Status)
			require.Equal(t, autoscalingv1.InvalidParameters, condition.Reason)
		})
	}
}
