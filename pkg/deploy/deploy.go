package deploy

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Applier gives an opportunity to the caller to initialize a deployment
// artifact before it is created or updated.
type Applier func(object metav1.Object)

func (a Applier) Apply(object metav1.Object) {
	a(object)
}

type Interface interface {
	Name() string
	IsAvailable() (available bool, err error)
	Get() (object runtime.Object, accessor metav1.Object, err error)
	Ensure(parent, child Applier) (object runtime.Object, accessor metav1.Object, err error)
}
