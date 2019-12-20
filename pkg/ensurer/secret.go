package ensurer

import (
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/dynamic"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func NewSecretEnsurer(client dynamic.Ensurer) *SecretEnsurer {
	return &SecretEnsurer{
		client: client,
	}
}

type SecretEnsurer struct {
	client dynamic.Ensurer
}

func (s *SecretEnsurer) Ensure(secret *corev1.Secret) (current *corev1.Secret, err error) {
	unstructured, errGot := s.client.Ensure(string(corev1.ResourceSecrets), secret)
	if errGot != nil {
		err = errGot
		return
	}

	current = &corev1.Secret{}
	if conversionErr := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), current); conversionErr != nil {
		err = conversionErr
		return
	}

	return
}
