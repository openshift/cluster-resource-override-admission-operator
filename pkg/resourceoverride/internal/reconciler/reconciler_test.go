package reconciler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	controllerreconciler "sigs.k8s.io/controller-runtime/pkg/reconcile"

	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/clientset/versioned/fake"
	autoscalingv1listers "github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/listers/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/resourceoverride/internal/condition"
)

func TestIsExemptNamespace(t *testing.T) {
	tests := []struct {
		namespace string
		want      bool
	}{
		{"default", false},
		{"my-namespace", false},
		{"mykube-system", false},
		{"openshift", true},
		{"openshift-monitoring", true},
		{"openshift-operators", true},
		{"kube", true},
		{"kube-system", true},
		{"kube-public", true},
		{"kubernetes", true},
		{"kubernetes-dashboard", true},
	}

	for _, test := range tests {
		t.Run(test.namespace, func(t *testing.T) {
			got := isExemptNamespace(test.namespace)
			require.Equal(t, test.want, got)
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name       string
		ro         *autoscalingv1.ResourceOverride
		wantStatus corev1.ConditionStatus
		wantReason string
	}{
		{
			name: "valid spec",
			ro: &autoscalingv1.ResourceOverride{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: autoscalingv1.ResourceOverrideSpec{
					PodResourceOverride: autoscalingv1.PodResourceOverrideSpec{
						MemoryRequestToLimitPercent: 50,
						CPURequestToLimitPercent:    25,
					},
				},
			},
			wantStatus: corev1.ConditionFalse,
			wantReason: "",
		},
		{
			name: "valid spec with nil podSelector",
			ro: &autoscalingv1.ResourceOverride{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: autoscalingv1.ResourceOverrideSpec{
					PodResourceOverride: autoscalingv1.PodResourceOverrideSpec{
						MemoryRequestToLimitPercent: 50,
					},
					PodSelector: nil,
				},
			},
			wantStatus: corev1.ConditionFalse,
			wantReason: "",
		},
		{
			name: "exempt namespace",
			ro: &autoscalingv1.ResourceOverride{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "openshift-monitoring",
				},
				Spec: autoscalingv1.ResourceOverrideSpec{
					PodResourceOverride: autoscalingv1.PodResourceOverrideSpec{
						MemoryRequestToLimitPercent: 50,
					},
				},
			},
			wantStatus: corev1.ConditionTrue,
			wantReason: autoscalingv1.ExemptNamespace,
		},
		{
			name: "invalid spec - out of range",
			ro: &autoscalingv1.ResourceOverride{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: autoscalingv1.ResourceOverrideSpec{
					PodResourceOverride: autoscalingv1.PodResourceOverrideSpec{
						MemoryRequestToLimitPercent: 200,
					},
				},
			},
			wantStatus: corev1.ConditionTrue,
			wantReason: autoscalingv1.InvalidParameters,
		},
		{
			name: "invalid podSelector",
			ro: &autoscalingv1.ResourceOverride{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: autoscalingv1.ResourceOverrideSpec{
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
			wantStatus: corev1.ConditionTrue,
			wantReason: autoscalingv1.InvalidParameters,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			Validate(test.ro)

			cond := condition.Find(&test.ro.Status, autoscalingv1.ValidationFailure)
			require.NotNil(t, cond)
			require.Equal(t, test.wantStatus, cond.Status)
			require.Equal(t, test.wantReason, cond.Reason)
		})
	}
}

func TestReconcile(t *testing.T) {
	t.Run("valid RO updates status", func(t *testing.T) {
		ro := &autoscalingv1.ResourceOverride{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-ro",
				Namespace: "default",
			},
			Spec: autoscalingv1.ResourceOverrideSpec{
				PodResourceOverride: autoscalingv1.PodResourceOverrideSpec{
					MemoryRequestToLimitPercent: 50,
				},
			},
		}

		fakeClient := fake.NewSimpleClientset(ro)
		indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
		indexer.Add(ro)
		lister := autoscalingv1listers.NewResourceOverrideLister(indexer)

		r := NewReconciler(fakeClient, lister)
		result, err := r.Reconcile(context.TODO(), controllerreconciler.Request{
			NamespacedName: types.NamespacedName{Namespace: "default", Name: "test-ro"},
		})

		require.NoError(t, err)
		require.Equal(t, controllerreconciler.Result{}, result)

		updated, getErr := fakeClient.AutoscalingV1().ResourceOverrides("default").Get(context.TODO(), "test-ro", metav1.GetOptions{})
		require.NoError(t, getErr)

		cond := condition.Find(&updated.Status, autoscalingv1.ValidationFailure)
		require.NotNil(t, cond)
		require.Equal(t, corev1.ConditionFalse, cond.Status)
	})

	t.Run("not found returns no error", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
		lister := autoscalingv1listers.NewResourceOverrideLister(indexer)

		r := NewReconciler(fakeClient, lister)
		_, err := r.Reconcile(context.TODO(), controllerreconciler.Request{
			NamespacedName: types.NamespacedName{Namespace: "default", Name: "nonexistent"},
		})

		require.NoError(t, err)
	})

	t.Run("invalid RO updates status with failure", func(t *testing.T) {
		ro := &autoscalingv1.ResourceOverride{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-ro-invalid",
				Namespace: "default",
			},
			Spec: autoscalingv1.ResourceOverrideSpec{
				PodResourceOverride: autoscalingv1.PodResourceOverrideSpec{
					MemoryRequestToLimitPercent: 200,
				},
			},
		}

		fakeClient := fake.NewSimpleClientset(ro)
		indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
		indexer.Add(ro)
		lister := autoscalingv1listers.NewResourceOverrideLister(indexer)

		r := NewReconciler(fakeClient, lister)
		result, err := r.Reconcile(context.TODO(), controllerreconciler.Request{
			NamespacedName: types.NamespacedName{Namespace: "default", Name: "test-ro-invalid"},
		})

		require.NoError(t, err)
		require.Equal(t, controllerreconciler.Result{}, result)

		updated, getErr := fakeClient.AutoscalingV1().ResourceOverrides("default").Get(context.TODO(), "test-ro-invalid", metav1.GetOptions{})
		require.NoError(t, getErr)

		cond := condition.Find(&updated.Status, autoscalingv1.ValidationFailure)
		require.NotNil(t, cond)
		require.Equal(t, corev1.ConditionTrue, cond.Status)
		require.Equal(t, autoscalingv1.InvalidParameters, cond.Reason)
	})
}
