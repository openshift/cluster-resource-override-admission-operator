package reconciler

import (
	"testing"

	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	controllerreconciler "sigs.k8s.io/controller-runtime/pkg/reconcile"

	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/asset"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/clientset/versioned/fake"
	autoscalingv1listers "github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/listers/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/resourceoverride/internal/condition"
)

func newNamespaceLister(namespaces ...*corev1.Namespace) corev1listers.NamespaceLister {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	for _, ns := range namespaces {
		indexer.Add(ns)
	}
	return corev1listers.NewNamespaceLister(indexer)
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
	t.Run("valid RO in opted-in namespace", func(t *testing.T) {
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

		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "default",
				Labels: map[string]string{asset.NamespaceOptInLabelKey: "true"},
			},
		}

		fakeClient := fake.NewSimpleClientset(ro)
		indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
		indexer.Add(ro)
		lister := autoscalingv1listers.NewResourceOverrideLister(indexer)
		nsLister := newNamespaceLister(ns)

		r := NewReconciler(fakeClient, lister, nsLister)
		result, err := r.Reconcile(t.Context(), controllerreconciler.Request{
			NamespacedName: types.NamespacedName{Namespace: "default", Name: "test-ro"},
		})

		require.NoError(t, err)
		require.Equal(t, controllerreconciler.Result{}, result)

		updated, getErr := fakeClient.AutoscalingV1().ResourceOverrides("default").Get(t.Context(), "test-ro", metav1.GetOptions{})
		require.NoError(t, getErr)

		validationCond := condition.Find(&updated.Status, autoscalingv1.ValidationFailure)
		require.NotNil(t, validationCond)
		require.Equal(t, corev1.ConditionFalse, validationCond.Status)

		ignoredCond := condition.Find(&updated.Status, autoscalingv1.Ignored)
		require.NotNil(t, ignoredCond)
		require.Equal(t, corev1.ConditionFalse, ignoredCond.Status)
	})

	t.Run("valid RO in non-opted-in namespace", func(t *testing.T) {
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

		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
			},
		}

		fakeClient := fake.NewSimpleClientset(ro)
		indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
		indexer.Add(ro)
		lister := autoscalingv1listers.NewResourceOverrideLister(indexer)
		nsLister := newNamespaceLister(ns)

		r := NewReconciler(fakeClient, lister, nsLister)
		result, err := r.Reconcile(t.Context(), controllerreconciler.Request{
			NamespacedName: types.NamespacedName{Namespace: "default", Name: "test-ro"},
		})

		require.NoError(t, err)
		require.Equal(t, controllerreconciler.Result{}, result)

		updated, getErr := fakeClient.AutoscalingV1().ResourceOverrides("default").Get(t.Context(), "test-ro", metav1.GetOptions{})
		require.NoError(t, getErr)

		ignoredCond := condition.Find(&updated.Status, autoscalingv1.Ignored)
		require.NotNil(t, ignoredCond)
		require.Equal(t, corev1.ConditionTrue, ignoredCond.Status)
		require.Equal(t, autoscalingv1.NamespaceNotOptedIn, ignoredCond.Reason)
	})

	t.Run("not found returns no error", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
		lister := autoscalingv1listers.NewResourceOverrideLister(indexer)
		nsLister := newNamespaceLister()

		r := NewReconciler(fakeClient, lister, nsLister)
		_, err := r.Reconcile(t.Context(), controllerreconciler.Request{
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

		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "default",
				Labels: map[string]string{asset.NamespaceOptInLabelKey: "true"},
			},
		}

		fakeClient := fake.NewSimpleClientset(ro)
		indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
		indexer.Add(ro)
		lister := autoscalingv1listers.NewResourceOverrideLister(indexer)
		nsLister := newNamespaceLister(ns)

		r := NewReconciler(fakeClient, lister, nsLister)
		result, err := r.Reconcile(t.Context(), controllerreconciler.Request{
			NamespacedName: types.NamespacedName{Namespace: "default", Name: "test-ro-invalid"},
		})

		require.NoError(t, err)
		require.Equal(t, controllerreconciler.Result{}, result)

		updated, getErr := fakeClient.AutoscalingV1().ResourceOverrides("default").Get(t.Context(), "test-ro-invalid", metav1.GetOptions{})
		require.NoError(t, getErr)

		cond := condition.Find(&updated.Status, autoscalingv1.ValidationFailure)
		require.NotNil(t, cond)
		require.Equal(t, corev1.ConditionTrue, cond.Status)
		require.Equal(t, autoscalingv1.InvalidParameters, cond.Reason)
	})
}
