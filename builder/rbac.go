package builder

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RbacBuilder struct {
	namespace          string
	serviceAccountName string
	roleName           string
	bindingName        string
}

func NewRbacBuilder(namespace string, serviceAccountName string, roleName string, bindingName string) *RbacBuilder {
	return &RbacBuilder{
		namespace:          namespace,
		serviceAccountName: serviceAccountName,
		roleName:           roleName,
		bindingName:        bindingName,
	}
}

func (builder *RbacBuilder) BuildServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      builder.serviceAccountName,
			Namespace: builder.namespace,
		},
	}
}

func (builder *RbacBuilder) BuildRole() *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      builder.roleName,
			Namespace: builder.namespace,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps", "secrets"},
				Verbs:     []string{"get", "watch", "list"},
			},
		},
	}
}

func (builder *RbacBuilder) BuildRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      builder.bindingName,
			Namespace: builder.namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     builder.roleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      builder.serviceAccountName,
				Namespace: builder.namespace,
			},
		},
	}
}
