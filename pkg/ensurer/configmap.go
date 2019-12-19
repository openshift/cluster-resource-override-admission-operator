package ensurer

import (
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/dynamic"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func NewConfigMapEnsurer(client dynamic.Ensurer) *ConfigMapEnsurer {
	return &ConfigMapEnsurer{
		client: client,
	}
}

type ConfigMapEnsurer struct {
	client dynamic.Ensurer
}

func (s *ConfigMapEnsurer) Ensure(configmap *corev1.ConfigMap) (current *corev1.ConfigMap, err error) {
	unstructured, errGot := s.client.Ensure(string(corev1.ResourceConfigMaps), configmap)
	if errGot != nil {
		err = errGot
		return
	}

	current = &corev1.ConfigMap{}
	if conversionErr := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), current); conversionErr != nil {
		err = conversionErr
		return
	}

	return
}
