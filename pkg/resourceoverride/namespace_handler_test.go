package resourceoverride

import (
	"testing"

	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	controllerreconciler "sigs.k8s.io/controller-runtime/pkg/reconcile"

	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	listers "github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/listers/autoscaling/v1"
)

func newTestROLister(ros ...*autoscalingv1.ResourceOverride) listers.ResourceOverrideLister {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	for _, ro := range ros {
		indexer.Add(ro)
	}
	return listers.NewResourceOverrideLister(indexer)
}

func TestNamespaceEventHandlerOnUpdate(t *testing.T) {
	tests := []struct {
		name         string
		oldNs        *corev1.Namespace
		newNs        *corev1.Namespace
		ros          []*autoscalingv1.ResourceOverride
		wantEnqueued int
	}{
		{
			name: "labels changed with ROs in namespace",
			oldNs: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: "test-ns"},
			},
			newNs: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-ns",
					Labels: map[string]string{"some-label": "true"},
				},
			},
			ros: []*autoscalingv1.ResourceOverride{
				{ObjectMeta: metav1.ObjectMeta{Name: "ro-1", Namespace: "test-ns"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "ro-2", Namespace: "test-ns"}},
			},
			wantEnqueued: 2,
		},
		{
			name: "labels unchanged",
			oldNs: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-ns",
					Labels: map[string]string{"key": "value"},
				},
			},
			newNs: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-ns",
					Labels: map[string]string{"key": "value"},
				},
			},
			ros: []*autoscalingv1.ResourceOverride{
				{ObjectMeta: metav1.ObjectMeta{Name: "ro-1", Namespace: "test-ns"}},
			},
			wantEnqueued: 0,
		},
		{
			name: "labels changed but no ROs in namespace",
			oldNs: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: "empty-ns"},
			},
			newNs: &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "empty-ns",
					Labels: map[string]string{"added": "true"},
				},
			},
			ros:          nil,
			wantEnqueued: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
			defer queue.ShutDown()

			handler := &namespaceEventHandler{
				roLister: newTestROLister(test.ros...),
				queue:    queue,
			}

			handler.OnUpdate(test.oldNs, test.newNs)

			require.Equal(t, test.wantEnqueued, queue.Len())

			for _, ro := range test.ros[:test.wantEnqueued] {
				item, _ := queue.Get()
				req := item.(controllerreconciler.Request)
				require.Equal(t, types.NamespacedName{Namespace: ro.Namespace, Name: ro.Name}, req.NamespacedName)
				queue.Done(item)
			}
		})
	}
}
