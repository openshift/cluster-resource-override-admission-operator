package handlers

import (
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/asset"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/cert"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/deploy"
	autoscalingv1listers "github.com/openshift/cluster-resource-override-admission-operator/pkg/generated/listers/autoscaling/v1"
	operatorruntime "github.com/openshift/cluster-resource-override-admission-operator/pkg/runtime"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/secondarywatch"
)

func NewReconcileRequestContext(oc operatorruntime.OperandContext) *ReconcileRequestContext {
	return &ReconcileRequestContext{
		OperandContext: oc,
	}
}

type Options struct {
	OperandContext  operatorruntime.OperandContext
	Client          *operatorruntime.Client
	PrimaryLister   autoscalingv1listers.ClusterResourceOverrideLister
	SecondaryLister *secondarywatch.Lister
	Asset           *asset.Asset
	Deploy          deploy.Interface
}

type ReconcileRequestContext struct {
	operatorruntime.OperandContext
	bundle *cert.Bundle
}

func (r *ReconcileRequestContext) SetBundle(bundle *cert.Bundle) {
	r.bundle = bundle
}

func (r *ReconcileRequestContext) GetBundle() *cert.Bundle {
	return r.bundle
}

func (r *ReconcileRequestContext) ControllerSetter() operatorruntime.SetControllerFunc {
	return operatorruntime.SetController
}
