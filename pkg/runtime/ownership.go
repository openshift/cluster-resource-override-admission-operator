package runtime

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Object is used to build an OwnerReference, and we need type and object metadata
type Object interface {
	metav1.Object
	runtime.Object
}

// SetControllerFunc sets ownership given an owner and an owned object.
type SetControllerFunc func(owned metav1.Object, owner Object)

func (s SetControllerFunc) Set(owned metav1.Object, owner Object) {
	s(owned, owner)
}

func SetController(owned metav1.Object, owner Object) {
	gvk := owner.GetObjectKind().GroupVersionKind()
	ownerReference := metav1.NewControllerRef(owner, gvk)

	if metav1.IsControlledBy(owned, owner) {
		return
	}

	refs := owned.GetOwnerReferences()
	if refs == nil {
		refs = []metav1.OwnerReference{
			*ownerReference,
		}

		owned.SetOwnerReferences(refs)
		return
	}

	refs = append(refs, *ownerReference)
	owned.SetOwnerReferences(refs)
}

func IsOwner(owned metav1.Object, owner metav1.OwnerReference) bool {
	for _, ref := range owned.GetOwnerReferences() {
		if ref.Kind == owner.Kind {
			if ref.Name == owner.Name && ref.UID == owner.UID {
				return true
			}
		}
	}

	return false
}

// NonBlockingOwner returns an owner reference to be added.
func NonBlockingOwner(owner Object) metav1.OwnerReference {
	var (
		controller         = false
		blockOwnerDeletion = false
	)

	gvk := owner.GetObjectKind().GroupVersionKind()
	apiVersion, kind := gvk.ToAPIVersionAndKind()

	return metav1.OwnerReference{
		APIVersion:         apiVersion,
		Kind:               kind,
		Name:               owner.GetName(),
		UID:                owner.GetUID(),
		BlockOwnerDeletion: &blockOwnerDeletion,
		Controller:         &controller,
	}
}
