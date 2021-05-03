package dynamic

import (
	"context"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	unstructuredv1 "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgodynamic "k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

func NewForConfig(config *rest.Config) (ensurer Ensurer, err error) {
	dynamic, err := clientgodynamic.NewForConfig(config)
	if err != nil {
		err = fmt.Errorf("error creating dynamic client - %v", err)
		return
	}

	ensurer = NewEnsurer(dynamic)
	return
}

func NewEnsurer(dynamic clientgodynamic.Interface) Ensurer {
	return &client{
		dynamic: dynamic,
	}
}

type client struct {
	dynamic clientgodynamic.Interface
}

func (c *client) Ensure(resource string, object runtime.Object) (current *unstructuredv1.Unstructured, err error) {
	modified, err := ToUnstructured(object)
	if err != nil {
		err = fmt.Errorf("failed to convert to unstructured - %s", err.Error())
		return
	}

	kind := modified.GetKind()
	client := c.resource(resource, modified)

	created, createErr := client.Create(context.TODO(), modified, metav1.CreateOptions{})
	if createErr == nil {
		current = created
		return
	}

	if !k8serrors.IsAlreadyExists(createErr) {
		err = fmt.Errorf("failed to create %s - %s", kind, createErr.Error())
		return
	}

	original, getErr := client.Get(context.TODO(), modified.GetName(), metav1.GetOptions{})
	if getErr != nil {
		err = fmt.Errorf("failed to retrieve %s - %s", kind, getErr.Error())
		return
	}

	modified.SetResourceVersion(original.GetResourceVersion())
	modified.SetUID(original.GetUID())

	bytes, patchErr := PatchWithUnstructured(original, modified, object)
	if patchErr != nil {
		err = fmt.Errorf("failed to generate patch %s - %s", kind, patchErr.Error())
	}

	current, err = client.Patch(context.TODO(), modified.GetName(), types.StrategicMergePatchType, bytes, metav1.PatchOptions{})
	return
}

func (c *client) resource(resource string, unstructured *unstructuredv1.Unstructured) clientgodynamic.ResourceInterface {
	gvr := GetGVR(resource, unstructured)
	client := c.dynamic.Resource(gvr)

	namespace := unstructured.GetNamespace()
	if namespace == metav1.NamespaceNone {
		return client
	}

	return client.Namespace(namespace)
}
