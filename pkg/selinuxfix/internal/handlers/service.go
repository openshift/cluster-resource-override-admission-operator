package handlers

import (
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/reference"
	selinuxfixv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/selinuxfix/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/asset-selinux"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/ensurer"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/secondarywatch"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/selinuxfix/internal/condition"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog"
	controllerreconciler "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func NewServiceHandler(o *Options) *serviceHandler {
	return &serviceHandler{
		dynamic: ensurer.NewServiceEnsurer(o.Client.Dynamic),
		lister:  o.SecondaryLister,
		asset:   o.Asset,
	}
}

type serviceHandler struct {
	dynamic *ensurer.ServiceEnsurer
	lister  *secondarywatch.Lister
	asset   *asset.Asset
}

func (s *serviceHandler) Handle(context *ReconcileRequestContext, original *selinuxfixv1.SelinuxFixOverride) (current *selinuxfixv1.SelinuxFixOverride, result controllerreconciler.Result, handleErr error) {
	current = original

	desired := s.asset.Service().New()
	context.ControllerSetter().Set(desired, original)

	name := s.asset.Service().Name()
	object, err := s.lister.CoreV1ServiceLister().Services(context.WebhookNamespace()).Get(name)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			handleErr = condition.NewInstallReadinessError(selinuxfixv1.CertNotAvailable, err)
			return
		}

		service, err := s.dynamic.Ensure(desired)
		if err != nil {
			handleErr = condition.NewInstallReadinessError(selinuxfixv1.CertNotAvailable, err)
			return
		}

		object = service
		klog.V(2).Infof("key=%s resource=%T/%s successfully created", original.Name, object, object.Name)
	}

	if ref := current.Status.Resources.ServiceRef; ref != nil && ref.ResourceVersion == object.ResourceVersion {
		klog.V(2).Infof("key=%s resource=%T/%s is in sync", original.Name, object, object.Name)
		return
	}

	newRef, err := reference.GetReference(object)
	if err != nil {
		handleErr = condition.NewInstallReadinessError(selinuxfixv1.CannotSetReference, err)
		return
	}

	klog.V(2).Infof("key=%s resource=%T/%s resource-version=%s setting object reference", original.Name, object, object.Name, newRef.ResourceVersion)

	current.Status.Resources.ServiceRef = newRef
	return
}

func (s *serviceHandler) Equal(this, that *corev1.Service) bool {
	return equality.Semantic.DeepDerivative(&this.Spec, &that.Spec) &&
		equality.Semantic.DeepDerivative(this.GetObjectMeta(), that.GetObjectMeta())
}
