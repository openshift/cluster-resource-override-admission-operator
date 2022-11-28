package reconciler

import (
	"context"
	"reflect"

	selinuxfixv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/selinuxfix/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StatusUpdater updates the status of a SelinuxFix resource.
type StatusUpdater struct {
	client versioned.Interface
}

// Update updates the status of a SelinuxFix resource.
// If the status inside of the desired object is equal to that of the observed then
// the function does not make an update call.
func (u *StatusUpdater) Update(observed, desired *selinuxfixv1.SelinuxFixOverride) error {
	if reflect.DeepEqual(&observed.Status, &desired.Status) {
		return nil
	}
	_, err := u.client.SelinuxfixV1().SelinuxFixOverrides().UpdateStatus(context.TODO(), desired, metav1.UpdateOptions{})
	return err
}
