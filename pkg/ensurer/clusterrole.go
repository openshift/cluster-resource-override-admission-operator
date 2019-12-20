package ensurer

import (
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/dynamic"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ClusterRoleEnsurer struct {
	client dynamic.Ensurer
}

func (c *ClusterRoleEnsurer) Ensure(role *rbacv1.ClusterRole) (current *rbacv1.ClusterRole, err error) {
	unstructured, errGot := c.client.Ensure("clusterroles", role)
	if errGot != nil {
		err = errGot
		return
	}

	current = &rbacv1.ClusterRole{}
	if conversionErr := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), current); conversionErr != nil {
		err = conversionErr
		return
	}

	return
}
