package handlers

import (
	selinuxfixv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/selinuxfix/v1"
	controllerreconciler "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func NewValidationHandler(o *Options) *validationHandler {
	return &validationHandler{}
}

type validationHandler struct {
}

func (c *validationHandler) Handle(context *ReconcileRequestContext, original *selinuxfixv1.SelinuxFixOverride) (current *selinuxfixv1.SelinuxFixOverride, result controllerreconciler.Result, handleErr error) {
	current = original
	return
}
