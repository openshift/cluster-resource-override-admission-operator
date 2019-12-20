package ensurer

import (
	"github.com/openshift/cluster-resource-override-admission-operator/pkg/dynamic"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type RoleEnsurer struct {
	client dynamic.Ensurer
}

func (r *RoleEnsurer) Ensure(role *rbacv1.Role) (current *rbacv1.Role, err error) {
	unstructured, errGot := r.client.Ensure("roles", role)
	if errGot != nil {
		err = errGot
		return
	}

	current = &rbacv1.Role{}
	if conversionErr := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.UnstructuredContent(), current); conversionErr != nil {
		err = conversionErr
		return
	}

	return
}
