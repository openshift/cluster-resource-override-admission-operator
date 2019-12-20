package asset

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (a *Asset) ServiceServingSecret() *serviceServingSecret {
	return &serviceServingSecret{
		values: a.values,
	}
}

const (
	SecretNamePrefix = "server-serving-cert"
)

type serviceServingSecret struct {
	values *Values
}

func (s *serviceServingSecret) Name() string {
	return fmt.Sprintf("%s-%s", SecretNamePrefix, s.values.Name)
}

func (s *serviceServingSecret) New() *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: s.values.Namespace,
			Name:      s.Name(),
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": nil,
			"tls.key": nil,
		},
	}
}
