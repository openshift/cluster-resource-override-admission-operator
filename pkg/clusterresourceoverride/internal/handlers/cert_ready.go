package handlers

import (
	"fmt"
	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/cert"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/clusterresourceoverride/internal/condition"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/secondarywatch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	controllerreconciler "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func NewCertReadyHandler(o *Options) *certReadyHandler {
	return &certReadyHandler{
		client: o.Client.Kubernetes,
		lister: o.SecondaryLister,
	}
}

type certReadyHandler struct {
	client kubernetes.Interface
	lister *secondarywatch.Lister
}

func (c *certReadyHandler) Handle(context *ReconcileRequestContext, original *autoscalingv1.ClusterResourceOverride) (current *autoscalingv1.ClusterResourceOverride, result controllerreconciler.Result, handleErr error) {
	current = original
	resources := original.Status.Resources

	if context.GetBundle() == nil {
		secret, err := c.lister.CoreV1SecretLister().Secrets(context.WebhookNamespace()).Get(resources.ServiceCertSecretRef.Name)
		if err != nil {
			handleErr = condition.NewInstallReadinessError(autoscalingv1.CertNotAvailable, err)
			return
		}

		configmap, err := c.lister.CoreV1ConfigMapLister().ConfigMaps(context.WebhookNamespace()).Get(resources.ServiceCAConfigMapRef.Name)
		if err != nil {
			handleErr = condition.NewInstallReadinessError(autoscalingv1.CertNotAvailable, err)
			return
		}

		servingCertCA := []byte(configmap.Data["service-ca.crt"])
		bundle := &cert.Bundle{
			Serving: cert.Serving{
				ServiceKey:  secret.Data["tls.key"],
				ServiceCert: secret.Data["tls.crt"],
			},
			ServingCertCA: servingCertCA,
		}

		if err := bundle.Validate(); err != nil {
			handleErr = condition.NewInstallReadinessError(autoscalingv1.CertNotAvailable, fmt.Errorf("certs not populated - %s", err.Error()))
			return
		}

		context.SetBundle(bundle)
	}

	bundle := context.GetBundle()
	current.Status.Hash.ServingCert = bundle.Hash()

	klog.V(2).Infof("key=%s cert check passed", original.Name)
	return
}
