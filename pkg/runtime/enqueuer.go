package runtime

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

// Enqueuer is an interface that wraps the Enqueue method.
//
// Enqueue looks up the owner from the given object (owned) and then
// enqueues the owner for reconciliation.
type Enqueuer interface {
	Enqueue(owned interface{}) error
}

func GetMetaObject(obj interface{}) (object metav1.Object, err error) {
	object, metaErr := meta.Accessor(obj)
	if metaErr == nil {
		return
	}

	tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
	if !ok {
		err = fmt.Errorf("error decoding object, type=%T - %s", obj, metaErr.Error())
		return
	}

	object, err = meta.Accessor(tombstone.Obj)
	return
}
