package controller

import (
	"context"
	"fmt"

	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

type WorkerFunc func(shutdown context.Context, controller Interface)

func (w WorkerFunc) Work(shutdown context.Context, controller Interface) {
	w(shutdown, controller)
}

// Work represents a worker function that pulls item(s) off of the underlying
// work queue and invokes the reconciler function associated with the controller.
func Work(shutdown context.Context, controller Interface) {
	klog.V(1).Infof("[controller] name=%s starting to process work item(s)", controller.Name())

	for processNextWorkItem(shutdown, controller) {
	}

	klog.V(1).Infof("[controller] name=%s shutting down", controller.Name())
}

func processNextWorkItem(shutdownCtx context.Context, controller Interface) bool {
	if shutdownCtx == nil || controller == nil {
		return false
	}

	obj, shutdown := controller.Queue().Get()

	if shutdown {
		return false
	}

	// We call Done here so the workqueue knows we have finished
	// processing this item. We also must remember to call Forget if we
	// do not want this work item being re-queued. For example, we do
	// not call Forget if a transient error occurs, instead the item is
	// put back on the workqueue and attempted again after a back-off
	// period.
	defer controller.Queue().Done(obj)

	request, ok := obj.(reconcile.Request)
	if !ok {
		// As the item in the workqueue is actually invalid, we call
		// Forget here else we'd go into a loop of attempting to
		// process a work item that is invalid.
		controller.Queue().Forget(obj)

		utilruntime.HandleError(fmt.Errorf("expected reconcile.Request in workqueue but got %#v", obj))
		return true
	}

	// Run the syncHandler, passing it the namespace/name string of the
	// Foo resource to be synced.
	result, err := controller.Reconciler().Reconcile(request)
	if err != nil {
		// Put the item back on the workqueue to handle any transient errors.
		controller.Queue().AddRateLimited(request)

		utilruntime.HandleError(fmt.Errorf("error syncing '%s': %s, requeuing", request, err.Error()))
		return true
	}

	if result.RequeueAfter > 0 {
		// The result.RequeueAfter request will be lost, if it is returned
		// along with a non-nil error. But this is intended as
		// We need to drive to stable reconcile loops before queuing due
		// to result.RequestAfter
		controller.Queue().Forget(obj)
		controller.Queue().AddAfter(request, result.RequeueAfter)

		return true
	}

	if result.Requeue {
		controller.Queue().AddRateLimited(request)
		return true
	}

	// Finally, if no error occurs we Forget this item so it does not
	// get queued again until another change happens.
	controller.Queue().Forget(obj)
	return true
}
