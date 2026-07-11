package infrastructure

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
)

var InfrastructureGVR = schema.GroupVersionResource{
	Group:    "config.openshift.io",
	Version:  "v1",
	Resource: "infrastructures",
}

// IsStandalone fetches the cluster Infrastructure resource and returns true
// if the control plane topology is NOT external (i.e. not a Hosted Control
// Plane cluster). On error it logs a warning and defaults to true (standalone)
// for backward compatibility.
func IsStandalone(ctx context.Context, dynClient dynamic.Interface) bool {
	obj, err := dynClient.Resource(InfrastructureGVR).Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		klog.Warningf("Failed to fetch Infrastructure resource, assuming standalone: %v", err)
		return true
	}

	infra := &configv1.Infrastructure{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, infra); err != nil {
		klog.Warningf("Failed to convert Infrastructure resource, assuming standalone: %v", err)
		return true
	}

	topology := infra.Status.ControlPlaneTopology
	klog.Infof("Detected control plane topology: %s", topology)
	return topology != configv1.ExternalTopologyMode
}
