package reconciler

import (
	"fmt"

	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/clusterresourceoverride/internal/handlers"
	dynamicclient "github.com/openshift/cluster-resource-override-admission-operator/pkg/dynamic"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/clientset/versioned"
	autoscalingv1listers "github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/listers/autoscaling/v1"
	operatorruntime "github.com/openshift/cluster-resource-override-admission-operator/pkg/runtime"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog"
	controllerreconciler "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	ClusterResourceOverrideGVK = schema.GroupVersionKind{
		Group:   autoscalingv1.GroupName,
		Version: autoscalingv1.GroupVersion,
		Kind:    autoscalingv1.ClusterResourceOverrideKind,
	}
)

func NewReconciler(options *handlers.Options) *reconciler {
	handlers := HandlerChain{
		handlers.NewAvailabilityHandler(options),
		handlers.NewValidationHandler(options),
		handlers.NewConfigurationHandler(options),
		handlers.NewCertGenerationHandler(options),
		handlers.NewCertReadyHandler(options),
		handlers.NewDaemonSetHandler(options),
		handlers.NewDeploymentReadyHandler(options),
		handlers.NewWebhookConfigurationHandlerHandler(options),
		handlers.NewAvailabilityHandler(options),
	}

	return &reconciler{
		client:   options.Client.Operator,
		lister:   options.PrimaryLister,
		handlers: handlers,
		updater: &StatusUpdater{
			client: options.Client.Operator,
		},
		operandContext: options.OperandContext,
		dynamic:        options.Client.Dynamic,
	}
}

type reconciler struct {
	client         versioned.Interface
	lister         autoscalingv1listers.ClusterResourceOverrideLister
	handlers       HandlerChain
	updater        *StatusUpdater
	operandContext operatorruntime.OperandContext
	dynamic        dynamicclient.Ensurer
}

func (r *reconciler) Reconcile(request controllerreconciler.Request) (result controllerreconciler.Result, err error) {
	klog.V(4).Infof("key=%s new request for reconcile", request.Name)

	result = controllerreconciler.Result{}

	// The operand is a singleton, so we are only interested in the CR specified in cluster
	if request.Name != r.operandContext.ResourceName() {
		klog.V(2).Infof("key=%s skipping reconcile", request.Name)
		return
	}

	original, getErr := r.lister.Get(request.Name)
	if getErr != nil {
		if k8serrors.IsNotFound(getErr) {
			klog.Errorf("[reconciler] key=%s object has been deleted - %s", request.Name, getErr.Error())
			return
		}

		// Otherwise, we will requeue.
		klog.Errorf("[reconciler] key=%s unexpected error - %s", request.Name, getErr.Error())
		err = getErr
		return
	}

	copy := original.DeepCopy()
	copy.SetGroupVersionKind(ClusterResourceOverrideGVK)

	reconcileContext := handlers.NewReconcileRequestContext(r.operandContext)
	current, result, err := r.handlers.Handle(reconcileContext, copy)

	updateErr := r.updater.Update(original, current)
	if updateErr != nil {
		klog.Errorf("[reconciler] key=%s failed to update status - %s", request.Name, updateErr.Error())

		if err != nil {
			err = fmt.Errorf("[reconciler] reconciliation error - %s -- update status error - %s", err.Error(), updateErr.Error())
			return
		}

		err = updateErr
	}

	return
}
