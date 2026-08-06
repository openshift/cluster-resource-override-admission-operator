package resourceoverride

import (
	"context"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	controllerreconciler "sigs.k8s.io/controller-runtime/pkg/reconcile"

	listers "github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/listers/autoscaling/v1"
)

// NamespaceWatchStarterFunc starts the namespace informer and waits for cache sync.
type NamespaceWatchStarterFunc func(ctx context.Context) error

type namespaceEventHandler struct {
	roLister listers.ResourceOverrideLister
	queue    workqueue.RateLimitingInterface
}

func (h *namespaceEventHandler) OnAdd(obj interface{}, isInInitialList bool) {}

func (h *namespaceEventHandler) OnUpdate(oldObj, newObj interface{}) {
	oldNs, ok := oldObj.(*corev1.Namespace)
	if !ok {
		return
	}
	newNs, ok := newObj.(*corev1.Namespace)
	if !ok {
		return
	}

	if reflect.DeepEqual(oldNs.Labels, newNs.Labels) {
		return
	}

	ros, err := h.roLister.ResourceOverrides(newNs.Name).List(labels.Everything())
	if err != nil || len(ros) == 0 {
		return
	}

	for _, ro := range ros {
		h.queue.Add(controllerreconciler.Request{
			NamespacedName: types.NamespacedName{
				Namespace: ro.Namespace,
				Name:      ro.Name,
			},
		})
	}

	klog.V(4).Infof("[resourceoverride] namespace=%s labels changed, enqueued %d ResourceOverride(s)", newNs.Name, len(ros))
}

func (h *namespaceEventHandler) OnDelete(obj interface{}) {}
