package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	operatorv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/operator/v1"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/asset"
	operatorruntime "github.com/openshift/cluster-resource-override-admission-operator/pkg/runtime"
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/tlsprofile"
)

// minimalHandler builds a deploymentHandler with only the asset field set, which
// is all that ApplyToToPodTemplate and ApplyToDeploymentObject require.
func minimalHandler(t *testing.T) *deploymentHandler {
	t.Helper()
	ctx := operatorruntime.NewOperandContext("clusterresourceoverride", "test-ns", "cluster", "test-image:latest", "1.0.0")
	return &deploymentHandler{
		asset: asset.New(ctx),
	}
}

// minimalStandaloneHandler builds a deploymentHandler with isStandalone=true.
func minimalStandaloneHandler(t *testing.T) *deploymentHandler {
	t.Helper()
	h := minimalHandler(t)
	h.isStandalone = true
	return h
}

// minimalCRO returns a ClusterResourceOverride with the GVK set (required by
// SetController) and a known configuration hash.
func minimalCRO() *operatorv1.ClusterResourceOverride {
	cro := &operatorv1.ClusterResourceOverride{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
	}
	cro.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   operatorv1.GroupName,
		Version: operatorv1.GroupVersion,
		Kind:    operatorv1.ClusterResourceOverrideKind,
	})
	cro.Status.Hash.Configuration = "testhash"
	return cro
}

func podTemplateWithBaseArgs() *corev1.PodTemplateSpec {
	return &corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "clusterresourceoverride",
					Args: []string{
						"--secure-port=9400",
						"--tls-cert-file=/var/serving-cert/tls.crt",
						"--tls-private-key-file=/var/serving-cert/tls.key",
						"--v=8",
					},
				},
			},
		},
	}
}

func TestApplyToToPodTemplate_EmptyTLSArgs(t *testing.T) {
	h := minimalHandler(t)
	ctx := NewReconcileRequestContext(operatorruntime.NewOperandContext("clusterresourceoverride", "test-ns", "cluster", "img", "1.0"))
	cro := minimalCRO()
	pt := podTemplateWithBaseArgs()

	applier := h.ApplyToToPodTemplate(ctx, cro, tlsprofile.Args{})
	applier.Apply(pt)

	args := pt.Spec.Containers[0].Args
	for _, arg := range args {
		assert.NotContains(t, arg, "--tls-min-version", "empty TLS args must not add --tls-min-version")
		assert.NotContains(t, arg, "--tls-cipher-suites", "empty TLS args must not add --tls-cipher-suites")
	}
}

func TestApplyToToPodTemplate_MinVersionOnly(t *testing.T) {
	h := minimalHandler(t)
	ctx := NewReconcileRequestContext(operatorruntime.NewOperandContext("clusterresourceoverride", "test-ns", "cluster", "img", "1.0"))
	cro := minimalCRO()
	pt := podTemplateWithBaseArgs()

	applier := h.ApplyToToPodTemplate(ctx, cro, tlsprofile.Args{MinVersion: "VersionTLS12"})
	applier.Apply(pt)

	args := pt.Spec.Containers[0].Args
	assert.Contains(t, args, "--tls-min-version=VersionTLS12")
	for _, arg := range args {
		assert.NotContains(t, arg, "--tls-cipher-suites", "only MinVersion was set; --tls-cipher-suites must be absent")
	}
}

func TestApplyToToPodTemplate_CipherSuitesOnly(t *testing.T) {
	h := minimalHandler(t)
	ctx := NewReconcileRequestContext(operatorruntime.NewOperandContext("clusterresourceoverride", "test-ns", "cluster", "img", "1.0"))
	cro := minimalCRO()
	pt := podTemplateWithBaseArgs()

	ciphers := "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
	applier := h.ApplyToToPodTemplate(ctx, cro, tlsprofile.Args{CipherSuites: ciphers})
	applier.Apply(pt)

	args := pt.Spec.Containers[0].Args
	assert.Contains(t, args, "--tls-cipher-suites="+ciphers)
	for _, arg := range args {
		assert.NotContains(t, arg, "--tls-min-version", "only CipherSuites was set; --tls-min-version must be absent")
	}
}

func TestApplyToToPodTemplate_BothTLSArgs(t *testing.T) {
	h := minimalHandler(t)
	ctx := NewReconcileRequestContext(operatorruntime.NewOperandContext("clusterresourceoverride", "test-ns", "cluster", "img", "1.0"))
	cro := minimalCRO()
	pt := podTemplateWithBaseArgs()

	tlsArgs := tlsprofile.Args{
		MinVersion:   "VersionTLS13",
		CipherSuites: "TLS_AES_128_GCM_SHA256",
	}
	applier := h.ApplyToToPodTemplate(ctx, cro, tlsArgs)
	applier.Apply(pt)

	args := pt.Spec.Containers[0].Args
	assert.Contains(t, args, "--tls-min-version=VersionTLS13")
	assert.Contains(t, args, "--tls-cipher-suites=TLS_AES_128_GCM_SHA256")
}

func TestApplyToToPodTemplate_BaseArgsUnchanged(t *testing.T) {
	h := minimalHandler(t)
	ctx := NewReconcileRequestContext(operatorruntime.NewOperandContext("clusterresourceoverride", "test-ns", "cluster", "img", "1.0"))
	cro := minimalCRO()
	pt := podTemplateWithBaseArgs()
	baseArgs := make([]string, len(pt.Spec.Containers[0].Args))
	copy(baseArgs, pt.Spec.Containers[0].Args)

	applier := h.ApplyToToPodTemplate(ctx, cro, tlsprofile.Args{MinVersion: "VersionTLS12"})
	applier.Apply(pt)

	// All original base args must still be present
	for _, base := range baseArgs {
		assert.Contains(t, pt.Spec.Containers[0].Args, base, "base arg %q must not be removed", base)
	}
}

func TestApplyToDeploymentObject_TLSHashAnnotation(t *testing.T) {
	h := minimalHandler(t)
	ctx := NewReconcileRequestContext(operatorruntime.NewOperandContext("clusterresourceoverride", "test-ns", "cluster", "img", "1.0"))
	cro := minimalCRO()

	tlsArgs := tlsprofile.Args{MinVersion: "VersionTLS12", CipherSuites: "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"}
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
	}

	applier := h.ApplyToDeploymentObject(ctx, cro, tlsArgs)
	applier.Apply(deployment)

	annotationKey := h.asset.Values().TLSProfileHashAnnotationKey
	require.NotEmpty(t, annotationKey)
	assert.Equal(t, tlsArgs.Hash(), deployment.GetAnnotations()[annotationKey],
		"TLS profile hash annotation must match Args.Hash()")
}

func TestApplyToDeploymentObject_EmptyTLSArgsProducesEmptyAnnotation(t *testing.T) {
	h := minimalHandler(t)
	ctx := NewReconcileRequestContext(operatorruntime.NewOperandContext("clusterresourceoverride", "test-ns", "cluster", "img", "1.0"))
	cro := minimalCRO()
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
	}

	applier := h.ApplyToDeploymentObject(ctx, cro, tlsprofile.Args{})
	applier.Apply(deployment)

	annotationKey := h.asset.Values().TLSProfileHashAnnotationKey
	assert.Equal(t, "", deployment.GetAnnotations()[annotationKey],
		"empty TLS args must produce empty annotation so existing deployments are not spuriously re-ensured")
}

func TestApplyToToPodTemplate_StandaloneNoUserOverride(t *testing.T) {
	h := minimalStandaloneHandler(t)
	ctx := NewReconcileRequestContext(operatorruntime.NewOperandContext("clusterresourceoverride", "test-ns", "cluster", "img", "1.0"))
	cro := minimalCRO()
	pt := podTemplateWithBaseArgs()

	applier := h.ApplyToToPodTemplate(ctx, cro, tlsprofile.Args{})
	applier.Apply(pt)

	assert.Equal(t, map[string]string{"node-role.kubernetes.io/control-plane": ""}, pt.Spec.NodeSelector,
		"standalone cluster should get control-plane node selector")

	hasMasterToleration := false
	hasControlPlaneToleration := false
	for _, tol := range pt.Spec.Tolerations {
		if tol.Key == "node-role.kubernetes.io/master" && tol.Effect == corev1.TaintEffectNoSchedule {
			hasMasterToleration = true
		}
		if tol.Key == "node-role.kubernetes.io/control-plane" && tol.Effect == corev1.TaintEffectNoSchedule {
			hasControlPlaneToleration = true
		}
	}
	assert.True(t, hasMasterToleration, "standalone cluster should tolerate master NoSchedule taint")
	assert.True(t, hasControlPlaneToleration, "standalone cluster should tolerate control-plane NoSchedule taint")
}

func TestApplyToToPodTemplate_ExternalNoUserOverride(t *testing.T) {
	h := minimalHandler(t) // isStandalone defaults to false
	ctx := NewReconcileRequestContext(operatorruntime.NewOperandContext("clusterresourceoverride", "test-ns", "cluster", "img", "1.0"))
	cro := minimalCRO()
	pt := podTemplateWithBaseArgs()

	applier := h.ApplyToToPodTemplate(ctx, cro, tlsprofile.Args{})
	applier.Apply(pt)

	assert.Empty(t, pt.Spec.NodeSelector, "HCP cluster should not set a control-plane node selector")

	for _, tol := range pt.Spec.Tolerations {
		assert.NotEqual(t, "node-role.kubernetes.io/master", tol.Key, "HCP cluster should not add master toleration")
		assert.NotEqual(t, "node-role.kubernetes.io/control-plane", tol.Key, "HCP cluster should not add control-plane toleration")
	}
}

func TestApplyToToPodTemplate_UserOverrideOnStandalone(t *testing.T) {
	h := minimalStandaloneHandler(t)
	ctx := NewReconcileRequestContext(operatorruntime.NewOperandContext("clusterresourceoverride", "test-ns", "cluster", "img", "1.0"))
	cro := minimalCRO()
	cro.Spec.DeploymentOverrides.NodeSelector = map[string]string{"node-role.kubernetes.io/worker": ""}
	cro.Spec.DeploymentOverrides.Tolerations = []corev1.Toleration{
		{Key: "custom", Operator: corev1.TolerationOpEqual, Value: "val", Effect: corev1.TaintEffectNoSchedule},
	}
	pt := podTemplateWithBaseArgs()

	applier := h.ApplyToToPodTemplate(ctx, cro, tlsprofile.Args{})
	applier.Apply(pt)

	assert.Equal(t, map[string]string{"node-role.kubernetes.io/worker": ""}, pt.Spec.NodeSelector,
		"user override should take precedence over auto control-plane selector")
	assert.Equal(t, cro.Spec.DeploymentOverrides.Tolerations, pt.Spec.Tolerations,
		"user override should take precedence over auto tolerations")
}
