package handlers

import (
	"context"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	"github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/reference"
	selinuxfixv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/selinuxfix/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/asset-selinux"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/deploy"
	dynamicclient "github.com/openshift/cluster-resource-override-admission-operator/pkg/dynamic"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/ensurer"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/secondarywatch"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/selinuxfix/internal/condition"
	controllerreconciler "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func NewDaemonSetHandler(o *Options) *daemonSetHandler {
	return &daemonSetHandler{
		client:     o.Client.Kubernetes,
		dynamic:    o.Client.Dynamic,
		deployment: ensurer.NewDaemonSetEnsurer(o.Client.Dynamic),
		asset:      o.Asset,
		lister:     o.SecondaryLister,
		deploy:     o.Deploy,
	}
}

type daemonSetHandler struct {
	client     kubernetes.Interface
	deployment *ensurer.DaemonSetEnsurer
	dynamic    dynamicclient.Ensurer
	lister     *secondarywatch.Lister
	asset      *asset.Asset

	deploy deploy.Interface
}

type Deployer interface {
	Exists(namespace, name string) (object metav1.Object, err error)
}

func (c *daemonSetHandler) Handle(context *ReconcileRequestContext, original *selinuxfixv1.SelinuxFixOverride) (current *selinuxfixv1.SelinuxFixOverride, result controllerreconciler.Result, handleErr error) {
	current = original
	ensure := false

	object, accessor, getErr := c.deploy.Get()
	if getErr != nil && !k8serrors.IsNotFound(getErr) {
		handleErr = condition.NewInstallReadinessError(selinuxfixv1.InternalError, getErr)
		return
	}

	values := c.asset.Values()
	switch {
	case k8serrors.IsNotFound(getErr):
		ensure = true
	case accessor.GetAnnotations()[values.ServingCertHashAnnotationKey] != current.Status.Hash.ServingCert:
		klog.V(2).Infof("key=%s resource=%T/%s serving cert hash mismatch", original.Name, object, accessor.GetName())
		ensure = true
	}

	if ensure {
		object, accessor, handleErr = c.Ensure(context, original)
		if handleErr != nil {
			return
		}

		klog.V(2).Infof("key=%s resource=%T/%s successfully ensured", original.Name, object, accessor.GetName())
	}

	if ref := current.Status.Resources.DeploymentRef; ref != nil && ref.ResourceVersion == accessor.GetResourceVersion() {
		klog.V(2).Infof("key=%s resource=%T/%s is in sync", original.Name, object, accessor.GetName())
		return
	}

	newRef, err := reference.GetReference(object)
	if err != nil {
		handleErr = condition.NewInstallReadinessError(selinuxfixv1.CertNotAvailable, err)
		return
	}

	klog.V(2).Infof("key=%s resource=%T/%s resource-version=%s setting object reference", original.Name, object, accessor.GetName(), newRef.ResourceVersion)
	current.Status.Resources.DeploymentRef = newRef

	return
}

func (c *daemonSetHandler) Ensure(ctx *ReconcileRequestContext, cro *selinuxfixv1.SelinuxFixOverride) (current runtime.Object, accessor metav1.Object, err error) {
	name := c.asset.NewMutatingWebhookConfiguration().Name()
	if deleteErr := c.client.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(context.TODO(), name, metav1.DeleteOptions{}); deleteErr != nil && !k8serrors.IsNotFound(deleteErr) {
		err = fmt.Errorf("failed to delete MutatingWebhookConfiguration - %s", deleteErr.Error())
		return
	}

	if err = c.EnsureRBAC(ctx, cro); err != nil {
		return
	}

	parent := c.ApplyToDeploymentObject(ctx, cro)
	child := c.ApplyToToPodTemplate(ctx, cro)
	current, accessor, err = c.deploy.Ensure(parent, child)
	return
}

func (c *daemonSetHandler) ApplyToDeploymentObject(context *ReconcileRequestContext, cro *selinuxfixv1.SelinuxFixOverride) deploy.Applier {
	values := c.asset.Values()

	return func(object metav1.Object) {
		if len(object.GetAnnotations()) == 0 {
			object.SetAnnotations(map[string]string{})
		}

		object.GetAnnotations()[values.ServingCertHashAnnotationKey] = cro.Status.Hash.ServingCert

		context.ControllerSetter().Set(object, cro)
	}
}

func (c *daemonSetHandler) ApplyToToPodTemplate(context *ReconcileRequestContext, cro *selinuxfixv1.SelinuxFixOverride) deploy.Applier {
	values := c.asset.Values()

	return func(object metav1.Object) {
		if len(object.GetAnnotations()) == 0 {
			object.SetAnnotations(map[string]string{})
		}

		object.GetAnnotations()[values.OwnerAnnotationKey] = cro.Name
		object.GetAnnotations()[values.ServingCertHashAnnotationKey] = cro.Status.Hash.ServingCert
	}
}

func (c *daemonSetHandler) EnsureRBAC(context *ReconcileRequestContext, in *selinuxfixv1.SelinuxFixOverride) error {
	list := c.asset.RBAC().New()
	for _, item := range list {
		context.ControllerSetter()(item.Object, in)

		current, err := c.dynamic.Ensure(item.Resource, item.Object)
		if err != nil {
			return fmt.Errorf("resource=%s failed to ensure RBAC - %s %v", item.Resource, err, item.Object)
		}

		klog.V(2).Infof("key=%s ensured RBAC resource %s", in.Name, current.GetName())
	}

	return nil
}
