package controller

import (
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// NewEventHandler returns a cache.ResourceEventHandler appropriate for
// reconciliation of ClusterResourceOverride object(s).
func NewEventHandler(queue workqueue.RateLimitingInterface) EventHandler {
	return EventHandler{
		queue: queue,
	}
}

var _ cache.ResourceEventHandler = EventHandler{}

type EventHandler struct {
	// The underlying work queue where the keys are added for reconciliation.
	queue workqueue.RateLimitingInterface
}

func (e EventHandler) OnAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)

	if err != nil {
		klog.Errorf("OnAdd: could not extract key, type=%T", obj)
		return
	}

	e.add(key, e.queue)
}

// OnUpdate creates UpdateEvent and calls Update on EventHandler
func (e EventHandler) OnUpdate(oldObj, newObj interface{}) {
	// We don't distinguish between an add and update.
	e.OnAdd(newObj)
}

func (e EventHandler) OnDelete(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}

	e.add(key, e.queue)
}

func (e EventHandler) add(key string, queue workqueue.RateLimitingInterface) {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return
	}

	request := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
	}

	queue.Add(request)
}
