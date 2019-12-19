package ensurer

import (
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/dynamic"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func NewDeploymentEnsurer(client dynamic.Ensurer) *DeploymentEnsurer {
	return &DeploymentEnsurer{
		client: client,
	}
}

type DeploymentEnsurer struct {
	client dynamic.Ensurer
}

func (s *DeploymentEnsurer) Ensure(deployment *appsv1.Deployment) (current *appsv1.Deployment, err error) {
	unstructured, errGot := s.client.Ensure("deployments", deployment)
	if errGot != nil {
		err = errGot
		return
	}

	current = &appsv1.Deployment{}
	if conversionErr := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), current); conversionErr != nil {
		err = conversionErr
		return
	}

	return
}
