package asset

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (a *Asset) CABundleConfigMap() *caBundleConfigMap {
	return &caBundleConfigMap{
		values: a.values,
	}
}

type caBundleConfigMap struct {
	values *Values
}

func (c *caBundleConfigMap) Name() string {
	return fmt.Sprintf("%s-service-serving", c.values.Name)
}

func (c *caBundleConfigMap) New() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Name(),
			Namespace: c.values.Namespace,
			Labels: map[string]string{
				c.values.OwnerLabelKey: c.values.OwnerLabelValue,
			},
			Annotations: map[string]string{
				"service-serving": "true",
			},
		},
	}
}
