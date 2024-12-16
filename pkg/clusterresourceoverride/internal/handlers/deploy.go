package handlers

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/reference"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/asset"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/clusterresourceoverride/internal/condition"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/deploy"
	dynamicclient "github.com/openshift/cluster-resource-override-admission-operator/pkg/dynamic"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/ensurer"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/secondarywatch"
	controllerreconciler "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func NewDeploymentHandler(o *Options) *deploymentHandler {
	return &deploymentHandler{
		client:     o.Client.Kubernetes,
		dynamic:    o.Client.Dynamic,
		deployment: ensurer.NewDeploymentEnsurer(o.Client.Dynamic),
		asset:      o.Asset,
		lister:     o.SecondaryLister,
		deploy:     o.Deploy,
	}
}

type deploymentHandler struct {
	client     kubernetes.Interface
	deployment *ensurer.DeploymentEnsurer
	dynamic    dynamicclient.Ensurer
	lister     *secondarywatch.Lister
	asset      *asset.Asset

	deploy deploy.Interface
}

func (c *deploymentHandler) Handle(ctx *ReconcileRequestContext, original *autoscalingv1.ClusterResourceOverride) (current *autoscalingv1.ClusterResourceOverride, result controllerreconciler.Result, handleErr error) {
	current = original

	// Remove the DaemonSet if it exists; used prior to v4.17.0
	dsName := c.asset.DaemonSet().Name()
	dsNamespace := c.asset.Values().Namespace
	if deleteErr := c.client.AppsV1().DaemonSets(dsNamespace).Delete(context.TODO(), dsName, metav1.DeleteOptions{}); deleteErr != nil && !k8serrors.IsNotFound(deleteErr) {
		handleErr = fmt.Errorf("failed to delete DaemonSet - %s", deleteErr.Error())
		return
	} else if deleteErr == nil {
		klog.V(2).Infof("Dangling DaemonSet %s in namespace %s deleted successfully", dsName, dsNamespace)
		return
	}

	ensure := false

	object, accessor, getErr := c.deploy.Get()
	if getErr != nil && !k8serrors.IsNotFound(getErr) {
		handleErr = condition.NewInstallReadinessError(autoscalingv1.InternalError, getErr)
		return
	}

	values := c.asset.Values()
	switch {
	case k8serrors.IsNotFound(getErr):
		ensure = true
	case accessor.GetAnnotations()[values.ConfigurationHashAnnotationKey] != current.Status.Hash.Configuration:
		klog.V(2).Infof("key=%s resource=%T/%s configuration hash mismatch", original.Name, object, accessor.GetName())
		ensure = true
	case values.OperandImage != current.Status.Image:
		klog.V(2).Infof("operand image mismatch: current: %s original: %s", current.Status.Image, values.OperandImage)
		ensure = true
	case values.OperandVersion != current.Status.Version:
		klog.V(2).Infof("operand version mismatch: current: %s original: %s", current.Status.Version, values.OperandVersion)
		ensure = true
	}

	if ensure {
		object, accessor, handleErr = c.Ensure(ctx, original)
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
		handleErr = condition.NewInstallReadinessError(autoscalingv1.CertNotAvailable, err)
		return
	}

	klog.V(2).Infof("key=%s resource=%T/%s resource-version=%s setting object reference", original.Name, object, accessor.GetName(), newRef.ResourceVersion)
	current.Status.Resources.DeploymentRef = newRef

	return
}

func (c *deploymentHandler) Ensure(ctx *ReconcileRequestContext, cro *autoscalingv1.ClusterResourceOverride) (current runtime.Object, accessor metav1.Object, err error) {
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

func (c *deploymentHandler) ApplyToDeploymentObject(context *ReconcileRequestContext, cro *autoscalingv1.ClusterResourceOverride) deploy.Applier {
	values := c.asset.Values()

	return func(object metav1.Object) {
		deployment, ok := object.(*appsv1.Deployment)
		if !ok {
			klog.Errorf("expected Deployment, got %T", object)
			return
		}
		if len(deployment.GetAnnotations()) == 0 {
			deployment.SetAnnotations(map[string]string{})
		}

		deployment.GetAnnotations()[values.ConfigurationHashAnnotationKey] = cro.Status.Hash.Configuration

		// Override replica count, if specified in the CRD
		if cro.Spec.DeploymentOverrides.Replicas != nil {
			deployment.Spec.Replicas = cro.Spec.DeploymentOverrides.Replicas
		}

		context.ControllerSetter().Set(object, cro)
	}
}

func (c *deploymentHandler) ApplyToToPodTemplate(context *ReconcileRequestContext, cro *autoscalingv1.ClusterResourceOverride) deploy.Applier {
	values := c.asset.Values()

	return func(object metav1.Object) {
		podTemplateSpec, ok := object.(*corev1.PodTemplateSpec)
		if !ok {
			klog.Errorf("expected PodTemplateSpec, got %T", object)
			return
		}

		if len(podTemplateSpec.GetAnnotations()) == 0 {
			podTemplateSpec.SetAnnotations(map[string]string{})
		}

		podTemplateSpec.GetAnnotations()[values.OwnerAnnotationKey] = cro.Name
		podTemplateSpec.GetAnnotations()[values.ConfigurationHashAnnotationKey] = cro.Status.Hash.Configuration

		// Replaces nodeSelector, if specified in the CR
		if len(cro.Spec.DeploymentOverrides.NodeSelector) > 0 {
			podTemplateSpec.Spec.NodeSelector = cro.Spec.DeploymentOverrides.NodeSelector
		}

		// Replaces tolerations, if specified in the CR
		if len(cro.Spec.DeploymentOverrides.Tolerations) > 0 {
			podTemplateSpec.Spec.Tolerations = cro.Spec.DeploymentOverrides.Tolerations
		}
	}
}

func (c *deploymentHandler) EnsureRBAC(context *ReconcileRequestContext, in *autoscalingv1.ClusterResourceOverride) error {
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
