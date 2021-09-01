package asset

import (
	"fmt"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	AutoRegisterManagedLabel = "kube-aggregator.kubernetes.io/automanaged"
)

func (a *Asset) NewMutatingWebhookConfiguration() *mutatingWebhookConfiguration {
	return &mutatingWebhookConfiguration{
		values: a.values,
	}
}

type mutatingWebhookConfiguration struct {
	values *Values
}

func (m *mutatingWebhookConfiguration) Name() string {
	return fmt.Sprintf("%s.%s", m.values.AdmissionAPIResource, m.values.AdmissionAPIGroup)
}

func (m *mutatingWebhookConfiguration) New() *admissionregistrationv1.MutatingWebhookConfiguration {
	url := fmt.Sprintf("https://localhost:9400/apis/%s/%s/%s", m.values.AdmissionAPIGroup, m.values.AdmissionAPIVersion, m.values.AdmissionAPIResource)
	policy := admissionregistrationv1.Fail
	matchPolicy := admissionregistrationv1.Equivalent
	namespaceMatchLabelKey := fmt.Sprintf("%s.%s/enabled", m.values.AdmissionAPIResource, m.values.AdmissionAPIGroup)
	timeoutSeconds := int32(5)
	sideEffects := admissionregistrationv1.SideEffectClassNone
	reinvoke := admissionregistrationv1.IfNeededReinvocationPolicy
	return &admissionregistrationv1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MutatingWebhookConfiguration",
			APIVersion: "admissionregistration.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: m.Name(),
			Labels: map[string]string{
				m.values.OwnerLabelKey: m.values.OwnerLabelValue,
			},
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: m.Name(),
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						namespaceMatchLabelKey: "true",
					},
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "openshift.io/run-level",
							Operator: metav1.LabelSelectorOpNotIn,
							Values: []string{
								"0",
								"1",
							},
						},
					},
				},
				MatchPolicy: &matchPolicy,
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					// CABundle will be injected at runtime
					CABundle: nil,
					URL:      &url,
				},
				Rules: []admissionregistrationv1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1.OperationType{
							admissionregistrationv1.Create,
							admissionregistrationv1.Update,
						},

						Rule: admissionregistrationv1.Rule{
							APIGroups: []string{
								"",
							},
							APIVersions: []string{
								"v1",
							},
							Resources: []string{
								"pods",
							},
						},
					},
				},
				FailurePolicy:           &policy,
				TimeoutSeconds:          &timeoutSeconds,
				SideEffects:             &sideEffects,
				ReinvocationPolicy:      &reinvoke,
				AdmissionReviewVersions: []string{"v1"},
			},
		},
	}
}
