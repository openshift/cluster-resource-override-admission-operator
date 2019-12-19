package dynamic

import (
	"encoding/json"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	unstructuredv1 "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

// Ensurer is an interface that has Ensure Methid.
//
// Ensure is an idempotent function that ensures that the object specified is stored.
// If the object does not exist it should be created
// If the object already exists then it should be updated to match the specified object.
type Ensurer interface {
	Ensure(resource string, object runtime.Object) (current *unstructuredv1.Unstructured, err error)
}

// ToUnstructured converts a given runtime.Object to a corresponding Unstructured type.
func ToUnstructured(object runtime.Object) (unstructured *unstructuredv1.Unstructured, err error) {
	raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
	if err != nil {
		return
	}

	unstructured = &unstructuredv1.Unstructured{
		Object: raw,
	}
	return
}

// GetGVR return returns group, version and resource from a given Unstructured object.
func GetGVR(resource string, unstructured *unstructuredv1.Unstructured) schema.GroupVersionResource {
	gvk := unstructured.GetObjectKind().GroupVersionKind()

	return schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: resource,
	}
}

// PatchWithUnstructured generates a patch that can be applied to match the desired object.
func PatchWithUnstructured(original, modified *unstructured.Unstructured, datastruct interface{}) (bytes []byte, err error) {
	originalData, err := original.MarshalJSON()
	if err != nil {
		return
	}

	modifiedData, err := modified.MarshalJSON()
	if err != nil {
		return
	}

	return strategicpatch.CreateTwoWayMergePatch(originalData, modifiedData, datastruct)
}

func PatchWithRuntimeObject(original, modified runtime.Object, datastruct interface{}) (bytes []byte, err error) {
	originalData, err := json.Marshal(original)
	if err != nil {
		return
	}

	modifiedData, err := json.Marshal(modified)
	if err != nil {
		return
	}

	return strategicpatch.CreateTwoWayMergePatch(originalData, modifiedData, datastruct)
}
