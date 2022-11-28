package handlers

import (
	"context"

	"github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/reference"
	selinuxfixv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/selinuxfix/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/asset-selinux"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/ensurer"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/selinuxfix/internal/condition"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	apiregistrationclientset "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	controllerreconciler "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func NewAPIServiceHandler(o *Options) *apiServiceHandler {
	return &apiServiceHandler{
		client:  o.Client.APIRegistration,
		ensurer: ensurer.NewAPIServiceEnsurer(o.Client.Dynamic),
		asset:   o.Asset,
	}
}

type apiServiceHandler struct {
	client  apiregistrationclientset.Interface
	ensurer *ensurer.APIServiceEnsurer
	asset   *asset.Asset
}

func (a *apiServiceHandler) Handle(ctx *ReconcileRequestContext, original *selinuxfixv1.SelinuxFixOverride) (current *selinuxfixv1.SelinuxFixOverride, result controllerreconciler.Result, handleErr error) {
	current = original

	name := a.asset.APIService().Name()
	object, err := a.client.ApiregistrationV1().APIServices().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			handleErr = condition.NewInstallReadinessError(selinuxfixv1.AdmissionWebhookNotAvailable, err)
			return
		}

		// No APIService object
		object := a.asset.APIService().New()
		object.Spec.CABundle = ctx.GetBundle().ServingCertCA
		ctx.ControllerSetter().Set(object, original)

		apiservice, err := a.ensurer.Ensure(object)
		if err != nil {
			handleErr = condition.NewInstallReadinessError(selinuxfixv1.AdmissionWebhookNotAvailable, err)
			return
		}

		object = apiservice
		klog.V(2).Infof("key=%s resource=%T/%s successfully created", original.Name, object, object.Name)
	}

	if ref := original.Status.Resources.APiServiceRef; ref != nil && ref.ResourceVersion == object.ResourceVersion {
		klog.V(2).Infof("key=%s resource=%T/%s is in sync", original.Name, object, object.Name)
		return
	}

	newRef, err := reference.GetReference(object)
	if err != nil {
		handleErr = condition.NewInstallReadinessError(selinuxfixv1.CertNotAvailable, err)
		return
	}

	klog.V(2).Infof("key=%s resource=%T/%s resource-version=%s setting object reference", original.Name, object, object.Name, newRef.ResourceVersion)
	current.Status.Resources.APiServiceRef = newRef

	return
}
