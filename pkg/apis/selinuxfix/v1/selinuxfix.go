package v1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (in *SelinuxFixOverride) IsTimeToRotateCert() bool {
	if in.Status.CertsRotateAt.IsZero() {
		return true
	}

	now := metav1.Now()
	return in.Status.CertsRotateAt.Before(&now)
}

func (in *SelinuxFixOverride) String() string {
	return fmt.Sprintf("Enabled=%v", in.Spec.Enabled)
}

func (in *SelinuxFixOverride) Validate() error {
	return nil
}
