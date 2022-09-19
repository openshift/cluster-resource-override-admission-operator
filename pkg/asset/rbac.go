package asset

import (
	"fmt"

	operatorruntime "github.com/openshift/cluster-resource-override-admission-operator/pkg/runtime"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (a *Asset) RBAC() *rbac {
	return &rbac{
		values: a.values,
	}
}

type RBACItem struct {
	Resource string
	Object   operatorruntime.Object
}

type rbac struct {
	values *Values
}

func (s *rbac) New() []*RBACItem {
	var (
		apiVersion = "rbac.authorization.k8s.io/v1"

		thisOperatorServiceAccount = rbacv1.Subject{
			Namespace: s.values.Namespace,
			Kind:      "ServiceAccount",
			Name:      s.values.ServiceAccountName,
		}

		defaultClusterRoleName = fmt.Sprintf("default-aggregated-apiserver-%s", s.values.Name)

		commonLabels = map[string]string{
			s.values.OwnerLabelKey: s.values.OwnerLabelValue,
		}
	)

	return []*RBACItem{
		// service account
		{
			Resource: "serviceaccounts",
			Object: &corev1.ServiceAccount{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ServiceAccount",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      s.values.Name,
					Namespace: s.values.Namespace,
				},
			},
		},

		// to read the config for terminating authentication
		{
			Resource: "rolebindings",
			Object: &rbacv1.RoleBinding{
				TypeMeta: metav1.TypeMeta{
					Kind:       "RoleBinding",
					APIVersion: apiVersion,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("extension-server-authentication-reader-%s", s.values.Name),
					Namespace: "kube-system",
					Labels:    commonLabels,
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     "extension-apiserver-authentication-reader",
				},
				Subjects: []rbacv1.Subject{
					thisOperatorServiceAccount,
				},
			},
		},

		// to let aggregated apiservers create admission reviews
		{
			Resource: "clusterroles",
			Object: &rbacv1.ClusterRole{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRole",
					APIVersion: apiVersion,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:   fmt.Sprintf("system:%s-requester", s.values.Name),
					Labels: commonLabels,
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{
							"autoscaling.openshift.io",
						},
						Resources: []string{
							s.values.Name,
						},
						Verbs: []string{
							"create",
						},
					},
				},
			},
		},

		// this should be a default for an aggregated apiserver
		{
			Resource: "clusterroles",
			Object: &rbacv1.ClusterRole{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRole",
					APIVersion: apiVersion,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:   defaultClusterRoleName,
					Labels: commonLabels,
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{
							"admissionregistration.k8s.io",
						},
						Resources: []string{
							"validatingwebhookconfigurations",
							"mutatingwebhookconfigurations",
						},
						Verbs: []string{
							"get",
							"list",
							"watch",
						},
					},
					// to give power to the operand to watch Namespace and LimitRange
					{
						APIGroups: []string{
							"",
						},
						Resources: []string{
							"namespaces",
							"limitranges",
						},
						Verbs: []string{
							"get",
							"list",
							"watch",
						},
					},
					// to give power to the operand to watch Namespace and LimitRange
					{
						APIGroups: []string{
							"flowcontrol.apiserver.k8s.io",
						},
						Resources: []string{
							"prioritylevelconfigurations",
							"flowschemas",
						},
						Verbs: []string{
							"get",
							"list",
							"watch",
						},
					},
				},
			},
		},

		// this should be a default for an aggregated apiserver
		{
			Resource: "clusterrolebindings",
			Object: &rbacv1.ClusterRoleBinding{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRoleBinding",
					APIVersion: apiVersion,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:   defaultClusterRoleName,
					Labels: commonLabels,
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     defaultClusterRoleName,
				},
				Subjects: []rbacv1.Subject{
					thisOperatorServiceAccount,
				},
			},
		},

		// to delegate authentication and authorization.
		{
			Resource: "clusterrolebindings",
			Object: &rbacv1.ClusterRoleBinding{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRoleBinding",
					APIVersion: apiVersion,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:   fmt.Sprintf("auth-delegator-%s", s.values.Name),
					Labels: commonLabels,
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     "system:auth-delegator",
				},
				Subjects: []rbacv1.Subject{
					thisOperatorServiceAccount,
				},
			},
		},

		// so that daemonset pods can use hostnetwork
		{
			Resource: "roles",
			Object: &rbacv1.Role{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Role",
					APIVersion: apiVersion,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-scc-hostnetwork-use", s.values.Name),
					Namespace: s.values.Namespace,
					Labels:    commonLabels,
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{
							"security.openshift.io",
						},
						Resources: []string{
							"securitycontextconstraints",
						},
						Verbs: []string{
							"use",
						},
						ResourceNames: []string{
							"hostnetwork-v2",
						},
					},
				},
			},
		},
		{
			Resource: "rolebindings",
			Object: &rbacv1.RoleBinding{
				TypeMeta: metav1.TypeMeta{
					Kind:       "RoleBinding",
					APIVersion: apiVersion,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-scc-hostnetwork-use", s.values.Name),
					Namespace: s.values.Namespace,
					Labels:    commonLabels,
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     fmt.Sprintf("%s-scc-hostnetwork-use", s.values.Name),
				},
				Subjects: []rbacv1.Subject{
					thisOperatorServiceAccount,
				},
			},
		},

		// so that kube-apiserver can directly call the webhook server
		{
			Resource: "clusterroles",
			Object: &rbacv1.ClusterRole{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRole",
					APIVersion: apiVersion,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:   fmt.Sprintf("%s-anonymous-access", s.values.Name),
					Labels: commonLabels,
				},
				Rules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{
							s.values.AdmissionAPIGroup,
						},
						Resources: []string{
							s.values.AdmissionAPIResource,
						},
						Verbs: []string{
							"create",
						},
					},
				},
			},
		},
		{
			Resource: "clusterrolebindings",
			Object: &rbacv1.ClusterRoleBinding{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRoleBinding",
					APIVersion: apiVersion,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:   fmt.Sprintf("%s-anonymous-access", s.values.Name),
					Labels: commonLabels,
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     fmt.Sprintf("%s-anonymous-access", s.values.Name),
				},
				Subjects: []rbacv1.Subject{
					{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "User",
						Name:     "system:anonymous",
					},
				},
			},
		},
	}
}
