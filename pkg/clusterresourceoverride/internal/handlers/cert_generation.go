package handlers

import (
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	controllerreconciler "sigs.k8s.io/controller-runtime/pkg/reconcile"

	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/reference"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/asset"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/cert"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/clusterresourceoverride/internal/condition"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/ensurer"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/secondarywatch"
)

var (
	// DefaultCertValidFor is the default duration a cert will be valid for.
	DefaultCertValidFor = time.Hour * 24 * 365

	// DefaultCertRotateThreshold is the default threshold preceding the expiration date
	// the operator will make attempt(s) to rotate the certs.
	DefaultCertRotateThreshold = time.Hour * 48

	Organization = "Red Hat, Inc."
)

func NewCertGenerationHandler(o *Options) *certGenerationHandler {
	return &certGenerationHandler{
		client:           o.Client.Kubernetes,
		secretEnsurer:    ensurer.NewSecretEnsurer(o.Client.Dynamic),
		configmapEnsurer: ensurer.NewConfigMapEnsurer(o.Client.Dynamic),
		lister:           o.SecondaryLister,
		asset:            o.Asset,
	}
}

type certGenerationHandler struct {
	client           kubernetes.Interface
	secretEnsurer    *ensurer.SecretEnsurer
	configmapEnsurer *ensurer.ConfigMapEnsurer
	lister           *secondarywatch.Lister
	asset            *asset.Asset
}

func (c *certGenerationHandler) Handle(context *ReconcileRequestContext, original *autoscalingv1.ClusterResourceOverride) (current *autoscalingv1.ClusterResourceOverride, result controllerreconciler.Result, handleErr error) {
	current = original
	ensure := false

	secretName := c.asset.ServiceServingSecret().Name()
	currentSecret, secretGetErr := c.lister.CoreV1SecretLister().Secrets(context.WebhookNamespace()).Get(secretName)
	if secretGetErr != nil && !k8serrors.IsNotFound(secretGetErr) {
		handleErr = condition.NewInstallReadinessError(autoscalingv1.InternalError, secretGetErr)
		return
	}

	configMapName := c.asset.CABundleConfigMap().Name()
	currentConfigMap, configMapGetErr := c.lister.CoreV1ConfigMapLister().ConfigMaps(context.WebhookNamespace()).Get(configMapName)
	if configMapGetErr != nil && !k8serrors.IsNotFound(configMapGetErr) {
		handleErr = condition.NewInstallReadinessError(autoscalingv1.InternalError, configMapGetErr)
		return
	}

	switch {
	case k8serrors.IsNotFound(secretGetErr) || k8serrors.IsNotFound(configMapGetErr):
		ensure = true
	case original.IsTimeToRotateCert() || !cert.IsPopulated(currentSecret):
		ensure = true
	case original.Status.CertsRotateAt.IsZero():
		ensure = true
	}

	if ensure {
		// generate cert.
		expiresAt := time.Now().Add(DefaultCertValidFor)
		bundle, err := cert.GenerateWithLocalhostServing(expiresAt, Organization)
		if err != nil {
			handleErr = condition.NewInstallReadinessError(autoscalingv1.CannotGenerateCert, err)
			return
		}

		// ensure that we have a serving Secret
		desiredSecret := c.asset.ServiceServingSecret().New()
		context.ControllerSetter().Set(desiredSecret, original)

		if len(desiredSecret.Data) == 0 {
			desiredSecret.Data = map[string][]byte{}
		}
		desiredSecret.Data["tls.key"] = bundle.Serving.ServiceKey
		desiredSecret.Data["tls.crt"] = bundle.Serving.ServiceCert

		secret, err := c.secretEnsurer.Ensure(desiredSecret)
		if err != nil {
			handleErr = condition.NewInstallReadinessError(autoscalingv1.CannotGenerateCert, err)
			return
		}

		// ensure that we have a configmap with the serving CA bundle
		desiredConfigMap := c.asset.CABundleConfigMap().New()

		context.ControllerSetter().Set(desiredConfigMap, original)
		if len(desiredConfigMap.Data) == 0 {
			desiredConfigMap.Data = map[string]string{}
		}
		desiredConfigMap.Data["service-ca.crt"] = string(bundle.ServingCertCA)

		configmap, err := c.configmapEnsurer.Ensure(desiredConfigMap)
		if err != nil {
			handleErr = condition.NewInstallReadinessError(autoscalingv1.CannotGenerateCert, err)
			return
		}

		context.SetBundle(bundle)
		current.Status.CertsRotateAt = metav1.NewTime(expiresAt.Add(-1 * DefaultCertRotateThreshold))

		currentSecret = secret
		klog.V(2).Infof("key=%s resource=%T/%s successfully ensured", original.Name, currentSecret, currentSecret.Name)

		currentConfigMap = configmap
		klog.V(2).Infof("key=%s resource=%T/%s successfully ensured", original.Name, currentConfigMap, currentConfigMap.Name)
	}

	if ref := current.Status.Resources.ServiceCertSecretRef; ref == nil || ref.ResourceVersion != currentSecret.ResourceVersion {
		newRef, err := reference.GetReference(currentSecret)
		if err != nil {
			handleErr = condition.NewInstallReadinessError(autoscalingv1.CannotSetReference, err)
			return
		}

		klog.V(2).Infof("key=%s resource=%T/%s resource-version=%s setting object reference", original.Name, currentSecret, currentSecret.Name, newRef.ResourceVersion)
		current.Status.Resources.ServiceCertSecretRef = newRef
	}
	klog.V(2).Infof("key=%s resource=%T/%s is original sync", original.Name, currentSecret, currentSecret.Name)

	if ref := current.Status.Resources.ServiceCAConfigMapRef; ref == nil || ref.ResourceVersion != currentConfigMap.ResourceVersion {
		newRef, err := reference.GetReference(currentConfigMap)
		if err != nil {
			handleErr = condition.NewInstallReadinessError(autoscalingv1.CannotSetReference, err)
			return
		}

		klog.V(2).Infof("key=%s resource=%T/%s resource-version=%s setting object reference", original.Name, currentConfigMap, currentConfigMap.Name, newRef.ResourceVersion)
		current.Status.Resources.ServiceCAConfigMapRef = newRef
	}
	klog.V(2).Infof("key=%s resource=%T/%s is original sync", original.Name, currentConfigMap, currentConfigMap.Name)

	return
}
