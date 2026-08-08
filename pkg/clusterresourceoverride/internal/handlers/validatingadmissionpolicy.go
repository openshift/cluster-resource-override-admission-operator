package handlers

import (
	operatorv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/operator/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/reference"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/asset"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/clusterresourceoverride/internal/condition"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/ensurer"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/secondarywatch"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	controllerreconciler "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func NewValidatingAdmissionPolicyHandler(o *Options) *validatingAdmissionPolicyHandler {
	return &validatingAdmissionPolicyHandler{
		policyEnsurer:  ensurer.NewValidatingAdmissionPolicyEnsurer(o.Client.Dynamic),
		bindingEnsurer: ensurer.NewValidatingAdmissionPolicyBindingEnsurer(o.Client.Dynamic),
		lister:         o.SecondaryLister,
		asset:          o.Asset,
	}
}

type validatingAdmissionPolicyHandler struct {
	policyEnsurer  *ensurer.ValidatingAdmissionPolicyEnsurer
	bindingEnsurer *ensurer.ValidatingAdmissionPolicyBindingEnsurer
	lister         *secondarywatch.Lister
	asset          *asset.Asset
}

func (v *validatingAdmissionPolicyHandler) Handle(context *ReconcileRequestContext, original *operatorv1.ClusterResourceOverride) (current *operatorv1.ClusterResourceOverride, result controllerreconciler.Result, handleErr error) {
	current = original

	// Ensure ValidatingAdmissionPolicy
	policyEnsure := false
	policyName := v.asset.NewValidatingAdmissionPolicy().Name()
	policyObject, err := v.lister.AdmissionRegistrationV1ValidatingAdmissionPolicyLister().Get(policyName)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			handleErr = condition.NewInstallReadinessError(operatorv1.InternalError, err)
			return
		}
		policyEnsure = true
	}

	if policyEnsure {
		desired := v.asset.NewValidatingAdmissionPolicy().New()
		context.ControllerSetter().Set(desired, original)

		policy, err := v.policyEnsurer.Ensure(desired)
		if err != nil {
			handleErr = condition.NewInstallReadinessError(operatorv1.InternalError, err)
			return
		}

		policyObject = policy
		klog.V(2).Infof("key=%s resource=%T/%s successfully created", original.Name, policyObject, policyObject.Name)
	}

	if ref := original.Status.Resources.ValidatingAdmissionPolicyRef; ref != nil && ref.ResourceVersion == policyObject.ResourceVersion {
		klog.V(2).Infof("key=%s resource=%T/%s is in sync", original.Name, policyObject, policyObject.Name)
	} else {
		newRef, err := reference.GetReference(policyObject)
		if err != nil {
			handleErr = condition.NewInstallReadinessError(operatorv1.CannotSetReference, err)
			return
		}

		klog.V(2).Infof("key=%s resource=%T/%s resource-version=%s setting object reference", original.Name, policyObject, policyObject.Name, newRef.ResourceVersion)
		current.Status.Resources.ValidatingAdmissionPolicyRef = newRef
	}

	// Ensure ValidatingAdmissionPolicyBinding
	bindingEnsure := false
	bindingName := v.asset.NewValidatingAdmissionPolicyBinding().Name()
	bindingObject, err := v.lister.AdmissionRegistrationV1ValidatingAdmissionPolicyBindingLister().Get(bindingName)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			handleErr = condition.NewInstallReadinessError(operatorv1.InternalError, err)
			return
		}
		bindingEnsure = true
	}

	if bindingEnsure {
		desired := v.asset.NewValidatingAdmissionPolicyBinding().New()
		context.ControllerSetter().Set(desired, original)

		binding, err := v.bindingEnsurer.Ensure(desired)
		if err != nil {
			handleErr = condition.NewInstallReadinessError(operatorv1.InternalError, err)
			return
		}

		bindingObject = binding
		klog.V(2).Infof("key=%s resource=%T/%s successfully created", original.Name, bindingObject, bindingObject.Name)
	}

	if ref := original.Status.Resources.ValidatingAdmissionPolicyBindingRef; ref != nil && ref.ResourceVersion == bindingObject.ResourceVersion {
		klog.V(2).Infof("key=%s resource=%T/%s is in sync", original.Name, bindingObject, bindingObject.Name)
		return
	}

	newRef, err := reference.GetReference(bindingObject)
	if err != nil {
		handleErr = condition.NewInstallReadinessError(operatorv1.CannotSetReference, err)
		return
	}

	klog.V(2).Infof("key=%s resource=%T/%s resource-version=%s setting object reference", original.Name, bindingObject, bindingObject.Name, newRef.ResourceVersion)
	current.Status.Resources.ValidatingAdmissionPolicyBindingRef = newRef

	return
}
