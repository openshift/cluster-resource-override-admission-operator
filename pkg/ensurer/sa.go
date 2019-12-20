package ensurer

import (
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/dynamic"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ServiceAccountEnsurer struct {
	client dynamic.Ensurer
}

func (s *ServiceAccountEnsurer) Ensure(object *corev1.ServiceAccount) (current *corev1.ServiceAccount, err error) {
	unstructured, errGot := s.client.Ensure("serviceaccounts", object)
	if errGot != nil {
		err = errGot
		return
	}

	current = &corev1.ServiceAccount{}
	if conversionErr := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), current); conversionErr != nil {
		err = conversionErr
		return
	}

	return
}
