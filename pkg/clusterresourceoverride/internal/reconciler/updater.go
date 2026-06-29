package reconciler

import (
	"context"
	"reflect"

	operatorv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/operator/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StatusUpdater updates the status of a ClusterResourceOverride resource.
type StatusUpdater struct {
	client versioned.Interface
}

// Update updates the status of a ClusterResourceOverride resource.
// If the status inside of the desired object is equal to that of the observed then
// the function does not make an update call.
func (u *StatusUpdater) Update(observed, desired *operatorv1.ClusterResourceOverride) error {
	if reflect.DeepEqual(&observed.Status, &desired.Status) {
		return nil
	}

	_, err := u.client.OperatorV1().ClusterResourceOverrides().UpdateStatus(context.TODO(), desired, metav1.UpdateOptions{})
	return err
}
