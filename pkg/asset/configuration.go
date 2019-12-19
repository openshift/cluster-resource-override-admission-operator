package asset

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (a *Asset) Configuration() *configuration {
	return &configuration{
		values: a.values,
	}
}

type configuration struct {
	values *Values
}

func (c *configuration) Name() string {
	return fmt.Sprintf("%s-configuration", c.values.Name)
}

func (c *configuration) New() *corev1.ConfigMap {
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
		},
		Data: map[string]string{
			// The configuration will get injected from the `spec` of the Custom Resource
			// So we are leaving it empty.
			c.values.ConfigurationKey: "",
		},
	}
}
