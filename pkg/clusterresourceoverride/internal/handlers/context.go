package handlers

import (
	"fmt"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling"
	autoscalingv1listers "github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/listers/autoscaling/v1"
	operatorruntime "github.com/openshift/cluster-resource-override-admission-operator/pkg/runtime"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/secondarywatch"
)

func NewReconcileRequestContext(oc operatorruntime.OperandContext) *ReconcileRequestContext {
	return &ReconcileRequestContext{
		OperandContext: oc,
		RequestContext: RequestContext{},
	}
}

type Options struct {
	OperandContext operatorruntime.OperandContext
	Client         *operatorruntime.Client
	CROLister      autoscalingv1listers.ClusterResourceOverrideLister
	KubeLister     *secondarywatch.Lister
}

type ReconcileRequestContext struct {
	operatorruntime.OperandContext
	RequestContext
}

type RequestContext struct {
	configurationHash string
}

func (r *RequestContext) ControllerSetter() operatorruntime.SetControllerFunc {
	return operatorruntime.SetController
}

func (r *ReconcileRequestContext) GetConfigurationHashAnnotationKey() string {
	return fmt.Sprintf("%s.%s/configuration.hash", r.WebhookName(), autoscaling.GroupName)
}

func (r *ReconcileRequestContext) GetServingCertHashAnnotationKey() string {
	return fmt.Sprintf("%s.%s/servingcert.hash", r.WebhookName(), autoscaling.GroupName)
}

func (r *ReconcileRequestContext) GetOwnerAnnotationKey() string {
	return fmt.Sprintf("%s.%s/owner", r.WebhookName(), autoscaling.GroupName)
}
