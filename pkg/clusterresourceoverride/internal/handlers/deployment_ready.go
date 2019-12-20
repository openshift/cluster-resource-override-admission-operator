package handlers

import (
	"fmt"

	"k8s.io/klog"
	controllerreconciler "sigs.k8s.io/controller-runtime/pkg/reconcile"

	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/clusterresourceoverride/internal/condition"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/deploy"
)

func NewDeploymentReadyHandler(o *Options) *deploymentReadyHandler {
	return &deploymentReadyHandler{
		deploy: o.Deploy,
	}
}

type deploymentReadyHandler struct {
	deploy deploy.Interface
}

func (c *deploymentReadyHandler) Handle(context *ReconcileRequestContext, original *autoscalingv1.ClusterResourceOverride) (current *autoscalingv1.ClusterResourceOverride, result controllerreconciler.Result, handleErr error) {
	current = original

	available, err := c.deploy.IsAvailable()
	if available {
		klog.V(2).Infof("key=%s resource=%s deployment is ready", original.Name, c.deploy.Name())

		condition.NewBuilderWithStatus(&current.Status).WithInstallReady()
		current.Status.Version = context.OperandVersion()
		current.Status.Image = context.OperandImage()
		return
	}

	klog.V(2).Infof("key=%s resource=%s deployment is not ready", original.Name, c.deploy.Name())

	if err == nil {
		err = fmt.Errorf("name=%s waiting for deployment to complete", c.deploy.Name())
	}

	handleErr = condition.NewInstallReadinessError(autoscalingv1.DeploymentNotReady, err)
	return
}
