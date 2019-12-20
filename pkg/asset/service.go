package asset

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	ServingCertSecretAnnotationName = "service.alpha.openshift.io/serving-cert-secret-name"
)

func (a *Asset) Service() *service {
	return &service{
		asset: a,
	}
}

type service struct {
	asset *Asset
}

func (s *service) Name() string {
	return s.asset.Values().Name
}

func (s *service) New() *corev1.Service {
	values := s.asset.Values()

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      values.Name,
			Namespace: values.Namespace,
			Labels: map[string]string{
				values.OwnerLabelKey: values.OwnerLabelValue,
			},
			Annotations: map[string]string{
				ServingCertSecretAnnotationName: s.asset.ServiceServingSecret().Name(),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				values.SelectorLabelKey: values.SelectorLabelValue,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       443,
					TargetPort: intstr.FromInt(8443),
				},
			},
		},
	}
}
