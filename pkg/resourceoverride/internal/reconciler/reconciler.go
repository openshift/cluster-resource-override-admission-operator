package reconciler

import (
	"context"
	"fmt"
	"strings"

	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/clientset/versioned"
	autoscalingv1listers "github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/listers/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/resourceoverride/internal/condition"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	controllerreconciler "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	ResourceOverrideGVK = schema.GroupVersionKind{
		Group:   autoscalingv1.GroupName,
		Version: autoscalingv1.GroupVersion,
		Kind:    autoscalingv1.ResourceOverrideKind,
	}
)

type reconciler struct {
	client  versioned.Interface
	lister  autoscalingv1listers.ResourceOverrideLister
	updater *StatusUpdater
}

func NewReconciler(client versioned.Interface, lister autoscalingv1listers.ResourceOverrideLister) *reconciler {
	return &reconciler{
		client: client,
		lister: lister,
		updater: &StatusUpdater{
			client: client,
		},
	}
}

func (r *reconciler) Reconcile(ctx context.Context, request controllerreconciler.Request) (result controllerreconciler.Result, err error) {
	klog.V(4).Infof("key=%s new request for reconcile", request.Name)

	original, getErr := r.lister.ResourceOverrides(request.Namespace).Get(request.Name)
	if getErr != nil {
		if k8serrors.IsNotFound(getErr) {
			klog.V(4).Infof("[reconciler] key=%s object has been deleted - %s", request.Name, getErr.Error())
			return
		}

		// Otherwise, we will requeue.
		klog.Errorf("[reconciler] key=%s unexpected error - %s", request.Name, getErr.Error())
		err = getErr
		return
	}

	copy := original.DeepCopy()
	copy.SetGroupVersionKind(ResourceOverrideGVK)

	Validate(copy)

	err = r.updater.Update(original, copy)
	if err != nil {
		klog.Errorf("[reconciler] key=%s failed to update status - %s", request.Name, err.Error())
	}

	return
}

func Validate(current *autoscalingv1.ResourceOverride) {
	builder := condition.NewBuilderWithStatus(&current.Status)

	if isExemptNamespace(current.Namespace) {
		builder.WithValidationFailure(autoscalingv1.ExemptNamespace, fmt.Sprintf("resourceoverride %s/%s is in an exempt namespace", current.Namespace, current.Name))
		return
	}

	validationErr := current.Spec.PodResourceOverride.Validate()
	if validationErr != nil {
		builder.WithValidationFailure(autoscalingv1.InvalidParameters, fmt.Sprintf("resourceoverride %s/%s has invalid parameters: %s", current.Namespace, current.Name, validationErr.Error()))
		return
	}

	if current.Spec.PodSelector != nil {
		if _, selectorErr := metav1.LabelSelectorAsSelector(current.Spec.PodSelector); selectorErr != nil {
			builder.WithValidationFailure(autoscalingv1.InvalidParameters, fmt.Sprintf("resourceoverride %s/%s has invalid podSelector field: %s", current.Namespace, current.Name, selectorErr.Error()))
			return
		}
	}

	builder.WithValidationCleared()
}

func isExemptNamespace(namespace string) bool {
	switch namespace {
	case "openshift", "kube", "kubernetes":
		return true
	}
	return strings.HasPrefix(namespace, "openshift-") ||
		strings.HasPrefix(namespace, "kube-") ||
		strings.HasPrefix(namespace, "kubernetes-")
}
