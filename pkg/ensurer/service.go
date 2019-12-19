package ensurer

import (
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/dynamic"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func NewServiceEnsurer(client dynamic.Ensurer) *ServiceEnsurer {
	return &ServiceEnsurer{
		client: client,
	}
}

type ServiceEnsurer struct {
	client dynamic.Ensurer
}

func (s *ServiceEnsurer) Ensure(service *corev1.Service) (current *corev1.Service, err error) {
	unstructured, errGot := s.client.Ensure(string(corev1.ResourceServices), service)
	if errGot != nil {
		err = errGot
		return
	}

	current = &corev1.Service{}
	if conversionErr := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), current); conversionErr != nil {
		err = conversionErr
		return
	}

	return
}
