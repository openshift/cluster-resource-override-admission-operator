package reconciler

import (
	"context"
	"fmt"

	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/asset"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/clientset/versioned"
	autoscalingv1listers "github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/listers/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/resourceoverride/internal/condition"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	corev1listers "k8s.io/client-go/listers/core/v1"
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
	client          versioned.Interface
	lister          autoscalingv1listers.ResourceOverrideLister
	namespaceLister corev1listers.NamespaceLister
	updater         *StatusUpdater
}

func NewReconciler(client versioned.Interface, lister autoscalingv1listers.ResourceOverrideLister, namespaceLister corev1listers.NamespaceLister) *reconciler {
	return &reconciler{
		client:          client,
		lister:          lister,
		namespaceLister: namespaceLister,
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

	if nsErr := r.checkNamespaceOptIn(copy); nsErr != nil {
		klog.Errorf("[reconciler] key=%s failed to check namespace opt-in - %s", request.Name, nsErr.Error())
		err = nsErr
		return
	}

	err = r.updater.Update(original, copy)
	if err != nil {
		klog.Errorf("[reconciler] key=%s failed to update status - %s", request.Name, err.Error())
	}

	return
}

func Validate(current *autoscalingv1.ResourceOverride) {
	builder := condition.NewBuilderWithStatus(&current.Status)

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

func (r *reconciler) checkNamespaceOptIn(current *autoscalingv1.ResourceOverride) error {
	ns, err := r.namespaceLister.Get(current.Namespace)
	if err != nil {
		return err
	}

	builder := condition.NewBuilderWithStatus(&current.Status)
	if ns.Labels[asset.NamespaceOptInLabelKey] != "true" {
		builder.WithIgnored(autoscalingv1.NamespaceNotOptedIn, fmt.Sprintf("namespace %q does not have the %s=true label", current.Namespace, asset.NamespaceOptInLabelKey))
		return nil
	}

	builder.WithIgnoredCleared()
	return nil
}
