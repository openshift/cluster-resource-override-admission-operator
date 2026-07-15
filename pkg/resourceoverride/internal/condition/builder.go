package condition

import (
	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/clock"
)

type Builder struct {
	clock  clock.Clock
	status *autoscalingv1.ResourceOverrideStatus
}

func NewBuilderWithStatus(status *autoscalingv1.ResourceOverrideStatus) *Builder {
	return &Builder{
		clock:  clock.RealClock{},
		status: status,
	}
}

func (b *Builder) init() {
	if b.status.Conditions == nil {
		b.status.Conditions = []autoscalingv1.ResourceOverrideCondition{}
	}
}

func (b *Builder) WithValidationFailure(reason string, message string) (builder *Builder) {
	b.init()

	desired := &autoscalingv1.ResourceOverrideCondition{
		Type:               autoscalingv1.ValidationFailure,
		Status:             corev1.ConditionTrue,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.NewTime(b.clock.Now()),
	}
	b.WithCondition(desired)

	return b
}

func (b *Builder) WithValidationCleared() (builder *Builder) {
	b.init()

	desired := &autoscalingv1.ResourceOverrideCondition{
		Type:               autoscalingv1.ValidationFailure,
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.NewTime(b.clock.Now()),
	}
	b.WithCondition(desired)

	return b
}

func (b *Builder) WithCondition(desired *autoscalingv1.ResourceOverrideCondition) {
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

func Find(status *autoscalingv1.ResourceOverrideStatus, conditionType autoscalingv1.ResourceOverrideConditionType) *autoscalingv1.ResourceOverrideCondition {
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
func Equal(this, that *autoscalingv1.ResourceOverrideCondition) bool {
	if this.Type == that.Type &&
		this.Status == that.Status &&
		this.Reason == that.Reason &&
		this.Message == that.Message {
		return true
	}

	return false
}
