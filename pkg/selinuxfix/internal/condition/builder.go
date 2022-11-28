package condition

import (
	"time"

	selinuxfixv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/selinuxfix/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/clock"
)

func NewBuilderWithStatus(status *selinuxfixv1.SelinuxFixOverrideStatus) *Builder {
	return &Builder{
		clock:  clock.RealClock{},
		status: status,
	}
}

func Find(status *selinuxfixv1.SelinuxFixOverrideStatus, conditionType selinuxfixv1.SelinuxFixOverrideConditionType) *selinuxfixv1.SelinuxFixOverrideCondition {
	for i := range status.Conditions {
		c := &status.Conditions[i]
		if c.Type == conditionType {
			return c
		}
	}

	return nil
}

// Equal returns true if the two given conditions are equal.
// We deem two conditions equal if Type, Status, Reason and Message are a match
// (despite LastTransitionTime being different).
func Equal(this, that *selinuxfixv1.SelinuxFixOverrideCondition) bool {
	if this.Type == that.Type &&
		this.Status == that.Status &&
		this.Reason == that.Reason &&
		this.Message == that.Message {
		return true
	}

	return false
}

func DeepCopyWithDefaultLastTransitionTime(status *selinuxfixv1.SelinuxFixOverrideStatus) (copy *selinuxfixv1.SelinuxFixOverrideStatus) {
	copy = status.DeepCopy()
	for i := range copy.Conditions {
		copy.Conditions[i].LastTransitionTime = metav1.NewTime(time.Time{})
	}

	return
}

type Builder struct {
	clock  clock.Clock
	status *selinuxfixv1.SelinuxFixOverrideStatus
}

func (b *Builder) WithError(err error) (builder *Builder) {
	builder = b
	if err == nil {
		return
	}

	b.init()

	desired := FromError(err, metav1.NewTime(b.clock.Now()))
	b.WithCondition(desired)

	return b
}

func (b *Builder) WithInstallReady() (builder *Builder) {
	b.init()

	desired := &selinuxfixv1.SelinuxFixOverrideCondition{
		Type:               selinuxfixv1.InstallReadinessFailure,
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.NewTime(b.clock.Now()),
	}
	b.WithCondition(desired)

	return b
}

func (b *Builder) WithAvailable(status corev1.ConditionStatus, message string) (builder *Builder) {
	b.init()

	desired := &selinuxfixv1.SelinuxFixOverrideCondition{
		Type:               selinuxfixv1.Available,
		Status:             status,
		Message:            message,
		LastTransitionTime: metav1.NewTime(b.clock.Now()),
	}
	b.WithCondition(desired)

	return b
}

func (b *Builder) WithCondition(desired *selinuxfixv1.SelinuxFixOverrideCondition) {
	if desired == nil {
		return
	}

	current := Find(b.status, desired.Type)
	if current == nil {
		b.status.Conditions = append(b.status.Conditions, *desired)
		return
	}

	if Equal(desired, current) {
		return
	}

	current.Reason = desired.Reason
	current.Message = desired.Message
	current.Status = desired.Status
	current.LastTransitionTime = desired.LastTransitionTime
}

func (b *Builder) init() {
	if b.status == nil {
		b.status.Conditions = []selinuxfixv1.SelinuxFixOverrideCondition{}
	}
}
