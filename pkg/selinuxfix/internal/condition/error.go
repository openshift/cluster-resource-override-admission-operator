package condition

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	selinuxfixv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/selinuxfix/v1"
)

func NewInstallReadinessError(reason string, err error) error {
	return &installReadinessError{
		reconciliationError{
			Reason: reason,
			Err:    err,
		},
	}
}

func NewAvailableError(reason string, err error) error {
	return &availableError{
		reconciliationError{
			Reason: reason,
			Err:    err,
		},
	}
}

func FromError(err error, time metav1.Time) *selinuxfixv1.SelinuxFixOverrideCondition {
	switch e := err.(type) {
	case *installReadinessError:
		return &selinuxfixv1.SelinuxFixOverrideCondition{
			Type:               selinuxfixv1.InstallReadinessFailure,
			Reason:             e.Reason,
			Message:            e.Error(),
			Status:             corev1.ConditionTrue,
			LastTransitionTime: time,
		}
	case *availableError:
		return &selinuxfixv1.SelinuxFixOverrideCondition{
			Type:               selinuxfixv1.Available,
			Reason:             e.Reason,
			Message:            e.Error(),
			Status:             corev1.ConditionFalse,
			LastTransitionTime: time,
		}
	}

	return nil
}

type installReadinessError struct {
	reconciliationError
}

type availableError struct {
	reconciliationError
}

type reconciliationError struct {
	Reason string
	Err    error
}

func (e *reconciliationError) Error() string {
	return e.Err.Error()
}
