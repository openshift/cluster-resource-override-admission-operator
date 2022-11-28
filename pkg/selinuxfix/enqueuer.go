package selinuxfix

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	selinuxfixv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/selinuxfix/v1"
	selinuxfixv1listers "github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/listers/selinuxfix/v1"
	operatorruntime "github.com/openshift/cluster-resource-override-admission-operator/pkg/runtime"
)

type enqueuer struct {
	queue              workqueue.RateLimitingInterface
	lister             selinuxfixv1listers.SelinuxFixOverrideLister
	ownerAnnotationKey string
}

func (e *enqueuer) Enqueue(obj interface{}) error {
	metaObj, err := operatorruntime.GetMetaObject(obj)
	if err != nil {
		return err
	}

	ownerName := getOwnerName(e.ownerAnnotationKey, metaObj)
	if ownerName == "" {
		return fmt.Errorf("could not find owner for %s/%s", metaObj.GetNamespace(), metaObj.GetName())
	}

	cro, err := e.lister.Get(ownerName)
	if err != nil {
		return fmt.Errorf("ignoring request to enqueue - %s", err.Error())
	}

	request := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Namespace: cro.GetNamespace(),
			Name:      cro.GetName(),
		},
	}

	e.queue.Add(request)
	return nil
}

func getOwnerName(ownerAnnotationKey string, object metav1.Object) string {
	// We check for annotations and owner references
	// If both exist, owner references takes precedence.
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil && ownerRef.Kind == selinuxfixv1.SelinuxFixOVerrideKind {
		return ownerRef.Name
	}

	annotations := object.GetAnnotations()
	if len(annotations) > 0 {
		owner, ok := annotations[ownerAnnotationKey]
		if ok && owner != "" {
			return owner
		}
	}

	return ""
}
