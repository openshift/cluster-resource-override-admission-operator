package asset

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
)

func (a *Asset) APIService() *apiservice {
	return &apiservice{
		values: a.values,
	}
}

type apiservice struct {
	values *Values
}

func (a *apiservice) Name() string {
	return fmt.Sprintf("%s.%s", a.values.AdmissionAPIVersion, a.values.AdmissionAPIGroup)
}

func (a *apiservice) New() *apiregistrationv1.APIService {
	return &apiregistrationv1.APIService{
		TypeMeta: metav1.TypeMeta{
			Kind:       "APIService",
			APIVersion: "apiregistration.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: a.Name(),
			Labels: map[string]string{
				a.values.OwnerLabelKey:   a.values.OwnerLabelValue,
				AutoRegisterManagedLabel: "onstart",
			},
		},
		Spec: apiregistrationv1.APIServiceSpec{
			Version:              a.values.AdmissionAPIVersion,
			Group:                a.values.AdmissionAPIGroup,
			GroupPriorityMinimum: 1000,
			VersionPriority:      15,
			Service: &apiregistrationv1.ServiceReference{
				Namespace: a.values.Namespace,
				Name:      a.values.Name,
			},

			// CABundle will be injected at runtime.
			CABundle: nil,
		},
	}
}
