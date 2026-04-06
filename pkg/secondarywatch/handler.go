package secondarywatch

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"github.com/openshift/cluster-resource-override-admission-operator/pkg/runtime"
)

func newResourceEventHandler(enqueuer runtime.Enqueuer) *resourceEventHandler {
	return &resourceEventHandler{
		enqueuer: enqueuer,
	}
}

var _ cache.ResourceEventHandler = &resourceEventHandler{}

type resourceEventHandler struct {
	// When a secondary resource is updated we enqueue the owner (primary resource)
	enqueuer runtime.Enqueuer
}

func (r *resourceEventHandler) OnAdd(obj interface{}, isInInitialList bool) {
	_ = isInInitialList
	metaObj, err := runtime.GetMetaObject(obj)
	if err != nil {
		klog.Errorf("[secondarywatch] OnAdd: invalid object, type=%T", obj)
		return
	}

	if err := r.enqueuer.Enqueue(metaObj); err != nil {
		klog.V(3).Infof("[secondarywatch] OnAdd: failed to enqueue owner - %s", err.Error())
	}
}

// OnUpdate creates UpdateEvent and calls Update on EventHandler
func (r *resourceEventHandler) OnUpdate(oldObj, newObj interface{}) {
	oldMetaObj, err := runtime.GetMetaObject(oldObj)
	if err != nil {
		klog.Errorf("[secondarywatch] OnUpdate: invalid object, type=%T", oldObj)
		return
	}

	newMetaObj, err := runtime.GetMetaObject(newObj)
	if err != nil {
		klog.Errorf("[secondarywatch] OnUpdate: invalid object, type=%T", newObj)
		return
	}

	if oldMetaObj.GetResourceVersion() == newMetaObj.GetResourceVersion() {
		klog.V(3).Infof("[secondarywatch] OnUpdate: resource version has not changed, not going to enqueue owner, type=%T resource-version=%s", newMetaObj, newMetaObj.GetResourceVersion())
		return
	}

	if err := r.enqueuer.Enqueue(newMetaObj); err != nil {
		klog.V(3).Infof("[secondarywatch] OnUpdate: failed to enqueue owner - %s", err.Error())
	}
}

func (r *resourceEventHandler) OnDelete(obj interface{}) {
	metaObj, err := runtime.GetMetaObject(obj)
	if err != nil {
		klog.Errorf("[secondarywatch] OnDelete: invalid object, type=%T", obj)
		return
	}

	if err := r.enqueuer.Enqueue(metaObj); err != nil {
		klog.V(3).Infof("[secondarywatch] OnDelete: failed to enqueue owner - %s", err.Error())
	}
}

type directEnqueueHandler struct {
	directEnqueuer runtime.DirectEnqueuer
	primaryCRName  string
}

var _ cache.ResourceEventHandler = &directEnqueueHandler{}

func newDirectEnqueueHandler(de runtime.DirectEnqueuer, primaryCRName string) cache.ResourceEventHandler {
	return &directEnqueueHandler{directEnqueuer: de, primaryCRName: primaryCRName}
}

func (d *directEnqueueHandler) OnAdd(_ interface{}, _ bool) {
	// No-op: we only care about updates
}

func (d *directEnqueueHandler) OnUpdate(oldObj, newObj interface{}) {
	oldAcc, err := meta.Accessor(oldObj)
	if err != nil {
		klog.Errorf("[secondarywatch] directEnqueue OnUpdate: cannot access old object metadata: %v", err)
		return
	}
	newAcc, err := meta.Accessor(newObj)
	if err != nil {
		klog.Errorf("[secondarywatch] directEnqueue OnUpdate: cannot access new object metadata: %v", err)
		return
	}
	if oldAcc.GetResourceVersion() == newAcc.GetResourceVersion() {
		return
	}
	klog.V(4).Infof("[secondarywatch] cluster APIServer config changed, enqueueing primary CR %q", d.primaryCRName)
	if err := d.directEnqueuer.EnqueueByName(d.primaryCRName); err != nil {
		klog.V(3).Infof("[secondarywatch] directEnqueue OnUpdate: failed to enqueue primary - %s", err.Error())
	}
}

func (d *directEnqueueHandler) OnDelete(_ interface{}) {}
