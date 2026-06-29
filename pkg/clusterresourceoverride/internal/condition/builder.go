package condition

import (
	operatorv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/operator/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/clock"
	"time"
)

func NewBuilderWithStatus(status *operatorv1.ClusterResourceOverrideStatus) *Builder {
	return &Builder{
		clock:  clock.RealClock{},
		status: status,
	}
}

func Find(status *operatorv1.ClusterResourceOverrideStatus, conditionType operatorv1.ClusterResourceOverrideConditionType) *operatorv1.ClusterResourceOverrideCondition {
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
func Equal(this, that *operatorv1.ClusterResourceOverrideCondition) bool {
	if this.Type == that.Type &&
		this.Status == that.Status &&
		this.Reason == that.Reason &&
		this.Message == that.Message {
		return true
	}

	return false
}

func DeepCopyWithDefaultLastTransitionTime(status *operatorv1.ClusterResourceOverrideStatus) (copy *operatorv1.ClusterResourceOverrideStatus) {
	copy = status.DeepCopy()
	for i := range copy.Conditions {
		copy.Conditions[i].LastTransitionTime = metav1.NewTime(time.Time{})
	}

	return
}

type Builder struct {
	clock  clock.Clock
	status *operatorv1.ClusterResourceOverrideStatus
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

	desired := &operatorv1.ClusterResourceOverrideCondition{
		Type:               operatorv1.InstallReadinessFailure,
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.NewTime(b.clock.Now()),
	}
	b.WithCondition(desired)

	return b
}

func (b *Builder) WithAvailable(status corev1.ConditionStatus, message string) (builder *Builder) {
	b.init()

	desired := &operatorv1.ClusterResourceOverrideCondition{
		Type:               operatorv1.Available,
		Status:             status,
		Message:            message,
		LastTransitionTime: metav1.NewTime(b.clock.Now()),
	}
	b.WithCondition(desired)

	return b
}

func (b *Builder) WithCondition(desired *operatorv1.ClusterResourceOverrideCondition) {
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
		b.status.Conditions = []operatorv1.ClusterResourceOverrideCondition{}
	}
}
