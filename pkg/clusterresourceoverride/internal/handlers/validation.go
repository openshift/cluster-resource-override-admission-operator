package handlers

import (
	operatorv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/operator/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/clusterresourceoverride/internal/condition"
	controllerreconciler "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func NewValidationHandler(o *Options) *validationHandler {
	return &validationHandler{}
}

type validationHandler struct {
}

func (c *validationHandler) Handle(context *ReconcileRequestContext, original *operatorv1.ClusterResourceOverride) (current *operatorv1.ClusterResourceOverride, result controllerreconciler.Result, handleErr error) {
	current = original

	podResourceOverrideValidationErr := original.Spec.PodResourceOverride.Spec.Validate()
	if podResourceOverrideValidationErr != nil {
		handleErr = condition.NewInstallReadinessError(operatorv1.InvalidParameters, podResourceOverrideValidationErr)
	}

	deploymentOverridesValidationErr := original.Spec.DeploymentOverrides.Validate()
	if deploymentOverridesValidationErr != nil {
		handleErr = condition.NewInstallReadinessError(operatorv1.InvalidParameters, deploymentOverridesValidationErr)
	}

	return
}
