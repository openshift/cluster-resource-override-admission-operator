package ensurer

import (
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/dynamic"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func NewMutatingWebhookConfigurationEnsurer(client dynamic.Ensurer) *MutatingWebhookConfigurationEnsurer {
	return &MutatingWebhookConfigurationEnsurer{
		client: client,
	}
}

type MutatingWebhookConfigurationEnsurer struct {
	client dynamic.Ensurer
}

func (m *MutatingWebhookConfigurationEnsurer) Ensure(configuration *admissionregistrationv1.MutatingWebhookConfiguration) (current *admissionregistrationv1.MutatingWebhookConfiguration, err error) {
	unstructured, errGot := m.client.Ensure("mutatingwebhookconfigurations", configuration)
	if errGot != nil {
		err = errGot
		return
	}

	current = &admissionregistrationv1.MutatingWebhookConfiguration{}
	if conversionErr := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), current); conversionErr != nil {
		err = conversionErr
		return
	}

	return
}
