package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	operatorv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/operator/v1"
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

func TestResourceOverrideExemptNamespace(t *testing.T) {
	exemptNamespaces := []string{
		"openshift-monitoring",
		"openshift",
		"kube-system",
		"kube",
		"kubernetes-dashboard",
		"kubernetes",
	}

	client := helper.NewClient(t, options.config)

	override := operatorv1.PodResourceOverride{
		Spec: operatorv1.PodResourceOverrideSpec{
			LimitCPUToMemoryPercent:     200,
			CPURequestToLimitPercent:    25,
			MemoryRequestToLimitPercent: 50,
		},
	}
	current, changed := helper.EnsureAdmissionWebhook(t, client.Operator, "cluster", override, nil)
	defer helper.RemoveAdmissionWebhook(t, client.Operator, current.GetName())
	helper.Wait(t, client.Operator, "cluster", helper.GetAvailableConditionFunc(current, changed))

	f := &helper.PreCondition{Client: client.Kubernetes}
	f.MustHaveValidatingAdmissionPolicy(t)

	assertRejection := func(t *testing.T, namespace string) {
		t.Helper()
		err := wait.PollUntilContextTimeout(t.Context(), 10*time.Second, helper.WaitTimeout, true, func(ctx context.Context) (bool, error) {
			request := &autoscalingv1.ResourceOverride{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-resourceoverride-exempt-namespace",
					Namespace: namespace,
				},
				Spec: autoscalingv1.ResourceOverrideSpec{
					PodResourceOverride: autoscalingv1.PodResourceOverrideSpec{
						MemoryRequestToLimitPercent: 50,
						CPURequestToRequestPercent:  50,
					},
				},
			}

			_, createErr := client.Operator.AutoscalingV1().ResourceOverrides(namespace).Create(ctx, request, metav1.CreateOptions{})
			if createErr == nil {
				// Created successfully when it should have been rejected; clean up and retry
				_ = client.Operator.AutoscalingV1().ResourceOverrides(namespace).Delete(ctx, request.Name, metav1.DeleteOptions{})
				t.Logf("VAP not yet enforcing for namespace %s, retrying", namespace)
				return false, nil
			}
			if k8serrors.IsForbidden(createErr) || k8serrors.IsInvalid(createErr) {
				return true, nil
			}
			return false, createErr
		})
		require.NoError(t, err, "timed out waiting for VAP to reject ResourceOverride in exempt namespace %s", namespace)
	}

	for _, namespace := range exemptNamespaces {
		t.Run(namespace, func(t *testing.T) {
			ns, nsDisposer := helper.NewExactNamespace(t, client.Kubernetes, namespace)
			defer nsDisposer.Dispose()
			assertRejection(t, ns.Name)
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
