package reference

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ref "k8s.io/client-go/tools/reference"
)

// GetReference returns an ObjectReference for the given object.
func GetReference(obj runtime.Object) (*corev1.ObjectReference, error) {
	return ref.GetReference(scheme.Scheme, obj)
}
