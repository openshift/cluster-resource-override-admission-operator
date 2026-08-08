package ensurer

import (
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/dynamic"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ValidatingAdmissionPolicyEnsurer struct {
	client dynamic.Ensurer
}

func NewValidatingAdmissionPolicyEnsurer(client dynamic.Ensurer) *ValidatingAdmissionPolicyEnsurer {
	return &ValidatingAdmissionPolicyEnsurer{
		client: client,
	}
}

func (v *ValidatingAdmissionPolicyEnsurer) Ensure(configuration *admissionregistrationv1.ValidatingAdmissionPolicy) (current *admissionregistrationv1.ValidatingAdmissionPolicy, err error) {
	unstructured, errGot := v.client.Ensure("validatingadmissionpolicies", configuration)
	if errGot != nil {
		err = errGot
		return
	}

	current = &admissionregistrationv1.ValidatingAdmissionPolicy{}
	if conversionErr := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), current); conversionErr != nil {
		err = conversionErr
		return
	}

	return
}

type ValidatingAdmissionPolicyBindingEnsurer struct {
	client dynamic.Ensurer
}

func NewValidatingAdmissionPolicyBindingEnsurer(client dynamic.Ensurer) *ValidatingAdmissionPolicyBindingEnsurer {
	return &ValidatingAdmissionPolicyBindingEnsurer{
		client: client,
	}
}

func (v *ValidatingAdmissionPolicyBindingEnsurer) Ensure(configuration *admissionregistrationv1.ValidatingAdmissionPolicyBinding) (current *admissionregistrationv1.ValidatingAdmissionPolicyBinding, err error) {
	unstructured, errGot := v.client.Ensure("validatingadmissionpolicybindings", configuration)
	if errGot != nil {
		err = errGot
		return
	}

	current = &admissionregistrationv1.ValidatingAdmissionPolicyBinding{}
	if conversionErr := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), current); conversionErr != nil {
		err = conversionErr
		return
	}

	return
}
