package ensurer

import (
	"k8s.io/apimachinery/pkg/runtime"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	"github.com/openshift/cluster-resource-override-admission-operator/pkg/dynamic"
)

func NewAPIServiceEnsurer(client dynamic.Ensurer) *APIServiceEnsurer {
	return &APIServiceEnsurer{
		client: client,
	}
}

type APIServiceEnsurer struct {
	client dynamic.Ensurer
}

func (s *APIServiceEnsurer) Ensure(apiservice *apiregistrationv1.APIService) (current *apiregistrationv1.APIService, err error) {
	unstructured, errGot := s.client.Ensure("apiservices", apiservice)
	if errGot != nil {
		err = errGot
		return
	}

	current = &apiregistrationv1.APIService{}
	if conversionErr := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), current); conversionErr != nil {
		err = conversionErr
		return
	}

	return
}
