package handlers

import (
	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/clusterresourceoverride/internal/condition"
	controllerreconciler "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func NewValidationHandler(o *Options) *validationHandler {
	return &validationHandler{}
}

type validationHandler struct {
}

func (c *validationHandler) Handle(context *ReconcileRequestContext, original *autoscalingv1.ClusterResourceOverride) (current *autoscalingv1.ClusterResourceOverride, result controllerreconciler.Result, handleErr error) {
	current = original

	podResourceOverrideValidationErr := original.Spec.PodResourceOverride.Spec.Validate()
	if podResourceOverrideValidationErr != nil {
		handleErr = condition.NewInstallReadinessError(autoscalingv1.InvalidParameters, podResourceOverrideValidationErr)
	}

	deploymentOverridesValidationErr := original.Spec.DeploymentOverrides.Validate()
	if deploymentOverridesValidationErr != nil {
		handleErr = condition.NewInstallReadinessError(autoscalingv1.InvalidParameters, deploymentOverridesValidationErr)
	}

	return
}
