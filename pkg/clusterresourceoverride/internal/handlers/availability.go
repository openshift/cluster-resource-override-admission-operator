package handlers

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	controllerreconciler "sigs.k8s.io/controller-runtime/pkg/reconcile"

	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/asset"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/clusterresourceoverride/internal/condition"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/deploy"
)

func NewAvailabilityHandler(o *Options) *availabilityHandler {
	return &availabilityHandler{
		asset:  o.Asset,
		deploy: o.Deploy,
	}
}

type availabilityHandler struct {
	asset  *asset.Asset
	deploy deploy.Interface
}

func (a *availabilityHandler) Handle(context *ReconcileRequestContext, original *autoscalingv1.ClusterResourceOverride) (current *autoscalingv1.ClusterResourceOverride, result controllerreconciler.Result, handleErr error) {
	current = original
	builder := condition.NewBuilderWithStatus(&current.Status)

	available, err := a.deploy.IsAvailable()

	switch {
	case available:
		builder.WithAvailable(corev1.ConditionTrue, "")
	case err == nil:
		builder.WithError(condition.NewAvailableError(autoscalingv1.AdmissionWebhookNotAvailable, fmt.Errorf("name=%s deployment not complete", a.deploy.Name())))
	case k8serrors.IsNotFound(err):
		builder.WithError(condition.NewAvailableError(autoscalingv1.AdmissionWebhookNotAvailable, err))
	default:
		builder.WithError(condition.NewAvailableError(autoscalingv1.InternalError, err))
	}

	return
}
