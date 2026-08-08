package asset

import (
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	validatingAdmissionPolicyName = "resourceoverride-exempt-namespace"
)

func (a *Asset) NewValidatingAdmissionPolicy() *validatingAdmissionPolicy {
	return &validatingAdmissionPolicy{
		values: a.values,
	}
}

type validatingAdmissionPolicy struct {
	values *Values
}

func (v *validatingAdmissionPolicy) Name() string {
	return validatingAdmissionPolicyName
}

func (v *validatingAdmissionPolicy) New() *admissionregistrationv1.ValidatingAdmissionPolicy {
	failurePolicy := admissionregistrationv1.Fail
	return &admissionregistrationv1.ValidatingAdmissionPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ValidatingAdmissionPolicy",
			APIVersion: "admissionregistration.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: v.Name(),
			Labels: map[string]string{
				v.values.OwnerLabelKey: v.values.OwnerLabelValue,
			},
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicySpec{
			FailurePolicy: &failurePolicy,
			MatchConstraints: &admissionregistrationv1.MatchResources{
				ResourceRules: []admissionregistrationv1.NamedRuleWithOperations{
					{
						RuleWithOperations: admissionregistrationv1.RuleWithOperations{
							Operations: []admissionregistrationv1.OperationType{
								admissionregistrationv1.Create,
							},
							Rule: admissionregistrationv1.Rule{
								APIGroups:   []string{"autoscaling.openshift.io"},
								APIVersions: []string{"v1"},
								Resources:   []string{"resourceoverrides"},
							},
						},
					},
				},
			},
			Validations: []admissionregistrationv1.Validation{
				{
					Expression: `!(object.metadata.namespace in ['openshift', 'kube', 'kubernetes'] || object.metadata.namespace.startsWith('openshift-') || object.metadata.namespace.startsWith('kube-') || object.metadata.namespace.startsWith('kubernetes-'))`,
					Message:    "ResourceOverride objects cannot be created in system namespaces (openshift, openshift-*, kube, kube-*, kubernetes, kubernetes-*)",
				},
			},
		},
	}
}

func (a *Asset) NewValidatingAdmissionPolicyBinding() *validatingAdmissionPolicyBinding {
	return &validatingAdmissionPolicyBinding{
		values: a.values,
	}
}

type validatingAdmissionPolicyBinding struct {
	values *Values
}

func (v *validatingAdmissionPolicyBinding) Name() string {
	return validatingAdmissionPolicyName
}

func (v *validatingAdmissionPolicyBinding) New() *admissionregistrationv1.ValidatingAdmissionPolicyBinding {
	return &admissionregistrationv1.ValidatingAdmissionPolicyBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ValidatingAdmissionPolicyBinding",
			APIVersion: "admissionregistration.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: v.Name(),
			Labels: map[string]string{
				v.values.OwnerLabelKey: v.values.OwnerLabelValue,
			},
		},
		Spec: admissionregistrationv1.ValidatingAdmissionPolicyBindingSpec{
			PolicyName: validatingAdmissionPolicyName,
			ValidationActions: []admissionregistrationv1.ValidationAction{
				admissionregistrationv1.Deny,
			},
		},
	}
}
