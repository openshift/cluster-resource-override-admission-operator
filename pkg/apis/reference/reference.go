package reference

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ref "k8s.io/client-go/tools/reference"

	autoscalingv1 "github.com/openshift/cluster-resource-override-admission-operator/pkg/apis/autoscaling/v1"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
)

func init() {
	// Register custom types with the scheme
	utilruntime.Must(autoscalingv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(apiregistrationv1.AddToScheme(scheme.Scheme))
}

// GetReference returns an ObjectReference for the given object.
func GetReference(obj runtime.Object) (*corev1.ObjectReference, error) {
	return ref.GetReference(scheme.Scheme, obj)
}
