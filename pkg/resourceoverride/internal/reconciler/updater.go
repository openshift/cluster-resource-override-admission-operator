package reconciler

import (
	"context"
	"reflect"

	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StatusUpdater updates the status of a ResourceOverride resource.
type StatusUpdater struct {
	client versioned.Interface
}

// Update updates the status of a ResourceOverride resource.
// If the status inside of the desired object is equal to that of the observed then
// the function does not make an update call.
func (u *StatusUpdater) Update(observed, desired *autoscalingv1.ResourceOverride) error {
	if reflect.DeepEqual(&observed.Status, &desired.Status) {
		return nil
	}

	_, err := u.client.AutoscalingV1().ResourceOverrides(desired.GetNamespace()).UpdateStatus(context.TODO(), desired, metav1.UpdateOptions{})
	return err
}
